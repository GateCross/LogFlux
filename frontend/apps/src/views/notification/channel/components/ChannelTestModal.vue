<script lang="ts" setup>
import { reactive, ref, watch } from 'vue';

import {
  Form,
  FormItem,
  Input,
  Modal,
  message,
} from 'antdv-next';

import type { NotificationApi } from '#/api/notification';
import { testNotificationChannelApi } from '#/api/notification';

const props = defineProps<{
  open: boolean;
  channel: NotificationApi.Channel | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'update:testing': [value: boolean];
}>();

const testing = ref(false);
const testFormState = reactive({
  content: '',
  title: '测试通知',
});

watch(() => props.open,
  (open) => {
    if (!open || !props.channel) return;
    testFormState.title = '测试通知';
    testFormState.content = `这是一条发送到「${props.channel.name}」的测试通知。`;
  },
);

async function handleConfirmTest() {
  if (!props.channel) return;
  if (!testFormState.title.trim() || !testFormState.content.trim()) {
    message.warning('请填写测试标题和内容');
    return;
  }

  testing.value = true;
  emit('update:testing', true);
  try {
    await testNotificationChannelApi({
      channelId: String(props.channel.id),
      content: testFormState.content,
      title: testFormState.title,
    });
    message.success('测试通知已发送');
    emit('update:open', false);
  } catch {
    message.error('测试通知发送失败');
  } finally {
    testing.value = false;
    emit('update:testing', false);
  }
}

function handleOpenChange(value: boolean) {
  emit('update:open', value);
}
</script>

<template>
  <Modal
    :open="open"
    :confirm-loading="testing"
    title="测试通知渠道"
    :width="560"
    @update:open="handleOpenChange"
    @ok="handleConfirmTest"
  >
    <Form
      :label-col="{ span: 5 }"
      :wrapper-col="{ span: 19 }"
      class="mt-4"
    >
      <FormItem label="渠道">
        <Input :value="channel?.name ?? ''" readonly />
      </FormItem>
      <FormItem label="标题" required>
        <Input v-model:value="testFormState.title" placeholder="测试标题" />
      </FormItem>
      <FormItem label="内容" required>
        <Input.TextArea
          v-model:value="testFormState.content"
          placeholder="测试内容"
          :rows="5"
        />
      </FormItem>
    </Form>
  </Modal>
</template>
