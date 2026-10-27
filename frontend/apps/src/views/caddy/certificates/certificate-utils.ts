/**
 * 证书中心展示纯函数：状态/来源文案、剩余天数徽章、有效期区间；
 * 另含「使用已有证书」下拉的选项构造与双路径推导（建站弹窗/站点抽屉共用）。
 * 只有声明没有真值的行（零值时间），展示层如实呈现"未知"。
 */

import type {
  CaddyCertificateApi,
} from '#/api/caddy/certificate';

export type CertificateItem = CaddyCertificateApi.CertificateItem;

/** 到期提醒阈值（天），与后端 expiring 判定一致 */
export const CERT_EXPIRING_DAYS = 14;

const STATUS_LABELS: Record<CertificateItem['status'], string> = {
  valid: '有效',
  expiring: '即将到期',
  expired: '已过期',
  missing: '缺失',
  pending: '仅声明',
};

const SOURCE_LABELS: Record<CertificateItem['source'], string> = {
  acme_http: 'ACME 自动',
  acme_dns: 'DNS 自动',
  internal: '内部 CA',
  upload: '上传',
  manual_declared: '手动指定',
};

export function certStatusLabel(status: CertificateItem['status']): string {
  return STATUS_LABELS[status] ?? status;
}

export function certSourceLabel(source: CertificateItem['source']): string {
  return SOURCE_LABELS[source] ?? source;
}

/** 零值时间（仅声明行）：go 零值 0001-01-01 或空串 */
export function isZeroIsoDate(value: string | null | undefined): boolean {
  if (!value) return true;
  return value.startsWith('0001-') || value.startsWith('0001-01-01');
}

/** ISO 字符串取日期部分展示；零值返回空串 */
function isoDatePart(value: string | null | undefined): string {
  if (!value || isZeroIsoDate(value)) return '';
  return value.trim().slice(0, 10);
}

/** 有效期区间文案：`2026-01-01 ~ 2027-01-01`；零值返回 '—' */
export function formatCertDateRange(
  notBefore: string | null | undefined,
  notAfter: string | null | undefined,
): string {
  const begin = isoDatePart(notBefore);
  const end = isoDatePart(notAfter);
  if (!begin && !end) return '—';
  if (!begin) return `? ~ ${end}`;
  if (!end) return `${begin} ~ ?`;
  return `${begin} ~ ${end}`;
}

export interface CertDaysBadge {
  text: string;
  /** antd Tag 预设色；default 表示中性灰 */
  color: 'default' | 'gold' | 'green' | 'red' | 'volcano';
}

/**
 * 剩余天数徽章：过期红、≤14 天黄、其余绿；
 * pending/missing（含零值时间）如实显示"—"，不假装知道。
 */
export function certDaysBadge(item: CertificateItem): CertDaysBadge {
  if (item.status === 'pending' || item.status === 'missing') {
    return { text: '—', color: 'default' };
  }
  if (item.status === 'expired' || item.daysLeft <= 0) {
    return { text: '已过期', color: 'red' };
  }
  if (item.status === 'expiring' || item.daysLeft <= CERT_EXPIRING_DAYS) {
    return { text: `剩 ${item.daysLeft} 天`, color: 'gold' };
  }
  return { text: `剩 ${item.daysLeft} 天`, color: 'green' };
}

/** 状态列 Tag 颜色 */
export function certStatusColor(status: CertificateItem['status']): string {
  if (status === 'valid') return 'green';
  if (status === 'expiring') return 'gold';
  if (status === 'expired') return 'red';
  if (status === 'missing') return 'volcano';
  return 'default';
}

/** 上传证书的私钥文件名 — 对齐 backend certUploadKeyFileName */
export const UPLOAD_KEY_FILE_NAME = 'privkey.pem';

/**
 * 从库存 storagePath 推导证书/私钥两个文件路径。
 * 上传证书落盘布局固定：/config/certs/<域名>/{fullchain,privkey}.pem，
 * storagePath 即证书文件完整路径；私钥取同目录 privkey.pem。
 * 路径为空或定位不到目录时返回 null；反斜杠归一为 Caddy 可用的正斜杠。
 */
export function deriveUploadCertPairPaths(
  storagePath: string | null | undefined,
): { certPath: string; keyPath: string } | null {
  const text = (storagePath ?? '').trim().replace(/\\/g, '/');
  if (!text) return null;
  const cut = text.lastIndexOf('/');
  if (cut <= 0) return null;
  return { certPath: text, keyPath: `${text.slice(0, cut)}/${UPLOAD_KEY_FILE_NAME}` };
}

/** "使用已有证书"下拉可选：上传来源且未失效（有效或即将到期） */
export function isSelectableUploadedCert(item: CertificateItem): boolean {
  if (item.source !== 'upload') return false;
  return item.status === 'valid' || item.status === 'expiring';
}

/** "使用已有证书"下拉选项 — value 为证书记录 id，附推导出的双路径 */
export interface CertSelectOption {
  value: number;
  label: string;
  certPath: string;
  keyPath: string;
}

/**
 * 构造"使用已有证书"下拉选项：上传来源 + 未失效，
 * 附推导出的证书/私钥路径；storagePath 推导不出的行剔除。
 */
export function buildUploadedCertOptions(
  list: CertificateItem[],
): CertSelectOption[] {
  const result: CertSelectOption[] = [];
  for (const item of list) {
    if (!isSelectableUploadedCert(item)) continue;
    const paths = deriveUploadCertPairPaths(item.storagePath);
    if (!paths) continue;
    const domain = item.primaryDomain || item.domains || `证书 #${item.id}`;
    result.push({
      value: item.id,
      label: `${domain}（${formatCertDateRange(item.notBefore, item.notAfter)}）`,
      certPath: paths.certPath,
      keyPath: paths.keyPath,
    });
  }
  return result;
}
