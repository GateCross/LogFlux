<template>
  <div class="flex flex-col gap-3">
    <n-card :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">策略工作区</div>
          <div class="mt-1 text-xs text-gray-500">围绕单一策略完成基础设置、CRS 调优、规则例外和作用域绑定，统一预览、校验、发布与修正路径。</div>
        </div>
        <div class="flex flex-wrap gap-2">
          <n-button :type="activeSection === 'runtime' ? 'primary' : 'default'" @click="navigateToTab('runtime')">基础设置</n-button>
          <n-button :type="activeSection === 'crs' ? 'primary' : 'default'" @click="navigateToTab('crs')">CRS 调优</n-button>
          <n-button :type="activeSection === 'exclusion' ? 'primary' : 'default'" @click="navigateToTab('exclusion')">规则例外</n-button>
          <n-button :type="activeSection === 'binding' ? 'primary' : 'default'" @click="navigateToTab('binding')">策略绑定</n-button>
        </div>
      </div>

      <n-grid cols="4" x-gap="12" y-gap="10" class="mt-4">
        <n-gi>
          <n-statistic label="策略数量" :value="policyTable.length" />
        </n-gi>
        <n-gi>
          <n-statistic label="默认策略" :value="defaultPolicyName || '-'" />
        </n-gi>
        <n-gi>
          <n-statistic label="例外条目" :value="exclusionTotal" />
        </n-gi>
        <n-gi>
          <n-statistic label="绑定冲突" :value="bindingConflictGroups.length" />
        </n-gi>
      </n-grid>

      <n-grid cols="24" x-gap="12" y-gap="12" class="mt-4">
        <n-gi span="8 s:24 m:8">
          <n-card size="small" :bordered="false" class="h-full bg-#fafafc">
            <div class="mb-2 text-sm font-semibold">工作区上下文</div>
            <div class="text-xs text-gray-500 leading-6">
              <div>当前策略：{{ selectedPolicyName }}</div>
              <div>当前分区：{{ activeSectionLabel }}</div>
              <div v-if="hasPendingCrsTuningChanges" class="text-#d97706">存在未保存的 CRS 调优改动，发布前会触发保存校验。</div>
              <div v-else>当前策略参数与持久化状态一致。</div>
            </div>
          </n-card>
        </n-gi>
        <n-gi span="16 s:24 m:16">
          <n-card size="small" :bordered="false" class="h-full bg-#fafafc">
            <div class="mb-2 text-sm font-semibold">统一操作区提示</div>
            <div class="flex flex-col gap-1 text-xs text-gray-500">
              <div v-for="item in policyWorkspaceActions" :key="item">- {{ item }}</div>
            </div>
          </n-card>
        </n-gi>
      </n-grid>
    </n-card>

    <n-card v-if="activeSection === 'runtime'" :bordered="false" class="rounded-8px shadow-sm">
      <RuntimeTabContent
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
        :selected-policy-name="selectedPolicyName"
        :active-section-label="activeSectionLabel"
        :policy-workspace-actions="policyWorkspaceActions"
        :policy-preview-policy-name="policyPreviewPolicyName"
        :policy-preview-loading="policyPreviewLoading"
        :policy-preview-directives="policyPreviewDirectives"
        :policy-revision-columns="policyRevisionColumns"
        :policy-revision-table="policyRevisionTable"
        :policy-revision-loading="policyRevisionLoading"
        :policy-revision-pagination="policyRevisionPagination"
        :handle-policy-revision-page-change="handlePolicyRevisionPageChange"
        :handle-policy-revision-page-size-change="handlePolicyRevisionPageSizeChange"
      />
    </n-card>

    <n-card v-else-if="activeSection === 'crs'" :bordered="false" class="rounded-8px shadow-sm">
      <n-alert type="info" :show-icon="true" class="mb-3">
        可按模板快速设置 `tx.paranoia_level` 与 anomaly 阈值，并独立发布 CRS 调优参数。
      </n-alert>

      <n-alert v-if="crsTuningForm.crsParanoiaLevel >= 3" type="warning" :show-icon="true" class="mb-3">
        当前 PL={{ crsTuningForm.crsParanoiaLevel }}，误拦截风险上升。建议先在 DetectionOnly 观察并完成业务回归后再发布。
      </n-alert>

      <div class="mb-3 flex flex-wrap gap-2 items-center">
        <n-select
          v-model:value="crsTuningForm.policyId"
          :options="crsPolicyOptions"
          placeholder="选择要调优的策略"
          class="w-320px"
          @update:value="handleCrsPolicyChange"
        />
        <n-tag :bordered="false" type="info">当前模板：{{ mapCrsTemplateLabel(crsTuningForm.crsTemplate) }}</n-tag>
        <n-button @click="handleRefreshCrsPolicy">刷新策略</n-button>
      </div>

      <div class="mb-3 flex flex-wrap gap-2 items-center">
        <n-button secondary @click="applyCrsTemplatePreset('low_fp')">低误报模板</n-button>
        <n-button secondary @click="applyCrsTemplatePreset('balanced')">平衡模板</n-button>
        <n-button secondary @click="applyCrsTemplatePreset('high_blocking')">高拦截模板</n-button>
      </div>

      <n-form ref="crsTuningFormRef" :model="crsTuningForm" :rules="crsTuningRules" label-placement="left" label-width="220">
        <n-grid cols="3" x-gap="12">
          <n-form-item-gi label="Paranoia Level (PL)" path="crsParanoiaLevel">
            <n-input-number v-model:value="crsTuningForm.crsParanoiaLevel" :show-button="false" :min="1" :max="4" class="w-full" />
          </n-form-item-gi>
          <n-form-item-gi label="Inbound 阈值" path="crsInboundAnomalyThreshold">
            <n-input-number
              v-model:value="crsTuningForm.crsInboundAnomalyThreshold"
              :show-button="false"
              :min="1"
              :max="20"
              class="w-full"
            />
          </n-form-item-gi>
          <n-form-item-gi label="Outbound 阈值" path="crsOutboundAnomalyThreshold">
            <n-input-number
              v-model:value="crsTuningForm.crsOutboundAnomalyThreshold"
              :show-button="false"
              :min="1"
              :max="20"
              class="w-full"
            />
          </n-form-item-gi>
        </n-grid>
      </n-form>

      <div class="mb-3 flex flex-wrap gap-2 items-center">
        <n-button type="primary" :loading="crsTuningSubmitting" @click="handleSaveCrsTuning">保存调优参数</n-button>
        <n-button type="info" secondary :loading="policyPreviewLoading" @click="handlePreviewCrsTuning">预览</n-button>
        <n-button type="success" secondary :loading="crsTuningSubmitting" @click="handleValidateCrsTuning">校验</n-button>
        <n-button type="warning" secondary :loading="crsTuningSubmitting" @click="handlePublishCrsTuning">发布</n-button>
      </div>

      <n-card :bordered="false" size="small">
        <div class="mb-2 text-sm font-semibold">CRS 调优指令预览 {{ policyPreviewPolicyName ? `(${policyPreviewPolicyName})` : '' }}</div>
        <n-spin :show="policyPreviewLoading">
          <n-input
            :value="policyPreviewDirectives"
            type="textarea"
            :autosize="{ minRows: 8, maxRows: 14 }"
            readonly
            placeholder="点击“预览”查看 CRS 调优参数渲染后的 directives"
          />
        </n-spin>
      </n-card>

      <div class="mt-3 text-sm font-semibold">调优发布记录</div>
      <n-data-table
        remote
        class="mt-2 min-h-220px"
        :columns="policyRevisionColumns"
        :data="policyRevisionTable"
        :loading="policyRevisionLoading"
        :pagination="policyRevisionPagination"
        :row-key="row => row.id"
        :scroll-x="1100"
        :resizable="true"
        @update:page="handlePolicyRevisionPageChange"
        @update:page-size="handlePolicyRevisionPageSizeChange"
      />
    </n-card>

    <n-card v-else-if="activeSection === 'exclusion'" :bordered="false" class="rounded-8px shadow-sm">
      <n-alert type="info" :show-icon="true" class="mb-3">
        用于处理误报，支持按全局/站点/路由维度配置 `removeById/removeByTag`。
      </n-alert>

      <div class="mb-3 flex flex-wrap gap-2 items-center">
        <n-select v-model:value="exclusionQuery.policyId" :options="crsPolicyOptions" clearable placeholder="策略" class="w-240px" />
        <n-select v-model:value="exclusionQuery.scopeType" :options="scopeTypeOptions" clearable placeholder="作用域" class="w-180px" />
        <n-input v-model:value="exclusionQuery.name" placeholder="按名称搜索" clearable class="w-220px" @keyup.enter="fetchExclusions" />
        <n-button type="primary" @click="fetchExclusions">
          <template #icon>
            <icon-carbon-search />
          </template>
          查询
        </n-button>
        <n-button @click="resetExclusionQuery">重置</n-button>
        <n-button type="primary" @click="handleAddExclusion">
          <template #icon>
            <icon-ic-round-plus />
          </template>
          新增例外
        </n-button>
      </div>

      <n-data-table
        remote
        :columns="exclusionColumns"
        :data="exclusionTable"
        :loading="exclusionLoading"
        :pagination="exclusionPagination"
        :row-key="row => row.id"
        :max-height="tableFixedHeight"
        class="min-h-260px"
        :scroll-x="1600"
        :resizable="true"
        @update:page="handleExclusionPageChange"
        @update:page-size="handleExclusionPageSizeChange"
      />
    </n-card>

    <n-card v-else :bordered="false" class="rounded-8px shadow-sm">
      <n-alert type="warning" :show-icon="true" class="mb-3">
        同一作用域 + 同一优先级仅允许一个生效绑定，冲突会阻止策略发布。
      </n-alert>

      <div class="mb-3 flex flex-wrap gap-2 items-center">
        <n-select v-model:value="bindingQuery.policyId" :options="crsPolicyOptions" clearable placeholder="策略" class="w-240px" />
        <n-select v-model:value="bindingQuery.scopeType" :options="scopeTypeOptions" clearable placeholder="作用域" class="w-180px" />
        <n-input v-model:value="bindingQuery.name" placeholder="按名称搜索" clearable class="w-220px" @keyup.enter="fetchBindings" />
        <n-button type="primary" @click="fetchBindings">
          <template #icon>
            <icon-carbon-search />
          </template>
          查询
        </n-button>
        <n-button @click="resetBindingQuery">重置</n-button>
        <n-button type="primary" @click="handleAddBinding">
          <template #icon>
            <icon-ic-round-plus />
          </template>
          新增绑定
        </n-button>
      </div>

      <n-data-table
        remote
        :columns="bindingColumns"
        :data="bindingTable"
        :loading="bindingLoading"
        :pagination="bindingPagination"
        :row-key="row => row.id"
        :max-height="tableFixedHeight"
        class="min-h-260px"
        :scroll-x="1600"
        :resizable="true"
        @update:page="handleBindingPageChange"
        @update:page-size="handleBindingPageSizeChange"
      />

      <n-alert v-if="bindingConflictGroups.length > 0" type="error" :show-icon="true" class="mt-3">
        检测到 {{ bindingConflictGroups.length }} 组作用域冲突（同作用域 + 同优先级），发布会被阻断。首条冲突：
        {{ bindingConflictGroups[0].scopeType }} /
        {{ bindingConflictGroups[0].host || '-' }} /
        {{ bindingConflictGroups[0].path || '-' }} /
        {{ bindingConflictGroups[0].method || '-' }} /
        priority={{ bindingConflictGroups[0].priority }} /
        count={{ bindingConflictGroups[0].count }}
      </n-alert>

      <n-card :bordered="false" size="small" class="mt-3">
        <div class="mb-2 text-sm font-semibold">策略叠加执行顺序（当前列表）</div>
        <n-data-table
          :columns="bindingEffectiveColumns"
          :data="bindingEffectivePreview"
          :pagination="false"
          :row-key="row => row.id"
          :max-height="280"
          class="min-h-120px"
        />
      </n-card>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumns, FormInst, FormRules, PaginationProps } from 'naive-ui';
import type {
  WafPolicyBindingItem,
  WafPolicyItem,
  WafPolicyRevisionItem,
  WafPolicyScopeType,
  WafRuleExclusionItem,
  WafPolicyCrsTemplate
} from '@/service/api/caddy-policy';
import RuntimeTabContent from '../tabs/RuntimeTabContent.vue';

type PolicySection = 'runtime' | 'crs' | 'exclusion' | 'binding';

interface BindingConflictGroup {
  scopeType: string;
  host: string;
  path: string;
  method: string;
  priority: number;
  count: number;
}

interface BindingEffectiveItem {
  id: number;
  order: number;
  policyId: number;
  policyName: string;
  scopeType: string;
  host: string;
  path: string;
  method: string;
  priority: number;
}

defineProps<{
  activeSection: PolicySection;
  navigateToTab: (tab: PolicySection) => void | Promise<void>;
  policyQuery: { name: string };
  policyColumns: DataTableColumns<WafPolicyItem>;
  policyTable: WafPolicyItem[];
  policyLoading: boolean;
  policyPagination: PaginationProps;
  tableFixedHeight: number;
  fetchPolicies: () => void | Promise<void>;
  resetPolicyQuery: () => void;
  handleAddPolicy: () => void;
  handlePolicyPageChange: (page: number) => void;
  handlePolicyPageSizeChange: (pageSize: number) => void;
  policyPreviewPolicyName: string;
  policyPreviewLoading: boolean;
  policyPreviewDirectives: string;
  policyRevisionColumns: DataTableColumns<WafPolicyRevisionItem>;
  policyRevisionTable: WafPolicyRevisionItem[];
  policyRevisionLoading: boolean;
  policyRevisionPagination: PaginationProps;
  handlePolicyRevisionPageChange: (page: number) => void;
  handlePolicyRevisionPageSizeChange: (pageSize: number) => void;
  defaultPolicyName: string;
  selectedPolicyName: string;
  activeSectionLabel: string;
  hasPendingCrsTuningChanges: boolean;
  policyWorkspaceActions: string[];
  exclusionTotal: number;
  crsTuningSubmitting: boolean;
  crsTuningFormRef: FormInst | null;
  crsTuningForm: {
    policyId: number;
    crsTemplate: WafPolicyCrsTemplate;
    crsParanoiaLevel: number;
    crsInboundAnomalyThreshold: number;
    crsOutboundAnomalyThreshold: number;
  };
  crsPolicyOptions: Array<{ label: string; value: number }>;
  crsTuningRules: FormRules;
  handleCrsPolicyChange: (policyId: number | null) => void;
  mapCrsTemplateLabel: (value: WafPolicyCrsTemplate) => string;
  handleRefreshCrsPolicy: () => void;
  applyCrsTemplatePreset: (template: Exclude<WafPolicyCrsTemplate, 'custom'>) => void;
  handleSaveCrsTuning: () => void | Promise<void>;
  handlePreviewCrsTuning: () => void | Promise<void>;
  handleValidateCrsTuning: () => void | Promise<void>;
  handlePublishCrsTuning: () => void | Promise<void>;
  exclusionQuery: { policyId: number | null; scopeType: '' | WafPolicyScopeType | null; name: string };
  scopeTypeOptions: Array<{ label: string; value: WafPolicyScopeType }>;
  fetchExclusions: () => void | Promise<void>;
  resetExclusionQuery: () => void;
  handleAddExclusion: () => void;
  exclusionColumns: DataTableColumns<WafRuleExclusionItem>;
  exclusionTable: WafRuleExclusionItem[];
  exclusionLoading: boolean;
  exclusionPagination: PaginationProps;
  handleExclusionPageChange: (page: number) => void;
  handleExclusionPageSizeChange: (pageSize: number) => void;
  bindingQuery: { policyId: number | null; scopeType: '' | WafPolicyScopeType | null; name: string };
  fetchBindings: () => void | Promise<void>;
  resetBindingQuery: () => void;
  handleAddBinding: () => void;
  bindingColumns: DataTableColumns<WafPolicyBindingItem>;
  bindingTable: WafPolicyBindingItem[];
  bindingLoading: boolean;
  bindingPagination: PaginationProps;
  handleBindingPageChange: (page: number) => void;
  handleBindingPageSizeChange: (pageSize: number) => void;
  bindingConflictGroups: BindingConflictGroup[];
  bindingEffectiveColumns: DataTableColumns<BindingEffectiveItem>;
  bindingEffectivePreview: BindingEffectiveItem[];
}>();
</script>
