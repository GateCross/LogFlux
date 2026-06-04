<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Button, Card, Space, Table, Tag } from 'ant-design-vue';

import { getWafReleaseListApi } from '#/api/caddy/release';
import { getWafSourceListApi } from '#/api/caddy/source';

defineOptions({ name: 'SecurityCrs' });

const loading = ref(false);
const sources = ref<any[]>([]);
const releases = ref<any[]>([]);

const sourceColumns = [
  { dataIndex: 'name', key: 'name', title: '名称' },
  { dataIndex: 'kind', key: 'kind', title: '类型', width: 120 },
  { dataIndex: 'mode', key: 'mode', title: '模式', width: 120 },
  { dataIndex: 'lastRelease', key: 'lastRelease', title: '最新发布' },
  { dataIndex: 'enabled', key: 'enabled', title: '启用', width: 100 },
];

const releaseColumns = [
  { dataIndex: 'version', key: 'version', title: '版本' },
  { dataIndex: 'kind', key: 'kind', title: '类型', width: 120 },
  { dataIndex: 'status', key: 'status', title: '状态', width: 120 },
  { dataIndex: 'createdAt', key: 'createdAt', title: '创建时间' },
];

async function fetchData() {
  loading.value = true;
  try {
    const [sourceList, releaseList] = await Promise.all([
      getWafSourceListApi(),
      getWafReleaseListApi(),
    ]);
    sources.value = sourceList;
    releases.value = releaseList;
  } finally {
    loading.value = false;
  }
}

onMounted(fetchData);
</script>

<template>
  <Page title="CRS 管理" description="查看 WAF 规则源与 CRS 发布版本。">
    <Space direction="vertical" size="large" style="width: 100%;">
      <Card title="规则源">
        <template #extra>
          <Button :loading="loading" @click="fetchData">刷新</Button>
        </template>
        <Table :columns="sourceColumns" :data-source="sources" :loading="loading" row-key="id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'enabled'">
              <Tag :color="record.enabled ? 'green' : 'default'">{{ record.enabled ? '启用' : '停用' }}</Tag>
            </template>
          </template>
        </Table>
      </Card>

      <Card title="发布版本">
        <Table :columns="releaseColumns" :data-source="releases" :loading="loading" row-key="id">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <Tag :color="record.status === 'active' ? 'green' : record.status === 'failed' ? 'red' : 'blue'">
                {{ record.status }}
              </Tag>
            </template>
          </template>
        </Table>
      </Card>
    </Space>
  </Page>
</template>
