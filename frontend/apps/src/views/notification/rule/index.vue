<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import { computed, onMounted, reactive, ref } from 'vue';

import {
  Button,
  Card,
  Form,
  FormItem,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  message,
} from 'ant-design-vue';

import {
  createNotificationRuleApi,
  deleteNotificationRuleApi,
  getNotificationChannelsApi,
  getNotificationRulesApi,
  getNotificationTemplatesApi,
  updateNotificationRuleApi,
} from '#/api/notification';

defineOptions({ name: 'NotificationRule' });

// ── List state ──────────────────────────────────────────────

const loading = ref(false);
const rules = ref<NotificationApi.Rule[]>([]);

async function fetchRules() {
  loading.value = true;
  try {
    rules.value = await getNotificationRulesApi();
  } finally {
    loading.value = false;
  }
}

// ── Reference data (channels & templates for selects) ───────

const channelOptions = ref<{ label: string; value: string }[]>([]);
const templateOptions = ref<{ label: string; value: string }[]>([]);

async function fetchReferenceData() {
  try {
    const [channels, templates] = await Promise.all([
      getNotificationChannelsApi(),
      getNotificationTemplatesApi(),
    ]);
    channelOptions.value = channels.map((c) => ({ label: c.name, value: c.id }));
    templateOptions.value = templates.map((t) => ({ label: t.name, value: t.id }));
  } catch {
    // Non-critical; selects will just be empty
  }
}

// ── Table columns ───────────────────────────────────────────

const columns = [
  { dataIndex: 'name', key: 'name', title: '名称' },
  { dataIndex: 'level', key: 'level', title: '事件级别' },
  { dataIndex: 'channelId', key: 'channelId', title: '渠道' },
  { dataIndex: 'enabled', key: 'enabled', title: '启用' },
  { key: 'actions', title: '操作', width: 180 },
];

function channelName(id: string) {
  return channelOptions.value.find((c) => c.value === id)?.label ?? id;
}

// ── 创建 / 编辑 modal ─────────────────────────────────────

const modalVisible = ref(false);
const modalLoading = ref(false);
const editingId = ref<string | null>(null);
const isEditing = computed(() => editingId.value !== null);

const levelOptions = [
  { label: '严重', value: 'critical' },
  { label: '错误', value: 'error' },
  { label: '警告', value: 'warning' },
  { label: '信息', value: 'info' },
  { label: '调试', value: 'debug' },
];

const formState = reactive<NotificationApi.RuleParams>({
  channelId: '',
  conditions: {},
  enabled: true,
  level: 'error',
  name: '',
  templateId: undefined,
});

function openCreate() {
  editingId.value = null;
  Object.assign(formState, {
    channelId: '',
    conditions: {},
    enabled: true,
    level: 'error',
    name: '',
    templateId: undefined,
  });
  modalVisible.value = true;
}

function openEdit(record: NotificationApi.Rule) {
  editingId.value = record.id;
  Object.assign(formState, {
    channelId: record.channelId,
    conditions: record.conditions,
    enabled: record.enabled,
    level: record.level,
    name: record.name,
    templateId: record.templateId,
  });
  modalVisible.value = true;
}

async function handleSubmit() {
  if (!formState.channelId) {
    message.warning('请选择通知渠道');
    return;
  }
  modalLoading.value = true;
  try {
    if (isEditing.value) {
      await updateNotificationRuleApi(editingId.value!, { ...formState });
      message.success('通知规则已更新');
    } else {
      await createNotificationRuleApi({ ...formState });
      message.success('通知规则已创建');
    }
    modalVisible.value = false;
    await fetchRules();
  } catch {
    message.error('操作失败');
  } finally {
    modalLoading.value = false;
  }
}

// ── 删除 ──────────────────────────────────────────────────

async function handleDelete(id: string) {
  try {
    await deleteNotificationRuleApi(id);
    message.success('通知规则已删除');
    await fetchRules();
  } catch {
    message.error('删除失败');
  }
}

function levelLabel(level: string) {
  return levelOptions.find((item) => item.value === level)?.label ?? level;
}

// ── Init ────────────────────────────────────────────────────

onMounted(async () => {
  await Promise.all([fetchRules(), fetchReferenceData()]);
});
</script>

<template>
  <div class="p-5">
    <Card title="通知规则">
      <template #extra>
        <Button type="primary" @click="openCreate">
          新增规则
        </Button>
      </template>

      <Table
        :columns="columns"
        :data-source="rules"
        :loading="loading"
        row-key="id"
        size="middle"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'level'">
            <Tag
              :color="
                record.level === 'critical'
                  ? 'red'
                  : record.level === 'error'
                    ? 'volcano'
                    : record.level === 'warning'
                      ? 'orange'
                      : record.level === 'info'
                        ? 'blue'
                        : 'default'
              "
            >
              {{ levelLabel(record.level) }}
            </Tag>
          </template>

          <template v-if="column.key === 'channelId'">
            {{ channelName(record.channelId) }}
          </template>

          <template v-if="column.key === 'enabled'">
            <Tag :color="record.enabled ? 'green' : 'default'">
              {{ record.enabled ? '启用' : '停用' }}
            </Tag>
          </template>

          <template v-if="column.key === 'actions'">
            <Space>
              <Button
                size="small"
                type="link"
                @click="openEdit(record as NotificationApi.Rule)"
              >
                编辑
              </Button>
              <Popconfirm
                title="确认删除该通知规则？"
                @confirm="handleDelete(record.id)"
              >
                <Button size="small" type="link" danger>
                  删除
                </Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <!-- ── 创建 / 编辑 modal ──────────────────────────────── -->
    <Modal
      v-model:open="modalVisible"
      :confirm-loading="modalLoading"
      :title="isEditing ? '编辑规则' : '新增规则'"
      :width="560"
      @ok="handleSubmit"
    >
      <Form
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
        class="mt-4"
      >
        <FormItem label="名称" required>
          <Input v-model:value="formState.name" placeholder="规则名称" />
        </FormItem>
        <FormItem label="事件级别" required>
          <Select
            v-model:value="formState.level"
            :options="levelOptions"
            placeholder="选择事件级别"
          />
        </FormItem>
        <FormItem label="渠道" required>
          <Select
            v-model:value="formState.channelId"
            :options="channelOptions"
            placeholder="选择通知渠道"
            show-search
            :filter-option="(input: string, option: any) => option.label.toLowerCase().includes(input.toLowerCase())"
          />
        </FormItem>
        <FormItem label="模板">
          <Select
            v-model:value="formState.templateId"
            :options="templateOptions"
            placeholder="选择模板（可选）"
            allow-clear
            show-search
            :filter-option="(input: string, option: any) => option.label.toLowerCase().includes(input.toLowerCase())"
          />
        </FormItem>
        <FormItem label="启用">
          <Switch v-model:checked="formState.enabled" />
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>
