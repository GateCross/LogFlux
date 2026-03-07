<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { type FormInst, type FormRules, NSelect, NSwitch, type UploadFileInfo, useDialog, useMessage } from 'naive-ui';
import { type WafKind, type WafSourceItem, fetchWafSourceList, uploadWafPackage } from '@/service/api/caddy-source';
import {
  type WafPolicyCrsTemplate,
  type WafPolicyEngineMode,
  type WafPolicyRemoveType,
  type WafPolicyRevisionStatus,
  type WafPolicyScopeType
} from '@/service/api/caddy-policy';
import { type WafPolicyFalsePositiveFeedbackItem } from '@/service/api/caddy-observe';
import { type WafJobItem, type WafJobStatus, type WafReleaseStatus } from '@/service/api/caddy-release-job';
import { request } from '@/service/request';
import {
  buildPolicyWorkspaceActions,
  formatBytes,
  mapCrsTemplateLabel,
  mapPolicyEngineModeLabel,
  mapPolicyRevisionStatusLabel,
  mapScopeTypeLabel
} from './security-policy-utils';
import {
  createBindingColumns,
  createBindingEffectiveColumns,
  createExclusionColumns,
  createJobColumns,
  createPolicyColumns,
  createPolicyFeedbackColumns,
  createPolicyRevisionColumns,
  createPolicyStatsColumns,
  createPolicyStatsDimensionColumns,
  createPolicyStatsTrendColumns,
  createReleaseColumns,
  createSourceColumns
} from './security-columns';
import { SECURITY_MENU_SCHEMA, type SecurityMenuKey, type SecurityTabKey } from './navigation';
import SecuritySourcePage from './pages/SecuritySourcePage.vue';
import SecurityPolicyPage from './pages/SecurityPolicyPage.vue';
import SecurityObservePage from './pages/SecurityObservePage.vue';
import SecurityOpsPage from './pages/SecurityOpsPage.vue';
import { useSecurityNavigation } from './composables/useSecurityNavigation';
import { useWafPolicy } from './composables/useWafPolicy';
import { useWafObserve } from './composables/useWafObserve';
import { useWafObserveFeedback } from './composables/useWafObserveFeedback';
import { useWafObserveExport } from './composables/useWafObserveExport';
import { useWafReleaseJob } from './composables/useWafReleaseJob';
import { useWafCrsTuning } from './composables/useWafCrsTuning';
import { useWafExclusion } from './composables/useWafExclusion';
import { useWafBinding } from './composables/useWafBinding';
import { useObserveDrilldown } from './composables/useObserveDrilldown';
import { usePolicyFeedbackExclusionDraft } from './composables/usePolicyFeedbackExclusionDraft';
import { useSecurityRefresh } from './composables/useSecurityRefresh';
import { useWafSource } from './composables/useWafSource';
import { useWafSourceRuntime } from './composables/useWafSourceRuntime';

const message = useMessage();
const dialog = useDialog();
const route = useRoute();
const router = useRouter();
const { activeMenu, activeTab, pageTitle, navigateToSecurityTab } = useSecurityNavigation({
  route,
  router
});

const securityMenus = Object.values(SECURITY_MENU_SCHEMA);
const securityMenuDescriptionMap: Record<SecurityMenuKey, string> = {
  source: '更新源、规则包上传与引擎检查。',
  policy: '运行模式、CRS、例外和绑定统一收束。',
  observe: '效果分析、下钻和误报处置。',
  ops: '版本发布、回滚、任务审计与清理。'
};
const securityPolicySectionLabelMap = {
  runtime: '基础设置',
  crs: 'CRS 调优',
  exclusion: '规则例外',
  binding: '策略绑定'
} as const;
const activeMenuDescription = computed(() => securityMenuDescriptionMap[activeMenu.value]);
const activePolicySection = computed(() => {
  if (activeTab.value === 'crs' || activeTab.value === 'exclusion' || activeTab.value === 'binding') {
    return activeTab.value;
  }
  return 'runtime';
});
const activeOpsSection = computed(() => (activeTab.value === 'job' ? 'job' : 'release'));
const observeActiveView = ref<'analysis' | 'feedback'>('analysis');

const tableFixedHeight = 480;

const modeOptions = [
  { label: '远程同步 (remote)', value: 'remote' },
  { label: '手动管理 (manual)', value: 'manual' }
];

const authTypeOptions = [
  { label: '无鉴权', value: 'none' },
  { label: 'Token', value: 'token' },
  { label: 'Basic', value: 'basic' }
];

const policyEngineModeOptions = [
  { label: 'On（阻断）', value: 'on' },
  { label: 'Off（关闭）', value: 'off' },
  { label: 'DetectionOnly（仅检测）', value: 'detectiononly' }
];

const policyAuditEngineOptions = [
  { label: 'RelevantOnly（推荐）', value: 'relevantonly' },
  { label: 'On（全量）', value: 'on' },
  { label: 'Off（关闭）', value: 'off' }
];

const policyAuditLogFormatOptions = [
  { label: 'JSON', value: 'json' },
  { label: 'Native', value: 'native' }
];

const scopeTypeOptions = [
  { label: '全局', value: 'global' as WafPolicyScopeType },
  { label: '站点', value: 'site' as WafPolicyScopeType },
  { label: '路由', value: 'route' as WafPolicyScopeType }
];

const removeTypeOptions = [
  { label: 'removeById', value: 'id' as WafPolicyRemoveType },
  { label: 'removeByTag', value: 'tag' as WafPolicyRemoveType }
];

const methodOptions = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'DELETE', value: 'DELETE' },
  { label: 'OPTIONS', value: 'OPTIONS' },
  { label: 'HEAD', value: 'HEAD' }
];

const policyFeedbackStatusOptions = [
  { label: '待确认', value: 'pending' as const },
  { label: '已确认', value: 'confirmed' as const },
  { label: '已处理', value: 'resolved' as const }
];

const policyFeedbackStatusFilterOptions = [{ label: '全部状态', value: '' }, ...policyFeedbackStatusOptions];

const policyFeedbackSLAStatusOptions = [
  { label: '全部SLA', value: 'all' as const },
  { label: '正常', value: 'normal' as const },
  { label: '已超时', value: 'overdue' as const },
  { label: '已解决', value: 'resolved' as const }
];

const crsTemplatePresetMap: Record<
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

const releaseStatusOptions = [
  { label: '全部', value: '' },
  { label: 'downloaded', value: 'downloaded' },
  { label: 'verified', value: 'verified' },
  { label: 'active', value: 'active' },
  { label: 'failed', value: 'failed' },
  { label: 'rolled_back', value: 'rolled_back' }
];

const jobStatusOptions = [
  { label: '全部', value: '' },
  { label: 'running', value: 'running' },
  { label: 'success', value: 'success' },
  { label: 'failed', value: 'failed' }
];

const jobActionOptions = [
  { label: '全部', value: '' },
  { label: '检查', value: 'check' },
  { label: '下载', value: 'download' },
  { label: '校验', value: 'verify' },
  { label: '激活', value: 'activate' },
  { label: '回滚', value: 'rollback' },
  { label: '引擎检查', value: 'engine_check' }
];

const jobSourceNameMap = ref<Record<number, string>>({});
const userNameMap = ref<Record<string, string>>({});
const userNameLoading = ref(false);
const fetchReleasesRef = ref<() => void | Promise<void>>(() => undefined);
const fetchJobsRef = ref<() => void | Promise<void>>(() => undefined);

const {
  sourceQuery,
  sourceLoading,
  sourceTable,
  sourcePagination,
  sourceModalVisible,
  sourceSubmitting,
  sourceFormRef,
  sourceForm,
  sourceModalTitle,
  sourceRules,
  fetchSources,
  resetSourceQuery,
  handleSourcePageChange,
  handleSourcePageSizeChange,
  handleAddSource,
  handleEditSource,
  handleSubmitSource,
  handleDeleteSource,
  handleSyncSource,
  applyDefaultSource
} = useWafSource({
  message,
  dialog,
  mergeJobSourceNameMap,
  onSyncSuccess: () => {
    Promise.resolve(fetchReleasesRef.value()).catch(() => undefined);
    if (activeTab.value === 'job') {
      Promise.resolve(fetchJobsRef.value()).catch(() => undefined);
    }
  }
});

const {
  engineLoading,
  engineChecking,
  engineUnavailable,
  engineStatus,
  integrationLoading,
  integrationSubmitting,
  integrationPreviewing,
  integrationUnavailable,
  integrationStatus,
  selectedIntegrationSites,
  integrationPreviewActions,
  displayEngineValue,
  fetchIntegrationStatus,
  handleRefreshIntegrationStatus,
  handleIntegrationSiteChange,
  handlePreviewIntegration,
  handleEnableIntegration,
  handleDisableIntegration,
  fetchEngineStatus,
  handleRefreshEngineStatus,
  handleCheckEngine
} = useWafSourceRuntime({
  message,
  onEngineChecked: () => {
    if (activeTab.value === 'job') {
      Promise.resolve(fetchJobsRef.value()).catch(() => undefined);
    }
  }
});

let resolveCurrentRevisionPolicyId: (() => number | undefined) | undefined;

const {
  policyQuery,
  policyLoading,
  policyTable,
  policyPagination,
  policyModalVisible,
  policySubmitting,
  policyFormRef,
  policyForm,
  policyModalTitle,
  policyPreviewLoading,
  policyPreviewPolicyName,
  policyPreviewDirectives,
  policyRevisionLoading,
  policyRevisionTable,
  policyRevisionPagination,
  crsPolicyOptions,
  fetchPolicies,
  resetPolicyQuery,
  handlePolicyPageChange,
  handlePolicyPageSizeChange,
  handleAddPolicy,
  handleEditPolicy,
  handleSubmitPolicy,
  handleDeletePolicy,
  handlePreviewPolicy,
  handleValidatePolicy,
  handlePublishPolicy,
  fetchPolicyRevisions,
  handlePolicyRevisionPageChange,
  handlePolicyRevisionPageSizeChange,
  handleRollbackPolicyRevision,
  getDefaultPolicyId
} = useWafPolicy({
  message,
  dialog,
  ensureUserNamesByIds,
  getCurrentRevisionPolicyId: () => resolveCurrentRevisionPolicyId?.()
});

const {
  crsTuningSubmitting,
  crsTuningFormRef,
  crsTuningForm,
  crsTuningRules,
  hasPolicyWorkspaceDraft,
  getCurrentRevisionPolicyId,
  handleCrsPolicyChange,
  handleRefreshCrsPolicy,
  applyCrsTemplatePreset,
  handleSaveCrsTuning,
  handlePreviewCrsTuning,
  handleValidateCrsTuning,
  handlePublishCrsTuning
} = useWafCrsTuning({
  message,
  dialog,
  activeTab,
  policyTable,
  crsTemplatePresetMap,
  previewPolicy: handlePreviewPolicy,
  validatePolicy: handleValidatePolicy,
  fetchPolicies,
  fetchPolicyRevisions,
  resetPolicyRevisionPage: () => {
    policyRevisionPagination.page = 1;
  }
});

resolveCurrentRevisionPolicyId = getCurrentRevisionPolicyId;

const {
  observeWindowOptions,
  policyStatsQuery,
  policyStatsLoading,
  policyStatsSummary,
  policyStatsTable,
  policyStatsTrend,
  policyStatsTopHosts,
  policyStatsTopPaths,
  policyStatsTopMethods,
  policyStatsRange,
  policyStatsPreviousSnapshot,
  policyFeedbackLoading,
  policyFeedbackTable,
  policyFeedbackCheckedRowKeys,
  policyFeedbackPagination,
  policyFeedbackStatusFilter,
  policyFeedbackAssigneeFilter,
  policyFeedbackSLAStatusFilter,
  policyStatsPolicyOptions,
  hasPolicyStatsDrillFilters,
  hasPolicyFeedbackSelection,
  policyFeedbackCheckedRowKeysInPage,
  fetchPolicyStats,
  resetPolicyStatsQuery,
  clearPolicyStatsDrillFilters,
  clearPolicyStatsDrillLevel,
  fetchPolicyFalsePositiveFeedbacks,
  resetPolicyFeedbackSelection,
  handlePolicyFeedbackPageChange,
  handlePolicyFeedbackPageSizeChange,
  handlePolicyFeedbackStatusFilterChange,
  handlePolicyFeedbackCheckedRowKeysChange,
  buildCurrentPolicyStatsSnapshot
} = useWafObserve({
  crsPolicyOptions,
  ensureUserNamesByIds
});

const {
  policyFeedbackModalVisible,
  policyFeedbackSubmitting,
  policyFeedbackFormRef,
  policyFeedbackForm,
  policyFeedbackProcessModalVisible,
  policyFeedbackProcessSubmitting,
  policyFeedbackProcessFormRef,
  policyFeedbackProcessForm,
  policyFeedbackBatchProcessModalVisible,
  policyFeedbackBatchProcessSubmitting,
  policyFeedbackBatchProcessFormRef,
  policyFeedbackBatchProcessForm,
  openPolicyFeedbackModal,
  openPolicyFeedbackProcessModal,
  openPolicyFeedbackBatchProcessModal,
  handleSubmitPolicyFeedback,
  handleSubmitPolicyFeedbackProcess,
  handleSubmitPolicyFeedbackBatchProcess
} = useWafObserveFeedback({
  message,
  policyStatsQuery,
  policyFeedbackAssigneeFilter,
  policyFeedbackCheckedRowKeys,
  policyFeedbackPagination,
  resetPolicyFeedbackSelection,
  fetchPolicyFalsePositiveFeedbacks
});

const observeExport = useWafObserveExport({
  message,
  route,
  router,
  activeTab,
  observeWindowOptions,
  policyStatsQuery,
  policyStatsRange,
  policyStatsSummary,
  policyStatsTable,
  policyStatsTrend,
  policyStatsTopHosts,
  policyStatsTopPaths,
  policyStatsTopMethods,
  policyStatsPreviousSnapshot,
  buildCurrentPolicyStatsSnapshot,
  formatRatePercent,
  formatDateTime
});
const { handleCopyPolicyStatsLink, handleExportPolicyStatsCsv, handleExportPolicyStatsCompareCsv } = observeExport;
const policyFeedbackRules: FormRules = {
  method: {
    validator(_rule, value: string) {
      const normalized = String(value || '')
        .trim()
        .toUpperCase();
      if (!normalized) {
        return true;
      }
      if (!methodOptions.some(item => item.value === normalized)) {
        return new Error('Method 不合法');
      }
      return true;
    },
    trigger: ['blur', 'change']
  },
  status: {
    validator(_rule, value: number) {
      const num = Number(value);
      if (!Number.isFinite(num) || num < 100 || num > 599) {
        return new Error('状态码必须在 100-599 之间');
      }
      return true;
    },
    trigger: ['blur', 'change']
  },
  dueAt: {
    validator(_rule, value: string) {
      const text = String(value || '').trim();
      if (!text) {
        return true;
      }
      if (!/^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}$/.test(text)) {
        return new Error('截止时间格式应为 YYYY-MM-DD HH:mm:ss');
      }
      return true;
    },
    trigger: ['blur', 'input']
  },
  reason: {
    required: true,
    message: '请填写误报原因',
    trigger: ['blur', 'input']
  }
};

const policyFeedbackProcessRules: FormRules = {
  feedbackStatus: {
    required: true,
    message: '请选择处理状态',
    trigger: 'change'
  },
  dueAt: {
    validator(_rule, value: string) {
      const text = String(value || '').trim();
      if (!text) {
        return true;
      }
      if (!/^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}$/.test(text)) {
        return new Error('截止时间格式应为 YYYY-MM-DD HH:mm:ss');
      }
      return true;
    },
    trigger: ['blur', 'input']
  }
};

const policyRules: FormRules = {
  name: { required: true, message: '请输入策略名称', trigger: 'blur' },
  engineMode: { required: true, message: '请选择引擎模式', trigger: 'change' },
  auditEngine: { required: true, message: '请选择审计模式', trigger: 'change' },
  auditLogFormat: {
    required: true,
    message: '请选择审计日志格式',
    trigger: 'change'
  },
  auditRelevantStatus: {
    validator(_rule, value: string) {
      const raw = String(value || '').trim();
      if (!raw) {
        return new Error('请输入审计状态匹配表达式');
      }
      try {
        // eslint-disable-next-line no-new
        new RegExp(raw);
        return true;
      } catch {
        return new Error('审计状态匹配表达式格式不合法');
      }
    },
    trigger: ['blur', 'input']
  },
  requestBodyLimit: {
    validator(_rule, value: number) {
      const num = Number(value);
      if (!Number.isFinite(num) || num <= 0) {
        return new Error('请求体限制必须大于 0');
      }
      if (num > 1024 * 1024 * 1024) {
        return new Error('请求体限制不能超过 1 GiB');
      }
      return true;
    },
    trigger: ['blur', 'change']
  },
  requestBodyNoFilesLimit: {
    validator(_rule, value: number) {
      const num = Number(value);
      if (!Number.isFinite(num) || num <= 0) {
        return new Error('无文件请求体限制必须大于 0');
      }
      if (num > 1024 * 1024 * 1024) {
        return new Error('无文件请求体限制不能超过 1 GiB');
      }
      return true;
    },
    trigger: ['blur', 'change']
  },
  config: {
    validator(_rule, value: string) {
      const raw = String(value || '').trim();
      if (!raw) return true;
      try {
        JSON.parse(raw);
        return true;
      } catch {
        return new Error('扩展配置必须是合法 JSON');
      }
    },
    trigger: 'blur'
  }
};

const {
  exclusionQuery,
  exclusionLoading,
  exclusionTable,
  exclusionPagination,
  exclusionModalVisible,
  exclusionModalMode,
  exclusionSubmitting,
  exclusionFormRef,
  exclusionRemoveValueInputRef,
  shouldFocusExclusionRemoveValue,
  exclusionForm,
  exclusionModalTitle,
  exclusionRules,
  fetchExclusions,
  resetExclusionQuery,
  handleExclusionPageChange,
  handleExclusionPageSizeChange,
  resetExclusionForm,
  handleAddExclusion,
  handleEditExclusion,
  handleSubmitExclusion,
  handleDeleteExclusion
} = useWafExclusion({
  message,
  getDefaultPolicyId
});

const {
  policyFeedbackExclusionDraftModalVisible,
  policyFeedbackExclusionDraft,
  policyFeedbackExclusionDraftCandidateKey,
  policyFeedbackExclusionCandidateOptions,
  policyFeedbackExclusionDraftDiffItems,
  handleCreateExclusionDraftFromFeedback,
  handlePolicyFeedbackExclusionCandidateChange,
  handlePolicyFeedbackExclusionDraftScopeChange,
  handleConfirmPolicyFeedbackExclusionDraft
} = usePolicyFeedbackExclusionDraft({
  message,
  policyTable,
  resetExclusionForm,
  getDefaultPolicyId,
  mapPolicyNameById,
  mapScopeTypeLabel,
  openExclusionEditor: (payload, focusRemoveValue) => {
    exclusionModalMode.value = 'add';
    exclusionForm.policyId = Number(payload.policyId);
    exclusionForm.name = String(payload.name || '').trim();
    exclusionForm.description = payload.description || '';
    exclusionForm.enabled = Boolean(payload.enabled);
    exclusionForm.scopeType = (payload.scopeType || 'global') as WafPolicyScopeType;
    exclusionForm.host = payload.host || '';
    exclusionForm.path = payload.path || '';
    exclusionForm.method = payload.method || '';
    exclusionForm.removeType = (payload.removeType || 'id') as WafPolicyRemoveType;
    exclusionForm.removeValue = payload.removeValue || '';
    shouldFocusExclusionRemoveValue.value = focusRemoveValue;
    exclusionModalVisible.value = true;
  },
  navigateToPolicyExclusion: () => handleNavigateToPolicySection('exclusion')
});

const {
  bindingQuery,
  bindingLoading,
  bindingTable,
  bindingPagination,
  bindingModalVisible,
  bindingSubmitting,
  bindingFormRef,
  bindingForm,
  bindingModalTitle,
  bindingRules,
  bindingConflictGroups,
  bindingEffectivePreview,
  fetchBindings,
  resetBindingQuery,
  handleBindingPageChange,
  handleBindingPageSizeChange,
  handleAddBinding,
  handleEditBinding,
  handleSubmitBinding,
  handleDeleteBinding
} = useWafBinding({
  message,
  getDefaultPolicyId,
  mapPolicyNameById
});

const {
  releaseQuery,
  releaseLoading,
  releaseTable,
  releasePagination,
  rollbackModalVisible,
  rollbackSubmitting,
  rollbackFormRef,
  rollbackForm,
  jobQuery,
  jobLoading,
  jobTable,
  jobPagination,
  fetchReleases,
  resetReleaseQuery,
  handleReleasePageChange,
  handleReleasePageSizeChange,
  handleActivateRelease,
  handleClearReleases,
  openRollbackModal,
  handleSubmitRollback,
  fetchJobs,
  resetJobQuery,
  handleJobPageChange,
  handleJobPageSizeChange,
  handleClearJobs
} = useWafReleaseJob({
  message,
  dialog,
  ensureSourceNamesByIds,
  ensureUserNamesByIds
});

const triggerOpsRefresh = () => {
  Promise.resolve(fetchReleases()).catch(() => undefined);
  Promise.resolve(fetchJobs()).catch(() => undefined);
};
fetchReleasesRef.value = fetchReleases;
fetchJobsRef.value = fetchJobs;

const uploadModalVisible = ref(false);
const uploadSubmitting = ref(false);
const uploadFormRef = ref<FormInst | null>(null);
const uploadForm = reactive({
  kind: 'crs' as WafKind,
  version: '',
  checksum: '',
  activateNow: false,
  file: null as File | null
});

const uploadRules: FormRules = {
  kind: { required: true, message: '请选择规则类型', trigger: 'change' },
  version: { required: true, message: '请输入版本号', trigger: 'blur' },
  file: {
    validator() {
      if (!uploadForm.file) {
        return new Error('请选择待上传规则包');
      }
      return true;
    },
    trigger: 'change'
  }
};

const rollbackRules: FormRules = {
  target: { required: true, message: '请选择回滚目标', trigger: 'change' },
  version: {
    validator() {
      if (rollbackForm.target === 'version' && !rollbackForm.version.trim()) {
        return new Error('指定版本回滚时必须填写版本号');
      }
      return true;
    },
    trigger: 'blur'
  }
};

const sourceColumns = createSourceColumns({
  handleSyncSource,
  handleEditSource,
  handleDeleteSource
});

const policyColumns = createPolicyColumns({
  mapPolicyEngineModeType,
  mapPolicyEngineModeLabel,
  mapCrsTemplateLabel,
  formatBytes,
  handlePreviewPolicy,
  handleValidatePolicy,
  handlePublishPolicy,
  handleEditPolicy,
  handleDeletePolicy
});

const policyRevisionColumns = createPolicyRevisionColumns({
  mapPolicyRevisionStatusType,
  mapPolicyRevisionStatusLabel,
  displayOperatorName,
  handleRollbackPolicyRevision
});

const exclusionColumns = createExclusionColumns({
  mapScopeTypeLabel,
  handleEditExclusion,
  handleDeleteExclusion
});

const bindingColumns = createBindingColumns({
  mapScopeTypeLabel,
  handleEditBinding,
  handleDeleteBinding
});

const bindingEffectiveColumns = createBindingEffectiveColumns({
  mapScopeTypeLabel
});

const policyStatsTrendColumns = createPolicyStatsTrendColumns();

const policyStatsColumns = createPolicyStatsColumns({
  formatRatePercent
});

const policyStatsDimensionColumns = createPolicyStatsDimensionColumns({
  formatRatePercent
});

function mapPolicyFeedbackStatusLabel(status: string) {
  switch (
    String(status || '')
      .trim()
      .toLowerCase()
  ) {
    case 'confirmed':
      return '已确认';
    case 'resolved':
      return '已处理';
    default:
      return '待确认';
  }
}

function mapPolicyFeedbackStatusTagType(status: string): 'default' | 'warning' | 'success' {
  switch (
    String(status || '')
      .trim()
      .toLowerCase()
  ) {
    case 'confirmed':
      return 'warning';
    case 'resolved':
      return 'success';
    default:
      return 'default';
  }
}

function mapPolicyFeedbackSLAStatusLabel(row: WafPolicyFalsePositiveFeedbackItem) {
  if ((row.feedbackStatus || '') === 'resolved') {
    return '已解决';
  }
  return row.isOverdue ? '已超时' : '正常';
}

function mapPolicyFeedbackSLAStatusTagType(row: WafPolicyFalsePositiveFeedbackItem): 'default' | 'warning' | 'success' {
  if ((row.feedbackStatus || '') === 'resolved') {
    return 'success';
  }
  return row.isOverdue ? 'warning' : 'default';
}

const policyFeedbackColumns = createPolicyFeedbackColumns({
  displayOperatorName,
  mapPolicyFeedbackStatusTagType,
  mapPolicyFeedbackStatusLabel,
  mapPolicyFeedbackSLAStatusTagType,
  mapPolicyFeedbackSLAStatusLabel,
  handleCreateExclusionDraftFromFeedback,
  openPolicyFeedbackProcessModal
});

const {
  policyStatsDrillHint,
  policyStatsDrillStatusLabel,
  isPolicyStatsDrillUnlocked,
  buildPolicyStatsDimensionRowProps
} = useObserveDrilldown({
  message,
  route,
  activeTab,
  observeRouteSyncing: observeExport.observeRouteSyncing,
  policyStatsQuery,
  applyObserveQueryFromRoute: observeExport.applyObserveQueryFromRoute,
  syncObserveStateToRouteQuery: observeExport.syncObserveStateToRouteQuery,
  fetchPolicyStats
});

const releaseColumns = createReleaseColumns({
  mapSourceNameById,
  formatBytes,
  mapReleaseStatusType,
  handleActivateRelease
});

const jobColumns = createJobColumns({
  mapJobSourceName,
  mapJobActionLabel,
  mapJobTriggerModeLabel,
  mapJobStatusType,
  mapJobStatusLabel,
  displayOperatorName,
  mapJobMessage
});

function mapReleaseStatusType(status: WafReleaseStatus) {
  switch (status) {
    case 'active':
      return 'success';
    case 'verified':
      return 'info';
    case 'failed':
      return 'error';
    case 'rolled_back':
      return 'warning';
    default:
      return 'default';
  }
}

function mapPolicyEngineModeType(mode: WafPolicyEngineMode) {
  switch (mode) {
    case 'on':
      return 'error';
    case 'detectiononly':
      return 'warning';
    case 'off':
      return 'default';
    default:
      return 'default';
  }
}

function mapPolicyRevisionStatusType(status: WafPolicyRevisionStatus) {
  switch (status) {
    case 'published':
      return 'success';
    case 'rolled_back':
      return 'warning';
    default:
      return 'default';
  }
}

function mapJobStatusType(status: WafJobStatus) {
  switch (status) {
    case 'success':
      return 'success';
    case 'failed':
      return 'error';
    default:
      return 'warning';
  }
}

function mapJobStatusLabel(status: string) {
  switch (status) {
    case 'running':
      return '执行中';
    case 'success':
      return '成功';
    case 'failed':
      return '失败';
    default:
      return status || '-';
  }
}

function mapJobActionLabel(action: string) {
  switch (action) {
    case 'check':
      return '检查';
    case 'download':
      return '下载';
    case 'verify':
      return '校验';
    case 'activate':
      return '激活';
    case 'rollback':
      return '回滚';
    case 'engine_check':
      return '引擎检查';
    default:
      return action || '-';
  }
}

function mapJobTriggerModeLabel(triggerMode: string) {
  switch (triggerMode) {
    case 'manual':
      return '手动';
    case 'upload':
      return '上传';
    case 'schedule':
      return '定时';
    case 'auto':
      return '自动';
    case 'system':
      return '系统';
    default:
      return triggerMode || '-';
  }
}

function mapSourceNameById(sourceId: number) {
  if (!sourceId || sourceId <= 0) {
    return '-';
  }

  const sourceName = jobSourceNameMap.value[sourceId];
  if (sourceName && sourceName.trim()) {
    return sourceName.trim();
  }

  return '未知更新源';
}

function mapPolicyNameById(policyId: number) {
  if (!policyId || policyId <= 0) {
    return '-';
  }

  const target = policyTable.value.find(item => item.id === policyId);
  if (!target) {
    return `#${policyId}`;
  }

  return target.name ? `${target.name}${target.isDefault ? '（默认）' : ''}` : `#${policyId}`;
}

function mapJobSourceName(row: WafJobItem) {
  if (row.action === 'engine_check') {
    return 'Coraza 引擎';
  }

  return mapSourceNameById(Number(row.sourceId || 0));
}

const defaultPolicyName = computed(() => {
  const target = policyTable.value.find(item => item.isDefault) || policyTable.value[0];
  return target?.name || '-';
});

const selectedPolicyName = computed(() => {
  const target = policyTable.value.find(item => item.id === crsTuningForm.policyId);
  return target?.name || defaultPolicyName.value || '-';
});
const policyWorkspaceActions = computed(() =>
  buildPolicyWorkspaceActions({
    activeSection: activePolicySection.value,
    hasPendingCrsTuningChanges: hasPolicyWorkspaceDraft.value,
    bindingConflictCount: bindingConflictGroups.value.length,
    selectedPolicyName: selectedPolicyName.value
  })
);

function mergeJobSourceNameMap(sourceList: WafSourceItem[]) {
  if (!Array.isArray(sourceList) || sourceList.length === 0) {
    return;
  }

  const nextMap: Record<number, string> = { ...jobSourceNameMap.value };
  sourceList.forEach(item => {
    const sourceId = Number(item?.id || 0);
    const sourceName = String(item?.name || '').trim();
    if (sourceId > 0 && sourceName) {
      nextMap[sourceId] = sourceName;
    }
  });
  jobSourceNameMap.value = nextMap;
}

async function ensureSourceNamesByIds(sourceIds: number[]) {
  const pendingIds = Array.from(
    new Set(sourceIds.filter(sourceId => sourceId > 0 && !jobSourceNameMap.value[sourceId]))
  );
  if (pendingIds.length === 0) {
    return;
  }

  const pageSize = 200;
  let currentPage = 1;
  let total = Number.POSITIVE_INFINITY;

  const loadNextPage = async (): Promise<void> => {
    if (currentPage > 20 || currentPage * pageSize >= total) {
      return;
    }

    const { data, error } = await fetchWafSourceList({
      page: currentPage,
      pageSize,
      name: undefined
    });

    if (error || !data) {
      return;
    }

    const sourceList = data.list || [];
    mergeJobSourceNameMap(sourceList);
    total = data.total || 0;

    if (pendingIds.every(sourceId => Boolean(jobSourceNameMap.value[sourceId]))) {
      return;
    }

    if (sourceList.length === 0 || currentPage * pageSize >= total) {
      return;
    }

    currentPage += 1;
    await loadNextPage();
  };

  await loadNextPage();
}

function mapJobMessage(rawMessage: string) {
  const messageText = String(rawMessage || '').trim();
  if (!messageText) {
    return '-';
  }

  const exactMap: Record<string, string> = {
    'check success': '检查成功',
    'sync success': '同步成功',
    'upload success': '上传成功',
    'activate success': '激活成功',
    'rollback success': '回滚成功',
    'engine source check success': '引擎源检查成功'
  };

  if (exactMap[messageText]) {
    return exactMap[messageText];
  }

  const replacementRules: Array<[RegExp, string]> = [
    [/context deadline exceeded/gi, '请求超时'],
    [/i\/o timeout/gi, '网络超时'],
    [/invalid proxy url:/gi, '代理地址不合法：'],
    [/invalid url:/gi, '无效地址：'],
    [/only https url is allowed/gi, '仅支持 HTTPS 地址'],
    [/only https scheme is allowed/gi, '仅允许 HTTPS 协议'],
    [/proxy url scheme must be http or https/gi, '代理地址协议仅支持 http/https'],
    [/source not found/gi, '未找到更新源'],
    [/source is disabled/gi, '更新源已禁用'],
    [/source mode is not remote/gi, '更新源模式不是 remote'],
    [/source url is empty/gi, '更新源地址为空'],
    [/move package failed:/gi, '移动安装包失败：'],
    [/create release dir failed:/gi, '创建版本目录失败：'],
    [/create release failed:/gi, '创建版本记录失败：'],
    [/fetch failed:/gi, '下载失败：'],
    [/host not allowed:/gi, '源域名不在允许列表：'],
    [/unexpected status code:/gi, '下载返回异常状态码：'],
    [/write temp file failed:/gi, '写入临时文件失败：'],
    [/close temp file failed:/gi, '关闭临时文件失败：'],
    [/move temp file failed:/gi, '移动临时文件失败：'],
    [/prepare waf store failed:/gi, '准备 Waf 存储目录失败：']
  ];

  let localizedMessage = messageText;
  for (const [pattern, replacement] of replacementRules) {
    localizedMessage = localizedMessage.replace(pattern, replacement);
  }

  return localizedMessage;
}

function formatRatePercent(value: number) {
  const numeric = Number(value || 0);
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return '0%';
  }
  return `${(numeric * 100).toFixed(2)}%`;
}

function formatDateTime(date: Date) {
  const pad = (num: number) => String(num).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

function isNumericUserId(value: unknown) {
  return /^\d+$/.test(String(value ?? '').trim());
}

function displayOperatorName(value: unknown) {
  const raw = String(value ?? '').trim();
  if (!raw) {
    return '-';
  }
  if (!isNumericUserId(raw)) {
    return raw;
  }
  return userNameMap.value[raw] || '-';
}

async function ensureUserNamesByIds(values: unknown[]) {
  const pendingIds = Array.from(
    new Set(
      values
        .map(value => String(value ?? '').trim())
        .filter(value => value && isNumericUserId(value) && !userNameMap.value[value])
    )
  );

  if (!pendingIds.length || userNameLoading.value) {
    return;
  }

  userNameLoading.value = true;
  try {
    const unresolved = new Set(pendingIds);
    const pageSize = 100;
    const loadUserPage = async (page: number, total = Number.POSITIVE_INFINITY): Promise<void> => {
      if (page > 50 || unresolved.size === 0 || page * pageSize >= total) {
        return;
      }

      const { data, error } = await request<any>({
        url: '/api/user/list',
        params: { page, pageSize }
      });

      if (error || !data) {
        return;
      }

      const list = Array.isArray(data.list) ? data.list : [];
      list.forEach((item: any) => {
        const id = String(item?.id ?? '').trim();
        const username = String(item?.username ?? '').trim();
        if (!id || !username) {
          return;
        }
        userNameMap.value[id] = username;
        unresolved.delete(id);
      });

      const nextTotal = Number(data.total || 0);
      if (list.length === 0 || page * pageSize >= nextTotal) {
        return;
      }

      await loadUserPage(page + 1, nextTotal);
    };

    await loadUserPage(1);
  } finally {
    userNameLoading.value = false;
  }
}

function setObserveActiveView(view: 'analysis' | 'feedback') {
  observeActiveView.value = view;
}

function handleNavigateToMenu(menu: SecurityMenuKey) {
  return navigateToSecurityTab(SECURITY_MENU_SCHEMA[menu].defaultTab);
}

function handleNavigateToPolicySection(tab: 'runtime' | 'crs' | 'exclusion' | 'binding') {
  return navigateToSecurityTab(tab);
}

function handleNavigateToOpsSection(tab: 'release' | 'job') {
  return navigateToSecurityTab(tab);
}

function openUploadModal() {
  uploadForm.kind = 'crs';
  uploadForm.version = '';
  uploadForm.checksum = '';
  uploadForm.activateNow = false;
  uploadForm.file = null;
  uploadModalVisible.value = true;
}

watch(
  () => sourceForm.mode,
  value => {
    if (value !== 'remote') {
      sourceForm.proxyUrl = '';
    }
  }
);

function handleBeforeUpload(data: { file: UploadFileInfo }) {
  const raw = data.file.file;
  if (!raw) return false;

  const name = raw.name.toLowerCase();
  if (!(name.endsWith('.zip') || name.endsWith('.tar.gz'))) {
    message.error('仅支持 .zip 或 .tar.gz 文件');
    return false;
  }

  uploadForm.file = raw;
  return false;
}

function handleRemoveUpload() {
  uploadForm.file = null;
  return true;
}

async function handleSubmitUpload() {
  await uploadFormRef.value?.validate();
  if (!uploadForm.file) {
    message.error('请先选择上传文件');
    return;
  }

  uploadSubmitting.value = true;
  try {
    const formData = new FormData();
    formData.append('kind', uploadForm.kind);
    formData.append('version', uploadForm.version.trim());
    if (uploadForm.checksum.trim()) {
      formData.append('checksum', uploadForm.checksum.trim());
    }
    formData.append('activateNow', String(uploadForm.activateNow));
    formData.append('file', uploadForm.file);

    const { error } = await uploadWafPackage(formData);
    if (!error) {
      message.success('上传成功，规则包已入库');
      uploadModalVisible.value = false;
      triggerOpsRefresh();
    }
  } finally {
    uploadSubmitting.value = false;
  }
}

const securityTabRefreshMap: Record<SecurityTabKey, () => void> = {
  source: () => {
    fetchIntegrationStatus();
    fetchEngineStatus();
    fetchSources();
  },
  runtime: () => {
    fetchPolicies();
    fetchPolicyRevisions();
  },
  crs: () => {
    fetchPolicies();
    fetchPolicyRevisions(getCurrentRevisionPolicyId());
  },
  exclusion: () => {
    fetchPolicies();
    fetchExclusions();
  },
  binding: () => {
    fetchPolicies();
    fetchBindings();
  },
  observe: () => {
    fetchPolicies();
    fetchPolicyStats();
  },
  release: () => {
    Promise.resolve(fetchReleases()).catch(() => undefined);
  },
  job: () => {
    Promise.resolve(fetchJobs()).catch(() => undefined);
  }
};

const securityDomainRefreshMap: Record<SecurityMenuKey, () => void> = {
  source: () => {
    fetchIntegrationStatus();
    fetchEngineStatus();
    fetchSources();
  },
  policy: () => {
    fetchPolicies();
    fetchPolicyRevisions(getCurrentRevisionPolicyId());
    fetchExclusions();
    fetchBindings();
  },
  observe: () => {
    fetchPolicies();
    fetchPolicyStats();
  },
  ops: () => {
    triggerOpsRefresh();
  }
};

const { refreshCurrentDomain } = useSecurityRefresh({
  activeMenu,
  activeTab,
  refreshByTab: securityTabRefreshMap,
  refreshByMenu: securityDomainRefreshMap
});
</script>

<template>
  <div class="h-full flex flex-col gap-3">
    <NCard :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">安全管理</div>
          <div class="mt-1 text-xs text-gray-500">
            {{ activeMenuDescription }}
          </div>
        </div>
        <div class="flex items-center gap-2 text-xs text-gray-500">
          <span>{{ pageTitle }}</span>
          <NButton secondary size="small" @click="refreshCurrentDomain">刷新当前领域</NButton>
        </div>
      </div>

      <NGrid cols="4" x-gap="12" y-gap="12" class="mt-4">
        <NGi v-for="item in securityMenus" :key="item.key">
          <NCard
            size="small"
            :bordered="activeMenu !== item.key"
            class="cursor-pointer transition-all duration-200"
            :class="activeMenu === item.key ? 'shadow-sm ring-1 ring-primary/30' : 'opacity-88 hover:opacity-100'"
            @click="handleNavigateToMenu(item.key)"
          >
            <div class="text-sm font-semibold">{{ item.title }}</div>
            <div class="mt-1 text-xs text-gray-500">
              {{ securityMenuDescriptionMap[item.key] }}
            </div>
          </NCard>
        </NGi>
      </NGrid>
    </NCard>

    <SecuritySourcePage
      v-if="activeMenu === 'source'"
      :page-title="pageTitle"
      :integration-loading="integrationLoading"
      :integration-submitting="integrationSubmitting"
      :integration-previewing="integrationPreviewing"
      :integration-unavailable="integrationUnavailable"
      :integration-status="integrationStatus"
      :selected-integration-sites="selectedIntegrationSites"
      :handle-refresh-integration-status="handleRefreshIntegrationStatus"
      :handle-preview-integration="handlePreviewIntegration"
      :handle-enable-integration="handleEnableIntegration"
      :handle-disable-integration="handleDisableIntegration"
      :handle-integration-site-change="handleIntegrationSiteChange"
      :integration-preview-actions="integrationPreviewActions"
      :engine-loading="engineLoading"
      :engine-checking="engineChecking"
      :engine-unavailable="engineUnavailable"
      :engine-status="engineStatus"
      :handle-refresh-engine-status="handleRefreshEngineStatus"
      :handle-check-engine="handleCheckEngine"
      :display-engine-value="displayEngineValue"
      :source-query="sourceQuery"
      :source-columns="sourceColumns"
      :source-table="sourceTable"
      :source-loading="sourceLoading"
      :source-pagination="sourcePagination"
      :table-fixed-height="tableFixedHeight"
      :fetch-sources="fetchSources"
      :reset-source-query="resetSourceQuery"
      :handle-add-source="handleAddSource"
      :open-upload-modal="openUploadModal"
      :handle-source-page-change="handleSourcePageChange"
      :handle-source-page-size-change="handleSourcePageSizeChange"
    />

    <SecurityPolicyPage
      v-else-if="activeMenu === 'policy'"
      :active-section="activePolicySection"
      :navigate-to-tab="handleNavigateToPolicySection"
      :policy-query="policyQuery"
      :policy-columns="policyColumns"
      :policy-table="policyTable"
      :policy-loading="policyLoading"
      :policy-pagination="policyPagination"
      :table-fixed-height="tableFixedHeight"
      :fetch-policies="fetchPolicies"
      :reset-policy-query="resetPolicyQuery"
      :handle-add-policy="handleAddPolicy"
      :handle-policy-page-change="handlePolicyPageChange"
      :handle-policy-page-size-change="handlePolicyPageSizeChange"
      :policy-preview-policy-name="policyPreviewPolicyName"
      :policy-preview-loading="policyPreviewLoading"
      :policy-preview-directives="policyPreviewDirectives"
      :policy-revision-columns="policyRevisionColumns"
      :policy-revision-table="policyRevisionTable"
      :policy-revision-loading="policyRevisionLoading"
      :policy-revision-pagination="policyRevisionPagination"
      :handle-policy-revision-page-change="handlePolicyRevisionPageChange"
      :handle-policy-revision-page-size-change="handlePolicyRevisionPageSizeChange"
      :default-policy-name="defaultPolicyName"
      :selected-policy-name="selectedPolicyName"
      :active-section-label="securityPolicySectionLabelMap[activePolicySection]"
      :has-pending-crs-tuning-changes="hasPolicyWorkspaceDraft"
      :policy-workspace-actions="policyWorkspaceActions"
      :exclusion-total="Number(exclusionPagination.itemCount || 0)"
      :crs-tuning-submitting="crsTuningSubmitting"
      :crs-tuning-form-ref="crsTuningFormRef"
      :crs-tuning-form="crsTuningForm"
      :crs-policy-options="crsPolicyOptions"
      :crs-tuning-rules="crsTuningRules"
      :handle-crs-policy-change="handleCrsPolicyChange"
      :map-crs-template-label="mapCrsTemplateLabel"
      :handle-refresh-crs-policy="handleRefreshCrsPolicy"
      :apply-crs-template-preset="applyCrsTemplatePreset"
      :handle-save-crs-tuning="handleSaveCrsTuning"
      :handle-preview-crs-tuning="handlePreviewCrsTuning"
      :handle-validate-crs-tuning="handleValidateCrsTuning"
      :handle-publish-crs-tuning="handlePublishCrsTuning"
      :exclusion-query="exclusionQuery"
      :scope-type-options="scopeTypeOptions"
      :fetch-exclusions="fetchExclusions"
      :reset-exclusion-query="resetExclusionQuery"
      :handle-add-exclusion="handleAddExclusion"
      :exclusion-columns="exclusionColumns"
      :exclusion-table="exclusionTable"
      :exclusion-loading="exclusionLoading"
      :exclusion-pagination="exclusionPagination"
      :handle-exclusion-page-change="handleExclusionPageChange"
      :handle-exclusion-page-size-change="handleExclusionPageSizeChange"
      :binding-query="bindingQuery"
      :fetch-bindings="fetchBindings"
      :reset-binding-query="resetBindingQuery"
      :handle-add-binding="handleAddBinding"
      :binding-columns="bindingColumns"
      :binding-table="bindingTable"
      :binding-loading="bindingLoading"
      :binding-pagination="bindingPagination"
      :handle-binding-page-change="handleBindingPageChange"
      :handle-binding-page-size-change="handleBindingPageSizeChange"
      :binding-conflict-groups="bindingConflictGroups"
      :binding-effective-columns="bindingEffectiveColumns"
      :binding-effective-preview="bindingEffectivePreview"
    />

    <SecurityObservePage
      v-else-if="activeMenu === 'observe'"
      :active-view="observeActiveView"
      :set-active-view="setObserveActiveView"
      :policy-stats-query="policyStatsQuery"
      :policy-stats-policy-options="policyStatsPolicyOptions"
      :observe-window-options="observeWindowOptions"
      :policy-stats-loading="policyStatsLoading"
      :fetch-policy-stats="fetchPolicyStats"
      :reset-policy-stats-query="resetPolicyStatsQuery"
      :has-policy-stats-drill-filters="hasPolicyStatsDrillFilters"
      :clear-policy-stats-drill-filters="clearPolicyStatsDrillFilters"
      :handle-copy-policy-stats-link="handleCopyPolicyStatsLink"
      :handle-export-policy-stats-compare-csv="handleExportPolicyStatsCompareCsv"
      :handle-export-policy-stats-csv="handleExportPolicyStatsCsv"
      :policy-stats-summary="policyStatsSummary"
      :policy-stats-range="policyStatsRange"
      :policy-stats-previous-snapshot="policyStatsPreviousSnapshot"
      :format-rate-percent="formatRatePercent"
      :clear-policy-stats-drill-level="clearPolicyStatsDrillLevel"
      :policy-stats-trend-columns="policyStatsTrendColumns"
      :policy-stats-trend="policyStatsTrend"
      :policy-stats-columns="policyStatsColumns"
      :policy-stats-table="policyStatsTable"
      :policy-stats-drill-hint="policyStatsDrillHint"
      :policy-stats-drill-status-label="policyStatsDrillStatusLabel"
      :is-policy-stats-drill-unlocked="isPolicyStatsDrillUnlocked"
      :policy-stats-dimension-columns="policyStatsDimensionColumns"
      :policy-stats-top-hosts="policyStatsTopHosts"
      :policy-stats-top-paths="policyStatsTopPaths"
      :policy-stats-top-methods="policyStatsTopMethods"
      :build-policy-stats-dimension-row-props="buildPolicyStatsDimensionRowProps"
      :policy-feedback-status-filter="policyFeedbackStatusFilter"
      :policy-feedback-status-filter-options="policyFeedbackStatusFilterOptions"
      :set-policy-feedback-status-filter="
        (value: '' | 'pending' | 'confirmed' | 'resolved') => {
          policyFeedbackStatusFilter = value;
        }
      "
      :policy-feedback-assignee-filter="policyFeedbackAssigneeFilter"
      :set-policy-feedback-assignee-filter="
        (value: string) => {
          policyFeedbackAssigneeFilter = value;
        }
      "
      :policy-feedback-sla-status-filter="policyFeedbackSLAStatusFilter"
      :policy-feedback-sla-status-options="policyFeedbackSLAStatusOptions"
      :set-policy-feedback-sla-status-filter="
        (value: 'all' | 'normal' | 'overdue' | 'resolved') => {
          policyFeedbackSLAStatusFilter = value;
        }
      "
      :handle-policy-feedback-status-filter-change="handlePolicyFeedbackStatusFilterChange"
      :open-policy-feedback-modal="openPolicyFeedbackModal"
      :open-policy-feedback-batch-process-modal="openPolicyFeedbackBatchProcessModal"
      :has-policy-feedback-selection="hasPolicyFeedbackSelection"
      :policy-feedback-checked-row-keys="policyFeedbackCheckedRowKeys"
      :policy-feedback-loading="policyFeedbackLoading"
      :fetch-policy-false-positive-feedbacks="fetchPolicyFalsePositiveFeedbacks"
      :policy-feedback-columns="policyFeedbackColumns"
      :policy-feedback-table="policyFeedbackTable"
      :policy-feedback-pagination="policyFeedbackPagination"
      :policy-feedback-checked-row-keys-in-page="policyFeedbackCheckedRowKeysInPage"
      :handle-policy-feedback-checked-row-keys-change="handlePolicyFeedbackCheckedRowKeysChange"
      :handle-policy-feedback-page-change="handlePolicyFeedbackPageChange"
      :handle-policy-feedback-page-size-change="handlePolicyFeedbackPageSizeChange"
    />

    <SecurityOpsPage
      v-else
      :active-section="activeOpsSection"
      :navigate-to-tab="handleNavigateToOpsSection"
      :release-query="releaseQuery"
      :release-status-options="releaseStatusOptions"
      :fetch-releases="fetchReleases"
      :reset-release-query="resetReleaseQuery"
      :open-rollback-modal="openRollbackModal"
      :handle-clear-releases="handleClearReleases"
      :release-columns="releaseColumns"
      :release-table="releaseTable"
      :release-loading="releaseLoading"
      :release-pagination="releasePagination"
      :table-fixed-height="tableFixedHeight"
      :handle-release-page-change="handleReleasePageChange"
      :handle-release-page-size-change="handleReleasePageSizeChange"
      :job-query="jobQuery"
      :job-status-options="jobStatusOptions"
      :job-action-options="jobActionOptions"
      :fetch-jobs="fetchJobs"
      :reset-job-query="resetJobQuery"
      :refresh-current-section="refreshCurrentDomain"
      :handle-clear-jobs="handleClearJobs"
      :job-columns="jobColumns"
      :job-table="jobTable"
      :job-loading="jobLoading"
      :job-pagination="jobPagination"
      :handle-job-page-change="handleJobPageChange"
      :handle-job-page-size-change="handleJobPageSizeChange"
    />

    <NModal v-model:show="sourceModalVisible" preset="card" :title="sourceModalTitle" class="w-720px">
      <NForm ref="sourceFormRef" :model="sourceForm" :rules="sourceRules" label-placement="left" label-width="120">
        <NGrid cols="2" x-gap="12">
          <NFormItemGi label="名称" path="name">
            <NInput v-model:value="sourceForm.name" placeholder="例如：official-crs" />
          </NFormItemGi>
          <NFormItemGi label="类型" path="kind">
            <NInput value="crs" disabled />
          </NFormItemGi>
          <NFormItemGi label="模式" path="mode">
            <NSelect v-model:value="sourceForm.mode" :options="modeOptions" />
          </NFormItemGi>
          <NFormItemGi label="鉴权类型" path="authType">
            <NSelect v-model:value="sourceForm.authType" :options="authTypeOptions" />
          </NFormItemGi>
        </NGrid>

        <NFormItem label="默认源">
          <div class="flex flex-wrap gap-2">
            <NButton size="small" secondary @click="applyDefaultSource">应用 CRS 默认源</NButton>
          </div>
        </NFormItem>

        <NFormItem v-if="sourceForm.mode === 'remote'" label="源地址" path="url">
          <NInput
            v-model:value="sourceForm.url"
            placeholder="https://api.github.com/repos/coreruleset/coreruleset/releases/latest"
          />
        </NFormItem>

        <NFormItem v-if="sourceForm.mode === 'remote'" label="校验地址" path="checksumUrl">
          <NInput v-model:value="sourceForm.checksumUrl" placeholder="可选，SHA256 清单地址" />
        </NFormItem>

        <NFormItem v-if="sourceForm.mode === 'remote'" label="代理地址" path="proxyUrl">
          <NInput v-model:value="sourceForm.proxyUrl" placeholder="可选，例如：http://127.0.0.1:7890" />
        </NFormItem>

        <NFormItem v-if="sourceForm.authType !== 'none'" label="鉴权密钥" path="authSecret">
          <NInput
            v-model:value="sourceForm.authSecret"
            type="password"
            show-password-on="mousedown"
            placeholder="Token 或 user:password"
          />
        </NFormItem>

        <NFormItem label="调度表达式" path="schedule">
          <NInput v-model:value="sourceForm.schedule" placeholder="例如：0 0 */6 * * *" />
        </NFormItem>

        <NFormItem label="附加元数据" path="meta">
          <NInput
            v-model:value="sourceForm.meta"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 5 }"
            placeholder="JSON 字符串，可选"
          />
        </NFormItem>

        <NGrid cols="2" x-gap="12">
          <NFormItemGi label="启用">
            <NSwitch v-model:value="sourceForm.enabled" />
          </NFormItemGi>
          <NFormItemGi label="自动检查">
            <NSwitch v-model:value="sourceForm.autoCheck" />
          </NFormItemGi>
          <NFormItemGi label="自动下载">
            <NSwitch v-model:value="sourceForm.autoDownload" />
          </NFormItemGi>
          <NFormItemGi label="自动激活">
            <NSwitch v-model:value="sourceForm.autoActivate" />
          </NFormItemGi>
        </NGrid>
      </NForm>

      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="sourceModalVisible = false">取消</NButton>
          <NButton type="primary" :loading="sourceSubmitting" @click="handleSubmitSource">保存</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="policyModalVisible" preset="card" :title="policyModalTitle" class="w-760px">
      <NForm ref="policyFormRef" :model="policyForm" :rules="policyRules" label-placement="left" label-width="150">
        <NGrid cols="2" x-gap="12">
          <NFormItemGi label="策略名称" path="name">
            <NInput v-model:value="policyForm.name" placeholder="例如：default-runtime-policy" />
          </NFormItemGi>
          <NFormItemGi label="是否默认策略">
            <NSwitch v-model:value="policyForm.isDefault" />
          </NFormItemGi>
          <NFormItemGi label="引擎模式" path="engineMode">
            <NSelect v-model:value="policyForm.engineMode" :options="policyEngineModeOptions" />
          </NFormItemGi>
          <NFormItemGi label="审计模式" path="auditEngine">
            <NSelect v-model:value="policyForm.auditEngine" :options="policyAuditEngineOptions" />
          </NFormItemGi>
          <NFormItemGi label="审计日志格式" path="auditLogFormat">
            <NSelect v-model:value="policyForm.auditLogFormat" :options="policyAuditLogFormatOptions" />
          </NFormItemGi>
          <NFormItemGi label="请求体访问">
            <NSwitch v-model:value="policyForm.requestBodyAccess" />
          </NFormItemGi>
          <NFormItemGi label="启用策略">
            <NSwitch v-model:value="policyForm.enabled" />
          </NFormItemGi>
        </NGrid>

        <NFormItem label="描述" path="description">
          <NInput v-model:value="policyForm.description" placeholder="可选，记录策略用途与变更说明" />
        </NFormItem>

        <NFormItem label="审计状态匹配" path="auditRelevantStatus">
          <NInput v-model:value="policyForm.auditRelevantStatus" placeholder="例如：^(?:5|4(?!04))" />
        </NFormItem>

        <NGrid cols="2" x-gap="12">
          <NFormItemGi label="请求体限制（字节）" path="requestBodyLimit">
            <NInputNumber
              v-model:value="policyForm.requestBodyLimit"
              :show-button="false"
              :min="1"
              :max="1024 * 1024 * 1024"
              class="w-full"
            />
          </NFormItemGi>
          <NFormItemGi label="无文件请求体限制（字节）" path="requestBodyNoFilesLimit">
            <NInputNumber
              v-model:value="policyForm.requestBodyNoFilesLimit"
              :show-button="false"
              :min="1"
              :max="1024 * 1024 * 1024"
              class="w-full"
            />
          </NFormItemGi>
        </NGrid>

        <NFormItem label="扩展配置(JSON)" path="config">
          <NInput
            v-model:value="policyForm.config"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 6 }"
            placeholder='可选，例如：{"custom_tag":"runtime"}'
          />
        </NFormItem>
      </NForm>

      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="policyModalVisible = false">取消</NButton>
          <NButton type="primary" :loading="policySubmitting" @click="handleSubmitPolicy">保存</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="uploadModalVisible" preset="card" title="上传规则包" class="w-640px">
      <NForm ref="uploadFormRef" :model="uploadForm" :rules="uploadRules" label-placement="left" label-width="110">
        <NFormItem label="类型" path="kind">
          <NInput value="crs" disabled />
        </NFormItem>
        <NFormItem label="版本号" path="version">
          <NInput v-model:value="uploadForm.version" placeholder="例如：v4.23.0-custom.1" />
        </NFormItem>
        <NFormItem label="SHA256" path="checksum">
          <NInput v-model:value="uploadForm.checksum" placeholder="可选，建议填写" />
        </NFormItem>
        <NFormItem label="立即激活" path="activateNow">
          <NSwitch v-model:value="uploadForm.activateNow" />
        </NFormItem>
        <NFormItem label="规则包" path="file">
          <NUpload
            :default-upload="false"
            :max="1"
            :show-file-list="true"
            accept=".zip,.tar.gz"
            @before-upload="handleBeforeUpload"
            @remove="handleRemoveUpload"
          >
            <NButton>选择文件</NButton>
          </NUpload>
        </NFormItem>
      </NForm>

      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="uploadModalVisible = false">取消</NButton>
          <NButton type="primary" :loading="uploadSubmitting" @click="handleSubmitUpload">上传并入库</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="exclusionModalVisible" preset="card" :title="exclusionModalTitle" class="w-760px">
      <NForm
        ref="exclusionFormRef"
        :model="exclusionForm"
        :rules="exclusionRules"
        label-placement="left"
        label-width="140"
      >
        <NGrid cols="2" x-gap="12">
          <NFormItemGi label="规则名称" path="name">
            <NInput v-model:value="exclusionForm.name" placeholder="例如：ignore-login-fp" />
          </NFormItemGi>
          <NFormItemGi label="关联策略" path="policyId">
            <NSelect v-model:value="exclusionForm.policyId" :options="crsPolicyOptions" />
          </NFormItemGi>
          <NFormItemGi label="作用域" path="scopeType">
            <NSelect v-model:value="exclusionForm.scopeType" :options="scopeTypeOptions" />
          </NFormItemGi>
          <NFormItemGi label="移除类型" path="removeType">
            <NSelect v-model:value="exclusionForm.removeType" :options="removeTypeOptions" />
          </NFormItemGi>
          <NFormItemGi v-if="exclusionForm.scopeType !== 'global'" label="Host" path="host">
            <NInput v-model:value="exclusionForm.host" placeholder="例如：app.example.com" />
          </NFormItemGi>
          <NFormItemGi v-if="exclusionForm.scopeType === 'route'" label="Path" path="path">
            <NInput v-model:value="exclusionForm.path" placeholder="例如：/api/login" />
          </NFormItemGi>
          <NFormItemGi v-if="exclusionForm.scopeType === 'route'" label="Method" path="method">
            <NSelect v-model:value="exclusionForm.method" :options="methodOptions" clearable placeholder="可选" />
          </NFormItemGi>
          <NFormItemGi label="是否启用">
            <NSwitch v-model:value="exclusionForm.enabled" />
          </NFormItemGi>
        </NGrid>

        <NFormItem label="移除值" path="removeValue">
          <NInput
            ref="exclusionRemoveValueInputRef"
            v-model:value="exclusionForm.removeValue"
            :placeholder="exclusionForm.removeType === 'id' ? '例如：920350' : '例如：attack-sqli'"
          />
        </NFormItem>
        <NFormItem label="描述" path="description">
          <NInput v-model:value="exclusionForm.description" placeholder="可选，记录误报场景与原因" />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="exclusionModalVisible = false">取消</NButton>
          <NButton type="primary" :loading="exclusionSubmitting" @click="handleSubmitExclusion">保存</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="bindingModalVisible" preset="card" :title="bindingModalTitle" class="w-760px">
      <NForm ref="bindingFormRef" :model="bindingForm" :rules="bindingRules" label-placement="left" label-width="140">
        <NGrid cols="2" x-gap="12">
          <NFormItemGi label="绑定名称" path="name">
            <NInput v-model:value="bindingForm.name" placeholder="例如：site-main-binding" />
          </NFormItemGi>
          <NFormItemGi label="关联策略" path="policyId">
            <NSelect v-model:value="bindingForm.policyId" :options="crsPolicyOptions" />
          </NFormItemGi>
          <NFormItemGi label="作用域" path="scopeType">
            <NSelect v-model:value="bindingForm.scopeType" :options="scopeTypeOptions" />
          </NFormItemGi>
          <NFormItemGi label="优先级" path="priority">
            <NInputNumber
              v-model:value="bindingForm.priority"
              :show-button="false"
              :min="1"
              :max="1000"
              class="w-full"
            />
          </NFormItemGi>
          <NFormItemGi v-if="bindingForm.scopeType !== 'global'" label="Host" path="host">
            <NInput v-model:value="bindingForm.host" placeholder="例如：app.example.com" />
          </NFormItemGi>
          <NFormItemGi v-if="bindingForm.scopeType === 'route'" label="Path" path="path">
            <NInput v-model:value="bindingForm.path" placeholder="例如：/api" />
          </NFormItemGi>
          <NFormItemGi v-if="bindingForm.scopeType === 'route'" label="Method" path="method">
            <NSelect v-model:value="bindingForm.method" :options="methodOptions" clearable placeholder="可选" />
          </NFormItemGi>
          <NFormItemGi label="是否启用">
            <NSwitch v-model:value="bindingForm.enabled" />
          </NFormItemGi>
        </NGrid>

        <NFormItem label="描述" path="description">
          <NInput v-model:value="bindingForm.description" placeholder="可选，记录生效范围和意图" />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="bindingModalVisible = false">取消</NButton>
          <NButton type="primary" :loading="bindingSubmitting" @click="handleSubmitBinding">保存</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="policyFeedbackModalVisible" preset="card" title="标记误报反馈" class="w-760px">
      <NForm
        ref="policyFeedbackFormRef"
        :model="policyFeedbackForm"
        :rules="policyFeedbackRules"
        label-placement="left"
        label-width="130"
      >
        <NGrid cols="2" x-gap="12">
          <NFormItemGi label="关联策略" path="policyId">
            <NSelect
              v-model:value="policyFeedbackForm.policyId"
              :options="crsPolicyOptions"
              clearable
              placeholder="可选，不填表示全部策略"
            />
          </NFormItemGi>
          <NFormItemGi label="状态码" path="status">
            <NInputNumber
              v-model:value="policyFeedbackForm.status"
              :show-button="false"
              :min="100"
              :max="599"
              class="w-full"
            />
          </NFormItemGi>
          <NFormItemGi label="责任人" path="assignee">
            <NInput v-model:value="policyFeedbackForm.assignee" placeholder="可选，例如 alice" />
          </NFormItemGi>
          <NFormItemGi label="截止时间" path="dueAt">
            <NInput v-model:value="policyFeedbackForm.dueAt" placeholder="可选，YYYY-MM-DD HH:mm:ss" />
          </NFormItemGi>
          <NFormItemGi label="Host" path="host">
            <NInput v-model:value="policyFeedbackForm.host" placeholder="可选，例如 app.example.com" />
          </NFormItemGi>
          <NFormItemGi label="Path" path="path">
            <NInput v-model:value="policyFeedbackForm.path" placeholder="可选，例如 /api/login" />
          </NFormItemGi>
          <NFormItemGi label="Method" path="method">
            <NSelect v-model:value="policyFeedbackForm.method" :options="methodOptions" clearable placeholder="可选" />
          </NFormItemGi>
          <NFormItemGi label="示例 URI" path="sampleUri">
            <NInput v-model:value="policyFeedbackForm.sampleUri" placeholder="可选，记录原始 URI 便于复盘" />
          </NFormItemGi>
        </NGrid>
        <NFormItem label="误报原因" path="reason">
          <NInput
            v-model:value="policyFeedbackForm.reason"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            placeholder="必填：为何判断为误报"
          />
        </NFormItem>
        <NFormItem label="建议动作" path="suggestion">
          <NInput
            v-model:value="policyFeedbackForm.suggestion"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            placeholder="可选：例如建议添加 removeById、放宽阈值或补白名单"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="policyFeedbackModalVisible = false">取消</NButton>
          <NButton type="warning" :loading="policyFeedbackSubmitting" @click="handleSubmitPolicyFeedback">
            提交反馈
          </NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="policyFeedbackProcessModalVisible" preset="card" title="处理误报反馈" class="w-640px">
      <NForm
        ref="policyFeedbackProcessFormRef"
        :model="policyFeedbackProcessForm"
        :rules="policyFeedbackProcessRules"
        label-placement="left"
        label-width="120"
      >
        <NFormItem label="处理状态" path="feedbackStatus">
          <NSelect v-model:value="policyFeedbackProcessForm.feedbackStatus" :options="policyFeedbackStatusOptions" />
        </NFormItem>
        <NFormItem label="责任人" path="assignee">
          <NInput v-model:value="policyFeedbackProcessForm.assignee" placeholder="可选，例如 alice" />
        </NFormItem>
        <NFormItem label="截止时间" path="dueAt">
          <NInput v-model:value="policyFeedbackProcessForm.dueAt" placeholder="可选，YYYY-MM-DD HH:mm:ss" />
        </NFormItem>
        <NFormItem label="处理备注" path="processNote">
          <NInput
            v-model:value="policyFeedbackProcessForm.processNote"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            placeholder="可选，记录确认依据或处理结果"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="policyFeedbackProcessModalVisible = false">取消</NButton>
          <NButton type="warning" :loading="policyFeedbackProcessSubmitting" @click="handleSubmitPolicyFeedbackProcess">
            保存状态
          </NButton>
        </div>
      </template>
    </NModal>

    <NModal
      v-model:show="policyFeedbackBatchProcessModalVisible"
      preset="card"
      title="批量处理误报反馈"
      class="w-640px"
    >
      <div class="mb-3 text-sm text-gray-600">已选择 {{ policyFeedbackCheckedRowKeys.length }} 条反馈记录</div>
      <NForm
        ref="policyFeedbackBatchProcessFormRef"
        :model="policyFeedbackBatchProcessForm"
        :rules="policyFeedbackProcessRules"
        label-placement="left"
        label-width="120"
      >
        <NFormItem label="处理状态" path="feedbackStatus">
          <NSelect
            v-model:value="policyFeedbackBatchProcessForm.feedbackStatus"
            :options="policyFeedbackStatusOptions"
          />
        </NFormItem>
        <NFormItem label="责任人" path="assignee">
          <NInput v-model:value="policyFeedbackBatchProcessForm.assignee" placeholder="可选，例如 alice" />
        </NFormItem>
        <NFormItem label="截止时间" path="dueAt">
          <NInput v-model:value="policyFeedbackBatchProcessForm.dueAt" placeholder="可选，YYYY-MM-DD HH:mm:ss" />
        </NFormItem>
        <NFormItem label="处理备注" path="processNote">
          <NInput
            v-model:value="policyFeedbackBatchProcessForm.processNote"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            placeholder="可选，批量处理说明"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="policyFeedbackBatchProcessModalVisible = false">取消</NButton>
          <NButton
            type="warning"
            :loading="policyFeedbackBatchProcessSubmitting"
            @click="handleSubmitPolicyFeedbackBatchProcess"
          >
            批量保存
          </NButton>
        </div>
      </template>
    </NModal>

    <NModal
      v-model:show="policyFeedbackExclusionDraftModalVisible"
      preset="card"
      title="确认生成例外草稿"
      class="w-760px"
    >
      <div v-if="policyFeedbackExclusionDraft" class="space-y-3">
        <div class="text-sm text-gray-600">来源反馈 #{{ policyFeedbackExclusionDraft.feedbackId }}</div>
        <NForm :model="policyFeedbackExclusionDraft" label-placement="left" label-width="120">
          <NGrid cols="2" x-gap="12">
            <NFormItemGi label="关联策略">
              <NSelect v-model:value="policyFeedbackExclusionDraft.policyId" :options="crsPolicyOptions" />
            </NFormItemGi>
            <NFormItemGi label="作用域">
              <NSelect
                v-model:value="policyFeedbackExclusionDraft.scopeType"
                :options="scopeTypeOptions"
                @update:value="handlePolicyFeedbackExclusionDraftScopeChange"
              />
            </NFormItemGi>
            <NFormItemGi v-if="policyFeedbackExclusionDraft.scopeType !== 'global'" label="Host">
              <NInput v-model:value="policyFeedbackExclusionDraft.host" placeholder="例如：app.example.com" />
            </NFormItemGi>
            <NFormItemGi v-if="policyFeedbackExclusionDraft.scopeType === 'route'" label="Path">
              <NInput v-model:value="policyFeedbackExclusionDraft.path" placeholder="例如：/api/login" />
            </NFormItemGi>
            <NFormItemGi v-if="policyFeedbackExclusionDraft.scopeType === 'route'" label="Method">
              <NSelect
                v-model:value="policyFeedbackExclusionDraft.method"
                :options="methodOptions"
                clearable
                placeholder="可选"
              />
            </NFormItemGi>
            <NFormItemGi label="移除类型">
              <NSelect v-model:value="policyFeedbackExclusionDraft.removeType" :options="removeTypeOptions" />
            </NFormItemGi>
          </NGrid>
          <NFormItem label="规则名称">
            <NInput v-model:value="policyFeedbackExclusionDraft.name" />
          </NFormItem>
        </NForm>
        <NAlert type="info" :show-icon="false">
          <template #header>草稿差异对比</template>
          <div v-if="policyFeedbackExclusionDraftDiffItems.length === 0" class="text-xs text-gray-500">
            当前草稿与原反馈关键字段一致
          </div>
          <ul v-else class="text-xs text-gray-600 leading-6">
            <li v-for="item in policyFeedbackExclusionDraftDiffItems" :key="item.field">
              {{ item.field }}：{{ item.before || '空' }} ->
              {{ item.after || '空' }}
            </li>
          </ul>
        </NAlert>
        <div v-if="policyFeedbackExclusionCandidateOptions.length > 1">
          <div class="mb-1 text-xs text-gray-500">候选移除值（建议文本匹配到多个候选）</div>
          <NSelect
            v-model:value="policyFeedbackExclusionDraftCandidateKey"
            :options="policyFeedbackExclusionCandidateOptions"
            placeholder="请选择 remove 值候选"
            @update:value="handlePolicyFeedbackExclusionCandidateChange"
          />
        </div>
        <div>
          <div class="text-xs text-gray-500">移除值</div>
          <NInput
            v-model:value="policyFeedbackExclusionDraft.removeValue"
            :placeholder="policyFeedbackExclusionDraft.removeType === 'id' ? '例如：920350' : '例如：attack-sqli'"
          />
        </div>
        <div>
          <div class="text-xs text-gray-500">描述草稿</div>
          <NInput
            v-model:value="policyFeedbackExclusionDraft.description"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
          />
        </div>
        <NAlert v-if="!policyFeedbackExclusionDraft.removeValue" type="warning" :show-icon="true">
          建议文本未解析到可用的 remove 值，请在下一步表单中补充后再保存。
        </NAlert>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="policyFeedbackExclusionDraftModalVisible = false">取消</NButton>
          <NButton type="primary" @click="handleConfirmPolicyFeedbackExclusionDraft">确认生成</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="rollbackModalVisible" preset="card" title="回滚版本" class="w-520px">
      <NForm
        ref="rollbackFormRef"
        :model="rollbackForm"
        :rules="rollbackRules"
        label-placement="left"
        label-width="110"
      >
        <NFormItem label="回滚目标" path="target">
          <NRadioGroup v-model:value="rollbackForm.target">
            <NSpace>
              <NRadio value="last_good">last_good</NRadio>
              <NRadio value="version">指定版本</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="rollbackForm.target === 'version'" label="版本号" path="version">
          <NInput v-model:value="rollbackForm.version" placeholder="例如：v4.23.0" />
        </NFormItem>
      </NForm>

      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="rollbackModalVisible = false">取消</NButton>
          <NButton type="warning" :loading="rollbackSubmitting" @click="handleSubmitRollback">确认回滚</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
:deep(.security-tabs-hide-nav > .n-tabs-nav) {
  display: none;
}

:deep(.n-data-table .n-data-table-th__title) {
  white-space: nowrap;
}
</style>
