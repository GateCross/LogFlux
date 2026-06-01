/**
 * 后端响应码分类与成功结果提取纯逻辑（任务 2.1 / Req 2.3、2.4、2.5、2.6、2.9）。
 *
 * 设计依据：design.md「Request_Layer」之「成功判定」与「响应码分支（onBackendFail）」，
 * 以及 Correctness Property 1（响应码分类与成功结果提取）。
 *
 * 本模块为「框架无关纯逻辑」：仅依据后端响应码（与可选的 `data`）做确定性分类，
 * 不依赖 axios、不读取/修改任何全局状态、无副作用，便于被 Property 1 的属性测试（任务 2.2）覆盖。
 * 真正的 axios 拦截器接线（含「HTTP 状态 401 → 过期处理」「单飞刷新 / 至多重放一次」「弹窗 / 登出副作用」）
 * 在任务 2.7 完成；本模块只负责「按业务 code 落到哪个分支」这一纯判定。
 *
 * 码集合的唯一来源是 `@/constants/service`（任务 1.2 建立），本模块不重新定义任何码集合。
 *
 * 行为对齐旧 Vue 版 `frontend/src/service/request/index.ts` 的 `isBackendSuccess` 与
 * `onBackendFail`：
 *  - `isBackendSuccess`：`code` 命中成功码集合 → 成功，并以 `response.data.data` 作为结果；
 *    `code` 缺失（`undefined`）视为非成功。
 *  - `onBackendFail`：依次判定 `logoutCodes → modalLogoutCodes → expiredTokenCodes`，
 *    其余（含缺失 code）为普通失败。
 *
 * 说明（与旧实现的边界对齐）：
 *  - 旧实现还在 `onBackendFail` 中单独处理「HTTP 状态 401」（位于 logout 与 modalLogout 之间），
 *    那是基于 **HTTP 状态码** 而非业务 `code` 的判定，属于 axios 层职责（任务 2.7）。
 *    因此本纯函数 **不** 依据业务 `code === '401'` 做过期分类——这与 Property 1 的口径一致：
 *    分类仅由「成功码 / 登出码 / 弹窗登出码 / 过期码」四个配置集合 + 普通失败构成。
 */
import {
  EXPIRED_TOKEN_CODES,
  LOGOUT_CODES,
  MODAL_LOGOUT_CODES,
  SUCCESS_CODES,
} from '@/constants/service';

/**
 * 响应分类类别（与配置的码集合一一对应，外加普通失败兜底）。
 *
 * - `success`：`code` 命中成功码集合（Req 2.3）。
 * - `logout`：`code` 命中登出码集合（Req 2.4）。
 * - `modalLogout`：`code` 命中弹窗登出码集合（Req 2.5）。
 * - `expired`：`code` 命中令牌过期码集合（Req 2.6）。
 * - `failure`：其余任意 `code`（含缺失 / 未知）→ 普通失败（Req 2.9）。
 */
export type ResponseCategory = 'success' | 'logout' | 'modalLogout' | 'expired' | 'failure';

/**
 * 分类结果（可辨识联合 / discriminated union）。
 *
 * 仅在 `success` 分支携带从响应中提取的 `data`；其余分支不含 `data`，
 * 由调用方（axios 层，任务 2.7）按类别执行对应副作用（登出 / 弹窗 / 刷新重放 / 报错）。
 */
export type ClassifiedResponse<T = unknown> =
  | { category: 'success'; data: T }
  | { category: 'logout' }
  | { category: 'modalLogout' }
  | { category: 'expired' }
  | { category: 'failure' };

/** 仅参与分类所需的后端响应字段（`code` 可缺失；成功时提取 `data`）。 */
export type ClassifiableResponse<T = unknown> = {
  /** 业务响应码（可能缺失）。 */
  code?: string | number | null;
  /** 响应数据（仅成功分支提取）。 */
  data?: T;
};

/**
 * 将后端业务 `code` 归类到 {@link ResponseCategory}（不涉及 `data`）。
 *
 * 这是 Property 1「一一对应、互不重叠、全覆盖」的核心：
 *  - **全覆盖（total）**：任意输入（含 `null` / `undefined` / 未知码）都恰好返回一个类别，
 *    缺失或未匹配任何集合的 `code` 一律归为 `failure`（Req 2.9）。
 *  - **互不重叠（确定性）**：采用固定优先级 `success → logout → modalLogout → expired → failure`
 *    判定。默认配置下四个集合本就互斥；即便环境变量误配导致集合相交，本优先级也保证
 *    每个 `code` 仅映射到唯一类别，结果稳定可预测。
 *
 * 归一化方式对齐旧实现：缺失 `code`（`null` / `undefined`）直接判为 `failure`；
 * 其余 `code` 一律 `String(code)` 后做集合成员判定（兼容后端返回数字或字符串码）。
 */
export function classifyCode(code: string | number | null | undefined): ResponseCategory {
  if (code === null || code === undefined) {
    return 'failure';
  }

  const normalized = String(code);

  if (SUCCESS_CODES.has(normalized)) {
    return 'success';
  }
  if (LOGOUT_CODES.has(normalized)) {
    return 'logout';
  }
  if (MODAL_LOGOUT_CODES.has(normalized)) {
    return 'modalLogout';
  }
  if (EXPIRED_TOKEN_CODES.has(normalized)) {
    return 'expired';
  }

  return 'failure';
}

/**
 * 对后端响应进行分类，并在成功时提取 `data` 作为结果。
 *
 * - 成功（`code` ∈ 成功码集合）→ `{ category: 'success', data }`（Req 2.3）。
 * - 登出 / 弹窗登出 / 过期 → 对应类别（Req 2.4 / 2.5 / 2.6），不携带 `data`。
 * - 其余（含 `code` 缺失）→ `{ category: 'failure' }`（Req 2.9）。
 *
 * 纯函数：对相同入参恒返回相同结果，无副作用。`response` 为 `null` / `undefined` 时
 * 等价于无有效 `code`，归为普通失败。
 */
export function classifyResponse<T = unknown>(
  response: ClassifiableResponse<T> | null | undefined,
): ClassifiedResponse<T> {
  const category = classifyCode(response?.code);

  if (category === 'success') {
    return { category: 'success', data: response?.data as T };
  }

  return { category };
}
