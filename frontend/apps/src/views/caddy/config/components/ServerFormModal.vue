<script lang="ts" setup>
import { Form, FormItem, Input, InputPassword, Modal, RadioButton, RadioGroup } from 'antdv-next';

import type { ServerFormState } from '../composables/useCaddyServers';

const props = defineProps<{
  open: boolean;
  type: 'add' | 'edit';
  form: ServerFormState;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  save: [];
}>();

function handleOpenChange(value: boolean) {
  emit('update:open', value);
}

function handleOk() {
  emit('save');
}
</script>

<template>
  <Modal
    :open="props.open"
    :title="props.type === 'add' ? '添加服务器' : '编辑服务器'"
    @update:open="handleOpenChange"
    @ok="handleOk"
  >
    <Form layout="vertical">
      <FormItem label="名称">
        <Input v-model:value="props.form.name" placeholder="服务器名称" />
      </FormItem>
      <FormItem label="地址">
        <Input v-model:value="props.form.url" placeholder="http://localhost:2019" />
      </FormItem>
      <FormItem label="类型">
        <RadioGroup v-model:value="props.form.type">
          <RadioButton value="local">本地</RadioButton>
          <RadioButton value="remote">远程</RadioButton>
        </RadioGroup>
      </FormItem>
      <FormItem v-if="props.form.type === 'remote'" label="凭证">
        <InputPassword v-model:value="props.form.token" placeholder="可选认证凭证" />
      </FormItem>
    </Form>
  </Modal>
</template>
