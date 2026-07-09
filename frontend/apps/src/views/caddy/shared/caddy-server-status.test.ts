import { describe, expect, it } from 'vitest';

import {
  formatLatency,
  mergeServerStatusRows,
  statusErrorSummary,
  statusLabel,
  statusTagColor,
  statusTooltip,
} from './caddy-server-status';

describe('caddy-server-status shared helpers', () => {
  it('formatLatency', () => {
    expect(formatLatency(undefined)).toBe('—');
    expect(formatLatency(12.6)).toBe('13 ms');
  });

  it('status labels', () => {
    expect(statusLabel(undefined)).toBe('未探测');
    expect(
      statusLabel({
        serverId: 1,
        name: 'n',
        reachable: true,
        latencyMs: 12,
        probedAt: '',
      }),
    ).toBe('在线');
    expect(
      statusTagColor({
        serverId: 1,
        name: 'n',
        reachable: false,
        latencyMs: 0,
        probedAt: '',
      }),
    ).toBe('error');
    expect(
      statusErrorSummary({
        serverId: 1,
        name: 'n',
        reachable: false,
        latencyMs: 0,
        probedAt: '',
        errorMessage: '超时',
      }),
    ).toBe('超时');
  });

  it('statusTooltip', () => {
    expect(statusTooltip(undefined)).toContain('尚未探测');
    expect(
      statusTooltip({
        serverId: 1,
        name: 'n',
        reachable: true,
        latencyMs: 8,
        probedAt: 't',
      }),
    ).toContain('8 ms');
  });

  it('mergeServerStatusRows', () => {
    const rows = mergeServerStatusRows(
      [{ id: 1, name: 'a', url: 'http://x' }],
      [
        {
          serverId: 1,
          name: 'a',
          reachable: true,
          latencyMs: 3,
          probedAt: 'now',
        },
      ],
      (s) => s.name,
    );
    expect(rows).toHaveLength(1);
    expect(rows[0]?.status?.latencyMs).toBe(3);
  });
});
