<script lang="ts" setup>
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  DescriptionsItem,
  Drawer,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
  FormItem,
} from 'antdv-next';

import type { CaddySimpleWafApi } from '#/api/caddy/simple-waf';

export type WafFormState = {
  audit: CaddySimpleWafApi.SimpleWafAudit;
  enabled: boolean;
  mode: CaddySimpleWafApi.SimpleWafMode;
  requestBodyAccess: boolean;
  requestBodyLimitMB: number;
  requestBodyNoFilesLimitMB: number;
  siteAddresses: string[];
  strength: CaddySimpleWafApi.SimpleWafStrength;
};

const props = defineProps<{
  wafLoading: boolean;
  wafSaving: boolean;
  wafApplying: boolean;
  wafPreviewing: boolean;
  wafStatusText: string;
  wafStatusColor: string;
  /** 读失败页内错误 */
  wafErrorMessage?: string | null;
  wafForm: WafFormState;
  wafAvailableSites: string[];
  wafModeOptions: Array<{ label: string; value: CaddySimpleWafApi.SimpleWafMode }>;
  wafStrengthOptions: Array<{ label: string; value: CaddySimpleWafApi.SimpleWafStrength }>;
  wafAuditOptions: Array<{ label: string; value: CaddySimpleWafApi.SimpleWafAudit }>;
  wafPreviewVisible: boolean;
  wafPreviewResult: CaddySimpleWafApi.SimpleWafConfig | null;
}>();

const emit = defineEmits<{
  'update:wafPreviewVisible': [value: boolean];
  save: [];
  preview: [];
  apply: [];
  refresh: [];
}>();
</script>

<template>
  <div class="waf-pane">
    <Spin :spinning="props.wafLoading">
      <Alert
        v-if="props.wafErrorMessage"
        class="mb-3"
        type="error"
        show-icon
        :message="props.wafErrorMessage"
      />
      <Card title="简易 WAF">
        <template #extra>
          <Tag :color="props.wafStatusColor">{{ props.wafStatusText }}</Tag>
        </template>
        <Form layout="vertical">
          <Row :gutter="16">
            <Col :xs="24" :md="8">
              <FormItem label="启用 WAF">
                <Switch v-model:checked="props.wafForm.enabled" />
              </FormItem>
            </Col>
            <Col :xs="24" :md="8">
              <FormItem label="引擎模式">
                <Select
                  v-model:value="props.wafForm.mode"
                  :disabled="!props.wafForm.enabled"
                  :options="props.wafModeOptions"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="8">
              <FormItem label="规则强度">
                <Select
                  v-model:value="props.wafForm.strength"
                  :disabled="!props.wafForm.enabled"
                  :options="props.wafStrengthOptions"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="8">
              <FormItem label="审计日志">
                <Select
                  v-model:value="props.wafForm.audit"
                  :disabled="!props.wafForm.enabled"
                  :options="props.wafAuditOptions"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="8">
              <FormItem label="请求体检测">
                <Switch
                  v-model:checked="props.wafForm.requestBodyAccess"
                  :disabled="!props.wafForm.enabled"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="4">
              <FormItem label="请求体 MB">
                <InputNumber
                  v-model:value="props.wafForm.requestBodyLimitMB"
                  :min="1"
                  :disabled="!props.wafForm.enabled"
                  class="w-full"
                />
              </FormItem>
            </Col>
            <Col :xs="24" :md="4">
              <FormItem label="无文件 MB">
                <InputNumber
                  v-model:value="props.wafForm.requestBodyNoFilesLimitMB"
                  :min="1"
                  :disabled="!props.wafForm.enabled"
                  class="w-full"
                />
              </FormItem>
            </Col>
            <Col :xs="24">
              <FormItem label="应用站点">
                <Select
                  v-model:value="props.wafForm.siteAddresses"
                  :options="props.wafAvailableSites.map((item: string) => ({ label: item, value: item }))"
                  :disabled="!props.wafForm.enabled"
                  mode="multiple"
                  placeholder="默认应用到所有可用站点"
                />
              </FormItem>
            </Col>
          </Row>
          <Space>
            <Button :loading="props.wafSaving" @click="emit('save')">保存设置</Button>
            <Button :loading="props.wafPreviewing" @click="emit('preview')">预览变更</Button>
            <Button type="primary" :loading="props.wafApplying" @click="emit('apply')">应用到 Caddy</Button>
            <Button @click="emit('refresh')">刷新</Button>
          </Space>
        </Form>
      </Card>
    </Spin>

    <Drawer
      :open="props.wafPreviewVisible"
      title="WAF 变更预览"
      :size="720"
      @update:open="emit('update:wafPreviewVisible', $event)"
    >
      <Descriptions v-if="props.wafPreviewResult" bordered size="small" class="mb-3">
        <DescriptionsItem label="状态">{{ props.wafPreviewResult.message || '已生成预览' }}</DescriptionsItem>
        <DescriptionsItem label="Coraza">{{ props.wafPreviewResult.corazaVersion || '-' }}</DescriptionsItem>
        <DescriptionsItem label="CRS">{{ props.wafPreviewResult.crsVersion || '-' }}</DescriptionsItem>
      </Descriptions>
      <Space v-if="props.wafPreviewResult?.actions?.length" wrap class="mb-3">
        <Tag v-for="item in props.wafPreviewResult.actions" :key="item">{{ item }}</Tag>
      </Space>
      <Input.TextArea
        :value="props.wafPreviewResult?.directives || props.wafPreviewResult?.config || ''"
        :auto-size="{ minRows: 18, maxRows: 28 }"
        class="code-textarea"
        readonly
      />
    </Drawer>
  </div>
</template>

<style scoped>
.w-full {
  width: 100%;
}

.code-textarea {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 13px;
}

.mb-3 {
  margin-bottom: 12px;
}
</style>
