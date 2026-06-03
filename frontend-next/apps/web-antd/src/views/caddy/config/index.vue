<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { Page } from '@vben/common-ui';

import {
  Button,
  Card,
  Drawer,
  Input,
  List,
  message,
  Modal,
  Select,
  SelectOption,
  Space,
  Spin,
  Tag,
} from 'ant-design-vue';

import {
  getCaddyConfigApi,
  getCaddyConfigHistoryListApi,
  getCaddyServerListApi,
  previewCaddyConfigApi,
  pushCaddyConfigApi,
  rollbackCaddyConfigApi,
} from '#/api/caddy/server';

import type { CaddyServerApi } from '#/api/caddy/server';

// --------------- state ---------------
const servers = ref<CaddyServerApi.CaddyServer[]>([]);
const selectedServerId = ref<number | undefined>(undefined);
const configContent = ref('');
const loading = ref(false);
const saving = ref(false);
const previewing = ref(false);

// history drawer
const historyDrawerVisible = ref(false);
const historyList = ref<CaddyServerApi.ConfigHistoryItem[]>([]);
const historyLoading = ref(false);

// preview drawer
const previewDrawerVisible = ref(false);
const previewResult = ref<CaddyServerApi.ConfigPreview | null>(null);
const previewLoading = ref(false);

// --------------- helpers ---------------
const serverOptions = computed(() =>
  servers.value.map((s) => ({
    label: s.name ?? s.host ?? `Server #${s.id}`,
    value: s.id,
  })),
);

// --------------- data fetching ---------------
async function fetchServers() {
  try {
    servers.value = await getCaddyServerListApi();
    if (servers.value.length > 0 && !selectedServerId.value) {
      selectedServerId.value = servers.value[0].id;
    }
  } catch {
    message.error('Failed to load server list');
  }
}

async function fetchConfig() {
  if (!selectedServerId.value) return;
  loading.value = true;
  try {
    const cfg = await getCaddyConfigApi(selectedServerId.value);
    configContent.value = typeof cfg === 'string' ? cfg : JSON.stringify(cfg, null, 2);
  } catch {
    message.error('Failed to load config');
  } finally {
    loading.value = false;
  }
}

async function handleSave() {
  if (!selectedServerId.value) {
    message.warning('Please select a server first');
    return;
  }
  saving.value = true;
  try {
    let parsed: any;
    try {
      parsed = JSON.parse(configContent.value);
    } catch {
      parsed = configContent.value; // send as raw string
    }
    await pushCaddyConfigApi(selectedServerId.value, parsed);
    message.success('Config saved successfully');
  } catch {
    message.error('Failed to save config');
  } finally {
    saving.value = false;
  }
}

async function handlePreview() {
  if (!selectedServerId.value) return;
  previewing.value = true;
  try {
    let parsed: any;
    try {
      parsed = JSON.parse(configContent.value);
    } catch {
      parsed = configContent.value;
    }
    previewResult.value = await previewCaddyConfigApi(selectedServerId.value, parsed);
    previewDrawerVisible.value = true;
  } catch {
    message.error('Failed to preview config');
  } finally {
    previewing.value = false;
  }
}

async function fetchHistory() {
  if (!selectedServerId.value) return;
  historyDrawerVisible.value = true;
  historyLoading.value = true;
  try {
    historyList.value = await getCaddyConfigHistoryListApi(selectedServerId.value);
  } catch {
    message.error('Failed to load history');
  } finally {
    historyLoading.value = false;
  }
}

function confirmRollback(historyId: number) {
  Modal.confirm({
    title: 'Confirm Rollback',
    content: 'Are you sure you want to rollback to this config version?',
    async onOk() {
      if (!selectedServerId.value) return;
      try {
        await rollbackCaddyConfigApi(selectedServerId.value, { historyId });
        message.success('Rollback successful');
        historyDrawerVisible.value = false;
        await fetchConfig();
      } catch {
        message.error('Rollback failed');
      }
    },
  });
}

// --------------- lifecycle ---------------
onMounted(async () => {
  await fetchServers();
  if (selectedServerId.value) {
    await fetchConfig();
  }
});

// watch server selection change
function handleServerChange(val: number) {
  selectedServerId.value = val;
  fetchConfig();
}
</script>

<template>
  <Page description="Manage Caddy server configurations" title="Caddy Config Editor">
    <div style="display: flex; gap: 16px; height: calc(100vh - 200px);">
      <!-- Left panel: Server list -->
      <Card title="Servers" style="width: 280px; flex-shrink: 0; overflow: auto;">
        <Select
          :value="selectedServerId"
          placeholder="Select a server"
          style="width: 100%; margin-bottom: 16px;"
          @change="handleServerChange"
        >
          <SelectOption v-for="opt in serverOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </SelectOption>
        </Select>

        <List :data-source="servers" size="small">
          <template #renderItem="{ item }">
            <List.Item
              :style="{
                cursor: 'pointer',
                background: item.id === selectedServerId ? '#e6f7ff' : 'transparent',
                borderRadius: '4px',
                padding: '8px',
              }"
              @click="handleServerChange(item.id)"
            >
              <List.Item.Meta>
                <template #title>
                  <span>{{ item.name ?? item.host ?? `Server #${item.id}` }}</span>
                </template>
                <template #description>
                  <Tag :color="item.status === 'online' ? 'green' : 'default'">
                    {{ item.status ?? 'unknown' }}
                  </Tag>
                </template>
              </List.Item.Meta>
            </List.Item>
          </template>
        </List>
      </Card>

      <!-- Right panel: Config editor -->
      <Card style="flex: 1; display: flex; flex-direction: column;" :body-style="{ flex: 1, display: 'flex', flexDirection: 'column' }">
        <template #title>
          <div style="display: flex; justify-content: space-between; align-items: center;">
            <span>Configuration</span>
            <Space>
              <Button :loading="previewing" @click="handlePreview">Preview</Button>
              <Button @click="fetchHistory">History</Button>
              <Button type="primary" :loading="saving" @click="handleSave">Save</Button>
            </Space>
          </div>
        </template>

        <Spin :spinning="loading">
          <Input.TextArea
            v-model:value="configContent"
            :auto-size="false"
            placeholder="Select a server to load its config..."
            style="width: 100%; height: 100%; min-height: 400px; font-family: monospace; font-size: 13px;"
          />
          <div style="margin-top: 8px; color: #999; font-size: 12px;">
            Note: Monaco Editor integration coming soon. Currently using a plain textarea.
          </div>
        </Spin>
      </Card>
    </div>

    <!-- History Drawer -->
    <Drawer
      :open="historyDrawerVisible"
      title="Config History"
      width="560"
      @close="historyDrawerVisible = false"
    >
      <Spin :spinning="historyLoading">
        <List :data-source="historyList">
          <template #renderItem="{ item }">
            <List.Item>
              <List.Item.Meta>
                <template #title>
                  <span>{{ item.createdAt ?? item.timestamp ?? `Version #${item.id}` }}</span>
                </template>
                <template #description>
                  <span>{{ item.summary ?? item.description ?? '' }}</span>
                </template>
              </List.Item.Meta>
              <template #actions>
                <Button size="small" type="link" @click="confirmRollback(item.id)">
                  Rollback
                </Button>
              </template>
            </List.Item>
          </template>
        </List>
      </Spin>
    </Drawer>

    <!-- Preview Drawer -->
    <Drawer
      :open="previewDrawerVisible"
      title="Config Preview"
      width="640"
      @close="previewDrawerVisible = false"
    >
      <pre style="white-space: pre-wrap; word-break: break-all;">{{ JSON.stringify(previewResult, null, 2) }}</pre>
    </Drawer>
  </Page>
</template>
