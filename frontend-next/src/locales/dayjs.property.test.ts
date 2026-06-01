import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import { LANGS, type LangType } from '@/utils/preferences';
import { normalizeLang, DEFAULT_LANG } from './dayjs';

/**
 * Property 7：语言归一化与回退（任务 5.2 / Req 5.6）。
 *
 * 被测纯逻辑：`normalizeLang(input)`（`src/locales/dayjs.ts`）。
 *
 * 规约（design.md「Correctness Properties ▸ Property 7」）：对任意输入（含 `undefined` /
 * `null` / 空串 / 非法字符串 / 大小写变体 / 合法语言码），输出恒属于 `{ 'zh-CN', 'en-US' }`；
 * 当且仅当输入是有效语言码时返回其自身，否则回退默认语言 `zh-CN`。
 *
 * 测试思路（避免自我循环 oracle）：构造「按构造已知合法/非法」的取值——
 *  - 合法：恰为枚举集合 `LANGS` 中的语言码，期望映射为自身；
 *  - 非法：`undefined` / `null` / 空串 / 随机非枚举字符串 / 大小写变体 / 非字符串类型，
 *    期望回退 `DEFAULT_LANG`。
 * 再断言：输出恒在合法集合内，且实际输出等于按构造预期的输出。
 */

/** 合法语言码集合（与实现判定一致，大小写敏感）。 */
const VALID_LANGS = LANGS as readonly string[];

/** 单个输入取值及其（按构造已知的）归一化预期输出。 */
interface LangCase {
  input: unknown;
  expected: LangType;
}

/** 合法：枚举内语言码 → 映射为自身。 */
const validLangArb: fc.Arbitrary<LangCase> = fc
  .constantFrom<LangType>(...(LANGS as LangType[]))
  .map((lang) => ({ input: lang, expected: lang }));

/** 非法：缺省 / 空 / 大小写变体 / 非枚举字符串 / 非字符串 → 回退默认语言。 */
const invalidLangArb: fc.Arbitrary<LangCase> = fc
  .oneof(
    // 缺省 / 空
    fc.constantFrom(undefined, null, ''),
    // 大小写变体（实现大小写敏感，故非法）
    fc.constantFrom('zh-cn', 'ZH-CN', 'Zh-Cn', 'en-us', 'EN-US', 'En-Us'),
    // 形近但非枚举
    fc.constantFrom('zh', 'en', 'zh_CN', 'en_US', 'zh-TW', 'fr-FR', ' zh-CN', 'zh-CN '),
    // 任意随机字符串，过滤掉恰好命中合法码的情形
    fc.string().filter((s) => !VALID_LANGS.includes(s)),
    // 非字符串类型
    fc.constantFrom(0, 1, true, false, NaN, {}, [], Symbol('zh-CN') as unknown),
  )
  .map((input) => ({ input, expected: DEFAULT_LANG }));

const langCaseArb: fc.Arbitrary<LangCase> = fc.oneof(validLangArb, invalidLangArb);

describe('normalizeLang 语言归一化与回退（Property 7）', () => {
  // Feature: frontend-umijs-max-migration, Property 7: 语言归一化与回退
  // Validates: Requirements 5.6
  it('输出恒属于 {zh-CN, en-US}；合法码映射自身，缺省/非法回退 zh-CN', () => {
    fc.assert(
      fc.property(langCaseArb, ({ input, expected }) => {
        const output = normalizeLang(input);

        // 全域不变式：输出恒为合法语言码。
        expect(VALID_LANGS).toContain(output);

        // 精确性：合法码映射自身，其余回退 DEFAULT_LANG。
        expect(output).toBe(expected);

        // 回退目标始终是 zh-CN。
        if (!(typeof input === 'string' && VALID_LANGS.includes(input))) {
          expect(output).toBe('zh-CN');
        }
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
