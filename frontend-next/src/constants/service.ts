/**
 * 后端响应码表（运行时常量，集中管理，可经环境变量覆盖）。
 *
 * 设计依据：design.md「Request_Layer」与「环境变量映射」。
 * 这些集合是 Request_Layer 响应码分类（Req 2.3 / 2.4 / 2.5 / 2.6 / 2.9）的唯一配置来源，
 * 后续也是 Property 1（响应码分类与成功结果提取）的被测对象。
 *
 * Umi 约定：`UMI_APP_` 前缀的环境变量在构建期注入到 `process.env.UMI_APP_*`，
 * 未配置时回退到此处的默认值（与旧 Vue 版 `.env` 取值保持一致）。
 */

/** 将逗号分隔的码字符串解析为去空白、去空项的字符串数组。 */
export function parseCodeList(raw: string | undefined): string[] {
  if (!raw) {
    return [];
  }
  return raw
    .split(',')
    .map(code => code.trim())
    .filter(code => code.length > 0);
}

/**
 * 成功码集合（Req 2.3）。
 *
 * 复刻旧实现 `new Set([String(env.SUCCESS_CODE), '0', '200'])`：
 * 由环境变量 `UMI_APP_SERVICE_SUCCESS_CODE` 配置的成功码，外加固定补充的 `0` 与 `200`。
 * 命中即视为请求成功并提取 `data` 字段。
 */
export const SUCCESS_CODES: ReadonlySet<string> = new Set(
  [String(process.env.UMI_APP_SERVICE_SUCCESS_CODE ?? '200'), '0', '200'].filter(Boolean),
);

/**
 * 登出码集合（Req 2.4）：默认 `3000,3001`。
 * 命中即触发登出并清空鉴权状态。
 */
export const LOGOUT_CODES: ReadonlySet<string> = new Set(
  parseCodeList(process.env.UMI_APP_SERVICE_LOGOUT_CODES ?? '3000,3001'),
);

/**
 * 弹窗登出码集合（Req 2.5）：默认 `7777,7778`。
 * 命中即展示一次性错误弹窗，确认后执行登出。
 */
export const MODAL_LOGOUT_CODES: ReadonlySet<string> = new Set(
  parseCodeList(process.env.UMI_APP_SERVICE_MODAL_LOGOUT_CODES ?? '7777,7778'),
);

/**
 * 令牌过期码集合（Req 2.6）：默认 `9999,9998,3333`。
 * 命中即触发一次令牌刷新，并在成功后对原请求至多重放一次。
 *
 * 注意：`/api/refreshToken` 自身不可返回属于本集合的码，否则会陷入死循环；
 * 刷新接口失效应返回登出码或弹窗登出码。
 */
export const EXPIRED_TOKEN_CODES: ReadonlySet<string> = new Set(
  parseCodeList(process.env.UMI_APP_SERVICE_EXPIRED_TOKEN_CODES ?? '9999,9998,3333'),
);

/** HTTP 未授权状态码（Req 2.6）：等同令牌过期处理。 */
export const UNAUTHORIZED_CODE = '401';
