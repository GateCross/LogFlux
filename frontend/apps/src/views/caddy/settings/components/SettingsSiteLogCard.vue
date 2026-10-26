<script lang="ts" setup>
/**
 * 站点日志卡：自动清理开关、保留天数与 (subdomain-log) 模板两段。
 * 保留策略由后端每天清理任务使用；模板保存后仍需在配置管理页应用才进入 Caddyfile。
 */
import { computed, ref, watch } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import {
  Alert,
  Button,
  Card,
  Form,
  FormItem,
  InputNumber,
  message,
  Modal,
  Space,
  Spin,
  Switch,
  TextArea,
} from 'antdv-next';

import {
  getLogRetentionApi,
  updateLogRetentionApi,
} from '#/api/caddy/log-retention';
import {
  getLogSnippetApi,
  resetLogSnippetApi,
  updateLogSnippetApi,
} from '#/api/caddy/log-snippet';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { useListDetailQuery } from '#/composables/use-list-detail-query';
import { apiErrorMessage } from '#/utils/api-error-message';

defineOptions({ name: 'SettingsSiteLogCard' });

const queryClient = useQueryClient();

/** 保留天数范围与后端校验一致 */
const RETENTION_MIN = 1;
const RETENTION_MAX = 3650;
/** 后端缺省值与本文一致 */
const RETENTION_DEFAULT = 7;

const TEMPLATE_SAVE_TIP = '模板已保存，需在配置管理页应用后生效';
const RETENTION_TIP =
  '早于该天数的站点日志每天自动清理一次；早于保留期的日志会被永久删除，看板与日志页更长时间窗的统计将随之失效';

// ---------- 保留天数与自动清理开关 ----------
const retentionDays = ref<number>(RETENTION_DEFAULT);
const retentionEnabled = ref(false);
const seededRetention = ref(false);
const savingRetention = ref(false);

const {
  data: retentionData,
  loading: retentionLoading,
  errorMessage: retentionError,
  refetch: refetchRetention,
} = useListDetailQuery({
  queryKey: ['caddy', 'log-retention'],
  queryFn: () => getLogRetentionApi(withListDetailErrorMode()),
  errorFallback: '加载日志保留天数失败',
});

watch(
  retentionData,
  (data) => {
    if (!data || seededRetention.value) return;
    retentionDays.value = Number(data.retentionDays) || RETENTION_DEFAULT;
    retentionEnabled.value = Boolean(data.enabled);
    seededRetention.value = true;
  },
  { immediate: true },
);

async function handleSaveRetention() {
  const days = Number(retentionDays.value);
  if (!Number.isInteger(days) || days < RETENTION_MIN || days > RETENTION_MAX) {
    message.warning(`保留天数需为 ${RETENTION_MIN} 至 ${RETENTION_MAX} 之间的整数`);
    return;
  }
  savingRetention.value = true;
  try {
    await updateLogRetentionApi(
      { retentionDays: days, enabled: retentionEnabled.value },
      withListDetailErrorMode(),
    );
    message.success('保留策略已保存');
    seededRetention.value = false;
    await invalidateListDetailQueries(queryClient, ['caddy', 'log-retention']);
    await refetchRetention();
  } catch (error) {
    message.error(apiErrorMessage(error, '保存保留策略失败'));
  } finally {
    savingRetention.value = false;
  }
}

// ---------- 日志模板 ----------
const snippetRaw = ref('');
const seededSnippet = ref(false);
const savingSnippet = ref(false);
const resettingSnippet = ref(false);

const {
  data: snippetData,
  loading: snippetLoading,
  errorMessage: snippetError,
  refetch: refetchSnippet,
} = useListDetailQuery({
  queryKey: ['caddy', 'log-snippet'],
  queryFn: () => getLogSnippetApi(withListDetailErrorMode()),
  errorFallback: '加载日志模板失败',
});

watch(
  snippetData,
  (data) => {
    if (!data || seededSnippet.value) return;
    snippetRaw.value = data.raw ?? '';
    seededSnippet.value = true;
  },
  { immediate: true },
);

/** 任一查询在途都禁用写操作，避免用旧值覆盖 */
const busy = computed(() => retentionLoading.value || snippetLoading.value);

async function handleSaveSnippet() {
  // 空模板会让应用时的站点 import 落空，与后端校验保持一致
  if (!snippetRaw.value.trim()) {
    message.warning('日志模板不能为空');
    return;
  }
  savingSnippet.value = true;
  try {
    await updateLogSnippetApi({ raw: snippetRaw.value }, withListDetailErrorMode());
    message.success(TEMPLATE_SAVE_TIP);
    seededSnippet.value = false;
    await invalidateListDetailQueries(queryClient, ['caddy', 'log-snippet']);
    await refetchSnippet();
  } catch (error) {
    message.error(apiErrorMessage(error, '保存日志模板失败'));
  } finally {
    savingSnippet.value = false;
  }
}

/** 恢复默认：二次确认后由后端重置，返回的默认原文直接回填编辑器 */
function handleResetSnippet() {
  Modal.confirm({
    title: '恢复默认日志模板',
    content:
      '将丢弃当前自定义模板，恢复为内置的 (subdomain-log) 定义（参数化 name/size/level、roll_keep 1）。此操作不可撤销。',
    okType: 'danger',
    okText: '恢复默认',
    cancelText: '取消',
    async onOk() {
      resettingSnippet.value = true;
      try {
        const data = await resetLogSnippetApi(withListDetailErrorMode());
        snippetRaw.value = data?.raw ?? '';
        message.success('已恢复默认模板');
        seededSnippet.value = false;
        await invalidateListDetailQueries(queryClient, ['caddy', 'log-snippet']);
        await refetchSnippet();
      } catch (error) {
        message.error(apiErrorMessage(error, '恢复默认失败'));
      } finally {
        resettingSnippet.value = false;
      }
    },
  });
}
</script>

<template>
  <Card title="站点日志" variant="borderless">
    <Alert
      v-if="retentionError"
      class="mb-4"
      type="error"
      show-icon
      :message="retentionError"
    />
    <Alert
      v-if="snippetError"
      class="mb-4"
      type="error"
      show-icon
      :message="snippetError"
    />

    <Spin :spinning="busy">
      <Form layout="vertical">
        <!-- 保留天数与清理开关：一并提交，后端每天按此清理 caddy_logs -->
        <FormItem label="日志保留天数" :extra="RETENTION_TIP">
          <Space>
            <InputNumber
              v-model:value="retentionDays"
              :min="RETENTION_MIN"
              :max="RETENTION_MAX"
              :precision="0"
              class="retention-input"
            />
            <Switch v-model:checked="retentionEnabled" />
            <span>启用自动清理</span>
            <Button
              type="primary"
              :loading="savingRetention"
              :disabled="busy"
              @click="handleSaveRetention"
            >
              保存
            </Button>
          </Space>
        </FormItem>

        <!-- 模板：站点侧通过 import subdomain-log 引用 -->
        <FormItem
          label="日志模板（subdomain-log）"
          extra="模板变更需重新应用站点配置才生效"
        >
          <TextArea
            v-model:value="snippetRaw"
            :rows="12"
            class="snippet-input"
            placeholder="(subdomain-log) { ... }"
          />
        </FormItem>

        <FormItem>
          <Space>
            <Button
              type="primary"
              :loading="savingSnippet"
              :disabled="busy"
              @click="handleSaveSnippet"
            >
              保存模板
            </Button>
            <Button
              danger
              :loading="resettingSnippet"
              :disabled="busy"
              @click="handleResetSnippet"
            >
              恢复默认
            </Button>
          </Space>
        </FormItem>
      </Form>
    </Spin>
  </Card>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}

.retention-input {
  width: 160px;
}

/* 模板原文按等宽字体展示，便于对齐缩进 */
.snippet-input {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.6;
}
</style>
