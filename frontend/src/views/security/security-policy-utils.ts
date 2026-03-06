import type { WafPolicyCrsTemplate, WafPolicyEngineMode, WafPolicyRevisionStatus, WafPolicyScopeType } from '@/service/api/caddy-policy';

export function mapPolicyEngineModeLabel(mode: WafPolicyEngineMode) {
  switch (mode) {
    case 'on':
      return 'On（阻断）';
    case 'detectiononly':
      return 'DetectionOnly（仅检测）';
    case 'off':
      return 'Off（关闭）';
    default:
      return mode || '-';
  }
}

export function mapCrsTemplateLabel(template: WafPolicyCrsTemplate | string) {
  switch (template) {
    case 'low_fp':
      return '低误报';
    case 'balanced':
      return '平衡';
    case 'high_blocking':
      return '高拦截';
    case 'custom':
      return '自定义';
    default:
      return template || '-';
  }
}

export function mapScopeTypeLabel(scopeType: WafPolicyScopeType | string) {
  switch (scopeType) {
    case 'global':
      return '全局';
    case 'site':
      return '站点';
    case 'route':
      return '路由';
    default:
      return scopeType || '-';
  }
}

export function mapPolicyRevisionStatusLabel(status: WafPolicyRevisionStatus) {
  switch (status) {
    case 'draft':
      return '草稿';
    case 'published':
      return '已发布';
    case 'rolled_back':
      return '已回滚';
    default:
      return status || '-';
  }
}

export function formatBytes(size: number) {
  const value = Number(size || 0);
  if (!Number.isFinite(value) || value <= 0) {
    return '-';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let current = value;
  let index = 0;
  while (current >= 1024 && index < units.length - 1) {
    current /= 1024;
    index += 1;
  }
  return `${current.toFixed(current >= 10 || index === 0 ? 0 : 2)} ${units[index]}`;
}

export function buildPolicyWorkspaceActions(options: {
  activeSection: 'runtime' | 'crs' | 'exclusion' | 'binding';
  hasPendingCrsTuningChanges: boolean;
  bindingConflictCount: number;
  selectedPolicyName: string;
}) {
  const actions: string[] = [];
  if (options.activeSection === 'runtime') {
    actions.push(`当前在基础设置区，可直接对策略 ${options.selectedPolicyName || '-'} 执行预览、校验和发布。`);
  }
  if (options.activeSection === 'crs') {
    actions.push(options.hasPendingCrsTuningChanges ? '当前 CRS 调优存在未保存改动，发布前会要求先保存。' : '当前 CRS 调优参数已与策略持久化状态一致。');
  }
  if (options.activeSection === 'binding') {
    actions.push(options.bindingConflictCount > 0 ? `当前存在 ${options.bindingConflictCount} 组绑定冲突，发布前需先处理。` : '当前未发现绑定冲突，可继续发布验证。');
  }
  if (options.activeSection === 'exclusion') {
    actions.push('规则例外会直接影响误报治理效果，建议在观测结果确认后再新增或调整。');
  }
  return actions;
}
