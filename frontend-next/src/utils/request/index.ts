/**
 * 统一请求层装配（任务 2.7 / Req 2.6、2.10、2.11、3.7、3.8、3.9）。
 *
 * 设计依据：design.md「Request_Layer」与「请求与鉴权流程」，复刻旧 Vue 版
 * `@sa/axios` 的 `createFlatRequest` 语义 + `frontend/src/service/request/index.ts` 的拦截器编排。
 *
 * 本模块把前序任务沉淀的三块「框架无关纯逻辑」装配进一个单例 axios 实例：
 *  - `classify.ts`（任务 2.1）：响应码分类与成功结果提取（成功/登出/弹窗登出/过期/普通失败）。
 *  - `err-msg.ts`（任务 2.3）：错误消息去重栈（同消息提示未关闭前不重复弹出）。
 *  - `waf-whitelist.ts`（任务 2.5）：WAF 可选接口 404/405 静默忽略白名单。
 *
 * 装配后暴露扁平返回的 `request<T>(config): Promise<{ data, error }>`（FlatResponse），
 * 与旧调用方签名保持一致，降低 Service_API 迁移成本。
 *
 * ──────────────────────────────────────────────────────────────────────────
 * 副作用边界（依赖注入）
 * ──────────────────────────────────────────────────────────────────────────
 * 鉴权读写、登出、刷新令牌、弹窗等副作用与具体框架/存储实现强相关，且其落地模块
 * （`src/utils/storage.ts` 任务 3.4、`src/models/auth.ts` 任务 4.x、AntD `Modal` 运行时
 * 任务 7.4）尚未建立。为保持本模块「可独立编译、可被属性/单元测试直接覆盖」，
 * 沿用 `err-msg.ts` 既有的「可注入显示器」约定，把这些副作用抽象为可注入的适配器：
 *  - {@link RequestAuthAdapter}：令牌读写与登出（默认 localStorage 兜底实现，应用启动后注入真实实现）。
 *  - {@link RefreshTokenCaller}：调用 `POST /api/refreshToken` 的刷新动作（默认走裸 axios 直连）。
 *  - {@link ModalLogoutHandler}：弹窗登出的 UI 展示（默认立即确认，应用注入 AntD `Modal.error`）。
 *
 * ──────────────────────────────────────────────────────────────────────────
 * 分支顺序（与 design.md / 旧实现一致）
 * ──────────────────────────────────────────────────────────────────────────
 * 后端业务码（HTTP 2xx 响应体）在响应成功拦截器中按
 *   `成功码 → logoutCodes → modalLogoutCodes → expiredTokenCodes → 普通错误`
 * 由 `classifyResponse` 统一裁决；HTTP 传输层的 `401`（axios 默认会 reject）在响应失败拦截器
 * 中独立处理（刷新 + 至多重放一次，失败登出），二者合起来即任务描述的
 *   `logoutCodes → HTTP 401 → modalLogoutCodes → expiredTokenCodes → 普通错误`。
 */
import axios, { AxiosError } from 'axios';
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { SERVICE_BASE_URL } from '@/constants/app';
import { EXPIRED_TOKEN_CODES, LOGOUT_CODES, MODAL_LOGOUT_CODES } from '@/constants/service';
import { classifyResponse } from './classify';
import { showErrorMsg } from './err-msg';
import { isSilentlyIgnored } from './waf-whitelist';

/** 默认请求超时（毫秒）（Req 2.11、3.7）。 */
export const REQUEST_TIMEOUT = 10000;

/** 令牌刷新接口路径（Req 3.7）。 */
export const REFRESH_TOKEN_URL = '/api/refreshToken';

/** 单飞刷新 promise 的置空延时（毫秒）（Req 3.8）。 */
export const REFRESH_TOKEN_PROMISE_CLEAR_DELAY = 1000;

/** 后端业务失败的合成错误码（与旧 `@sa/axios` 的 `BACKEND_ERROR_CODE` 对齐）。 */
export const BACKEND_ERROR_CODE = 'BACKEND_ERROR';

/** HTTP 未授权状态码（Req 2.6）：触发刷新 + 至多重放一次。 */
const HTTP_UNAUTHORIZED = 401;

/**
 * 扁平响应结构（design.md「Request_Layer」接口签名）。
 *
 * - 成功：`{ data: <后端 data 字段>, error: null }`。
 * - 失败（业务失败码 / HTTP 错误 / 超时 / 网络错误）：`{ data: null, error }`。
 * - `response` 为可选的原始 axios 响应（便于调用方读取 HTTP 状态等），不影响 `{ data, error }` 契约。
 */
export interface FlatResponse<T = unknown> {
  /** 成功时为后端 `data` 字段；失败时为 `null`。 */
  data: T | null;
  /** 失败时为错误对象；成功时为 `null`。 */
  error: Error | null;
  /** 原始 axios 响应（可能缺失，如网络错误时）。 */
  response?: AxiosResponse<BackendResponse<T>>;
}

/**
 * 请求实例内部状态（design.md「RequestInstanceState」）。
 *
 * - `errMsgStack`：错误消息去重栈，复用 `err-msg.ts` 的去重逻辑（Req 2.7）。
 * - `refreshTokenPromise`：在途刷新单飞 promise（Req 3.8）。
 */
export interface RequestInstanceState {
  errMsgStack: string[];
  refreshTokenPromise: Promise<boolean> | null;
}

/** 携带「至多重放一次」标记的请求配置。 */
interface RetriableConfig extends AxiosRequestConfig {
  /** 标记该请求已因刷新令牌而重放过一次，避免无限重放（Req 2.6）。 */
  __isRetry?: boolean;
}

/**
 * 鉴权适配器（可注入的副作用边界）。
 *
 * 默认实现为最小化的 localStorage 兜底（见 {@link createLocalStorageAuthAdapter}）；
 * 应用启动后由 Auth_Module（任务 4.x）经 {@link setRequestAuthAdapter} 注入与真实存储一致的实现。
 */
export interface RequestAuthAdapter {
  /** 读取访问令牌（Req 2.2）。 */
  getToken(): string | null;
  /** 读取刷新令牌（Req 3.7）。 */
  getRefreshToken(): string | null;
  /** 持久化新的访问令牌（刷新成功后，Req 3.7）。 */
  setToken(token: string | null): void;
  /** 持久化新的刷新令牌（刷新成功后，Req 3.7）。 */
  setRefreshToken(token: string | null): void;
  /** 清空鉴权状态并登出（Req 2.4、2.10、3.9）。 */
  onLogout(): void;
}

/**
 * 刷新令牌调用器（可注入）。
 *
 * 返回新的令牌对表示刷新成功；返回 `null` 表示刷新失败（凭据失效 / 超时 / 网络错误）。
 * 默认实现 {@link defaultRefreshTokenCaller} 走裸 axios 直连 `POST /api/refreshToken`，
 * 不经过本模块的响应拦截器，避免刷新自身再次触发过期处理而陷入循环（Req 3.7、3.9）。
 */
export type RefreshTokenCaller = () => Promise<Api.Auth.LoginToken | null>;

/**
 * 弹窗登出处理器（可注入的 UI 边界，Req 2.5）。
 *
 * 实现方展示一次性错误弹窗，并在用户确认 / 关闭时回调一次 `onConfirm` 以执行登出。
 * 默认实现立即回调 `onConfirm`（无 UI 兜底），应用运行时注入 AntD `Modal.error`。
 */
export type ModalLogoutHandler = (message: string, onConfirm: () => void) => void;

/** 从后端响应体取错误消息（`message` 优先于兼容字段 `msg`）。 */
function getBackendMessage(data: Partial<BackendResponse> | undefined | null, fallback = ''): string {
  return data?.message || data?.msg || fallback;
}

/** localStorage 兜底鉴权适配器使用的存储前缀（与 `UMI_APP_STORAGE_PREFIX` 一致）。 */
const STORAGE_PREFIX = process.env.UMI_APP_STORAGE_PREFIX ?? 'LF_';

/**
 * 创建最小化的 localStorage 兜底鉴权适配器。
 *
 * 仅做原始字符串读写（令牌本身即字符串），用于应用注入真实适配器之前的早期阶段与测试兜底。
 * 真实的存储序列化格式由 storage.ts（任务 3.4）/ Auth_Module（任务 4.x）统一，并经
 * {@link setRequestAuthAdapter} 覆盖此默认实现。
 */
export function createLocalStorageAuthAdapter(prefix: string = STORAGE_PREFIX): RequestAuthAdapter {
  const keyOf = (key: string) => `${prefix}${key}`;
  const safeGet = (key: string): string | null => {
    try {
      return typeof localStorage !== 'undefined' ? localStorage.getItem(keyOf(key)) : null;
    } catch {
      return null;
    }
  };
  const safeSet = (key: string, value: string | null): void => {
    try {
      if (typeof localStorage === 'undefined') return;
      if (value === null) {
        localStorage.removeItem(keyOf(key));
      } else {
        localStorage.setItem(keyOf(key), value);
      }
    } catch {
      /* ignore storage errors */
    }
  };
  const safeRemove = (key: string): void => {
    try {
      if (typeof localStorage !== 'undefined') localStorage.removeItem(keyOf(key));
    } catch {
      /* ignore storage errors */
    }
  };

  return {
    getToken: () => safeGet('token'),
    getRefreshToken: () => safeGet('refreshToken'),
    setToken: token => safeSet('token', token),
    setRefreshToken: token => safeSet('refreshToken', token),
    onLogout: () => {
      safeRemove('token');
      safeRemove('refreshToken');
    },
  };
}

/** 当前生效的鉴权适配器（默认 localStorage 兜底；可注入覆盖）。 */
let authAdapter: RequestAuthAdapter = createLocalStorageAuthAdapter();

/** 当前生效的弹窗登出处理器（默认立即确认；可注入覆盖）。 */
let modalLogoutHandler: ModalLogoutHandler = (_message, onConfirm) => {
  onConfirm();
};

/**
 * 默认刷新令牌调用器：裸 axios 直连 `POST /api/refreshToken`（不经过本模块拦截器）。
 *
 * 复刻旧 `shared.ts#handleRefreshToken`：以 `refreshToken` 为请求体；
 * 后端返回成功码则取出新令牌对，否则（含 HTTP 错误 / 超时 / 网络错误）视为刷新失败。
 */
const defaultRefreshTokenCaller: RefreshTokenCaller = async () => {
  const refreshToken = authAdapter.getRefreshToken() ?? '';
  try {
    const resp = await rawRefreshInstance.post<BackendResponse<Api.Auth.LoginToken>>(REFRESH_TOKEN_URL, {
      refreshToken,
    });
    const classified = classifyResponse<Api.Auth.LoginToken>(resp.data);
    if (classified.category === 'success' && classified.data) {
      return classified.data;
    }
  } catch {
    // 网络错误 / 超时 / 非 2xx → 统一视为刷新失败。
  }
  return null;
};

/** 当前生效的刷新令牌调用器（默认裸 axios 直连；可注入覆盖以便测试）。 */
let refreshTokenCaller: RefreshTokenCaller = defaultRefreshTokenCaller;

/** 注入鉴权适配器（应用启动 / 测试用）。 */
export function setRequestAuthAdapter(adapter: RequestAuthAdapter): void {
  authAdapter = adapter;
}

/** 重置鉴权适配器为 localStorage 兜底（主要用于测试隔离）。 */
export function resetRequestAuthAdapter(): void {
  authAdapter = createLocalStorageAuthAdapter();
}

/** 注入刷新令牌调用器（测试 / 自定义刷新策略用）。 */
export function setRefreshTokenCaller(caller: RefreshTokenCaller): void {
  refreshTokenCaller = caller;
}

/** 重置刷新令牌调用器为默认实现（主要用于测试隔离）。 */
export function resetRefreshTokenCaller(): void {
  refreshTokenCaller = defaultRefreshTokenCaller;
}

/** 注入弹窗登出处理器（应用运行时注入 AntD Modal）。 */
export function setModalLogoutHandler(handler: ModalLogoutHandler): void {
  modalLogoutHandler = handler;
}

/** 重置弹窗登出处理器为默认实现（主要用于测试隔离）。 */
export function resetModalLogoutHandler(): void {
  modalLogoutHandler = (_message, onConfirm) => {
    onConfirm();
  };
}

/** 请求层单例内部状态（错误去重栈 + 在途刷新单飞 promise）。 */
export const requestState: RequestInstanceState = {
  errMsgStack: [],
  refreshTokenPromise: null,
};

/** 重置请求层内部状态（主要用于测试隔离）。 */
export function resetRequestState(): void {
  requestState.errMsgStack = [];
  requestState.refreshTokenPromise = null;
}

/**
 * 刷新令牌（复刻旧 `shared.ts#handleRefreshToken`）。
 *
 * 成功：持久化新的 `token` / `refreshToken` 并返回 `true`（Req 3.7）。
 * 失败：清空鉴权状态并返回 `false`（Req 2.10、3.9）。
 */
async function handleRefreshToken(): Promise<boolean> {
  const tokens = await refreshTokenCaller();
  if (tokens) {
    authAdapter.setToken(tokens.token);
    authAdapter.setRefreshToken(tokens.refreshToken);
    return true;
  }
  authAdapter.onLogout();
  return false;
}

/**
 * 令牌刷新单飞去重（复刻旧 `shared.ts#handleExpiredRequest`，Req 3.8）。
 *
 * 复用单一在途 `refreshTokenPromise`：并发的过期请求共享同一刷新，不发起重复刷新；
 * 刷新结束后延时 {@link REFRESH_TOKEN_PROMISE_CLEAR_DELAY}（1s）置空，便于后续再次过期时重新刷新。
 *
 * @param state 请求层状态（默认作用于 {@link requestState} 单例）。
 * @returns 刷新是否成功。
 */
export async function handleExpiredRequest(state: RequestInstanceState = requestState): Promise<boolean> {
  if (!state.refreshTokenPromise) {
    state.refreshTokenPromise = handleRefreshToken();
  }

  const success = await state.refreshTokenPromise;

  setTimeout(() => {
    state.refreshTokenPromise = null;
  }, REFRESH_TOKEN_PROMISE_CLEAR_DELAY);

  return success;
}

/** 向请求配置注入 `Authorization: Bearer <token>`（Req 2.2）。 */
function injectAuthHeader(config: AxiosRequestConfig): void {
  const token = authAdapter.getToken();
  if (token) {
    config.headers = { ...(config.headers ?? {}), Authorization: `Bearer ${token}` };
  }
}

/** 一次性弹窗登出：去重展示 + 确认后登出并清理去重栈（Req 2.5）。 */
function handleModalLogout(message: string): void {
  if (requestState.errMsgStack.includes(message)) {
    return;
  }
  requestState.errMsgStack.push(message);
  modalLogoutHandler(message, () => {
    authAdapter.onLogout();
    requestState.errMsgStack = requestState.errMsgStack.filter(msg => msg !== message);
  });
}

/** 裸 axios 实例：仅供默认刷新调用器使用，不挂任何拦截器，避免刷新自身被过期处理拦截。 */
const rawRefreshInstance: AxiosInstance = axios.create({
  baseURL: SERVICE_BASE_URL,
  timeout: REQUEST_TIMEOUT,
  headers: { 'Content-Type': 'application/json' },
});

/** 请求层单例 axios 实例（`timeout: 10000`、构建期基址、含拦截器）。 */
export const instance: AxiosInstance = axios.create({
  baseURL: SERVICE_BASE_URL,
  timeout: REQUEST_TIMEOUT,
  headers: { 'Content-Type': 'application/json' },
  // 与旧 `@sa/axios` 的 `isHttpSuccess` 一致：2xx 或 304 视为 HTTP 成功，其余 reject。
  validateStatus: status => (status >= 200 && status < 300) || status === 304,
});

// 请求拦截器：注入鉴权头（Req 2.2）。
instance.interceptors.request.use(config => {
  injectAuthHeader(config);
  return config;
});

/**
 * 后端业务失败分支处理（HTTP 2xx 但业务码非成功）。
 *
 * 按 `classifyResponse` 的类别分派（顺序对齐 design.md / 旧实现）：
 *  - `logout`（Req 2.4）：清空鉴权并登出。
 *  - `modalLogout`（Req 2.5）：一次性弹窗，确认后登出。
 *  - `expired`（Req 2.6）：单飞刷新；成功则对原请求至多重放一次，失败由 {@link handleRefreshToken} 登出。
 *  - `failure`（Req 2.9）：不在此恢复，返回 `null` 交由 onError 提示。
 *
 * @returns 恢复成功时返回重放后的响应（promise），否则返回 `null`。
 */
async function onBackendFail(
  response: AxiosResponse<BackendResponse>,
  config: RetriableConfig,
): Promise<AxiosResponse | null> {
  const classified = classifyResponse(response.data);
  const backendMessage = getBackendMessage(response.data);

  switch (classified.category) {
    case 'logout':
      authAdapter.onLogout();
      return null;

    case 'modalLogout':
      handleModalLogout(backendMessage);
      return null;

    case 'expired': {
      // 至多重放一次：已重放过的请求不再触发刷新（Req 2.6）。
      if (config.__isRetry) {
        return null;
      }
      const success = await handleExpiredRequest(requestState);
      if (success) {
        config.__isRetry = true;
        injectAuthHeader(config);
        return instance.request(config);
      }
      return null;
    }

    default:
      // 'failure'（普通错误码，含缺失 code）→ 交由 onError 展示后端消息（Req 2.9）。
      return null;
  }
}

/**
 * 错误处理（展示提示 / 静默忽略）。
 *
 * 对网络错误、超时、HTTP 错误以及合成的后端失败错误统一裁决是否展示提示：
 *  - WAF 可选接口 404/405 → 静默忽略（Req 2.8）。
 *  - 登出 / 弹窗登出 / 过期码 → 其副作用与提示已在 onBackendFail 处理，避免重复提示。
 *  - 其余 → 经去重栈展示错误消息（Req 2.9、2.11、2.7）。
 */
async function onError(error: AxiosError<BackendResponse>): Promise<void> {
  let message = error.message;
  let backendErrorCode = '';
  const requestUrl = String(error.config?.url ?? '');
  const status = error.response?.status;

  if (error.code === BACKEND_ERROR_CODE) {
    message = getBackendMessage(error.response?.data, message);
    backendErrorCode = String(error.response?.data?.code ?? '');
  }

  // WAF 可选接口 404/405 静默忽略（Req 2.8）。
  if (isSilentlyIgnored(requestUrl, status)) {
    return;
  }

  // 这些码的副作用/提示已在 onBackendFail 处理过，避免重复提示。
  if (LOGOUT_CODES.has(backendErrorCode) || MODAL_LOGOUT_CODES.has(backendErrorCode) || EXPIRED_TOKEN_CODES.has(backendErrorCode)) {
    return;
  }

  showErrorMsg(requestState, message);
}

// 响应拦截器：成功直接放行；业务失败走分支恢复，否则合成后端错误并 reject。
instance.interceptors.response.use(
  async response => {
    const responseType = response.config?.responseType ?? 'json';
    if (responseType !== 'json') {
      return response;
    }

    const classified = classifyResponse(response.data as BackendResponse);
    if (classified.category === 'success') {
      return response;
    }

    const recovered = await onBackendFail(response as AxiosResponse<BackendResponse>, response.config as RetriableConfig);
    if (recovered) {
      return recovered;
    }

    const backendError = new AxiosError<BackendResponse>(
      'the backend request error',
      BACKEND_ERROR_CODE,
      response.config,
      response.request,
      response as AxiosResponse<BackendResponse>,
    );
    backendError.response = response as AxiosResponse<BackendResponse>;

    await onError(backendError);

    return Promise.reject(backendError);
  },
  async (error: AxiosError<BackendResponse>) => {
    const status = error.response?.status;
    const config = error.config as RetriableConfig | undefined;

    // HTTP 401：单飞刷新 + 至多重放一次，失败登出（Req 2.6、2.10）。
    if (status === HTTP_UNAUTHORIZED) {
      if (config && !config.__isRetry) {
        const success = await handleExpiredRequest(requestState);
        if (success) {
          config.__isRetry = true;
          injectAuthHeader(config);
          return instance.request(config);
        }
        // 刷新失败：handleRefreshToken 已登出，直接 reject（不展示通用错误）。
      } else {
        // 无配置可重放，或已重放过一次仍 401 → 放弃并确保登出（Req 2.10）。
        authAdapter.onLogout();
      }
      return Promise.reject(error);
    }

    await onError(error);
    return Promise.reject(error);
  },
);

/**
 * 发起请求，返回扁平结构 `{ data, error }`（design.md「Request_Layer」接口签名）。
 *
 * - 成功：`data` 为后端响应体的 `data` 字段，`error` 为 `null`（Req 2.3）。
 * - 失败：`data` 为 `null`，`error` 为错误对象（业务失败 / HTTP 错误 / 超时 / 网络错误）。
 *
 * 所有分类、去重、白名单、刷新重放等行为均由拦截器在内部完成；调用方只需消费扁平结果。
 */
export async function request<T = unknown>(config: AxiosRequestConfig): Promise<FlatResponse<T>> {
  try {
    const response = await instance.request<BackendResponse<T>>(config);
    const responseType = response.config?.responseType ?? 'json';

    if (responseType !== 'json') {
      return { data: response.data as unknown as T, error: null, response };
    }

    const body = response.data;
    return { data: (body?.data ?? null) as T | null, error: null, response };
  } catch (error) {
    const axiosError = error as AxiosError<BackendResponse<T>>;
    return { data: null, error: axiosError, response: axiosError.response };
  }
}

export default request;
