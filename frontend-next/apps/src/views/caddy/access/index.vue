<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { Alert, Button, Card, Form, Select, Spin, Switch, Tag, message } from 'ant-design-vue';
import { getIpRegionConfigApi, updateIpRegionConfigApi } from '#/api/caddy/ip-region';

const loading = ref(false);
const saving = ref(false);
const enabled = ref(true);
const allowList = ref<string[]>(['中国']);

const regionOptions = [
  { label: '中国', value: '中国' },
  { label: '美国', value: '美国' },
  { label: '日本', value: '日本' },
  { label: '韩国', value: '韩国' },
  { label: '新加坡', value: '新加坡' },
  { label: '德国', value: '德国' },
  { label: '英国', value: '英国' },
  { label: '法国', value: '法国' },
  { label: '加拿大', value: '加拿大' },
  { label: '澳大利亚', value: '澳大利亚' },
  { label: '印度', value: '印度' },
  { label: '巴西', value: '巴西' },
  { label: '俄罗斯', value: '俄罗斯' },
  { label: '荷兰', value: '荷兰' },
  { label: '泰国', value: '泰国' },
  { label: '越南', value: '越南' },
  { label: '马来西亚', value: '马来西亚' },
  { label: '印度尼西亚', value: '印度尼西亚' },
  { label: '菲律宾', value: '菲律宾' },
  { label: '土耳其', value: '土耳其' },
];

async function loadConfig() {
  loading.value = true;
  try {
    const data = await getIpRegionConfigApi();
    if (data) {
      enabled.value = data.enabled;
      allowList.value = data.allowList?.length ? [...data.allowList] : ['中国'];
    }
  } catch {
    // error handled by request interceptor
  } finally {
    loading.value = false;
  }
}

async function handleSave() {
  saving.value = true;
  try {
    await updateIpRegionConfigApi({
      enabled: enabled.value,
      allowList: allowList.value,
    });
    message.success('保存成功');
  } catch {
    message.error('保存失败');
  } finally {
    saving.value = false;
  }
}

onMounted(loadConfig);
</script>

<template>
  <div class="p-4">
    <Card title="访问控制">
      <template #extra>
        <Tag :color="enabled ? 'green' : 'default'">
          {{ enabled ? '已启用' : '已禁用' }}
        </Tag>
      </template>

      <Spin :spinning="loading">
        <Form layout="vertical" class="max-w-lg">
          <Form.Item label="启用 IP 区域限制">
            <Switch v-model:checked="enabled" />
          </Form.Item>

          <Form.Item label="允许访问的地区">
            <Select
              v-model:value="allowList"
              :options="regionOptions"
              :disabled="!enabled"
              mode="multiple"
              placeholder="选择允许访问的地区"
            />
          </Form.Item>

          <Form.Item label="说明">
            <Alert type="info" :show-icon="true">
              <template #message>
                配置后，仅允许列表中的地区 IP 访问系统。Caddy 代理模式下通过 forward_auth
                在网关层拦截，后端中间件作为第二层防护。修改后立即生效，无需重启服务。
              </template>
            </Alert>
          </Form.Item>

          <Form.Item>
            <Button type="primary" :loading="saving" :disabled="loading" @click="handleSave">
              保存
            </Button>
          </Form.Item>
        </Form>
      </Spin>
    </Card>
  </div>
</template>
