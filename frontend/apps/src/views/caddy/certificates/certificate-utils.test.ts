import { describe, expect, it } from 'vitest';

import {
  buildUploadedCertOptions,
  certDaysBadge,
  certSourceLabel,
  certStatusLabel,
  certStatusColor,
  deriveUploadCertPairPaths,
  formatCertDateRange,
  isLocalServer,
  isSelectableUploadedCert,
  isZeroIsoDate,
  type CertificateItem,
} from './certificate-utils';

function item(overrides: Partial<CertificateItem>): CertificateItem {
  return {
    id: 1,
    serverId: 1,
    domains: 'a.example.com',
    primaryDomain: 'a.example.com',
    issuer: "Let's Encrypt",
    source: 'acme_http',
    notBefore: '2026-01-01T00:00:00Z',
    notAfter: '2027-01-01T00:00:00Z',
    lastSyncedAt: '2026-10-25T00:00:00Z',
    storagePath: '/data/caddy/certificates/acme/a.example.com',
    status: 'valid',
    daysLeft: 60,
    ...overrides,
  };
}

describe('certStatusLabel / certSourceLabel', () => {
  it('状态与来源映射为中文', () => {
    expect(certStatusLabel('valid')).toBe('有效');
    expect(certStatusLabel('expiring')).toBe('即将到期');
    expect(certStatusLabel('expired')).toBe('已过期');
    expect(certStatusLabel('missing')).toBe('缺失');
    expect(certStatusLabel('pending')).toBe('远端（仅声明）');
    expect(certSourceLabel('acme_http')).toBe('ACME 自动');
    expect(certSourceLabel('acme_dns')).toBe('DNS 自动');
    expect(certSourceLabel('internal')).toBe('内部 CA');
    expect(certSourceLabel('upload')).toBe('上传');
    expect(certSourceLabel('manual_declared')).toBe('手动指定');
  });
});

describe('certDaysBadge', () => {
  it('剩余天数驱动颜色：>14 绿、≤14 黄、≤0 红', () => {
    expect(certDaysBadge(item({ daysLeft: 60 }))).toEqual({
      text: '剩 60 天',
      color: 'green',
    });
    expect(certDaysBadge(item({ daysLeft: 14 }))).toEqual({
      text: '剩 14 天',
      color: 'gold',
    });
    expect(certDaysBadge(item({ status: 'expiring', daysLeft: 3 }))).toEqual({
      text: '剩 3 天',
      color: 'gold',
    });
    expect(certDaysBadge(item({ status: 'expired', daysLeft: 0 }))).toEqual({
      text: '已过期',
      color: 'red',
    });
    expect(certDaysBadge(item({ daysLeft: -2 }))).toEqual({
      text: '已过期',
      color: 'red',
    });
  });

  it('pending/missing 行如实显示占位，不假装知道天数', () => {
    expect(certDaysBadge(item({ status: 'pending', daysLeft: 0 }))).toEqual({
      text: '—',
      color: 'default',
    });
    expect(certDaysBadge(item({ status: 'missing', daysLeft: 0 }))).toEqual({
      text: '—',
      color: 'default',
    });
  });
});

describe('certStatusColor', () => {
  it('状态对应 Tag 颜色', () => {
    expect(certStatusColor('valid')).toBe('green');
    expect(certStatusColor('expiring')).toBe('gold');
    expect(certStatusColor('expired')).toBe('red');
    expect(certStatusColor('missing')).toBe('volcano');
    expect(certStatusColor('pending')).toBe('default');
  });
});

describe('formatCertDateRange / isZeroIsoDate', () => {
  it('正常区间取日期部分拼接', () => {
    expect(
      formatCertDateRange('2026-01-01T08:00:00Z', '2027-01-01T08:00:00Z'),
    ).toBe('2026-01-01 ~ 2027-01-01');
  });

  it('零值时间显示占位：远端声明行只有一端或全无', () => {
    expect(isZeroIsoDate('0001-01-01T00:00:00Z')).toBe(true);
    expect(isZeroIsoDate('')).toBe(true);
    expect(isZeroIsoDate(undefined)).toBe(true);
    expect(formatCertDateRange('0001-01-01T00:00:00Z', '0001-01-01T00:00:00Z')).toBe(
      '—',
    );
    expect(formatCertDateRange('0001-01-01T00:00:00Z', '2027-01-01T00:00:00Z')).toBe(
      '? ~ 2027-01-01',
    );
  });
});

describe('isLocalServer', () => {
  it('remote 禁选上传，缺省 type 视为 local', () => {
    expect(isLocalServer({ type: 'local' })).toBe(true);
    expect(isLocalServer({})).toBe(true);
    expect(isLocalServer({ type: 'remote' })).toBe(false);
  });
});

describe('deriveUploadCertPairPaths', () => {
  it('storagePath 为证书文件路径：证书原样、私钥取同目录 privkey.pem', () => {
    expect(
      deriveUploadCertPairPaths('/config/certs/1/a.example.com/fullchain.pem'),
    ).toEqual({
      certPath: '/config/certs/1/a.example.com/fullchain.pem',
      keyPath: '/config/certs/1/a.example.com/privkey.pem',
    });
  });

  it('反斜杠归一为 Caddy 可用的正斜杠', () => {
    expect(
      deriveUploadCertPairPaths('\\config\\certs\\2\\b.example.com\\fullchain.pem'),
    ).toEqual({
      certPath: '/config/certs/2/b.example.com/fullchain.pem',
      keyPath: '/config/certs/2/b.example.com/privkey.pem',
    });
  });

  it.each([
    [''],
    ['   '],
    ['fullchain.pem'],
    ['/fullchain.pem'],
    [undefined],
    [null],
  ])('无法定位目录时返回 null：%s', (value) => {
    expect(deriveUploadCertPairPaths(value as string | null | undefined)).toBeNull();
  });
});

describe('isSelectableUploadedCert / buildUploadedCertOptions', () => {
  it('仅上传来源且未失效（有效/即将到期）可选', () => {
    expect(isSelectableUploadedCert(item({ source: 'upload', status: 'valid' }))).toBe(
      true,
    );
    expect(
      isSelectableUploadedCert(item({ source: 'upload', status: 'expiring' })),
    ).toBe(true);
    expect(isSelectableUploadedCert(item({ source: 'upload', status: 'expired' }))).toBe(
      false,
    );
    expect(isSelectableUploadedCert(item({ source: 'upload', status: 'missing' }))).toBe(
      false,
    );
    expect(isSelectableUploadedCert(item({ source: 'upload', status: 'pending' }))).toBe(
      false,
    );
    expect(isSelectableUploadedCert(item({ source: 'acme_http' }))).toBe(false);
  });

  it('按 serverId 过滤并附推导出的双路径', () => {
    const list: CertificateItem[] = [
      item({
        id: 1,
        serverId: 1,
        source: 'upload',
        storagePath: '/config/certs/1/a.example.com/fullchain.pem',
      }),
      item({
        id: 2,
        serverId: 2,
        source: 'upload',
        storagePath: '/config/certs/2/b.example.com/fullchain.pem',
      }),
      item({
        id: 3,
        serverId: 1,
        source: 'upload',
        status: 'expired',
        storagePath: '/config/certs/1/c.example.com/fullchain.pem',
      }),
      item({
        id: 4,
        serverId: 1,
        source: 'internal',
        storagePath: '/config/certs/1/d.example.com/fullchain.pem',
      }),
    ];
    const options = buildUploadedCertOptions(list, 1);
    expect(options).toHaveLength(1);
    expect(options[0]).toMatchObject({
      value: 1,
      certPath: '/config/certs/1/a.example.com/fullchain.pem',
      keyPath: '/config/certs/1/a.example.com/privkey.pem',
    });
    // 标签含主域名与有效期区间
    expect(options[0]!.label).toContain('a.example.com');
    expect(options[0]!.label).toContain('2027-01-01');
  });

  it('storagePath 推导不出的行剔除', () => {
    const options = buildUploadedCertOptions(
      [item({ id: 9, source: 'upload', storagePath: '' })],
      1,
    );
    expect(options).toEqual([]);
  });

  it('serverId 缺省时不按服务器过滤', () => {
    const list: CertificateItem[] = [
      item({
        id: 1,
        serverId: 1,
        source: 'upload',
        storagePath: '/config/certs/1/a.example.com/fullchain.pem',
      }),
      item({
        id: 2,
        serverId: 2,
        source: 'upload',
        storagePath: '/config/certs/2/b.example.com/fullchain.pem',
      }),
    ];
    expect(buildUploadedCertOptions(list)).toHaveLength(2);
  });
});
