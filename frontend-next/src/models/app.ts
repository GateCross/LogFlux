/**
 * 全局应用模型（偏好更新等）。
 * 任务 3.4 / Req 6.5
 */
import { useState, useCallback } from 'react';
import { fetchUpdateUserPreferences } from '@/services/auth';
import { showErrorMsg } from '@/utils/request/err-msg';
import { 
  type UserPreferences, 
  DEFAULT_PREFERENCES,
  serializePreferences,
  deserializePreferences
} from '@/utils/preferences';
import { getStorage, setStorage } from '@/utils/storage';

const PREFERENCES_KEY = 'preferences';

export default function useAppModel() {
  const [preferences, setPreferencesState] = useState<UserPreferences>(() => {
    const saved = getStorage<string>(PREFERENCES_KEY);
    if (!saved) return DEFAULT_PREFERENCES;
    const { preferences: parsed } = deserializePreferences(saved);
    return parsed;
  });

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
  };
}
