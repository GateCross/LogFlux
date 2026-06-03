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
  Switch,
  Table,
  Tag,
} from 'ant-design-vue';

import {
  createWafPolicyApi,
  deleteWafPolicyApi,
  getWafPolicyListApi,
  publishWafPolicyApi,
  updateWafPolicyApi,
} from '#/api/caddy/policy';

defineOptions({ name: 'SecurityPolicy' });

const loading = ref(false);
const dataList = ref<any[]>([]);
const modalVisible = ref(false);
const modalTitle = ref('创建 Policy');
const editingId = ref<number | null>(null);

const formState = reactive<Record<string, any>>({
  name: '',
  description: '',
  mode: 'block',
  enabled: true,
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '名称', dataIndex: 'name', key: 'name', width: 200 },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: 'Mode', dataIndex: 'mode', key: 'mode', width: 120 },
  { title: 'Version', dataIndex: 'version', key: 'version', width: 100 },
  { title: '状态', dataIndex: 'enabled', key: 'enabled', width: 100 },
  { title: '操作', key: 'actions', width: 300, fixed: 'right' as const },
];

async function fetchData() {
  loading.value = true;
  try {
    const res = await getWafPolicyListApi();
    dataList.value = Array.isArray(res) ? res : (res?.data ?? []);
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false;
  }
}

function handleCreate() {
  editingId.value = null;
  modalTitle.value = '创建 Policy';
  Object.assign(formState, { name: '', description: '', mode: 'block', enabled: true });
  modalVisible.value = true;
}

function handle编辑(record: any) {
  editingId.value = record.id;
  modalTitle.value = '编辑 Policy';
  Object.assign(formState, {
    name: record.name ?? '',
    description: record.description ?? '',
    mode: record.mode ?? 'block',
    enabled: record.enabled ?? true,
  });
  modalVisible.value = true;
}

async function handleSubmit() {
  try {
    if (editingId.value !== null) {
      await updateWafPolicyApi(editingId.value, { ...formState });
      message.success('Updated 成功');
    } else {
      await createWafPolicyApi({ ...formState });
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
    await deleteWafPolicyApi(id);
    message.success('删除d 成功');
    await fetchData();
  } catch {
    // error handled by interceptor
  }
}

async function handlePublish(id: number) {
  try {
    await publishWafPolicyApi(id);
    message.success('Published 成功');
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
  <Page title="WAF Policies" description="Manage WAF security policies. Create, edit, and publish policies.">
    <Card>
      <template #extra>
        <Space>
          <Button type="primary" @click="handleCreate">创建 Policy</Button>
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
          <template v-if="column.key === 'mode'">
            <Tag :color="record.mode === 'block' ? 'red' : 'orange'">
              {{ record.mode }}
            </Tag>
          </template>
          <template v-if="column.key === 'actions'">
            <Space>
              <Button size="small" type="link" @click="handle编辑(record)">编辑</Button>
              <Popconfirm
                title="Publish this policy version?"
                @confirm="handlePublish(record.id)"
              >
                <Button size="small" type="link">Publish</Button>
              </Popconfirm>
              <Popconfirm
                title="Are you sure to delete this policy?"
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
          <Input v-model:value="formState.name" placeholder="Policy name" />
        </FormItem>
        <FormItem label="Description">
          <Input v-model:value="formState.description" placeholder="Policy description" />
        </FormItem>
        <FormItem label="Mode">
          <Input v-model:value="formState.mode" placeholder="block or detect" />
        </FormItem>
      </Form>
    </Modal>
  </Page>
</template>
