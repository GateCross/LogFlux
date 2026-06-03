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
  { dataIndex: 'level', key: 'level', title: 'Event' },
  { dataIndex: 'channelId', key: 'channelId', title: '渠道' },
  { dataIndex: 'enabled', key: 'enabled', title: '启用' },
  { key: 'actions', title: '操作', width: 180 },
];

function channelName(id: string) {
  return channelOptions.value.find((c) => c.value === id)?.label ?? id;
}

function templateName(id?: string) {
  if (!id) return '-';
  return templateOptions.value.find((t) => t.value === id)?.label ?? id;
}

// ── 创建 / 编辑 modal ─────────────────────────────────────

const modalVisible = ref(false);
const modalLoading = ref(false);
const editingId = ref<string | null>(null);
const is编辑ing = computed(() => editingId.value !== null);

const levelOptions = [
  { label: 'Critical', value: 'critical' },
  { label: 'Error', value: 'error' },
  { label: 'Warning', value: 'warning' },
  { label: 'Info', value: 'info' },
  { label: 'Debug', value: 'debug' },
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

function open编辑(record: NotificationApi.Rule) {
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
    message.warning('Please select a channel');
    return;
  }
  modalLoading.value = true;
  try {
    if (is编辑ing.value) {
      await updateNotificationRuleApi(editingId.value!, { ...formState });
      message.success('Rule updated 成功');
    } else {
      await createNotificationRuleApi({ ...formState });
      message.success('Rule created 成功');
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

async function handle删除(id: string) {
  try {
    await deleteNotificationRuleApi(id);
    message.success('Rule deleted');
    await fetchRules();
  } catch {
    message.error('删除 failed');
  }
}

// ── Init ────────────────────────────────────────────────────

onMounted(async () => {
  await Promise.all([fetchRules(), fetchReferenceData()]);
});
</script>

<template>
  <div class="p-5">
    <Card title="Notification Rules">
      <template #extra>
        <Button type="primary" @click="openCreate">
          新增 Rule
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
              {{ record.level }}
            </Tag>
          </template>

          <template v-if="column.key === 'channelId'">
            {{ channelName(record.channelId) }}
          </template>

          <template v-if="column.key === 'enabled'">
            <Tag :color="record.enabled ? 'green' : 'default'">
              {{ record.enabled ? 'Enabled' : 'Disabled' }}
            </Tag>
          </template>

          <template v-if="column.key === 'actions'">
            <Space>
              <Button
                size="small"
                type="link"
                @click="open编辑(record)"
              >
                编辑
              </Button>
              <Popconfirm
                title="Are you sure to delete this rule?"
                @confirm="handle删除(record.id)"
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
      :title="is编辑ing ? '编辑 Rule' : '新增 Rule'"
      :width="560"
      @ok="handleSubmit"
    >
      <Form
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
        class="mt-4"
      >
        <FormItem label="Name" required>
          <Input v-model:value="formState.name" placeholder="Rule name" />
        </FormItem>
        <FormItem label="Event Level" required>
          <Select
            v-model:value="formState.level"
            :options="levelOptions"
            placeholder="Select event level"
          />
        </FormItem>
        <FormItem label="Channel" required>
          <Select
            v-model:value="formState.channelId"
            :options="channelOptions"
            placeholder="Select notification channel"
            show-search
            :filter-option="(input: string, option: any) => option.label.toLowerCase().includes(input.toLowerCase())"
          />
        </FormItem>
        <FormItem label="Template">
          <Select
            v-model:value="formState.templateId"
            :options="templateOptions"
            placeholder="Select template (optional)"
            allow-clear
            show-search
            :filter-option="(input: string, option: any) => option.label.toLowerCase().includes(input.toLowerCase())"
          />
        </FormItem>
        <FormItem label="Enabled">
          <Switch v-model:checked="formState.enabled" />
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>
