/**
 * 响应码分类与成功结果提取属性测试（任务 2.2）。
 *
 * 对应 design.md 的 Correctness Property 1，验证 `classifyCode` / `classifyResponse`
 * 这一「框架无关纯逻辑」满足：分类全覆盖（total）、与配置码集合一一对应、
 * 互不重叠（固定优先级 success→logout→modalLogout→expired→failure）、成功时提取 `data`。
 *
 * 码集合直接取自被测对象的唯一来源 `@/constants/service`，因此本测试对环境变量配置保持鲁棒；
 * 生成器覆盖：四个配置集合各自的码、未知码、数字与字符串两种形态、以及缺失 code（null/undefined）。
 */
import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import {
  EXPIRED_TOKEN_CODES,
  LOGOUT_CODES,
  MODAL_LOGOUT_CODES,
  SUCCESS_CODES,
} from '@/constants/service';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import {
  classifyCode,
  classifyResponse,
  type ResponseCategory,
} from './classify';

/** 全部合法分类类别（用于「全覆盖 / total」断言）。 */
const ALL_CATEGORIES: ReadonlySet<ResponseCategory> = new Set<ResponseCategory>([
  'success',
  'logout',
  'modalLogout',
  'expired',
  'failure',
]);

/** 任意配置集合中出现过的全部码（用于「未知码」生成器的排除集）。 */
const ALL_CONFIGURED_CODES = new Set<string>([
  ...SUCCESS_CODES,
  ...LOGOUT_CODES,
  ...MODAL_LOGOUT_CODES,
  ...EXPIRED_TOKEN_CODES,
]);

/**
 * 将一个字符串码扩展为「字符串 + 等价数字」两种形态（覆盖后端可能返回数字或字符串码）。
 * 仅当该字符串是某数字的规范字符串表示时才追加数字形态（避免 '00'、'1e3' 等被错误数字化）。
 */
function codeVariants(code: string): Array<string | number> {
  const variants: Array<string | number> = [code];
  if (/^-?\d+$/.test(code) && String(Number(code)) === code) {
    variants.push(Number(code));
  }
  return variants;
}

/** 由某配置集合构造「该集合全部码（含数字变体）」的生成器。 */
function arbCodeFromSet(set: ReadonlySet<string>) {
  return fc.constantFrom(...[...set].flatMap(codeVariants));
}

/** 待分类输入与其「独立基准」期望类别（期望由生成它的分区决定，与实现无关）。 */
type ClassifyCase = {
  code: string | number | null | undefined;
  expected: ResponseCategory;
};

/** 已知码不会出现的「未知码」字符串（含空串等边界）。 */
const arbUnknownString = fc.string().filter(s => !ALL_CONFIGURED_CODES.has(s));

/** 已知码不会出现的「未知码」数字（含负数、0 之外的大数、NaN 等）。 */
const arbUnknownNumber = fc
  .oneof(fc.integer(), fc.double(), fc.constant(Number.NaN))
  .filter(n => !ALL_CONFIGURED_CODES.has(String(n)));

/** 覆盖所有分区的输入生成器，每个分区携带其独立期望类别。 */
const arbClassifyCase: fc.Arbitrary<ClassifyCase> = fc.oneof(
  arbCodeFromSet(SUCCESS_CODES).map(code => ({ code, expected: 'success' as const })),
  arbCodeFromSet(LOGOUT_CODES).map(code => ({ code, expected: 'logout' as const })),
  arbCodeFromSet(MODAL_LOGOUT_CODES).map(code => ({ code, expected: 'modalLogout' as const })),
  arbCodeFromSet(EXPIRED_TOKEN_CODES).map(code => ({ code, expected: 'expired' as const })),
  arbUnknownString.map(code => ({ code, expected: 'failure' as const })),
  arbUnknownNumber.map(code => ({ code, expected: 'failure' as const })),
  fc.constantFrom(null, undefined).map(code => ({ code, expected: 'failure' as const })),
);

describe('classify — 响应码分类与成功结果提取', () => {
  // 结构前置：默认配置下四个码集合两两互不重叠，是「一一对应 / 互不重叠」的前提（Property 1）。
  it('四个配置码集合两两互不重叠', () => {
    const sets: Array<[string, ReadonlySet<string>]> = [
      ['success', SUCCESS_CODES],
      ['logout', LOGOUT_CODES],
      ['modalLogout', MODAL_LOGOUT_CODES],
      ['expired', EXPIRED_TOKEN_CODES],
    ];
    for (let i = 0; i < sets.length; i += 1) {
      for (let j = i + 1; j < sets.length; j += 1) {
        const overlap = [...sets[i][1]].filter(code => sets[j][1].has(code));
        expect(overlap, `${sets[i][0]} 与 ${sets[j][0]} 不应有交集`).toEqual([]);
      }
    }
  });

  // Feature: frontend-umijs-max-migration, Property 1: 响应码分类与成功结果提取
  // Validates: Requirements 2.3, 2.4, 2.5, 2.6, 2.9
  it('分类全覆盖、与码集合一一对应、互不重叠，且成功时提取 data', () => {
    fc.assert(
      fc.property(arbClassifyCase, fc.anything(), ({ code, expected }, data) => {
        const actual = classifyCode(code);

        // 全覆盖（total）：任意输入都恰好落到一个合法类别，绝不抛错或返回越界值。
        expect(ALL_CATEGORIES.has(actual)).toBe(true);

        // 一一对应：来自某配置集合的码归到对应类别；未知码 / 缺失 code 归为 failure（Req 2.3-2.6、2.9）。
        expect(actual).toBe(expected);

        // 确定性（互不重叠的体现）：同一输入恒返回同一唯一类别。
        expect(classifyCode(code)).toBe(actual);

        // classifyResponse 的类别与 classifyCode 一致，且仅成功分支提取并原样携带 data（Req 2.3）。
        const result = classifyResponse({ code, data });
        expect(result.category).toBe(expected);
        if (expected === 'success') {
          expect(result).toStrictEqual({ category: 'success', data });
          // data 被原样提取（同值 / 同引用），不被改写。
          expect((result as { category: 'success'; data: unknown }).data).toBe(data);
        } else {
          // 非成功分支不携带 data，由调用方按类别执行副作用。
          expect('data' in result).toBe(false);
        }
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
