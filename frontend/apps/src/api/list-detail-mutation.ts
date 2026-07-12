/**
 * list-detail 写后缓存失效：成功 mutation 后 invalidate 相关 qk 前缀
 */
import type { QueryClient, QueryKey } from '@tanstack/vue-query';

/** 使单个 queryKey 前缀失效 */
export async function invalidateListDetailQueries(
  queryClient: QueryClient,
  queryKey: QueryKey,
): Promise<void> {
  await queryClient.invalidateQueries({ queryKey });
}

/** 一次失效多个 queryKey */
export async function invalidateListDetailQueryKeys(
  queryClient: QueryClient,
  queryKeys: QueryKey[],
): Promise<void> {
  await Promise.all(
    queryKeys.map((queryKey) =>
      queryClient.invalidateQueries({ queryKey }),
    ),
  );
}
