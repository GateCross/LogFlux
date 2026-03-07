import { h } from 'vue';
import { type DataTableColumns, NButton, NPopconfirm, NSpace, NTag } from 'naive-ui';
import type {
  WafPolicyFalsePositiveFeedbackItem,
  WafPolicyStatsDimensionItem,
  WafPolicyStatsItem,
  WafPolicyStatsTrendItem
} from '@/service/api/caddy-observe';
import type { WafJobItem, WafReleaseItem } from '@/service/api/caddy-release-job';
import type {
  WafPolicyBindingItem,
  WafPolicyEngineMode,
  WafPolicyItem,
  WafPolicyRevisionItem,
  WafPolicyRevisionStatus,
  WafPolicyScopeType,
  WafRuleExclusionItem
} from '@/service/api/caddy-policy';
import type { WafSourceItem } from '@/service/api/caddy-source';
import type { BindingEffectiveItem } from './composables/useWafBinding';

export function createSourceColumns(options: {
  handleSyncSource: (row: WafSourceItem, activateNow: boolean) => void;
  handleEditSource: (row: WafSourceItem) => void;
  handleDeleteSource: (row: WafSourceItem) => void;
}) {
  const { handleSyncSource, handleEditSource, handleDeleteSource } = options;

  return [
    { title: 'ID', key: 'id', width: 80 },
    { title: '名称', key: 'name', minWidth: 140 },
    {
      title: '类型',
      key: 'kind',
      width: 130,
      render(row: WafSourceItem) {
        return h(
          NTag,
          { type: row.kind === 'crs' ? 'success' : 'warning', bordered: false },
          { default: () => row.kind }
        );
      }
    },
    {
      title: '模式',
      key: 'mode',
      width: 110,
      render(row: WafSourceItem) {
        return h(
          NTag,
          { type: row.mode === 'remote' ? 'info' : 'default', bordered: false },
          { default: () => row.mode }
        );
      }
    },
    {
      title: '地址',
      key: 'url',
      minWidth: 260,
      ellipsis: { tooltip: true },
      render(row: WafSourceItem) {
        return row.url || '-';
      }
    },
    {
      title: '代理',
      key: 'proxyUrl',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render(row: WafSourceItem) {
        return row.proxyUrl || '-';
      }
    },
    {
      title: '调度',
      key: 'schedule',
      width: 160,
      ellipsis: { tooltip: true },
      render: (row: WafSourceItem) => row.schedule || '-'
    },
    {
      title: '开关',
      key: 'switches',
      minWidth: 200,
      render(row: WafSourceItem) {
        const labels = [
          row.enabled ? '启用' : '禁用',
          row.autoCheck ? '自动检查' : '手动检查',
          row.autoDownload ? '自动下载' : '手动下载',
          row.autoActivate ? '自动激活' : '手动激活'
        ];
        return h(
          NSpace,
          { size: 4, wrapItem: true },
          {
            default: () => labels.map(label => h(NTag, { size: 'small', bordered: false }, { default: () => label }))
          }
        );
      }
    },
    {
      title: '最近版本',
      key: 'lastRelease',
      width: 140,
      render: (row: WafSourceItem) => row.lastRelease || '-'
    },
    {
      title: '最近错误',
      key: 'lastError',
      minWidth: 220,
      ellipsis: { tooltip: true },
      render(row: WafSourceItem) {
        if (!row.lastError) return '-';
        return h(NTag, { type: 'error', bordered: false }, { default: () => row.lastError });
      }
    },
    { title: '更新时间', key: 'updatedAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      width: 280,
      fixed: 'right',
      render(row: WafSourceItem) {
        return h(
          NSpace,
          { size: 4 },
          {
            default: () => [
              h(
                NButton,
                {
                  size: 'small',
                  type: 'primary',
                  secondary: true,
                  onClick: () => handleSyncSource(row, false)
                },
                { default: () => '同步' }
              ),
              h(
                NButton,
                {
                  size: 'small',
                  type: 'success',
                  secondary: true,
                  onClick: () => handleSyncSource(row, true)
                },
                { default: () => '同步并激活' }
              ),
              h(NButton, { size: 'small', onClick: () => handleEditSource(row) }, { default: () => '编辑' }),
              h(
                NPopconfirm,
                { onPositiveClick: () => handleDeleteSource(row) },
                {
                  trigger: () =>
                    h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
                  default: () => '删除后不可恢复，确认继续吗？'
                }
              )
            ]
          }
        );
      }
    }
  ] satisfies DataTableColumns<WafSourceItem>;
}

export function createPolicyColumns(options: {
  mapPolicyEngineModeType: (mode: WafPolicyEngineMode) => 'default' | 'warning' | 'error' | 'success' | 'info';
  mapPolicyEngineModeLabel: (mode: WafPolicyEngineMode) => string;
  mapCrsTemplateLabel: (value: WafPolicyItem['crsTemplate']) => string;
  formatBytes: (value: number) => string;
  handlePreviewPolicy: (row: WafPolicyItem) => void;
  handleValidatePolicy: (row: WafPolicyItem) => void;
  handlePublishPolicy: (row: WafPolicyItem) => void;
  handleEditPolicy: (row: WafPolicyItem) => void;
  handleDeletePolicy: (row: WafPolicyItem) => void;
}) {
  const {
    mapPolicyEngineModeType,
    mapPolicyEngineModeLabel,
    mapCrsTemplateLabel,
    formatBytes,
    handlePreviewPolicy,
    handleValidatePolicy,
    handlePublishPolicy,
    handleEditPolicy,
    handleDeletePolicy
  } = options;

  return [
    { title: 'ID', key: 'id', width: 80 },
    { title: '策略名称', key: 'name', minWidth: 180 },
    {
      title: '默认策略',
      key: 'isDefault',
      width: 110,
      render(row: WafPolicyItem) {
        return h(
          NTag,
          { type: row.isDefault ? 'success' : 'default', bordered: false },
          { default: () => (row.isDefault ? '是' : '否') }
        );
      }
    },
    {
      title: '启用',
      key: 'enabled',
      width: 100,
      render(row: WafPolicyItem) {
        return h(
          NTag,
          { type: row.enabled ? 'success' : 'warning', bordered: false },
          { default: () => (row.enabled ? '启用' : '禁用') }
        );
      }
    },
    {
      title: '引擎模式',
      key: 'engineMode',
      width: 170,
      render(row: WafPolicyItem) {
        return h(
          NTag,
          { type: mapPolicyEngineModeType(row.engineMode), bordered: false },
          { default: () => mapPolicyEngineModeLabel(row.engineMode) }
        );
      }
    },
    { title: '审计模式', key: 'auditEngine', width: 130 },
    {
      title: 'CRS 模板',
      key: 'crsTemplate',
      width: 140,
      render(row: WafPolicyItem) {
        return mapCrsTemplateLabel(row.crsTemplate);
      }
    },
    { title: 'PL', key: 'crsParanoiaLevel', width: 90 },
    {
      title: '请求体限制',
      key: 'requestBodyLimit',
      width: 150,
      render: (row: WafPolicyItem) => formatBytes(row.requestBodyLimit)
    },
    { title: '更新时间', key: 'updatedAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      width: 380,
      fixed: 'right',
      render(row: WafPolicyItem) {
        return h(
          NSpace,
          { size: 4 },
          {
            default: () => [
              h(
                NButton,
                {
                  size: 'small',
                  type: 'info',
                  secondary: true,
                  onClick: () => handlePreviewPolicy(row)
                },
                { default: () => '预览' }
              ),
              h(
                NButton,
                {
                  size: 'small',
                  type: 'success',
                  secondary: true,
                  onClick: () => handleValidatePolicy(row)
                },
                { default: () => '校验' }
              ),
              h(
                NButton,
                {
                  size: 'small',
                  type: 'warning',
                  secondary: true,
                  onClick: () => handlePublishPolicy(row)
                },
                { default: () => '发布' }
              ),
              h(NButton, { size: 'small', onClick: () => handleEditPolicy(row) }, { default: () => '编辑' }),
              h(
                NPopconfirm,
                { onPositiveClick: () => handleDeletePolicy(row) },
                {
                  trigger: () =>
                    h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
                  default: () => '删除后不可恢复，确认继续吗？'
                }
              )
            ]
          }
        );
      }
    }
  ] satisfies DataTableColumns<WafPolicyItem>;
}

export function createPolicyRevisionColumns(options: {
  mapPolicyRevisionStatusType: (
    status: WafPolicyRevisionStatus
  ) => 'default' | 'warning' | 'error' | 'success' | 'info';
  mapPolicyRevisionStatusLabel: (status: WafPolicyRevisionStatus) => string;
  displayOperatorName: (value: unknown) => string;
  handleRollbackPolicyRevision: (row: WafPolicyRevisionItem) => void;
}) {
  const {
    mapPolicyRevisionStatusType,
    mapPolicyRevisionStatusLabel,
    displayOperatorName,
    handleRollbackPolicyRevision
  } = options;
  return [
    { title: 'ID', key: 'id', width: 80 },
    {
      title: '策略',
      key: 'policyName',
      minWidth: 180,
      render: (row: WafPolicyRevisionItem) => row.policyName || `#${row.policyId}`
    },
    { title: '策略ID', key: 'policyId', width: 100 },
    {
      title: '版本',
      key: 'version',
      width: 100,
      render: (row: WafPolicyRevisionItem) => `v${row.version}`
    },
    {
      title: '状态',
      key: 'status',
      width: 120,
      render(row: WafPolicyRevisionItem) {
        return h(
          NTag,
          { type: mapPolicyRevisionStatusType(row.status), bordered: false },
          { default: () => mapPolicyRevisionStatusLabel(row.status) }
        );
      }
    },
    {
      title: '操作人',
      key: 'operator',
      width: 120,
      render: (row: WafPolicyRevisionItem) => displayOperatorName(row.operator)
    },
    {
      title: '变更摘要',
      key: 'changeSummary',
      minWidth: 220,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyRevisionItem) => row.changeSummary || row.message || '-'
    },
    {
      title: '描述',
      key: 'message',
      minWidth: 160,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyRevisionItem) => row.message || '-'
    },
    { title: '创建时间', key: 'createdAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      width: 140,
      fixed: 'right',
      render(row: WafPolicyRevisionItem) {
        return h(
          NButton,
          {
            size: 'small',
            type: 'warning',
            secondary: true,
            onClick: () => handleRollbackPolicyRevision(row)
          },
          { default: () => '回滚到此版本' }
        );
      }
    }
  ] satisfies DataTableColumns<WafPolicyRevisionItem>;
}

export function createExclusionColumns(options: {
  mapScopeTypeLabel: (scopeType: WafPolicyScopeType) => string;
  handleEditExclusion: (row: WafRuleExclusionItem) => void;
  handleDeleteExclusion: (row: WafRuleExclusionItem) => void;
}) {
  const { mapScopeTypeLabel, handleEditExclusion, handleDeleteExclusion } = options;
  return [
    { title: 'ID', key: 'id', width: 80 },
    { title: '策略ID', key: 'policyId', width: 100 },
    {
      title: '名称',
      key: 'name',
      minWidth: 160,
      render: (row: WafRuleExclusionItem) => row.name || '-'
    },
    {
      title: '启用',
      key: 'enabled',
      width: 100,
      render(row: WafRuleExclusionItem) {
        return h(
          NTag,
          { type: row.enabled ? 'success' : 'warning', bordered: false },
          { default: () => (row.enabled ? '启用' : '禁用') }
        );
      }
    },
    {
      title: '作用域',
      key: 'scopeType',
      width: 100,
      render: (row: WafRuleExclusionItem) => mapScopeTypeLabel(row.scopeType)
    },
    {
      title: 'Host',
      key: 'host',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: WafRuleExclusionItem) => row.host || '-'
    },
    {
      title: 'Path',
      key: 'path',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: WafRuleExclusionItem) => row.path || '-'
    },
    {
      title: 'Method',
      key: 'method',
      width: 100,
      render: (row: WafRuleExclusionItem) => row.method || '-'
    },
    {
      title: '类型',
      key: 'removeType',
      width: 120,
      render: (row: WafRuleExclusionItem) => (row.removeType === 'id' ? 'removeById' : 'removeByTag')
    },
    {
      title: '移除值',
      key: 'removeValue',
      minWidth: 180,
      ellipsis: { tooltip: true }
    },
    { title: '更新时间', key: 'updatedAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      width: 180,
      fixed: 'right',
      render(row: WafRuleExclusionItem) {
        return h(
          NSpace,
          { size: 4 },
          {
            default: () => [
              h(NButton, { size: 'small', onClick: () => handleEditExclusion(row) }, { default: () => '编辑' }),
              h(
                NPopconfirm,
                { onPositiveClick: () => handleDeleteExclusion(row) },
                {
                  trigger: () =>
                    h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
                  default: () => '删除后不可恢复，确认继续吗？'
                }
              )
            ]
          }
        );
      }
    }
  ] satisfies DataTableColumns<WafRuleExclusionItem>;
}

export function createBindingColumns(options: {
  mapScopeTypeLabel: (scopeType: WafPolicyScopeType) => string;
  handleEditBinding: (row: WafPolicyBindingItem) => void;
  handleDeleteBinding: (row: WafPolicyBindingItem) => void;
}) {
  const { mapScopeTypeLabel, handleEditBinding, handleDeleteBinding } = options;
  return [
    { title: 'ID', key: 'id', width: 80 },
    { title: '策略ID', key: 'policyId', width: 100 },
    {
      title: '名称',
      key: 'name',
      minWidth: 160,
      render: (row: WafPolicyBindingItem) => row.name || '-'
    },
    {
      title: '启用',
      key: 'enabled',
      width: 100,
      render(row: WafPolicyBindingItem) {
        return h(
          NTag,
          { type: row.enabled ? 'success' : 'warning', bordered: false },
          { default: () => (row.enabled ? '启用' : '禁用') }
        );
      }
    },
    {
      title: '作用域',
      key: 'scopeType',
      width: 100,
      render: (row: WafPolicyBindingItem) => mapScopeTypeLabel(row.scopeType)
    },
    {
      title: 'Host',
      key: 'host',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyBindingItem) => row.host || '-'
    },
    {
      title: 'Path',
      key: 'path',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyBindingItem) => row.path || '-'
    },
    {
      title: 'Method',
      key: 'method',
      width: 100,
      render: (row: WafPolicyBindingItem) => row.method || '-'
    },
    { title: '优先级', key: 'priority', width: 100 },
    { title: '更新时间', key: 'updatedAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      width: 180,
      fixed: 'right',
      render(row: WafPolicyBindingItem) {
        return h(
          NSpace,
          { size: 4 },
          {
            default: () => [
              h(NButton, { size: 'small', onClick: () => handleEditBinding(row) }, { default: () => '编辑' }),
              h(
                NPopconfirm,
                { onPositiveClick: () => handleDeleteBinding(row) },
                {
                  trigger: () =>
                    h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
                  default: () => '删除后不可恢复，确认继续吗？'
                }
              )
            ]
          }
        );
      }
    }
  ] satisfies DataTableColumns<WafPolicyBindingItem>;
}

export function createBindingEffectiveColumns(options: { mapScopeTypeLabel: (scopeType: string) => string }) {
  const { mapScopeTypeLabel } = options;
  return [
    { title: '顺位', key: 'order', width: 80 },
    {
      title: '策略',
      key: 'policyName',
      minWidth: 180,
      render: (row: BindingEffectiveItem) => row.policyName || `#${row.policyId}`
    },
    {
      title: '作用域',
      key: 'scopeType',
      width: 100,
      render: (row: BindingEffectiveItem) => mapScopeTypeLabel(row.scopeType)
    },
    {
      title: 'Host',
      key: 'host',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: BindingEffectiveItem) => row.host || '-'
    },
    {
      title: 'Path',
      key: 'path',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: BindingEffectiveItem) => row.path || '-'
    },
    {
      title: 'Method',
      key: 'method',
      width: 100,
      render: (row: BindingEffectiveItem) => row.method || '-'
    },
    { title: '优先级', key: 'priority', width: 100 }
  ] satisfies DataTableColumns<BindingEffectiveItem>;
}

export function createPolicyStatsTrendColumns() {
  return [
    { title: '时间', key: 'time', width: 140 },
    { title: '命中', key: 'hitCount', width: 100 },
    { title: '拦截', key: 'blockedCount', width: 100 },
    { title: '放行', key: 'allowedCount', width: 100 }
  ] satisfies DataTableColumns<WafPolicyStatsTrendItem>;
}

export function createPolicyStatsColumns(options: { formatRatePercent: (value: number) => string }) {
  const { formatRatePercent } = options;
  return [
    {
      title: '策略',
      key: 'policyName',
      minWidth: 180,
      render: (row: WafPolicyStatsItem) => row.policyName || `#${row.policyId}`
    },
    { title: '命中', key: 'hitCount', width: 100 },
    { title: '拦截', key: 'blockedCount', width: 100 },
    { title: '放行', key: 'allowedCount', width: 100 },
    { title: '疑似误报', key: 'suspectedFalsePositiveCount', width: 120 },
    {
      title: '拦截率',
      key: 'blockRate',
      width: 120,
      render: (row: WafPolicyStatsItem) => formatRatePercent(row.blockRate)
    }
  ] satisfies DataTableColumns<WafPolicyStatsItem>;
}

export function createPolicyStatsDimensionColumns(options: { formatRatePercent: (value: number) => string }) {
  const { formatRatePercent } = options;
  return [
    {
      title: '维度值',
      key: 'key',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyStatsDimensionItem) => row.key || '-'
    },
    { title: '命中', key: 'hitCount', width: 100 },
    { title: '拦截', key: 'blockedCount', width: 100 },
    { title: '放行', key: 'allowedCount', width: 100 },
    {
      title: '拦截率',
      key: 'blockRate',
      width: 120,
      render: (row: WafPolicyStatsDimensionItem) => formatRatePercent(row.blockRate)
    }
  ] satisfies DataTableColumns<WafPolicyStatsDimensionItem>;
}

export function createPolicyFeedbackColumns(options: {
  displayOperatorName: (value: unknown) => string;
  mapPolicyFeedbackStatusTagType: (status: string) => 'default' | 'warning' | 'success';
  mapPolicyFeedbackStatusLabel: (status: string) => string;
  mapPolicyFeedbackSLAStatusTagType: (row: WafPolicyFalsePositiveFeedbackItem) => 'default' | 'warning' | 'success';
  mapPolicyFeedbackSLAStatusLabel: (row: WafPolicyFalsePositiveFeedbackItem) => string;
  handleCreateExclusionDraftFromFeedback: (row: WafPolicyFalsePositiveFeedbackItem) => void;
  openPolicyFeedbackProcessModal: (row: WafPolicyFalsePositiveFeedbackItem) => void;
}) {
  const {
    displayOperatorName,
    mapPolicyFeedbackStatusTagType,
    mapPolicyFeedbackStatusLabel,
    mapPolicyFeedbackSLAStatusTagType,
    mapPolicyFeedbackSLAStatusLabel,
    handleCreateExclusionDraftFromFeedback,
    openPolicyFeedbackProcessModal
  } = options;

  return [
    { type: 'selection', width: 48 },
    {
      title: '策略',
      key: 'policyName',
      minWidth: 160,
      render: (row: WafPolicyFalsePositiveFeedbackItem) => row.policyName || `#${row.policyId}`
    },
    {
      title: 'Host',
      key: 'host',
      minWidth: 160,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyFalsePositiveFeedbackItem) => row.host || '-'
    },
    {
      title: 'Path',
      key: 'path',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyFalsePositiveFeedbackItem) => row.path || '-'
    },
    {
      title: 'Method',
      key: 'method',
      width: 100,
      render: (row: WafPolicyFalsePositiveFeedbackItem) => row.method || '-'
    },
    {
      title: '状态码',
      key: 'status',
      width: 100,
      render: (row: WafPolicyFalsePositiveFeedbackItem) => (row.status > 0 ? row.status : '-')
    },
    {
      title: '处理状态',
      key: 'feedbackStatus',
      width: 110,
      render: (row: WafPolicyFalsePositiveFeedbackItem) =>
        h(
          NTag,
          {
            bordered: false,
            type: mapPolicyFeedbackStatusTagType(row.feedbackStatus)
          },
          { default: () => mapPolicyFeedbackStatusLabel(row.feedbackStatus) }
        )
    },
    {
      title: '责任人',
      key: 'assignee',
      width: 120,
      render: (row: WafPolicyFalsePositiveFeedbackItem) => displayOperatorName(row.assignee)
    },
    {
      title: '截止时间',
      key: 'dueAt',
      width: 180,
      render: (row: WafPolicyFalsePositiveFeedbackItem) => row.dueAt || '-'
    },
    {
      title: 'SLA',
      key: 'isOverdue',
      width: 90,
      render: (row: WafPolicyFalsePositiveFeedbackItem) =>
        h(
          NTag,
          { bordered: false, type: mapPolicyFeedbackSLAStatusTagType(row) },
          { default: () => mapPolicyFeedbackSLAStatusLabel(row) }
        )
    },
    {
      title: '误报原因',
      key: 'reason',
      minWidth: 220,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyFalsePositiveFeedbackItem) => row.reason || '-'
    },
    {
      title: '建议动作',
      key: 'suggestion',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyFalsePositiveFeedbackItem) => row.suggestion || '-'
    },
    {
      title: '处理备注',
      key: 'processNote',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row: WafPolicyFalsePositiveFeedbackItem) => row.processNote || '-'
    },
    {
      title: '处理人',
      key: 'processedBy',
      width: 120,
      render: (row: WafPolicyFalsePositiveFeedbackItem) => displayOperatorName(row.processedBy)
    },
    {
      title: '处理时间',
      key: 'processedAt',
      width: 180,
      render: (row: WafPolicyFalsePositiveFeedbackItem) => row.processedAt || '-'
    },
    {
      title: '提交人',
      key: 'operator',
      width: 120,
      render: (row: WafPolicyFalsePositiveFeedbackItem) => displayOperatorName(row.operator)
    },
    { title: '提交时间', key: 'createdAt', width: 180 },
    {
      title: '操作',
      key: 'actions',
      width: 230,
      fixed: 'right',
      render: (row: WafPolicyFalsePositiveFeedbackItem) =>
        h(
          NSpace,
          { size: 6 },
          {
            default: () => [
              h(
                NButton,
                {
                  size: 'small',
                  tertiary: true,
                  type: 'info',
                  onClick: () => handleCreateExclusionDraftFromFeedback(row)
                },
                { default: () => '生成例外草稿' }
              ),
              h(
                NButton,
                {
                  size: 'small',
                  tertiary: true,
                  type: 'warning',
                  onClick: () => openPolicyFeedbackProcessModal(row)
                },
                { default: () => '处理' }
              )
            ]
          }
        )
    }
  ] satisfies DataTableColumns<WafPolicyFalsePositiveFeedbackItem>;
}

export function createReleaseColumns(options: {
  mapSourceNameById: (sourceId: number) => string;
  formatBytes: (value: number) => string;
  mapReleaseStatusType: (status: WafReleaseItem['status']) => 'default' | 'warning' | 'error' | 'success' | 'info';
  handleActivateRelease: (row: WafReleaseItem) => void;
}) {
  const { mapSourceNameById, formatBytes, mapReleaseStatusType, handleActivateRelease } = options;
  return [
    { title: 'ID', key: 'id', width: 80 },
    {
      title: '更新源',
      key: 'sourceName',
      minWidth: 160,
      render: (row: WafReleaseItem) => mapSourceNameById(row.sourceId)
    },
    {
      title: '版本',
      key: 'version',
      minWidth: 180,
      ellipsis: { tooltip: true }
    },
    { title: '包类型', key: 'artifactType', width: 110 },
    {
      title: '大小',
      key: 'sizeBytes',
      width: 120,
      render: (row: WafReleaseItem) => formatBytes(row.sizeBytes)
    },
    {
      title: '校验值',
      key: 'checksum',
      minWidth: 220,
      ellipsis: { tooltip: true },
      render: (row: WafReleaseItem) => row.checksum || '-'
    },
    {
      title: '状态',
      key: 'status',
      width: 120,
      render(row: WafReleaseItem) {
        return h(NTag, { type: mapReleaseStatusType(row.status), bordered: false }, { default: () => row.status });
      }
    },
    {
      title: '路径',
      key: 'storagePath',
      minWidth: 260,
      ellipsis: { tooltip: true }
    },
    { title: '更新时间', key: 'updatedAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      width: 120,
      fixed: 'right',
      render(row: WafReleaseItem) {
        return h(
          NSpace,
          { size: 4 },
          {
            default: () => [
              h(
                NButton,
                {
                  size: 'small',
                  type: 'primary',
                  secondary: true,
                  disabled: row.status === 'active',
                  onClick: () => handleActivateRelease(row)
                },
                { default: () => '激活' }
              )
            ]
          }
        );
      }
    }
  ] satisfies DataTableColumns<WafReleaseItem>;
}

export function createJobColumns(options: {
  mapJobSourceName: (row: WafJobItem) => string;
  mapJobActionLabel: (action: string) => string;
  mapJobTriggerModeLabel: (triggerMode: string) => string;
  mapJobStatusType: (status: WafJobItem['status']) => 'default' | 'warning' | 'error' | 'success' | 'info';
  mapJobStatusLabel: (status: string) => string;
  displayOperatorName: (value: unknown) => string;
  mapJobMessage: (message: string) => string;
}) {
  const {
    mapJobSourceName,
    mapJobActionLabel,
    mapJobTriggerModeLabel,
    mapJobStatusType,
    mapJobStatusLabel,
    displayOperatorName,
    mapJobMessage
  } = options;
  return [
    { title: 'ID', key: 'id', width: 80 },
    {
      title: '更新源',
      key: 'sourceName',
      minWidth: 160,
      render: (row: WafJobItem) => mapJobSourceName(row)
    },
    {
      title: '动作',
      key: 'action',
      width: 120,
      render: (row: WafJobItem) => mapJobActionLabel(row.action)
    },
    {
      title: '触发方式',
      key: 'triggerMode',
      width: 120,
      render: (row: WafJobItem) => mapJobTriggerModeLabel(row.triggerMode)
    },
    {
      title: '状态',
      key: 'status',
      width: 110,
      render(row: WafJobItem) {
        return h(
          NTag,
          { type: mapJobStatusType(row.status), bordered: false },
          { default: () => mapJobStatusLabel(row.status) }
        );
      }
    },
    {
      title: '操作人',
      key: 'operator',
      width: 120,
      render: (row: WafJobItem) => displayOperatorName(row.operator)
    },
    {
      title: '开始时间',
      key: 'startedAt',
      width: 180,
      render: (row: WafJobItem) => row.startedAt || '-'
    },
    {
      title: '结束时间',
      key: 'finishedAt',
      width: 180,
      render: (row: WafJobItem) => row.finishedAt || '-'
    },
    {
      title: '消息',
      key: 'message',
      minWidth: 320,
      ellipsis: { tooltip: true },
      render: (row: WafJobItem) => mapJobMessage(row.message)
    }
  ] satisfies DataTableColumns<WafJobItem>;
}
