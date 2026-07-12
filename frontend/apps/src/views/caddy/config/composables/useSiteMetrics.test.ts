import { describe, expect, it } from 'vitest';

import {
  collectHostsFromSites,
  formatMetricCount,
  metricCountColor,
  normalizeSiteHost,
  sitePrimaryHost,
} from './useSiteMetrics';

describe('useSiteMetrics pure helpers', () => {
  describe('normalizeSiteHost', () => {
    it('happy path: strips protocol / port / path and lowercases', () => {
      expect(normalizeSiteHost('https://API.Example.com:443/path')).toBe('api.example.com');
      expect(normalizeSiteHost('www.example.com:8080')).toBe('www.example.com');
    });

    it('boundary: empty / listen-only address → empty string', () => {
      expect(normalizeSiteHost('')).toBe('');
      expect(normalizeSiteHost('   ')).toBe('');
      expect(normalizeSiteHost(':8080')).toBe('');
      expect(normalizeSiteHost(undefined as unknown as string)).toBe('');
    });
  });

  describe('sitePrimaryHost', () => {
    it('happy path: returns first valid host', () => {
      expect(sitePrimaryHost([':443', 'www.example.com', 'api.example.com'])).toBe(
        'www.example.com',
      );
    });

    it('boundary: undefined / all invalid → empty', () => {
      expect(sitePrimaryHost(undefined)).toBe('');
      expect(sitePrimaryHost([])).toBe('');
      expect(sitePrimaryHost([':8080', ''])).toBe('');
    });
  });

  describe('collectHostsFromSites', () => {
    it('happy path: dedupes hosts from simple + complex sites', () => {
      const hosts = collectHostsFromSites(
        [{ domains: ['a.example.com', 'https://B.example.com'] }],
        [{ domains: ['a.example.com', 'c.example.com:443'] }],
      );
      expect(hosts.sort()).toEqual(['a.example.com', 'b.example.com', 'c.example.com']);
    });

    it('boundary: empty lists / invalid domains → empty array', () => {
      expect(collectHostsFromSites([], [])).toEqual([]);
      expect(collectHostsFromSites([{ domains: [':80', ''] }], [{ domains: undefined }])).toEqual(
        [],
      );
    });
  });

  describe('formatMetricCount', () => {
    it('happy path: non-negative rounded count as string', () => {
      expect(formatMetricCount(12)).toBe('12');
      expect(formatMetricCount(3.6)).toBe('4');
    });

    it('boundary: null / undefined / NaN → em dash; negatives clamp to 0', () => {
      expect(formatMetricCount(null)).toBe('—');
      expect(formatMetricCount(undefined)).toBe('—');
      expect(formatMetricCount(Number.NaN)).toBe('—');
      expect(formatMetricCount(-2)).toBe('0');
    });
  });

  describe('metricCountColor', () => {
    it('happy path: positive 4xx warning, positive 5xx error', () => {
      expect(metricCountColor(1, '4xx')).toBe('warning');
      expect(metricCountColor(2, '5xx')).toBe('error');
    });

    it('boundary: missing / zero → default', () => {
      expect(metricCountColor(undefined, '4xx')).toBe('default');
      expect(metricCountColor(null, '5xx')).toBe('default');
      expect(metricCountColor(0, '4xx')).toBe('default');
    });
  });
});
