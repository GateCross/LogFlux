import { describe, expect, it } from 'vitest';

import { bytesToMB, mbToBytes } from './useSimpleWaf';

describe('useSimpleWaf pure helpers', () => {
  describe('mbToBytes', () => {
    it('happy path: converts positive MB to bytes', () => {
      expect(mbToBytes(1)).toBe(1024 * 1024);
      expect(mbToBytes(10)).toBe(10 * 1024 * 1024);
    });

    it('boundary: zero / negative / NaN clamp to at least 1MB', () => {
      expect(mbToBytes(0)).toBe(1024 * 1024);
      expect(mbToBytes(-5)).toBe(1024 * 1024);
      expect(mbToBytes(Number.NaN)).toBe(1024 * 1024);
    });
  });

  describe('bytesToMB', () => {
    it('happy path: converts positive bytes to rounded MB', () => {
      expect(bytesToMB(10 * 1024 * 1024, 99)).toBe(10);
      expect(bytesToMB(1.6 * 1024 * 1024, 99)).toBe(2);
    });

    it('boundary: zero / negative / missing fall back to fallback', () => {
      expect(bytesToMB(0, 7)).toBe(7);
      expect(bytesToMB(-1, 3)).toBe(3);
      // falsy NaN also falls back
      expect(bytesToMB(Number.NaN, 11)).toBe(11);
    });
  });
});
