<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import { computed, ref } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import { Page } from '@vben/common-ui';
import { Alert, Button, Card, message } from 'antdv-next';

import { deleteNotificationChannelApi, getNotificationChannelsApi } from '#/api/notification';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

import ChannelFormModal from './components/ChannelFormModal.vue';
import ChannelList from './components/ChannelList.vue';
import ChannelTestModal from './components/ChannelTestModal.vue';

defineOptions({ name: 'NotificationChannel' });

const queryClient = useQueryClient();

const formOpen = ref(false);
const editingChannel = ref<NotificationApi.Channel | null>(null);
const testOpen = ref(false);
const testingChannel = ref<NotificationApi.Channel | null>(null);
const testing = ref(false);
const testTargetId = ref('');

const {
  data: channelsData,
  loading,
  errorMessage,
  refetch,
} = useListDetailQuery({
  queryKey: qk.notification.channels(),
  queryFn: () => getNotificationChannelsApi(withListDetailErrorMode()),
  errorFallback: '加载通知渠道失败',
});

const channels = computed(() => channelsData.value ?? []);

function openCreate() {
  editingChannel.value = null;
  formOpen.value = true;
}

function openEdit(record: NotificationApi.Channel) {
  editingChannel.value = record;
  formOpen.value = true;
}

function openTest(record: NotificationApi.Channel) {
  testingChannel.value = record;
  testTargetId.value = String(record.id);
  testOpen.value = true;
}

async function handleDelete(id: string) {
  try {
    await deleteNotificationChannelApi(id);
    message.success('删除成功');
    await invalidateListDetailQueries(queryClient, qk.notification.channels());
  } catch {
    message.error('删除失败');
  }
}

async function handleFormSuccess() {
  await invalidateListDetailQueries(queryClient, qk.notification.channels());
  await refetch();
}

function onTestingChange(value: boolean) {
  testing.value = value;
  if (!value) {
    testTargetId.value = '';
  }
}
</script>

<template>
  <Page title="通知渠道" description="配置告警与消息推送渠道">
    <Alert
      v-if="errorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="errorMessage"
    />

    <Card variant="borderless">
      <div class="mb-4">
        <Button type="primary" @click="openCreate">新增渠道</Button>
      </div>

      <ChannelList
        :channels="channels"
        :loading="loading"
        :testing="testing"
        :test-target-id="testTargetId"
        @test="openTest"
        @edit="openEdit"
        @delete="handleDelete"
      />
    </Card>

    <ChannelFormModal
      :open="formOpen"
      :channel="editingChannel"
      @update:open="formOpen = $event"
      @success="handleFormSuccess"
    />

    <ChannelTestModal
      :open="testOpen"
      :channel="testingChannel"
      @update:open="testOpen = $event"
      @update:testing="onTestingChange"
    />
  </Page>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
