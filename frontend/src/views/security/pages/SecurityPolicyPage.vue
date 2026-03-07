<script setup lang="ts">
import { computed } from 'vue';
import type { DataTableColumns, FormRules, PaginationProps } from 'naive-ui';
import type {
  WafPolicyBindingItem,
  WafPolicyCrsTemplate,
  WafPolicyItem,
  WafPolicyRevisionItem,
  WafPolicyScopeType,
  WafRuleExclusionItem
} from '@/service/api/caddy-policy';
import type { BindingConflictGroup, BindingEffectiveItem } from '../composables/useWafBinding';
import RuntimeTabContent from '../tabs/RuntimeTabContent.vue';

type PolicySection = 'runtime' | 'crs' | 'exclusion' | 'binding';

const props = defineProps<{
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
  exclusionQuery: {
    policyId: number | null;
    scopeType: '' | WafPolicyScopeType | null;
    name: string;
  };
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
  bindingQuery: {
    policyId: number | null;
    scopeType: '' | WafPolicyScopeType | null;
    name: string;
  };
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

const crsTuningFormModel = computed({
  get: () => props.crsTuningForm,
  set: () => undefined
});

const exclusionQueryModel = computed({
  get: () => props.exclusionQuery,
  set: () => undefined
});

const bindingQueryModel = computed({
  get: () => props.bindingQuery,
  set: () => undefined
});
</script>

<template>
  <div class="flex flex-col gap-3">
    <NCard :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">策略工作区</div>
          <div class="mt-1 text-xs text-gray-500">
            围绕单一策略完成基础设置、CRS 调优、规则例外和作用域绑定，统一预览、校验、发布与修正路径。
          </div>
        </div>
        <div class="flex flex-wrap gap-2">
          <NButton :type="activeSection === 'runtime' ? 'primary' : 'default'" @click="navigateToTab('runtime')">
            基础设置
          </NButton>
          <NButton :type="activeSection === 'crs' ? 'primary' : 'default'" @click="navigateToTab('crs')">
            CRS 调优
          </NButton>
          <NButton :type="activeSection === 'exclusion' ? 'primary' : 'default'" @click="navigateToTab('exclusion')">
            规则例外
          </NButton>
          <NButton :type="activeSection === 'binding' ? 'primary' : 'default'" @click="navigateToTab('binding')">
            策略绑定
          </NButton>
        </div>
      </div>

      <NGrid cols="4" x-gap="12" y-gap="10" class="mt-4">
        <NGi>
          <NStatistic label="策略数量" :value="policyTable.length" />
        </NGi>
        <NGi>
          <NStatistic label="默认策略" :value="defaultPolicyName || '-'" />
        </NGi>
        <NGi>
          <NStatistic label="例外条目" :value="exclusionTotal" />
        </NGi>
        <NGi>
          <NStatistic label="绑定冲突" :value="bindingConflictGroups.length" />
        </NGi>
      </NGrid>

      <NGrid cols="24" x-gap="12" y-gap="12" class="mt-4">
        <NGi span="8 s:24 m:8">
          <NCard size="small" :bordered="false" class="h-full bg-#fafafc">
            <div class="mb-2 text-sm font-semibold">工作区上下文</div>
            <div class="text-xs text-gray-500 leading-6">
              <div>当前策略：{{ selectedPolicyName }}</div>
              <div>当前分区：{{ activeSectionLabel }}</div>
              <div v-if="hasPendingCrsTuningChanges" class="text-#d97706">
                存在未保存的 CRS 调优改动，发布前会触发保存校验。
              </div>
              <div v-else>当前策略参数与持久化状态一致。</div>
            </div>
          </NCard>
        </NGi>
        <NGi span="16 s:24 m:16">
          <NCard size="small" :bordered="false" class="h-full bg-#fafafc">
            <div class="mb-2 text-sm font-semibold">统一操作区提示</div>
            <div class="flex flex-col gap-1 text-xs text-gray-500">
              <div v-for="item in policyWorkspaceActions" :key="item">- {{ item }}</div>
            </div>
          </NCard>
        </NGi>
      </NGrid>
    </NCard>

    <NCard v-if="activeSection === 'runtime'" :bordered="false" class="rounded-8px shadow-sm">
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
    </NCard>

    <NCard v-else-if="activeSection === 'crs'" :bordered="false" class="rounded-8px shadow-sm">
      <NAlert type="info" :show-icon="true" class="mb-3">
        可按模板快速设置 `tx.paranoia_level` 与 anomaly 阈值，并独立发布 CRS 调优参数。
      </NAlert>

      <NAlert v-if="crsTuningFormModel.crsParanoiaLevel >= 3" type="warning" :show-icon="true" class="mb-3">
        当前 PL={{ crsTuningFormModel.crsParanoiaLevel }}，误拦截风险上升。建议先在 DetectionOnly
        观察并完成业务回归后再发布。
      </NAlert>

      <div class="mb-3 flex flex-wrap items-center gap-2">
        <NSelect
          v-model:value="crsTuningFormModel.policyId"
          :options="crsPolicyOptions"
          placeholder="选择要调优的策略"
          class="w-320px"
          @update:value="handleCrsPolicyChange"
        />
        <NTag :bordered="false" type="info">当前模板：{{ mapCrsTemplateLabel(crsTuningFormModel.crsTemplate) }}</NTag>
        <NButton @click="handleRefreshCrsPolicy">刷新策略</NButton>
      </div>

      <div class="mb-3 flex flex-wrap items-center gap-2">
        <NButton secondary @click="applyCrsTemplatePreset('low_fp')">低误报模板</NButton>
        <NButton secondary @click="applyCrsTemplatePreset('balanced')">平衡模板</NButton>
        <NButton secondary @click="applyCrsTemplatePreset('high_blocking')">高拦截模板</NButton>
      </div>

      <NForm :model="crsTuningFormModel" :rules="crsTuningRules" label-placement="left" label-width="220">
        <NGrid cols="3" x-gap="12">
          <NFormItemGi label="Paranoia Level (PL)" path="crsParanoiaLevel">
            <NInputNumber
              v-model:value="crsTuningFormModel.crsParanoiaLevel"
              :show-button="false"
              :min="1"
              :max="4"
              class="w-full"
            />
          </NFormItemGi>
          <NFormItemGi label="Inbound 阈值" path="crsInboundAnomalyThreshold">
            <NInputNumber
              v-model:value="crsTuningFormModel.crsInboundAnomalyThreshold"
              :show-button="false"
              :min="1"
              :max="20"
              class="w-full"
            />
          </NFormItemGi>
          <NFormItemGi label="Outbound 阈值" path="crsOutboundAnomalyThreshold">
            <NInputNumber
              v-model:value="crsTuningFormModel.crsOutboundAnomalyThreshold"
              :show-button="false"
              :min="1"
              :max="20"
              class="w-full"
            />
          </NFormItemGi>
        </NGrid>
      </NForm>

      <div class="mb-3 flex flex-wrap items-center gap-2">
        <NButton type="primary" :loading="crsTuningSubmitting" @click="handleSaveCrsTuning">保存调优参数</NButton>
        <NButton type="info" secondary :loading="policyPreviewLoading" @click="handlePreviewCrsTuning">预览</NButton>
        <NButton type="success" secondary :loading="crsTuningSubmitting" @click="handleValidateCrsTuning">校验</NButton>
        <NButton type="warning" secondary :loading="crsTuningSubmitting" @click="handlePublishCrsTuning">发布</NButton>
      </div>

      <NCard :bordered="false" size="small">
        <div class="mb-2 text-sm font-semibold">
          CRS 调优指令预览
          {{ policyPreviewPolicyName ? `(${policyPreviewPolicyName})` : '' }}
        </div>
        <NSpin :show="policyPreviewLoading">
          <NInput
            :value="policyPreviewDirectives"
            type="textarea"
            :autosize="{ minRows: 8, maxRows: 14 }"
            readonly
            placeholder="点击“预览”查看 CRS 调优参数渲染后的 directives"
          />
        </NSpin>
      </NCard>

      <div class="mt-3 text-sm font-semibold">调优发布记录</div>
      <NDataTable
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
    </NCard>

    <NCard v-else-if="activeSection === 'exclusion'" :bordered="false" class="rounded-8px shadow-sm">
      <NAlert type="info" :show-icon="true" class="mb-3">
        用于处理误报，支持按全局/站点/路由维度配置 `removeById/removeByTag`。
      </NAlert>

      <div class="mb-3 flex flex-wrap items-center gap-2">
        <NSelect
          v-model:value="exclusionQueryModel.policyId"
          :options="crsPolicyOptions"
          clearable
          placeholder="策略"
          class="w-240px"
        />
        <NSelect
          v-model:value="exclusionQueryModel.scopeType"
          :options="scopeTypeOptions"
          clearable
          placeholder="作用域"
          class="w-180px"
        />
        <NInput
          v-model:value="exclusionQueryModel.name"
          placeholder="按名称搜索"
          clearable
          class="w-220px"
          @keyup.enter="fetchExclusions"
        />
        <NButton type="primary" @click="fetchExclusions">
          <template #icon>
            <icon-carbon-search />
          </template>
          查询
        </NButton>
        <NButton @click="resetExclusionQuery">重置</NButton>
        <NButton type="primary" @click="handleAddExclusion">
          <template #icon>
            <icon-ic-round-plus />
          </template>
          新增例外
        </NButton>
      </div>

      <NDataTable
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
    </NCard>

    <NCard v-else :bordered="false" class="rounded-8px shadow-sm">
      <NAlert type="warning" :show-icon="true" class="mb-3">
        同一作用域 + 同一优先级仅允许一个生效绑定，冲突会阻止策略发布。
      </NAlert>

      <div class="mb-3 flex flex-wrap items-center gap-2">
        <NSelect
          v-model:value="bindingQueryModel.policyId"
          :options="crsPolicyOptions"
          clearable
          placeholder="策略"
          class="w-240px"
        />
        <NSelect
          v-model:value="bindingQueryModel.scopeType"
          :options="scopeTypeOptions"
          clearable
          placeholder="作用域"
          class="w-180px"
        />
        <NInput
          v-model:value="bindingQueryModel.name"
          placeholder="按名称搜索"
          clearable
          class="w-220px"
          @keyup.enter="fetchBindings"
        />
        <NButton type="primary" @click="fetchBindings">
          <template #icon>
            <icon-carbon-search />
          </template>
          查询
        </NButton>
        <NButton @click="resetBindingQuery">重置</NButton>
        <NButton type="primary" @click="handleAddBinding">
          <template #icon>
            <icon-ic-round-plus />
          </template>
          新增绑定
        </NButton>
      </div>

      <NDataTable
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

      <NAlert v-if="bindingConflictGroups.length > 0" type="error" :show-icon="true" class="mt-3">
        检测到 {{ bindingConflictGroups.length }} 组作用域冲突（同作用域 + 同优先级），发布会被阻断。首条冲突：
        {{ bindingConflictGroups[0].scopeType }} / {{ bindingConflictGroups[0].host || '-' }} /
        {{ bindingConflictGroups[0].path || '-' }} / {{ bindingConflictGroups[0].method || '-' }} / priority={{
          bindingConflictGroups[0].priority
        }}
        / count={{ bindingConflictGroups[0].count }}
      </NAlert>

      <NCard :bordered="false" size="small" class="mt-3">
        <div class="mb-2 text-sm font-semibold">策略叠加执行顺序（当前列表）</div>
        <NDataTable
          :columns="bindingEffectiveColumns"
          :data="bindingEffectivePreview"
          :pagination="false"
          :row-key="row => row.id"
          :max-height="280"
          class="min-h-120px"
        />
      </NCard>
    </NCard>
  </div>
</template>
