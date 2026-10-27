/**
 * 「使用已有证书」下拉数据源：拉取上传证书库存并构造选项，
 * 建站弹窗与站点抽屉共用；queryKey 与证书中心同维度、共享缓存。
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue';

import { getCertificateListApi } from '#/api/caddy/certificate';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

import { buildUploadedCertOptions, type CertSelectOption } from './certificate-utils';

/** enabled 控制仅弹窗/抽屉打开时拉取；关闭时不发请求、选项为空。 */
export function useUploadedCertOptions(
  enabled?: MaybeRefOrGetter<boolean>,
) {
  const {
    data,
    loading,
    errorMessage,
    refetch,
  } = useListDetailQuery({
    queryKey: qk.caddy.certificates(),
    queryFn: () => getCertificateListApi(withListDetailErrorMode()),
    errorFallback: '获取证书列表失败',
    enabled: () => toValue(enabled ?? true),
  });

  const options = computed<CertSelectOption[]>(() =>
    buildUploadedCertOptions(data.value ?? []),
  );

  return { options, loading, errorMessage, refetch };
}
