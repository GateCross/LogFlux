<script lang="ts" setup>
import { onMounted, reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';

import {
  Button,
  Card,
  Form,
  FormItem,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';

import {
  checkWafSourceApi,
  createWafSourceApi,
  deleteWafSourceApi,
  getWafSourceListApi,
  syncWafSourceApi,
  updateWafSourceApi,
} from '#/api/caddy/source';

defineOptions({ name: 'SecuritySource' });

const loading = ref(false);
const dataList = ref<any[]>([]);
const modalVisible = ref(false);
const modalTitle = ref('创建 WAF Source');
const editingId = ref<number | null>(null);

const formState = reactive<Record<string, any>>({
  name: '',
  url: '',
  type: '',
  description: '',
  enabled: true,
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '名称', dataIndex: 'name', key: 'name', width: 180 },
  { title: 'URL', dataIndex: 'url', key: 'url', ellipsis: true },
  { title: '类型', dataIndex: 'type', key: 'type', width: 120 },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '状态', dataIndex: 'enabled', key: 'enabled', width: 100 },
  { title: '操作', key: 'actions', width: 320, fixed: 'right' as const },
];

async function fetchData() {
  loading.value = true;
  try {
    const res = await getWafSourceListApi();
    dataList.value = Array.isArray(res) ? res : (res?.data ?? []);
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false;
  }
}

function handleCreate() {
  editingId.value = null;
  modalTitle.value = '创建 WAF Source';
  Object.assign(formState, { name: '', url: '', type: '', description: '', enabled: true });
  modalVisible.value = true;
}

function handle编辑(record: any) {
  editingId.value = record.id;
  modalTitle.value = '编辑 WAF Source';
  Object.assign(formState, {
    name: record.name ?? '',
    url: record.url ?? '',
    type: record.type ?? '',
    description: record.description ?? '',
    enabled: record.enabled ?? true,
  });
  modalVisible.value = true;
}

async function handleSubmit() {
  try {
    if (editingId.value !== null) {
      await updateWafSourceApi(editingId.value, { ...formState });
      message.success('Updated 成功');
    } else {
      await createWafSourceApi({ ...formState });
      message.success('Created 成功');
    }
    modalVisible.value = false;
    await fetchData();
  } catch {
    // error handled by interceptor
  }
}

async function handle删除(id: number) {
  try {
    await deleteWafSourceApi(id);
    message.success('删除d 成功');
    await fetchData();
  } catch {
    // error handled by interceptor
  }
}

async function handleCheck(id: number) {
  try {
    await checkWafSourceApi(id);
    message.success('Check triggered');
  } catch {
    // error handled by interceptor
  }
}

async function handleSync(id: number) {
  try {
    await syncWafSourceApi(id);
    message.success('Sync completed');
    await fetchData();
  } catch {
    // error handled by interceptor
  }
}

onMounted(() => {
  fetchData();
});
</script>

<template>
  <Page title="WAF Sources" description="Manage WAF rule sources. Add, edit, check, and sync rule sources.">
    <Card>
      <template #extra>
        <Space>
          <Button type="primary" @click="handleCreate">新增 Source</Button>
          <Button @click="fetchData">Refresh</Button>
        </Space>
      </template>

      <Table
        :columns="columns"
        :data-source="dataList"
        :loading="loading"
        row-key="id"
        :scroll="{ x: 1000 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'enabled'">
            <Tag :color="record.enabled ? 'green' : 'red'">
              {{ record.enabled ? 'Enabled' : 'Disabled' }}
            </Tag>
          </template>
          <template v-if="column.key === 'actions'">
            <Space>
              <Button size="small" type="link" @click="handle编辑(record)">编辑</Button>
              <Button size="small" type="link" @click="handleCheck(record.id)">Check</Button>
              <Popconfirm
                title="Sync this source? This may take a while."
                @confirm="handleSync(record.id)"
              >
                <Button size="small" type="link">Sync</Button>
              </Popconfirm>
              <Popconfirm
                title="Are you sure to delete this source?"
                @confirm="handle删除(record.id)"
              >
                <Button size="small" type="link" danger>删除</Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <Modal
      v-model:open="modalVisible"
      :title="modalTitle"
      @ok="handleSubmit"
      @cancel="() => (modalVisible = false)"
    >
      <Form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <FormItem label="Name">
          <Input v-model:value="formState.name" placeholder="Source name" />
        </FormItem>
        <FormItem label="URL">
          <Input v-model:value="formState.url" placeholder="Source URL" />
        </FormItem>
        <FormItem label="Type">
          <Input v-model:value="formState.type" placeholder="e.g. coreruleset, custom" />
        </FormItem>
        <FormItem label="Description">
          <Input v-model:value="formState.description" placeholder="Description" />
        </FormItem>
      </Form>
    </Modal>
  </Page>
</template>
