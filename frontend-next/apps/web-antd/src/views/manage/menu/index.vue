<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue';

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
  Select,
  SelectOption,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';

import { requestClient } from '#/api/request';

defineOptions({ name: 'ManageMenu' });

// --------------- types ---------------
interface MenuItem {
  id: number;
  title: string;
  path: string;
  icon: string;
  order: number;
  type: 'directory' | 'menu';
  status: string;
  parentId: number | null;
  children?: MenuItem[];
}

// --------------- placeholder API calls ---------------
async function getMenuListApi() {
  return requestClient.get<MenuItem[]>('/system/menu/list');
}

async function createMenuApi(data: Omit<MenuItem, 'children' | 'id'>) {
  return requestClient.post('/system/menu', data);
}

async function updateMenuApi(
  id: number,
  data: Partial<Omit<MenuItem, 'children' | 'id'>>,
) {
  return requestClient.put(`/system/menu/${id}`, data);
}

async function deleteMenuApi(id: number) {
  return requestClient.delete(`/system/menu/${id}`);
}

// --------------- state ---------------
const loading = ref(false);
const dataSource = ref<MenuItem[]>([]);
const flatList = ref<MenuItem[]>([]);
const modalVisible = ref(false);
const modalTitle = ref('Add Menu');
const editingId = ref<number | null>(null);
const submitting = ref(false);

const formState = reactive({
  icon: '',
  order: 0,
  parentId: null as number | null,
  path: '',
  title: '',
  type: 'menu' as 'directory' | 'menu',
});

// --------------- columns ---------------
const columns = [
  { dataIndex: 'title', key: 'title', title: 'Title', width: 200 },
  { dataIndex: 'path', key: 'path', title: 'Path', width: 200 },
  { dataIndex: 'icon', key: 'icon', title: 'Icon', width: 120 },
  { dataIndex: 'order', key: 'order', title: 'Order', width: 80 },
  { dataIndex: 'type', key: 'type', title: 'Type', width: 100 },
  { dataIndex: 'status', key: 'status', title: 'Status', width: 100 },
  { key: 'actions', title: 'Actions', width: 180 },
];

// --------------- helpers ---------------
/** Flatten nested tree to a flat list for parent select options */
function flattenTree(nodes: MenuItem[], result: MenuItem[] = []): MenuItem[] {
  for (const node of nodes) {
    result.push(node);
    if (node.children?.length) {
      flattenTree(node.children, result);
    }
  }
  return result;
}

/** Parent options excluding the item being edited */
const parentOptions = computed(() => {
  return flatList.value.filter(
    (item) => item.id !== editingId.value && item.type === 'directory',
  );
});

// --------------- data fetching ---------------
async function fetchData() {
  loading.value = true;
  try {
    dataSource.value = await getMenuListApi();
    flatList.value = flattenTree([...dataSource.value]);
  } catch {
    message.error('Failed to load menu list');
  } finally {
    loading.value = false;
  }
}

// --------------- modal actions ---------------
function openAddModal(parentId?: number) {
  modalTitle.value = 'Add Menu';
  editingId.value = null;
  formState.title = '';
  formState.path = '';
  formState.icon = '';
  formState.order = 0;
  formState.parentId = parentId ?? null;
  formState.type = 'menu';
  modalVisible.value = true;
}

function openEditModal(record: MenuItem) {
  modalTitle.value = 'Edit Menu';
  editingId.value = record.id;
  formState.title = record.title;
  formState.path = record.path;
  formState.icon = record.icon;
  formState.order = record.order;
  formState.parentId = record.parentId;
  formState.type = record.type;
  modalVisible.value = true;
}

async function handleSubmit() {
  if (!formState.title) {
    message.warning('Title is required');
    return;
  }
  if (!formState.path) {
    message.warning('Path is required');
    return;
  }

  submitting.value = true;
  try {
    if (editingId.value) {
      await updateMenuApi(editingId.value, { ...formState });
      message.success('Menu updated successfully');
    } else {
      await createMenuApi({ ...formState });
      message.success('Menu created successfully');
    }
    modalVisible.value = false;
    await fetchData();
  } catch {
    message.error('Operation failed');
  } finally {
    submitting.value = false;
  }
}

async function handleDelete(id: number) {
  try {
    await deleteMenuApi(id);
    message.success('Menu deleted successfully');
    await fetchData();
  } catch {
    message.error('Failed to delete menu');
  }
}
</script>

<template>
  <div class="p-5">
    <Card title="Menu Management">
      <template #extra>
        <Button type="primary" @click="openAddModal()">Add Menu</Button>
      </template>

      <Table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        :pagination="false"
        :default-expand-all-rows="true"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <Tag :color="record.type === 'directory' ? 'orange' : 'blue'">
              {{ record.type }}
            </Tag>
          </template>
          <template v-if="column.key === 'status'">
            <Tag :color="record.status === 'active' ? 'green' : 'red'">
              {{ record.status }}
            </Tag>
          </template>
          <template v-if="column.key === 'actions'">
            <Space>
              <Button
                v-if="record.type === 'directory'"
                type="link"
                size="small"
                @click="openAddModal(record.id)"
              >
                Add Child
              </Button>
              <Button type="link" size="small" @click="openEditModal(record)">
                Edit
              </Button>
              <Popconfirm
                title="Are you sure to delete this menu? Child items will also be removed."
                ok-text="Yes"
                cancel-text="No"
                @confirm="handleDelete(record.id)"
              >
                <Button type="link" size="small" danger>Delete</Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <Modal
      :open="modalVisible"
      :title="modalTitle"
      :confirm-loading="submitting"
      @cancel="modalVisible = false"
      @ok="handleSubmit"
    >
      <Form layout="vertical" style="margin-top: 16px;">
        <FormItem label="Title" required>
          <Input
            v-model:value="formState.title"
            placeholder="Enter menu title"
          />
        </FormItem>
        <FormItem label="Path" required>
          <Input
            v-model:value="formState.path"
            placeholder="e.g. /system/user"
          />
        </FormItem>
        <FormItem label="Icon">
          <Input
            v-model:value="formState.icon"
            placeholder="Icon name, e.g. UserOutlined"
          />
        </FormItem>
        <FormItem label="Parent">
          <Select
            v-model:value="formState.parentId"
            allow-clear
            placeholder="Select parent (none = root)"
          >
            <SelectOption :value="null">-- Root --</SelectOption>
            <SelectOption
              v-for="item in parentOptions"
              :key="item.id"
              :value="item.id"
            >
              {{ item.title }}
            </SelectOption>
          </Select>
        </FormItem>
        <FormItem label="Order">
          <InputNumber
            v-model:value="formState.order"
            :min="0"
            style="width: 100%;"
          />
        </FormItem>
        <FormItem label="Type">
          <Select v-model:value="formState.type">
            <SelectOption value="directory">Directory</SelectOption>
            <SelectOption value="menu">Menu</SelectOption>
          </Select>
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>
