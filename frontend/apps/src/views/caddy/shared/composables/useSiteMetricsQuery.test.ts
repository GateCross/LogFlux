/**
 * 站点指标共享封装的纯函数契约。
 *
 * 三个页面（目录 / 工作台 / 详情）不再各写一套请求，这层锁住它们共享的语义：
 * 分块大小、queryKey 签名、时间窗隔离。
 */
import { describe, expect, it } from 'vitest';

import {
  chunkHosts,
  hostsSignature,
  SITE_METRICS_RECENT_MINUTES,
  SITE_METRICS_TODAY_MINUTES,
} from './useSiteMetricsQuery';
import { SITE_METRICS_HOST_CHUNK } from '../../catalog/service-catalog-utils';
import { collectHostsFromSites } from '../../config/composables/useSiteMetrics';

describe('useSiteMetricsQuery 纯函数', () => {
  describe('chunkHosts', () => {
    it('按后端上限 50 切块', () => {
      const hosts = Array.from({ length: 120 }, (_, i) => `h${i}.example.com`);
      const chunks = chunkHosts(hosts);

      expect(chunks).toHaveLength(3);
      expect(chunks[0]).toHaveLength(SITE_METRICS_HOST_CHUNK);
      expect(chunks[1]).toHaveLength(SITE_METRICS_HOST_CHUNK);
      expect(chunks[2]).toHaveLength(20);
      // 拼回去必须与原列表等序等长，不能丢 host
      expect(chunks.flat()).toEqual(hosts);
    });

    it('边界：空列表不产生块，恰好一整块也只是一块', () => {
      expect(chunkHosts([])).toEqual([]);
      expect(chunkHosts(['a'])).toEqual([['a']]);
      const exact = Array.from({ length: SITE_METRICS_HOST_CHUNK }, (_, i) => `h${i}`);
      expect(chunkHosts(exact)).toHaveLength(1);
    });
  });

  describe('hostsSignature', () => {
    it('同一批 host 顺序不同得到同一个 key，不产生两份缓存', () => {
      expect(hostsSignature(['b.com', 'a.com'])).toBe(hostsSignature(['a.com', 'b.com']));
    });

    it('host 集合不同则 key 不同', () => {
      expect(hostsSignature(['a.com'])).not.toBe(hostsSignature(['a.com', 'b.com']));
    });

    it('不修改入参顺序（排序只作用于副本）', () => {
      const hosts = ['b.com', 'a.com'];
      hostsSignature(hosts);
      expect(hosts).toEqual(['b.com', 'a.com']);
    });
  });

  describe('collectHostsFromSites', () => {
    it('简单 + 复杂站点域名合并去重，保留可识别 host', () => {
      const hosts = collectHostsFromSites(
        [{ domains: ['a.example.com', 'https://B.example.com'] }],
        [{ domains: ['a.example.com', 'c.example.com:443'] }],
      );
      expect(hosts.toSorted()).toEqual(['a.example.com', 'b.example.com', 'c.example.com']);
    });

    it('边界：空列表 / 全是监听地址 → 空数组', () => {
      expect(collectHostsFromSites([], [])).toEqual([]);
      expect(collectHostsFromSites([{ domains: [':80', ''] }], [{ domains: undefined }])).toEqual(
        [],
      );
    });
  });

  describe('时间窗常量', () => {
    it('今日 24h 与近窗 15min 是两个独立隔离维度', () => {
      expect(SITE_METRICS_TODAY_MINUTES).toBe(1440);
      expect(SITE_METRICS_RECENT_MINUTES).toBe(15);
      expect(SITE_METRICS_TODAY_MINUTES).not.toBe(SITE_METRICS_RECENT_MINUTES);
    });
  });
});
