<script lang="ts" setup>
import { Alert, Input, Modal, Space, Tag } from 'antdv-next';

import { formatCaddyfile } from '../caddy-config-utils';
import type { SavePreviewKind } from '../composables/useCaddyConfigIO';

const props = defineProps<{
  open: boolean;
  saving: boolean;
  kind: SavePreviewKind;
  actions: string[];
  errors: string[];
  config: string;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  confirm: [];
}>();
</script>

<template>
  <Modal
    :open="props.open"
    width="900px"
    title="保存预览"
    :confirm-loading="props.saving"
    @update:open="emit('update:open', $event)"
    @ok="emit('confirm')"
  >
    <Space wrap class="mb-3">
      <Tag color="blue">{{ props.kind === 'raw' ? '原始配置' : '分块配置' }}</Tag>
      <Tag v-for="item in props.actions" :key="item">{{ item }}</Tag>
    </Space>
    <Alert
      v-if="props.errors.length"
      class="mb-3"
      type="error"
      :message="props.errors[0]"
      show-icon
    />
    <Input.TextArea
      :value="formatCaddyfile(props.config)"
      :auto-size="{ minRows: 18, maxRows: 28 }"
      class="code-textarea"
      readonly
    />
  </Modal>
</template>

<style scoped>
.code-textarea {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 13px;
}

.mb-3 {
  margin-bottom: 12px;
}
</style>
