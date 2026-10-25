<script lang="ts" setup>
import type { CaddyWafSourceApi } from '#/api/caddy/source';

import { computed, reactive, ref } from 'vue';

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
} from 'antdv-next';

import {
  createWafSourceApi,
  deleteWafSourceApi,
  getWafSourceListApi,
  syncWafSourceApi,
  updateWafSourceApi,
} from '#/api/caddy/source';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';
import { apiErrorMessage } from '#/utils/api-error-message';

/**
 * WAF 来源卡：由原「WAF 来源」页面（/caddy/source）并入全局设置，
 * 保留源列表增删改与「同步并激活」核心能力。
 */
defineOptions({ name: 'WafSourcesCard' });

const queryClient = useQueryClient();

const modalVisible = ref(false);
const modalTitle = ref('新增源');
const formState = reactive({
  name: '',
  url: '',
  type: '',
});
const editingId = ref<null | number>(null);

const {
  data: sourcesData,
  loading,
  errorMessage,
  refetch,
} = useListDetailQuery({
  queryKey: qk.caddy.source(),
  queryFn: () => getWafSourceListApi(withListDetailErrorMode()),
  errorFallback: '加载 WAF 源失败',
});

const dataSource = computed(() => sourcesData.value ?? []);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
  { title: '名称', dataIndex: 'name', key: 'name', width: 200 },
  { title: 'URL', dataIndex: 'url', key: 'url', ellipsis: true },
  { title: '类型', dataIndex: 'kind', key: 'type', width: 100 },
  { title: '状态', dataIndex: 'enabled', key: 'status', width: 100 },
  { title: '操作', key: 'actions', width: 280, fixed: 'right' as const },
];

function handleAdd() {
  editingId.value = null;
  modalTitle.value = '新增源';
  Object.assign(formState, { name: '', url: '', type: '' });
  modalVisible.value = true;
}

function handleEdit(record: CaddyWafSourceApi.WafSource) {
  editingId.value = record.id;
  modalTitle.value = '编辑源';
  Object.assign(formState, {
    name: record.name ?? '',
    url: record.url ?? '',
    type: record.kind || record.type || '',
  });
  modalVisible.value = true;
}

async function handleModalOk() {
  try {
    const payload = {
      name: formState.name,
      url: formState.url,
      type: formState.type,
      kind: formState.type,
    };
    if (editingId.value) {
      await updateWafSourceApi(editingId.value, payload);
      message.success('更新成功');
    } else {
      await createWafSourceApi(payload);
      message.success('创建成功');
    }
    modalVisible.value = false;
    await invalidateListDetailQueries(queryClient, qk.caddy.source());
    await refetch();
  } catch {
    message.error('操作失败');
  }
}

function handleDelete(record: CaddyWafSourceApi.WafSource) {
  Modal.confirm({
    title: '确认删除',
    content: `确定删除「${record.name ?? record.id}」？`,
    async onOk() {
      try {
        await deleteWafSourceApi(record.id);
        message.success('删除成功');
        await invalidateListDetailQueries(queryClient, qk.caddy.source());
        await refetch();
      } catch {
        message.error('删除失败');
      }
    },
  });
}

/** 下载规则包并激活，激活后 /config/security/active 才会有规则文件 */
const syncingId = ref<null | number>(null);
async function handleSync(record: CaddyWafSourceApi.WafSource) {
  syncingId.value = record.id;
  try {
    await syncWafSourceApi(record.id, true);
    message.success('同步并激活成功');
    await invalidateListDetailQueries(queryClient, qk.caddy.source());
    await refetch();
  } catch (error) {
    message.error(apiErrorMessage(error, '同步失败'));
  } finally {
    syncingId.value = null;
  }
}
</script>

<template>
  <Card title="WAF 来源" variant="borderless">
    <template #extra>
      <Button type="primary" @click="handleAdd">新增源</Button>
    </template>

    <Alert
      v-if="errorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="errorMessage"
    />
    <Alert
      v-else
      class="mb-4"
      type="info"
      :show-icon="true"
      message="规则源维护 CRS 规则包来源；「同步并激活」会下载规则包、重构规则布局并热加载 Caddy。"
    />

    <Table
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      row-key="id"
      size="middle"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <Tag :color="record.enabled ? 'green' : 'default'">
            {{ record.enabled ? '启用' : '停用' }}
          </Tag>
        </template>
        <template v-if="column.key === 'actions'">
          <Space :size="6">
            <Button
              size="small"
              type="primary"
              :loading="syncingId === record.id"
              class="table-action-btn"
              @click="handleSync(record as CaddyWafSourceApi.WafSource)"
            >
              同步并激活
            </Button>
            <Button
              size="small"
              class="table-action-btn"
              @click="handleEdit(record as CaddyWafSourceApi.WafSource)"
            >
              编辑
            </Button>
            <Button
              size="small"
              danger
              class="table-action-btn table-action-btn--danger"
              @click="handleDelete(record as CaddyWafSourceApi.WafSource)"
            >
              删除
            </Button>
          </Space>
        </template>
      </template>
    </Table>

    <Modal
      :open="modalVisible"
      :title="modalTitle"
      @cancel="modalVisible = false"
      @ok="handleModalOk"
    >
      <Form
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 16 }"
        style="margin-top: 16px"
      >
        <FormItem label="名称">
          <Input v-model:value="formState.name" placeholder="源名称" />
        </FormItem>
        <FormItem label="URL">
          <Input v-model:value="formState.url" placeholder="源 URL" />
        </FormItem>
        <FormItem label="类型">
          <Input
            v-model:value="formState.type"
            placeholder="如 crs / coraza_engine"
          />
        </FormItem>
      </Form>
    </Modal>
  </Card>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
