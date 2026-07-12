<script lang="ts" setup>
import {
  Col,
  Empty,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  Switch,
  Tag,
  FormItem,
} from 'antdv-next';

import type { HealthCheck } from '../types';
import type { QuickSiteDraft } from '../quick-config-utils';

const props = defineProps<{
  activeQuickSite: QuickSiteDraft | null | undefined;
  quickModeOptions: Array<{ label: string; value: string }>;
  tlsModeOptions: Array<{ label: string; value: string }>;
  lbPolicyOptions: Array<{ label: string; value: string }>;
  upstreamPoolOptions: Array<{ label: string; value: string }>;
  isSiteHealthEnabled: (site: QuickSiteDraft) => boolean;
  setSiteHealthEnabled: (site: QuickSiteDraft, enabled: boolean) => void;
  ensureSiteHealth: (site: QuickSiteDraft) => HealthCheck;
}>();
</script>

<template>
  <div>
    <div class="panel-subtitle mb-3">域名、站点类型、TLS、负载策略与可选健康检查。</div>

    <Empty v-if="!props.activeQuickSite" description="请选择或新建一个站点" />
    <Form v-else layout="vertical">
      <Row :gutter="16">
        <Col :xs="24" :md="12">
          <FormItem label="站点名称">
            <Input v-model:value="props.activeQuickSite.name" placeholder="例如：官网反代" />
          </FormItem>
        </Col>
        <Col :xs="24" :md="12">
          <FormItem label="启用状态">
            <Switch v-model:checked="props.activeQuickSite.enabled" />
          </FormItem>
        </Col>
        <Col :xs="24">
          <FormItem label="域名 / 监听地址">
            <Select
              v-model:value="props.activeQuickSite.domains"
              mode="tags"
              placeholder="example.com 或 :8080"
            />
          </FormItem>
        </Col>
        <Col :xs="24" :md="12">
          <FormItem label="站点类型">
            <Select v-model:value="props.activeQuickSite.mode" :options="props.quickModeOptions" />
          </FormItem>
        </Col>
        <Col :xs="24" :md="12">
          <FormItem label="TLS">
            <Select v-model:value="props.activeQuickSite.tlsMode" :options="props.tlsModeOptions" />
          </FormItem>
        </Col>
        <template v-if="props.activeQuickSite.mode === 'reverse_proxy'">
          <Col :xs="24" :md="12">
            <FormItem label="上游地址或池名">
              <Input
                v-model:value="props.activeQuickSite.upstream"
                placeholder="127.0.0.1:8080 或上游池名称"
              />
              <div v-if="props.upstreamPoolOptions.length" class="upstream-pool-hints">
                <span class="text-muted">点击填入上游池：</span>
                <Tag
                  v-for="opt in props.upstreamPoolOptions"
                  :key="opt.value"
                  class="pool-hint-tag"
                  color="blue"
                  @click="props.activeQuickSite!.upstream = String(opt.value)"
                >
                  {{ opt.value }}
                </Tag>
              </div>
            </FormItem>
          </Col>
          <Col :xs="24" :md="12">
            <FormItem label="负载策略">
              <Select
                :value="props.activeQuickSite.lbPolicy || 'round_robin'"
                :options="props.lbPolicyOptions"
                placeholder="负载策略"
                @change="(value: any) => { props.activeQuickSite!.lbPolicy = value; }"
              />
            </FormItem>
          </Col>
          <Col :xs="24">
            <FormItem label="站点级健康检查（可选，启用后覆盖上游池配置）">
              <div class="health-row site-health-row">
                <div class="health-toggle">
                  <span class="health-label">启用</span>
                  <Switch
                    :checked="props.isSiteHealthEnabled(props.activeQuickSite)"
                    checked-children="开"
                    un-checked-children="关"
                    @change="(checked: any) => props.setSiteHealthEnabled(props.activeQuickSite!, Boolean(checked))"
                  />
                </div>
                <template v-if="props.isSiteHealthEnabled(props.activeQuickSite)">
                  <Input
                    :value="props.ensureSiteHealth(props.activeQuickSite).path"
                    placeholder="探测路径，如 /health"
                    @update:value="(value: string) => { props.ensureSiteHealth(props.activeQuickSite!).path = value; }"
                  />
                  <Input
                    :value="props.ensureSiteHealth(props.activeQuickSite).interval"
                    placeholder="间隔，如 10s"
                    @update:value="(value: string) => { props.ensureSiteHealth(props.activeQuickSite!).interval = value; }"
                  />
                  <Input
                    :value="props.ensureSiteHealth(props.activeQuickSite).timeout"
                    placeholder="超时，如 5s"
                    @update:value="(value: string) => { props.ensureSiteHealth(props.activeQuickSite!).timeout = value; }"
                  />
                </template>
              </div>
            </FormItem>
          </Col>
        </template>
        <template v-if="props.activeQuickSite.mode === 'file_server'">
          <Col :xs="24" :md="18">
            <FormItem label="站点根目录">
              <Input v-model:value="props.activeQuickSite.root" placeholder="/srv/www/site" />
            </FormItem>
          </Col>
          <Col :xs="24" :md="6">
            <FormItem label="目录浏览">
              <Switch v-model:checked="props.activeQuickSite.browse" />
            </FormItem>
          </Col>
        </template>
        <template v-if="props.activeQuickSite.mode === 'redirect'">
          <Col :xs="24" :md="18">
            <FormItem label="跳转地址">
              <Input v-model:value="props.activeQuickSite.redirectTo" placeholder="https://example.com" />
            </FormItem>
          </Col>
          <Col :xs="24" :md="6">
            <FormItem label="状态码">
              <InputNumber
                v-model:value="props.activeQuickSite.redirectCode"
                :min="300"
                :max="399"
                class="w-full"
              />
            </FormItem>
          </Col>
        </template>
      </Row>
    </Form>
  </div>
</template>

<style scoped>
.panel-subtitle {
  margin-top: 2px;
  color: #667085;
  font-size: 12px;
}

.text-muted {
  color: #667085;
  font-size: 12px;
}

.health-row {
  display: grid;
  grid-template-columns: auto minmax(120px, 1fr) minmax(90px, 0.7fr) minmax(90px, 0.7fr);
  gap: 8px;
  align-items: center;
  margin-top: 8px;
}

.site-health-row {
  width: 100%;
}

.health-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.health-label {
  color: #475467;
  font-size: 13px;
}

.upstream-pool-hints {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
}

.pool-hint-tag {
  cursor: pointer;
}

.w-full {
  width: 100%;
}

.mb-3 {
  margin-bottom: 12px;
}

@media (max-width: 960px) {
  .health-row {
    grid-template-columns: 1fr;
  }
}
</style>
