<script lang="ts" setup>
import { computed, h, reactive, ref } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import { Page } from '@vben/common-ui';
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

import type { CaddyWafSourceApi } from '#/api/caddy/source';
import {
  createWafSourceApi,
  deleteWafSourceApi,
  getWafSourceListApi,
  updateWafSourceApi,
} from '#/api/caddy/source';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

defineOptions({ name: 'CaddySource' });

const queryClient = useQueryClient();

const modalVisible = ref(false);
const modalTitle = ref('新增源');
const formState = reactive({
  name: '',
  url: '',
  type: '',
});
const editingId = ref<number | null>(null);

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
  {
    title: '类型',
    dataIndex: 'kind',
    key: 'type',
    width: 100,
    customRender: ({ record }: { record: CaddyWafSourceApi.WafSource }) =>
      record.kind || record.type || '-',
  },
  {
    title: '状态',
    dataIndex: 'enabled',
    key: 'status',
    width: 100,
    customRender: ({ record }: { record: CaddyWafSourceApi.WafSource }) => {
      const enabled = record.enabled;
      const text =
        record.status ??
        (enabled === false ? 'inactive' : enabled ? 'active' : 'unknown');
      const colorMap: Record<string, string> = {
        active: 'green',
        inactive: 'default',
        error: 'red',
        synced: 'blue',
      };
      return h(Tag, { color: colorMap[text] ?? 'default' }, () => text);
    },
  },
  { title: '操作', key: 'actions', width: 180, fixed: 'right' as const },
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
</script>

<template>
  <Page title="WAF 源">
    <Alert
      v-if="errorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="errorMessage"
    />
    <Card>
      <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
        <span style="font-size: 16px; font-weight: 500">源列表</span>
        <Button type="primary" @click="handleAdd">新增源</Button>
      </div>

      <Table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'actions'">
            <Space :size="6">
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
    </Card>

    <Modal
      :open="modalVisible"
      :title="modalTitle"
      @cancel="modalVisible = false"
      @ok="handleModalOk"
    >
      <Form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }" style="margin-top: 16px">
        <FormItem label="名称">
          <Input v-model:value="formState.name" placeholder="源名称" />
        </FormItem>
        <FormItem label="URL">
          <Input v-model:value="formState.url" placeholder="源 URL" />
        </FormItem>
        <FormItem label="类型">
          <Input v-model:value="formState.type" placeholder="如 crs / coraza_engine" />
        </FormItem>
      </Form>
    </Modal>
  </Page>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
