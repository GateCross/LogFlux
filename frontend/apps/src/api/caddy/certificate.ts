import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace CaddyCertificateApi {
  /** 证书来源 — 对齐 backend CertificateItem.source */
  export type CertificateSource =
    | 'acme_dns'
    | 'acme_http'
    | 'internal'
    | 'manual_declared'
    | 'upload';

  /** 证书状态 — 对齐 backend CertificateItem.status；expiring 阈值 ≤14 天 */
  export type CertificateStatus =
    | 'expired'
    | 'expiring'
    | 'missing'
    | 'pending'
    | 'valid';

  /**
   * 证书库存项 — 对齐 backend CertificateItem（JSON 小驼峰）。
   * 只有 Caddyfile 声明、卷里没有文件时：status=missing、有效期零值、daysLeft=0。
   */
  export interface CertificateItem {
    id: number;
    /** SAN 列表，逗号分隔 */
    domains: string;
    primaryDomain: string;
    issuer: string;
    source: CertificateSource;
    notBefore: string;
    notAfter: string;
    lastSyncedAt: string;
    storagePath: string;
    status: CertificateStatus;
    /** 距到期天数 */
    daysLeft: number;
  }

  export interface CertificateListResult {
    list: CertificateItem[];
  }

  /** 上传载荷 — multipart 字段：cert / key / domains（可选） */
  export interface CertificateUploadPayload {
    certFile: File;
    keyFile: File;
    /** 可选域名备注，逗号分隔；留空由后端从证书 SAN 解析 */
    domains?: string;
  }
}

/**
 * 获取证书库存 — GET /caddy/certificates
 */
export async function getCertificateListApi(config?: RequestClientConfig) {
  const resp = await requestClient.get<CaddyCertificateApi.CertificateListResult>(
    '/caddy/certificates',
    config,
  );
  return listOf(resp);
}

/**
 * 上传证书（PEM + 私钥）— POST /caddy/certificates/upload
 */
export async function uploadCertificateApi(
  data: CaddyCertificateApi.CertificateUploadPayload,
  config?: RequestClientConfig,
) {
  const formData = new FormData();
  formData.append('cert', data.certFile);
  formData.append('key', data.keyFile);
  if (data.domains?.trim()) formData.append('domains', data.domains.trim());
  return requestClient.post<void>('/caddy/certificates/upload', formData, config);
}

/**
 * 触发证书库存重新扫描（卷扫描 + Caddyfile 声明合并）— POST /caddy/certificates/refresh
 */
export async function refreshCertificatesApi() {
  return requestClient.post<void>('/caddy/certificates/refresh');
}

/**
 * 删除证书记录 — DELETE /caddy/certificates/:id
 */
export async function deleteCertificateApi(id: number) {
  return requestClient.delete<void>(`/caddy/certificates/${id}`);
}
