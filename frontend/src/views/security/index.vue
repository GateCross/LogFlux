<template>
  <div class="h-full flex flex-col gap-3">
    <n-alert type="info" :show-icon="true" class="rounded-8px">
      <template #header>{{ pageTitle }}</template>
      <div>
        CRS 支持在线同步（含检查）、上传、激活与回滚；Coraza 引擎依赖 Caddy 二进制，仅提供 GitHub Release 版本检查，不支持在线替换引擎。
      </div>
    </n-alert>

    <n-card :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">Coraza 引擎版本检查</div>
          <div class="text-xs text-gray-500 mt-1">用于发现 Coraza 引擎新版本并生成升级建议（需通过镜像发布流程升级）。</div>
        </div>
        <div class="flex gap-2">
          <n-button size="small" :loading="engineLoading" @click="handleRefreshEngineStatus">刷新状态</n-button>
          <n-button size="small" type="primary" :loading="engineChecking" @click="handleCheckEngine">检查上游版本</n-button>
        </div>
      </div>

      <n-grid cols="4" x-gap="12" y-gap="10" class="mt-4">
        <n-gi>
          <div class="text-xs text-gray-500">当前版本</div>
          <div class="text-sm font-medium">{{ displayEngineValue(engineStatus?.currentVersion) }}</div>
        </n-gi>
        <n-gi>
          <div class="text-xs text-gray-500">最新版本</div>
          <div class="text-sm font-medium">{{ displayEngineValue(engineStatus?.latestVersion) }}</div>
        </n-gi>
        <n-gi>
          <div class="text-xs text-gray-500">可升级</div>
          <div class="text-sm font-medium">
            <n-tag :type="engineStatus?.canUpgrade ? 'warning' : 'success'" :bordered="false">
              {{ engineStatus?.canUpgrade ? '是' : '否' }}
            </n-tag>
          </div>
        </n-gi>
        <n-gi>
          <div class="text-xs text-gray-500">最近检查时间</div>
          <div class="text-sm font-medium">{{ displayEngineValue(engineStatus?.checkedAt) }}</div>
        </n-gi>
      </n-grid>

      <n-alert v-if="engineUnavailable" type="warning" :show-icon="true" class="mt-4">
        当前引擎状态接口暂不可用，已切换为占位模式，请检查后端日志。
      </n-alert>
      <n-alert v-else-if="engineStatus?.message" type="info" :show-icon="true" class="mt-4">
        {{ engineStatus?.message }}
      </n-alert>
    </n-card>

    <n-card :bordered="false" class="rounded-8px shadow-sm">
      <n-tabs v-model:value="activeTab" type="line" animated>
        <n-tab-pane name="source" tab="更新源配置">
          <div class="mb-3 flex flex-wrap gap-2 items-center">
            <n-input v-model:value="sourceQuery.name" placeholder="按名称搜索" clearable class="w-220px" @keyup.enter="fetchSources" />
            <n-button type="primary" @click="fetchSources">
              <template #icon>
                <icon-carbon-search />
              </template>
              查询
            </n-button>
            <n-button @click="resetSourceQuery">重置</n-button>
            <n-button type="primary" @click="handleAddSource">
              <template #icon>
                <icon-ic-round-plus />
              </template>
              新增源
            </n-button>
            <n-button type="success" @click="openUploadModal">
              <template #icon>
                <icon-carbon-cloud-upload />
              </template>
              上传规则包
            </n-button>
          </div>

          <n-data-table
            remote
            :columns="sourceColumns"
            :data="sourceTable"
            :loading="sourceLoading"
            :pagination="sourcePagination"
            :row-key="row => row.id"
            :max-height="tableFixedHeight"
            class="min-h-260px"
            @update:page="handleSourcePageChange"
            @update:page-size="handleSourcePageSizeChange"
          />
        </n-tab-pane>

        <n-tab-pane name="runtime" tab="运行模式">
          <n-alert type="warning" :show-icon="true" class="mb-3">
            建议先使用 DetectionOnly（仅检测）观察，再切换到 On（阻断）。On 模式发布会触发二次确认。
          </n-alert>

          <div class="mb-3 flex flex-wrap gap-2 items-center">
            <n-input v-model:value="policyQuery.name" placeholder="按策略名称搜索" clearable class="w-220px" @keyup.enter="fetchPolicies" />
            <n-button type="primary" @click="fetchPolicies">
              <template #icon>
                <icon-carbon-search />
              </template>
              查询
            </n-button>
            <n-button @click="resetPolicyQuery">重置</n-button>
            <n-button type="primary" @click="handleAddPolicy">
              <template #icon>
                <icon-ic-round-plus />
              </template>
              新增策略
            </n-button>
          </div>

          <n-data-table
            remote
            :columns="policyColumns"
            :data="policyTable"
            :loading="policyLoading"
            :pagination="policyPagination"
            :row-key="row => row.id"
            :max-height="tableFixedHeight"
            class="min-h-260px"
            :scroll-x="1700"
            :resizable="true"
            @update:page="handlePolicyPageChange"
            @update:page-size="handlePolicyPageSizeChange"
          />

          <n-card :bordered="false" size="small" class="mt-3">
            <div class="text-sm font-semibold mb-2">策略指令预览 {{ policyPreviewPolicyName ? `(${policyPreviewPolicyName})` : '' }}</div>
            <n-spin :show="policyPreviewLoading">
              <n-input
                :value="policyPreviewDirectives"
                type="textarea"
                :autosize="{ minRows: 8, maxRows: 14 }"
                readonly
                placeholder="点击策略列表中的“预览”查看渲染后的 Coraza directives"
              />
            </n-spin>
          </n-card>

          <div class="mt-3 text-sm font-semibold">最近发布记录</div>
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
        </n-tab-pane>

        <n-tab-pane name="crs" tab="CRS 调优">
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
                <n-input-number
                  v-model:value="crsTuningForm.crsParanoiaLevel"
                  :show-button="false"
                  :min="1"
                  :max="4"
                  class="w-full"
                />
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
            <div class="text-sm font-semibold mb-2">CRS 调优指令预览 {{ policyPreviewPolicyName ? `(${policyPreviewPolicyName})` : '' }}</div>
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
        </n-tab-pane>

        <n-tab-pane name="exclusion" tab="规则例外">
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
        </n-tab-pane>

        <n-tab-pane name="binding" tab="策略绑定">
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
            <div class="text-sm font-semibold mb-2">策略叠加执行顺序（当前列表）</div>
            <n-data-table
              :columns="bindingEffectiveColumns"
              :data="bindingEffectivePreview"
              :pagination="false"
              :row-key="row => row.id"
              :max-height="280"
              class="min-h-120px"
            />
          </n-card>
        </n-tab-pane>

        <n-tab-pane name="observe" tab="策略观测">
          <n-alert type="info" :show-icon="true" class="mb-3">
            统计口径基于策略绑定作用域与请求日志；“疑似误报”当前为启发式指标（安全端点被拦截），用于辅助调优参考。
          </n-alert>

          <div class="mb-3 flex flex-wrap gap-2 items-center">
            <n-select
              v-model:value="policyStatsQuery.policyId"
              :options="policyStatsPolicyOptions"
              clearable
              placeholder="策略范围"
              class="w-240px"
            />
            <n-select v-model:value="policyStatsQuery.window" :options="observeWindowOptions" class="w-180px" />
            <n-input-number
              v-model:value="policyStatsQuery.intervalSec"
              :show-button="false"
              :min="60"
              :max="86400"
              placeholder="趋势粒度（秒）"
              class="w-180px"
            />
            <n-input-number
              v-model:value="policyStatsQuery.topN"
              :show-button="false"
              :min="1"
              :max="50"
              placeholder="TopN"
              class="w-120px"
            />
            <n-button type="primary" :loading="policyStatsLoading" @click="fetchPolicyStats">
              <template #icon>
                <icon-carbon-search />
              </template>
              查询
            </n-button>
            <n-button @click="resetPolicyStatsQuery">重置</n-button>
            <n-button :disabled="!hasPolicyStatsDrillFilters" @click="clearPolicyStatsDrillFilters">清空下钻</n-button>
            <n-select
              v-model:value="policyFeedbackStatusFilter"
              :options="policyFeedbackStatusFilterOptions"
              placeholder="反馈状态"
              class="w-160px"
              @update:value="handlePolicyFeedbackStatusFilterChange"
            />
            <n-input
              v-model:value="policyFeedbackAssigneeFilter"
              clearable
              placeholder="责任人"
              class="w-160px"
              @keyup.enter="handlePolicyFeedbackStatusFilterChange"
            />
            <n-select
              v-model:value="policyFeedbackSLAStatusFilter"
              :options="policyFeedbackSLAStatusOptions"
              class="w-160px"
              @update:value="handlePolicyFeedbackStatusFilterChange"
            />
            <n-button type="warning" secondary @click="openPolicyFeedbackModal">标记误报</n-button>
            <n-button type="warning" secondary :disabled="!hasPolicyFeedbackSelection" @click="openPolicyFeedbackBatchProcessModal">
              批量处理（{{ policyFeedbackCheckedRowKeys.length }}）
            </n-button>
            <n-button secondary :loading="policyFeedbackLoading" @click="fetchPolicyFalsePositiveFeedbacks">刷新反馈</n-button>
            <n-button secondary @click="handleCopyPolicyStatsLink">
              <template #icon>
                <icon-carbon-link />
              </template>
              复制筛选链接
            </n-button>
            <n-button secondary :disabled="!policyStatsPreviousSnapshot" @click="handleExportPolicyStatsCompareCsv">导出对比 CSV</n-button>
            <n-button secondary @click="handleExportPolicyStatsCsv">导出 CSV</n-button>
          </div>

          <n-grid cols="5" x-gap="12" y-gap="10">
            <n-gi>
              <n-card size="small" :bordered="false">
                <div class="text-xs text-gray-500">命中</div>
                <div class="text-lg font-semibold">{{ policyStatsSummary.hitCount || 0 }}</div>
              </n-card>
            </n-gi>
            <n-gi>
              <n-card size="small" :bordered="false">
                <div class="text-xs text-gray-500">拦截</div>
                <div class="text-lg font-semibold">{{ policyStatsSummary.blockedCount || 0 }}</div>
              </n-card>
            </n-gi>
            <n-gi>
              <n-card size="small" :bordered="false">
                <div class="text-xs text-gray-500">放行</div>
                <div class="text-lg font-semibold">{{ policyStatsSummary.allowedCount || 0 }}</div>
              </n-card>
            </n-gi>
            <n-gi>
              <n-card size="small" :bordered="false">
                <div class="text-xs text-gray-500">疑似误报</div>
                <div class="text-lg font-semibold">{{ policyStatsSummary.suspectedFalsePositiveCount || 0 }}</div>
              </n-card>
            </n-gi>
            <n-gi>
              <n-card size="small" :bordered="false">
                <div class="text-xs text-gray-500">拦截率</div>
                <div class="text-lg font-semibold">{{ formatRatePercent(policyStatsSummary.blockRate) }}</div>
              </n-card>
            </n-gi>
          </n-grid>

          <div class="mt-3 text-xs text-gray-500">
            统计区间：{{ policyStatsRange.startTime || '-' }} ~ {{ policyStatsRange.endTime || '-' }}，粒度 {{ policyStatsRange.intervalSec || 0 }} 秒
          </div>
          <div v-if="policyStatsPreviousSnapshot" class="mt-1 text-xs text-gray-500">对比基线：{{ policyStatsPreviousSnapshot.capturedAt }}</div>
          <div class="mt-1 text-xs text-gray-500">
            下钻过滤：Host={{ policyStatsQuery.host || '-' }} / Path={{ policyStatsQuery.path || '-' }} / Method={{ policyStatsQuery.method || '-' }}
          </div>
          <div class="mt-1 text-xs text-gray-500">下钻顺序：先点 Top Host，再点 Top Path，最后点 Top Method。</div>
          <div class="mt-2 flex flex-wrap gap-2 items-center">
            <span class="text-xs text-gray-500">当前下钻标签：</span>
            <n-tag v-if="policyStatsQuery.host" closable size="small" @close="() => clearPolicyStatsDrillLevel('host')">
              Host: {{ policyStatsQuery.host }}
            </n-tag>
            <n-tag v-if="policyStatsQuery.path" closable size="small" type="info" @close="() => clearPolicyStatsDrillLevel('path')">
              Path: {{ policyStatsQuery.path }}
            </n-tag>
            <n-tag v-if="policyStatsQuery.method" closable size="small" type="warning" @close="() => clearPolicyStatsDrillLevel('method')">
              Method: {{ policyStatsQuery.method }}
            </n-tag>
            <span v-if="!hasPolicyStatsDrillFilters" class="text-xs text-gray-400">-</span>
          </div>

          <n-card :bordered="false" size="small" class="mt-3">
            <div class="text-sm font-semibold mb-2">命中趋势</div>
            <n-data-table
              :columns="policyStatsTrendColumns"
              :data="policyStatsTrend"
              :loading="policyStatsLoading"
              :pagination="false"
              :row-key="row => row.time"
              :max-height="260"
              class="min-h-120px"
            />
          </n-card>

          <n-card :bordered="false" size="small" class="mt-3">
            <div class="text-sm font-semibold mb-2">策略命中统计</div>
            <n-data-table
              :columns="policyStatsColumns"
              :data="policyStatsTable"
              :loading="policyStatsLoading"
              :pagination="false"
              :row-key="row => row.policyId"
              :max-height="320"
              class="min-h-160px"
            />
          </n-card>

          <n-card :bordered="false" size="small" class="mt-3">
            <div class="text-sm font-semibold mb-2">人工误报反馈（当前筛选口径）</div>
            <n-data-table
              remote
              :columns="policyFeedbackColumns"
              :data="policyFeedbackTable"
              :loading="policyFeedbackLoading"
              :pagination="policyFeedbackPagination"
              :checked-row-keys="policyFeedbackCheckedRowKeysInPage"
              :row-key="row => row.id"
              :max-height="300"
              class="min-h-140px"
              :scroll-x="1800"
              @update:checked-row-keys="handlePolicyFeedbackCheckedRowKeysChange"
              @update:page="handlePolicyFeedbackPageChange"
              @update:page-size="handlePolicyFeedbackPageSizeChange"
            />
          </n-card>

          <n-grid cols="3" x-gap="12" y-gap="12" class="mt-3">
            <n-gi>
              <n-card :bordered="false" size="small">
                <div class="text-sm font-semibold mb-2 flex items-center gap-2">
                  <span>Top Host</span>
                  <n-tooltip trigger="hover">
                    <template #trigger>
                      <span class="inline-flex items-center text-green-600">
                        <icon-carbon-unlocked />
                      </span>
                    </template>
                    {{ policyStatsDrillHint('host') }}
                  </n-tooltip>
                  <n-tag size="small" type="success" :bordered="false">{{ policyStatsDrillStatusLabel('host') }}</n-tag>
                </div>
                <n-data-table
                  :columns="policyStatsDimensionColumns"
                  :data="policyStatsTopHosts"
                  :loading="policyStatsLoading"
                  :pagination="false"
                  :row-props="buildPolicyStatsDimensionRowProps('host')"
                  :row-key="row => `host-${row.key}`"
                  :max-height="260"
                  class="min-h-120px"
                />
              </n-card>
            </n-gi>
            <n-gi>
              <n-card :bordered="false" size="small">
                <div class="text-sm font-semibold mb-2 flex items-center gap-2">
                  <span>Top Path</span>
                  <n-tooltip trigger="hover">
                    <template #trigger>
                      <span class="inline-flex items-center" :class="isPolicyStatsDrillUnlocked('path') ? 'text-green-600' : 'text-gray-400'">
                        <icon-carbon-unlocked v-if="isPolicyStatsDrillUnlocked('path')" />
                        <icon-carbon-locked v-else />
                      </span>
                    </template>
                    {{ policyStatsDrillHint('path') }}
                  </n-tooltip>
                  <n-tag size="small" :type="isPolicyStatsDrillUnlocked('path') ? 'success' : 'default'" :bordered="false">
                    {{ policyStatsDrillStatusLabel('path') }}
                  </n-tag>
                </div>
                <n-data-table
                  :columns="policyStatsDimensionColumns"
                  :data="policyStatsTopPaths"
                  :loading="policyStatsLoading"
                  :pagination="false"
                  :row-props="buildPolicyStatsDimensionRowProps('path')"
                  :row-key="row => `path-${row.key}`"
                  :max-height="260"
                  class="min-h-120px"
                />
              </n-card>
            </n-gi>
            <n-gi>
              <n-card :bordered="false" size="small">
                <div class="text-sm font-semibold mb-2 flex items-center gap-2">
                  <span>Top Method</span>
                  <n-tooltip trigger="hover">
                    <template #trigger>
                      <span class="inline-flex items-center" :class="isPolicyStatsDrillUnlocked('method') ? 'text-green-600' : 'text-gray-400'">
                        <icon-carbon-unlocked v-if="isPolicyStatsDrillUnlocked('method')" />
                        <icon-carbon-locked v-else />
                      </span>
                    </template>
                    {{ policyStatsDrillHint('method') }}
                  </n-tooltip>
                  <n-tag size="small" :type="isPolicyStatsDrillUnlocked('method') ? 'success' : 'default'" :bordered="false">
                    {{ policyStatsDrillStatusLabel('method') }}
                  </n-tag>
                </div>
                <n-data-table
                  :columns="policyStatsDimensionColumns"
                  :data="policyStatsTopMethods"
                  :loading="policyStatsLoading"
                  :pagination="false"
                  :row-props="buildPolicyStatsDimensionRowProps('method')"
                  :row-key="row => `method-${row.key}`"
                  :max-height="260"
                  class="min-h-120px"
                />
              </n-card>
            </n-gi>
          </n-grid>
        </n-tab-pane>

        <n-tab-pane name="release" tab="版本发布管理">
          <div class="mb-3 flex flex-wrap gap-2 items-center">
            <n-select v-model:value="releaseQuery.status" :options="releaseStatusOptions" clearable placeholder="状态" class="w-160px" />
            <n-button type="primary" @click="fetchReleases">
              <template #icon>
                <icon-carbon-search />
              </template>
              查询
            </n-button>
            <n-button @click="resetReleaseQuery">重置</n-button>
            <n-button type="warning" @click="openRollbackModal">回滚到历史版本</n-button>
            <n-button type="error" @click="handleClearReleases">清空非激活版本</n-button>
          </div>

          <n-data-table
            remote
            :columns="releaseColumns"
            :data="releaseTable"
            :loading="releaseLoading"
            :pagination="releasePagination"
            :row-key="row => row.id"
            :max-height="tableFixedHeight"
            class="min-h-260px"
            @update:page="handleReleasePageChange"
            @update:page-size="handleReleasePageSizeChange"
          />
        </n-tab-pane>

        <n-tab-pane name="job" tab="任务日志">
          <div class="mb-3 flex flex-wrap gap-2 items-center">
            <n-select v-model:value="jobQuery.status" :options="jobStatusOptions" clearable placeholder="状态" class="w-160px" />
            <n-select v-model:value="jobQuery.action" :options="jobActionOptions" clearable placeholder="动作" class="w-160px" />
            <n-button type="primary" @click="fetchJobs">
              <template #icon>
                <icon-carbon-search />
              </template>
              查询
            </n-button>
            <n-button @click="resetJobQuery">重置</n-button>
            <n-button type="success" @click="refreshCurrentTab">刷新</n-button>
            <n-button type="error" @click="handleClearJobs">清空任务日志</n-button>
          </div>

          <n-data-table
            remote
            :columns="jobColumns"
            :data="jobTable"
            :loading="jobLoading"
            :pagination="jobPagination"
            :row-key="row => row.id"
            :max-height="tableFixedHeight"
            class="min-h-260px"
            :scroll-x="1500"
            :resizable="true"
            @update:page="handleJobPageChange"
            @update:page-size="handleJobPageSizeChange"
          />
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <n-modal v-model:show="sourceModalVisible" preset="card" :title="sourceModalTitle" class="w-720px">
      <n-form ref="sourceFormRef" :model="sourceForm" :rules="sourceRules" label-placement="left" label-width="120">
        <n-grid cols="2" x-gap="12">
          <n-form-item-gi label="名称" path="name">
            <n-input v-model:value="sourceForm.name" placeholder="例如：official-crs" />
          </n-form-item-gi>
          <n-form-item-gi label="类型" path="kind">
            <n-input value="crs" disabled />
          </n-form-item-gi>
          <n-form-item-gi label="模式" path="mode">
            <n-select v-model:value="sourceForm.mode" :options="modeOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="鉴权类型" path="authType">
            <n-select v-model:value="sourceForm.authType" :options="authTypeOptions" />
          </n-form-item-gi>
        </n-grid>

        <n-form-item label="默认源">
          <div class="flex flex-wrap gap-2">
            <n-button size="small" secondary @click="applyDefaultSource">应用 CRS 默认源</n-button>
          </div>
        </n-form-item>

        <n-form-item label="源地址" path="url" v-if="sourceForm.mode === 'remote'">
          <n-input v-model:value="sourceForm.url" placeholder="https://api.github.com/repos/coreruleset/coreruleset/releases/latest" />
        </n-form-item>

        <n-form-item label="校验地址" path="checksumUrl" v-if="sourceForm.mode === 'remote'">
          <n-input v-model:value="sourceForm.checksumUrl" placeholder="可选，SHA256 清单地址" />
        </n-form-item>

        <n-form-item label="代理地址" path="proxyUrl" v-if="sourceForm.mode === 'remote'">
          <n-input v-model:value="sourceForm.proxyUrl" placeholder="可选，例如：http://127.0.0.1:7890" />
        </n-form-item>

        <n-form-item label="鉴权密钥" path="authSecret" v-if="sourceForm.authType !== 'none'">
          <n-input v-model:value="sourceForm.authSecret" type="password" show-password-on="mousedown" placeholder="Token 或 user:password" />
        </n-form-item>

        <n-form-item label="调度表达式" path="schedule">
          <n-input v-model:value="sourceForm.schedule" placeholder="例如：0 0 */6 * * *" />
        </n-form-item>

        <n-form-item label="附加元数据" path="meta">
          <n-input v-model:value="sourceForm.meta" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" placeholder="JSON 字符串，可选" />
        </n-form-item>

        <n-grid cols="2" x-gap="12">
          <n-form-item-gi label="启用">
            <n-switch v-model:value="sourceForm.enabled" />
          </n-form-item-gi>
          <n-form-item-gi label="自动检查">
            <n-switch v-model:value="sourceForm.autoCheck" />
          </n-form-item-gi>
          <n-form-item-gi label="自动下载">
            <n-switch v-model:value="sourceForm.autoDownload" />
          </n-form-item-gi>
          <n-form-item-gi label="自动激活">
            <n-switch v-model:value="sourceForm.autoActivate" />
          </n-form-item-gi>
        </n-grid>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="sourceModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="sourceSubmitting" @click="handleSubmitSource">保存</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="policyModalVisible" preset="card" :title="policyModalTitle" class="w-760px">
      <n-form ref="policyFormRef" :model="policyForm" :rules="policyRules" label-placement="left" label-width="150">
        <n-grid cols="2" x-gap="12">
          <n-form-item-gi label="策略名称" path="name">
            <n-input v-model:value="policyForm.name" placeholder="例如：default-runtime-policy" />
          </n-form-item-gi>
          <n-form-item-gi label="是否默认策略">
            <n-switch v-model:value="policyForm.isDefault" />
          </n-form-item-gi>
          <n-form-item-gi label="引擎模式" path="engineMode">
            <n-select v-model:value="policyForm.engineMode" :options="policyEngineModeOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="审计模式" path="auditEngine">
            <n-select v-model:value="policyForm.auditEngine" :options="policyAuditEngineOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="审计日志格式" path="auditLogFormat">
            <n-select v-model:value="policyForm.auditLogFormat" :options="policyAuditLogFormatOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="请求体访问">
            <n-switch v-model:value="policyForm.requestBodyAccess" />
          </n-form-item-gi>
          <n-form-item-gi label="启用策略">
            <n-switch v-model:value="policyForm.enabled" />
          </n-form-item-gi>
        </n-grid>

        <n-form-item label="描述" path="description">
          <n-input v-model:value="policyForm.description" placeholder="可选，记录策略用途与变更说明" />
        </n-form-item>

        <n-form-item label="审计状态匹配" path="auditRelevantStatus">
          <n-input v-model:value="policyForm.auditRelevantStatus" placeholder="例如：^(?:5|4(?!04))" />
        </n-form-item>

        <n-grid cols="2" x-gap="12">
          <n-form-item-gi label="请求体限制（字节）" path="requestBodyLimit">
            <n-input-number v-model:value="policyForm.requestBodyLimit" :show-button="false" :min="1" :max="1024 * 1024 * 1024" class="w-full" />
          </n-form-item-gi>
          <n-form-item-gi label="无文件请求体限制（字节）" path="requestBodyNoFilesLimit">
            <n-input-number
              v-model:value="policyForm.requestBodyNoFilesLimit"
              :show-button="false"
              :min="1"
              :max="1024 * 1024 * 1024"
              class="w-full"
            />
          </n-form-item-gi>
        </n-grid>

        <n-form-item label="扩展配置(JSON)" path="config">
          <n-input
            v-model:value="policyForm.config"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 6 }"
            placeholder='可选，例如：{"custom_tag":"runtime"}'
          />
        </n-form-item>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="policyModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="policySubmitting" @click="handleSubmitPolicy">保存</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="uploadModalVisible" preset="card" title="上传规则包" class="w-640px">
      <n-form ref="uploadFormRef" :model="uploadForm" :rules="uploadRules" label-placement="left" label-width="110">
        <n-form-item label="类型" path="kind">
          <n-input value="crs" disabled />
        </n-form-item>
        <n-form-item label="版本号" path="version">
          <n-input v-model:value="uploadForm.version" placeholder="例如：v4.23.0-custom.1" />
        </n-form-item>
        <n-form-item label="SHA256" path="checksum">
          <n-input v-model:value="uploadForm.checksum" placeholder="可选，建议填写" />
        </n-form-item>
        <n-form-item label="立即激活" path="activateNow">
          <n-switch v-model:value="uploadForm.activateNow" />
        </n-form-item>
        <n-form-item label="规则包" path="file">
          <n-upload
            :default-upload="false"
            :max="1"
            :show-file-list="true"
            accept=".zip,.tar.gz"
            @before-upload="handleBeforeUpload"
            @remove="handleRemoveUpload"
          >
            <n-button>选择文件</n-button>
          </n-upload>
        </n-form-item>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="uploadModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="uploadSubmitting" @click="handleSubmitUpload">上传并入库</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="exclusionModalVisible" preset="card" :title="exclusionModalTitle" class="w-760px">
      <n-form ref="exclusionFormRef" :model="exclusionForm" :rules="exclusionRules" label-placement="left" label-width="140">
        <n-grid cols="2" x-gap="12">
          <n-form-item-gi label="规则名称" path="name">
            <n-input v-model:value="exclusionForm.name" placeholder="例如：ignore-login-fp" />
          </n-form-item-gi>
          <n-form-item-gi label="关联策略" path="policyId">
            <n-select v-model:value="exclusionForm.policyId" :options="crsPolicyOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="作用域" path="scopeType">
            <n-select v-model:value="exclusionForm.scopeType" :options="scopeTypeOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="移除类型" path="removeType">
            <n-select v-model:value="exclusionForm.removeType" :options="removeTypeOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="Host" path="host" v-if="exclusionForm.scopeType !== 'global'">
            <n-input v-model:value="exclusionForm.host" placeholder="例如：app.example.com" />
          </n-form-item-gi>
          <n-form-item-gi label="Path" path="path" v-if="exclusionForm.scopeType === 'route'">
            <n-input v-model:value="exclusionForm.path" placeholder="例如：/api/login" />
          </n-form-item-gi>
          <n-form-item-gi label="Method" path="method" v-if="exclusionForm.scopeType === 'route'">
            <n-select v-model:value="exclusionForm.method" :options="methodOptions" clearable placeholder="可选" />
          </n-form-item-gi>
          <n-form-item-gi label="是否启用">
            <n-switch v-model:value="exclusionForm.enabled" />
          </n-form-item-gi>
        </n-grid>

        <n-form-item label="移除值" path="removeValue">
          <n-input
            ref="exclusionRemoveValueInputRef"
            v-model:value="exclusionForm.removeValue"
            :placeholder="exclusionForm.removeType === 'id' ? '例如：920350' : '例如：attack-sqli'"
          />
        </n-form-item>
        <n-form-item label="描述" path="description">
          <n-input v-model:value="exclusionForm.description" placeholder="可选，记录误报场景与原因" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="exclusionModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="exclusionSubmitting" @click="handleSubmitExclusion">保存</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="bindingModalVisible" preset="card" :title="bindingModalTitle" class="w-760px">
      <n-form ref="bindingFormRef" :model="bindingForm" :rules="bindingRules" label-placement="left" label-width="140">
        <n-grid cols="2" x-gap="12">
          <n-form-item-gi label="绑定名称" path="name">
            <n-input v-model:value="bindingForm.name" placeholder="例如：site-main-binding" />
          </n-form-item-gi>
          <n-form-item-gi label="关联策略" path="policyId">
            <n-select v-model:value="bindingForm.policyId" :options="crsPolicyOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="作用域" path="scopeType">
            <n-select v-model:value="bindingForm.scopeType" :options="scopeTypeOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="优先级" path="priority">
            <n-input-number v-model:value="bindingForm.priority" :show-button="false" :min="1" :max="1000" class="w-full" />
          </n-form-item-gi>
          <n-form-item-gi label="Host" path="host" v-if="bindingForm.scopeType !== 'global'">
            <n-input v-model:value="bindingForm.host" placeholder="例如：app.example.com" />
          </n-form-item-gi>
          <n-form-item-gi label="Path" path="path" v-if="bindingForm.scopeType === 'route'">
            <n-input v-model:value="bindingForm.path" placeholder="例如：/api" />
          </n-form-item-gi>
          <n-form-item-gi label="Method" path="method" v-if="bindingForm.scopeType === 'route'">
            <n-select v-model:value="bindingForm.method" :options="methodOptions" clearable placeholder="可选" />
          </n-form-item-gi>
          <n-form-item-gi label="是否启用">
            <n-switch v-model:value="bindingForm.enabled" />
          </n-form-item-gi>
        </n-grid>

        <n-form-item label="描述" path="description">
          <n-input v-model:value="bindingForm.description" placeholder="可选，记录生效范围和意图" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="bindingModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="bindingSubmitting" @click="handleSubmitBinding">保存</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="policyFeedbackModalVisible" preset="card" title="标记误报反馈" class="w-760px">
      <n-form ref="policyFeedbackFormRef" :model="policyFeedbackForm" :rules="policyFeedbackRules" label-placement="left" label-width="130">
        <n-grid cols="2" x-gap="12">
          <n-form-item-gi label="关联策略" path="policyId">
            <n-select v-model:value="policyFeedbackForm.policyId" :options="crsPolicyOptions" clearable placeholder="可选，不填表示全部策略" />
          </n-form-item-gi>
          <n-form-item-gi label="状态码" path="status">
            <n-input-number v-model:value="policyFeedbackForm.status" :show-button="false" :min="100" :max="599" class="w-full" />
          </n-form-item-gi>
          <n-form-item-gi label="责任人" path="assignee">
            <n-input v-model:value="policyFeedbackForm.assignee" placeholder="可选，例如 alice" />
          </n-form-item-gi>
          <n-form-item-gi label="截止时间" path="dueAt">
            <n-input v-model:value="policyFeedbackForm.dueAt" placeholder="可选，YYYY-MM-DD HH:mm:ss" />
          </n-form-item-gi>
          <n-form-item-gi label="Host" path="host">
            <n-input v-model:value="policyFeedbackForm.host" placeholder="可选，例如 app.example.com" />
          </n-form-item-gi>
          <n-form-item-gi label="Path" path="path">
            <n-input v-model:value="policyFeedbackForm.path" placeholder="可选，例如 /api/login" />
          </n-form-item-gi>
          <n-form-item-gi label="Method" path="method">
            <n-select v-model:value="policyFeedbackForm.method" :options="methodOptions" clearable placeholder="可选" />
          </n-form-item-gi>
          <n-form-item-gi label="示例 URI" path="sampleUri">
            <n-input v-model:value="policyFeedbackForm.sampleUri" placeholder="可选，记录原始 URI 便于复盘" />
          </n-form-item-gi>
        </n-grid>
        <n-form-item label="误报原因" path="reason">
          <n-input v-model:value="policyFeedbackForm.reason" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="必填：为何判断为误报" />
        </n-form-item>
        <n-form-item label="建议动作" path="suggestion">
          <n-input
            v-model:value="policyFeedbackForm.suggestion"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            placeholder="可选：例如建议添加 removeById、放宽阈值或补白名单"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="policyFeedbackModalVisible = false">取消</n-button>
          <n-button type="warning" :loading="policyFeedbackSubmitting" @click="handleSubmitPolicyFeedback">提交反馈</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="policyFeedbackProcessModalVisible" preset="card" title="处理误报反馈" class="w-640px">
      <n-form ref="policyFeedbackProcessFormRef" :model="policyFeedbackProcessForm" :rules="policyFeedbackProcessRules" label-placement="left" label-width="120">
        <n-form-item label="处理状态" path="feedbackStatus">
          <n-select v-model:value="policyFeedbackProcessForm.feedbackStatus" :options="policyFeedbackStatusOptions" />
        </n-form-item>
        <n-form-item label="责任人" path="assignee">
          <n-input v-model:value="policyFeedbackProcessForm.assignee" placeholder="可选，例如 alice" />
        </n-form-item>
        <n-form-item label="截止时间" path="dueAt">
          <n-input v-model:value="policyFeedbackProcessForm.dueAt" placeholder="可选，YYYY-MM-DD HH:mm:ss" />
        </n-form-item>
        <n-form-item label="处理备注" path="processNote">
          <n-input
            v-model:value="policyFeedbackProcessForm.processNote"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            placeholder="可选，记录确认依据或处理结果"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="policyFeedbackProcessModalVisible = false">取消</n-button>
          <n-button type="warning" :loading="policyFeedbackProcessSubmitting" @click="handleSubmitPolicyFeedbackProcess">保存状态</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="policyFeedbackBatchProcessModalVisible" preset="card" title="批量处理误报反馈" class="w-640px">
      <div class="mb-3 text-sm text-gray-600">已选择 {{ policyFeedbackCheckedRowKeys.length }} 条反馈记录</div>
      <n-form ref="policyFeedbackBatchProcessFormRef" :model="policyFeedbackBatchProcessForm" :rules="policyFeedbackProcessRules" label-placement="left" label-width="120">
        <n-form-item label="处理状态" path="feedbackStatus">
          <n-select v-model:value="policyFeedbackBatchProcessForm.feedbackStatus" :options="policyFeedbackStatusOptions" />
        </n-form-item>
        <n-form-item label="责任人" path="assignee">
          <n-input v-model:value="policyFeedbackBatchProcessForm.assignee" placeholder="可选，例如 alice" />
        </n-form-item>
        <n-form-item label="截止时间" path="dueAt">
          <n-input v-model:value="policyFeedbackBatchProcessForm.dueAt" placeholder="可选，YYYY-MM-DD HH:mm:ss" />
        </n-form-item>
        <n-form-item label="处理备注" path="processNote">
          <n-input
            v-model:value="policyFeedbackBatchProcessForm.processNote"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            placeholder="可选，批量处理说明"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="policyFeedbackBatchProcessModalVisible = false">取消</n-button>
          <n-button type="warning" :loading="policyFeedbackBatchProcessSubmitting" @click="handleSubmitPolicyFeedbackBatchProcess">批量保存</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="policyFeedbackExclusionDraftModalVisible" preset="card" title="确认生成例外草稿" class="w-760px">
      <div v-if="policyFeedbackExclusionDraft" class="space-y-3">
        <div class="text-sm text-gray-600">来源反馈 #{{ policyFeedbackExclusionDraft.feedbackId }}</div>
        <n-form :model="policyFeedbackExclusionDraft" label-placement="left" label-width="120">
          <n-grid cols="2" x-gap="12">
            <n-form-item-gi label="关联策略">
              <n-select v-model:value="policyFeedbackExclusionDraft.policyId" :options="crsPolicyOptions" />
            </n-form-item-gi>
            <n-form-item-gi label="作用域">
              <n-select
                v-model:value="policyFeedbackExclusionDraft.scopeType"
                :options="scopeTypeOptions"
                @update:value="handlePolicyFeedbackExclusionDraftScopeChange"
              />
            </n-form-item-gi>
            <n-form-item-gi label="Host" v-if="policyFeedbackExclusionDraft.scopeType !== 'global'">
              <n-input v-model:value="policyFeedbackExclusionDraft.host" placeholder="例如：app.example.com" />
            </n-form-item-gi>
            <n-form-item-gi label="Path" v-if="policyFeedbackExclusionDraft.scopeType === 'route'">
              <n-input v-model:value="policyFeedbackExclusionDraft.path" placeholder="例如：/api/login" />
            </n-form-item-gi>
            <n-form-item-gi label="Method" v-if="policyFeedbackExclusionDraft.scopeType === 'route'">
              <n-select v-model:value="policyFeedbackExclusionDraft.method" :options="methodOptions" clearable placeholder="可选" />
            </n-form-item-gi>
            <n-form-item-gi label="移除类型">
              <n-select v-model:value="policyFeedbackExclusionDraft.removeType" :options="removeTypeOptions" />
            </n-form-item-gi>
          </n-grid>
          <n-form-item label="规则名称">
            <n-input v-model:value="policyFeedbackExclusionDraft.name" />
          </n-form-item>
        </n-form>
        <div v-if="policyFeedbackExclusionCandidateOptions.length > 1">
          <div class="text-xs text-gray-500 mb-1">候选移除值（建议文本匹配到多个候选）</div>
          <n-select
            v-model:value="policyFeedbackExclusionDraftCandidateKey"
            :options="policyFeedbackExclusionCandidateOptions"
            placeholder="请选择 remove 值候选"
            @update:value="handlePolicyFeedbackExclusionCandidateChange"
          />
        </div>
        <div>
          <div class="text-xs text-gray-500">移除值</div>
          <n-input v-model:value="policyFeedbackExclusionDraft.removeValue" :placeholder="policyFeedbackExclusionDraft.removeType === 'id' ? '例如：920350' : '例如：attack-sqli'" />
        </div>
        <div>
          <div class="text-xs text-gray-500">描述草稿</div>
          <n-input v-model:value="policyFeedbackExclusionDraft.description" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" />
        </div>
        <n-alert v-if="!policyFeedbackExclusionDraft.removeValue" type="warning" :show-icon="true">
          建议文本未解析到可用的 remove 值，请在下一步表单中补充后再保存。
        </n-alert>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="policyFeedbackExclusionDraftModalVisible = false">取消</n-button>
          <n-button type="primary" @click="handleConfirmPolicyFeedbackExclusionDraft">确认生成</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="rollbackModalVisible" preset="card" title="回滚版本" class="w-520px">
      <n-form ref="rollbackFormRef" :model="rollbackForm" :rules="rollbackRules" label-placement="left" label-width="110">
        <n-form-item label="回滚目标" path="target">
          <n-radio-group v-model:value="rollbackForm.target">
            <n-space>
              <n-radio value="last_good">last_good</n-radio>
              <n-radio value="version">指定版本</n-radio>
            </n-space>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="版本号" path="version" v-if="rollbackForm.target === 'version'">
          <n-input v-model:value="rollbackForm.version" placeholder="例如：v4.23.0" />
        </n-form-item>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="rollbackModalVisible = false">取消</n-button>
          <n-button type="warning" :loading="rollbackSubmitting" @click="handleSubmitRollback">确认回滚</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onMounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
  NButton,
  NPopconfirm,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  useDialog,
  useMessage,
  type DataTableColumns,
  type FormInst,
  type FormRules,
  type InputInst,
  type PaginationProps,
  type UploadFileInfo
} from 'naive-ui';
import {
  activateWafRelease,
  checkWafEngine,
  clearWafJobs,
  clearWafReleases,
  batchUpdateWafPolicyFalsePositiveFeedbackStatus,
  createWafPolicyFalsePositiveFeedback,
  createWafPolicyBinding,
  createWafPolicy,
  createWafRuleExclusion,
  createWafSource,
  deleteWafPolicyBinding,
  deleteWafPolicy,
  deleteWafRuleExclusion,
  deleteWafSource,
  fetchWafEngineStatus,
  fetchWafJobList,
  fetchWafPolicyFalsePositiveFeedbackList,
  fetchWafPolicyBindingList,
  fetchWafPolicyList,
  fetchWafPolicyStats,
  fetchWafPolicyRevisionList,
  fetchWafRuleExclusionList,
  fetchWafReleaseList,
  fetchWafSourceList,
  previewWafPolicy,
  publishWafPolicy,
  rollbackWafPolicy,
  rollbackWafRelease,
  syncWafSource,
  updateWafPolicyFalsePositiveFeedbackStatus,
  updateWafPolicyBinding,
  updateWafPolicy,
  updateWafRuleExclusion,
  updateWafSource,
  uploadWafPackage,
  validateWafPolicy,
  type WafAuthType,
  type WafPolicyAuditEngine,
  type WafPolicyAuditLogFormat,
  type WafPolicyCrsTemplate,
  type WafPolicyEngineMode,
  type WafPolicyItem,
  type WafPolicyBindingItem,
  type WafPolicyBindingPayload,
  type WafPolicyFalsePositiveFeedbackItem,
  type WafPolicyFalsePositiveFeedbackBatchStatusUpdatePayload,
  type WafPolicyFalsePositiveFeedbackPayload,
  type WafPolicyFalsePositiveFeedbackStatusUpdatePayload,
  type WafPolicyRemoveType,
  type WafPolicyStatsDimensionItem,
  type WafPolicyStatsItem,
  type WafPolicyStatsTrendItem,
  type WafPolicyRevisionItem,
  type WafPolicyRevisionStatus,
  type WafPolicyScopeType,
  type WafRuleExclusionItem,
  type WafRuleExclusionPayload,
  type WafJobItem,
  type WafJobStatus,
  type WafKind,
  type WafMode,
  type WafEngineStatusResp,
  type WafReleaseItem,
  type WafReleaseStatus,
  type WafSourceItem
} from '@/service/api/caddy';
import {
  buildExclusionCandidateKey,
  collectExclusionCandidatesFromFeedbackSuggestion,
  mergePolicyFeedbackCheckedRowKeys,
  parseExclusionCandidateKey,
  parseExclusionFromFeedbackSuggestion
} from './policy-feedback-draft';

const message = useMessage();
const dialog = useDialog();
const route = useRoute();
const router = useRouter();

const engineLoading = ref(false);
const engineChecking = ref(false);
const engineUnavailable = ref(false);
const engineStatus = ref<WafEngineStatusResp | null>(null);

const activeTab = ref<'source' | 'runtime' | 'crs' | 'exclusion' | 'binding' | 'observe' | 'release' | 'job'>('source');
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

const policyFeedbackStatusFilterOptions = [
  { label: '全部状态', value: '' },
  ...policyFeedbackStatusOptions
];

const policyFeedbackSLAStatusOptions = [
  { label: '全部SLA', value: 'all' as const },
  { label: '正常', value: 'normal' as const },
  { label: '已超时', value: 'overdue' as const },
  { label: '已解决', value: 'resolved' as const }
];

const crsTemplateOptions = [
  { label: '低误报（PL1 / In 10 / Out 8）', value: 'low_fp' as WafPolicyCrsTemplate },
  { label: '平衡（PL2 / In 5 / Out 4）', value: 'balanced' as WafPolicyCrsTemplate },
  { label: '高拦截（PL3 / In 3 / Out 2）', value: 'high_blocking' as WafPolicyCrsTemplate },
  { label: '自定义', value: 'custom' as WafPolicyCrsTemplate }
];

const crsTemplatePresetMap: Record<
  Exclude<WafPolicyCrsTemplate, 'custom'>,
  { crsParanoiaLevel: number; crsInboundAnomalyThreshold: number; crsOutboundAnomalyThreshold: number }
> = {
  low_fp: { crsParanoiaLevel: 1, crsInboundAnomalyThreshold: 10, crsOutboundAnomalyThreshold: 8 },
  balanced: { crsParanoiaLevel: 2, crsInboundAnomalyThreshold: 5, crsOutboundAnomalyThreshold: 4 },
  high_blocking: { crsParanoiaLevel: 3, crsInboundAnomalyThreshold: 3, crsOutboundAnomalyThreshold: 2 }
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

const sourceQuery = reactive({
  name: ''
});

const sourceLoading = ref(false);
const sourceTable = ref<WafSourceItem[]>([]);
const jobSourceNameMap = ref<Record<number, string>>({});
const sourcePagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

const sourceModalVisible = ref(false);
const sourceModalMode = ref<'add' | 'edit'>('add');
const sourceSubmitting = ref(false);
const sourceFormRef = ref<FormInst | null>(null);
const sourceForm = reactive({
  id: 0,
  name: '',
  kind: 'crs' as WafKind,
  mode: 'remote' as WafMode,
  url: '',
  checksumUrl: '',
  proxyUrl: '',
  authType: 'none' as WafAuthType,
  authSecret: '',
  schedule: '',
  enabled: true,
  autoCheck: true,
  autoDownload: true,
  autoActivate: false,
  meta: ''
});

const sourceModalTitle = computed(() => (sourceModalMode.value === 'add' ? '新增更新源' : '编辑更新源'));
const pageTitle = computed(() => '安全升级管理');

const sourceRules: FormRules = {
  name: { required: true, message: '请输入源名称', trigger: 'blur' },
  kind: { required: true, message: '请选择类型', trigger: 'change' },
  mode: { required: true, message: '请选择模式', trigger: 'change' },
  authType: { required: true, message: '请选择鉴权类型', trigger: 'change' },
  url: {
    validator(_rule, value: string) {
      if (sourceForm.mode !== 'remote') return true;
      if (!value?.trim()) return new Error('remote 模式必须填写源地址');
      return true;
    },
    trigger: ['blur', 'input']
  },
  meta: {
    validator(_rule, value: string) {
      const raw = value?.trim();
      if (!raw) return true;
      try {
        JSON.parse(raw);
        return true;
      } catch {
        return new Error('meta 必须是合法 JSON');
      }
    },
    trigger: 'blur'
  }
};

const policyQuery = reactive({
  name: ''
});

const policyLoading = ref(false);
const policyTable = ref<WafPolicyItem[]>([]);
const policyPagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

const policyModalVisible = ref(false);
const policyModalMode = ref<'add' | 'edit'>('add');
const policySubmitting = ref(false);
const policyFormRef = ref<FormInst | null>(null);
const policyForm = reactive({
  id: 0,
  name: '',
  description: '',
  enabled: true,
  isDefault: false,
  engineMode: 'detectiononly' as WafPolicyEngineMode,
  auditEngine: 'relevantonly' as WafPolicyAuditEngine,
  auditLogFormat: 'json' as WafPolicyAuditLogFormat,
  auditRelevantStatus: '^(?:5|4(?!04))',
  requestBodyAccess: true,
  requestBodyLimit: 10 * 1024 * 1024,
  requestBodyNoFilesLimit: 1024 * 1024,
  config: ''
});

const policyModalTitle = computed(() => (policyModalMode.value === 'add' ? '新增运行策略' : '编辑运行策略'));
const policyPreviewLoading = ref(false);
const policyPreviewPolicyName = ref('');
const policyPreviewDirectives = ref('');
const crsTuningSubmitting = ref(false);
const crsTuningFormRef = ref<FormInst | null>(null);
const crsTuningForm = reactive({
  policyId: 0,
  crsTemplate: 'low_fp' as WafPolicyCrsTemplate,
  crsParanoiaLevel: 1,
  crsInboundAnomalyThreshold: 10,
  crsOutboundAnomalyThreshold: 8
});
const crsPolicyOptions = computed(() =>
  policyTable.value.map(item => ({
    label: `${item.name}${item.isDefault ? '（默认）' : ''}`,
    value: item.id
  }))
);

const policyRevisionLoading = ref(false);
const policyRevisionTable = ref<WafPolicyRevisionItem[]>([]);
const policyRevisionPagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50]
});

const observeWindowOptions = [
  { label: '最近 1 小时', value: '1h' },
  { label: '最近 6 小时', value: '6h' },
  { label: '最近 24 小时', value: '24h' },
  { label: '最近 7 天', value: '7d' }
];

const policyStatsQuery = reactive({
  policyId: '' as number | '' | null,
  window: '24h' as '1h' | '6h' | '24h' | '7d',
  intervalSec: 300,
  topN: 8,
  host: '',
  path: '',
  method: ''
});

const policyStatsLoading = ref(false);
const policyStatsSummary = ref<WafPolicyStatsItem>({
  policyId: 0,
  policyName: '全部策略',
  hitCount: 0,
  blockedCount: 0,
  allowedCount: 0,
  suspectedFalsePositiveCount: 0,
  blockRate: 0
});
const policyStatsTable = ref<WafPolicyStatsItem[]>([]);
const policyStatsTrend = ref<WafPolicyStatsTrendItem[]>([]);
const policyStatsTopHosts = ref<WafPolicyStatsDimensionItem[]>([]);
const policyStatsTopPaths = ref<WafPolicyStatsDimensionItem[]>([]);
const policyStatsTopMethods = ref<WafPolicyStatsDimensionItem[]>([]);
const policyStatsRange = ref({ startTime: '', endTime: '', intervalSec: 300 });
type PolicyStatsSnapshot = {
  capturedAt: string;
  query: {
    policyId: number | '' | null;
    window: '1h' | '6h' | '24h' | '7d';
    intervalSec: number;
    topN: number;
    host: string;
    path: string;
    method: string;
  };
  range: {
    startTime: string;
    endTime: string;
    intervalSec: number;
  };
  summary: WafPolicyStatsItem;
  list: WafPolicyStatsItem[];
  trend: WafPolicyStatsTrendItem[];
  topHosts: WafPolicyStatsDimensionItem[];
  topPaths: WafPolicyStatsDimensionItem[];
  topMethods: WafPolicyStatsDimensionItem[];
};
const policyStatsPreviousSnapshot = ref<PolicyStatsSnapshot | null>(null);
const policyFeedbackLoading = ref(false);
const policyFeedbackTable = ref<WafPolicyFalsePositiveFeedbackItem[]>([]);
const policyFeedbackCheckedRowKeys = ref<number[]>([]);
const policyFeedbackPagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50]
});
const policyFeedbackStatusFilter = ref<'' | 'pending' | 'confirmed' | 'resolved'>('');
const policyFeedbackAssigneeFilter = ref('');
const policyFeedbackSLAStatusFilter = ref<'all' | 'normal' | 'overdue' | 'resolved'>('all');
const policyFeedbackModalVisible = ref(false);
const policyFeedbackSubmitting = ref(false);
const policyFeedbackFormRef = ref<FormInst | null>(null);
const policyFeedbackForm = reactive({
  policyId: null as number | null,
  host: '',
  path: '',
  method: '' as string | null,
  status: 403,
  assignee: '',
  dueAt: '',
  sampleUri: '',
  reason: '',
  suggestion: ''
});
const policyFeedbackProcessModalVisible = ref(false);
const policyFeedbackProcessSubmitting = ref(false);
const policyFeedbackProcessFormRef = ref<FormInst | null>(null);
const policyFeedbackProcessForm = reactive({
  id: 0,
  feedbackStatus: 'confirmed' as 'pending' | 'confirmed' | 'resolved',
  processNote: '',
  assignee: '',
  dueAt: ''
});
const policyFeedbackBatchProcessModalVisible = ref(false);
const policyFeedbackBatchProcessSubmitting = ref(false);
const policyFeedbackBatchProcessFormRef = ref<FormInst | null>(null);
const policyFeedbackBatchProcessForm = reactive({
  feedbackStatus: 'confirmed' as 'pending' | 'confirmed' | 'resolved',
  processNote: '',
  assignee: '',
  dueAt: ''
});
type PolicyFeedbackExclusionDraft = {
  feedbackId: number;
  policyId: number;
  policyName: string;
  name: string;
  description: string;
  scopeType: WafPolicyScopeType;
  host: string;
  path: string;
  method: string;
  removeType: WafPolicyRemoveType;
  removeValue: string;
  candidates: Array<{
    removeType: WafPolicyRemoveType;
    removeValue: string;
  }>;
};
const policyFeedbackExclusionDraftModalVisible = ref(false);
const policyFeedbackExclusionDraft = ref<PolicyFeedbackExclusionDraft | null>(null);
const policyFeedbackExclusionDraftCandidateKey = ref('');
const observeWindowValueSet = new Set(observeWindowOptions.map(item => item.value));
const observeRouteSyncing = ref(false);

const policyStatsPolicyOptions = computed(() => [
  { label: '全部策略', value: '' },
  ...crsPolicyOptions.value
]);

const hasPolicyStatsDrillFilters = computed(
  () => !!(policyStatsQuery.host.trim() || policyStatsQuery.path.trim() || policyStatsQuery.method.trim())
);
const hasPolicyFeedbackSelection = computed(() => policyFeedbackCheckedRowKeys.value.length > 0);
const policyFeedbackCheckedRowKeysInPage = computed(() => {
  const selectedKeySet = new Set(policyFeedbackCheckedRowKeys.value);
  return policyFeedbackTable.value.map(item => Number(item.id || 0)).filter(id => id > 0 && selectedKeySet.has(id));
});
const policyFeedbackExclusionCandidateOptions = computed(() => {
  const candidates = policyFeedbackExclusionDraft.value?.candidates || [];
  return candidates.map(item => ({
    label: `${item.removeType === 'id' ? 'removeById' : 'removeByTag'}: ${item.removeValue}`,
    value: buildExclusionCandidateKey(item.removeType, item.removeValue)
  }));
});

const policyFeedbackRules: FormRules = {
  method: {
    validator(_rule, value: string) {
      const normalized = String(value || '').trim().toUpperCase();
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
  auditLogFormat: { required: true, message: '请选择审计日志格式', trigger: 'change' },
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

const crsTuningRules: FormRules = {
  policyId: {
    validator(_rule, value: number) {
      if (!Number(value)) {
        return new Error('请选择策略');
      }
      return true;
    },
    trigger: 'change'
  },
  crsParanoiaLevel: {
    validator(_rule, value: number) {
      const num = Number(value);
      if (!Number.isFinite(num) || num < 1 || num > 4) {
        return new Error('PL 必须在 1 到 4 之间');
      }
      return true;
    },
    trigger: ['blur', 'change']
  },
  crsInboundAnomalyThreshold: {
    validator(_rule, value: number) {
      const num = Number(value);
      if (!Number.isFinite(num) || num < 1 || num > 20) {
        return new Error('Inbound 阈值必须在 1 到 20 之间');
      }
      return true;
    },
    trigger: ['blur', 'change']
  },
  crsOutboundAnomalyThreshold: {
    validator(_rule, value: number) {
      const num = Number(value);
      if (!Number.isFinite(num) || num < 1 || num > 20) {
        return new Error('Outbound 阈值必须在 1 到 20 之间');
      }
      return true;
    },
    trigger: ['blur', 'change']
  }
};

const exclusionQuery = reactive({
  policyId: null as number | null,
  scopeType: '' as '' | WafPolicyScopeType | null,
  name: ''
});

const exclusionLoading = ref(false);
const exclusionTable = ref<WafRuleExclusionItem[]>([]);
const exclusionPagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

const exclusionModalVisible = ref(false);
const exclusionModalMode = ref<'add' | 'edit'>('add');
const exclusionSubmitting = ref(false);
const exclusionFormRef = ref<FormInst | null>(null);
const exclusionRemoveValueInputRef = ref<InputInst | null>(null);
const shouldFocusExclusionRemoveValue = ref(false);
const exclusionForm = reactive({
  id: 0,
  policyId: 0,
  name: '',
  description: '',
  enabled: true,
  scopeType: 'global' as WafPolicyScopeType,
  host: '',
  path: '',
  method: '' as string | null,
  removeType: 'id' as WafPolicyRemoveType,
  removeValue: ''
});

const exclusionModalTitle = computed(() => (exclusionModalMode.value === 'add' ? '新增规则例外' : '编辑规则例外'));
const exclusionRules: FormRules = {
  policyId: {
    validator(_rule, value: number) {
      if (!Number(value)) return new Error('请选择关联策略');
      return true;
    },
    trigger: 'change'
  },
  scopeType: { required: true, message: '请选择作用域', trigger: 'change' },
  removeType: { required: true, message: '请选择移除类型', trigger: 'change' },
  removeValue: { required: true, message: '请输入移除值', trigger: 'blur' },
  host: {
    validator(_rule, value: string) {
      if (exclusionForm.scopeType === 'site' && !String(value || '').trim()) {
        return new Error('站点作用域必须填写 host');
      }
      return true;
    },
    trigger: ['blur', 'input']
  },
  path: {
    validator(_rule, value: string) {
      if (exclusionForm.scopeType === 'route' && !String(value || '').trim()) {
        return new Error('路由作用域必须填写 path');
      }
      return true;
    },
    trigger: ['blur', 'input']
  }
};

const bindingQuery = reactive({
  policyId: null as number | null,
  scopeType: '' as '' | WafPolicyScopeType | null,
  name: ''
});

const bindingLoading = ref(false);
const bindingTable = ref<WafPolicyBindingItem[]>([]);
const bindingPagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

const bindingModalVisible = ref(false);
const bindingModalMode = ref<'add' | 'edit'>('add');
const bindingSubmitting = ref(false);
const bindingFormRef = ref<FormInst | null>(null);
const bindingForm = reactive({
  id: 0,
  policyId: 0,
  name: '',
  description: '',
  enabled: true,
  scopeType: 'global' as WafPolicyScopeType,
  host: '',
  path: '',
  method: '' as string | null,
  priority: 100
});

const bindingModalTitle = computed(() => (bindingModalMode.value === 'add' ? '新增策略绑定' : '编辑策略绑定'));
const bindingRules: FormRules = {
  policyId: {
    validator(_rule, value: number) {
      if (!Number(value)) return new Error('请选择关联策略');
      return true;
    },
    trigger: 'change'
  },
  scopeType: { required: true, message: '请选择作用域', trigger: 'change' },
  priority: {
    validator(_rule, value: number) {
      const num = Number(value);
      if (!Number.isFinite(num) || num < 1 || num > 1000) {
        return new Error('优先级必须在 1 到 1000 之间');
      }
      return true;
    },
    trigger: ['blur', 'change']
  },
  host: {
    validator(_rule, value: string) {
      if (bindingForm.scopeType === 'site' && !String(value || '').trim()) {
        return new Error('站点作用域必须填写 host');
      }
      return true;
    },
    trigger: ['blur', 'input']
  },
  path: {
    validator(_rule, value: string) {
      if (bindingForm.scopeType === 'route' && !String(value || '').trim()) {
        return new Error('路由作用域必须填写 path');
      }
      return true;
    },
    trigger: ['blur', 'input']
  }
};

const releaseQuery = reactive({
  status: '' as '' | WafReleaseStatus
});

const releaseLoading = ref(false);
const releaseTable = ref<WafReleaseItem[]>([]);
const releasePagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

const jobQuery = reactive({
  status: '' as '' | WafJobStatus,
  action: ''
});

const jobLoading = ref(false);
const jobTable = ref<WafJobItem[]>([]);
const jobPagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

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

const rollbackModalVisible = ref(false);
const rollbackSubmitting = ref(false);
const rollbackFormRef = ref<FormInst | null>(null);
const rollbackForm = reactive({
  target: 'last_good' as 'last_good' | 'version',
  version: ''
});

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

const sourceColumns: DataTableColumns<WafSourceItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '名称', key: 'name', minWidth: 140 },
  {
    title: '类型',
    key: 'kind',
    width: 130,
    render(row) {
      return h(NTag, { type: row.kind === 'crs' ? 'success' : 'warning', bordered: false }, { default: () => row.kind });
    }
  },
  {
    title: '模式',
    key: 'mode',
    width: 110,
    render(row) {
      return h(NTag, { type: row.mode === 'remote' ? 'info' : 'default', bordered: false }, { default: () => row.mode });
    }
  },
  {
    title: '地址',
    key: 'url',
    minWidth: 260,
    ellipsis: { tooltip: true },
    render(row) {
      return row.url || '-';
    }
  },
  {
    title: '代理',
    key: 'proxyUrl',
    minWidth: 180,
    ellipsis: { tooltip: true },
    render(row) {
      return row.proxyUrl || '-';
    }
  },
  { title: '调度', key: 'schedule', width: 160, ellipsis: { tooltip: true }, render: row => row.schedule || '-' },
  {
    title: '开关',
    key: 'switches',
    minWidth: 200,
    render(row) {
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
  { title: '最近版本', key: 'lastRelease', width: 140, render: row => row.lastRelease || '-' },
  {
    title: '最近错误',
    key: 'lastError',
    minWidth: 220,
    ellipsis: { tooltip: true },
    render(row) {
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
    render(row) {
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
            h(
              NButton,
              {
                size: 'small',
                onClick: () => handleEditSource(row)
              },
              { default: () => '编辑' }
            ),
            h(
              NPopconfirm,
              { onPositiveClick: () => handleDeleteSource(row) },
              {
                trigger: () => h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
                default: () => '删除后不可恢复，确认继续吗？'
              }
            )
          ]
        }
      );
    }
  }
];

const policyColumns: DataTableColumns<WafPolicyItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '策略名称', key: 'name', minWidth: 180 },
  {
    title: '默认策略',
    key: 'isDefault',
    width: 110,
    render(row) {
      return h(NTag, { type: row.isDefault ? 'success' : 'default', bordered: false }, { default: () => (row.isDefault ? '是' : '否') });
    }
  },
  {
    title: '启用',
    key: 'enabled',
    width: 100,
    render(row) {
      return h(NTag, { type: row.enabled ? 'success' : 'warning', bordered: false }, { default: () => (row.enabled ? '启用' : '禁用') });
    }
  },
  {
    title: '引擎模式',
    key: 'engineMode',
    width: 170,
    render(row) {
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
    render(row) {
      return mapCrsTemplateLabel(row.crsTemplate);
    }
  },
  { title: 'PL', key: 'crsParanoiaLevel', width: 90 },
  { title: '请求体限制', key: 'requestBodyLimit', width: 150, render: row => formatBytes(row.requestBodyLimit) },
  { title: '更新时间', key: 'updatedAt', width: 180 },
  {
    title: '操作',
    key: 'action',
    width: 380,
    fixed: 'right',
    render(row) {
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
            h(
              NButton,
              {
                size: 'small',
                onClick: () => handleEditPolicy(row)
              },
              { default: () => '编辑' }
            ),
            h(
              NPopconfirm,
              { onPositiveClick: () => handleDeletePolicy(row) },
              {
                trigger: () => h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
                default: () => '删除后不可恢复，确认继续吗？'
              }
            )
          ]
        }
      );
    }
  }
];

const policyRevisionColumns: DataTableColumns<WafPolicyRevisionItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '策略', key: 'policyName', minWidth: 180, render: row => row.policyName || `#${row.policyId}` },
  { title: '策略ID', key: 'policyId', width: 100 },
  { title: '版本', key: 'version', width: 100, render: row => `v${row.version}` },
  {
    title: '状态',
    key: 'status',
    width: 120,
    render(row) {
      return h(
        NTag,
        { type: mapPolicyRevisionStatusType(row.status), bordered: false },
        { default: () => mapPolicyRevisionStatusLabel(row.status) }
      );
    }
  },
  { title: '操作人', key: 'operator', width: 120, render: row => row.operator || '-' },
  { title: '变更摘要', key: 'changeSummary', minWidth: 220, ellipsis: { tooltip: true }, render: row => row.changeSummary || row.message || '-' },
  { title: '描述', key: 'message', minWidth: 160, ellipsis: { tooltip: true }, render: row => row.message || '-' },
  { title: '创建时间', key: 'createdAt', width: 180 },
  {
    title: '操作',
    key: 'action',
    width: 140,
    fixed: 'right',
    render(row) {
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
];

const exclusionColumns: DataTableColumns<WafRuleExclusionItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '策略ID', key: 'policyId', width: 100 },
  { title: '名称', key: 'name', minWidth: 160, render: row => row.name || '-' },
  {
    title: '启用',
    key: 'enabled',
    width: 100,
    render(row) {
      return h(NTag, { type: row.enabled ? 'success' : 'warning', bordered: false }, { default: () => (row.enabled ? '启用' : '禁用') });
    }
  },
  { title: '作用域', key: 'scopeType', width: 100, render: row => mapScopeTypeLabel(row.scopeType) },
  { title: 'Host', key: 'host', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.host || '-' },
  { title: 'Path', key: 'path', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.path || '-' },
  { title: 'Method', key: 'method', width: 100, render: row => row.method || '-' },
  { title: '类型', key: 'removeType', width: 120, render: row => (row.removeType === 'id' ? 'removeById' : 'removeByTag') },
  { title: '移除值', key: 'removeValue', minWidth: 180, ellipsis: { tooltip: true } },
  { title: '更新时间', key: 'updatedAt', width: 180 },
  {
    title: '操作',
    key: 'action',
    width: 180,
    fixed: 'right',
    render(row) {
      return h(
        NSpace,
        { size: 4 },
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                onClick: () => handleEditExclusion(row)
              },
              { default: () => '编辑' }
            ),
            h(
              NPopconfirm,
              { onPositiveClick: () => handleDeleteExclusion(row) },
              {
                trigger: () => h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
                default: () => '删除后不可恢复，确认继续吗？'
              }
            )
          ]
        }
      );
    }
  }
];

const bindingColumns: DataTableColumns<WafPolicyBindingItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '策略ID', key: 'policyId', width: 100 },
  { title: '名称', key: 'name', minWidth: 160, render: row => row.name || '-' },
  {
    title: '启用',
    key: 'enabled',
    width: 100,
    render(row) {
      return h(NTag, { type: row.enabled ? 'success' : 'warning', bordered: false }, { default: () => (row.enabled ? '启用' : '禁用') });
    }
  },
  { title: '作用域', key: 'scopeType', width: 100, render: row => mapScopeTypeLabel(row.scopeType) },
  { title: 'Host', key: 'host', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.host || '-' },
  { title: 'Path', key: 'path', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.path || '-' },
  { title: 'Method', key: 'method', width: 100, render: row => row.method || '-' },
  { title: '优先级', key: 'priority', width: 100 },
  { title: '更新时间', key: 'updatedAt', width: 180 },
  {
    title: '操作',
    key: 'action',
    width: 180,
    fixed: 'right',
    render(row) {
      return h(
        NSpace,
        { size: 4 },
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                onClick: () => handleEditBinding(row)
              },
              { default: () => '编辑' }
            ),
            h(
              NPopconfirm,
              { onPositiveClick: () => handleDeleteBinding(row) },
              {
                trigger: () => h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
                default: () => '删除后不可恢复，确认继续吗？'
              }
            )
          ]
        }
      );
    }
  }
];

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

const bindingConflictGroups = computed<BindingConflictGroup[]>(() => {
  const groups = new Map<string, BindingConflictGroup>();
  bindingTable.value
    .filter(item => item.enabled)
    .forEach(item => {
      const key = [
        item.scopeType || '',
        String(item.host || '').toLowerCase(),
        item.path || '',
        String(item.method || '').toUpperCase(),
        Number(item.priority || 0)
      ].join('|');
      const current = groups.get(key);
      if (!current) {
        groups.set(key, {
          scopeType: item.scopeType,
          host: item.host || '',
          path: item.path || '',
          method: item.method || '',
          priority: Number(item.priority || 0),
          count: 1
        });
      } else {
        current.count += 1;
      }
    });

  return Array.from(groups.values())
    .filter(item => item.count > 1)
    .sort((a, b) => b.count - a.count || a.priority - b.priority);
});

const bindingEffectivePreview = computed<BindingEffectiveItem[]>(() => {
  const scopeWeightMap: Record<string, number> = {
    global: 1,
    site: 2,
    route: 3
  };

  const sorted = [...bindingTable.value]
    .filter(item => item.enabled)
    .sort((a, b) => {
      const scopeWeightA = scopeWeightMap[a.scopeType] || 99;
      const scopeWeightB = scopeWeightMap[b.scopeType] || 99;
      if (scopeWeightA !== scopeWeightB) return scopeWeightA - scopeWeightB;
      if (a.priority !== b.priority) return a.priority - b.priority;
      return a.id - b.id;
    });

  return sorted.map((item, index) => ({
    id: item.id,
    order: index + 1,
    policyId: item.policyId,
    policyName: mapPolicyNameById(item.policyId),
    scopeType: item.scopeType,
    host: item.host || '',
    path: item.path || '',
    method: item.method || '',
    priority: item.priority
  }));
});

const bindingEffectiveColumns: DataTableColumns<BindingEffectiveItem> = [
  { title: '顺位', key: 'order', width: 80 },
  { title: '策略', key: 'policyName', minWidth: 180, render: row => row.policyName || `#${row.policyId}` },
  { title: '作用域', key: 'scopeType', width: 100, render: row => mapScopeTypeLabel(row.scopeType) },
  { title: 'Host', key: 'host', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.host || '-' },
  { title: 'Path', key: 'path', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.path || '-' },
  { title: 'Method', key: 'method', width: 100, render: row => row.method || '-' },
  { title: '优先级', key: 'priority', width: 100 }
];

const policyStatsTrendColumns: DataTableColumns<WafPolicyStatsTrendItem> = [
  { title: '时间', key: 'time', width: 140 },
  { title: '命中', key: 'hitCount', width: 100 },
  { title: '拦截', key: 'blockedCount', width: 100 },
  { title: '放行', key: 'allowedCount', width: 100 }
];

const policyStatsColumns: DataTableColumns<WafPolicyStatsItem> = [
  { title: '策略', key: 'policyName', minWidth: 180, render: row => row.policyName || `#${row.policyId}` },
  { title: '命中', key: 'hitCount', width: 100 },
  { title: '拦截', key: 'blockedCount', width: 100 },
  { title: '放行', key: 'allowedCount', width: 100 },
  { title: '疑似误报', key: 'suspectedFalsePositiveCount', width: 120 },
  {
    title: '拦截率',
    key: 'blockRate',
    width: 120,
    render: row => formatRatePercent(row.blockRate)
  }
];

const policyStatsDimensionColumns: DataTableColumns<WafPolicyStatsDimensionItem> = [
  { title: '维度值', key: 'key', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.key || '-' },
  { title: '命中', key: 'hitCount', width: 100 },
  { title: '拦截', key: 'blockedCount', width: 100 },
  { title: '放行', key: 'allowedCount', width: 100 },
  {
    title: '拦截率',
    key: 'blockRate',
    width: 120,
    render: row => formatRatePercent(row.blockRate)
  }
];

function mapPolicyFeedbackStatusLabel(status: string) {
  switch (String(status || '').trim().toLowerCase()) {
    case 'confirmed':
      return '已确认';
    case 'resolved':
      return '已处理';
    default:
      return '待确认';
  }
}

function mapPolicyFeedbackStatusTagType(status: string): 'default' | 'warning' | 'success' {
  switch (String(status || '').trim().toLowerCase()) {
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

const policyFeedbackColumns: DataTableColumns<WafPolicyFalsePositiveFeedbackItem> = [
  {
    type: 'selection',
    width: 48
  },
  { title: '策略', key: 'policyName', minWidth: 160, render: row => row.policyName || `#${row.policyId}` },
  { title: 'Host', key: 'host', minWidth: 160, ellipsis: { tooltip: true }, render: row => row.host || '-' },
  { title: 'Path', key: 'path', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.path || '-' },
  { title: 'Method', key: 'method', width: 100, render: row => row.method || '-' },
  {
    title: '状态码',
    key: 'status',
    width: 100,
    render: row => (row.status > 0 ? row.status : '-')
  },
  {
    title: '处理状态',
    key: 'feedbackStatus',
    width: 110,
    render: row => h(NTag, { bordered: false, type: mapPolicyFeedbackStatusTagType(row.feedbackStatus) }, { default: () => mapPolicyFeedbackStatusLabel(row.feedbackStatus) })
  },
  { title: '责任人', key: 'assignee', width: 120, render: row => row.assignee || '-' },
  { title: '截止时间', key: 'dueAt', width: 180, render: row => row.dueAt || '-' },
  {
    title: 'SLA',
    key: 'isOverdue',
    width: 90,
    render: row => h(NTag, { bordered: false, type: mapPolicyFeedbackSLAStatusTagType(row) }, { default: () => mapPolicyFeedbackSLAStatusLabel(row) })
  },
  { title: '误报原因', key: 'reason', minWidth: 220, ellipsis: { tooltip: true }, render: row => row.reason || '-' },
  { title: '建议动作', key: 'suggestion', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.suggestion || '-' },
  { title: '处理备注', key: 'processNote', minWidth: 180, ellipsis: { tooltip: true }, render: row => row.processNote || '-' },
  { title: '处理人', key: 'processedBy', width: 120, render: row => row.processedBy || '-' },
  { title: '处理时间', key: 'processedAt', width: 180, render: row => row.processedAt || '-' },
  { title: '提交人', key: 'operator', width: 120, render: row => row.operator || '-' },
  { title: '提交时间', key: 'createdAt', width: 180 },
  {
    title: '操作',
    key: 'actions',
    width: 230,
    fixed: 'right',
    render: row =>
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
];

type PolicyStatsDimensionType = 'host' | 'path' | 'method';

function normalizePolicyStatsDrillValue(type: PolicyStatsDimensionType, raw: string) {
  const text = String(raw || '').trim();
  if (type === 'host') {
    if (text === '(empty)') return '(empty)';
    return text.toLowerCase();
  }
  if (type === 'method') {
    return text.toUpperCase();
  }
  return text;
}

function isPolicyStatsDrillUnlocked(type: PolicyStatsDimensionType) {
  if (type === 'host') return true;
  if (type === 'path') return !!policyStatsQuery.host.trim();
  return !!(policyStatsQuery.host.trim() && policyStatsQuery.path.trim());
}

function policyStatsDrillStatusLabel(type: PolicyStatsDimensionType) {
  if (type === 'host') return '入口层';
  return isPolicyStatsDrillUnlocked(type) ? '已解锁' : '待解锁';
}

function policyStatsDrillHint(type: PolicyStatsDimensionType) {
  if (type === 'host') {
    return '第一层下钻入口：点击 Host 可进入 Host 维度过滤。';
  }
  if (type === 'path') {
    if (!isPolicyStatsDrillUnlocked(type)) {
      return '待解锁：请先在 Top Host 中选择一个 Host。';
    }
    return `已解锁：当前 Host=${policyStatsQuery.host || '-'}，点击 Path 继续下钻。`;
  }
  if (!isPolicyStatsDrillUnlocked(type)) {
    return '待解锁：请先完成 Host + Path 下钻。';
  }
  return `已解锁：当前 Host=${policyStatsQuery.host || '-'}，Path=${policyStatsQuery.path || '-'}。点击 Method 继续下钻。`;
}

function canPolicyStatsDrillDimension(type: PolicyStatsDimensionType) {
  if (type === 'host') return true;
  if (type === 'path') return !!policyStatsQuery.host.trim();
  return !!(policyStatsQuery.host.trim() && policyStatsQuery.path.trim());
}

function isPolicyStatsDimensionSelected(type: PolicyStatsDimensionType, row: WafPolicyStatsDimensionItem) {
  const key = normalizePolicyStatsDrillValue(type, String(row?.key || ''));
  if (!key || key === '-') return false;
  if (type === 'host') {
    return key === normalizePolicyStatsDrillValue('host', policyStatsQuery.host);
  }
  if (type === 'path') {
    return key === normalizePolicyStatsDrillValue('path', policyStatsQuery.path);
  }
  return key === normalizePolicyStatsDrillValue('method', policyStatsQuery.method);
}

function handlePolicyStatsDimensionDrill(type: PolicyStatsDimensionType, row: WafPolicyStatsDimensionItem) {
  const key = String(row?.key || '').trim();
  if (!key || key === '-') {
    return;
  }

  if (!canPolicyStatsDrillDimension(type)) {
    if (type === 'path') {
      message.warning('请先从 Top Host 选择一个 Host，再下钻 Path');
    } else if (type === 'method') {
      message.warning('请先完成 Host + Path 下钻，再下钻 Method');
    }
    return;
  }

  if (type === 'host') {
    policyStatsQuery.host = key;
    policyStatsQuery.path = '';
    policyStatsQuery.method = '';
  } else if (type === 'path') {
    policyStatsQuery.path = key;
    policyStatsQuery.method = '';
  } else {
    policyStatsQuery.method = key;
  }

  fetchPolicyStats();
}

function buildPolicyStatsDimensionRowProps(type: PolicyStatsDimensionType) {
  return (row: WafPolicyStatsDimensionItem) => {
    const clickable = canPolicyStatsDrillDimension(type);
    const selected = isPolicyStatsDimensionSelected(type, row);
    const styleParts = ['transition: background-color 0.2s ease'];
    if (clickable) {
      styleParts.push('cursor: pointer');
    } else {
      styleParts.push('cursor: not-allowed');
      styleParts.push('opacity: 0.65');
    }
    if (selected) {
      styleParts.push('background: rgba(24, 160, 88, 0.14)');
      styleParts.push('font-weight: 600');
      styleParts.push('box-shadow: inset 3px 0 0 rgba(24, 160, 88, 0.9)');
    }
    const lockedHint = type === 'path' ? '请先从 Top Host 选择一个 Host' : '请先完成 Host + Path 下钻';
    return {
      style: styleParts.join(';'),
      title: clickable ? '点击下钻' : lockedHint,
      onClick: () => {
        if (!clickable) return;
        handlePolicyStatsDimensionDrill(type, row);
      }
    };
  };
}

const releaseColumns: DataTableColumns<WafReleaseItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '更新源', key: 'sourceName', minWidth: 160, render: row => mapSourceNameById(row.sourceId) },
  { title: '版本', key: 'version', minWidth: 180, ellipsis: { tooltip: true } },
  { title: '包类型', key: 'artifactType', width: 110 },
  {
    title: '大小',
    key: 'sizeBytes',
    width: 120,
    render(row) {
      return formatBytes(row.sizeBytes);
    }
  },
  { title: '校验值', key: 'checksum', minWidth: 220, ellipsis: { tooltip: true }, render: row => row.checksum || '-' },
  {
    title: '状态',
    key: 'status',
    width: 120,
    render(row) {
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
    render(row) {
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
];

const jobColumns: DataTableColumns<WafJobItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '更新源', key: 'sourceName', minWidth: 160, render: row => mapJobSourceName(row) },
  { title: '动作', key: 'action', width: 120, render: row => mapJobActionLabel(row.action) },
  { title: '触发方式', key: 'triggerMode', width: 120, render: row => mapJobTriggerModeLabel(row.triggerMode) },
  {
    title: '状态',
    key: 'status',
    width: 110,
    render(row) {
      return h(NTag, { type: mapJobStatusType(row.status), bordered: false }, { default: () => mapJobStatusLabel(row.status) });
    }
  },
  { title: '操作人', key: 'operator', width: 120, render: row => row.operator || '-' },
  { title: '开始时间', key: 'startedAt', width: 180, render: row => row.startedAt || '-' },
  { title: '结束时间', key: 'finishedAt', width: 180, render: row => row.finishedAt || '-' },
  {
    title: '消息',
    key: 'message',
    minWidth: 320,
    ellipsis: { tooltip: true },
    render(row) {
      return mapJobMessage(row.message);
    }
  }
];

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

function mapPolicyEngineModeLabel(mode: WafPolicyEngineMode) {
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

function mapCrsTemplateLabel(template: WafPolicyCrsTemplate | string) {
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

function mapScopeTypeLabel(scopeType: WafPolicyScopeType | string) {
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

function inferCrsTemplateByValues(crsParanoiaLevel: number, crsInboundAnomalyThreshold: number, crsOutboundAnomalyThreshold: number): WafPolicyCrsTemplate {
  for (const option of crsTemplateOptions) {
    if (option.value === 'custom') {
      continue;
    }
    const preset = crsTemplatePresetMap[option.value];
    if (
      preset.crsParanoiaLevel === crsParanoiaLevel &&
      preset.crsInboundAnomalyThreshold === crsInboundAnomalyThreshold &&
      preset.crsOutboundAnomalyThreshold === crsOutboundAnomalyThreshold
    ) {
      return option.value;
    }
  }
  return 'custom';
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

function mapPolicyRevisionStatusLabel(status: WafPolicyRevisionStatus) {
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
  const pendingIds = Array.from(new Set(sourceIds.filter(sourceId => sourceId > 0 && !jobSourceNameMap.value[sourceId])));
  if (pendingIds.length === 0) {
    return;
  }

  const pageSize = 200;
  let page = 1;
  let total = 0;

  while (page <= 20) {
    const { data, error } = await fetchWafSourceList({
      page,
      pageSize,
      name: undefined
    });

    if (error || !data) {
      break;
    }

    const sourceList = data.list || [];
    mergeJobSourceNameMap(sourceList);
    total = data.total || 0;

    const hasAllPending = pendingIds.every(sourceId => !!jobSourceNameMap.value[sourceId]);
    if (hasAllPending) {
      break;
    }

    if (sourceList.length === 0 || page * pageSize >= total) {
      break;
    }

    page += 1;
  }
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

function formatBytes(size: number) {
  if (!size || size <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = size;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unitIndex]}`;
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

function pickRouteQueryValue(value: unknown) {
  if (Array.isArray(value)) {
    return String(value[0] ?? '').trim();
  }
  if (value == null) {
    return '';
  }
  return String(value).trim();
}

function parseRangedInteger(rawValue: string, defaultValue: number, min: number, max: number) {
  const parsed = Number.parseInt(rawValue, 10);
  if (!Number.isFinite(parsed)) {
    return defaultValue;
  }
  return Math.min(max, Math.max(min, parsed));
}

function buildQuerySignature(query: Record<string, unknown>) {
  const normalized: Record<string, string> = {};
  Object.entries(query).forEach(([key, value]) => {
    const resolved = pickRouteQueryValue(value);
    if (resolved) {
      normalized[key] = resolved;
    }
  });
  const orderedKeys = Object.keys(normalized).sort();
  return JSON.stringify(
    orderedKeys.map(key => [key, normalized[key]])
  );
}

function applyObserveQueryFromRoute(query: Record<string, unknown>) {
  const routeTab = pickRouteQueryValue(query.activeTab);
  const shouldEnterObserve = routeTab === 'observe';
  if (shouldEnterObserve && activeTab.value !== 'observe') {
    activeTab.value = 'observe';
  }
  if (activeTab.value !== 'observe') {
    return false;
  }

  const nextPolicyIdRaw = pickRouteQueryValue(query.policyId);
  const nextPolicyIdParsed = Number.parseInt(nextPolicyIdRaw, 10);
  const nextPolicyId = Number.isInteger(nextPolicyIdParsed) && nextPolicyIdParsed > 0 ? nextPolicyIdParsed : '';
  const nextWindowRaw = pickRouteQueryValue(query.window);
  const nextWindow = observeWindowValueSet.has(nextWindowRaw)
    ? (nextWindowRaw as (typeof policyStatsQuery)['window'])
    : '24h';
  const nextIntervalSec = parseRangedInteger(pickRouteQueryValue(query.intervalSec), 300, 60, 86400);
  const nextTopN = parseRangedInteger(pickRouteQueryValue(query.topN), 8, 1, 50);
  const nextHost = pickRouteQueryValue(query.host);
  const nextPath = pickRouteQueryValue(query.path);
  const nextMethod = pickRouteQueryValue(query.method).toUpperCase();

  const changed =
    policyStatsQuery.policyId !== nextPolicyId ||
    policyStatsQuery.window !== nextWindow ||
    Number(policyStatsQuery.intervalSec) !== nextIntervalSec ||
    Number(policyStatsQuery.topN) !== nextTopN ||
    policyStatsQuery.host !== nextHost ||
    policyStatsQuery.path !== nextPath ||
    policyStatsQuery.method !== nextMethod;

  if (changed) {
    policyStatsQuery.policyId = nextPolicyId;
    policyStatsQuery.window = nextWindow;
    policyStatsQuery.intervalSec = nextIntervalSec;
    policyStatsQuery.topN = nextTopN;
    policyStatsQuery.host = nextHost;
    policyStatsQuery.path = nextPath;
    policyStatsQuery.method = nextMethod;
  }

  return changed;
}

async function syncObserveStateToRouteQuery() {
  if (activeTab.value !== 'observe') {
    return;
  }

  const nextQuery: Record<string, string> = {};
  Object.entries(route.query).forEach(([key, value]) => {
    const resolved = pickRouteQueryValue(value);
    if (resolved) {
      nextQuery[key] = resolved;
    }
  });

  nextQuery.activeTab = 'observe';
  if (policyStatsQuery.policyId) {
    nextQuery.policyId = String(policyStatsQuery.policyId);
  } else {
    delete nextQuery.policyId;
  }
  nextQuery.window = policyStatsQuery.window;
  nextQuery.intervalSec = String(parseRangedInteger(String(policyStatsQuery.intervalSec), 300, 60, 86400));
  nextQuery.topN = String(parseRangedInteger(String(policyStatsQuery.topN), 8, 1, 50));

  const host = policyStatsQuery.host.trim();
  const path = policyStatsQuery.path.trim();
  const method = policyStatsQuery.method.trim().toUpperCase();
  if (host) {
    nextQuery.host = host;
  } else {
    delete nextQuery.host;
  }
  if (path) {
    nextQuery.path = path;
  } else {
    delete nextQuery.path;
  }
  if (method) {
    nextQuery.method = method;
  } else {
    delete nextQuery.method;
  }

  if (buildQuerySignature(route.query as Record<string, unknown>) === buildQuerySignature(nextQuery)) {
    return;
  }

  observeRouteSyncing.value = true;
  try {
    await router.replace({ query: nextQuery });
  } finally {
    observeRouteSyncing.value = false;
  }
}

async function handleCopyPolicyStatsLink() {
  await syncObserveStateToRouteQuery();
  const currentUrl = window.location.href;

  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(currentUrl);
      message.success('已复制当前筛选链接');
      return;
    } catch {
      // ignore and fallback to execCommand
    }
  }

  const textArea = document.createElement('textarea');
  textArea.value = currentUrl;
  textArea.style.position = 'fixed';
  textArea.style.opacity = '0';
  document.body.appendChild(textArea);
  textArea.focus();
  textArea.select();
  const copied = document.execCommand('copy');
  document.body.removeChild(textArea);
  if (copied) {
    message.success('已复制当前筛选链接');
  } else {
    message.warning('复制失败，请手动复制浏览器地址栏链接');
  }
}

function resolvePolicyStatsWindowRange() {
  const end = new Date();
  const start = new Date(end.getTime());
  switch (policyStatsQuery.window) {
    case '1h':
      start.setHours(start.getHours() - 1);
      break;
    case '6h':
      start.setHours(start.getHours() - 6);
      break;
    case '7d':
      start.setDate(start.getDate() - 7);
      break;
    default:
      start.setDate(start.getDate() - 1);
      break;
  }
  return {
    startTime: formatDateTime(start),
    endTime: formatDateTime(end)
  };
}

function buildCurrentPolicyStatsSnapshot(): PolicyStatsSnapshot {
  return {
    capturedAt: formatDateTime(new Date()),
    query: {
      policyId: policyStatsQuery.policyId,
      window: policyStatsQuery.window,
      intervalSec: Number(policyStatsQuery.intervalSec || 300),
      topN: Number(policyStatsQuery.topN || 8),
      host: policyStatsQuery.host.trim(),
      path: policyStatsQuery.path.trim(),
      method: policyStatsQuery.method.trim().toUpperCase()
    },
    range: {
      startTime: policyStatsRange.value.startTime || '',
      endTime: policyStatsRange.value.endTime || '',
      intervalSec: Number(policyStatsRange.value.intervalSec || 0)
    },
    summary: { ...policyStatsSummary.value },
    list: policyStatsTable.value.map(item => ({ ...item })),
    trend: policyStatsTrend.value.map(item => ({ ...item })),
    topHosts: policyStatsTopHosts.value.map(item => ({ ...item })),
    topPaths: policyStatsTopPaths.value.map(item => ({ ...item })),
    topMethods: policyStatsTopMethods.value.map(item => ({ ...item }))
  };
}

function shouldCapturePolicyStatsSnapshot() {
  if (policyStatsTable.value.length > 0 || policyStatsTrend.value.length > 0) {
    return true;
  }
  if (Number(policyStatsSummary.value.hitCount || 0) > 0) {
    return true;
  }
  return !!(policyStatsRange.value.startTime || policyStatsRange.value.endTime);
}

function buildPolicyFeedbackListParams() {
  return {
    page: Number(policyFeedbackPagination.page || 1),
    pageSize: Number(policyFeedbackPagination.pageSize || 10),
    policyId: policyStatsQuery.policyId ? Number(policyStatsQuery.policyId) : undefined,
    host: policyStatsQuery.host.trim() || undefined,
    path: policyStatsQuery.path.trim() || undefined,
    method: policyStatsQuery.method.trim().toUpperCase() || undefined,
    feedbackStatus: policyFeedbackStatusFilter.value || undefined,
    assignee: policyFeedbackAssigneeFilter.value.trim() || undefined,
    slaStatus: policyFeedbackSLAStatusFilter.value || undefined
  };
}

async function fetchPolicyFalsePositiveFeedbacks() {
  policyFeedbackLoading.value = true;
  try {
    const { data, error } = await fetchWafPolicyFalsePositiveFeedbackList(buildPolicyFeedbackListParams());
    if (!error && data) {
      policyFeedbackTable.value = data.list || [];
      policyFeedbackPagination.itemCount = data.total || 0;
    }
  } finally {
    policyFeedbackLoading.value = false;
  }
}

function resetPolicyFeedbackSelection() {
  policyFeedbackCheckedRowKeys.value = [];
}

function resetPolicyFeedbackForm() {
  policyFeedbackForm.policyId = policyStatsQuery.policyId ? Number(policyStatsQuery.policyId) : null;
  policyFeedbackForm.host = policyStatsQuery.host.trim();
  policyFeedbackForm.path = policyStatsQuery.path.trim();
  policyFeedbackForm.method = policyStatsQuery.method.trim().toUpperCase() || null;
  policyFeedbackForm.status = 403;
  policyFeedbackForm.assignee = policyFeedbackAssigneeFilter.value.trim();
  policyFeedbackForm.dueAt = '';
  policyFeedbackForm.sampleUri = '';
  policyFeedbackForm.reason = '';
  policyFeedbackForm.suggestion = '';
}

function openPolicyFeedbackModal() {
  resetPolicyFeedbackForm();
  policyFeedbackModalVisible.value = true;
}

function handlePolicyFeedbackPageChange(page: number) {
  policyFeedbackPagination.page = page;
  fetchPolicyFalsePositiveFeedbacks();
}

function handlePolicyFeedbackPageSizeChange(pageSize: number) {
  policyFeedbackPagination.pageSize = pageSize;
  policyFeedbackPagination.page = 1;
  fetchPolicyFalsePositiveFeedbacks();
}

function handlePolicyFeedbackStatusFilterChange() {
  policyFeedbackPagination.page = 1;
  resetPolicyFeedbackSelection();
  fetchPolicyFalsePositiveFeedbacks();
}

function handlePolicyFeedbackCheckedRowKeysChange(keys: Array<string | number>) {
  const currentPageIDs = policyFeedbackTable.value.map(item => Number(item.id || 0)).filter(id => id > 0);
  policyFeedbackCheckedRowKeys.value = mergePolicyFeedbackCheckedRowKeys(policyFeedbackCheckedRowKeys.value, currentPageIDs, keys);
}

function openPolicyFeedbackProcessModal(row: WafPolicyFalsePositiveFeedbackItem) {
  policyFeedbackProcessForm.id = Number(row.id || 0);
  policyFeedbackProcessForm.feedbackStatus = (row.feedbackStatus || 'pending') as 'pending' | 'confirmed' | 'resolved';
  policyFeedbackProcessForm.processNote = row.processNote || '';
  policyFeedbackProcessForm.assignee = row.assignee || '';
  policyFeedbackProcessForm.dueAt = row.dueAt || '';
  policyFeedbackProcessModalVisible.value = true;
}

async function handleSubmitPolicyFeedbackProcess() {
  await policyFeedbackProcessFormRef.value?.validate();
  if (!policyFeedbackProcessForm.id) {
    message.error('反馈记录无效');
    return;
  }
  policyFeedbackProcessSubmitting.value = true;
  try {
    const payload: WafPolicyFalsePositiveFeedbackStatusUpdatePayload = {
      feedbackStatus: policyFeedbackProcessForm.feedbackStatus,
      processNote: policyFeedbackProcessForm.processNote.trim() || undefined,
      assignee: policyFeedbackProcessForm.assignee.trim() || undefined,
      dueAt: policyFeedbackProcessForm.dueAt.trim() || undefined
    };
    const { error } = await updateWafPolicyFalsePositiveFeedbackStatus(policyFeedbackProcessForm.id, payload);
    if (!error) {
      message.success('误报反馈状态已更新');
      policyFeedbackProcessModalVisible.value = false;
      fetchPolicyFalsePositiveFeedbacks();
    }
  } finally {
    policyFeedbackProcessSubmitting.value = false;
  }
}

function resetPolicyFeedbackBatchProcessForm() {
  policyFeedbackBatchProcessForm.feedbackStatus = 'confirmed';
  policyFeedbackBatchProcessForm.processNote = '';
  policyFeedbackBatchProcessForm.assignee = policyFeedbackAssigneeFilter.value.trim();
  policyFeedbackBatchProcessForm.dueAt = '';
}

function openPolicyFeedbackBatchProcessModal() {
  if (!policyFeedbackCheckedRowKeys.value.length) {
    message.warning('请先选择要处理的反馈记录');
    return;
  }
  resetPolicyFeedbackBatchProcessForm();
  policyFeedbackBatchProcessModalVisible.value = true;
}

async function handleSubmitPolicyFeedbackBatchProcess() {
  await policyFeedbackBatchProcessFormRef.value?.validate();
  const selectedIDs = Array.from(new Set(policyFeedbackCheckedRowKeys.value.map(id => Number(id)).filter(id => Number.isInteger(id) && id > 0)));
  if (!selectedIDs.length) {
    message.warning('未选择可处理的反馈记录');
    return;
  }

  policyFeedbackBatchProcessSubmitting.value = true;
  try {
    const payload: WafPolicyFalsePositiveFeedbackBatchStatusUpdatePayload = {
      ids: selectedIDs,
      feedbackStatus: policyFeedbackBatchProcessForm.feedbackStatus,
      processNote: policyFeedbackBatchProcessForm.processNote.trim() || undefined,
      assignee: policyFeedbackBatchProcessForm.assignee.trim() || undefined,
      dueAt: policyFeedbackBatchProcessForm.dueAt.trim() || undefined
    };
    const { data, error } = await batchUpdateWafPolicyFalsePositiveFeedbackStatus(payload);
    if (!error) {
      const affectedCount = Number(data?.affectedCount || 0);
      message.success(affectedCount > 0 ? `批量处理完成，已更新 ${affectedCount} 条反馈` : '批量处理完成');
      policyFeedbackBatchProcessModalVisible.value = false;
      resetPolicyFeedbackSelection();
      fetchPolicyFalsePositiveFeedbacks();
    }
  } finally {
    policyFeedbackBatchProcessSubmitting.value = false;
  }
}

async function handleSubmitPolicyFeedback() {
  await policyFeedbackFormRef.value?.validate();
  policyFeedbackSubmitting.value = true;
  try {
    const payload: WafPolicyFalsePositiveFeedbackPayload = {
      policyId: policyFeedbackForm.policyId || undefined,
      host: policyFeedbackForm.host.trim() || undefined,
      path: policyFeedbackForm.path.trim() || undefined,
      method: String(policyFeedbackForm.method || '').trim().toUpperCase() || undefined,
      status: Number(policyFeedbackForm.status || 403),
      assignee: policyFeedbackForm.assignee.trim() || undefined,
      dueAt: policyFeedbackForm.dueAt.trim() || undefined,
      sampleUri: policyFeedbackForm.sampleUri.trim() || undefined,
      reason: policyFeedbackForm.reason.trim(),
      suggestion: policyFeedbackForm.suggestion.trim() || undefined
    };
    const { error } = await createWafPolicyFalsePositiveFeedback(payload);
    if (!error) {
      message.success('误报反馈已提交');
      policyFeedbackModalVisible.value = false;
      policyFeedbackPagination.page = 1;
      resetPolicyFeedbackSelection();
      fetchPolicyFalsePositiveFeedbacks();
    }
  } finally {
    policyFeedbackSubmitting.value = false;
  }
}

async function fetchPolicyStats() {
  const previousSnapshot = shouldCapturePolicyStatsSnapshot() ? buildCurrentPolicyStatsSnapshot() : null;
  policyStatsLoading.value = true;
  try {
    const { startTime, endTime } = resolvePolicyStatsWindowRange();
    const { data, error } = await fetchWafPolicyStats({
      policyId: policyStatsQuery.policyId ? Number(policyStatsQuery.policyId) : undefined,
      startTime,
      endTime,
      intervalSec: Number(policyStatsQuery.intervalSec || 300),
      topN: Number(policyStatsQuery.topN || 8),
      host: policyStatsQuery.host.trim() || undefined,
      path: policyStatsQuery.path.trim() || undefined,
      method: policyStatsQuery.method.trim() || undefined
    });
    if (!error && data) {
      policyStatsSummary.value = data.summary || {
        policyId: 0,
        policyName: '全部策略',
        hitCount: 0,
        blockedCount: 0,
        allowedCount: 0,
        suspectedFalsePositiveCount: 0,
        blockRate: 0
      };
      policyStatsTable.value = data.list || [];
      policyStatsTrend.value = data.trend || [];
      policyStatsTopHosts.value = data.topHosts || [];
      policyStatsTopPaths.value = data.topPaths || [];
      policyStatsTopMethods.value = data.topMethods || [];
      policyStatsRange.value = data.range || { startTime: '', endTime: '', intervalSec: Number(policyStatsQuery.intervalSec || 300) };
      policyStatsPreviousSnapshot.value = previousSnapshot;
    }
  } finally {
    policyStatsLoading.value = false;
  }
  resetPolicyFeedbackSelection();
  fetchPolicyFalsePositiveFeedbacks();
}

function resetPolicyStatsQuery() {
  policyStatsQuery.policyId = '';
  policyStatsQuery.window = '24h';
  policyStatsQuery.intervalSec = 300;
  policyStatsQuery.topN = 8;
  policyStatsQuery.host = '';
  policyStatsQuery.path = '';
  policyStatsQuery.method = '';
  fetchPolicyStats();
}

function clearPolicyStatsDrillFilters() {
  policyStatsQuery.host = '';
  policyStatsQuery.path = '';
  policyStatsQuery.method = '';
  fetchPolicyStats();
}

function clearPolicyStatsDrillLevel(level: PolicyStatsDimensionType) {
  if (level === 'host') {
    policyStatsQuery.host = '';
    policyStatsQuery.path = '';
    policyStatsQuery.method = '';
  } else if (level === 'path') {
    policyStatsQuery.path = '';
    policyStatsQuery.method = '';
  } else {
    policyStatsQuery.method = '';
  }
  fetchPolicyStats();
}

function escapeCsvCell(value: unknown) {
  const text = String(value ?? '');
  if (text.includes('"') || text.includes(',') || text.includes('\n')) {
    return `"${text.replace(/"/g, '""')}"`;
  }
  return text;
}

function buildDimensionCsvRows(
  section: string,
  rows: WafPolicyStatsDimensionItem[]
) {
  const lines: string[] = [escapeCsvCell(section), '维度值,命中,拦截,放行,拦截率'];
  rows.forEach(row => {
    lines.push(
      [
        escapeCsvCell(row.key || '-'),
        row.hitCount,
        row.blockedCount,
        row.allowedCount,
        escapeCsvCell(formatRatePercent(row.blockRate))
      ].join(',')
    );
  });
  lines.push('');
  return lines;
}

function buildDimensionCompareCsvRows(
  section: string,
  currentRows: WafPolicyStatsDimensionItem[],
  previousRows: WafPolicyStatsDimensionItem[]
) {
  const lines: string[] = [escapeCsvCell(section), '维度值,当前命中,基线命中,命中变化,当前拦截,基线拦截,拦截变化,当前放行,基线放行,放行变化,当前拦截率,基线拦截率,拦截率变化(pp)'];
  const currentMap = new Map<string, WafPolicyStatsDimensionItem>();
  const previousMap = new Map<string, WafPolicyStatsDimensionItem>();
  currentRows.forEach(item => currentMap.set(String(item.key || '-'), item));
  previousRows.forEach(item => previousMap.set(String(item.key || '-'), item));
  const allKeys = Array.from(new Set([...currentMap.keys(), ...previousMap.keys()])).sort((a, b) => a.localeCompare(b));
  allKeys.forEach(key => {
    const current = currentMap.get(key);
    const previous = previousMap.get(key);
    const currentHit = Number(current?.hitCount || 0);
    const previousHit = Number(previous?.hitCount || 0);
    const currentBlocked = Number(current?.blockedCount || 0);
    const previousBlocked = Number(previous?.blockedCount || 0);
    const currentAllowed = Number(current?.allowedCount || 0);
    const previousAllowed = Number(previous?.allowedCount || 0);
    const currentRate = Number(current?.blockRate || 0);
    const previousRate = Number(previous?.blockRate || 0);
    lines.push(
      [
        escapeCsvCell(key || '-'),
        currentHit,
        previousHit,
        currentHit - previousHit,
        currentBlocked,
        previousBlocked,
        currentBlocked - previousBlocked,
        currentAllowed,
        previousAllowed,
        currentAllowed - previousAllowed,
        escapeCsvCell(formatRatePercent(currentRate)),
        escapeCsvCell(formatRatePercent(previousRate)),
        `${((currentRate - previousRate) * 100).toFixed(2)}pp`
      ].join(',')
    );
  });
  lines.push('');
  return lines;
}

function handleExportPolicyStatsCsv() {
  const lines: string[] = [
    'LogFlux WAF Policy Stats Export',
    `导出时间,${escapeCsvCell(formatDateTime(new Date()))}`,
    `统计区间开始,${escapeCsvCell(policyStatsRange.value.startTime || '-')}`,
    `统计区间结束,${escapeCsvCell(policyStatsRange.value.endTime || '-')}`,
    `趋势粒度秒,${policyStatsRange.value.intervalSec || 0}`,
    `下钻Host,${escapeCsvCell(policyStatsQuery.host || '-')}`,
    `下钻Path,${escapeCsvCell(policyStatsQuery.path || '-')}`,
    `下钻Method,${escapeCsvCell(policyStatsQuery.method || '-')}`,
    ''
  ];

  lines.push('总览');
  lines.push('策略,命中,拦截,放行,疑似误报,拦截率');
  lines.push(
    [
      escapeCsvCell(policyStatsSummary.value.policyName || '-'),
      policyStatsSummary.value.hitCount,
      policyStatsSummary.value.blockedCount,
      policyStatsSummary.value.allowedCount,
      policyStatsSummary.value.suspectedFalsePositiveCount,
      escapeCsvCell(formatRatePercent(policyStatsSummary.value.blockRate))
    ].join(',')
  );
  lines.push('');

  lines.push('策略统计');
  lines.push('策略,命中,拦截,放行,疑似误报,拦截率');
  policyStatsTable.value.forEach(row => {
    lines.push(
      [
        escapeCsvCell(row.policyName || `#${row.policyId}`),
        row.hitCount,
        row.blockedCount,
        row.allowedCount,
        row.suspectedFalsePositiveCount,
        escapeCsvCell(formatRatePercent(row.blockRate))
      ].join(',')
    );
  });
  lines.push('');

  lines.push('趋势');
  lines.push('时间,命中,拦截,放行');
  policyStatsTrend.value.forEach(row => {
    lines.push([escapeCsvCell(row.time), row.hitCount, row.blockedCount, row.allowedCount].join(','));
  });
  lines.push('');

  lines.push(...buildDimensionCsvRows('Top Host', policyStatsTopHosts.value));
  lines.push(...buildDimensionCsvRows('Top Path', policyStatsTopPaths.value));
  lines.push(...buildDimensionCsvRows('Top Method', policyStatsTopMethods.value));

  const content = `\ufeff${lines.join('\n')}`;
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `logflux-waf-policy-stats-${Date.now()}.csv`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
  message.success('策略观测统计已导出');
}

function buildPolicyStatsSnapshotScopeText(snapshot: PolicyStatsSnapshot) {
  const policyLabel = snapshot.query.policyId ? `#${snapshot.query.policyId}` : '全部策略';
  return `策略=${policyLabel}, window=${snapshot.query.window}, interval=${snapshot.query.intervalSec}s, topN=${snapshot.query.topN}, host=${snapshot.query.host || '-'}, path=${snapshot.query.path || '-'}, method=${snapshot.query.method || '-'}`;
}

function handleExportPolicyStatsCompareCsv() {
  const previous = policyStatsPreviousSnapshot.value;
  if (!previous) {
    message.warning('暂无可对比的历史快照');
    return;
  }

  const current = buildCurrentPolicyStatsSnapshot();
  const delta = (next: number, prev: number) => Number(next || 0) - Number(prev || 0);
  const lines: string[] = [
    'LogFlux WAF Policy Stats Compare Export',
    `导出时间,${escapeCsvCell(formatDateTime(new Date()))}`,
    `当前快照时间,${escapeCsvCell(current.capturedAt)}`,
    `对比基线时间,${escapeCsvCell(previous.capturedAt)}`,
    `当前筛选,${escapeCsvCell(buildPolicyStatsSnapshotScopeText(current))}`,
    `基线筛选,${escapeCsvCell(buildPolicyStatsSnapshotScopeText(previous))}`,
    ''
  ];

  lines.push('总览对比');
  lines.push('指标,当前,基线,变化');
  lines.push(['命中', current.summary.hitCount, previous.summary.hitCount, delta(current.summary.hitCount, previous.summary.hitCount)].join(','));
  lines.push(['拦截', current.summary.blockedCount, previous.summary.blockedCount, delta(current.summary.blockedCount, previous.summary.blockedCount)].join(','));
  lines.push(['放行', current.summary.allowedCount, previous.summary.allowedCount, delta(current.summary.allowedCount, previous.summary.allowedCount)].join(','));
  lines.push(
    ['疑似误报', current.summary.suspectedFalsePositiveCount, previous.summary.suspectedFalsePositiveCount, delta(current.summary.suspectedFalsePositiveCount, previous.summary.suspectedFalsePositiveCount)].join(',')
  );
  lines.push(
    [
      '拦截率',
      escapeCsvCell(formatRatePercent(current.summary.blockRate)),
      escapeCsvCell(formatRatePercent(previous.summary.blockRate)),
      `${(delta(current.summary.blockRate, previous.summary.blockRate) * 100).toFixed(2)}pp`
    ].join(',')
  );
  lines.push('');

  lines.push('策略维度对比');
  lines.push('策略,当前命中,基线命中,命中变化,当前拦截,基线拦截,拦截变化');
  const currentMap = new Map<number, WafPolicyStatsItem>();
  const previousMap = new Map<number, WafPolicyStatsItem>();
  current.list.forEach(item => currentMap.set(Number(item.policyId || 0), item));
  previous.list.forEach(item => previousMap.set(Number(item.policyId || 0), item));
  const allPolicyIds = Array.from(new Set([...currentMap.keys(), ...previousMap.keys()])).sort((a, b) => a - b);
  allPolicyIds.forEach(policyId => {
    const currentItem = currentMap.get(policyId);
    const previousItem = previousMap.get(policyId);
    const policyName = currentItem?.policyName || previousItem?.policyName || `#${policyId}`;
    const currentHit = Number(currentItem?.hitCount || 0);
    const previousHit = Number(previousItem?.hitCount || 0);
    const currentBlocked = Number(currentItem?.blockedCount || 0);
    const previousBlocked = Number(previousItem?.blockedCount || 0);
    lines.push(
      [
        escapeCsvCell(policyName),
        currentHit,
        previousHit,
        delta(currentHit, previousHit),
        currentBlocked,
        previousBlocked,
        delta(currentBlocked, previousBlocked)
      ].join(',')
    );
  });
  lines.push('');

  lines.push(...buildDimensionCompareCsvRows('Top Host 对比', current.topHosts, previous.topHosts));
  lines.push(...buildDimensionCompareCsvRows('Top Path 对比', current.topPaths, previous.topPaths));
  lines.push(...buildDimensionCompareCsvRows('Top Method 对比', current.topMethods, previous.topMethods));

  const content = `\ufeff${lines.join('\n')}`;
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `logflux-waf-policy-stats-compare-${Date.now()}.csv`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
  message.success('策略观测对比统计已导出');
}

function displayEngineValue(value: unknown) {
  if (value === undefined || value === null || value === '') {
    return '-';
  }
  return String(value);
}

async function fetchEngineStatus() {
  if (engineUnavailable.value) {
    return;
  }

  engineLoading.value = true;
  try {
    const { data, error } = await fetchWafEngineStatus();
    if (!error && data) {
      engineStatus.value = data;
      engineUnavailable.value = false;
      return;
    }

    if (error) {
      const status = Number((error as any)?.response?.status || 0);
      if (status === 404 || status === 405) {
        engineUnavailable.value = true;
      }
    }
  } finally {
    engineLoading.value = false;
  }
}

function handleRefreshEngineStatus() {
  fetchEngineStatus();
}

async function handleCheckEngine() {
  if (engineUnavailable.value) {
    message.warning('后端接口尚未开放，当前仅展示占位状态');
    return;
  }

  engineChecking.value = true;
  try {
    const { error } = await checkWafEngine();
    if (!error) {
      message.success('引擎检查任务已提交');
      fetchEngineStatus();
      if (activeTab.value === 'job') {
        fetchJobs();
      }
      return;
    }

    const status = Number((error as any)?.response?.status || 0);
    if (status === 404 || status === 405) {
      engineUnavailable.value = true;
      message.warning('后端接口尚未开放，已切换占位模式');
      return;
    }
  } finally {
    engineChecking.value = false;
  }
}

async function fetchSources() {
  sourceLoading.value = true;
  try {
    const { data, error } = await fetchWafSourceList({
      page: sourcePagination.page as number,
      pageSize: sourcePagination.pageSize as number,
      kind: 'crs',
      name: sourceQuery.name.trim() || undefined
    });
    if (!error && data) {
      const list = data.list || [];
      const total = data.total || 0;

      if (!sourceQuery.name.trim() && total > 0 && list.length === 0 && (sourcePagination.page as number) > 1) {
        sourcePagination.page = 1;
        await fetchSources();
        return;
      }

      sourceTable.value = list;
      mergeJobSourceNameMap(list);
      sourcePagination.itemCount = total;
    }
  } finally {
    sourceLoading.value = false;
  }
}

function resetSourceQuery() {
  sourceQuery.name = '';
  sourcePagination.page = 1;
  fetchSources();
}

function handleSourcePageChange(page: number) {
  sourcePagination.page = page;
  fetchSources();
}

function handleSourcePageSizeChange(pageSize: number) {
  sourcePagination.pageSize = pageSize;
  sourcePagination.page = 1;
  fetchSources();
}

function resetSourceForm() {
  sourceForm.id = 0;
  sourceForm.name = '';
  sourceForm.kind = 'crs';
  sourceForm.mode = 'remote';
  sourceForm.url = '';
  sourceForm.checksumUrl = '';
  sourceForm.proxyUrl = '';
  sourceForm.authType = 'none';
  sourceForm.authSecret = '';
  sourceForm.schedule = '';
  sourceForm.enabled = true;
  sourceForm.autoCheck = true;
  sourceForm.autoDownload = true;
  sourceForm.autoActivate = false;
  sourceForm.meta = '';
}

function handleAddSource() {
  sourceModalMode.value = 'add';
  resetSourceForm();
  applyDefaultSource();
  sourceModalVisible.value = true;
}

function buildAvailableSourceName(baseName: string) {
  const normalized = baseName.trim();
  if (!normalized) return baseName;

  const names = new Set(sourceTable.value.map(item => item.name));
  if (!names.has(normalized)) {
    return normalized;
  }

  let index = 2;
  let candidate = `${normalized}-${index}`;
  while (names.has(candidate)) {
    index += 1;
    candidate = `${normalized}-${index}`;
  }
  return candidate;
}

function applyDefaultSource() {
  sourceForm.kind = 'crs';
  sourceForm.mode = 'remote';
  sourceForm.authType = 'none';
  sourceForm.authSecret = '';
  sourceForm.enabled = true;
  sourceForm.autoCheck = true;
  sourceForm.autoDownload = true;
  sourceForm.autoActivate = false;

  sourceForm.name = buildAvailableSourceName('default-crs');
  sourceForm.url = 'https://codeload.github.com/coreruleset/coreruleset/tar.gz/refs/heads/main';
  sourceForm.checksumUrl = '';
  sourceForm.proxyUrl = '';
  sourceForm.schedule = '0 0 */6 * * *';
  sourceForm.meta = '{"default":true,"official":true,"repo":"https://github.com/coreruleset/coreruleset"}';
}

function handleEditSource(row: WafSourceItem) {
  sourceModalMode.value = 'edit';
  sourceForm.id = row.id;
  sourceForm.name = row.name;
  sourceForm.kind = row.kind;
  sourceForm.mode = row.mode;
  sourceForm.url = row.url;
  sourceForm.checksumUrl = row.checksumUrl;
  sourceForm.proxyUrl = row.proxyUrl || '';
  sourceForm.authType = row.authType;
  sourceForm.authSecret = '';
  sourceForm.schedule = row.schedule;
  sourceForm.enabled = row.enabled;
  sourceForm.autoCheck = row.autoCheck;
  sourceForm.autoDownload = row.autoDownload;
  sourceForm.autoActivate = row.autoActivate;
  sourceForm.meta = '';
  sourceModalVisible.value = true;
}

async function handleSubmitSource() {
  await sourceFormRef.value?.validate();
  sourceSubmitting.value = true;
  try {
    const payload = {
      name: sourceForm.name.trim(),
      kind: sourceForm.kind,
      mode: sourceForm.mode,
      url: sourceForm.url.trim(),
      checksumUrl: sourceForm.checksumUrl.trim(),
      proxyUrl: sourceForm.proxyUrl.trim(),
      authType: sourceForm.authType,
      authSecret: sourceForm.authSecret.trim(),
      schedule: sourceForm.schedule.trim(),
      enabled: sourceForm.enabled,
      autoCheck: sourceForm.autoCheck,
      autoDownload: sourceForm.autoDownload,
      autoActivate: sourceForm.autoActivate,
      meta: sourceForm.meta.trim()
    };

    const request =
      sourceModalMode.value === 'add'
        ? createWafSource(payload)
        : updateWafSource(sourceForm.id, payload);

    const { error } = await request;
    if (!error) {
      message.success(sourceModalMode.value === 'add' ? '新增更新源成功' : '更新更新源成功');
      sourceModalVisible.value = false;
      fetchSources();
    }
  } finally {
    sourceSubmitting.value = false;
  }
}

function handleDeleteSource(row: WafSourceItem) {
  deleteWafSource(row.id).then(({ error }) => {
    if (!error) {
      message.success('删除成功');
      fetchSources();
    }
  });
}

async function fetchPolicies() {
  policyLoading.value = true;
  try {
    const { data, error } = await fetchWafPolicyList({
      page: policyPagination.page as number,
      pageSize: policyPagination.pageSize as number,
      name: policyQuery.name.trim() || undefined
    });
    if (!error && data) {
      const list = data.list || [];
      const total = data.total || 0;

      if (!policyQuery.name.trim() && total > 0 && list.length === 0 && (policyPagination.page as number) > 1) {
        policyPagination.page = 1;
        await fetchPolicies();
        return;
      }

      policyTable.value = list;
      policyPagination.itemCount = total;
      syncCrsTuningFromPolicyTable();
    }
  } finally {
    policyLoading.value = false;
  }
}

function syncCrsTuningFromPolicy(policy: WafPolicyItem | null | undefined) {
  if (!policy) {
    crsTuningForm.policyId = 0;
    crsTuningForm.crsTemplate = 'low_fp';
    crsTuningForm.crsParanoiaLevel = 1;
    crsTuningForm.crsInboundAnomalyThreshold = 10;
    crsTuningForm.crsOutboundAnomalyThreshold = 8;
    return;
  }

  const crsParanoiaLevel = Number(policy.crsParanoiaLevel || 1);
  const crsInboundAnomalyThreshold = Number(policy.crsInboundAnomalyThreshold || 10);
  const crsOutboundAnomalyThreshold = Number(policy.crsOutboundAnomalyThreshold || 8);
  const inferredTemplate = inferCrsTemplateByValues(crsParanoiaLevel, crsInboundAnomalyThreshold, crsOutboundAnomalyThreshold);

  crsTuningForm.policyId = policy.id;
  crsTuningForm.crsParanoiaLevel = crsParanoiaLevel;
  crsTuningForm.crsInboundAnomalyThreshold = crsInboundAnomalyThreshold;
  crsTuningForm.crsOutboundAnomalyThreshold = crsOutboundAnomalyThreshold;
  crsTuningForm.crsTemplate = (policy.crsTemplate as WafPolicyCrsTemplate) || inferredTemplate;
}

function syncCrsTuningFromPolicyTable() {
  if (!policyTable.value.length) {
    syncCrsTuningFromPolicy(null);
    return;
  }

  const current = policyTable.value.find(item => item.id === crsTuningForm.policyId);
  if (current) {
    syncCrsTuningFromPolicy(current);
    return;
  }

  const preferred = policyTable.value.find(item => item.isDefault) || policyTable.value[0];
  syncCrsTuningFromPolicy(preferred);
}

function handleCrsPolicyChange(policyId: number | null) {
  const policy = policyTable.value.find(item => item.id === Number(policyId || 0));
  syncCrsTuningFromPolicy(policy);
  policyRevisionPagination.page = 1;
  fetchPolicyRevisions(getCurrentRevisionPolicyId());
}

function handleRefreshCrsPolicy() {
  fetchPolicies();
  fetchPolicyRevisions(getCurrentRevisionPolicyId());
}

function applyCrsTemplatePreset(template: Exclude<WafPolicyCrsTemplate, 'custom'>) {
  const preset = crsTemplatePresetMap[template];
  crsTuningForm.crsTemplate = template;
  crsTuningForm.crsParanoiaLevel = preset.crsParanoiaLevel;
  crsTuningForm.crsInboundAnomalyThreshold = preset.crsInboundAnomalyThreshold;
  crsTuningForm.crsOutboundAnomalyThreshold = preset.crsOutboundAnomalyThreshold;
}

function buildCrsTuningPayload() {
  const crsParanoiaLevel = Number(crsTuningForm.crsParanoiaLevel);
  const crsInboundAnomalyThreshold = Number(crsTuningForm.crsInboundAnomalyThreshold);
  const crsOutboundAnomalyThreshold = Number(crsTuningForm.crsOutboundAnomalyThreshold);
  const inferredTemplate = inferCrsTemplateByValues(
    crsParanoiaLevel,
    crsInboundAnomalyThreshold,
    crsOutboundAnomalyThreshold
  );

  return {
    crsTemplate: inferredTemplate,
    crsParanoiaLevel,
    crsInboundAnomalyThreshold,
    crsOutboundAnomalyThreshold
  };
}

function getCurrentCrsPolicy() {
  return policyTable.value.find(item => item.id === crsTuningForm.policyId) || null;
}

function hasPendingCrsTuningChanges() {
  const policy = getCurrentCrsPolicy();
  if (!policy) {
    return false;
  }

  const payload = buildCrsTuningPayload();
  const currentTemplate = inferCrsTemplateByValues(
    Number(policy.crsParanoiaLevel || 1),
    Number(policy.crsInboundAnomalyThreshold || 10),
    Number(policy.crsOutboundAnomalyThreshold || 8)
  );

  return (
    Number(payload.crsParanoiaLevel) !== Number(policy.crsParanoiaLevel) ||
    Number(payload.crsInboundAnomalyThreshold) !== Number(policy.crsInboundAnomalyThreshold) ||
    Number(payload.crsOutboundAnomalyThreshold) !== Number(policy.crsOutboundAnomalyThreshold) ||
    payload.crsTemplate !== currentTemplate
  );
}

async function persistCrsTuning(showSuccessMessage = true) {
  await crsTuningFormRef.value?.validate();
  if (!crsTuningForm.policyId) {
    message.warning('请先选择策略');
    return false;
  }

  const { error } = await updateWafPolicy(crsTuningForm.policyId, buildCrsTuningPayload());
  if (error) {
    return false;
  }

  if (showSuccessMessage) {
    message.success('CRS 调优参数已保存');
  }
  await fetchPolicies();
  await fetchPolicyRevisions(getCurrentRevisionPolicyId());
  return true;
}

async function handleSaveCrsTuning() {
  crsTuningSubmitting.value = true;
  try {
    await persistCrsTuning(true);
  } finally {
    crsTuningSubmitting.value = false;
  }
}

async function handlePreviewCrsTuning() {
  if (!crsTuningForm.policyId) {
    message.warning('请先选择策略');
    return;
  }
  if (hasPendingCrsTuningChanges()) {
    message.warning('当前调优参数尚未保存，请先点击“保存调优参数”');
    return;
  }

  const policy = getCurrentCrsPolicy();
  if (policy) {
    await handlePreviewPolicy(policy);
  }
}

async function handleValidateCrsTuning() {
  if (!crsTuningForm.policyId) {
    message.warning('请先选择策略');
    return;
  }
  if (hasPendingCrsTuningChanges()) {
    message.warning('当前调优参数尚未保存，请先点击“保存调优参数”');
    return;
  }

  const policy = getCurrentCrsPolicy();
  if (policy) {
    await handleValidatePolicy(policy);
  }
}

function handlePublishCrsTuning() {
  if (!crsTuningForm.policyId) {
    message.warning('请先选择策略');
    return;
  }

  const policy = policyTable.value.find(item => item.id === crsTuningForm.policyId);
  if (!policy) {
    message.warning('未找到对应策略，请先刷新');
    return;
  }

  const highRisk = Number(crsTuningForm.crsParanoiaLevel) >= 3;
  const content = highRisk
    ? `当前 PL=${crsTuningForm.crsParanoiaLevel}，误拦截风险较高。确认保存调优参数并发布策略 ${policy.name} 吗？`
    : `确认保存调优参数并发布策略 ${policy.name} 吗？`;

  dialog.warning({
    title: highRisk ? '高风险调优发布确认' : 'CRS 调优发布确认',
    content,
    positiveText: '确认发布',
    negativeText: '取消',
    async onPositiveClick() {
      crsTuningSubmitting.value = true;
      try {
        if (hasPendingCrsTuningChanges()) {
          const persisted = await persistCrsTuning(false);
          if (!persisted) {
            return;
          }
        }

        const { error } = await publishWafPolicy(crsTuningForm.policyId);
        if (!error) {
          message.success('CRS 调优参数发布成功');
          await fetchPolicies();
          await fetchPolicyRevisions(getCurrentRevisionPolicyId());
        }
      } finally {
        crsTuningSubmitting.value = false;
      }
    }
  });
}

function resetPolicyQuery() {
  policyQuery.name = '';
  policyPagination.page = 1;
  fetchPolicies();
}

function handlePolicyPageChange(page: number) {
  policyPagination.page = page;
  fetchPolicies();
}

function handlePolicyPageSizeChange(pageSize: number) {
  policyPagination.pageSize = pageSize;
  policyPagination.page = 1;
  fetchPolicies();
}

function resetPolicyForm() {
  policyForm.id = 0;
  policyForm.name = '';
  policyForm.description = '';
  policyForm.enabled = true;
  policyForm.isDefault = false;
  policyForm.engineMode = 'detectiononly';
  policyForm.auditEngine = 'relevantonly';
  policyForm.auditLogFormat = 'json';
  policyForm.auditRelevantStatus = '^(?:5|4(?!04))';
  policyForm.requestBodyAccess = true;
  policyForm.requestBodyLimit = 10 * 1024 * 1024;
  policyForm.requestBodyNoFilesLimit = 1024 * 1024;
  policyForm.config = '';
}

function handleAddPolicy() {
  policyModalMode.value = 'add';
  resetPolicyForm();
  policyModalVisible.value = true;
}

function handleEditPolicy(row: WafPolicyItem) {
  policyModalMode.value = 'edit';
  policyForm.id = row.id;
  policyForm.name = row.name;
  policyForm.description = row.description || '';
  policyForm.enabled = row.enabled;
  policyForm.isDefault = row.isDefault;
  policyForm.engineMode = row.engineMode;
  policyForm.auditEngine = row.auditEngine;
  policyForm.auditLogFormat = row.auditLogFormat;
  policyForm.auditRelevantStatus = row.auditRelevantStatus || '^(?:5|4(?!04))';
  policyForm.requestBodyAccess = row.requestBodyAccess;
  policyForm.requestBodyLimit = row.requestBodyLimit;
  policyForm.requestBodyNoFilesLimit = row.requestBodyNoFilesLimit;
  policyForm.config = row.config || '';
  policyModalVisible.value = true;
}

function buildPolicyPayload() {
  return {
    name: policyForm.name.trim(),
    description: policyForm.description.trim(),
    enabled: policyForm.enabled,
    isDefault: policyForm.isDefault,
    engineMode: policyForm.engineMode,
    auditEngine: policyForm.auditEngine,
    auditLogFormat: policyForm.auditLogFormat,
    auditRelevantStatus: policyForm.auditRelevantStatus.trim(),
    requestBodyAccess: policyForm.requestBodyAccess,
    requestBodyLimit: Number(policyForm.requestBodyLimit),
    requestBodyNoFilesLimit: Number(policyForm.requestBodyNoFilesLimit),
    config: policyForm.config.trim()
  };
}

async function handleSubmitPolicy() {
  await policyFormRef.value?.validate();
  policySubmitting.value = true;
  try {
    const payload = buildPolicyPayload();
    const request =
      policyModalMode.value === 'add' ? createWafPolicy(payload) : updateWafPolicy(policyForm.id, payload);

    const { error } = await request;
    if (!error) {
      message.success(policyModalMode.value === 'add' ? '策略创建成功' : '策略更新成功');
      policyModalVisible.value = false;
      await fetchPolicies();
      await fetchPolicyRevisions(getCurrentRevisionPolicyId());
    }
  } finally {
    policySubmitting.value = false;
  }
}

function handleDeletePolicy(row: WafPolicyItem) {
  deleteWafPolicy(row.id).then(async ({ error }) => {
    if (!error) {
      message.success('策略删除成功');
      if (policyPreviewPolicyName.value === row.name) {
        policyPreviewPolicyName.value = '';
        policyPreviewDirectives.value = '';
      }
      await fetchPolicies();
      await fetchPolicyRevisions(getCurrentRevisionPolicyId());
    }
  });
}

async function handlePreviewPolicy(row: WafPolicyItem) {
  policyPreviewLoading.value = true;
  try {
    const { data, error } = await previewWafPolicy(row.id);
    if (!error && data) {
      policyPreviewPolicyName.value = row.name;
      policyPreviewDirectives.value = data.directives || '';
      message.success('已生成策略预览');
    }
  } finally {
    policyPreviewLoading.value = false;
  }
}

async function handleValidatePolicy(row: WafPolicyItem) {
  const { error } = await validateWafPolicy(row.id);
  if (!error) {
    message.success(`策略 ${row.name} 校验通过`);
  }
}

function handlePublishPolicy(row: WafPolicyItem) {
  const isBlockingMode = row.engineMode === 'on';
  const highRiskParanoia = Number(row.crsParanoiaLevel || 0) >= 3;
  const warningParts: string[] = [];

  if (isBlockingMode) {
    warningParts.push('当前为 On（阻断）模式');
  }
  if (highRiskParanoia) {
    warningParts.push(`CRS PL=${row.crsParanoiaLevel}`);
  }

  dialog.warning({
    title: warningParts.length ? '高风险发布确认' : '发布确认',
    content: warningParts.length
      ? `策略 ${row.name} ${warningParts.join('，')}，发布后可能引发误拦截，确认继续发布吗？`
      : `确认发布策略 ${row.name} 吗？`,
    positiveText: '确认发布',
    negativeText: '取消',
    async onPositiveClick() {
      const { error } = await publishWafPolicy(row.id);
      if (!error) {
        message.success('策略发布成功');
        await fetchPolicies();
        await fetchPolicyRevisions(getCurrentRevisionPolicyId());
      }
    }
  });
}

function getCurrentRevisionPolicyId() {
  return activeTab.value === 'crs' ? crsTuningForm.policyId || undefined : undefined;
}

async function fetchPolicyRevisions(policyId?: number) {
  policyRevisionLoading.value = true;
  try {
    const { data, error } = await fetchWafPolicyRevisionList({
      page: policyRevisionPagination.page as number,
      pageSize: policyRevisionPagination.pageSize as number,
      policyId
    });
    if (!error && data) {
      policyRevisionTable.value = data.list || [];
      policyRevisionPagination.itemCount = data.total || 0;
    }
  } finally {
    policyRevisionLoading.value = false;
  }
}

function handlePolicyRevisionPageChange(page: number) {
  policyRevisionPagination.page = page;
  fetchPolicyRevisions(getCurrentRevisionPolicyId());
}

function handlePolicyRevisionPageSizeChange(pageSize: number) {
  policyRevisionPagination.pageSize = pageSize;
  policyRevisionPagination.page = 1;
  fetchPolicyRevisions(getCurrentRevisionPolicyId());
}

function handleRollbackPolicyRevision(row: WafPolicyRevisionItem) {
  dialog.warning({
    title: '策略回滚确认',
    content: `确认回滚到策略 ${row.policyId} 的版本 v${row.version} 吗？`,
    positiveText: '确认回滚',
    negativeText: '取消',
    async onPositiveClick() {
      const { error } = await rollbackWafPolicy({ revisionId: row.id });
      if (!error) {
        message.success('策略回滚成功');
        await fetchPolicies();
        await fetchPolicyRevisions(getCurrentRevisionPolicyId());
      }
    }
  });
}

function getDefaultPolicyId() {
  const preferred = policyTable.value.find(item => item.isDefault) || policyTable.value[0];
  return Number(preferred?.id || 0);
}

async function fetchExclusions() {
  exclusionLoading.value = true;
  try {
    const { data, error } = await fetchWafRuleExclusionList({
      page: exclusionPagination.page as number,
      pageSize: exclusionPagination.pageSize as number,
      policyId: exclusionQuery.policyId || undefined,
      scopeType: exclusionQuery.scopeType || undefined,
      name: exclusionQuery.name.trim() || undefined
    });
    if (!error && data) {
      exclusionTable.value = data.list || [];
      exclusionPagination.itemCount = data.total || 0;
    }
  } finally {
    exclusionLoading.value = false;
  }
}

function resetExclusionQuery() {
  exclusionQuery.policyId = null;
  exclusionQuery.scopeType = '';
  exclusionQuery.name = '';
  exclusionPagination.page = 1;
  fetchExclusions();
}

function handleExclusionPageChange(page: number) {
  exclusionPagination.page = page;
  fetchExclusions();
}

function handleExclusionPageSizeChange(pageSize: number) {
  exclusionPagination.pageSize = pageSize;
  exclusionPagination.page = 1;
  fetchExclusions();
}

function resetExclusionForm() {
  exclusionForm.id = 0;
  exclusionForm.policyId = getDefaultPolicyId();
  exclusionForm.name = '';
  exclusionForm.description = '';
  exclusionForm.enabled = true;
  exclusionForm.scopeType = 'global';
  exclusionForm.host = '';
  exclusionForm.path = '';
  exclusionForm.method = '';
  exclusionForm.removeType = 'id';
  exclusionForm.removeValue = '';
}

function handleAddExclusion() {
  exclusionModalMode.value = 'add';
  resetExclusionForm();
  exclusionModalVisible.value = true;
}

function buildExclusionDraftFromFeedback(row: WafPolicyFalsePositiveFeedbackItem): PolicyFeedbackExclusionDraft {
  const policyId = Number(row.policyId || 0) > 0 ? Number(row.policyId) : getDefaultPolicyId();
  const host = String(row.host || '').trim();
  const path = String(row.path || '').trim();
  const method = String(row.method || '').trim().toUpperCase() || '';
  const scopeType: WafPolicyScopeType = path ? 'route' : host ? 'site' : 'global';
  const candidates = collectExclusionCandidatesFromFeedbackSuggestion(row.suggestion || '');
  const parsed = parseExclusionFromFeedbackSuggestion(row.suggestion || '');
  const reason = String(row.reason || '').trim();
  const suggestion = String(row.suggestion || '').trim();

  return {
    feedbackId: Number(row.id || 0),
    policyId,
    policyName: mapPolicyNameById(policyId),
    name: `fp-${Number(row.id || 0) || Date.now()}`,
    description: suggestion ? `来源反馈#${row.id}：${reason}；建议：${suggestion}` : `来源反馈#${row.id}：${reason}`,
    scopeType,
    host,
    path,
    method,
    removeType: parsed.removeType,
    removeValue: parsed.removeValue,
    candidates
  };
}

function handleCreateExclusionDraftFromFeedback(row: WafPolicyFalsePositiveFeedbackItem) {
  const draft = buildExclusionDraftFromFeedback(row);
  policyFeedbackExclusionDraft.value = draft;
  policyFeedbackExclusionDraftCandidateKey.value =
    draft.removeValue ? buildExclusionCandidateKey(draft.removeType, draft.removeValue) : '';
  policyFeedbackExclusionDraftModalVisible.value = true;
}

function handlePolicyFeedbackExclusionCandidateChange(value: string) {
  const draft = policyFeedbackExclusionDraft.value;
  if (!draft) {
    return;
  }
  const selected = parseExclusionCandidateKey(value);
  if (!selected) {
    return;
  }
  draft.removeType = selected.removeType;
  draft.removeValue = selected.removeValue;
}

function handlePolicyFeedbackExclusionDraftScopeChange(scopeType: WafPolicyScopeType) {
  const draft = policyFeedbackExclusionDraft.value;
  if (!draft) {
    return;
  }
  draft.scopeType = scopeType;
  if (scopeType === 'global') {
    draft.host = '';
    draft.path = '';
    draft.method = '';
  } else if (scopeType === 'site') {
    draft.path = '';
    draft.method = '';
  }
}

function handleConfirmPolicyFeedbackExclusionDraft() {
  const draft = policyFeedbackExclusionDraft.value;
  if (!draft) {
    message.warning('例外草稿为空');
    return;
  }
  if (!Number(draft.policyId || 0)) {
    message.warning('请选择关联策略');
    return;
  }
  if (draft.scopeType === 'site' && !String(draft.host || '').trim()) {
    message.warning('站点作用域必须填写 Host');
    return;
  }
  if (draft.scopeType === 'route' && !String(draft.path || '').trim()) {
    message.warning('路由作用域必须填写 Path');
    return;
  }
  if (!String(draft.name || '').trim()) {
    message.warning('请填写规则名称');
    return;
  }

  exclusionModalMode.value = 'add';
  resetExclusionForm();
  exclusionForm.policyId = Number(draft.policyId);
  exclusionForm.name = String(draft.name || '').trim();
  exclusionForm.description = draft.description;
  exclusionForm.scopeType = draft.scopeType;
  exclusionForm.host = String(draft.host || '').trim();
  exclusionForm.path = String(draft.path || '').trim();
  exclusionForm.method = String(draft.method || '').trim().toUpperCase();
  exclusionForm.removeType = draft.removeType;
  exclusionForm.removeValue = String(draft.removeValue || '').trim();

  policyFeedbackExclusionDraftModalVisible.value = false;
  policyFeedbackExclusionDraft.value = null;
  policyFeedbackExclusionDraftCandidateKey.value = '';
  activeTab.value = 'exclusion';
  shouldFocusExclusionRemoveValue.value = !exclusionForm.removeValue;
  exclusionModalVisible.value = true;
  if (!exclusionForm.removeValue) {
    message.warning('已生成例外草稿，请补充移除值（removeById / removeByTag）后保存');
  } else {
    message.success('已根据误报反馈生成例外草稿');
  }
}

function handleEditExclusion(row: WafRuleExclusionItem) {
  exclusionModalMode.value = 'edit';
  exclusionForm.id = row.id;
  exclusionForm.policyId = row.policyId;
  exclusionForm.name = row.name || '';
  exclusionForm.description = row.description || '';
  exclusionForm.enabled = row.enabled;
  exclusionForm.scopeType = row.scopeType;
  exclusionForm.host = row.host || '';
  exclusionForm.path = row.path || '';
  exclusionForm.method = row.method || '';
  exclusionForm.removeType = row.removeType;
  exclusionForm.removeValue = row.removeValue || '';
  exclusionModalVisible.value = true;
}

function buildExclusionPayload(): WafRuleExclusionPayload {
  return {
    policyId: Number(exclusionForm.policyId),
    name: exclusionForm.name.trim(),
    description: exclusionForm.description.trim(),
    enabled: exclusionForm.enabled,
    scopeType: exclusionForm.scopeType,
    host: exclusionForm.host.trim(),
    path: exclusionForm.path.trim(),
    method: String(exclusionForm.method || '').trim(),
    removeType: exclusionForm.removeType,
    removeValue: exclusionForm.removeValue.trim()
  };
}

async function handleSubmitExclusion() {
  await exclusionFormRef.value?.validate();
  exclusionSubmitting.value = true;
  try {
    const payload = buildExclusionPayload();
    const request =
      exclusionModalMode.value === 'add'
        ? createWafRuleExclusion(payload)
        : updateWafRuleExclusion(exclusionForm.id, payload);
    const { error } = await request;
    if (!error) {
      message.success(exclusionModalMode.value === 'add' ? '规则例外创建成功' : '规则例外更新成功');
      exclusionModalVisible.value = false;
      fetchExclusions();
    }
  } finally {
    exclusionSubmitting.value = false;
  }
}

function handleDeleteExclusion(row: WafRuleExclusionItem) {
  deleteWafRuleExclusion(row.id).then(({ error }) => {
    if (!error) {
      message.success('规则例外删除成功');
      fetchExclusions();
    }
  });
}

async function fetchBindings() {
  bindingLoading.value = true;
  try {
    const { data, error } = await fetchWafPolicyBindingList({
      page: bindingPagination.page as number,
      pageSize: bindingPagination.pageSize as number,
      policyId: bindingQuery.policyId || undefined,
      scopeType: bindingQuery.scopeType || undefined,
      name: bindingQuery.name.trim() || undefined
    });
    if (!error && data) {
      bindingTable.value = data.list || [];
      bindingPagination.itemCount = data.total || 0;
    }
  } finally {
    bindingLoading.value = false;
  }
}

function resetBindingQuery() {
  bindingQuery.policyId = null;
  bindingQuery.scopeType = '';
  bindingQuery.name = '';
  bindingPagination.page = 1;
  fetchBindings();
}

function handleBindingPageChange(page: number) {
  bindingPagination.page = page;
  fetchBindings();
}

function handleBindingPageSizeChange(pageSize: number) {
  bindingPagination.pageSize = pageSize;
  bindingPagination.page = 1;
  fetchBindings();
}

function resetBindingForm() {
  bindingForm.id = 0;
  bindingForm.policyId = getDefaultPolicyId();
  bindingForm.name = '';
  bindingForm.description = '';
  bindingForm.enabled = true;
  bindingForm.scopeType = 'global';
  bindingForm.host = '';
  bindingForm.path = '';
  bindingForm.method = '';
  bindingForm.priority = 100;
}

function handleAddBinding() {
  bindingModalMode.value = 'add';
  resetBindingForm();
  bindingModalVisible.value = true;
}

function handleEditBinding(row: WafPolicyBindingItem) {
  bindingModalMode.value = 'edit';
  bindingForm.id = row.id;
  bindingForm.policyId = row.policyId;
  bindingForm.name = row.name || '';
  bindingForm.description = row.description || '';
  bindingForm.enabled = row.enabled;
  bindingForm.scopeType = row.scopeType;
  bindingForm.host = row.host || '';
  bindingForm.path = row.path || '';
  bindingForm.method = row.method || '';
  bindingForm.priority = row.priority;
  bindingModalVisible.value = true;
}

function buildBindingPayload(): WafPolicyBindingPayload {
  return {
    policyId: Number(bindingForm.policyId),
    name: bindingForm.name.trim(),
    description: bindingForm.description.trim(),
    enabled: bindingForm.enabled,
    scopeType: bindingForm.scopeType,
    host: bindingForm.host.trim(),
    path: bindingForm.path.trim(),
    method: String(bindingForm.method || '').trim(),
    priority: Number(bindingForm.priority)
  };
}

async function handleSubmitBinding() {
  await bindingFormRef.value?.validate();
  bindingSubmitting.value = true;
  try {
    const payload = buildBindingPayload();
    const request =
      bindingModalMode.value === 'add'
        ? createWafPolicyBinding(payload)
        : updateWafPolicyBinding(bindingForm.id, payload);
    const { error } = await request;
    if (!error) {
      message.success(bindingModalMode.value === 'add' ? '策略绑定创建成功' : '策略绑定更新成功');
      bindingModalVisible.value = false;
      fetchBindings();
    }
  } finally {
    bindingSubmitting.value = false;
  }
}

function handleDeleteBinding(row: WafPolicyBindingItem) {
  deleteWafPolicyBinding(row.id).then(({ error }) => {
    if (!error) {
      message.success('策略绑定删除成功');
      fetchBindings();
    }
  });
}

function handleSyncSource(row: WafSourceItem, activateNow: boolean) {
  const allowActivate = activateNow;
  const content = allowActivate ? '将下载、校验并立即激活该源对应版本，确认继续？' : '将下载并校验该源对应版本，确认继续？';

  dialog.warning({
    title: allowActivate ? '同步并激活确认' : '同步确认',
    content,
    positiveText: '确认',
    negativeText: '取消',
    async onPositiveClick() {
      const { error } = await syncWafSource(row.id, allowActivate);
      if (!error) {
        message.success(allowActivate ? '同步并激活成功' : '同步成功');
        fetchSources();
        fetchReleases();
        if (activeTab.value === 'job') {
          fetchJobs();
        }
      } else {
        const backendMsg = (error as any)?.response?.data?.msg;
        const rawMessage = String(backendMsg || error.message || '');
        if (rawMessage.includes('context deadline exceeded')) {
          message.error('同步超时：请配置代理后重试，或稍后再试');
        }
      }
    }
  });
}

async function fetchReleases() {
  releaseLoading.value = true;
  try {
    const queryKind: WafKind = 'crs';
    const { data, error } = await fetchWafReleaseList({
      page: releasePagination.page as number,
      pageSize: releasePagination.pageSize as number,
      kind: queryKind,
      status: releaseQuery.status
    });
    if (!error && data) {
      const list = data.list || [];
      await ensureSourceNamesByIds(list.map(item => Number(item.sourceId || 0)));
      releaseTable.value = list;
      releasePagination.itemCount = data.total || 0;
    }
  } finally {
    releaseLoading.value = false;
  }
}

function resetReleaseQuery() {
  releaseQuery.status = '';
  releasePagination.page = 1;
  fetchReleases();
}

function handleReleasePageChange(page: number) {
  releasePagination.page = page;
  fetchReleases();
}

function handleReleasePageSizeChange(pageSize: number) {
  releasePagination.pageSize = pageSize;
  releasePagination.page = 1;
  fetchReleases();
}

function handleActivateRelease(row: WafReleaseItem) {
  dialog.warning({
    title: '激活确认',
    content: `确认激活版本 ${row.version} 吗？`,
    positiveText: '确认',
    negativeText: '取消',
    async onPositiveClick() {
      const { error } = await activateWafRelease(row.id);
      if (!error) {
        message.success('激活已提交');
        fetchReleases();
        fetchJobs();
      }
    }
  });
}

function handleClearReleases() {
  dialog.warning({
    title: '清空确认',
    content: '将清空版本发布管理中所有非激活的 CRS 版本（含文件目录），确认继续？',
    positiveText: '确认清空',
    negativeText: '取消',
    async onPositiveClick() {
      const { error } = await clearWafReleases({ kind: 'crs' });
      if (!error) {
        message.success('已清空非激活版本');
        fetchReleases();
        fetchJobs();
      }
    }
  });
}

function openRollbackModal() {
  rollbackForm.target = 'last_good';
  rollbackForm.version = '';
  rollbackModalVisible.value = true;
}

async function handleSubmitRollback() {
  await rollbackFormRef.value?.validate();
  rollbackSubmitting.value = true;
  try {
    const payload =
      rollbackForm.target === 'version'
        ? { target: 'version' as const, version: rollbackForm.version.trim() }
        : { target: 'last_good' as const };

    const { error } = await rollbackWafRelease(payload);
    if (!error) {
      message.success('回滚任务已提交');
      rollbackModalVisible.value = false;
      fetchReleases();
      fetchJobs();
    }
  } finally {
    rollbackSubmitting.value = false;
  }
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
  () => route.query,
  query => {
    if (observeRouteSyncing.value) {
      return;
    }
    const prevTab = activeTab.value;
    const queryChanged = applyObserveQueryFromRoute(query as Record<string, unknown>);
    if (queryChanged && prevTab === 'observe' && activeTab.value === 'observe') {
      fetchPolicyStats();
    }
  },
  { immediate: true }
);

watch(
  () => [
    activeTab.value,
    policyStatsQuery.policyId,
    policyStatsQuery.window,
    policyStatsQuery.intervalSec,
    policyStatsQuery.topN,
    policyStatsQuery.host,
    policyStatsQuery.path,
    policyStatsQuery.method
  ],
  () => {
    if (observeRouteSyncing.value) {
      return;
    }
    void syncObserveStateToRouteQuery();
  }
);

watch(
  () => sourceForm.mode,
  value => {
    if (value !== 'remote') {
      sourceForm.proxyUrl = '';
    }
  }
);

watch(
  () => [crsTuningForm.crsParanoiaLevel, crsTuningForm.crsInboundAnomalyThreshold, crsTuningForm.crsOutboundAnomalyThreshold],
  values => {
    const [crsParanoiaLevel, crsInboundAnomalyThreshold, crsOutboundAnomalyThreshold] = values.map(value => Number(value));
    if (!Number.isFinite(crsParanoiaLevel) || !Number.isFinite(crsInboundAnomalyThreshold) || !Number.isFinite(crsOutboundAnomalyThreshold)) {
      return;
    }
    crsTuningForm.crsTemplate = inferCrsTemplateByValues(crsParanoiaLevel, crsInboundAnomalyThreshold, crsOutboundAnomalyThreshold);
  }
);

watch(
  () => exclusionForm.scopeType,
  value => {
    if (value === 'global') {
      exclusionForm.host = '';
      exclusionForm.path = '';
      exclusionForm.method = '';
    } else if (value === 'site') {
      exclusionForm.path = '';
      exclusionForm.method = '';
    }
  }
);

watch(exclusionModalVisible, value => {
  if (!value || !shouldFocusExclusionRemoveValue.value) {
    return;
  }
  nextTick(() => {
    exclusionRemoveValueInputRef.value?.focus();
    shouldFocusExclusionRemoveValue.value = false;
  });
});

watch(
  () => bindingForm.scopeType,
  value => {
    if (value === 'global') {
      bindingForm.host = '';
      bindingForm.path = '';
      bindingForm.method = '';
    } else if (value === 'site') {
      bindingForm.path = '';
      bindingForm.method = '';
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
      fetchReleases();
      fetchJobs();
    }
  } finally {
    uploadSubmitting.value = false;
  }
}

async function fetchJobs() {
  jobLoading.value = true;
  try {
    const { data, error } = await fetchWafJobList({
      page: jobPagination.page as number,
      pageSize: jobPagination.pageSize as number,
      status: jobQuery.status,
      action: jobQuery.action || undefined
    });
    if (!error && data) {
      const list = data.list || [];
      await ensureSourceNamesByIds(list.map(item => Number(item.sourceId || 0)));
      jobTable.value = list;
      jobPagination.itemCount = data.total || 0;
    }
  } finally {
    jobLoading.value = false;
  }
}

function resetJobQuery() {
  jobQuery.status = '';
  jobQuery.action = '';
  jobPagination.page = 1;
  fetchJobs();
}

function handleJobPageChange(page: number) {
  jobPagination.page = page;
  fetchJobs();
}

function handleJobPageSizeChange(pageSize: number) {
  jobPagination.pageSize = pageSize;
  jobPagination.page = 1;
  fetchJobs();
}

function handleClearJobs() {
  dialog.warning({
    title: '清空确认',
    content: '将清空全部任务日志记录，确认继续？',
    positiveText: '确认清空',
    negativeText: '取消',
    async onPositiveClick() {
      const { error } = await clearWafJobs();
      if (!error) {
        message.success('任务日志已清空');
        fetchJobs();
      }
    }
  });
}

function refreshCurrentTab() {
  if (activeTab.value === 'source') {
    fetchSources();
    return;
  }
  if (activeTab.value === 'runtime') {
    fetchPolicies();
    fetchPolicyRevisions();
    return;
  }
  if (activeTab.value === 'crs') {
    fetchPolicies();
    fetchPolicyRevisions(getCurrentRevisionPolicyId());
    return;
  }
  if (activeTab.value === 'exclusion') {
    fetchPolicies();
    fetchExclusions();
    return;
  }
  if (activeTab.value === 'binding') {
    fetchPolicies();
    fetchBindings();
    return;
  }
  if (activeTab.value === 'observe') {
    fetchPolicies();
    fetchPolicyStats();
    return;
  }
  if (activeTab.value === 'release') {
    fetchReleases();
    return;
  }
  fetchJobs();
}

watch(activeTab, value => {
  if (value === 'source') {
    fetchSources();
  } else if (value === 'runtime') {
    fetchPolicies();
    fetchPolicyRevisions();
  } else if (value === 'crs') {
    fetchPolicies();
    fetchPolicyRevisions(getCurrentRevisionPolicyId());
  } else if (value === 'exclusion') {
    fetchPolicies();
    fetchExclusions();
  } else if (value === 'binding') {
    fetchPolicies();
    fetchBindings();
  } else if (value === 'observe') {
    fetchPolicies();
    fetchPolicyStats();
  } else if (value === 'release') {
    fetchReleases();
  } else {
    fetchJobs();
  }
});

onMounted(() => {
  fetchEngineStatus();
  refreshCurrentTab();
});
</script>

<style scoped>
:deep(.n-data-table .n-data-table-th__title) {
  white-space: nowrap;
}
</style>
