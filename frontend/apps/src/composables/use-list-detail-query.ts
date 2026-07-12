/**
 * list-detail 远程读：useQuery + 页内 errorMessage（queryFn 需 withListDetailErrorMode）
 */
import { useQuery, type QueryKey } from '@tanstack/vue-query';
import {
  computed,
  type ComputedRef,
  type MaybeRefOrGetter,
  type Ref,
  unref,
} from 'vue';

import {
  isListDetailEmpty,
  toListDetailErrorMessage,
} from '#/api/list-detail';

/** queryKey 可为 Ref/ComputedRef/值，不接受 getter */
export type ListDetailQueryKey =
  | QueryKey
  | Ref<QueryKey>
  | ComputedRef<QueryKey>;

export interface UseListDetailQueryOptions<TData = unknown> {
  queryKey: ListDetailQueryKey;
  queryFn: () => Promise<TData>;
  errorFallback: string;
  enabled?: MaybeRefOrGetter<boolean>;
  staleTime?: number;
}

export interface UseListDetailQueryResult<TData = unknown> {
  data: ComputedRef<TData | undefined>;
  loading: ComputedRef<boolean>;
  errorMessage: ComputedRef<string | null>;
  isEmpty: ComputedRef<boolean>;
  refetch: () => Promise<unknown>;
  isPending: ComputedRef<boolean>;
  isFetching: ComputedRef<boolean>;
  isError: ComputedRef<boolean>;
}

export function useListDetailQuery<TData = unknown>(
  options: UseListDetailQueryOptions<TData>,
): UseListDetailQueryResult<TData> {
  const { errorFallback, queryKey, queryFn, enabled, staleTime } = options;

  const query = useQuery<TData>({
    queryKey,
    queryFn,
    enabled,
    staleTime,
  });

  const data = computed(() => query.data.value as TData | undefined);

  const loading = computed(
    () => Boolean(query.isPending.value) || Boolean(query.isFetching.value),
  );

  const errorMessage = computed(() =>
    toListDetailErrorMessage(query.error.value, errorFallback),
  );

  const isEmpty = computed(
    () =>
      !loading.value &&
      errorMessage.value == null &&
      isListDetailEmpty(unref(data)),
  );

  return {
    data,
    loading,
    errorMessage,
    isEmpty,
    refetch: () => query.refetch(),
    isPending: computed(() => Boolean(query.isPending.value)),
    isFetching: computed(() => Boolean(query.isFetching.value)),
    isError: computed(() => Boolean(query.isError.value)),
  };
}
