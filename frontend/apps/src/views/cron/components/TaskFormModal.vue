<script lang="ts" setup>
import type { CronApi } from '#/api/cron';

import { computed } from 'vue';

import {
  Button,
  Form,
  FormItem,
  Input,
  InputNumber,
  Modal,
  Radio,
  RadioGroup,
  Switch,
  Upload,
} from 'antdv-next';

import MonacoCodeEditor from '#/components/code-editor/MonacoCodeEditor.vue';

const props = defineProps<{
  open: boolean;
  mode: 'add' | 'edit';
  submitting: boolean;
  taskForm: CronApi.TaskPayload;
  editingTask: CronApi.Task | null;
  taskCanEnable: boolean;
  beforeTaskFileUpload: (file: File) => boolean | typeof Upload.LIST_IGNORE;
  removeTaskFile: () => void;
  syncFileModeStatus: () => void;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submit: [];
}>();

const taskModalTitle = computed(() =>
  props.mode === 'add' ? '新增任务' : '编辑任务',
);
</script>

<template>
  <Modal
    :confirm-loading="submitting"
    :open="open"
    :title="taskModalTitle"
    :width="720"
    @cancel="emit('update:open', false)"
    @ok="emit('submit')"
  >
    <Form :label-col="{ span: 5 }" :wrapper-col="{ span: 19 }" class="mt-4">
      <FormItem label="任务名称" required>
        <Input v-model:value="taskForm.name" />
      </FormItem>

      <FormItem label="Cron 表达式" required>
        <Input
          v-model:value="taskForm.schedule"
          placeholder="例如 0/5 * * * * ?"
        />
      </FormItem>

      <FormItem label="脚本来源">
        <RadioGroup v-model:value="taskForm.scriptMode" @change="syncFileModeStatus">
          <Radio value="inline">手写脚本</Radio>
          <Radio value="file">上传脚本</Radio>
        </RadioGroup>
      </FormItem>

      <FormItem v-if="taskForm.scriptMode === 'inline'" label="执行脚本" required>
        <MonacoCodeEditor
          :model-value="taskForm.script ?? ''"
          language="shell"
          :height="240"
          @update:model-value="(v) => (taskForm.script = v)"
        />
      </FormItem>

      <FormItem v-else label="脚本文件" :required="mode === 'add'">
        <Upload
          v-if="mode === 'add'"
          :before-upload="beforeTaskFileUpload"
          :max-count="1"
          accept=".sh"
          @remove="removeTaskFile"
        >
          <Button>选择脚本文件</Button>
        </Upload>
        <div v-else class="text-sm text-gray-500">
          如需上传新版本，请在脚本管理中操作。
          <template v-if="editingTask?.currentFileName">
            当前脚本：v{{ editingTask.currentFileVersion }} ·
            {{ editingTask.currentFileName }}
          </template>
        </div>
      </FormItem>

      <FormItem label="超时时间">
        <InputNumber
          v-model:value="taskForm.timeout"
          :min="1"
          :max="3600"
          class="w-full"
        />
      </FormItem>

      <FormItem label="启用">
        <Switch
          v-model:checked="taskForm.status"
          :checked-value="1"
          :unchecked-value="0"
          :disabled="!taskCanEnable"
        />
      </FormItem>
    </Form>
  </Modal>
</template>
