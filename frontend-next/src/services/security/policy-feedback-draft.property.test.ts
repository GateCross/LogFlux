import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import {
  buildExclusionCandidateKey,
  parseExclusionCandidateKey,
  type ExclusionCandidateRemoveType,
} from './policy-feedback-draft';

/**
 * Property 14：排除项候选键往返一致（高优先级，任务 11.5）。
 *
 * 被测纯逻辑：`buildExclusionCandidateKey` / `parseExclusionCandidateKey`
 * （`policy-feedback-draft.ts`，分隔符为 `\u0000`）。
 *
 * 实现要点（影响断言）：
 *  - 往返：`parse(build(removeType, removeValue))` 会对 removeValue 执行 `trim()`，
 *    因此期望候选的 removeValue 取 `removeValue.trim()`；当 removeValue 本身无首尾空白时
 *    即等于原始候选（精确往返）。
 *  - 非法键返回 `null` 的条件：分隔符缺失或位于索引 0（`separatorIndex <= 0`）、
 *    removeType 非 `'id'`/`'tag'`、或值 `trim()` 后为空。
 */

const SEPARATOR = '\u0000';

const removeTypeArb = fc.constantFrom<ExclusionCandidateRemoveType>('id', 'tag');

/** String.prototype.trim 会移除的空白片段（含 Unicode 空白）。 */
const whitespaceArb = fc.constantFrom('', ' ', '  ', '\t', '\n', '\r', '\f', '\v', ' \t\n ', '\u00A0', '\uFEFF');

/** 一个 trim 后保证非空的核心值，覆盖 Unicode。 */
const coreArb = fc.fullUnicodeString({ minLength: 1 }).filter((s) => s.trim().length > 0);

/** 合法 removeValue：核心值附加可选首尾空白，保证 trim 后非空，覆盖首尾空白与 Unicode。 */
const removeValueArb = fc
  .tuple(whitespaceArb, coreArb, whitespaceArb)
  .map(([lead, core, trail]) => `${lead}${core}${trail}`);

type RoundTripCase =
  | { kind: 'valid'; removeType: ExclusionCandidateRemoveType; removeValue: string }
  | { kind: 'invalid'; key: string };

const validCaseArb: fc.Arbitrary<RoundTripCase> = fc.record({
  kind: fc.constant('valid' as const),
  removeType: removeTypeArb,
  removeValue: removeValueArb,
});

/** 控制字符片段（\u0001..\u001F，排除作为分隔符的 \u0000），用于覆盖「含控制字符」非法键。 */
const controlCharsArb = fc
  .array(fc.integer({ min: 1, max: 31 }), { minLength: 1, maxLength: 4 })
  .map((codes) => codes.map((c) => String.fromCharCode(c)).join(''));

/** 非法键：每种构造都保证 `parseExclusionCandidateKey` 返回 null。 */
const invalidCaseArb: fc.Arbitrary<RoundTripCase> = fc.oneof(
  // (a) 缺少分隔符 → indexOf === -1 → null
  fc
    .fullUnicodeString()
    .filter((s) => !s.includes(SEPARATOR))
    .map((s) => ({ kind: 'invalid' as const, key: s })),
  // (b) 分隔符位于索引 0 → separatorIndex <= 0 → null
  fc.fullUnicodeString().map((s) => ({ kind: 'invalid' as const, key: `${SEPARATOR}${s}` })),
  // (c) 合法 removeType 但值为空/纯空白 → trim 后为空 → null
  fc
    .tuple(
      removeTypeArb,
      fc.array(whitespaceArb).map((parts) => parts.join('')),
    )
    .map(([removeType, blankValue]) => ({ kind: 'invalid' as const, key: `${removeType}${SEPARATOR}${blankValue}` })),
  // (d) 未知 removeType（任意非 id/tag 前缀，含空白/Unicode）→ null
  fc
    .tuple(
      fc.fullUnicodeString({ minLength: 1 }).filter((p) => !p.includes(SEPARATOR) && p !== 'id' && p !== 'tag'),
      fc.fullUnicodeString(),
    )
    .map(([removeType, value]) => ({ kind: 'invalid' as const, key: `${removeType}${SEPARATOR}${value}` })),
  // (e) 控制字符 removeType → 未知类型 → null
  fc
    .tuple(controlCharsArb, fc.fullUnicodeString())
    .map(([removeType, value]) => ({ kind: 'invalid' as const, key: `${removeType}${SEPARATOR}${value}` })),
);

const caseArb: fc.Arbitrary<RoundTripCase> = fc.oneof(validCaseArb, invalidCaseArb);

describe('policy-feedback-draft 排除项候选键往返（Property 14）', () => {
  // Feature: frontend-umijs-max-migration, Property 14: 排除项候选键往返一致
  // Validates: Requirements 13.7, 17.1
  it('合法候选 build→parse 往返一致（含 trim 语义）；非法键解析为 null', () => {
    fc.assert(
      fc.property(caseArb, (testCase) => {
        if (testCase.kind === 'valid') {
          const key = buildExclusionCandidateKey(testCase.removeType, testCase.removeValue);
          const parsed = parseExclusionCandidateKey(key);
          expect(parsed).toEqual({
            removeType: testCase.removeType,
            removeValue: testCase.removeValue.trim(),
          });
        } else {
          expect(parseExclusionCandidateKey(testCase.key)).toBeNull();
        }
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
