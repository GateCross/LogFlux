import { describe, expect, it } from 'vitest';

import { diffSideClass, historyDescription, shortHash } from './useConfigHistory';

describe('useConfigHistory pure helpers', () => {
  describe('shortHash', () => {
    it('happy path: short text unchanged; long text abbreviated', () => {
      expect(shortHash('abc123')).toBe('abc123');
      expect(shortHash('0123456789abcdef0123456789abcdef')).toBe('01234567...89abcdef');
    });

    it('boundary: null / undefined / whitespace-only', () => {
      expect(shortHash(null)).toBe('');
      expect(shortHash(undefined)).toBe('');
      expect(shortHash('   ')).toBe('');
      // length 16 stays full (boundary of truncation threshold)
      expect(shortHash('1234567890123456')).toBe('1234567890123456');
    });
  });

  describe('historyDescription', () => {
    it('happy path: uses item.hash via shortHash', () => {
      expect(
        historyDescription({
          hash: 'deadbeefcafebabe01234567',
        } as Parameters<typeof historyDescription>[0]),
      ).toBe(shortHash('deadbeefcafebabe01234567'));
    });

    it('boundary: missing hash → empty description', () => {
      expect(
        historyDescription({ hash: '' } as Parameters<typeof historyDescription>[0]),
      ).toBe('');
      expect(
        historyDescription({} as Parameters<typeof historyDescription>[0]),
      ).toBe('');
    });
  });

  describe('diffSideClass', () => {
    it('happy path: same / removed-left / added-right', () => {
      expect(
        diffSideClass({ left: 'a', right: 'a', type: 'same' }, 'left'),
      ).toBe('same');
      expect(
        diffSideClass({ left: 'a', right: null, type: 'removed' }, 'left'),
      ).toBe('removed');
      expect(
        diffSideClass({ left: null, right: 'b', type: 'added' }, 'right'),
      ).toBe('added');
    });

    it('boundary: blank side when value is null on changed/added/removed rows', () => {
      expect(
        diffSideClass({ left: null, right: 'b', type: 'added' }, 'left'),
      ).toBe('blank');
      expect(
        diffSideClass({ left: 'a', right: null, type: 'removed' }, 'right'),
      ).toBe('blank');
      // changed with both sides present still maps to removed/added classes
      expect(
        diffSideClass({ left: 'a', right: 'b', type: 'changed' }, 'left'),
      ).toBe('removed');
      expect(
        diffSideClass({ left: 'a', right: 'b', type: 'changed' }, 'right'),
      ).toBe('added');
    });
  });
});
