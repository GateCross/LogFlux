/**
 * 鉴权模型（Auth_Module，任务 4.2）。
 *
 * 设计依据：design.md「Auth_Module」，迁移自旧 Vue 版 `frontend/src/store/modules/auth/index.ts`。
 *
 * 职责：
 *  - 登录输入校验（Req 3.4、7.2）：用户名非空且 ≤64、密码非空且 ≤128。
 *  - 登录流程（Req 3.1、3.2、3.3、3.5、3.6、3.10）：
 *    `encrypt` → `POST /api/login` → 持久化 token → `GET /api/user/info` → 存储用户信息。
 *  - 令牌持久化与读取（token / refreshToken）。
 *  - 用户信息初始化（`initUserInfo`）。
 *  - 登出（`resetStore` / `logout`）：清本地鉴权并跳登录页。
 *  - 超级角色判定（`isSuperRole`）。
 *
 * 支撑的需求：3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.10, 2.4
 */
import { useState, useCallback } from 'react';
import { history } from '@umijs/max';
import { fetchLogin, fetchGetUserInfo } from '@/services/auth';
import { encrypt } from '@/utils/crypto';
import { getStorage, setStorage, removeStorage } from '@/utils/storage';
import { SUPER_ROLE, ROUTE_HOME } from '@/constants/app';
import { showErrorMsg } from '@/utils/request/err-msg';
import { setRequestAuthAdapter } from '@/utils/request';

// ──────────────────────────────────────────────────────────────────────────
// 鉴权存储键
// ──────────────────────────────────────────────────────────────────────────

const TOKEN_KEY = 'token';
const REFRESH_TOKEN_KEY = 'refreshToken';
const LAST_LOGIN_USER_ID_KEY = 'lastLoginUserId';

// ──────────────────────────────────────────────────────────────────────────
// 令牌读写
// ──────────────────────────────────────────────────────────────────────────

export function getToken(): string {
  return getStorage<string>(TOKEN_KEY) || '';
}

export function getRefreshToken(): string {
  return getStorage<string>(REFRESH_TOKEN_KEY) || '';
}

export function setToken(token: string): void {
  setStorage(TOKEN_KEY, token);
}

export function setRefreshToken(token: string): void {
  setStorage(REFRESH_TOKEN_KEY, token);
}

export function clearAuthStorage(): void {
  removeStorage(TOKEN_KEY);
  removeStorage(REFRESH_TOKEN_KEY);
}

// ──────────────────────────────────────────────────────────────────────────
// 登录输入校验（Req 3.4、7.2）
// ──────────────────────────────────────────────────────────────────────────

export interface LoginValidationResult {
  valid: boolean;
  usernameError?: string;
  passwordError?: string;
}

const MAX_USERNAME_LENGTH = 64;
const MAX_PASSWORD_LENGTH = 128;

export function validateLoginInput(username: string, password: string): LoginValidationResult {
  const result: LoginValidationResult = { valid: true };

  if (!username || username.trim().length === 0) {
    result.usernameError = '用户名不能为空';
    result.valid = false;
  } else if (username.length > MAX_USERNAME_LENGTH) {
    result.usernameError = `用户名长度不能超过 ${MAX_USERNAME_LENGTH}`;
    result.valid = false;
  }

  if (!password || password.length === 0) {
    result.passwordError = '密码不能为空';
    result.valid = false;
  } else if (password.length > MAX_PASSWORD_LENGTH) {
    result.passwordError = `密码长度不能超过 ${MAX_PASSWORD_LENGTH}`;
    result.valid = false;
  }

  return result;
}

// ──────────────────────────────────────────────────────────────────────────
// 用户信息类型
// ──────────────────────────────────────────────────────────────────────────

export interface UserInfoState {
  userId: string | number;
  username: string;
  roles: string[];
  buttons: string[];
  preferences: string;
}

const EMPTY_USER_INFO: UserInfoState = {
  userId: '',
  username: '',
  roles: [],
  buttons: [],
  preferences: '',
};

// ──────────────────────────────────────────────────────────────────────────
// Auth Model Hook
// ──────────────────────────────────────────────────────────────────────────

export default function useAuthModel() {
  const [token, setTokenState] = useState<string>(() => getToken());
  const [userInfo, setUserInfo] = useState<UserInfoState>({ ...EMPTY_USER_INFO });
  const [loginLoading, setLoginLoading] = useState(false);

  /** 是否已登录 */
  const isLogin = Boolean(token);

  /** 是否超级角色（Req 4.5） */
  const isSuperRole = userInfo.roles.includes(SUPER_ROLE);

  // 注入鉴权适配器到 Request_Layer
  // 确保 request 层能读取最新的 token
  useState(() => {
    setRequestAuthAdapter({
      getToken: () => getToken(),
      getRefreshToken: () => getRefreshToken(),
      setToken: (t) => {
        if (t) setToken(t);
        else removeStorage(TOKEN_KEY);
        setTokenState(t || '');
      },
      setRefreshToken: (t) => {
        if (t) setRefreshToken(t);
        else removeStorage(REFRESH_TOKEN_KEY);
      },
      onLogout: () => {
        clearAuthStorage();
        setTokenState('');
        setUserInfo({ ...EMPTY_USER_INFO });
      },
    });
  });

  /**
   * 获取用户信息（Req 3.2）。
   * 失败时清空已持久化的令牌并提示（Req 3.6）。
   */
  const getUserInfo = useCallback(async (): Promise<boolean> => {
    const { data, error } = await fetchGetUserInfo();
    if (!error && data) {
      setUserInfo({
        userId: data.userId,
        username: data.username,
        roles: data.roles || [],
        buttons: data.buttons || [],
        preferences: data.preferences || '',
      });
      return true;
    }
    return false;
  }, []);

  /**
   * 初始化用户信息（Req 3.2）。
   * 如果本地有 token 则拉取用户信息；拉取失败则清空鉴权。
   */
  const initUserInfo = useCallback(async () => {
    const hasToken = getToken();
    if (hasToken) {
      setTokenState(hasToken);
      const pass = await getUserInfo();
      if (!pass) {
        resetStore();
      }
    }
  }, [getUserInfo]);

  /**
   * 记录上一次登录的用户 ID，用于下次登录时比对是否需要清空页签。
   */
  const recordUserId = useCallback(() => {
    if (!userInfo.userId) return;
    setStorage(LAST_LOGIN_USER_ID_KEY, String(userInfo.userId));
  }, [userInfo.userId]);

  /**
   * 检查是否需要清空页签（不同用户登录时清空）。
   */
  const checkTabClear = useCallback((): boolean => {
    if (!userInfo.userId) return false;
    const lastId = getStorage<string>(LAST_LOGIN_USER_ID_KEY);
    if (!lastId || lastId !== String(userInfo.userId)) {
      removeStorage('globalTabs');
      removeStorage(LAST_LOGIN_USER_ID_KEY);
      return true;
    }
    removeStorage(LAST_LOGIN_USER_ID_KEY);
    return false;
  }, [userInfo.userId]);

  /**
   * 登录流程（Req 3.1、3.2、3.3、3.5）。
   *
   * 流程：encrypt → POST /api/login → 持久化 token → GET /api/user/info。
   * 凭据错误展示后端消息不写状态（Req 3.5）。
   * 网络/超时失败提示不写状态（Req 3.5）。
   * user/info 失败清空已持久化令牌并提示（Req 3.6）。
   */
  const login = useCallback(
    async (username: string, password: string, redirect = true) => {
      // 1. 输入校验（Req 3.4、7.2）
      const validation = validateLoginInput(username, password);
      if (!validation.valid) {
        return { success: false, validation };
      }

      setLoginLoading(true);

      try {
        // 2. 加密密码并调用登录接口（Req 3.1）
        const { data: loginToken, error } = await fetchLogin(username, encrypt(password));

        if (error || !loginToken) {
          // 凭据错误/网络失败：展示消息，不写状态（Req 3.5）
          return { success: false, validation: { valid: true } };
        }

        // 3. 持久化令牌（Req 3.3）
        setToken(loginToken.token);
        setRefreshToken(loginToken.refreshToken);
        setTokenState(loginToken.token);

        // 4. 获取用户信息（Req 3.2）
        const pass = await getUserInfo();
        if (!pass) {
          // user/info 失败：清空已持久化令牌并提示（Req 3.6）
          clearAuthStorage();
          setTokenState('');
          showErrorMsg('获取用户信息失败');
          return { success: false, validation: { valid: true } };
        }

        // 5. 检查是否需要清空页签
        checkTabClear();

        // 6. 重定向（Req 3.10）
        if (redirect) {
          const redirectUrl = new URLSearchParams(window.location.search).get('redirect');
          history.push(redirectUrl || `/${ROUTE_HOME}`);
        }

        return { success: true, validation: { valid: true } };
      } finally {
        setLoginLoading(false);
      }
    },
    [getUserInfo, checkTabClear],
  );

  /**
   * 登出（Req 2.4、3.10）：清本地鉴权并跳登录页。
   */
  const resetStore = useCallback(() => {
    recordUserId();
    clearAuthStorage();
    setTokenState('');
    setUserInfo({ ...EMPTY_USER_INFO });

    if (window.location.pathname !== '/login') {
      history.push('/login');
    }
  }, [recordUserId]);

  const logout = resetStore;

  return {
    token,
    userInfo,
    isLogin,
    isSuperRole,
    loginLoading,
    login,
    logout,
    resetStore,
    initUserInfo,
    getUserInfo,
  };
}
