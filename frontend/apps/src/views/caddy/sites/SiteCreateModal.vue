<script lang="ts" setup>
import { computed, reactive, ref, watch } from 'vue';
import { message } from 'antdv-next';
import { useRouter } from 'vue-router';
import {
  Alert,
  Button,
  Collapse,
  CollapsePanel,
  Input,
  Modal,
  RadioGroup,
  RadioButton,
  Select,
  Switch,
  Tooltip,
} from 'antdv-next';

import {
  applySimpleWafConfigApi,
  getSimpleWafConfigApi,
  type CaddySimpleWafApi,
} from '#/api/caddy/simple-waf';
import { previewCaddyConfigApi, pushCaddyConfigApi } from '#/api/caddy/server';
import { apiErrorMessage } from '#/utils/api-error-message';

import { useUploadedCertOptions } from '../certificates/use-uploaded-certs';

import {
  appendSiteBlockToConfig,
  buildCreatedSiteBlock,
  createDefaultSiteForm,
  createdSiteAddress,
  validateCreateSiteForm,
  type CreateLbPolicy,
  type CreateSiteForm,
  type CreateTlsMode,
} from './site-create-utils';

/**
 * 一步建站弹窗：一个表单 + 一次 diff 确认。
 * HTTPS 支持「已有证书」：从证书库下拉选择上传证书，按 manual 双路径写入站点块。
 * 确认后走「后端 /adapt 校验 → /load 应用 → 落库」，任一步失败不落库不热加载；
 * 勾选 WAF 时在站点保存成功后追加挂载（复用简单 WAF 应用链路，不静默改全局策略）。
 */
const props = defineProps<{
  open: boolean;
  config: string;
  baseHash?: string;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  created: [];
}>();

const form = reactive<CreateSiteForm>(createDefaultSiteForm());
const step = ref<'form' | 'confirm'>('form');
const formErrors = ref<string[]>([]);
const confirmBusy = ref(false);
const confirmError = ref('');
const wafBusy = ref(false);

const wafStatus = ref<CaddySimpleWafApi.SimpleWafConfig | null>(null);
const wafStatusError = ref('');

const lbPolicyOptions = [
  { label: '轮询（默认）', value: 'round_robin' },
  { label: '最少连接', value: 'least_conn' },
  { label: 'IP 哈希', value: 'ip_hash' },
];

const tlsOptions: Array<{ value: CreateTlsMode; label: string; hint: string }> = [
  { value: 'auto', label: '自动签发', hint: 'HTTP-01/TLS-ALPN，需公网 80/443 可达' },
  { value: 'dns', label: 'DNS 签发', hint: 'Cloudflare DNS-01，支持泛域名与内网部署' },
  { value: 'internal', label: '内部 CA', hint: 'LogFlux 本地 CA 签发，浏览器需信任' },
  { value: 'manual', label: '已有证书', hint: '从证书库选择已上传证书，按 tls 双路径写入站点块' },
  { value: 'off', label: '暂不启用', hint: '仅 HTTP，内网或已有前置网关时使用' },
];

// 已有证书下拉：仅弹窗打开时拉取（source=upload 且未失效）
const router = useRouter();
const {
  options: certOptions,
  loading: certLoading,
  errorMessage: certErrorMessage,
  refetch: refetchCerts,
} = useUploadedCertOptions(() => props.open);
/** 当前选中的证书记录 id；路径写入 form，选中态只作回显 */
const selectedCertId = ref<number | undefined>();

function handleCertSelect(value: unknown) {
  const option = certOptions.value.find((item) => item.value === value);
  selectedCertId.value = option?.value;
  form.manualCertPath = option?.certPath ?? '';
  form.manualKeyPath = option?.keyPath ?? '';
}

/** 空态引导跳证书中心上传 */
function goCertificates() {
  closeModal();
  router.push('/caddy/certificates');
}

const nextConfig = computed(() =>
  appendSiteBlockToConfig(props.config, buildCreatedSiteBlock(form)),
);

/** 纯追加场景无需全量 diff：直接展示自己生成的站点块 */
const createdBlock = computed(() => buildCreatedSiteBlock(form));
const unchangedLineCount = computed(() =>
  props.config.trim() ? props.config.trim().split('\n').length : 0,
);
const domainInputValue = computed({
  get: () => form.domains,
  set: (value: string[]) => {
    form.domains = value.map((item) => item.trim()).filter(Boolean);
  },
});

watch(
  () => props.open,
  (open) => {
    if (!open) return;
    Object.assign(form, createDefaultSiteForm());
    selectedCertId.value = undefined;
    step.value = 'form';
    formErrors.value = [];
    confirmError.value = '';
    void fetchWafStatus();
  },
);

async function fetchWafStatus() {
  wafStatusError.value = '';
  wafBusy.value = true;
  try {
    wafStatus.value = await getSimpleWafConfigApi();
  } catch (error) {
    wafStatus.value = null;
    wafStatusError.value = apiErrorMessage(error, '获取 WAF 状态失败');
  } finally {
    wafBusy.value = false;
  }
}

function closeModal() {
  emit('update:open', false);
}

function handleSaveClick() {
  formErrors.value = validateCreateSiteForm(form, props.config);
  if (formErrors.value.length > 0) return;
  confirmError.value = '';
  step.value = 'confirm';
}

async function handleConfirm() {
  confirmBusy.value = true;
  confirmError.value = '';
  try {
    // 先走后端 /adapt 校验，坏配置在这一步被拦下，不会触达运行态
    const preview = await previewCaddyConfigApi({
      mode: 'raw',
      config: nextConfig.value,
      baseHash: props.baseHash || undefined,
    });
    if (preview && preview.valid === false) {
      confirmError.value = preview.errors?.[0] || '配置校验未通过';
      return;
    }
    await pushCaddyConfigApi(
      nextConfig.value,
      undefined,
      props.baseHash || undefined,
    );

    // 防护跟随全局默认：挂载前以最新配置判断全局是否启用并仅追加本站点地址
    const wafMounted = await mountSiteToWaf();
    message.success(wafMounted ? '站点已上线并挂载 WAF' : '站点已上线');
    emit('created');
    closeModal();
  } catch (error) {
    confirmError.value = apiErrorMessage(error, '保存失败');
  } finally {
    confirmBusy.value = false;
  }
}

/** 挂载 WAF：重取最新 simple-config 再仅追加本站点地址，避免用弹窗打开时的旧快照覆盖并发改动 */
async function mountSiteToWaf(): Promise<boolean> {
  const address = createdSiteAddress(form);
  try {
    const latest = await getSimpleWafConfigApi();
    if (!latest?.enabled) return false;
    const siteAddresses = [...new Set([...(latest.siteAddresses ?? []), address])];
    await applySimpleWafConfigApi({
      enabled: latest.enabled,
      mode: latest.mode as CaddySimpleWafApi.SimpleWafMode,
      strength: latest.strength as CaddySimpleWafApi.SimpleWafStrength,
      audit: latest.audit as CaddySimpleWafApi.SimpleWafAudit,
      requestBodyAccess: latest.requestBodyAccess,
      requestBodyLimit: latest.requestBodyLimit,
      requestBodyNoFilesLimit: latest.requestBodyNoFilesLimit,
      siteAddresses,
    });
    return true;
  } catch (error) {
    // 站点本体已保存成功，WAF 挂载失败只降级提示，不回滚站点
    message.warning(`站点已保存，但 WAF 挂载失败：${apiErrorMessage(error, '请到防火墙页重试')}`);
    return false;
  }
}
</script>

<template>
  <Modal
    :open="props.open"
    :title="step === 'form' ? '添加站点' : '确认变更'"
    :width="720"
    :mask-closable="false"
    :footer="undefined"
    @update:open="emit('update:open', $event)"
  >
    <!-- 第一步：单表单 -->
    <div v-if="step === 'form'" class="create-form">
      <Alert
        v-if="formErrors.length"
        class="mb-3"
        type="error"
        show-icon
        :message="formErrors[0]"
      />

      <div class="field">
        <div class="field-label">域名 / 监听地址</div>
        <Select
          v-model:value="domainInputValue"
          mode="tags"
          placeholder="回车添加，如 app.example.com 或 :80"
          :token-separators="[',', ' ']"
          :open="false"
          class="w-full"
        />
      </div>

      <div class="field">
        <div class="field-label">上游地址</div>
        <Input v-model:value="form.upstream" placeholder="localhost:8080" />
      </div>

      <div class="field">
        <div class="field-label">HTTPS</div>
        <RadioGroup v-model:value="form.tlsMode" class="tls-group">
          <Tooltip v-for="option in tlsOptions" :key="option.value" :title="option.hint">
            <RadioButton :value="option.value">{{ option.label }}</RadioButton>
          </Tooltip>
        </RadioGroup>
        <template v-if="form.tlsMode === 'manual'">
          <Select
            class="cert-select"
            :value="selectedCertId"
            :options="certOptions"
            :loading="certLoading"
            allow-clear
            placeholder="选择已上传证书"
            @change="handleCertSelect"
          >
            <template #notFoundContent>
              <div class="cert-empty">
                暂无已上传证书，
                <a @click="goCertificates">前往证书中心上传</a>
              </div>
            </template>
          </Select>
          <div v-if="certErrorMessage" class="cert-note cert-error">
            {{ certErrorMessage }}，
            <a @click="() => refetchCerts()">重试</a>
          </div>
          <div v-else-if="selectedCertId" class="cert-note">
            证书 {{ form.manualCertPath }} · 私钥 {{ form.manualKeyPath }}
          </div>
        </template>
      </div>

      <Collapse :bordered="false" class="advanced">
        <CollapsePanel key="advanced" header="高级（WAF · 负载 · 健康检查）">
          <div class="field-row">
            <div class="field-label">WAF 防护</div>
            <Tooltip title="跟随全局默认：全局已启用则保存后自动挂载；如需单独调整请到「防火墙」页">
              <span class="waf-default-hint">
                {{ wafBusy ? '获取中…' : wafStatus?.enabled ? '跟随全局（已启用）' : '跟随全局（未启用）' }}
              </span>
            </Tooltip>
          </div>
          <div class="field-row">
            <div class="field-label">负载策略</div>
            <Select
              v-model:value="form.lbPolicy as CreateLbPolicy"
              :options="lbPolicyOptions"
              class="policy-select"
            />
          </div>
          <div class="field-row">
            <div class="field-label">健康检查</div>
            <Switch v-model:checked="form.healthEnabled" />
          </div>
          <div v-if="form.healthEnabled" class="health-row">
            <Input v-model:value="form.healthPath" placeholder="路径 /health" class="health-input" />
            <Input v-model:value="form.healthInterval" placeholder="间隔 10s" class="health-input" />
            <Input v-model:value="form.healthTimeout" placeholder="超时 5s" class="health-input" />
          </div>
        </CollapsePanel>
      </Collapse>

      <div class="footer-actions">
        <Button @click="closeModal">取消</Button>
        <Button type="primary" @click="handleSaveClick">保存</Button>
      </div>
    </div>

    <!-- 第二步：diff 确认 -->
    <div v-else class="confirm-step">
      <Alert
        v-if="confirmError"
        class="mb-3"
        type="error"
        show-icon
        :message="confirmError"
      />
      <div class="diff-hint">
        追加到配置末尾（其余 {{ unchangedLineCount }} 行不变），确认后执行「校验 → 应用 → 落库」：
      </div>
      <div class="diff-box">
        <div
          v-for="(line, index) in createdBlock.split('\n')"
          :key="index"
          class="diff-line diff-added"
        >
          <span class="diff-sign">+</span>
          <span class="diff-text">{{ line }}</span>
        </div>
      </div>
      <div class="footer-actions">
        <Button :disabled="confirmBusy" @click="step = 'form'">返回修改</Button>
        <Button type="primary" :loading="confirmBusy" @click="handleConfirm">
          确认并上线
        </Button>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.mb-3 {
  margin-bottom: 12px;
}

.field {
  margin-bottom: 16px;
}

.field-label {
  margin-bottom: 6px;
  font-weight: 500;
}

.field-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.health-row {
  display: flex;
  gap: 8px;
  margin-bottom: 4px;
}

.health-input {
  flex: 1;
}

.policy-select {
  width: 180px;
}

.advanced {
  margin: 4px -16px 12px;
  background: transparent;
}

.cert-select {
  width: 100%;
  margin-top: 8px;
}

.cert-empty {
  padding: 8px 12px;
  color: rgba(0, 0, 0, 45%);
}

.cert-note {
  margin-top: 4px;
  font-size: 12px;
  color: hsl(var(--muted-foreground));
  word-break: break-all;
}

.cert-error {
  color: #cf1322;
}

.footer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}

.diff-hint {
  margin-bottom: 8px;
  color: rgba(0, 0, 0, 65%);
}

.diff-box {
  max-height: 380px;
  padding: 8px 0;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.6;
  background: rgba(0, 0, 0, 3%);
  border-radius: 6px;
}

.diff-line {
  display: flex;
  padding: 0 12px;
  white-space: pre;
}

.diff-sign {
  width: 18px;
  flex-shrink: 0;
  color: rgba(0, 0, 0, 45%);
}

.diff-added {
  background: rgba(82, 196, 26, 18%);
}

.waf-default-hint {
  color: rgba(0, 0, 0, 65%);
}
</style>
