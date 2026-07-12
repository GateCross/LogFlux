<script lang="ts" setup>
import { Page } from '@vben/common-ui';
import { Alert, Button, Card, Input, Space } from 'antdv-next';

import ScriptManagerModal from './components/ScriptManagerModal.vue';
import TaskFormModal from './components/TaskFormModal.vue';
import TaskList from './components/TaskList.vue';
import TaskLogDrawer from './components/TaskLogDrawer.vue';
import { useCronPage } from './composables/useCronPage';

defineOptions({ name: 'CronManagement' });

const {
  loading,
  listErrorMessage,
  tasks,
  searchName,
  taskPagination,
  taskModalVisible,
  taskModalMode,
  editingTask,
  taskSubmitting,
  taskForm,
  scriptModalVisible,
  currentScriptTask,
  scriptHistory,
  scriptHistoryLoading,
  scriptPagination,
  scriptUploading,
  activatingScriptIds,
  logDrawerVisible,
  logs,
  logsLoading,
  logPagination,
  logDetailVisible,
  detailLoading,
  currentLog,
  triggeringTaskIds,
  deletingTaskIds,
  scriptModalTitle,
  logDrawerTitle,
  taskCanEnable,
  beforeTaskFileUpload,
  beforeScriptFileUpload,
  removeTaskFile,
  removeScriptFile,
  syncFileModeStatus,
  canTriggerTask,
  handleSearch,
  handleReset,
  openCreate,
  openEdit,
  submitTask,
  deleteTask,
  triggerTask,
  handleTaskTableChange,
  openLogs,
  fetchLogs,
  handleLogTableChange,
  openLogDetail,
  openScriptManager,
  handleScriptTableChange,
  uploadScript,
  activateScript,
} = useCronPage();
</script>

<template>
  <Page title="定时任务" description="管理系统定时任务与执行日志">
    <Alert
      v-if="listErrorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="listErrorMessage"
    />

    <Card variant="borderless">
      <div class="mb-4 flex flex-wrap items-center gap-3">
        <Space>
          <Button type="primary" @click="openCreate">新增任务</Button>
          <Button @click="openLogs()">全部日志</Button>
        </Space>
        <Input
          v-model:value="searchName"
          allow-clear
          placeholder="按任务名称搜索"
          style="width: 280px"
          @press-enter="handleSearch"
        />
        <Space>
          <Button type="primary" @click="handleSearch">搜索</Button>
          <Button @click="handleReset">重置</Button>
        </Space>
      </div>

      <TaskList
        :tasks="tasks"
        :loading="loading"
        :pagination="taskPagination"
        :triggering-task-ids="triggeringTaskIds"
        :deleting-task-ids="deletingTaskIds"
        :can-trigger-task="canTriggerTask"
        @table-change="handleTaskTableChange"
        @trigger="triggerTask"
        @open-script="openScriptManager"
        @open-logs="openLogs"
        @open-edit="openEdit"
        @delete="deleteTask"
      />
    </Card>

    <TaskFormModal
      v-model:open="taskModalVisible"
      :mode="taskModalMode"
      :submitting="taskSubmitting"
      :task-form="taskForm"
      :editing-task="editingTask"
      :task-can-enable="taskCanEnable"
      :before-task-file-upload="beforeTaskFileUpload"
      :remove-task-file="removeTaskFile"
      :sync-file-mode-status="syncFileModeStatus"
      @submit="submitTask"
    />

    <ScriptManagerModal
      v-model:open="scriptModalVisible"
      :title="scriptModalTitle"
      :current-task="currentScriptTask"
      :script-history="scriptHistory"
      :history-loading="scriptHistoryLoading"
      :pagination="scriptPagination"
      :script-uploading="scriptUploading"
      :activating-script-ids="activatingScriptIds"
      :before-script-file-upload="beforeScriptFileUpload"
      :remove-script-file="removeScriptFile"
      @upload="uploadScript"
      @activate="activateScript"
      @table-change="handleScriptTableChange"
    />

    <TaskLogDrawer
      v-model:open="logDrawerVisible"
      :title="logDrawerTitle"
      :logs="logs"
      :logs-loading="logsLoading"
      :pagination="logPagination"
      :detail-visible="logDetailVisible"
      :detail-loading="detailLoading"
      :current-log="currentLog"
      @update:detail-visible="logDetailVisible = $event"
      @refresh="fetchLogs"
      @table-change="handleLogTableChange"
      @open-detail="openLogDetail"
    />
  </Page>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
