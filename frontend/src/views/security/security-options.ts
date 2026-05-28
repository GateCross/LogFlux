import type { WafPolicyCrsTemplate, WafPolicyRemoveType, WafPolicyScopeType } from '@/service/api/caddy-policy';
import type { WafMode } from '@/service/api/caddy-source';

export const modeOptions = [
  { label: '远程同步 (remote)', value: 'remote' as WafMode },
  { label: '手动管理 (manual)', value: 'manual' as WafMode }
];

export const authTypeOptions = [
  { label: '无鉴权', value: 'none' },
  { label: 'Token', value: 'token' },
  { label: 'Basic', value: 'basic' }
];

export const policyEngineModeOptions = [
  { label: 'On（阻断）', value: 'on' },
  { label: 'Off（关闭）', value: 'off' },
  { label: 'DetectionOnly（仅检测）', value: 'detectiononly' }
];

export const policyAuditEngineOptions = [
  { label: 'RelevantOnly（推荐）', value: 'relevantonly' },
  { label: 'On（全量）', value: 'on' },
  { label: 'Off（关闭）', value: 'off' }
];

export const policyAuditLogFormatOptions = [
  { label: 'JSON', value: 'json' },
  { label: 'Native', value: 'native' }
];

export const scopeTypeOptions = [
  { label: '全局', value: 'global' as WafPolicyScopeType },
  { label: '站点', value: 'site' as WafPolicyScopeType },
  { label: '路由', value: 'route' as WafPolicyScopeType }
];

export const removeTypeOptions = [
  { label: 'removeById', value: 'id' as WafPolicyRemoveType },
  { label: 'removeByTag', value: 'tag' as WafPolicyRemoveType }
];

export const methodOptions = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'DELETE', value: 'DELETE' },
  { label: 'OPTIONS', value: 'OPTIONS' },
  { label: 'HEAD', value: 'HEAD' }
];

export const policyFeedbackStatusOptions = [
  { label: '待确认', value: 'pending' as const },
  { label: '已确认', value: 'confirmed' as const },
  { label: '已处理', value: 'resolved' as const }
];

export const policyFeedbackStatusFilterOptions = [{ label: '全部状态', value: '' }, ...policyFeedbackStatusOptions];

export const policyFeedbackSLAStatusOptions = [
  { label: '全部SLA', value: 'all' as const },
  { label: '正常', value: 'normal' as const },
  { label: '已超时', value: 'overdue' as const },
  { label: '已解决', value: 'resolved' as const }
];

export const crsTemplatePresetMap: Record<
  Exclude<WafPolicyCrsTemplate, 'custom'>,
  {
    crsParanoiaLevel: number;
    crsInboundAnomalyThreshold: number;
    crsOutboundAnomalyThreshold: number;
  }
> = {
  low_fp: {
    crsParanoiaLevel: 1,
    crsInboundAnomalyThreshold: 10,
    crsOutboundAnomalyThreshold: 8
  },
  balanced: {
    crsParanoiaLevel: 2,
    crsInboundAnomalyThreshold: 5,
    crsOutboundAnomalyThreshold: 4
  },
  high_blocking: {
    crsParanoiaLevel: 3,
    crsInboundAnomalyThreshold: 3,
    crsOutboundAnomalyThreshold: 2
  }
};

export const releaseStatusOptions = [
  { label: '全部', value: '' },
  { label: 'downloaded', value: 'downloaded' },
  { label: 'verified', value: 'verified' },
  { label: 'active', value: 'active' },
  { label: 'failed', value: 'failed' },
  { label: 'rolled_back', value: 'rolled_back' }
];

export const jobStatusOptions = [
  { label: '全部', value: '' },
  { label: 'running', value: 'running' },
  { label: 'success', value: 'success' },
  { label: 'failed', value: 'failed' }
];

export const jobActionOptions = [
  { label: '全部', value: '' },
  { label: '检查', value: 'check' },
  { label: '下载', value: 'download' },
  { label: '校验', value: 'verify' },
  { label: '激活', value: 'activate' },
  { label: '回滚', value: 'rollback' },
  { label: '引擎检查', value: 'engine_check' }
];
