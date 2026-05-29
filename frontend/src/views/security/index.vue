<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { type FormInst, type FormRules, type UploadFileInfo, useDialog, useMessage } from 'naive-ui';
import { type WafKind, type WafSourceItem, fetchWafSourceList, uploadWafPackage } from '@/service/api/caddy-source';
import { type WafPolicyRemoveType, type WafPolicyScopeType } from '@/service/api/caddy-policy';
import { type WafJobItem } from '@/service/api/caddy-release-job';
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
  formatDateTime,
  formatRatePercent,
  mapJobActionLabel,
  mapJobMessage,
  mapJobStatusLabel,
  mapJobStatusType,
  mapJobTriggerModeLabel,
  mapPolicyEngineModeType,
  mapPolicyFeedbackSLAStatusLabel,
  mapPolicyFeedbackSLAStatusTagType,
  mapPolicyFeedbackStatusLabel,
  mapPolicyFeedbackStatusTagType,
  mapPolicyRevisionStatusType,
  mapReleaseStatusType
} from './security-mappers';
import {
  crsTemplatePresetMap,
  jobActionOptions,
  jobStatusOptions,
  methodOptions,
  policyFeedbackSLAStatusOptions,
  policyFeedbackStatusFilterOptions,
  releaseStatusOptions,
  scopeTypeOptions
} from './security-options';
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
import SourceFormModal from './modals/SourceFormModal.vue';
import PolicyFormModal from './modals/PolicyFormModal.vue';
import UploadPackageModal from './modals/UploadPackageModal.vue';
import ExclusionFormModal from './modals/ExclusionFormModal.vue';
import BindingFormModal from './modals/BindingFormModal.vue';
import FeedbackFormModal from './modals/FeedbackFormModal.vue';
import FeedbackProcessModal from './modals/FeedbackProcessModal.vue';
import FeedbackBatchProcessModal from './modals/FeedbackBatchProcessModal.vue';
import ExclusionDraftModal from './modals/ExclusionDraftModal.vue';
import RollbackModal from './modals/RollbackModal.vue';
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
import { createDateTimeValidator, createMethodValidator, createStatusCodeValidator } from './security-validators';

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
  method: createMethodValidator(methodOptions.map(item => item.value)),
  status: createStatusCodeValidator(),
  dueAt: createDateTimeValidator(),
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
  dueAt: createDateTimeValidator()
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
  return true;
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
      v-else-if="activeMenu === 'ops'"
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

    <SourceFormModal
      v-model:show="sourceModalVisible"
      :form="sourceForm"
      :form-ref="sourceFormRef"
      :rules="sourceRules"
      :submitting="sourceSubmitting"
      :title="sourceModalTitle"
      :handle-submit-source="handleSubmitSource"
      :apply-default-source="applyDefaultSource"
    />

    <PolicyFormModal
      v-model:show="policyModalVisible"
      :form="policyForm"
      :form-ref="policyFormRef"
      :rules="policyRules"
      :submitting="policySubmitting"
      :title="policyModalTitle"
      :handle-submit-policy="handleSubmitPolicy"
    />

    <UploadPackageModal
      v-model:show="uploadModalVisible"
      :form="uploadForm"
      :form-ref="uploadFormRef"
      :rules="uploadRules"
      :submitting="uploadSubmitting"
      :handle-submit-upload="handleSubmitUpload"
      :handle-before-upload="handleBeforeUpload"
      :handle-remove-upload="handleRemoveUpload"
    />

    <ExclusionFormModal
      v-model:show="exclusionModalVisible"
      :form="exclusionForm"
      :form-ref="exclusionFormRef"
      :rules="exclusionRules"
      :submitting="exclusionSubmitting"
      :title="exclusionModalTitle"
      :crs-policy-options="crsPolicyOptions"
      :should-focus-remove-value="shouldFocusExclusionRemoveValue"
      :handle-submit-exclusion="handleSubmitExclusion"
      @focused-remove-value="shouldFocusExclusionRemoveValue = false"
    />

    <BindingFormModal
      v-model:show="bindingModalVisible"
      :form="bindingForm"
      :form-ref="bindingFormRef"
      :rules="bindingRules"
      :submitting="bindingSubmitting"
      :title="bindingModalTitle"
      :crs-policy-options="crsPolicyOptions"
      :handle-submit-binding="handleSubmitBinding"
    />

    <FeedbackFormModal
      v-model:show="policyFeedbackModalVisible"
      :form="policyFeedbackForm"
      :form-ref="policyFeedbackFormRef"
      :rules="policyFeedbackRules"
      :submitting="policyFeedbackSubmitting"
      :crs-policy-options="crsPolicyOptions"
      :handle-submit-policy-feedback="handleSubmitPolicyFeedback"
    />

    <FeedbackProcessModal
      v-model:show="policyFeedbackProcessModalVisible"
      :form="policyFeedbackProcessForm"
      :form-ref="policyFeedbackProcessFormRef"
      :rules="policyFeedbackProcessRules"
      :submitting="policyFeedbackProcessSubmitting"
      :handle-submit-policy-feedback-process="handleSubmitPolicyFeedbackProcess"
    />

    <FeedbackBatchProcessModal
      v-model:show="policyFeedbackBatchProcessModalVisible"
      :form="policyFeedbackBatchProcessForm"
      :form-ref="policyFeedbackBatchProcessFormRef"
      :rules="policyFeedbackProcessRules"
      :submitting="policyFeedbackBatchProcessSubmitting"
      :checked-row-keys="policyFeedbackCheckedRowKeys"
      :handle-submit-policy-feedback-batch-process="handleSubmitPolicyFeedbackBatchProcess"
    />

    <ExclusionDraftModal
      v-model:show="policyFeedbackExclusionDraftModalVisible"
      :draft="policyFeedbackExclusionDraft"
      :crs-policy-options="crsPolicyOptions"
      :draft-candidate-key="policyFeedbackExclusionDraftCandidateKey"
      :candidate-options="policyFeedbackExclusionCandidateOptions"
      :diff-items="policyFeedbackExclusionDraftDiffItems"
      :handle-confirm-policy-feedback-exclusion-draft="handleConfirmPolicyFeedbackExclusionDraft"
      :handle-policy-feedback-exclusion-candidate-change="handlePolicyFeedbackExclusionCandidateChange"
      :handle-policy-feedback-exclusion-draft-scope-change="handlePolicyFeedbackExclusionDraftScopeChange"
    />

    <RollbackModal
      v-model:show="rollbackModalVisible"
      :form="rollbackForm"
      :form-ref="rollbackFormRef"
      :rules="rollbackRules"
      :submitting="rollbackSubmitting"
      :handle-submit-rollback="handleSubmitRollback"
    />
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
