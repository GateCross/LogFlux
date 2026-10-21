<script lang="ts" setup>
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  DescriptionsItem,
  Drawer,
  Form,
  FormItem,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  TabPane,
  Tabs,
  Tag,
} from 'antdv-next';

import type { CaddySimpleWafApi } from '#/api/caddy/simple-waf';
import type {
  WafFormState,
  WafLoadState,
} from '../composables/useSimpleWaf';
import { WAF_REQUEST_BODY_MAX_MB } from '../composables/useSimpleWaf';
import type { DiffRow } from '../caddy-config-diff';
import { diffSideClass } from '../composables/useConfigHistory';

/** 站点在"已生效 / 待应用 / 将移除"三态中的归属 */
type SiteState = 'active' | 'pending' | 'remove';

const props = defineProps<{
  wafLoading: boolean;
  wafSaving: boolean;
  wafApplying: boolean;
  wafPreviewing: boolean;
  wafStatusText: string;
  wafStatusColor: string;
  /** 策略已开启但尚未挂载到 Caddyfile（草稿态） */
  wafDraftPending?: boolean;
  wafStatus?: CaddySimpleWafApi.SimpleWafConfig | null;
  /** 读失败页内错误 */
  wafErrorMessage?: string | null;
  /** 显式加载态：非 ready 时禁止写操作 */
  wafLoadState: WafLoadState;
  /** 状态不属于当前服务器或读失败时为 true */
  wafActionsDisabled: boolean;
  /** 表单相对最近一次读库快照有未保存改动 */
  wafDirty?: boolean;
  wafForm: WafFormState;
  wafAvailableSites: string[];
  /** 站点三态（已生效/待应用/将移除），由状态层计算 */
  wafSiteStates?: Array<{ site: string; state: SiteState }>;
  wafModeOptions: Array<{ label: string; value: CaddySimpleWafApi.SimpleWafMode }>;
  wafStrengthOptions: Array<{ label: string; value: CaddySimpleWafApi.SimpleWafStrength }>;
  wafAuditOptions: Array<{ label: string; value: CaddySimpleWafApi.SimpleWafAudit }>;
  wafPreviewVisible: boolean;
  wafPreviewResult: CaddySimpleWafApi.SimpleWafConfig | null;
  /** 候选 Caddyfile 与当前配置的逐行差异 */
  wafPreviewDiff?: DiffRow[];
  /** 表单校验错误；有值时提示并阻止提交 */
  wafValidationError?: null | string;
  /** 最近一次成功应用的结果 */
  wafLastApply?: { actions: string[]; at: string; message: string } | null;
}>();

const emit = defineEmits<{
  'update:wafPreviewVisible': [value: boolean];
  save: [];
  preview: [];
  apply: [];
  refresh: [];
}>();

const router = useRouter();
const rulesDrawerVisible = ref(false);

const siteStateMeta: Record<SiteState, { color: string; label: string }> = {
  active: { color: 'green', label: '已生效' },
  pending: { color: 'orange', label: '待应用' },
  remove: { color: 'default', label: '将移除' },
};

/** 强度直接决定启用的 CRS 规则层数与误报面，必须把代价说清楚。 */
const strengthHints: Record<string, string> = {
  low_fp: '仅启用 PL1 规则（约 213 条），误报最少，适合直接上阻断。',
  balanced: '启用 PL1~PL2（约 302 条），拦截更全，误报会明显增加。',
  high_blocking: '启用 PL1~PL3（约 332 条），拦截最严，建议先「仅检测」观察。',
};

const strengthHint = computed(
  () => strengthHints[props.wafForm.strength] ?? '',
);

/** 差异行数（不含未变化行），用于抽屉标题提示改动规模。 */
const previewChangeCount = computed(
  () => (props.wafPreviewDiff ?? []).filter((row) => row.type !== 'same').length,
);

function goToWafLogs() {
  router.push('/logs/access?onlyWaf=true');
}

function goToSourceManage() {
  router.push('/caddy/source');
}

const ruleColumns = [
  { title: '规则文件', dataIndex: 'file', key: 'file', width: 320 },
  { title: '安全分类', dataIndex: 'category', key: 'category', width: 160 },
];
</script>

<template>
  <div class="waf-pane">
    <Spin :spinning="props.wafLoading">
      <Alert
        v-if="props.wafErrorMessage"
        class="mb-3"
        type="error"
        show-icon
        :message="props.wafErrorMessage"
      >
        <template #description>
          <Space>
            <span>当前服务器状态未知，写操作已禁用，避免用默认值覆盖线上策略。</span>
            <Button size="small" @click="emit('refresh')">重试</Button>
          </Space>
        </template>
      </Alert>

      <!-- WAF 配置 + 已挂载规则概览 -->
      <Card title="WAF 防护配置" size="small">
        <template #extra>
          <Space>
            <Tag :color="props.wafStatusColor">{{ props.wafStatusText }}</Tag>
            <Tag v-if="props.wafDraftPending" color="warning">未生效</Tag>
            <Button size="small" @click="rulesDrawerVisible = true">挂载规则清单</Button>
            <Button size="small" type="primary" @click="goToWafLogs">WAF 防护日志</Button>
          </Space>
        </template>

        <!-- 后端接入诊断：说明为什么"未生效"以及下一步该做什么 -->
        <Alert
          v-if="props.wafStatus?.message && !props.wafStatus?.integrated"
          class="mb-3"
          type="warning"
          show-icon
          message="WAF 尚未完整接入"
          :description="props.wafStatus.message"
        />

        <!-- 应用结果留痕：toast 消失后仍能看到改了哪些站点 -->
        <Alert
          v-if="props.wafLastApply"
          class="mb-3"
          type="success"
          show-icon
          :message="`${props.wafLastApply.message}（${props.wafLastApply.at}）`"
        >
          <template v-if="props.wafLastApply.actions.length > 0" #description>
            <Space wrap size="small">
              <Tag v-for="item in props.wafLastApply.actions" :key="item">{{ item }}</Tag>
            </Space>
          </template>
        </Alert>

        <div class="flex flex-wrap items-center gap-4 mb-2.5 text-sm">
          <div>
            <span class="text-gray-500">CRS 版本：</span>
            <Tag color="blue">{{ props.wafStatus?.crsVersion || '-' }}</Tag>
          </div>
          <div>
            <span class="text-gray-500">Coraza 引擎：</span>
            <Tag color="cyan">{{ props.wafStatus?.corazaVersion || '-' }}</Tag>
          </div>
          <div>
            <span class="text-gray-500">已挂载规则文件：</span>
            <Tag color="purple">{{ props.wafStatus?.rulesInfo?.total || 0 }} 个</Tag>
          </div>
        </div>

        <div
          v-if="props.wafStatus?.rulesInfo?.categories?.length"
          class="bg-gray-50 dark:bg-gray-900 p-2.5 mb-3 rounded border border-gray-100 dark:border-gray-800"
        >
          <Space wrap size="small">
            <Tag v-for="cat in props.wafStatus.rulesInfo.categories" :key="cat" color="geekblue">
              {{ cat }}
            </Tag>
          </Space>
        </div>

        <Form layout="vertical">
          <Row :gutter="16">
            <Col :xs="24" :md="8">
              <FormItem label="启用 WAF">
                <Switch
                  v-model:checked="props.wafForm.enabled"
                  :disabled="props.wafActionsDisabled"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="8">
              <FormItem label="引擎模式">
                <Select
                  v-model:value="props.wafForm.mode"
                  :disabled="props.wafActionsDisabled || !props.wafForm.enabled"
                  :options="props.wafModeOptions"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="8">
              <FormItem label="规则强度">
                <Select
                  v-model:value="props.wafForm.strength"
                  :disabled="props.wafActionsDisabled || !props.wafForm.enabled"
                  :options="props.wafStrengthOptions"
                />
                <div v-if="strengthHint" class="hint-text">{{ strengthHint }}</div>
              </FormItem>
            </Col>
            <Col :xs="24" :md="8">
              <FormItem label="审计日志">
                <Select
                  v-model:value="props.wafForm.audit"
                  :disabled="props.wafActionsDisabled || !props.wafForm.enabled"
                  :options="props.wafAuditOptions"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="8">
              <FormItem label="请求体检测">
                <Switch
                  v-model:checked="props.wafForm.requestBodyAccess"
                  :disabled="props.wafActionsDisabled || !props.wafForm.enabled"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="4">
              <FormItem label="请求体 MB">
                <InputNumber
                  v-model:value="props.wafForm.requestBodyLimitMB"
                  :min="1"
                  :max="WAF_REQUEST_BODY_MAX_MB"
                  :disabled="props.wafActionsDisabled || !props.wafForm.enabled"
                  class="w-full"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="4">
              <FormItem label="无文件 MB">
                <InputNumber
                  v-model:value="props.wafForm.requestBodyNoFilesLimitMB"
                  :min="1"
                  :max="WAF_REQUEST_BODY_MAX_MB"
                  :disabled="props.wafActionsDisabled || !props.wafForm.enabled"
                  class="w-full"
                />
              </FormItem>
            </Col>
            <Col v-if="props.wafValidationError" :xs="24">
              <Alert class="mb-3" type="error" show-icon :message="props.wafValidationError" />
            </Col>
            <Col :xs="24">
              <FormItem label="应用站点">
                <Select
                  v-model:value="props.wafForm.siteAddresses"
                  :options="props.wafAvailableSites.map((item: string) => ({ label: item, value: item }))"
                  :disabled="props.wafActionsDisabled || !props.wafForm.enabled"
                  mode="multiple"
                  placeholder="默认应用到所有可用站点"
                />
                <div v-if="(props.wafSiteStates?.length ?? 0) > 0" class="site-state-list mt-2">
                  <span
                    v-for="item in props.wafSiteStates"
                    :key="item.site"
                    class="site-state-item"
                  >
                    <span class="site-state-name">{{ item.site }}</span>
                    <Tag :color="siteStateMeta[item.state].color">
                      {{ siteStateMeta[item.state].label }}
                    </Tag>
                  </span>
                </div>
              </FormItem>
            </Col>
          </Row>
          <Space>
            <Button
              :loading="props.wafSaving"
              :disabled="props.wafActionsDisabled"
              @click="emit('save')"
            >
              保存草稿
            </Button>
            <Button
              :loading="props.wafPreviewing"
              :disabled="props.wafActionsDisabled"
              @click="emit('preview')"
            >
              预览变更
            </Button>
            <Button
              type="primary"
              :loading="props.wafApplying"
              :disabled="props.wafActionsDisabled"
              @click="emit('apply')"
            >
              应用到 Caddy
            </Button>
            <Button @click="emit('refresh')">刷新</Button>
            <Tag v-if="props.wafDirty" color="warning">有未保存修改</Tag>
          </Space>
          <div class="hint-text mt-2">
            保存草稿只写策略，不改 Caddyfile；点「应用到 Caddy」才会挂载到所选站点。
          </div>
        </Form>
      </Card>

    </Spin>

    <!-- 挂载规则清单 Drawer -->
    <Drawer
      :open="rulesDrawerVisible"
      title="当前挂载的 OWASP CRS 防护规则清单"
      :size="800"
      @update:open="rulesDrawerVisible = $event"
    >
      <div class="mb-3 text-sm text-gray-500">
        共挂载 <strong>{{ props.wafStatus?.rulesInfo?.total || 0 }}</strong> 个规则模块文件，按防护等级（Paranoia Level）与异常评分机制（Anomaly Scoring）执行实时流量分析。
      </div>
      <Alert
        v-if="!props.wafStatus?.rulesInfo?.total"
        class="mb-3"
        type="warning"
        show-icon
        message="尚未挂载任何 CRS 规则文件"
      >
        <template #description>
          <div class="flex items-center gap-3">
            <span>规则目录为空，WAF 实际不会拦截流量。请到来源管理下载规则包并激活。</span>
            <Button size="small" type="primary" @click="goToSourceManage">去同步规则</Button>
          </div>
        </template>
      </Alert>
      <Table
        :columns="ruleColumns"
        :data-source="props.wafStatus?.rulesInfo?.rules || []"
        :pagination="false"
        row-key="file"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'file'">
            <code class="text-xs bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded">{{ record.file }}</code>
          </template>
          <template v-else-if="column.key === 'category'">
            <Tag color="geekblue">{{ record.category }}</Tag>
          </template>
        </template>
      </Table>
    </Drawer>

    <!-- 变更预览 Drawer -->
    <Drawer
      :open="props.wafPreviewVisible"
      title="WAF 变更预览"
      :size="900"
      @update:open="emit('update:wafPreviewVisible', $event)"
    >
      <Descriptions v-if="props.wafPreviewResult" bordered size="small" class="mb-3">
        <DescriptionsItem label="状态">{{ props.wafPreviewResult.message || '已生成预览' }}</DescriptionsItem>
        <DescriptionsItem label="Coraza">{{ props.wafPreviewResult.corazaVersion || '-' }}</DescriptionsItem>
        <DescriptionsItem label="CRS">{{ props.wafPreviewResult.crsVersion || '-' }}</DescriptionsItem>
      </Descriptions>
      <Space v-if="props.wafPreviewResult?.actions?.length" wrap class="mb-3">
        <Tag v-for="item in props.wafPreviewResult.actions" :key="item">{{ item }}</Tag>
      </Space>

      <Tabs v-if="props.wafPreviewDiff?.length" size="small">
        <TabPane key="diff" :tab="`Caddyfile 差异（${previewChangeCount} 行）`">
          <div class="diff-view">
            <div
              v-for="row in props.wafPreviewDiff"
              :key="row.key"
              class="diff-line-row"
              :class="diffSideClass(row, 'right')"
            >
              <span class="diff-no">{{ row.rightNo ?? '' }}</span>
              <span class="diff-line">{{ row.right ?? '' }}</span>
            </div>
          </div>
        </TabPane>
        <TabPane key="directives" tab="Coraza 指令">
          <Input.TextArea
            :value="props.wafPreviewResult?.directives || ''"
            :auto-size="{ minRows: 18, maxRows: 28 }"
            class="code-textarea"
            readonly
          />
        </TabPane>
      </Tabs>
      <Input.TextArea
        v-else
        :value="props.wafPreviewResult?.directives || props.wafPreviewResult?.config || ''"
        :auto-size="{ minRows: 18, maxRows: 28 }"
        class="code-textarea"
        readonly
      />
    </Drawer>
  </div>
</template>

<style scoped>
.w-full {
  width: 100%;
}

.code-textarea {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 13px;
}

.mb-3 {
  margin-bottom: 12px;
}

.hint-text {
  font-size: 12px;
  color: #8c8c8c;
}

.site-state-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 12px;
}

.site-state-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}

.site-state-name {
  color: #8c8c8c;
}

.diff-view {
  max-height: 60vh;
  overflow: auto;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
}

.diff-line-row {
  display: grid;
  grid-template-columns: 40px 1fr;
  gap: 8px;
  padding: 4px 8px;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  border-left: 3px solid transparent;
}

.diff-line-row.added {
  color: hsl(var(--success));
  background: hsl(var(--success) / 12%);
  border-left-color: hsl(var(--success));
}

.diff-line-row.removed {
  color: hsl(var(--destructive));
  background: hsl(var(--destructive) / 12%);
  border-left-color: hsl(var(--destructive));
}

.diff-line-row.blank {
  color: transparent;
  background: hsl(var(--muted) / 40%);
}

.diff-no {
  color: hsl(var(--muted-foreground) / 80%);
  text-align: right;
  user-select: none;
}

.diff-line {
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
