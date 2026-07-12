import { describe, expect, it } from 'vitest';

import {
  blockKindColor,
  blockKindLabel,
  createEmptyFormModel,
  defaultHealthCheck,
} from './useCaddyConfigIO';
import type { PreservedCaddyBlock } from '../types';

describe('useCaddyConfigIO pure helpers', () => {
  describe('createEmptyFormModel', () => {
    it('happy path: empty structured model scaffold', () => {
      expect(createEmptyFormModel()).toEqual({
        schemaVersion: 1,
        global: { raw: '' },
        upstreams: [],
        sites: [],
      });
    });

    it('boundary: nested arrays/objects are fresh per call', () => {
      const a = createEmptyFormModel();
      const b = createEmptyFormModel();
      expect(a).not.toBe(b);
      expect(a.upstreams).not.toBe(b.upstreams);
      expect(a.sites).not.toBe(b.sites);
      expect(a.global).not.toBe(b.global);
      a.sites.push({
        id: 'x',
        name: 'n',
        enabled: true,
        domains: [],
        tls: { mode: 'auto' },
        imports: [],
        geoip2Vars: [],
        encode: [],
        routes: [],
      } as (typeof a.sites)[number]);
      a.global.raw = 'dirty';
      expect(b.sites).toEqual([]);
      expect(b.global.raw).toBe('');
    });
  });

  describe('defaultHealthCheck', () => {
    it('happy path: default path/interval/timeout', () => {
      expect(defaultHealthCheck()).toEqual({
        path: '/health',
        interval: '10s',
        timeout: '5s',
      });
    });

    it('boundary: each call returns independent object', () => {
      const a = defaultHealthCheck();
      const b = defaultHealthCheck();
      expect(a).not.toBe(b);
      a.path = '/ready';
      expect(b.path).toBe('/health');
    });
  });

  describe('blockKindLabel / blockKindColor', () => {
    it('happy path: known kinds map to labels and colors', () => {
      const kinds: Array<PreservedCaddyBlock['kind']> = ['global', 'snippet', 'site'];
      expect(kinds.map(blockKindLabel)).toEqual(['全局', 'snippet', 'site']);
      expect(kinds.map(blockKindColor)).toEqual(['green', 'blue', 'orange']);
    });

    it('boundary: unknown kind → 未知 / default', () => {
      const unknown = 'other' as PreservedCaddyBlock['kind'];
      expect(blockKindLabel(unknown)).toBe('未知');
      expect(blockKindColor(unknown)).toBe('default');
    });
  });
});
