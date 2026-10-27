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
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';
import { apiErrorMessage } from '#/utils/api-error-message';

import {
  certDaysBadge,
  certSourceLabel,
  certStatusLabel,
  certStatusColor,
  formatCertDateRange,
} from './certificate-utils';

/**
 * 证书中心：双源合并库存（卷扫描真值 + Caddyfile 声明）。
 * 只有声明没有真值的行如实显示占位。
 */
defineOptions({ name: 'CaddyCertificates' });

const queryClient = useQueryClient();

const {
  data: certData,
  loading: loadingCerts,
  errorMessage: certsErrorMessage,
  refetch,
} = useListDetailQuery({
  queryKey: qk.caddy.certificates(),
  queryFn: () => getCertificateListApi(withListDetailErrorMode()),
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
  { title: '操作', key: 'actions', width: 90, fixed: 'right' as const },
];

/** 复制存储路径，供站点 TLS「手动证书」直接引用 */
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

/** 失效证书库存缓存 */
async function invalidateCertificates() {
  await invalidateListDetailQueries(queryClient, ['caddy', 'certificates']);
}

const refreshing = ref(false);
async function handleRefresh() {
  refreshing.value = true;
  try {
    await refreshCertificatesApi();
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

// 上传：证书落盘到本机 /config/certs/，没有服务器可选
const uploadOpen = ref(false);
const uploading = ref(false);
const uploadError = ref('');
const certFile = ref<File | null>(null);
const keyFile = ref<File | null>(null);
const uploadForm = reactive({
  domains: '',
});

function openUpload() {
  uploadError.value = '';
  certFile.value = null;
  keyFile.value = null;
  uploadForm.domains = '';
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
      v-if="certsErrorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="certsErrorMessage"
    />

    <Card variant="borderless">
      <div class="toolbar">
        <Space>
          <Tooltip title="重新扫描证书卷并合并 Caddyfile 声明">
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
      @ok="handleUploadOk"
    >
      <Alert
        v-if="uploadError"
        class="mb-3"
        type="error"
        show-icon
        :message="uploadError"
      />
      <Form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
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
        上传后证书存入 /config/certs/ 目录，可在站点 TLS「手动证书」中引用其路径。
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
  justify-content: flex-end;
  margin-bottom: 16px;
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
