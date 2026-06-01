/**
 * Security 既有单元测试移植（任务 11.3，Req 17.2 / 17.5）。
 *
 * 来源：旧 Vue 版 `frontend/src/views/security/policy-feedback-draft.test.ts`
 * （原以 `node:test` + `node:assert/strict` 通过 `tsx --test` 运行）。
 * 改写为 vitest（`describe`/`it`/`expect`），断言逻辑与覆盖范围保持不变。
 */
import { describe, expect, it } from 'vitest';

import {
  buildExclusionCandidateKey,
  collectExclusionCandidatesFromFeedbackSuggestion,
  mergePolicyFeedbackCheckedRowKeys,
  parseExclusionCandidateKey,
  parseExclusionFromFeedbackSuggestion,
} from './policy-feedback-draft';

describe('policy-feedback-draft', () => {
  it('buildExclusionCandidateKey and parseExclusionCandidateKey round trip', () => {
    const key = buildExclusionCandidateKey('id', '920350');
    expect(parseExclusionCandidateKey(key)).toEqual({ removeType: 'id', removeValue: '920350' });
    expect(parseExclusionCandidateKey('')).toBe(null);
    expect(parseExclusionCandidateKey('invalid')).toBe(null);
  });

  it('collectExclusionCandidatesFromFeedbackSuggestion supports multi templates and de-duplication', () => {
    const suggestion = [
      '建议 removeById: 920350',
      '并且 removeByTag attack-sqli',
      '可追加 ruleRemoveByTag=attack-rce',
      '规则id 920350',
    ].join(' ; ');
    const candidates = collectExclusionCandidatesFromFeedbackSuggestion(suggestion);

    expect(
      [...candidates].sort(
        (a, b) => a.removeType.localeCompare(b.removeType) || a.removeValue.localeCompare(b.removeValue),
      ),
    ).toEqual([
      { removeType: 'id', removeValue: '920350' },
      { removeType: 'tag', removeValue: 'attack-rce' },
      { removeType: 'tag', removeValue: 'attack-sqli' },
    ]);
    expect(new Set(candidates.map(item => `${item.removeType}:${item.removeValue}`)).size).toBe(3);
    expect(candidates.every(item => item.removeValue.length > 0)).toBe(true);
  });

  it('parseExclusionFromFeedbackSuggestion returns first candidate or empty default', () => {
    const parsed = parseExclusionFromFeedbackSuggestion('removeById 941120');
    expect(parsed).toEqual({ removeType: 'id', removeValue: '941120' });
    expect(parseExclusionFromFeedbackSuggestion('无明确建议')).toEqual({ removeType: 'id', removeValue: '' });
  });

  it('collectExclusionCandidatesFromFeedbackSuggestion supports quoted values and chinese punctuation', () => {
    const suggestion = ['建议：移除标签："attack-xss"，', "按ID移除：'942100'；", 'ruleRemoveByTag=`attack-lfi`'].join(
      ' ',
    );
    const candidates = collectExclusionCandidatesFromFeedbackSuggestion(suggestion);

    expect(
      [...candidates].sort(
        (a, b) => a.removeType.localeCompare(b.removeType) || a.removeValue.localeCompare(b.removeValue),
      ),
    ).toEqual([
      { removeType: 'id', removeValue: '942100' },
      { removeType: 'tag', removeValue: 'attack-lfi' },
      { removeType: 'tag', removeValue: 'attack-xss' },
    ]);
  });

  it('collectExclusionCandidatesFromFeedbackSuggestion tolerates dirty fragments and broken tokens', () => {
    const suggestion = [
      '建议 removeByTag：attack-rce；',
      'rule id=9x2134（坏样本）',
      'removeById: 949110###',
      '标签=attack-sqli,,',
      '脏数据<script>alert(1)</script>',
      'fallback id: 950001',
    ].join(' ');

    const candidates = collectExclusionCandidatesFromFeedbackSuggestion(suggestion);
    expect(
      [...candidates].sort(
        (a, b) => a.removeType.localeCompare(b.removeType) || a.removeValue.localeCompare(b.removeValue),
      ),
    ).toEqual([
      { removeType: 'id', removeValue: '949110' },
      { removeType: 'id', removeValue: '950001' },
      { removeType: 'tag', removeValue: 'attack-rce' },
      { removeType: 'tag', removeValue: 'attack-sqli' },
    ]);
  });

  it('parseExclusionCandidateKey rejects malformed values', () => {
    expect(parseExclusionCandidateKey('tag')).toBe(null);
    expect(parseExclusionCandidateKey('\u0000attack-sqli')).toBe(null);
    expect(parseExclusionCandidateKey('other\u0000attack-sqli')).toBe(null);
    expect(parseExclusionCandidateKey('id\u0000')).toBe(null);
  });

  it('mergePolicyFeedbackCheckedRowKeys keeps cross-page selections and updates current page', () => {
    const previous = [1, 2, 8];
    const currentPageIDs = [1, 2, 3];
    const nextPageChecked = [2, 3];
    const merged = mergePolicyFeedbackCheckedRowKeys(previous, currentPageIDs, nextPageChecked);
    expect(merged.sort((a, b) => a - b)).toEqual([2, 3, 8]);
  });
});
