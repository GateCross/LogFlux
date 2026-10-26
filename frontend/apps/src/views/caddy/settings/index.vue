<script setup lang="ts">
import { ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useQueryClient } from '@tanstack/vue-query';

import { Page } from '@vben/common-ui';

import {
  Alert,
  Button,
  Card,
  Form,
  FormItem,
  Input,
  Modal,
  Select,
  Space,
  Spin,
  Switch,
  TextArea,
  message,
} from 'antdv-next';

import {
  getEntryBlockApi,
  getGlobalOptionsApi,
  resetEntryBlockApi,
  updateEntryBlockApi,
  updateGlobalOptionsApi,
} from '#/api/caddy/global-options';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import MonacoCodeEditor from '#/components/code-editor/MonacoCodeEditor.vue';
import { useListDetailQuery } from '#/composables/use-list-detail-query';
import { apiErrorMessage } from '#/utils/api-error-message';
import SettingsSiteLogCard from './components/SettingsSiteLogCard.vue';
import SettingsWafCard from './components/SettingsWafCard.vue';
import WafSourcesCard from './components/WafSourcesCard.vue';

defineOptions({ name: 'CaddySettings' });

const route = useRoute();
const vueRouter = useRouter();

/** 保存成功提示：写明还需在「配置管理」页应用（后端仅落库，不触发应用） */
const SAVE_SUCCESS_TIP = '已保存，需在配置管理页应用后生效';
/** 恢复默认成功提示 */
const RESET_SUCCESS_TIP = '已恢复默认模板';

const queryClient = useQueryClient();

// ---------- WAF 节点与「高级配置」入口 ----------
/** WAF 卡当前节点：由本页持有，深链 ?serverId= 与高级配置跳转共用 */
const advancedServerId = ref<number>();

// 深链初始化：/caddy/settings?serverId=123 直达指定节点
const initialServerId = Number(route.query.serverId);
if (Number.isFinite(initialServerId) && initialServerId > 0) {
  advancedServerId.value = initialServerId;
}

/** 进入配置管理页（菜单已隐藏，路由保留）：带上当前节点上下文 */
function goAdvancedConfig() {
  void vueRouter.push({
    name: 'CaddyConfig',
    query: advancedServerId.value
      ? { serverId: String(advancedServerId.value) }
      : undefined,
  });
}

// ---------- 全局选项 ----------
const savingOptions = ref(false);
const seededOptions = ref(false);
/** undefined 表示已清空，提交时映射回 ''（不写入显式取值） */
const autoHttps = ref<string | undefined>('');
const trustedProxies = ref<string[]>([]);
const debug = ref(false);
const email = ref('');
const extraRawOptions = ref('');

/** 仅列两个显式取值，空值语义见卡内"说明" */
const autoHttpsOptions = [
  { label: '关闭跳转（disable_redirects）', value: 'disable_redirects' },
  { label: '完全关闭（off）', value: 'off' },
];

const {
  data: remoteOptions,
  loading: optionsLoading,
  errorMessage: optionsError,
  refetch: refetchOptions,
} = useListDetailQuery({
  queryKey: qk.caddy.globalOptions(),
  queryFn: () => getGlobalOptionsApi(withListDetailErrorMode()),
  errorFallback: '加载全局选项失败',
});

watch(
  remoteOptions,
  (data) => {
    if (!data || seededOptions.value) return;
    autoHttps.value = data.autoHttps ?? '';
    trustedProxies.value = [...(data.trustedProxies ?? [])];
    debug.value = Boolean(data.debug);
    email.value = data.email ?? '';
    extraRawOptions.value = data.extraRawOptions ?? '';
    seededOptions.value = true;
  },
  { immediate: true },
);

async function handleSaveOptions() {
  savingOptions.value = true;
  try {
    await updateGlobalOptionsApi(
      {
        // 清空(undefined)映射回空串，与契约的"默认"对齐
        autoHttps: autoHttps.value ?? '',
        debug: debug.value,
        email: email.value,
        extraRawOptions: extraRawOptions.value,
        trustedProxies: trustedProxies.value,
      },
      withListDetailErrorMode(),
    );
    message.success(SAVE_SUCCESS_TIP);
    seededOptions.value = false;
    await invalidateListDetailQueries(queryClient, qk.caddy.globalOptions());
    await refetchOptions();
  } catch (error) {
    message.error(apiErrorMessage(error, '保存全局选项失败'));
  } finally {
    savingOptions.value = false;
  }
}

// ---------- 系统入口(:80) ----------
const savingEntry = ref(false);
const resettingEntry = ref(false);
const entryRaw = ref('');
const seededEntry = ref(false);

const {
  data: remoteEntry,
  loading: entryLoading,
  errorMessage: entryError,
  refetch: refetchEntry,
} = useListDetailQuery({
  queryKey: qk.caddy.entryBlock(),
  queryFn: () => getEntryBlockApi(withListDetailErrorMode()),
  errorFallback: '加载系统入口配置失败',
});

watch(
  remoteEntry,
  (data) => {
    if (!data || seededEntry.value) return;
    entryRaw.value = data.raw ?? '';
    seededEntry.value = true;
  },
  { immediate: true },
);

async function handleSaveEntry() {
  // 空值兜底：后端同样拒绝空模板，文案与其保持一致
  if (!entryRaw.value.trim()) {
    message.warning('入口块模板不能为空');
    return;
  }
  savingEntry.value = true;
  try {
    await updateEntryBlockApi(
      { raw: entryRaw.value },
      withListDetailErrorMode(),
    );
    message.success(SAVE_SUCCESS_TIP);
    seededEntry.value = false;
    await invalidateListDetailQueries(queryClient, qk.caddy.entryBlock());
    await refetchEntry();
  } catch (error) {
    message.error(apiErrorMessage(error, '保存系统入口配置失败'));
  } finally {
    savingEntry.value = false;
  }
}

/** 恢复默认：二次确认后调用重置接口，经 invalidate + refetch 让 watch 把编辑器刷成默认模板 */
function handleResetEntry() {
  Modal.confirm({
    title: '恢复默认模板',
    content:
      '将丢弃当前自定义 :80 模板，恢复为内置默认模板（内网来源按 Host 分流、不跳 HTTPS；其余来源跳转 HTTPS）。此操作不可撤销。',
    okType: 'danger',
    okText: '恢复默认',
    cancelText: '取消',
    async onOk() {
      resettingEntry.value = true;
      try {
        await resetEntryBlockApi(withListDetailErrorMode());
        message.success(RESET_SUCCESS_TIP);
        seededEntry.value = false;
        await invalidateListDetailQueries(queryClient, qk.caddy.entryBlock());
        await refetchEntry();
      } catch (error) {
        message.error(apiErrorMessage(error, '恢复默认失败'));
      } finally {
        resettingEntry.value = false;
      }
    },
  });
}
</script>

<template>
  <Page
    title="全局设置"
    description="全局选项、WAF 防护与系统入口(:80) 模板；全局选项与入口模板保存后需在「配置管理」页应用才会合并进 Caddyfile"
  >
    <div class="settings-page">
      <Card title="全局选项" variant="borderless">
        <Spin :spinning="optionsLoading">
          <Alert
            v-if="optionsError"
            class="mb-4"
            type="error"
            show-icon
            :message="optionsError"
          />
          <Form layout="vertical" class="options-form">
            <FormItem label="自动 HTTPS">
              <Select
                v-model:value="autoHttps"
                :options="autoHttpsOptions"
                allow-clear
                placeholder="默认：由 Caddy 自动处理"
              />
            </FormItem>

            <FormItem label="受信任代理">
              <Select
                v-model:value="trustedProxies"
                mode="tags"
                :open="false"
                :suffix-icon="null"
                placeholder="输入 IP 或 CIDR 后回车，可多个"
              />
            </FormItem>

            <FormItem label="调试模式">
              <Switch v-model:checked="debug" />
            </FormItem>

            <FormItem label="ACME 邮箱">
              <Input
                v-model:value="email"
                placeholder="用于证书申请的邮箱，可留空"
                autocomplete="off"
              />
            </FormItem>

            <FormItem label="附加全局选项" class="form-item-full">
              <TextArea
                v-model:value="extraRawOptions"
                :rows="6"
                placeholder="追加到全局 options 块的 Caddyfile 原文，可多行"
              />
            </FormItem>

            <FormItem label="说明" class="form-item-full">
              <Alert type="info" :show-icon="true">
                <template #message>
                  自动 HTTPS
                  未选择或清空时由 Caddy 自动处理（默认）；附加原文与上方设置同名/同类的指令将被设置项接管；附加原文逐行写入全局块，不支持大括号块结构。
                </template>
              </Alert>
            </FormItem>

            <FormItem class="form-item-full">
              <Button
                type="primary"
                :loading="savingOptions"
                :disabled="optionsLoading"
                @click="handleSaveOptions"
              >
                保存
              </Button>
            </FormItem>
          </Form>
        </Spin>
      </Card>

      <!-- 站点日志：保留天数与 (subdomain-log) 模板 -->
      <SettingsSiteLogCard />

      <!-- WAF 防护：复用 simple-waf 接口与 WafPanel，节点级状态 -->
      <SettingsWafCard v-model:server-id="advancedServerId" />

      <!-- WAF 来源：原「WAF 来源」页面并入 -->
      <WafSourcesCard />

      <Card title="系统入口(:80)" variant="borderless">
        <Spin :spinning="entryLoading">
          <Alert
            v-if="entryError"
            class="mb-4"
            type="error"
            show-icon
            :message="entryError"
          />
          <Form layout="vertical">
            <FormItem label="Caddyfile 模板">
              <MonacoCodeEditor
                v-model="entryRaw"
                language="shell"
                :height="480"
              />
            </FormItem>

            <FormItem label="说明">
              <Alert type="info" :show-icon="true">
                <template #message>
                  此模板在配置应用时整体替换 Caddyfile 中的
                  :80 站点块；模板不能为空，需要内置模板可点「恢复默认」；内网直连不跳
                  HTTPS 需同时在全局选项把自动 HTTPS 设为「关闭跳转」。
                </template>
              </Alert>
            </FormItem>

            <FormItem>
              <Space>
                <Button
                  type="primary"
                  :loading="savingEntry"
                  :disabled="entryLoading"
                  @click="handleSaveEntry"
                >
                  保存
                </Button>
                <Button
                  danger
                  :loading="resettingEntry"
                  :disabled="entryLoading"
                  @click="handleResetEntry"
                >
                  恢复默认
                </Button>
              </Space>
            </FormItem>
          </Form>
        </Spin>
      </Card>

      <!-- 高级配置入口：配置管理已从菜单隐藏，这里是唯一常规入口 -->
      <Card title="高级配置" variant="borderless">
        <div class="advanced-row">
          <div class="advanced-text">
            站点块 / Caddyfile
            原文编辑、Docker 服务发现、配置历史与回滚等高级操作，已收敛到配置管理页。
          </div>
          <Button type="primary" @click="goAdvancedConfig">
            进入高级配置
          </Button>
        </div>
      </Card>
    </div>
  </Page>
</template>

<style scoped>
.settings-page {
  display: grid;
  gap: 16px;
  padding: 16px;
}

/* 全局选项短字段两列铺满卡宽，长内容通栏；窄屏（<720px）退回单列 */
.options-form {
  column-gap: 24px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.form-item-full {
  grid-column: 1 / -1;
}

/* 高级配置入口：文案与按钮左右分布，窄屏换行 */
.advanced-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.advanced-text {
  color: #8c8c8c;
  font-size: 13px;
}

@media (max-width: 720px) {
  .advanced-row {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 720px) {
  .options-form {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
