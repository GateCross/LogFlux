<script lang="ts" setup>
import { computed, reactive, ref, watch } from 'vue';
import { message } from 'antdv-next';
import { useRouter } from 'vue-router';
import {
  Alert,
  Button,
  Drawer,
  Input,
  RadioButton,
  RadioGroup,
  Select,
  Switch,
  TabPane,
  Tabs,
  Tooltip,
} from 'antdv-next';

import { previewCaddyConfigApi, pushCaddyConfigApi } from '#/api/caddy/server';
import { apiErrorMessage } from '#/utils/api-error-message';

import { useUploadedCertOptions } from '../certificates/use-uploaded-certs';

import {
  applyCapabilityToSiteBlock,
  extractSiteCapabilities,
  renderCapabilityLines,
  type SiteCapabilityKey,
  type SiteCapabilitiesSnapshot,
} from '../config/site-capabilities';
import type { CatalogSiteCard } from '../catalog/service-catalog-utils';
import {
  applyBasicEditsToSiteBlock,
  parseManualTlsPaths,
  replaceSiteBlockInConfig,
  type EditableTlsMode,
} from './site-block-edit';

/**
 * 站点抽屉：点卡片就地查看与编辑，不跳页。
 * 基本 Tab：简单站点可改域名/上游/TLS（外科编辑），复杂站点只读原文；
 * manual 模式可从证书库下拉选择自动填充双路径，也保留手动输入。
 * 防护 Tab：能力开关逐项覆盖全局默认，unknown 时禁用引导原文模式。
 * 保存链路与建站一致：diff 确认 → /adapt 校验 → /load → 落库。
 */
const props = defineProps<{
  open: boolean;
  serverId?: number;
  config: string;
  baseHash?: string;
  site: CatalogSiteCard | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  saved: [];
  openWorkbench: [];
}>();

const step = ref<'edit' | 'confirm'>('edit');
const saving = ref(false);
const confirmError = ref('');

const basicEditable = computed(() => props.site?.kind === 'simple' && Boolean(props.site?.raw));

const basic = reactive({
  domains: [] as string[],
  upstream: '',
  tlsMode: 'auto' as EditableTlsMode,
  /** manual 模式的证书/私钥路径（Caddy 容器内路径，可从证书中心复制） */
  manualCertPath: '',
  manualKeyPath: '',
});

/** 能力开关当前值；undefined 表示该能力状态 unknown，开关禁用 */
const toggles = reactive<Record<SiteCapabilityKey, boolean | undefined>>({
  waf: false,
  geo: false,
  compression: false,
  securityHeaders: false,
  accessLog: false,
  errorPage: false,
});

// 从证书库选择：抽屉打开且基本 Tab 可编辑时拉取，按当前服务器过滤上传证书
const router = useRouter();
const {
  options: certOptions,
  loading: certLoading,
  errorMessage: certErrorMessage,
  refetch: refetchCerts,
} = useUploadedCertOptions(
  () => props.serverId,
  () => props.open && basicEditable.value,
);
/** 当前选中的证书记录 id；清空选择不动已填路径，仍允许手动输入 */
const selectedCertId = ref<number | undefined>();

/** 既有 manual 双路径与某上传证书一致时回显选中（不清除用户已做的选择） */
function syncCertSelection() {
  if (selectedCertId.value != null) return;
  const cert = basic.manualCertPath.trim();
  const key = basic.manualKeyPath.trim();
  if (!cert || !key) return;
  selectedCertId.value = certOptions.value.find(
    (item) => item.certPath === cert && item.keyPath === key,
  )?.value;
}

// 选项异步就绪时补一次回显；切换站点由 resetState 同步触发
watch(certOptions, syncCertSelection);

function handleCertSelect(value: unknown) {
  const option = certOptions.value.find((item) => item.value === value);
  selectedCertId.value = option?.value;
  if (option) {
    basic.manualCertPath = option.certPath;
    basic.manualKeyPath = option.keyPath;
  }
}

/** 空态引导跳证书中心上传 */
function goCertificates() {
  emit('update:open', false);
  router.push('/caddy/certificates');
}

const capabilityItems: Array<{ key: SiteCapabilityKey; label: string; hint: string }> = [
  { key: 'waf', label: 'WAF 防护', hint: '挂载 import waf_protect，强度跟随全局策略' },
  { key: 'geo', label: 'IP 区域限制', hint: '挂载 import geo_auth，国家/地区白名单在「访问控制」配置' },
  { key: 'securityHeaders', label: '安全响应头', hint: '挂载 import security_headers 片段' },
  { key: 'errorPage', label: '403 错误页跳转', hint: '挂载 import deny_403_redirect 片段' },
  { key: 'compression', label: '压缩', hint: '挂载 import compression（zstd/gzip）' },
  { key: 'accessLog', label: '独立访问日志', hint: '挂载 import subdomain-log，按域名分文件' },
];

const capabilities = computed<SiteCapabilitiesSnapshot | null>(() => {
  if (!props.site?.raw) return null;
  return props.site.capabilities ?? extractSiteCapabilities(props.site.raw);
});

const logName = computed(() => {
  const raw = props.site?.raw ?? '';
  const match = raw.match(/^[ \t]*import\s+subdomain-log\s+(\S+)/m);
  if (match?.[1]) return match[1];
  const host = props.site?.primaryHost ?? '';
  return host.split('.')[0] || 'site';
});

function resetState() {
  const site = props.site;
  const caps = capabilities.value;
  basic.domains = [...(site?.domains ?? [])];
  basic.upstream = site?.upstream ?? '';
  basic.tlsMode = caps?.tls.mode ?? 'auto';
  const manualPaths =
    caps?.tls.mode === 'manual' ? parseManualTlsPaths(site?.raw ?? '') : null;
  basic.manualCertPath = manualPaths?.certPath ?? '';
  basic.manualKeyPath = manualPaths?.keyPath ?? '';
  selectedCertId.value = undefined;
  syncCertSelection();
  for (const item of capabilityItems) {
    const status = caps?.[item.key]?.status;
    toggles[item.key] = status === 'unknown' ? undefined : status === 'on';
  }
  step.value = 'edit';
  confirmError.value = '';
}

watch(
  () => [props.open, props.site?.id],
  ([open]) => {
    if (open) resetState();
  },
);

function parseUpstreamFromRaw(raw: string): string {
  const match = raw.match(/^[ \t]*reverse_proxy\s+(.+?)\s*\{?\s*$/m);
  return match?.[1]?.trim() ?? '';
}

/** 生成下一个站点块与整份配置；无变化返回 null */
function buildNext(): { block: string; config: string } | null {
  const site = props.site;
  const raw = site?.raw;
  if (!site || !raw) return null;

  let nextBlock = raw;
  if (basicEditable.value) {
    const edits: {
      domains?: string[];
      upstream?: string;
      tlsMode?: EditableTlsMode;
      tlsCertPath?: string;
      tlsKeyPath?: string;
    } = {};
    if (basic.domains.join(' ') !== site.domains.join(' ')) edits.domains = basic.domains;
    const parsedUpstream = site.upstream || parseUpstreamFromRaw(raw);
    if (basic.upstream.trim() && basic.upstream.trim() !== parsedUpstream) {
      edits.upstream = basic.upstream.trim();
    }
    const currentTls = capabilities.value?.tls.mode ?? 'auto';
    if (basic.tlsMode === 'manual') {
      const cert = basic.manualCertPath.trim();
      const key = basic.manualKeyPath.trim();
      const currentPaths =
        currentTls === 'manual' ? parseManualTlsPaths(raw) : null;
      const changed =
        !currentPaths ||
        currentPaths.certPath !== cert ||
        currentPaths.keyPath !== key;
      // 路径齐全且相对现状有变化才写：与现状一致的重复保存保持字节不变
      if (cert && key && changed) {
        edits.tlsMode = 'manual';
        edits.tlsCertPath = cert;
        edits.tlsKeyPath = key;
      }
    } else if (basic.tlsMode !== currentTls) {
      edits.tlsMode = basic.tlsMode;
    }
    if (Object.keys(edits).length > 0) {
      nextBlock = applyBasicEditsToSiteBlock(nextBlock, edits);
    }
  }

  for (const item of capabilityItems) {
    const caps = capabilities.value;
    const current = caps?.[item.key]?.status;
    const target = toggles[item.key];
    if (target === undefined || current === 'unknown') continue;
    if (target === (current === 'on')) continue;
    nextBlock = applyCapabilityToSiteBlock(
      nextBlock,
      item.key,
      renderCapabilityLines(item.key, { enabled: target, name: logName.value }),
    );
  }

  if (nextBlock === raw) return null;
  const nextConfig = replaceSiteBlockInConfig(props.config, raw, nextBlock);
  if (nextConfig === props.config) return null;
  return { block: nextBlock, config: nextConfig };
}

const nextPreview = computed(() => buildNext());
const unchangedLineCount = computed(() =>
  props.config.trim() ? props.config.trim().split('\n').length : 0,
);

function handleSaveClick() {
  if (
    basic.tlsMode === 'manual' &&
    (!basic.manualCertPath.trim() || !basic.manualKeyPath.trim())
  ) {
    message.warning('手动证书模式需同时填写证书与私钥路径');
    return;
  }
  if (!nextPreview.value) {
    message.info('没有需要保存的修改');
    return;
  }
  confirmError.value = '';
  step.value = 'confirm';
}

async function handleConfirm() {
  const next = nextPreview.value;
  if (!next || !props.serverId) return;
  saving.value = true;
  confirmError.value = '';
  try {
    const preview = await previewCaddyConfigApi(props.serverId, {
      mode: 'raw',
      config: next.config,
      baseHash: props.baseHash || undefined,
    });
    if (preview && preview.valid === false) {
      confirmError.value = preview.errors?.[0] || '配置校验未通过';
      return;
    }
    await pushCaddyConfigApi(props.serverId, next.config, undefined, props.baseHash || undefined);
    message.success('站点配置已更新');
    emit('saved');
    emit('update:open', false);
  } catch (error) {
    confirmError.value = apiErrorMessage(error, '保存失败');
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <Drawer
    :open="props.open"
    :title="props.site?.name || '站点'"
    width="580"
    @update:open="emit('update:open', $event)"
  >
    <!-- 确认视图：展示新站点块，其余行不动 -->
    <div v-if="step === 'confirm'">
      <Alert v-if="confirmError" class="mb-3" type="error" show-icon :message="confirmError" />
      <div class="diff-hint">
        仅替换该站点块（其余 {{ unchangedLineCount }} 行不变），确认后执行「校验 → 应用 → 落库」：
      </div>
      <div class="diff-box">
        <div
          v-for="(line, index) in (nextPreview?.block ?? '').split('\n')"
          :key="index"
          class="diff-line diff-added"
        >
          <span class="diff-sign">+</span>
          <span class="diff-text">{{ line }}</span>
        </div>
      </div>
      <div class="footer-actions">
        <Button :disabled="saving" @click="step = 'edit'">返回修改</Button>
        <Button type="primary" :loading="saving" @click="handleConfirm">确认并应用</Button>
      </div>
    </div>

    <!-- 编辑视图 -->
    <div v-else-if="props.site">
      <Alert
        v-if="!props.site.raw"
        class="mb-3"
        type="warning"
        show-icon
        message="未取到该站点的配置块原文，请刷新后重试"
      />

      <Tabs>
        <TabPane key="basic" tab="基本">
          <template v-if="basicEditable">
            <div class="field">
              <div class="field-label">域名 / 监听地址</div>
              <Select
                v-model:value="basic.domains"
                mode="tags"
                placeholder="回车添加"
                :token-separators="[',', ' ']"
                :open="false"
                class="w-full"
              />
            </div>
            <div class="field">
              <div class="field-label">上游地址</div>
              <Input v-model:value="basic.upstream" placeholder="localhost:8080" />
            </div>
            <div class="field">
              <div class="field-label">HTTPS</div>
              <Tooltip
                :title="
                  capabilities?.tls.status === 'unknown'
                    ? '无法识别当前 TLS 配置，请用原文模式调整'
                    : undefined
                "
              >
                <RadioGroup
                  v-model:value="basic.tlsMode"
                  class="tls-group"
                  :disabled="capabilities?.tls.status === 'unknown'"
                >
                  <RadioButton value="auto">自动</RadioButton>
                  <RadioButton value="dns">DNS</RadioButton>
                  <RadioButton value="internal">内部 CA</RadioButton>
                  <RadioButton value="manual">手动</RadioButton>
                  <RadioButton value="off">关闭</RadioButton>
                </RadioGroup>
              </Tooltip>
              <div v-if="capabilities?.tls.status === 'unknown'" class="cap-note">
                {{ capabilities?.tls.note }}
              </div>
              <template v-if="basic.tlsMode === 'manual'">
                <Select
                  class="tls-cert-select"
                  :value="selectedCertId"
                  :options="certOptions"
                  :loading="certLoading"
                  allow-clear
                  placeholder="从证书库选择，自动填充下方路径（也可手动填写）"
                  @change="handleCertSelect"
                >
                  <template #notFoundContent>
                    <div class="cert-empty">
                      暂无已上传证书，
                      <a @click="goCertificates">前往证书中心上传</a>
                    </div>
                  </template>
                </Select>
                <div v-if="certErrorMessage" class="cap-note cert-error">
                  {{ certErrorMessage }}，
                  <a @click="() => refetchCerts()">重试</a> 或直接手动填写路径
                </div>
                <Input
                  v-model:value="basic.manualCertPath"
                  class="tls-path-input"
                  placeholder="证书路径（容器内），如 /config/certs/1/example.com/fullchain.pem"
                />
                <Input
                  v-model:value="basic.manualKeyPath"
                  class="tls-path-input"
                  placeholder="私钥路径（容器内），如 /config/certs/1/example.com/privkey.pem"
                />
                <div class="cap-note">
                  路径须为 Caddy 容器内可读位置，可从证书中心复制存储路径
                </div>
              </template>
            </div>
          </template>
          <template v-else>
            <Alert
              class="mb-3"
              type="info"
              show-icon
              message="复杂站点：基本字段只读，防护开关仍可逐项调整；完整编辑请用原文模式"
            />
            <pre class="raw-block">{{ props.site.raw }}</pre>
            <Button @click="emit('openWorkbench')">在原文模式打开</Button>
          </template>
        </TabPane>

        <TabPane key="protect" tab="防护">
          <div class="field-label field-title">逐项覆盖全局默认（不改动即跟随全局）</div>
          <div v-for="item in capabilityItems" :key="item.key" class="field-row">
            <div>
              <div>{{ item.label }}</div>
              <div v-if="capabilities?.[item.key]?.status === 'unknown'" class="cap-note">
                {{ capabilities?.[item.key]?.note }}
              </div>
              <div v-else class="cap-note">{{ item.hint }}</div>
            </div>
            <Tooltip
              :title="capabilities?.[item.key]?.status === 'unknown' ? '无法识别当前配置，请用原文模式调整' : item.hint"
            >
              <Switch
                v-model:checked="(toggles[item.key] as boolean)"
                :disabled="capabilities?.[item.key]?.status === 'unknown'"
              />
            </Tooltip>
          </div>
          <div v-if="capabilities?.otherImports?.length" class="cap-note">
            其他业务片段：{{ capabilities.otherImports.join('、') }}（在原文模式维护）
          </div>
        </TabPane>
      </Tabs>

      <div class="footer-actions">
        <Button @click="emit('update:open', false)">取消</Button>
        <Button type="primary" @click="handleSaveClick">保存</Button>
      </div>
    </div>
  </Drawer>
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

.field-title {
  margin-bottom: 12px;
}

.field-row {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid hsl(var(--border) / 60%);
}

.tls-cert-select {
  width: 100%;
  margin-top: 8px;
}

.cert-empty {
  padding: 8px 12px;
  color: rgba(0, 0, 0, 45%);
}

.tls-path-input {
  margin-top: 8px;
}

.cap-note {
  margin-top: 2px;
  font-size: 12px;
  color: hsl(var(--muted-foreground));
}

.cap-note.cert-error {
  margin-top: 4px;
  color: #cf1322;
}

.raw-block {
  max-height: 320px;
  padding: 8px 12px;
  overflow: auto;
  font-size: 12.5px;
  line-height: 1.6;
  white-space: pre;
  background: rgba(0, 0, 0, 3%);
  border-radius: 6px;
}

.footer-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 16px;
}

.diff-hint {
  margin-bottom: 8px;
  color: rgba(0, 0, 0, 65%);
}

.diff-box {
  max-height: 420px;
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
</style>
