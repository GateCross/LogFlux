<script setup lang="ts">
import type { useSimpleWafBlock } from '../composables/useSimpleWafBlock';

/** 接受 composable 返回值作为单个 prop */
defineProps<{
  waf: ReturnType<typeof useSimpleWafBlock>;
}>();
</script>

<template>
  <div class="h-full min-h-0 overflow-auto">
    <Spin :show="waf.simpleWafLoading.value">
      <Space vertical size="large">
        <Card size="small" :bordered="false">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div class="font-semibold">防火墙设置</div>
              <Tag size="small" :type="waf.simpleWafStatusType.value" :bordered="false">
                {{ waf.simpleWafStatusText.value }}
              </Tag>
            </div>
          </template>

          <div class="grid mb-4 gap-3 lg:grid-cols-2">
            <div class="border border-#e5e7eb rounded-8px px-3 py-2">
              <div class="text-xs text-gray-500">Coraza 版本</div>
              <div class="mt-1 font-medium">{{ waf.formatVersion(waf.simpleWafStatus.value?.corazaVersion) }}</div>
            </div>
            <div class="border border-#e5e7eb rounded-8px px-3 py-2">
              <div class="text-xs text-gray-500">CRS 版本</div>
              <div class="mt-1 font-medium">{{ waf.formatVersion(waf.simpleWafStatus.value?.crsVersion) }}</div>
            </div>
          </div>

          <Form label-placement="top">
            <div class="grid gap-4 lg:grid-cols-3">
              <Form.Item label="启用">
                <Switch v-model:value="waf.simpleWafForm.enabled" />
              </Form.Item>
              <Form.Item label="模式">
                <Select v-model:value="waf.simpleWafForm.mode" :options="waf.simpleWafModeOptions" :disabled="!waf.simpleWafForm.enabled" />
              </Form.Item>
              <Form.Item label="强度">
                <Select v-model:value="waf.simpleWafForm.strength" :options="waf.simpleWafStrengthOptions" :disabled="!waf.simpleWafForm.enabled" />
              </Form.Item>
            </div>

            <div class="grid gap-4 lg:grid-cols-3">
              <Form.Item label="审计日志">
                <Select v-model:value="waf.simpleWafForm.audit" :options="waf.simpleWafAuditOptions" />
              </Form.Item>
              <Form.Item label="请求体上限(MB)">
                <InputNumber v-model:value="waf.simpleWafForm.requestBodyLimitMB" :min="1" :max="1024" />
              </Form.Item>
              <Form.Item label="无文件请求体上限(MB)">
                <InputNumber v-model:value="waf.simpleWafForm.requestBodyNoFilesLimitMB" :min="1" :max="1024" />
              </Form.Item>
            </div>

            <Form.Item label="请求体检查">
              <Switch v-model:value="waf.simpleWafForm.requestBodyAccess" />
            </Form.Item>

            <Form.Item label="适用站点">
              <Checkbox.Group :value="waf.simpleWafForm.siteAddresses" @update:value="waf.handleSimpleWafSiteChange">
                <Space wrap>
                  <Checkbox v-for="item in waf.simpleWafSiteOptions.value" :key="item.value" :value="item.value" :label="item.label" />
                </Space>
              </Checkbox.Group>
            </Form.Item>
          </Form>

          <Alert v-if="waf.simpleWafStatus.value?.message" type="info" :show-icon="true">
            {{ waf.simpleWafStatus.value.message }}
          </Alert>

          <div class="mt-4 flex flex-wrap justify-end gap-2">
            <Button :loading="waf.simpleWafSaving.value" @click="waf.saveSimpleWafConfig">保存设置</Button>
            <Button :loading="waf.simpleWafPreviewing.value" @click="waf.previewSimpleWaf">预览变更</Button>
            <Button type="primary" :loading="waf.simpleWafSubmitting.value" @click="waf.applySimpleWaf">
              应用到 Caddy
            </Button>
          </div>
        </Card>

        <Card v-if="waf.simpleWafStatus.value?.directives" size="small" :bordered="false">
          <template #header>当前指令</template>
          <pre style="white-space: pre-wrap; word-break: break-all; max-height: 400px; overflow: auto; background: #f5f5f5; padding: 12px; border-radius: 6px; font-size: 12px;">{{ waf.simpleWafStatus.value.directives }}</pre>
        </Card>
      </Space>
    </Spin>

    <Modal v-model:open="waf.showSimpleWafPreview.value" title="防火墙配置预览" width="90vw" :body-style="{ maxWidth: '64rem', margin: '0 auto' }">
      <Space vertical size="large">
        <div v-if="waf.simpleWafPreviewResult.value?.actions?.length" class="flex flex-wrap gap-2">
          <Tag v-for="item in waf.simpleWafPreviewResult.value.actions" :key="item" size="small" type="info" :bordered="false">
            {{ item }}
          </Tag>
        </div>
        <pre v-if="waf.simpleWafPreviewResult.value?.directives" style="white-space: pre-wrap; word-break: break-all; max-height: 400px; overflow: auto; background: #f5f5f5; padding: 12px; border-radius: 6px; font-size: 12px;">{{ waf.simpleWafPreviewResult.value.directives }}</pre>
        <Input.TextArea
          v-if="waf.simpleWafPreviewResult.value?.config"
          readonly
          :value="waf.simpleWafPreviewResult.value.config"
          :auto-size="{ minRows: 12, maxRows: 24 }"
        />
      </Space>
    </Modal>
  </div>
</template>
