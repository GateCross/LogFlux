<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useMessage } from 'naive-ui';
import {
  applySimpleWafConfig,
  fetchSimpleWafConfig,
  previewSimpleWafConfig,
  updateSimpleWafConfig,
  type SimpleWafAudit,
  type SimpleWafConfigPayload,
  type SimpleWafConfigResp,
  type SimpleWafMode,
  type SimpleWafStrength
} from '@/service/api/caddy-simple-waf';

const props = defineProps<{
  serverId: number | null;
  onApplied?: () => void | Promise<void>;
}>();

const message = useMessage();
const loading = ref(false);
const submitting = ref(false);
const previewing = ref(false);
const saving = ref(false);
const currentStatus = ref<SimpleWafConfigResp | null>(null);
const previewResult = ref<SimpleWafConfigResp | null>(null);
const showPreviewModal = ref(false);

const form = reactive({
  enabled: false,
  mode: 'detectiononly' as SimpleWafMode,
  strength: 'low_fp' as SimpleWafStrength,
  audit: 'relevantonly' as SimpleWafAudit,
  requestBodyAccess: true,
  requestBodyLimitMB: 10,
  requestBodyNoFilesLimitMB: 1,
  siteAddresses: [] as string[]
});

const modeOptions = [
  { label: '仅检测', value: 'detectiononly' },
  { label: '阻断', value: 'on' },
  { label: '关闭', value: 'off' }
];

const strengthOptions = [
  { label: '低误报', value: 'low_fp' },
  { label: '平衡', value: 'balanced' },
  { label: '严格', value: 'high_blocking' }
];

const auditOptions = [
  { label: '相关请求', value: 'relevantonly' },
  { label: '全量', value: 'on' },
  { label: '关闭', value: 'off' }
];

const statusType = computed(() => {
  if (!currentStatus.value) return 'default';
  if (currentStatus.value.enabled && currentStatus.value.mode === 'on') return 'success';
  if (currentStatus.value.enabled && currentStatus.value.mode === 'detectiononly') return 'warning';
  return 'default';
});

const statusText = computed(() => {
  if (!currentStatus.value) return '未加载';
  if (!currentStatus.value.enabled) return '关闭';
  if (currentStatus.value.mode === 'on') return '阻断';
  if (currentStatus.value.mode === 'detectiononly') return '仅检测';
  return '关闭';
});

const siteOptions = computed(() =>
  (currentStatus.value?.availableSites || []).map(item => ({
    label: item,
    value: item
  }))
);

function formatVersion(value?: string) {
  const trimmed = String(value || '').trim();
  return trimmed || '未检测到';
}

function mbToBytes(value: number) {
  return Math.max(1, Math.round(Number(value || 0))) * 1024 * 1024;
}

function bytesToMB(value: number, fallback: number) {
  if (!value || value <= 0) return fallback;
  return Math.max(1, Math.round(value / 1024 / 1024));
}

function syncForm(data: SimpleWafConfigResp) {
  currentStatus.value = data;
  form.enabled = Boolean(data.enabled);
  form.mode = data.mode === 'off' ? 'detectiononly' : data.mode || 'detectiononly';
  form.strength = data.strength || 'low_fp';
  form.audit = data.audit || 'relevantonly';
  form.requestBodyAccess = data.requestBodyAccess;
  form.requestBodyLimitMB = bytesToMB(data.requestBodyLimit, 10);
  form.requestBodyNoFilesLimitMB = bytesToMB(data.requestBodyNoFilesLimit, 1);
  form.siteAddresses = data.siteAddresses?.length ? [...data.siteAddresses] : [...(data.availableSites || [])];
}

async function fetchData() {
  if (!props.serverId) return;
  loading.value = true;
  try {
    const { data, error } = await fetchSimpleWafConfig(props.serverId);
    if (error || !data) {
      message.error('获取防火墙配置失败');
      return;
    }
    syncForm(data);
  } finally {
    loading.value = false;
  }
}

function buildPayload(): SimpleWafConfigPayload {
  const enabled = Boolean(form.enabled);
  return {
    serverId: props.serverId || undefined,
    enabled,
    mode: enabled ? form.mode : 'off',
    strength: form.strength,
    audit: form.audit,
    requestBodyAccess: form.requestBodyAccess,
    requestBodyLimit: mbToBytes(form.requestBodyLimitMB),
    requestBodyNoFilesLimit: mbToBytes(form.requestBodyNoFilesLimitMB),
    siteAddresses: [...form.siteAddresses]
  };
}

async function handleSave() {
  if (!props.serverId) return;
  saving.value = true;
  try {
    const { error } = await updateSimpleWafConfig(buildPayload());
    if (error) {
      message.error('保存防火墙设置失败');
      return;
    }
    message.success('防火墙设置已保存');
    await fetchData();
  } finally {
    saving.value = false;
  }
}

async function handlePreview() {
  if (!props.serverId) return;
  previewing.value = true;
  try {
    const { data, error } = await previewSimpleWafConfig(buildPayload());
    if (error || !data) {
      message.error('生成防火墙预览失败');
      return;
    }
    previewResult.value = data;
    showPreviewModal.value = true;
  } finally {
    previewing.value = false;
  }
}

async function handleApply() {
  if (!props.serverId) return;
  submitting.value = true;
  try {
    const { data, error } = await applySimpleWafConfig(buildPayload());
    if (error || !data) {
      message.error('应用防火墙配置失败');
      return;
    }
    message.success(data.message || '防火墙配置已应用');
    syncForm(data);
    await props.onApplied?.();
  } finally {
    submitting.value = false;
  }
}

function handleSiteChange(value: Array<string | number>) {
  form.siteAddresses = value.map(item => String(item));
}

watch(
  () => props.serverId,
  () => {
    previewResult.value = null;
    void fetchData();
  }
);

watch(
  () => form.enabled,
  enabled => {
    if (enabled && form.mode === 'off') {
      form.mode = 'detectiononly';
    }
  }
);

onMounted(fetchData);
</script>

<template>
  <div class="h-full min-h-0 overflow-auto">
    <n-spin :show="loading">
      <n-space vertical size="large">
        <n-card size="small" :bordered="false">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div class="font-semibold">防火墙设置</div>
              <n-tag size="small" :type="statusType" :bordered="false">{{ statusText }}</n-tag>
            </div>
          </template>

          <div class="mb-4 grid gap-3 lg:grid-cols-2">
            <div class="rounded-8px border border-#e5e7eb px-3 py-2">
              <div class="text-xs text-gray-500">Coraza 版本</div>
              <div class="mt-1 font-medium">{{ formatVersion(currentStatus?.corazaVersion) }}</div>
            </div>
            <div class="rounded-8px border border-#e5e7eb px-3 py-2">
              <div class="text-xs text-gray-500">CRS 版本</div>
              <div class="mt-1 font-medium">{{ formatVersion(currentStatus?.crsVersion) }}</div>
            </div>
          </div>

          <n-form label-placement="top">
            <div class="grid gap-4 lg:grid-cols-3">
              <n-form-item label="启用">
                <n-switch v-model:value="form.enabled" />
              </n-form-item>
              <n-form-item label="模式">
                <n-select v-model:value="form.mode" :options="modeOptions" :disabled="!form.enabled" />
              </n-form-item>
              <n-form-item label="强度">
                <n-select v-model:value="form.strength" :options="strengthOptions" :disabled="!form.enabled" />
              </n-form-item>
            </div>

            <div class="grid gap-4 lg:grid-cols-3">
              <n-form-item label="审计日志">
                <n-select v-model:value="form.audit" :options="auditOptions" />
              </n-form-item>
              <n-form-item label="请求体上限(MB)">
                <n-input-number v-model:value="form.requestBodyLimitMB" :min="1" :max="1024" />
              </n-form-item>
              <n-form-item label="无文件请求体上限(MB)">
                <n-input-number v-model:value="form.requestBodyNoFilesLimitMB" :min="1" :max="1024" />
              </n-form-item>
            </div>

            <n-form-item label="请求体检查">
              <n-switch v-model:value="form.requestBodyAccess" />
            </n-form-item>

            <n-form-item label="适用站点">
              <n-checkbox-group :value="form.siteAddresses" @update:value="handleSiteChange">
                <n-space wrap>
                  <n-checkbox v-for="item in siteOptions" :key="item.value" :value="item.value" :label="item.label" />
                </n-space>
              </n-checkbox-group>
            </n-form-item>
          </n-form>

          <n-alert v-if="currentStatus?.message" type="info" :show-icon="true">
            {{ currentStatus.message }}
          </n-alert>

          <div class="mt-4 flex flex-wrap justify-end gap-2">
            <n-button :loading="saving" :disabled="!props.serverId" @click="handleSave">保存设置</n-button>
            <n-button :loading="previewing" :disabled="!props.serverId" @click="handlePreview">预览变更</n-button>
            <n-button type="primary" :loading="submitting" :disabled="!props.serverId" @click="handleApply">
              应用到 Caddy
            </n-button>
          </div>
        </n-card>

        <n-card v-if="currentStatus?.directives" size="small" :bordered="false">
          <template #header>当前指令</template>
          <n-code :code="currentStatus.directives" language="shell" word-wrap />
        </n-card>
      </n-space>
    </n-spin>

    <n-modal v-model:show="showPreviewModal" preset="card" title="防火墙配置预览" class="w-[90vw] max-w-5xl">
      <n-space vertical size="large">
        <div v-if="previewResult?.actions?.length" class="flex flex-wrap gap-2">
          <n-tag v-for="item in previewResult.actions" :key="item" size="small" type="info" :bordered="false">
            {{ item }}
          </n-tag>
        </div>
        <n-code v-if="previewResult?.directives" :code="previewResult.directives" language="shell" word-wrap />
        <n-input
          v-if="previewResult?.config"
          type="textarea"
          readonly
          :value="previewResult.config"
          :autosize="{ minRows: 12, maxRows: 24 }"
        />
      </n-space>
    </n-modal>
  </div>
</template>
