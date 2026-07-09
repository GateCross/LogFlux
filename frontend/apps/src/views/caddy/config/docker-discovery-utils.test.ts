import { describe, expect, it } from 'vitest';

import { buildSiteFromQuickDraft } from './quick-config-utils';
import {
  applyDiscoveryScanToSession,
  buildDraftsFromSelectedCandidates,
  buildQuickSiteDraftFromDockerCandidate,
  clearCandidateSelection,
  createEmptyDiscoverySession,
  discoveryAutoLoadEnabled,
  discoveryAllowsDraftOnly,
  selectAllValidCandidates,
  toggleCandidateSelection,
  validateDiscoverySelection,
  type DockerDiscoveryCandidate,
} from './docker-discovery-utils';

function cand(partial: Partial<DockerDiscoveryCandidate>): DockerDiscoveryCandidate {
  return {
    candidateId: partial.candidateId ?? 'docker-abc-host',
    containerId: partial.containerId ?? 'abc',
    containerName: partial.containerName ?? 'svc',
    status: partial.status ?? 'Up',
    name: partial.name ?? '服务',
    domains: partial.domains ?? ['app.example.com'],
    upstream: partial.upstream ?? 'svc:8080',
    lbPolicy: partial.lbPolicy ?? 'round_robin',
    tlsMode: partial.tlsMode ?? 'auto',
    healthPath: partial.healthPath,
    healthInterval: partial.healthInterval,
    healthTimeout: partial.healthTimeout,
    reason: partial.reason,
    valid: partial.valid ?? true,
  };
}

describe('docker-discovery-utils', () => {
  it('硬约束：禁止自动 /load，允许仅草稿', () => {
    expect(discoveryAutoLoadEnabled()).toBe(false);
    expect(discoveryAllowsDraftOnly()).toBe(true);
  });

  it('applyDiscoveryScanToSession 写入会话且默认勾选有效项', () => {
    const session = applyDiscoveryScanToSession(createEmptyDiscoverySession(), {
      scannedAt: '2026-01-01 12:00:00',
      message: 'ok',
      list: [
        cand({ candidateId: 'a', valid: true }),
        cand({ candidateId: 'b', valid: false, reason: '缺少 host', domains: [] }),
      ],
    });
    expect(session.candidates).toHaveLength(2);
    expect(session.selectedIds).toEqual(['a']);
    expect(session.scannedAt).toBe('2026-01-01 12:00:00');
  });

  it('buildQuickSiteDraftFromDockerCandidate 保留 health/lb/tls 并可 round-trip', () => {
    const draft = buildQuickSiteDraftFromDockerCandidate(
      cand({
        candidateId: 'docker-1-api',
        name: 'API',
        domains: ['api.example.com'],
        upstream: 'api:9000',
        lbPolicy: 'least_conn',
        tlsMode: 'internal',
        healthPath: '/ready',
        healthInterval: '5s',
        healthTimeout: '2s',
      }),
    );
    expect(draft.id).toBe('docker-1-api');
    expect(draft.mode).toBe('reverse_proxy');
    expect(draft.lbPolicy).toBe('least_conn');
    expect(draft.tlsMode).toBe('internal');
    expect(draft.healthCheck).toEqual({
      path: '/ready',
      interval: '5s',
      timeout: '2s',
    });

    const site = buildSiteFromQuickDraft(draft);
    const handle = site.routes[0]?.handles[0];
    expect(handle?.upstream).toBe('api:9000');
    expect(handle?.lbPolicy).toBe('least_conn');
    expect(handle?.healthCheck?.path).toBe('/ready');
  });

  it('关闭 health 时 draft 不含 healthCheck', () => {
    const draft = buildQuickSiteDraftFromDockerCandidate(
      cand({ healthPath: undefined, healthInterval: '10s' }),
    );
    expect(draft.healthCheck).toBeUndefined();
  });

  it('buildDraftsFromSelectedCandidates 仅输出勾选且 valid 的项', () => {
    let session = applyDiscoveryScanToSession(createEmptyDiscoverySession(), {
      list: [
        cand({ candidateId: 'ok1', name: 'A' }),
        cand({ candidateId: 'bad', valid: false, domains: [] }),
        cand({ candidateId: 'ok2', name: 'B' }),
      ],
    });
    session = clearCandidateSelection(session);
    session = toggleCandidateSelection(session, 'ok2', true);
    session = toggleCandidateSelection(session, 'bad', true);
    const drafts = buildDraftsFromSelectedCandidates(session);
    expect(drafts.map((d) => d.name)).toEqual(['B']);
  });

  it('validateDiscoverySelection 空选时报中文错误', () => {
    const session = clearCandidateSelection(
      applyDiscoveryScanToSession(createEmptyDiscoverySession(), {
        list: [cand({ candidateId: 'x' })],
      }),
    );
    expect(validateDiscoverySelection(session)[0]).toMatch(/勾选/);
  });

  it('selectAllValidCandidates 只选 valid', () => {
    const session = selectAllValidCandidates(
      applyDiscoveryScanToSession(createEmptyDiscoverySession(), {
        list: [
          cand({ candidateId: 'v1', valid: true }),
          cand({ candidateId: 'i1', valid: false }),
        ],
      }),
    );
    expect(session.selectedIds).toEqual(['v1']);
  });
});
