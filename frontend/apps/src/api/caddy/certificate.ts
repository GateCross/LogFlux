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
   * remote 服务器只有 Caddyfile 声明行：status=pending、有效期零值、daysLeft=0。
   */
  export interface CertificateItem {
    id: number;
    serverId: number;
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

  /** 上传载荷 — multipart 字段：cert / key / serverId / domains（可选） */
  export interface CertificateUploadPayload {
    certFile: File;
    keyFile: File;
    serverId: number;
    /** 可选域名备注，逗号分隔；留空由后端从证书 SAN 解析 */
    domains?: string;
  }
}

/**
 * 获取证书库存 — GET /caddy/certificates?serverId=
 * serverId 缺省时返回全部服务器的证书。
 */
export async function getCertificateListApi(
  serverId?: number,
  config?: RequestClientConfig,
) {
  const resp = await requestClient.get<CaddyCertificateApi.CertificateListResult>(
    '/caddy/certificates',
    { params: serverId ? { serverId } : undefined, ...config },
  );
  return listOf(resp);
}

/**
 * 上传证书（PEM + 私钥）— POST /caddy/certificates/upload
 * 仅 local 服务器可上传；remote 无文件通道。
 */
export async function uploadCertificateApi(
  data: CaddyCertificateApi.CertificateUploadPayload,
  config?: RequestClientConfig,
) {
  const formData = new FormData();
  formData.append('cert', data.certFile);
  formData.append('key', data.keyFile);
  formData.append('serverId', String(data.serverId));
  if (data.domains?.trim()) formData.append('domains', data.domains.trim());
  return requestClient.post<void>('/caddy/certificates/upload', formData, config);
}

/**
 * 触发证书库存重新扫描（卷扫描 + Caddyfile 声明合并）— POST /caddy/certificates/refresh
 * serverId 缺省时刷新全部服务器。
 */
export async function refreshCertificatesApi(serverId?: number) {
  return requestClient.post<void>('/caddy/certificates/refresh', {
    serverId,
  });
}

/**
 * 删除证书记录 — DELETE /caddy/certificates/:id
 */
export async function deleteCertificateApi(id: number) {
  return requestClient.delete<void>(`/caddy/certificates/${id}`);
}
