import { describe, expect, it } from 'vitest';

import { createEmptyServerForm, serverLabel } from './useCaddyServers';

describe('useCaddyServers pure helpers', () => {
  describe('createEmptyServerForm', () => {
    it('happy path: returns default local server form', () => {
      expect(createEmptyServerForm()).toEqual({
        id: undefined,
        name: '',
        token: '',
        type: 'local',
        url: 'http://localhost:2019',
      });
    });

    it('boundary: each call returns a fresh object (no shared mutable state)', () => {
      const a = createEmptyServerForm();
      const b = createEmptyServerForm();
      expect(a).not.toBe(b);
      a.name = 'mutated';
      a.url = 'http://other';
      expect(b.name).toBe('');
      expect(b.url).toBe('http://localhost:2019');
    });
  });

  describe('serverLabel', () => {
    it('happy path: prefers name, then url', () => {
      expect(
        serverLabel({
          id: 1,
          name: 'prod',
          url: 'http://caddy:2019',
        } as Parameters<typeof serverLabel>[0]),
      ).toBe('prod');
      expect(
        serverLabel({
          id: 2,
          name: '',
          url: 'http://caddy:2019',
        } as Parameters<typeof serverLabel>[0]),
      ).toBe('http://caddy:2019');
    });

    it('boundary: empty name and url fall back to Server #id', () => {
      expect(
        serverLabel({
          id: 42,
          name: '',
          url: '',
        } as Parameters<typeof serverLabel>[0]),
      ).toBe('Server #42');
      expect(
        serverLabel({
          id: 0,
          name: undefined,
          url: undefined,
        } as unknown as Parameters<typeof serverLabel>[0]),
      ).toBe('Server #0');
    });
  });
});
