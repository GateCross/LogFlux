/**
 * WAF 可选接口 404/405 静默忽略属性测试（任务 2.6）。
 *
 * 对应 design.md 的 Correctness Property 3，验证 `isSilentlyIgnored(url, status)`
 * 这一「框架无关纯逻辑」满足「当且仅当」语义：
 *   返回 true  ⟺  URL 命中 WAF 可选接口白名单（含带 / 不带 `/api` 前缀变体）
 *                 且 HTTP 状态码为 404 或 405。
 * 其余任意 URL 或状态码组合均不被静默。
 *
 * 测试以「按构造给出基准真值」的方式生成输入：每个用例自带 `hits`（是否命中白名单）
 * 与 `silent`（状态码是否属于 404/405）两个独立标记，断言
 *   isSilentlyIgnored(url, status) === (hits && silent)
 * 期望值来自生成它的分区（与被测实现无关），避免测试沦为实现的镜像。
 */
import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import { WAF_OPTIONAL_PATHS, isSilentlyIgnored } from './waf-whitelist';

/**
 * 白名单基准路径的全部变体：不带 `/api` 前缀的规范形式 + 带 `/api` 前缀的形式。
 * 二者均应被视为「命中」。
 */
const WHITELIST_VARIANTS: readonly string[] = [
  ...WAF_OPTIONAL_PATHS,
  ...WAF_OPTIONAL_PATHS.map(p => `/api${p}`),
];

/**
 * 所有白名单基准路径都包含公共片段 `/caddy/waf/`。
 * 因此：一个字符串只要不含 `/caddy/waf/`，就一定不含任何完整白名单路径。
 * 这一更强（更简单）的充分条件用于「按构造」保证非命中分区的基准真值，
 * 而非复用被测实现的匹配逻辑，避免循环论证。
 */
const COMMON_FRAGMENT = '/caddy/waf/';

/** 不含 `?`、`#` 的「安全前缀」：保证其后拼接的白名单路径不会被查询串/锚点裁掉。 */
const arbSafePrefix = fc
  .string({ maxLength: 16 })
  .filter(s => !s.includes('?') && !s.includes('#'));

/**
 * 命中白名单的 URL 生成器：`安全前缀 + 白名单变体 + 任意尾串`。
 * 前缀不含 `?`/`#`，白名单变体本身不含 `?`/`#`，故去除查询串/锚点后的路径
 * 必然仍包含该白名单变体 → 命中（hits = true）。尾串可为任意内容（含 `?`/`#`）。
 */
const arbWhitelistUrl: fc.Arbitrary<string> = fc
  .record({
    prefix: arbSafePrefix,
    base: fc.constantFrom(...WHITELIST_VARIANTS),
    tail: fc.string(),
  })
  .map(({ prefix, base, tail }) => `${prefix}${base}${tail}`);

/**
 * 不命中白名单的 URL 生成器，覆盖多种典型非命中形态：
 * - 现实中存在但不构成完整白名单路径的接口路径（含「近似但不相等」的诱饵）；
 * - 白名单路径仅出现在查询串中（`?` 之前的路径部分不含白名单 → 不应被静默）；
 * - 随机字符串（强制不含公共片段 `/caddy/waf/`，故必不含任何白名单路径）；
 * - 空串 / null / undefined（URL 缺失，恒不命中）。
 */
const arbNonWhitelistUrl: fc.Arbitrary<string | null | undefined> = fc.oneof(
  fc.constantFrom(
    '/api/login',
    '/api/user/info',
    '/caddy/server/list',
    '/caddy/waf/engine', // 前缀但缺少 /status|check
    '/caddy/waf/integration', // 前缀但缺少 /status|apply
    '/caddy/waf/engine/statux', // 近似诱饵：末字符不同，不含完整白名单路径
    '/caddy/waf/engine/stat', // 近似诱饵：白名单路径的真前缀
  ).filter(s => !WHITELIST_VARIANTS.some(w => s.includes(w))),
  // 白名单路径只出现在查询串里：路径部分（? 之前）为安全前缀，必不命中。
  fc
    .record({ prefix: arbSafePrefix, base: fc.constantFrom(...WHITELIST_VARIANTS), tail: fc.string() })
    .map(({ prefix, base, tail }) => `${prefix}?redirect=${base}${tail}`),
  // 随机字符串：排除公共片段即可保证不含任何白名单路径。
  fc.string().filter(s => !s.includes(COMMON_FRAGMENT)),
  fc.constantFrom('', null, undefined),
);

/** 命中标记齐全的 URL 用例（hits 来自生成它的分区，与实现无关）。 */
const arbUrlCase: fc.Arbitrary<{ url: string | null | undefined; hits: boolean }> = fc.oneof(
  arbWhitelistUrl.map(url => ({ url, hits: true })),
  arbNonWhitelistUrl.map(url => ({ url, hits: false })),
);

/** 静默状态码（404/405）与非静默状态码（其余整数、缺失值）。 */
const arbStatusCase: fc.Arbitrary<{ status: number | null | undefined; silent: boolean }> = fc.oneof(
  fc.constantFrom(404, 405).map(status => ({ status, silent: true })),
  fc
    .oneof(
      fc.constantFrom(200, 201, 204, 301, 400, 401, 403, 500, 502, 0),
      fc.integer().filter(n => n !== 404 && n !== 405),
      fc.constantFrom(null, undefined),
    )
    .map(status => ({ status, silent: false })),
);

describe('waf-whitelist — WAF 可选接口 404/405 静默忽略', () => {
  // Feature: frontend-umijs-max-migration, Property 3: WAF 可选接口 404/405 静默忽略
  // Validates: Requirements 2.8
  it('当且仅当 URL 命中白名单且状态码为 404/405 时静默忽略', () => {
    fc.assert(
      fc.property(arbUrlCase, arbStatusCase, ({ url, hits }, { status, silent }) => {
        const actual = isSilentlyIgnored(url, status);

        // iff：静默 ⟺ 命中白名单 且 状态码属 404/405。
        expect(actual).toBe(hits && silent);

        // 确定性：纯函数对相同入参恒返回相同结果。
        expect(isSilentlyIgnored(url, status)).toBe(actual);
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
