import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildExclusionCandidateKey,
  collectExclusionCandidatesFromFeedbackSuggestion,
  mergePolicyFeedbackCheckedRowKeys,
  parseExclusionCandidateKey,
  parseExclusionFromFeedbackSuggestion
} from './policy-feedback-draft';

test('buildExclusionCandidateKey and parseExclusionCandidateKey round trip', () => {
  const key = buildExclusionCandidateKey('id', '920350');
  assert.deepEqual(parseExclusionCandidateKey(key), { removeType: 'id', removeValue: '920350' });
  assert.equal(parseExclusionCandidateKey(''), null);
  assert.equal(parseExclusionCandidateKey('invalid'), null);
});

test('collectExclusionCandidatesFromFeedbackSuggestion supports multi templates and de-duplication', () => {
  const suggestion = [
    '建议 removeById: 920350',
    '并且 removeByTag attack-sqli',
    '可追加 ruleRemoveByTag=attack-rce',
    '规则id 920350'
  ].join(' ; ');
  const candidates = collectExclusionCandidatesFromFeedbackSuggestion(suggestion);

  assert.deepEqual(
    [...candidates].sort((a, b) => a.removeType.localeCompare(b.removeType) || a.removeValue.localeCompare(b.removeValue)),
    [
      { removeType: 'id', removeValue: '920350' },
      { removeType: 'tag', removeValue: 'attack-rce' },
      { removeType: 'tag', removeValue: 'attack-sqli' }
    ]
  );
  assert.equal(new Set(candidates.map(item => `${item.removeType}:${item.removeValue}`)).size, 3);
  assert.ok(candidates.every(item => item.removeValue.length > 0));
});

test('parseExclusionFromFeedbackSuggestion returns first candidate or empty default', () => {
  const parsed = parseExclusionFromFeedbackSuggestion('removeById 941120');
  assert.deepEqual(parsed, { removeType: 'id', removeValue: '941120' });
  assert.deepEqual(parseExclusionFromFeedbackSuggestion('无明确建议'), { removeType: 'id', removeValue: '' });
});

test('collectExclusionCandidatesFromFeedbackSuggestion supports quoted values and chinese punctuation', () => {
  const suggestion = [
    '建议：移除标签："attack-xss"，',
    '按ID移除：\'942100\'；',
    'ruleRemoveByTag=`attack-lfi`'
  ].join(' ');
  const candidates = collectExclusionCandidatesFromFeedbackSuggestion(suggestion);

  assert.deepEqual(
    [...candidates].sort((a, b) => a.removeType.localeCompare(b.removeType) || a.removeValue.localeCompare(b.removeValue)),
    [
      { removeType: 'id', removeValue: '942100' },
      { removeType: 'tag', removeValue: 'attack-lfi' },
      { removeType: 'tag', removeValue: 'attack-xss' }
    ]
  );
});

test('collectExclusionCandidatesFromFeedbackSuggestion tolerates dirty fragments and broken tokens', () => {
  const suggestion = [
    '建议 removeByTag：attack-rce；',
    'rule id=9x2134（坏样本）',
    'removeById: 949110###',
    '标签=attack-sqli,,',
    '脏数据<script>alert(1)</script>',
    'fallback id: 950001'
  ].join(' ');

  const candidates = collectExclusionCandidatesFromFeedbackSuggestion(suggestion);
  assert.deepEqual(
    [...candidates].sort((a, b) => a.removeType.localeCompare(b.removeType) || a.removeValue.localeCompare(b.removeValue)),
    [
      { removeType: 'id', removeValue: '949110' },
      { removeType: 'id', removeValue: '950001' },
      { removeType: 'tag', removeValue: 'attack-rce' },
      { removeType: 'tag', removeValue: 'attack-sqli' }
    ]
  );
});

test('parseExclusionCandidateKey rejects malformed values', () => {
  assert.equal(parseExclusionCandidateKey('tag'), null);
  assert.equal(parseExclusionCandidateKey('\u0000attack-sqli'), null);
  assert.equal(parseExclusionCandidateKey('other\u0000attack-sqli'), null);
  assert.equal(parseExclusionCandidateKey('id\u0000'), null);
});

test('mergePolicyFeedbackCheckedRowKeys keeps cross-page selections and updates current page', () => {
  const previous = [1, 2, 8];
  const currentPageIDs = [1, 2, 3];
  const nextPageChecked = [2, 3];
  const merged = mergePolicyFeedbackCheckedRowKeys(previous, currentPageIDs, nextPageChecked);
  assert.deepEqual(merged.sort((a, b) => a - b), [2, 3, 8]);
});
