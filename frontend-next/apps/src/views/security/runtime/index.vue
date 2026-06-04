<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Button, Card, Descriptions, DescriptionsItem, Space, Tag, message } from 'ant-design-vue';

import {
  checkWafEngineUpdateApi,
  getWafEngineStatusApi,
} from '#/api/caddy/source';

defineOptions({ name: 'SecurityRuntime' });

const loading = ref(false);
const checking = ref(false);
const status = ref<Record<string, any> | null>(null);

async function fetchStatus() {
  loading.value = true;
  try {
    status.value = await getWafEngineStatusApi();
  } finally {
    loading.value = false;
  }
}

async function handleCheck() {
  checking.value = true;
  try {
    await checkWafEngineUpdateApi();
    message.success('WAF 引擎检查已触发');
    await fetchStatus();
  } finally {
    checking.value = false;
  }
}

onMounted(fetchStatus);
</script>

<template>
  <Page title="WAF 运行时" description="查看 Coraza/WAF 引擎版本与升级状态。">
    <Card :loading="loading">
      <template #extra>
        <Space>
          <Button @click="fetchStatus">刷新</Button>
          <Button type="primary" :loading="checking" @click="handleCheck">检查更新</Button>
        </Space>
      </template>

      <Descriptions bordered :column="1" size="small">
        <DescriptionsItem label="服务器 ID">{{ status?.serverId ?? '-' }}</DescriptionsItem>
        <DescriptionsItem label="当前版本">{{ status?.currentVersion || '-' }}</DescriptionsItem>
        <DescriptionsItem label="最新版本">{{ status?.latestVersion || '-' }}</DescriptionsItem>
        <DescriptionsItem label="可升级">
          <Tag :color="status?.canUpgrade ? 'orange' : 'green'">
            {{ status?.canUpgrade ? '是' : '否' }}
          </Tag>
        </DescriptionsItem>
        <DescriptionsItem label="检查时间">{{ status?.checkedAt || '-' }}</DescriptionsItem>
        <DescriptionsItem label="来源">{{ status?.source || '-' }}</DescriptionsItem>
        <DescriptionsItem label="消息">{{ status?.message || '-' }}</DescriptionsItem>
      </Descriptions>
    </Card>
  </Page>
</template>
