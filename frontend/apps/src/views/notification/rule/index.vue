<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import { computed, reactive, ref } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import {
  Alert,
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
} from 'antdv-next';

import {
  createNotificationRuleApi,
  deleteNotificationRuleApi,
  getNotificationChannelsApi,
  getNotificationEventsApi,
  getNotificationRulesApi,
  getNotificationTemplatesApi,
  updateNotificationRuleApi,
} from '#/api/notification';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

defineOptions({ name: 'NotificationRule' });

const queryClient = useQueryClient();

const {
  data: rulesData,
  loading,
  errorMessage,
  refetch: refetchRules,
} = useListDetailQuery({
  queryKey: qk.notification.rules(),
  queryFn: () => getNotificationRulesApi(withListDetailErrorMode()),
  errorFallback: '加载通知规则失败',
});

const rules = computed(() => rulesData.value ?? []);

const { data: channelsData } = useListDetailQuery({
  queryKey: qk.notification.channels({ for: 'rule-ref' }),
  queryFn: () => getNotificationChannelsApi(withListDetailErrorMode()),
  errorFallback: '加载渠道失败',
});

const { data: templatesData } = useListDetailQuery({
  queryKey: qk.notification.templates({ for: 'rule-ref' }),
  queryFn: () => getNotificationTemplatesApi(withListDetailErrorMode()),
  errorFallback: '加载模板失败',
});

const { data: eventsData } = useListDetailQuery({
  queryKey: qk.notification.events(),
  queryFn: () => getNotificationEventsApi(withListDetailErrorMode()),
  errorFallback: '加载事件失败',
});

const channelOptions = computed(() =>
  (channelsData.value ?? []).map((c) => ({ label: c.name, value: String(c.id) })),
);
const templateOptions = computed(() =>
  (templatesData.value ?? []).map((t) => ({ label: t.name, value: t.name })),
);

const levelOptions = computed(() => {
  const options = [{ label: '不限', value: '*' }];
  for (const e of eventsData.value ?? []) {
    if (e.group === '事件级别') {
      options.push({ label: e.label, value: e.value });
    }
  }
  return options;
});

const eventTypeOptions = computed(() => {
  const groups = new Map<string, { label: string; value: string }[]>();
  for (const e of eventsData.value ?? []) {
    if (e.group === '事件级别') continue;
    const opts = groups.get(e.group) || [];
    opts.push({ label: e.label, value: e.value });
    groups.set(e.group, opts);
  }
  return Array.from(groups.entries()).map(([label, options]) => ({
    label,
    options,
  }));
});

const flatEventTypeOptions = computed(() => {
  const list: { label: string; value: string }[] = [];
  for (const e of eventsData.value ?? []) {
    if (e.group !== '事件级别') {
      list.push({ label: e.label, value: e.value });
    }
  }
  return list;
});

const columns = [
  { dataIndex: 'name', key: 'name', title: '名称' },
  { dataIndex: 'eventLevel', key: 'eventLevel', title: '事件等级' },
  { dataIndex: 'eventTypes', key: 'eventTypes', title: '特定事件' },
  { dataIndex: 'channelId', key: 'channelId', title: '渠道' },
  { dataIndex: 'enabled', key: 'enabled', title: '启用' },
  { key: 'actions', title: '操作', width: 200 },
];

function channelName(id: string) {
  return channelOptions.value.find((c) => c.value === id)?.label ?? id;
}

const modalVisible = ref(false);
const modalLoading = ref(false);
const editingId = ref<string | null>(null);
const isEditing = computed(() => editingId.value !== null);

const formState = reactive<NotificationApi.RuleParams>({
  channelId: '',
  conditions: {},
  enabled: true,
  eventLevel: '*',
  eventTypes: [],
  name: '',
  templateId: undefined,
});

function openCreate() {
  editingId.value = null;
  Object.assign(formState, {
    channelId: '',
    conditions: {},
    enabled: true,
    eventLevel: '*',
    eventTypes: [],
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
    eventLevel: record.eventLevel,
    eventTypes: record.eventTypes,
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
    await invalidateListDetailQueries(queryClient, qk.notification.rules());
    await refetchRules();
  } catch {
    message.error('操作失败');
  } finally {
    modalLoading.value = false;
  }
}

async function handleDelete(id: string) {
  try {
    await deleteNotificationRuleApi(id);
    message.success('通知规则已删除');
    await invalidateListDetailQueries(queryClient, qk.notification.rules());
    await refetchRules();
  } catch {
    message.error('删除失败');
  }
}

function levelLabel(level: string) {
  if (level === '*') return '不限';
  return levelOptions.value.find((item) => item.value === level)?.label ?? level;
}

function typeLabel(type: string) {
  return flatEventTypeOptions.value.find((item) => item.value === type)?.label ?? type;
}
</script>

<template>
  <div class="p-5">
    <Alert
      v-if="errorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="errorMessage"
    />
    <Card title="通知规则">
      <template #extra>
        <Button type="primary" @click="openCreate">新增规则</Button>
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
          <template v-if="column.key === 'eventLevel'">
            <Tag
              :color="
                record.eventLevel === 'critical'
                  ? 'red'
                  : record.eventLevel === 'error'
                    ? 'volcano'
                    : record.eventLevel === 'warning'
                      ? 'orange'
                      : record.eventLevel === 'info'
                        ? 'blue'
                        : 'default'
              "
            >
              {{ levelLabel(record.eventLevel) }}
            </Tag>
          </template>

          <template v-if="column.key === 'eventTypes'">
            <template v-if="record.eventTypes && record.eventTypes.length > 0">
              <Space wrap>
                <Tag v-for="type in record.eventTypes" :key="type" color="purple">
                  {{ typeLabel(type) }}
                </Tag>
              </Space>
            </template>
            <span v-else class="text-gray-400">不限</span>
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
            <Space :size="6">
              <Button
                size="small"
                class="table-action-btn"
                @click="openEdit(record as NotificationApi.Rule)"
              >
                编辑
              </Button>
              <Popconfirm title="确认删除该通知规则？" @confirm="handleDelete(record.id)">
                <Button size="small" danger class="table-action-btn table-action-btn--danger">
                  删除
                </Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <Modal
      v-model:open="modalVisible"
      :confirm-loading="modalLoading"
      :title="isEditing ? '编辑规则' : '新增规则'"
      :width="560"
      @ok="handleSubmit"
    >
      <Form :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }" class="mt-4">
        <FormItem label="名称" required>
          <Input v-model:value="formState.name" placeholder="规则名称" />
        </FormItem>
        <FormItem label="事件等级" required>
          <Select
            v-model:value="formState.eventLevel"
            :options="levelOptions"
            placeholder="选择拦截的事件等级"
          />
        </FormItem>
        <FormItem label="特定事件">
          <Select
            v-model:value="formState.eventTypes"
            mode="multiple"
            :options="eventTypeOptions"
            placeholder="选择特定事件（可多选），留空表示不限"
            show-search
            :filter-option="
              (input, option) =>
                (option?.label as string)?.toLowerCase().includes(input.toLowerCase()) ?? false
            "
          />
        </FormItem>
        <FormItem label="渠道" required>
          <Select
            v-model:value="formState.channelId"
            :options="channelOptions"
            placeholder="选择通知渠道"
            show-search
            :filter-option="
              (input: string, option: any) =>
                option.label.toLowerCase().includes(input.toLowerCase())
            "
          />
        </FormItem>
        <FormItem label="模板">
          <Select
            v-model:value="formState.templateId"
            :options="templateOptions"
            placeholder="选择模板（可选）"
            allow-clear
            show-search
            :filter-option="
              (input: string, option: any) =>
                option.label.toLowerCase().includes(input.toLowerCase())
            "
          />
        </FormItem>
        <FormItem label="启用">
          <Switch v-model:checked="formState.enabled" />
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
