<script lang="ts" setup>
import { h, onMounted, reactive, ref } from 'vue';
import { Page } from '@vben/common-ui';

import {
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
} from 'ant-design-vue';

import {
  createWafSourceApi,
  deleteWafSourceApi,
  getWafSourceListApi,
  updateWafSourceApi,
} from '#/api/caddy/source';
import type { CaddyWafSourceApi } from '#/api/caddy/source';

// --------------- state ---------------
const loading = ref(false);
const dataSource = ref<CaddyWafSourceApi.WafSource[]>([]);

// modal
const modalVisible = ref(false);
const modalTitle = ref('Add Log Source');
const formState = reactive<Record<string, any>>({});
const editingId = ref<number | null>(null);

// --------------- columns ---------------
const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
    width: 70,
  },
  {
    title: 'Name',
    dataIndex: 'name',
    key: 'name',
    width: 200,
  },
  {
    title: 'URL',
    dataIndex: 'url',
    key: 'url',
    ellipsis: true,
  },
  {
    title: 'Type',
    dataIndex: 'type',
    key: 'type',
    width: 100,
  },
  {
    title: 'Status',
    dataIndex: 'status',
    key: 'status',
    width: 100,
    customRender: ({ text }: { text: string }) => {
      const colorMap: Record<string, string> = {
        active: 'green',
        inactive: 'default',
        error: 'red',
        synced: 'blue',
      };
      return h(Tag, { color: colorMap[text] ?? 'default' }, () => text ?? 'unknown');
    },
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 160,
    fixed: 'right' as const,
  },
];

// --------------- data fetching ---------------
async function fetchSources() {
  loading.value = true;
  try {
    dataSource.value = await getWafSourceListApi();
  } catch {
    message.error('Failed to load log sources');
  } finally {
    loading.value = false;
  }
}

// --------------- CRUD handlers ---------------
function handleAdd() {
  editingId.value = null;
  modalTitle.value = 'Add Log Source';
  Object.assign(formState, { name: '', url: '', type: '' });
  modalVisible.value = true;
}

function handleEdit(record: CaddyWafSourceApi.WafSource) {
  editingId.value = record.id;
  modalTitle.value = 'Edit Log Source';
  Object.assign(formState, {
    name: record.name ?? '',
    url: record.url ?? '',
    type: record.type ?? '',
  });
  modalVisible.value = true;
}

async function handleModalOk() {
  try {
    if (editingId.value) {
      await updateWafSourceApi(editingId.value, { ...formState });
      message.success('Updated successfully');
    } else {
      await createWafSourceApi({ ...formState });
      message.success('Created successfully');
    }
    modalVisible.value = false;
    await fetchSources();
  } catch {
    message.error('Operation failed');
  }
}

function handleDelete(record: CaddyWafSourceApi.WafSource) {
  Modal.confirm({
    title: 'Confirm Delete',
    content: `Are you sure you want to delete "${record.name ?? record.id}"?`,
    async onOk() {
      try {
        await deleteWafSourceApi(record.id);
        message.success('Deleted successfully');
        await fetchSources();
      } catch {
        message.error('Delete failed');
      }
    },
  });
}

// --------------- lifecycle ---------------
onMounted(() => {
  fetchSources();
});
</script>

<template>
  <Page title="Log Sources">
    <Card>
      <div style="display: flex; justify-content: space-between; margin-bottom: 16px;">
        <span style="font-size: 16px; font-weight: 500;">Log Source List</span>
        <Button type="primary" @click="handleAdd">Add Source</Button>
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
            <Space>
              <Button size="small" type="link" @click="handleEdit(record)">Edit</Button>
              <Button size="small" type="link" danger @click="handleDelete(record)">
                Delete
              </Button>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <!-- Add/Edit Modal -->
    <Modal
      :open="modalVisible"
      :title="modalTitle"
      @cancel="modalVisible = false"
      @ok="handleModalOk"
    >
      <Form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }" style="margin-top: 16px;">
        <FormItem label="Name">
          <Input v-model:value="formState.name" placeholder="Source name" />
        </FormItem>
        <FormItem label="URL">
          <Input v-model:value="formState.url" placeholder="Source URL" />
        </FormItem>
        <FormItem label="Type">
          <Input v-model:value="formState.type" placeholder="e.g. waf, log, rule" />
        </FormItem>
      </Form>
    </Modal>
  </Page>
</template>
