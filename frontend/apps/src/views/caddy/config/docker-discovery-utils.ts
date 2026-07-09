/**
 * Docker 发现 → 会话候选草稿（纯函数层）
 *
 * 约束（Req 1.2 / design Phase 2）：
 * - 仅内存/会话候选，不建平行 discovery DB
 * - 默认不 /load；dry-run Preview → 用户确认 → 既有 Apply_Path
 */

import {
  createQuickSiteDraft,
  type QuickLbPolicy,
  type QuickSiteDraft,
  type QuickTlsMode,
} from './quick-config-utils';
import type { HealthCheck } from './types';

/** 后端 Docker 发现候选项（与 GET /caddy/discovery/docker 对齐） */
export interface DockerDiscoveryCandidate {
  candidateId: string;
  containerId: string;
  containerName: string;
  status: string;
  name: string;
  domains: string[];
  upstream: string;
  lbPolicy?: string;
  tlsMode: string;
  healthPath?: string;
  healthInterval?: string;
  healthTimeout?: string;
  reason?: string;
  valid: boolean;
}

export interface DockerDiscoverySessionState {
  /** 最近一次扫描时间（展示用） */
  scannedAt: string;
  /** 后端 message（含不可用说明） */
  message: string;
  /** 会话内候选列表（可编辑选择，不落库） */
  candidates: DockerDiscoveryCandidate[];
  /** 用户勾选的 candidateId 集合 */
  selectedIds: string[];
}

const SIMPLE_LB = new Set<string>(['round_robin', 'least_conn', 'ip_hash']);
const TLS_MODES = new Set<string>(['auto', 'off', 'internal']);

export function createEmptyDiscoverySession(): DockerDiscoverySessionState {
  return {
    scannedAt: '',
    message: '',
    candidates: [],
    selectedIds: [],
  };
}

/**
 * 将 API 响应写入会话状态（覆盖候选；保留仍有效的勾选）。
 * 纯内存，无持久化。
 */
export function applyDiscoveryScanToSession(
  prev: DockerDiscoverySessionState,
  payload: {
    list?: DockerDiscoveryCandidate[] | null;
    scannedAt?: string;
    message?: string;
  },
): DockerDiscoverySessionState {
  const candidates = Array.isArray(payload.list) ? payload.list.map(normalizeCandidate) : [];
  const validIds = new Set(candidates.map((c) => c.candidateId));
  const selectedIds = (prev.selectedIds ?? []).filter((id) => validIds.has(id));
  // 首次扫描：默认勾选全部 valid 项
  const nextSelected =
    selectedIds.length > 0
      ? selectedIds
      : candidates.filter((c) => c.valid).map((c) => c.candidateId);

  return {
    scannedAt: payload.scannedAt?.trim() || prev.scannedAt || '',
    message: payload.message?.trim() || '',
    candidates,
    selectedIds: nextSelected,
  };
}

function normalizeCandidate(raw: DockerDiscoveryCandidate): DockerDiscoveryCandidate {
  return {
    candidateId: String(raw.candidateId ?? ''),
    containerId: String(raw.containerId ?? ''),
    containerName: String(raw.containerName ?? ''),
    status: String(raw.status ?? ''),
    name: String(raw.name ?? ''),
    domains: Array.isArray(raw.domains)
      ? raw.domains.map((d) => String(d).trim()).filter(Boolean)
      : [],
    upstream: String(raw.upstream ?? '').trim(),
    lbPolicy: raw.lbPolicy ? String(raw.lbPolicy) : undefined,
    tlsMode: String(raw.tlsMode ?? 'auto'),
    healthPath: raw.healthPath ? String(raw.healthPath) : undefined,
    healthInterval: raw.healthInterval ? String(raw.healthInterval) : undefined,
    healthTimeout: raw.healthTimeout ? String(raw.healthTimeout) : undefined,
    reason: raw.reason ? String(raw.reason) : undefined,
    valid: Boolean(raw.valid),
  };
}

export function toggleCandidateSelection(
  state: DockerDiscoverySessionState,
  candidateId: string,
  selected: boolean,
): DockerDiscoverySessionState {
  const set = new Set(state.selectedIds);
  if (selected) set.add(candidateId);
  else set.delete(candidateId);
  return { ...state, selectedIds: [...set] };
}

export function selectAllValidCandidates(
  state: DockerDiscoverySessionState,
): DockerDiscoverySessionState {
  return {
    ...state,
    selectedIds: state.candidates.filter((c) => c.valid).map((c) => c.candidateId),
  };
}

export function clearCandidateSelection(
  state: DockerDiscoverySessionState,
): DockerDiscoverySessionState {
  return { ...state, selectedIds: [] };
}

function toLbPolicy(value?: string): QuickLbPolicy {
  const v = (value ?? 'round_robin').trim();
  if (SIMPLE_LB.has(v)) return v as QuickLbPolicy;
  return 'round_robin';
}

function toTlsMode(value?: string): QuickTlsMode {
  const v = (value ?? 'auto').trim();
  if (TLS_MODES.has(v)) return v as QuickTlsMode;
  return 'auto';
}

function buildHealthCheck(c: DockerDiscoveryCandidate): HealthCheck | undefined {
  const path = c.healthPath?.trim();
  if (!path) return undefined;
  return {
    path,
    interval: c.healthInterval?.trim() || undefined,
    timeout: c.healthTimeout?.trim() || undefined,
  };
}

/**
 * 将单个 Docker 候选转为 QuickSiteDraft（会话草稿，不触发 API）。
 * 使用 candidateId 作为 draft.id，便于再次导入时覆盖同会话项。
 */
export function buildQuickSiteDraftFromDockerCandidate(
  c: DockerDiscoveryCandidate,
): QuickSiteDraft {
  const domains = (c.domains ?? []).map((d) => d.trim()).filter(Boolean);
  const name = c.name?.trim() || domains[0] || c.containerName || 'Docker 站点';
  return createQuickSiteDraft({
    id: c.candidateId || undefined,
    name,
    enabled: true,
    domains,
    tlsMode: toTlsMode(c.tlsMode),
    mode: 'reverse_proxy',
    upstream: (c.upstream ?? '').trim(),
    lbPolicy: toLbPolicy(c.lbPolicy),
    healthCheck: buildHealthCheck(c),
  });
}

/**
 * 将勾选的有效候选转为草稿列表（顺序与候选列表一致）。
 */
export function buildDraftsFromSelectedCandidates(
  state: DockerDiscoverySessionState,
): QuickSiteDraft[] {
  const selected = new Set(state.selectedIds);
  return state.candidates
    .filter((c) => c.valid && selected.has(c.candidateId))
    .map(buildQuickSiteDraftFromDockerCandidate);
}

/**
 * 校验：至少选中一个有效候选。中文错误。
 */
export function validateDiscoverySelection(state: DockerDiscoverySessionState): string[] {
  const drafts = buildDraftsFromSelectedCandidates(state);
  if (drafts.length === 0) {
    return ['请至少勾选一个有效的发现候选'];
  }
  const errors: string[] = [];
  for (const d of drafts) {
    if (!d.domains.length) {
      errors.push(`候选「${d.name}」缺少域名`);
    }
    if (!d.upstream.trim()) {
      errors.push(`候选「${d.name}」缺少上游地址`);
    }
  }
  return errors;
}

/** 硬约束：发现路径禁止自动 /load */
export function discoveryAutoLoadEnabled(): false {
  return false;
}

/** 允许仅写入会话草稿 */
export function discoveryAllowsDraftOnly(): true {
  return true;
}

/**
 * 标签约定说明（中文，供 UI 展示）。
 */
export const DOCKER_DISCOVERY_LABEL_HELP: string[] = [
  'logflux.enable=true — 纳入发现（必填）',
  'logflux.host=app.example.com — 域名，逗号分隔多个（必填）',
  'logflux.port=8080 — 容器内端口（可选，默认可推断）',
  'logflux.upstream=host:port — 上游覆盖（可选）',
  'logflux.name / logflux.tls / logflux.lb_policy — 可选',
  'logflux.health.path / interval / timeout — 可选健康检查',
];
