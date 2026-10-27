<script lang="ts" setup>
import type { CaddyCertificateApi } from '#/api/caddy/certificate';

import { computed, reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';

import { useQueryClient } from '@tanstack/vue-query';
import {
  Alert,
  Button,
  Card,
  Form,
  FormItem,
  Input,
  message,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Upload,
} from 'antdv-next';

import { getCertificateListApi } from '#/api/caddy/certificate';
import {
  deleteCertificateApi,
  refreshCertificatesApi,
  uploadCertificateApi,
} from '#/api/caddy/certificate';
import { getCaddyServerListApi } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';
import { apiErrorMessage } from '#/utils/api-error-message';

import { serverLabel } from '../catalog/composables/useServiceCatalogPage';
import {
  certDaysBadge,
  certSourceLabel,
  certStatusLabel,
  certStatusColor,
  formatCertDateRange,
  isLocalServer,
} from './certificate-utils';

/**
 * 证书中心：双源合并库存（卷扫描真值 + Caddyfile 声明）。
 * remote 服务器只有声明行（远端（仅声明）），有效期与天数如实显示占位。
 */
defineOptions({ name: 'CaddyCertificates' });

const queryClient = useQueryClient();

const {
  data: serversData,
  loading: loadingServers,
  errorMessage: serversErrorMessage,
} = useListDetailQuery({
  queryKey: qk.caddy.servers(),
  queryFn: () => getCaddyServerListApi(withListDetailErrorMode()),
  errorFallback: '获取服务器列表失败',
});

const servers = computed(() => serversData.value ?? []);
const serverNameMap = computed(
  () => new Map(servers.value.map((server) => [Number(server.id), serverLabel(server)])),
);

/** 库存过滤：undefined = 全部服务器 */
const filterServerId = ref<number | undefined>();
const filterOptions = computed(() =>
  servers.value.map((server) => ({
    label: serverLabel(server),
    value: Number(server.id),
  })),
);

function handleFilterChange(value: unknown) {
  filterServerId.value = value == null || value === '' ? undefined : Number(value);
}

const {
  data: certData,
  loading: loadingCerts,
  errorMessage: certsErrorMessage,
  refetch,
} = useListDetailQuery({
  queryKey: computed(() => qk.caddy.certificates(filterServerId.value)),
  queryFn: () =>
    getCertificateListApi(filterServerId.value, withListDetailErrorMode()),
  errorFallback: '获取证书库存失败',
});

const dataSource = computed(() => certData.value ?? []);

const columns = [
  { title: '域名', dataIndex: 'primaryDomain', key: 'domains', ellipsis: true },
  { title: '签发者', dataIndex: 'issuer', key: 'issuer', width: 160, ellipsis: true },
  { title: '有效期', key: 'validity', width: 190 },
  { title: '剩余天数', key: 'daysLeft', width: 110 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 130 },
  { title: '来源', dataIndex: 'source', key: 'source', width: 110 },
  { title: '存储路径', dataIndex: 'storagePath', key: 'storagePath', width: 220, ellipsis: true },
  { title: '所属服务器', key: 'server', width: 150, ellipsis: true },
  { title: '操作', key: 'actions', width: 90, fixed: 'right' as const },
];

/** 复制存储路径，供站点抽屉「手动证书」直接引用 */
async function handleCopyStoragePath(record: CaddyCertificateApi.CertificateItem) {
  const text = (record.storagePath || '').trim();
  if (!text) {
    message.warning('该证书没有存储路径');
    return;
  }
  try {
    await navigator.clipboard.writeText(text);
    message.success('存储路径已复制');
  } catch {
    message.error('复制失败');
  }
}

/** 失效全部证书库存缓存（前缀匹配所有 serverId 维度） */
async function invalidateCertificates() {
  await invalidateListDetailQueries(queryClient, ['caddy', 'certificates']);
}

const refreshing = ref(false);
async function handleRefresh() {
  refreshing.value = true;
  try {
    await refreshCertificatesApi(filterServerId.value);
    message.success('证书库存已刷新');
    await invalidateCertificates();
    await refetch();
  } catch (error) {
    message.error(apiErrorMessage(error, '刷新证书库存失败'));
  } finally {
    refreshing.value = false;
  }
}

function handleDelete(record: CaddyCertificateApi.CertificateItem) {
  Modal.confirm({
    title: '确认删除',
    content: `确定删除「${record.primaryDomain || record.id}」的证书记录？`,
    async onOk() {
      try {
        await deleteCertificateApi(record.id);
        message.success('删除成功');
        await invalidateCertificates();
        await refetch();
      } catch {
        // 失败提示由全局请求拦截器统一 toast
      }
    },
  });
}

// 上传：仅 local 服务器有文件通道，remote 禁选并标注
const uploadOpen = ref(false);
const uploading = ref(false);
const uploadError = ref('');
const certFile = ref<File | null>(null);
const keyFile = ref<File | null>(null);
const uploadForm = reactive({
  serverId: undefined as number | undefined,
  domains: '',
});

const uploadServerOptions = computed(() =>
  servers.value.map((server) => {
    const label = serverLabel(server);
    return {
      label: isLocalServer(server) ? label : `${label}（远端，无文件通道）`,
      value: Number(server.id),
      disabled: !isLocalServer(server),
    };
  }),
);
const hasLocalServer = computed(() => servers.value.some(isLocalServer));

function openUpload() {
  uploadError.value = '';
  certFile.value = null;
  keyFile.value = null;
  uploadForm.domains = '';
  const firstLocal = servers.value.find(isLocalServer);
  uploadForm.serverId = firstLocal ? Number(firstLocal.id) : undefined;
  uploadOpen.value = true;
}

function beforeCertUpload(file: File) {
  certFile.value = file;
  return false;
}

function beforeKeyUpload(file: File) {
  keyFile.value = file;
  return false;
}

function removeCertFile() {
  certFile.value = null;
}

function removeKeyFile() {
  keyFile.value = null;
}

async function handleUploadOk() {
  if (!uploadForm.serverId) {
    uploadError.value = '请选择上传目标服务器';
    return;
  }
  if (!certFile.value || !keyFile.value) {
    uploadError.value = '请选择证书文件与私钥文件';
    return;
  }
  uploading.value = true;
  uploadError.value = '';
  try {
    await uploadCertificateApi(
      {
        certFile: certFile.value,
        keyFile: keyFile.value,
        serverId: uploadForm.serverId,
        domains: uploadForm.domains,
      },
      withListDetailErrorMode(),
    );
    message.success('证书已上传');
    uploadOpen.value = false;
    await invalidateCertificates();
    await refetch();
  } catch (error) {
    uploadError.value = apiErrorMessage(error, '上传证书失败');
  } finally {
    uploading.value = false;
  }
}
</script>

<template>
  <Page title="证书中心" description="证书库存与到期状态：ACME 自动续期由 Caddy 负责，面板只读展示">
    <Alert
      v-if="serversErrorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="serversErrorMessage"
    />
    <Alert
      v-if="certsErrorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="certsErrorMessage"
    />

    <Card variant="borderless">
      <div class="toolbar">
        <Select
          v-if="filterOptions.length > 1"
          :value="filterServerId"
          :options="filterOptions"
          :loading="loadingServers"
          allow-clear
          placeholder="全部服务器"
          class="filter-select"
          @change="handleFilterChange"
        />
        <Space>
          <Tooltip title="重新扫描证书卷并合并 Caddyfile 声明（remote 服务器仅同步声明）">
            <Button :loading="refreshing" @click="handleRefresh">手动刷新</Button>
          </Tooltip>
          <Button type="primary" @click="openUpload">上传证书</Button>
        </Space>
      </div>

      <Table
        :columns="columns"
        :data-source="dataSource"
        :loading="loadingCerts"
        :locale="{ emptyText: '暂无证书记录' }"
        row-key="id"
        size="middle"
        :scroll="{ x: 1320 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'domains'">
            <Tooltip :title="record.domains || record.primaryDomain">
              <div>{{ record.primaryDomain || '—' }}</div>
            </Tooltip>
            <div
              v-if="record.domains && record.domains !== record.primaryDomain"
              class="san-note"
            >
              {{ record.domains }}
            </div>
          </template>
          <template v-else-if="column.key === 'validity'">
            {{ formatCertDateRange(record.notBefore, record.notAfter) }}
          </template>
          <template v-else-if="column.key === 'daysLeft'">
            <Tag :color="certDaysBadge(record).color">
              {{ certDaysBadge(record).text }}
            </Tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <Tag :color="certStatusColor(record.status)">
              {{ certStatusLabel(record.status) }}
            </Tag>
          </template>
          <template v-else-if="column.key === 'source'">
            {{ certSourceLabel(record.source) }}
          </template>
          <template v-else-if="column.key === 'storagePath'">
            <div v-if="record.storagePath" class="storage-path-cell">
              <Tooltip :title="record.storagePath">
                <span class="storage-path-text">{{ record.storagePath }}</span>
              </Tooltip>
              <Button
                size="small"
                type="text"
                class="table-action-btn"
                @click="handleCopyStoragePath(record as CaddyCertificateApi.CertificateItem)"
              >
                复制
              </Button>
            </div>
            <span v-else>—</span>
          </template>
          <template v-else-if="column.key === 'server'">
            {{ serverNameMap.get(record.serverId) || `Server #${record.serverId}` }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <Button
              size="small"
              danger
              class="table-action-btn table-action-btn--danger"
              @click="handleDelete(record as CaddyCertificateApi.CertificateItem)"
            >
              删除
            </Button>
          </template>
        </template>
      </Table>
    </Card>

    <Modal
      v-model:open="uploadOpen"
      title="上传证书"
      :confirm-loading="uploading"
      :ok-button-props="{ disabled: !hasLocalServer }"
      @ok="handleUploadOk"
    >
      <Alert
        v-if="!hasLocalServer"
        class="mb-3"
        type="warning"
        show-icon
        message="没有可上传的本地服务器：远端节点无文件通道，不能上传证书文件"
      />
      <Alert
        v-if="uploadError"
        class="mb-3"
        type="error"
        show-icon
        :message="uploadError"
      />
      <Form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <FormItem label="目标服务器">
          <Select
            v-model:value="uploadForm.serverId"
            :options="uploadServerOptions"
            placeholder="选择本地服务器"
          />
        </FormItem>
        <FormItem label="证书文件">
          <Upload
            :before-upload="beforeCertUpload"
            :max-count="1"
            accept=".pem,.crt,.cer"
            @remove="removeCertFile"
          >
            <Button>选择证书文件（PEM）</Button>
          </Upload>
        </FormItem>
        <FormItem label="私钥文件">
          <Upload
            :before-upload="beforeKeyUpload"
            :max-count="1"
            accept=".pem,.key"
            @remove="removeKeyFile"
          >
            <Button>选择私钥文件（PEM）</Button>
          </Upload>
        </FormItem>
        <FormItem label="域名备注">
          <Input
            v-model:value="uploadForm.domains"
            placeholder="可选，多个域名用逗号分隔；留空则从证书 SAN 解析"
          />
        </FormItem>
      </Form>
      <div class="upload-hint">
        上传后证书存入服务器 /config/certs/ 目录，可在站点抽屉 TLS「手动证书」中引用其路径。
      </div>
    </Modal>
  </Page>
</template>

<style scoped>
.mb-3 {
  margin-bottom: 12px;
}

.mb-4 {
  margin-bottom: 16px;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.filter-select {
  width: 240px;
}

.san-note {
  font-size: 12px;
  color: hsl(var(--muted-foreground));
}

.storage-path-cell {
  display: flex;
  gap: 4px;
  align-items: center;
  min-width: 0;
}

.storage-path-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.upload-hint {
  margin-top: 4px;
  font-size: 12px;
  color: hsl(var(--muted-foreground));
}
</style>
