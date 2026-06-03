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
} from 'ant-design-vue';

import {
  createWafPolicyExclusionApi,
  deleteWafPolicyExclusionApi,
  getWafPolicyExclusionListApi,
  updateWafPolicyExclusionApi,
} from '#/api/caddy/policy';

defineOptions({ name: 'SecurityExclusion' });

const loading = ref(false);
const dataList = ref<any[]>([]);
const modalVisible = ref(false);
const modalTitle = ref('Create Exclusion');
const editingId = ref<number | null>(null);

const formState = reactive<Record<string, any>>({
  ruleId: '',
  path: '',
  description: '',
  exclusionType: 'path',
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: 'Rule ID', dataIndex: 'ruleId', key: 'ruleId', width: 160 },
  { title: 'Path', dataIndex: 'path', key: 'path', ellipsis: true },
  { title: 'Type', dataIndex: 'exclusionType', key: 'exclusionType', width: 120 },
  { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: 'Actions', key: 'actions', width: 160, fixed: 'right' as const },
];

async function fetchData() {
  loading.value = true;
  try {
    const res = await getWafPolicyExclusionListApi();
    dataList.value = Array.isArray(res) ? res : (res?.data ?? []);
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false;
  }
}

function handleCreate() {
  editingId.value = null;
  modalTitle.value = 'Create Exclusion';
  Object.assign(formState, { ruleId: '', path: '', description: '', exclusionType: 'path' });
  modalVisible.value = true;
}

function handleEdit(record: any) {
  editingId.value = record.id;
  modalTitle.value = 'Edit Exclusion';
  Object.assign(formState, {
    ruleId: record.ruleId ?? '',
    path: record.path ?? '',
    description: record.description ?? '',
    exclusionType: record.exclusionType ?? 'path',
  });
  modalVisible.value = true;
}

async function handleSubmit() {
  try {
    if (editingId.value !== null) {
      await updateWafPolicyExclusionApi(editingId.value, { ...formState });
      message.success('Updated successfully');
    } else {
      await createWafPolicyExclusionApi({ ...formState });
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
    await deleteWafPolicyExclusionApi(id);
    message.success('Deleted successfully');
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
  <Page title="Rule Exclusions" description="Manage WAF rule exclusions. Define paths and rules to exclude from WAF inspection.">
    <Card>
      <template #extra>
        <Space>
          <Button type="primary" @click="handleCreate">Add Exclusion</Button>
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
          <template v-if="column.key === 'actions'">
            <Space>
              <Button size="small" type="link" @click="handleEdit(record)">Edit</Button>
              <Popconfirm
                title="Are you sure to delete this exclusion?"
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
        <FormItem label="Rule ID">
          <Input v-model:value="formState.ruleId" placeholder="e.g. 942100" />
        </FormItem>
        <FormItem label="Path">
          <Input v-model:value="formState.path" placeholder="e.g. /api/upload" />
        </FormItem>
        <FormItem label="Type">
          <Input v-model:value="formState.exclusionType" placeholder="path, header, param" />
        </FormItem>
        <FormItem label="Description">
          <Input v-model:value="formState.description" placeholder="Description" />
        </FormItem>
      </Form>
    </Modal>
  </Page>
</template>
