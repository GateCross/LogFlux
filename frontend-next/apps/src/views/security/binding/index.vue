<script lang="ts" setup>
import { onMounted, reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';

import {
  Button,
  Card,
  Form,
  FormItem,
  Input,
  message,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';

import {
  createWafPolicyBindingApi,
  deleteWafPolicyBindingApi,
  getWafPolicyBindingListApi,
  updateWafPolicyBindingApi,
} from '#/api/caddy/policy';

defineOptions({ name: 'SecurityBinding' });

const loading = ref(false);
const dataList = ref<any[]>([]);
const modalVisible = ref(false);
const modalTitle = ref('创建 Binding');
const editingId = ref<number | null>(null);

const formState = reactive<Record<string, any>>({
  policyId: '',
  serverName: '',
  path: '',
  enabled: true,
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: 'Policy ID', dataIndex: 'policyId', key: 'policyId', width: 120 },
  { title: 'Server Name', dataIndex: 'serverName', key: 'serverName', width: 200 },
  { title: 'Path', dataIndex: 'path', key: 'path', ellipsis: true },
  { title: '状态', dataIndex: 'enabled', key: 'enabled', width: 100 },
  { title: '操作', key: 'actions', width: 160, fixed: 'right' as const },
];

async function fetchData() {
  loading.value = true;
  try {
    const res = await getWafPolicyBindingListApi();
    dataList.value = Array.isArray(res) ? res : (res?.data ?? []);
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false;
  }
}

function handleCreate() {
  editingId.value = null;
  modalTitle.value = '创建 Binding';
  Object.assign(formState, { policyId: '', serverName: '', path: '', enabled: true });
  modalVisible.value = true;
}

function handle编辑(record: any) {
  editingId.value = record.id;
  modalTitle.value = '编辑 Binding';
  Object.assign(formState, {
    policyId: record.policyId ?? '',
    serverName: record.serverName ?? '',
    path: record.path ?? '',
    enabled: record.enabled ?? true,
  });
  modalVisible.value = true;
}

async function handleSubmit() {
  try {
    if (editingId.value !== null) {
      await updateWafPolicyBindingApi(editingId.value, { ...formState });
      message.success('Updated 成功');
    } else {
      await createWafPolicyBindingApi({ ...formState });
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
    await deleteWafPolicyBindingApi(id);
    message.success('删除d 成功');
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
  <Page title="Policy Bindings" description="Bind WAF policies to server names and URL paths.">
    <Card>
      <template #extra>
        <Space>
          <Button type="primary" @click="handleCreate">新增 Binding</Button>
          <Button @click="fetchData">Refresh</Button>
        </Space>
      </template>

      <Table
        :columns="columns"
        :data-source="dataList"
        :loading="loading"
        row-key="id"
        :scroll="{ x: 800 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'enabled'">
            <Tag :color="record.enabled ? 'green' : 'red'">
              {{ record.enabled ? 'Active' : 'Inactive' }}
            </Tag>
          </template>
          <template v-if="column.key === 'actions'">
            <Space>
              <Button size="small" type="link" @click="handle编辑(record)">编辑</Button>
              <Popconfirm
                title="Are you sure to delete this binding?"
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
        <FormItem label="Policy ID">
          <Input v-model:value="formState.policyId" placeholder="Policy ID" />
        </FormItem>
        <FormItem label="Server Name">
          <Input v-model:value="formState.serverName" placeholder="e.g. example.com" />
        </FormItem>
        <FormItem label="Path">
          <Input v-model:value="formState.path" placeholder="e.g. /api/*" />
        </FormItem>
      </Form>
    </Modal>
  </Page>
</template>
