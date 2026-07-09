<script lang="ts" setup>
import { computed, reactive, ref, watch } from 'vue';

import {
  Alert,
  Button,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Steps,
  Switch,
  Tag,
} from 'ant-design-vue';

import type { QuickSiteDraft } from './quick-config-utils';
import {
  buildQuickSiteDraftFromWizard,
  buildWizardLocalCaddyfilePreview,
  createSiteWizardState,
  SITE_WIZARD_STEPS,
  type SiteWizardState,
  type SiteWizardStep,
  validateWizardStep,
  validateWizardUpTo,
  wizardAutoLoadEnabled,
  wizardStepAt,
  wizardStepIndex,
} from './site-wizard-utils';

const props = defineProps<{
  open: boolean;
  /** 当前工作台 form model，用于合并预览 */
  baseFormModel?: import('./types').CaddyFormModel;
  previewing?: boolean;
  applying?: boolean;
  /** 是否已选择服务器（服务端 Preview/Apply 需要） */
  hasServer?: boolean;
  upstreamPoolOptions?: Array<{ label: string; value: string }>;
}>();
const emit = defineEmits<{
  'update:open': [value: boolean];
  /** 仅写入草稿，不调用 /load */
  'commit-draft': [draft: QuickSiteDraft, meta: { waf: SiteWizardState['waf'] }];
  /** 请求 Preview（仅 /adapt） */
  preview: [draft: QuickSiteDraft];
  /** 用户确认后走 Apply_Path */
  apply: [draft: QuickSiteDraft, meta: { waf: SiteWizardState['waf'] }];
}>();

const currentStep = ref(0);
const stepErrors = ref<string[]>([]);
const formState = reactive<SiteWizardState>(createSiteWizardState());

const tlsModeOptions = [
  { label: '自动 HTTPS', value: 'auto' },
  { label: '内部证书', value: 'internal' },
  { label: '关闭 TLS', value: 'off' },
];

const lbPolicyOptions = [
  { label: '轮询', value: 'round_robin' },
  { label: '最少连接', value: 'least_conn' },
  { label: 'IP Hash', value: 'ip_hash' },
];

const wafModeOptions = [
  { label: '仅检测', value: 'detectiononly' },
  { label: '阻断', value: 'on' },
];

const wafStrengthOptions = [
  { label: '低误报', value: 'low_fp' },
  { label: '平衡', value: 'balanced' },
  { label: '严格', value: 'high_blocking' },
];

const wafAuditOptions = [
  { label: '相关请求', value: 'relevantonly' },
  { label: '全量', value: 'on' },
  { label: '关闭', value: 'off' },
];

const activeStepKey = computed<SiteWizardStep>(() => wizardStepAt(currentStep.value));

const stepItems = computed(() =>
  SITE_WIZARD_STEPS.map((item) => ({
    title: item.title,
    description: item.description,
  })),
);

const draftPreview = computed(() => buildQuickSiteDraftFromWizard(formState));

/** 本地生成 Caddyfile 预览（不调用 /adapt 或 /load） */
const localCaddyfilePreview = computed(() =>
  buildWizardLocalCaddyfilePreview(formState, props.baseFormModel),
);

const canGoNext = computed(() => currentStep.value < SITE_WIZARD_STEPS.length - 1);
const canGoPrev = computed(() => currentStep.value > 0);

function resetWizard() {
  Object.assign(formState, createSiteWizardState());
  currentStep.value = 0;
  stepErrors.value = [];
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      resetWizard();
    }
  },
);

function close() {
  emit('update:open', false);
}

function validateCurrent(): boolean {
  const errors = validateWizardStep(formState, activeStepKey.value);
  stepErrors.value = errors;
  return errors.length === 0;
}

function goNext() {
  if (!validateCurrent()) return;
  if (canGoNext.value) {
    currentStep.value += 1;
    stepErrors.value = [];
  }
}

function goPrev() {
  if (canGoPrev.value) {
    currentStep.value -= 1;
    stepErrors.value = [];
  }
}

function ensureDraftValid(): QuickSiteDraft | null {
  const errors = validateWizardUpTo(formState, 'preview');
  stepErrors.value = errors;
  if (errors.length) {
    // 跳回第一个失败步骤
    if (errors.some((e) => e.includes('域名') || e.includes('名称'))) {
      currentStep.value = wizardStepIndex('domain');
    } else if (errors.some((e) => e.includes('上游') || e.includes('健康'))) {
      currentStep.value = wizardStepIndex('upstream');
    }
    return null;
  }
  return buildQuickSiteDraftFromWizard(formState);
}

function handleCommitDraft() {
  const draft = ensureDraftValid();
  if (!draft) return;
  emit('commit-draft', draft, { waf: { ...formState.waf } });
}

function handlePreview() {
  const draft = ensureDraftValid();
  if (!draft) return;
  emit('preview', draft);
  currentStep.value = wizardStepIndex('preview');
}

function handleApply() {
  const draft = ensureDraftValid();
  if (!draft) return;
  if (!props.hasServer) {
    stepErrors.value = ['请先选择 Caddy 服务器后再应用配置'];
    return;
  }
  emit('apply', draft, { waf: { ...formState.waf } });
  currentStep.value = wizardStepIndex('apply');
}

function fillPool(name: string) {
  formState.upstream = name;
}
</script>

<template>
  <Modal
    :open="open"
    title="站点创建向导"
    width="920px"
    :footer="null"
    destroy-on-close
    @cancel="close"
  >
    <div class="site-wizard">
      <Alert
        class="mb-3"
        type="info"
        show-icon
        message="向导默认只产出草稿，不会自动热加载（/load）。预览仅走校验；应用须您确认后走既有 Apply_Path。"
      />

      <Steps
        size="small"
        :current="currentStep"
        :items="stepItems"
        class="wizard-steps mb-4"
      />

      <Alert
        v-if="stepErrors.length"
        class="mb-3"
        type="error"
        show-icon
        :message="stepErrors[0]"
      />

      <!-- 1. 域名 -->
      <div v-show="activeStepKey === 'domain'" class="wizard-pane">
        <Form layout="vertical">
          <Row :gutter="16">
            <Col :span="12">
              <Form.Item label="站点名称" required>
                <Input v-model:value="formState.name" placeholder="例如：官网反代" />
              </Form.Item>
            </Col>
            <Col :span="12">
              <Form.Item label="域名 / 监听地址" required>
                <Select
                  v-model:value="formState.domains"
                  mode="tags"
                  placeholder="example.com 或 :8080，回车添加"
                  :token-separators="[',', ' ']"
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </div>

      <!-- 2. 上游 -->
      <div v-show="activeStepKey === 'upstream'" class="wizard-pane">
        <Form layout="vertical">
          <Row :gutter="16">
            <Col :span="12">
              <Form.Item label="上游地址或池名" required>
                <Input
                  v-model:value="formState.upstream"
                  placeholder="127.0.0.1:8080 或上游池名称"
                />
                <div v-if="upstreamPoolOptions?.length" class="pool-hints">
                  <span class="hint-label">点击填入上游池：</span>
                  <Tag
                    v-for="opt in upstreamPoolOptions"
                    :key="opt.value"
                    color="blue"
                    class="pool-tag"
                    @click="fillPool(String(opt.value))"
                  >
                    {{ opt.value }}
                  </Tag>
                </div>
              </Form.Item>
            </Col>
            <Col :span="12">
              <Form.Item label="负载策略">
                <Select v-model:value="formState.lbPolicy" :options="lbPolicyOptions" />
              </Form.Item>
            </Col>
            <Col :span="24">
              <Form.Item label="站点级健康检查（可选）">
                <div class="health-row">
                  <div class="health-toggle">
                    <span>启用</span>
                    <Switch
                      v-model:checked="formState.healthEnabled"
                      checked-children="开"
                      un-checked-children="关"
                    />
                  </div>
                  <template v-if="formState.healthEnabled">
                    <Input v-model:value="formState.healthPath" placeholder="路径，如 /health" />
                    <Input v-model:value="formState.healthInterval" placeholder="间隔，如 10s" />
                    <Input v-model:value="formState.healthTimeout" placeholder="超时，如 5s" />
                  </template>
                </div>
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </div>

      <!-- 3. TLS -->
      <div v-show="activeStepKey === 'tls'" class="wizard-pane">
        <Form layout="vertical">
          <Form.Item label="TLS 模式">
            <Select v-model:value="formState.tlsMode" :options="tlsModeOptions" style="max-width: 320px" />
          </Form.Item>
          <div class="hint-text">
            简单模式支持自动 HTTPS、内部证书或关闭 TLS；高级证书路径请在复杂/原始配置中维护。
          </div>
        </Form>
      </div>

      <!-- 4. 可选 WAF -->
      <div v-show="activeStepKey === 'waf'" class="wizard-pane">
        <Form layout="vertical">
          <Form.Item label="启用 WAF 意图（可选）">
            <Switch
              v-model:checked="formState.waf.enabled"
              checked-children="开"
              un-checked-children="关"
            />
          </Form.Item>
          <Alert
            class="mb-3"
            type="warning"
            show-icon
            message="WAF 意图仅随草稿记录；写入草稿不会调用防火墙接口。仅在您确认「应用配置」后，才可按 Apply_Path 一并处理。"
          />
          <Row v-if="formState.waf.enabled" :gutter="16">
            <Col :span="8">
              <Form.Item label="引擎模式">
                <Select v-model:value="formState.waf.mode" :options="wafModeOptions" />
              </Form.Item>
            </Col>
            <Col :span="8">
              <Form.Item label="规则强度">
                <Select v-model:value="formState.waf.strength" :options="wafStrengthOptions" />
              </Form.Item>
            </Col>
            <Col :span="8">
              <Form.Item label="审计日志">
                <Select v-model:value="formState.waf.audit" :options="wafAuditOptions" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </div>

      <!-- 5. Preview -->
      <div v-show="activeStepKey === 'preview'" class="wizard-pane">
        <div class="summary-card mb-3">
          <div><strong>站点：</strong>{{ draftPreview.name }}</div>
          <div><strong>域名：</strong>{{ draftPreview.domains.join(', ') || '—' }}</div>
          <div><strong>上游：</strong>{{ draftPreview.upstream || '—' }}</div>
          <div><strong>TLS：</strong>{{ draftPreview.tlsMode }}</div>
          <div><strong>负载策略：</strong>{{ draftPreview.lbPolicy || 'round_robin' }}</div>
          <div>
            <strong>健康检查：</strong>
            <template v-if="draftPreview.healthCheck?.path">
              {{ draftPreview.healthCheck.path }}
              <span v-if="draftPreview.healthCheck.interval"> / {{ draftPreview.healthCheck.interval }}</span>
              <span v-if="draftPreview.healthCheck.timeout"> / {{ draftPreview.healthCheck.timeout }}</span>
            </template>
            <template v-else>关闭</template>
          </div>
          <div>
            <strong>WAF 意图：</strong>
            <Tag :color="formState.waf.enabled ? 'orange' : 'default'">
              {{ formState.waf.enabled ? `${formState.waf.mode} / ${formState.waf.strength}` : '未启用' }}
            </Tag>
          </div>
          <div class="hint-text mt-2">
            自动热加载：{{ wizardAutoLoadEnabled() ? '是' : '否（默认）' }}
          </div>
        </div>
        <div class="preview-label">合并后 Caddyfile 预览（本地生成，未调用 /load）</div>
        <Input.TextArea
          :value="localCaddyfilePreview"
          :auto-size="{ minRows: 10, maxRows: 18 }"
          class="code-textarea"
          readonly
        />
      </div>

      <!-- 6. Apply -->
      <div v-show="activeStepKey === 'apply'" class="wizard-pane">
        <Alert
          class="mb-3"
          type="info"
          show-icon
          message="推荐先「仅写入草稿」在工作台继续编辑；需要热加载时再「预览校验」并确认应用（唯一 /load 入口）。"
        />
        <div class="summary-card">
          <div>将站点 <strong>{{ draftPreview.name }}</strong> 合并进当前配置工作台。</div>
          <div class="hint-text mt-2">
            应用路径：Preview（/adapt）→ 您确认 → Update（/load + history）。
            向导本身不会在无确认时调用 /load。
          </div>
          <div v-if="formState.waf.enabled" class="hint-text mt-2">
            已记录 WAF 意图；应用站点后可在「防火墙」页完善并单独应用 WAF（不会静默执行）。
          </div>
        </div>
      </div>

      <div class="wizard-footer">
        <Space wrap>
          <Button :disabled="!canGoPrev" @click="goPrev">上一步</Button>
          <Button v-if="canGoNext" type="primary" @click="goNext">下一步</Button>
          <Button @click="handleCommitDraft">仅写入草稿</Button>
          <Button
            :loading="previewing"
            :disabled="!hasServer"
            @click="handlePreview"
          >
            预览校验
          </Button>
          <Button
            type="primary"
            danger
            :loading="applying"
            :disabled="!hasServer"
            @click="handleApply"
          >
            预览并应用…
          </Button>
          <Button @click="close">关闭</Button>
        </Space>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.site-wizard {
  min-height: 360px;
}

.wizard-steps {
  margin-bottom: 16px;
}

.wizard-pane {
  min-height: 220px;
  padding: 4px 0 12px;
}

.wizard-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
  margin-top: 8px;
  border-top: 1px solid #eef0f4;
}

.health-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.health-toggle {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 100px;
}

.pool-hints {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  margin-top: 8px;
}

.pool-tag {
  cursor: pointer;
}

.hint-label,
.hint-text {
  color: #667085;
  font-size: 12px;
}

.summary-card {
  padding: 12px 14px;
  background: #fbfcfd;
  border: 1px solid #eef0f4;
  border-radius: 8px;
  line-height: 1.7;
}

.preview-label {
  margin-bottom: 6px;
  color: #667085;
  font-size: 12px;
}

.code-textarea {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono',
    'Courier New', monospace;
  font-size: 12px;
}

.mb-3 {
  margin-bottom: 12px;
}

.mb-4 {
  margin-bottom: 16px;
}

.mt-2 {
  margin-top: 8px;
}
</style>
