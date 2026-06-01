/**
 * WAF 可选接口 404/405 静默忽略白名单（任务 2.5 / Req 2.8）。
 *
 * 设计依据：design.md「Request_Layer」之「WAF 可选接口白名单」。
 * 后端的 WAF 引擎 / 集成相关接口为「可选能力」：在未启用对应模块的环境下，
 * 这些端点可能返回 404（未注册路由）或 405（方法不允许）。前端不应将其视为错误弹窗，
 * 而应静默忽略，避免对用户造成无意义的报错打扰。
 *
 * 本模块为「框架无关纯逻辑」：仅依据 URL 与 HTTP 状态码做确定性判定，不依赖 axios、
 * 不读取任何全局状态，便于被 Property 3（WAF 可选接口 404/405 静默忽略）的属性测试覆盖。
 * axios 拦截器接线在任务 2.7 完成。
 *
 * 行为对齐旧 Vue 版 `frontend/src/service/request/index.ts` 中的 `isWafEngineOptionalApi`
 * 判定逻辑（8 个路径变体 + 404/405）。
 */

/** 被静默忽略的 WAF 可选接口的 HTTP 状态码集合（Req 2.8）。 */
const SILENT_STATUS_CODES: ReadonlySet<number> = new Set([404, 405]);

/**
 * WAF 可选接口路径白名单（不含 `/api` 前缀的规范形式）。
 *
 * 旧实现同时列出了带 `/api` 前缀（如 `/api/caddy/waf/engine/status`）与不带前缀
 * （如 `/caddy/waf/engine/status`）两类变体。由于带前缀变体必然包含不带前缀变体作为子串，
 * 故此处只需以「不带前缀」的形式做子串匹配，即可同时覆盖两类变体，避免重复枚举。
 */
export const WAF_OPTIONAL_PATHS: readonly string[] = [
  '/caddy/waf/engine/status',
  '/caddy/waf/engine/check',
  '/caddy/waf/integration/status',
  '/caddy/waf/integration/apply',
];

/**
 * 去除 URL 中的查询串（`?...`）与片段（`#...`），仅保留路径部分。
 *
 * 这样可以「谨慎处理查询串」：避免查询参数取值（例如 `?redirect=/caddy/waf/engine/status`）
 * 被误判为命中白名单路径；同时也容忍尾部的查询串 / 锚点差异。
 */
function stripQueryAndHash(url: string): string {
  const queryIndex = url.indexOf('?');
  const withoutQuery = queryIndex >= 0 ? url.slice(0, queryIndex) : url;
  const hashIndex = withoutQuery.indexOf('#');
  return hashIndex >= 0 ? withoutQuery.slice(0, hashIndex) : withoutQuery;
}

/**
 * 判定某请求是否应被「静默忽略」（不弹错误提示）。
 *
 * 当且仅当满足以下两个条件时返回 `true`：
 * 1. 请求 URL 的路径部分命中 WAF 可选接口白名单（{@link WAF_OPTIONAL_PATHS}，
 *    含带 `/api` 前缀与不带前缀两类变体）；
 * 2. HTTP 状态码为 404 或 405。
 *
 * 其余任意 URL 或状态码组合均返回 `false`。函数为纯函数，对相同入参恒返回相同结果。
 *
 * @param url 请求 URL（可为相对路径或绝对 URL；可包含查询串 / 锚点）。
 * @param status HTTP 响应状态码。
 */
export function isSilentlyIgnored(url: string | null | undefined, status: number | null | undefined): boolean {
  if (typeof status !== 'number' || !SILENT_STATUS_CODES.has(status)) {
    return false;
  }

  if (typeof url !== 'string' || url.length === 0) {
    return false;
  }

  const path = stripQueryAndHash(url);
  return WAF_OPTIONAL_PATHS.some(whitelisted => path.includes(whitelisted));
}
