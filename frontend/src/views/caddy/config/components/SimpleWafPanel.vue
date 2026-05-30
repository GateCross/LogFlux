<script setup lang="ts">
import type { useSimpleWafBlock } from '../composables/useSimpleWafBlock';

/** 接受 composable 返回值作为单个 prop */
defineProps<{
  waf: ReturnType<typeof useSimpleWafBlock>;
}>();
</script>

<template>
  <div class="h-full min-h-0 overflow-auto">
    <NSpin :show="waf.simpleWafLoading.value">
      <NSpace vertical size="large">
        <NCard size="small" :bordered="false">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div class="font-semibold">防火墙设置</div>
              <NTag size="small" :type="waf.simpleWafStatusType.value" :bordered="false">
                {{ waf.simpleWafStatusText.value }}
              </NTag>
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

          <NForm label-placement="top">
            <div class="grid gap-4 lg:grid-cols-3">
              <NFormItem label="启用">
                <NSwitch v-model:value="waf.simpleWafForm.enabled" />
              </NFormItem>
              <NFormItem label="模式">
                <NSelect v-model:value="waf.simpleWafForm.mode" :options="waf.simpleWafModeOptions" :disabled="!waf.simpleWafForm.enabled" />
              </NFormItem>
              <NFormItem label="强度">
                <NSelect v-model:value="waf.simpleWafForm.strength" :options="waf.simpleWafStrengthOptions" :disabled="!waf.simpleWafForm.enabled" />
              </NFormItem>
            </div>

            <div class="grid gap-4 lg:grid-cols-3">
              <NFormItem label="审计日志">
                <NSelect v-model:value="waf.simpleWafForm.audit" :options="waf.simpleWafAuditOptions" />
              </NFormItem>
              <NFormItem label="请求体上限(MB)">
                <NInputNumber v-model:value="waf.simpleWafForm.requestBodyLimitMB" :min="1" :max="1024" />
              </NFormItem>
              <NFormItem label="无文件请求体上限(MB)">
                <NInputNumber v-model:value="waf.simpleWafForm.requestBodyNoFilesLimitMB" :min="1" :max="1024" />
              </NFormItem>
            </div>

            <NFormItem label="请求体检查">
              <NSwitch v-model:value="waf.simpleWafForm.requestBodyAccess" />
            </NFormItem>

            <NFormItem label="适用站点">
              <NCheckboxGroup :value="waf.simpleWafForm.siteAddresses" @update:value="waf.handleSimpleWafSiteChange">
                <NSpace wrap>
                  <NCheckbox v-for="item in waf.simpleWafSiteOptions.value" :key="item.value" :value="item.value" :label="item.label" />
                </NSpace>
              </NCheckboxGroup>
            </NFormItem>
          </NForm>

          <NAlert v-if="waf.simpleWafStatus.value?.message" type="info" :show-icon="true">
            {{ waf.simpleWafStatus.value.message }}
          </NAlert>

          <div class="mt-4 flex flex-wrap justify-end gap-2">
            <NButton :loading="waf.simpleWafSaving.value" @click="waf.saveSimpleWafConfig">保存设置</NButton>
            <NButton :loading="waf.simpleWafPreviewing.value" @click="waf.previewSimpleWaf">预览变更</NButton>
            <NButton type="primary" :loading="waf.simpleWafSubmitting.value" @click="waf.applySimpleWaf">
              应用到 Caddy
            </NButton>
          </div>
        </NCard>

        <NCard v-if="waf.simpleWafStatus.value?.directives" size="small" :bordered="false">
          <template #header>当前指令</template>
          <NCode :code="waf.simpleWafStatus.value.directives" language="shell" word-wrap />
        </NCard>
      </NSpace>
    </NSpin>

    <NModal v-model:show="waf.showSimpleWafPreview.value" preset="card" title="防火墙配置预览" class="max-w-5xl w-[90vw]">
      <NSpace vertical size="large">
        <div v-if="waf.simpleWafPreviewResult.value?.actions?.length" class="flex flex-wrap gap-2">
          <NTag v-for="item in waf.simpleWafPreviewResult.value.actions" :key="item" size="small" type="info" :bordered="false">
            {{ item }}
          </NTag>
        </div>
        <NCode v-if="waf.simpleWafPreviewResult.value?.directives" :code="waf.simpleWafPreviewResult.value.directives" language="shell" word-wrap />
        <NInput
          v-if="waf.simpleWafPreviewResult.value?.config"
          type="textarea"
          readonly
          :value="waf.simpleWafPreviewResult.value.config"
          :autosize="{ minRows: 12, maxRows: 24 }"
        />
      </NSpace>
    </NModal>
  </div>
</template>
