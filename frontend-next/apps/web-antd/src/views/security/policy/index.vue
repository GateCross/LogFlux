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
const modalTitle = ref('Create Policy');
const editingId = ref<number | null>(null);

const formState = reactive<Record<string, any>>({
  name: '',
  description: '',
  mode: 'block',
  enabled: true,
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: 'Name', dataIndex: 'name', key: 'name', width: 200 },
  { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: 'Mode', dataIndex: 'mode', key: 'mode', width: 120 },
  { title: 'Version', dataIndex: 'version', key: 'version', width: 100 },
  { title: 'Status', dataIndex: 'enabled', key: 'enabled', width: 100 },
  { title: 'Actions', key: 'actions', width: 300, fixed: 'right' as const },
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
  modalTitle.value = 'Create Policy';
  Object.assign(formState, { name: '', description: '', mode: 'block', enabled: true });
  modalVisible.value = true;
}

function handleEdit(record: any) {
  editingId.value = record.id;
  modalTitle.value = 'Edit Policy';
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
      message.success('Updated successfully');
    } else {
      await createWafPolicyApi({ ...formState });
      message.success('Created successfully');
    }
    modalVisible.value = false;
    await fetchData();
  } catch {
    // error handled by interceptor
  }
}

async function handleDelete(id: number) {
  try {
    await deleteWafPolicyApi(id);
    message.success('Deleted successfully');
    await fetchData();
  } catch {
    // error handled by interceptor
  }
}

async function handlePublish(id: number) {
  try {
    await publishWafPolicyApi(id);
    message.success('Published successfully');
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
          <Button type="primary" @click="handleCreate">Create Policy</Button>
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
              <Button size="small" type="link" @click="handleEdit(record)">Edit</Button>
              <Popconfirm
                title="Publish this policy version?"
                @confirm="handlePublish(record.id)"
              >
                <Button size="small" type="link">Publish</Button>
              </Popconfirm>
              <Popconfirm
                title="Are you sure to delete this policy?"
                @confirm="handleDelete(record.id)"
              >
                <Button size="small" type="link" danger>Delete</Button>
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
