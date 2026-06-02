/**
 * 全局应用模型（偏好更新等）。
 * 任务 3.4 / Req 6.5
 */
import { useState, useCallback, useRef } from 'react';
import { fetchUpdateUserPreferences } from '@/services/auth';
import { showErrorMsg } from '@/utils/request/err-msg';
import {
  type UserPreferences,
  DEFAULT_PREFERENCES,
  serializePreferences,
  deserializePreferences,
  loadPreferences
} from '@/utils/preferences';
import { getStorage, setStorage } from '@/utils/storage';

const PREFERENCES_KEY = 'preferences';

export default function useAppModel() {
  const [preferences, setPreferencesState] = useState<UserPreferences>(() => {
    const saved = getStorage<string>(PREFERENCES_KEY);
    if (!saved) return DEFAULT_PREFERENCES;
    const { preferences: parsed } = loadPreferences(saved);
    return parsed;
  });

  /** 标记是否已执行过远端同步，避免重复写入 */
  const remoteSyncedRef = useRef(false);

  /**
   * 从远端用户信息同步偏好到本地。
   * - 仅当远端偏好与本地不同时更新
   * - 不触发远程写入（避免循环）
   * - 幂等：仅首次调用生效
   */
  const syncFromRemote = useCallback((remotePreferencesStr: string) => {
    if (remoteSyncedRef.current || !remotePreferencesStr) return;

    try {
      const remotePrefs = deserializePreferences(remotePreferencesStr);
      const localSerialized = serializePreferences(preferences);
      const remoteSerialized = serializePreferences(remotePrefs);

      // 仅当远端与本地不同时更新
      if (remoteSerialized !== localSerialized) {
        setPreferencesState(remotePrefs);
        setStorage(PREFERENCES_KEY, remoteSerialized);
      }
    } catch {
      // 远端偏好解析失败，忽略，保持本地值
    }

    remoteSyncedRef.current = true;
  }, [preferences]);

  /**
   * 更新偏好设置，执行双写事务（本地 + 远程）。
   * - 2000ms 超时竞速
   * - 失败或超时则回滚本地状态并提示
   */
  const updatePreferences = useCallback(async (newPrefs: UserPreferences) => {
    const oldPrefs = preferences;
    const serialized = serializePreferences(newPrefs);

    // 乐观更新本地状态
    setPreferencesState(newPrefs);

    try {
      const syncRemote = fetchUpdateUserPreferences(serialized);

      const persistLocal = new Promise<void>((resolve) => {
        setStorage(PREFERENCES_KEY, serialized);
        resolve();
      });

      const timeoutRace = new Promise<never>((_, reject) => {
        setTimeout(() => reject(new Error('timeout')), 2000);
      });

      await Promise.race([
        Promise.all([persistLocal, syncRemote.then(res => {
          if (res.error) throw res.error;
        })]),
        timeoutRace
      ]);
    } catch (error) {
      // 发生错误或超时，回滚本地状态并持久化旧值
      setPreferencesState(oldPrefs);
      setStorage(PREFERENCES_KEY, serializePreferences(oldPrefs));
      showErrorMsg('偏好设置同步失败');
      return false;
    }

    return true;
  }, [preferences]);

  return {
    preferences,
    updatePreferences,
    syncFromRemote,
  };
}
