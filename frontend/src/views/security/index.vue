<template>
  <div class="h-full flex flex-col gap-3">
    <n-alert type="info" :show-icon="true" class="rounded-8px">
      <template #header>{{ pageTitle }}</template>
      <div>
        CRS 支持在线同步、上传、激活与回滚；Coraza 引擎依赖 Caddy 二进制，当前版本仅提供升级源配置与版本记录，不支持在线替换引擎。
      </div>
    </n-alert>

    <n-card :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">Coraza 引擎版本检查</div>
          <div class="text-xs text-gray-500 mt-1">用于发现 Coraza 引擎新版本并生成升级建议（需通过镜像发布流程升级）。</div>
        </div>
        <div class="flex gap-2">
          <n-button size="small" :loading="engineLoading" @click="handleRefreshEngineStatus">刷新状态</n-button>
          <n-button size="small" type="primary" :loading="engineChecking" @click="handleCheckEngine">检查上游版本</n-button>
        </div>
      </div>

      <n-grid cols="4" x-gap="12" y-gap="10" class="mt-4">
        <n-gi>
          <div class="text-xs text-gray-500">当前版本</div>
          <div class="text-sm font-medium">{{ displayEngineValue(engineStatus?.currentVersion) }}</div>
        </n-gi>
        <n-gi>
          <div class="text-xs text-gray-500">最新版本</div>
          <div class="text-sm font-medium">{{ displayEngineValue(engineStatus?.latestVersion) }}</div>
        </n-gi>
        <n-gi>
          <div class="text-xs text-gray-500">可升级</div>
          <div class="text-sm font-medium">
            <n-tag :type="engineStatus?.canUpgrade ? 'warning' : 'success'" :bordered="false">
              {{ engineStatus?.canUpgrade ? '是' : '否' }}
            </n-tag>
          </div>
        </n-gi>
        <n-gi>
          <div class="text-xs text-gray-500">最近检查时间</div>
          <div class="text-sm font-medium">{{ displayEngineValue(engineStatus?.checkedAt) }}</div>
        </n-gi>
      </n-grid>

      <n-alert v-if="engineUnavailable" type="warning" :show-icon="true" class="mt-4">
        当前引擎状态接口暂不可用，已切换为占位模式，请检查后端日志。
      </n-alert>
      <n-alert v-else-if="engineStatus?.message" type="info" :show-icon="true" class="mt-4">
        {{ engineStatus?.message }}
      </n-alert>
    </n-card>

    <n-card :bordered="false" class="rounded-8px shadow-sm">
      <n-tabs v-model:value="activeTab" type="line" animated>
        <n-tab-pane name="source" tab="更新源配置">
          <div class="mb-3 flex flex-wrap gap-2 items-center">
            <n-input v-model:value="sourceQuery.name" placeholder="按名称搜索" clearable class="w-220px" @keyup.enter="fetchSources" />
            <n-select
              v-model:value="sourceQuery.kind"
              :options="kindFilterOptions"
              :disabled="Boolean(routeKind)"
              clearable
              placeholder="类型"
              class="w-160px"
            />
            <n-button type="primary" @click="fetchSources">
              <template #icon>
                <icon-carbon-search />
              </template>
              查询
            </n-button>
            <n-button @click="resetSourceQuery">重置</n-button>
            <n-button type="primary" @click="handleAddSource">
              <template #icon>
                <icon-ic-round-plus />
              </template>
              新增源
            </n-button>
            <n-button type="success" @click="openUploadModal">
              <template #icon>
                <icon-carbon-cloud-upload />
              </template>
              上传规则包
            </n-button>
          </div>

          <n-data-table
            remote
            :columns="sourceColumns"
            :data="sourceTable"
            :loading="sourceLoading"
            :pagination="sourcePagination"
            :row-key="row => row.id"
            class="min-h-260px"
            @update:page="handleSourcePageChange"
            @update:page-size="handleSourcePageSizeChange"
          />
        </n-tab-pane>

        <n-tab-pane name="release" tab="版本发布管理">
          <div class="mb-3 flex flex-wrap gap-2 items-center">
            <n-select
              v-model:value="releaseQuery.kind"
              :options="kindFilterOptions"
              :disabled="Boolean(routeKind)"
              clearable
              placeholder="类型"
              class="w-160px"
            />
            <n-select v-model:value="releaseQuery.status" :options="releaseStatusOptions" clearable placeholder="状态" class="w-160px" />
            <n-button type="primary" @click="fetchReleases">
              <template #icon>
                <icon-carbon-search />
              </template>
              查询
            </n-button>
            <n-button @click="resetReleaseQuery">重置</n-button>
            <n-button type="warning" @click="openRollbackModal">回滚到历史版本</n-button>
          </div>

          <n-data-table
            remote
            :columns="releaseColumns"
            :data="releaseTable"
            :loading="releaseLoading"
            :pagination="releasePagination"
            :row-key="row => row.id"
            class="min-h-260px"
            @update:page="handleReleasePageChange"
            @update:page-size="handleReleasePageSizeChange"
          />
        </n-tab-pane>

        <n-tab-pane name="job" tab="任务日志">
          <div class="mb-3 flex flex-wrap gap-2 items-center">
            <n-select v-model:value="jobQuery.status" :options="jobStatusOptions" clearable placeholder="状态" class="w-160px" />
            <n-select v-model:value="jobQuery.action" :options="jobActionOptions" clearable placeholder="动作" class="w-160px" />
            <n-button type="primary" @click="fetchJobs">
              <template #icon>
                <icon-carbon-search />
              </template>
              查询
            </n-button>
            <n-button @click="resetJobQuery">重置</n-button>
            <n-button type="success" @click="refreshCurrentTab">刷新</n-button>
          </div>

          <n-data-table
            remote
            :columns="jobColumns"
            :data="jobTable"
            :loading="jobLoading"
            :pagination="jobPagination"
            :row-key="row => row.id"
            class="min-h-260px"
            :scroll-x="1500"
            :resizable="true"
            @update:page="handleJobPageChange"
            @update:page-size="handleJobPageSizeChange"
          />
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <n-modal v-model:show="sourceModalVisible" preset="card" :title="sourceModalTitle" class="w-720px">
      <n-form ref="sourceFormRef" :model="sourceForm" :rules="sourceRules" label-placement="left" label-width="120">
        <n-grid cols="2" x-gap="12">
          <n-form-item-gi label="名称" path="name">
            <n-input v-model:value="sourceForm.name" placeholder="例如：official-crs" />
          </n-form-item-gi>
          <n-form-item-gi label="类型" path="kind">
            <n-select v-model:value="sourceForm.kind" :options="kindOptions" :disabled="Boolean(routeKind)" />
          </n-form-item-gi>
          <n-form-item-gi label="模式" path="mode">
            <n-select v-model:value="sourceForm.mode" :options="modeOptions" />
          </n-form-item-gi>
          <n-form-item-gi label="鉴权类型" path="authType">
            <n-select v-model:value="sourceForm.authType" :options="authTypeOptions" />
          </n-form-item-gi>
        </n-grid>

        <n-form-item label="默认源">
          <div class="flex flex-wrap gap-2">
            <n-button size="small" secondary @click="applyDefaultSource('crs')">应用 CRS 默认源</n-button>
            <n-button size="small" secondary @click="applyDefaultSource('coraza_engine')">应用 Coraza 默认源</n-button>
          </div>
        </n-form-item>

        <n-form-item label="源地址" path="url" v-if="sourceForm.mode === 'remote'">
          <n-input v-model:value="sourceForm.url" placeholder="https://api.github.com/repos/coreruleset/coreruleset/releases/latest" />
        </n-form-item>

        <n-form-item label="校验地址" path="checksumUrl" v-if="sourceForm.mode === 'remote'">
          <n-input v-model:value="sourceForm.checksumUrl" placeholder="可选，SHA256 清单地址" />
        </n-form-item>

        <n-form-item label="代理地址" path="proxyUrl" v-if="sourceForm.mode === 'remote'">
          <n-input v-model:value="sourceForm.proxyUrl" placeholder="可选，例如：http://127.0.0.1:7890" />
        </n-form-item>

        <n-form-item label="鉴权密钥" path="authSecret" v-if="sourceForm.authType !== 'none'">
          <n-input v-model:value="sourceForm.authSecret" type="password" show-password-on="mousedown" placeholder="Token 或 user:password" />
        </n-form-item>

        <n-form-item label="调度表达式" path="schedule">
          <n-input v-model:value="sourceForm.schedule" placeholder="例如：0 0 */6 * * *" />
        </n-form-item>

        <n-form-item label="附加元数据" path="meta">
          <n-input v-model:value="sourceForm.meta" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" placeholder="JSON 字符串，可选" />
        </n-form-item>

        <n-grid cols="2" x-gap="12">
          <n-form-item-gi label="启用">
            <n-switch v-model:value="sourceForm.enabled" />
          </n-form-item-gi>
          <n-form-item-gi label="自动检查">
            <n-switch v-model:value="sourceForm.autoCheck" />
          </n-form-item-gi>
          <n-form-item-gi label="自动下载">
            <n-switch v-model:value="sourceForm.autoDownload" />
          </n-form-item-gi>
          <n-form-item-gi label="自动激活">
            <n-switch v-model:value="sourceForm.autoActivate" />
          </n-form-item-gi>
        </n-grid>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="sourceModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="sourceSubmitting" @click="handleSubmitSource">保存</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="uploadModalVisible" preset="card" title="上传规则包" class="w-640px">
      <n-form ref="uploadFormRef" :model="uploadForm" :rules="uploadRules" label-placement="left" label-width="110">
        <n-form-item label="类型" path="kind">
          <n-select v-model:value="uploadForm.kind" :options="kindOptions" :disabled="Boolean(routeKind)" />
        </n-form-item>
        <n-form-item label="版本号" path="version">
          <n-input v-model:value="uploadForm.version" placeholder="例如：v4.23.0-custom.1" />
        </n-form-item>
        <n-form-item label="SHA256" path="checksum">
          <n-input v-model:value="uploadForm.checksum" placeholder="可选，建议填写" />
        </n-form-item>
        <n-form-item label="立即激活" path="activateNow">
          <n-switch v-model:value="uploadForm.activateNow" />
        </n-form-item>
        <n-form-item label="规则包" path="file">
          <n-upload
            :default-upload="false"
            :max="1"
            :show-file-list="true"
            accept=".zip,.tar.gz"
            @before-upload="handleBeforeUpload"
            @remove="handleRemoveUpload"
          >
            <n-button>选择文件</n-button>
          </n-upload>
        </n-form-item>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="uploadModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="uploadSubmitting" @click="handleSubmitUpload">上传并入库</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="rollbackModalVisible" preset="card" title="回滚版本" class="w-520px">
      <n-form ref="rollbackFormRef" :model="rollbackForm" :rules="rollbackRules" label-placement="left" label-width="110">
        <n-form-item label="回滚目标" path="target">
          <n-radio-group v-model:value="rollbackForm.target">
            <n-space>
              <n-radio value="last_good">last_good</n-radio>
              <n-radio value="version">指定版本</n-radio>
            </n-space>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="版本号" path="version" v-if="rollbackForm.target === 'version'">
          <n-input v-model:value="rollbackForm.version" placeholder="例如：v4.23.0" />
        </n-form-item>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="rollbackModalVisible = false">取消</n-button>
          <n-button type="warning" :loading="rollbackSubmitting" @click="handleSubmitRollback">确认回滚</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref, watch } from 'vue';
import {
  NButton,
  NPopconfirm,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  useDialog,
  useMessage,
  type DataTableColumns,
  type FormInst,
  type FormRules,
  type PaginationProps,
  type UploadFileInfo
} from 'naive-ui';
import {
  activateWafRelease,
  checkWafSource,
  checkWafEngine,
  createWafSource,
  deleteWafSource,
  fetchWafEngineStatus,
  fetchWafJobList,
  fetchWafReleaseList,
  fetchWafSourceList,
  rollbackWafRelease,
  syncWafSource,
  updateWafSource,
  uploadWafPackage,
  type WafAuthType,
  type WafJobItem,
  type WafJobStatus,
  type WafKind,
  type WafMode,
  type WafEngineStatusResp,
  type WafReleaseItem,
  type WafReleaseStatus,
  type WafSourceItem
} from '@/service/api/caddy';

const message = useMessage();
const dialog = useDialog();

const engineLoading = ref(false);
const engineChecking = ref(false);
const engineUnavailable = ref(false);
const engineStatus = ref<WafEngineStatusResp | null>(null);

const activeTab = ref<'source' | 'release' | 'job'>('source');

const kindOptions = [
  { label: 'CRS 规则集', value: 'crs' },
  { label: 'Coraza 引擎', value: 'coraza_engine' }
];

const kindFilterOptions = [{ label: '全部', value: '' }, ...kindOptions];

const modeOptions = [
  { label: '远程同步 (remote)', value: 'remote' },
  { label: '手动管理 (manual)', value: 'manual' }
];

const authTypeOptions = [
  { label: '无鉴权', value: 'none' },
  { label: 'Token', value: 'token' },
  { label: 'Basic', value: 'basic' }
];

const releaseStatusOptions = [
  { label: '全部', value: '' },
  { label: 'downloaded', value: 'downloaded' },
  { label: 'verified', value: 'verified' },
  { label: 'active', value: 'active' },
  { label: 'failed', value: 'failed' },
  { label: 'rolled_back', value: 'rolled_back' }
];

const jobStatusOptions = [
  { label: '全部', value: '' },
  { label: 'running', value: 'running' },
  { label: 'success', value: 'success' },
  { label: 'failed', value: 'failed' }
];

const jobActionOptions = [
  { label: '全部', value: '' },
  { label: '检查', value: 'check' },
  { label: '下载', value: 'download' },
  { label: '校验', value: 'verify' },
  { label: '激活', value: 'activate' },
  { label: '回滚', value: 'rollback' },
  { label: '引擎检查', value: 'engine_check' }
];

const sourceQuery = reactive({
  name: '',
  kind: '' as '' | WafKind
});

const routeKind = computed<'' | WafKind>(() => '');

const sourceLoading = ref(false);
const sourceTable = ref<WafSourceItem[]>([]);
const sourcePagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

const sourceModalVisible = ref(false);
const sourceModalMode = ref<'add' | 'edit'>('add');
const sourceSubmitting = ref(false);
const sourceFormRef = ref<FormInst | null>(null);
const sourceForm = reactive({
  id: 0,
  name: '',
  kind: 'crs' as WafKind,
  mode: 'remote' as WafMode,
  url: '',
  checksumUrl: '',
  proxyUrl: '',
  authType: 'none' as WafAuthType,
  authSecret: '',
  schedule: '',
  enabled: true,
  autoCheck: true,
  autoDownload: true,
  autoActivate: false,
  meta: ''
});

const sourceModalTitle = computed(() => (sourceModalMode.value === 'add' ? '新增更新源' : '编辑更新源'));
const pageTitle = computed(() => '安全升级管理');

const sourceRules: FormRules = {
  name: { required: true, message: '请输入源名称', trigger: 'blur' },
  kind: { required: true, message: '请选择类型', trigger: 'change' },
  mode: { required: true, message: '请选择模式', trigger: 'change' },
  authType: { required: true, message: '请选择鉴权类型', trigger: 'change' },
  url: {
    validator(_rule, value: string) {
      if (sourceForm.mode !== 'remote') return true;
      if (!value?.trim()) return new Error('remote 模式必须填写源地址');
      return true;
    },
    trigger: ['blur', 'input']
  },
  meta: {
    validator(_rule, value: string) {
      const raw = value?.trim();
      if (!raw) return true;
      try {
        JSON.parse(raw);
        return true;
      } catch {
        return new Error('meta 必须是合法 JSON');
      }
    },
    trigger: 'blur'
  }
};

const releaseQuery = reactive({
  kind: '' as '' | WafKind,
  status: '' as '' | WafReleaseStatus
});

const releaseLoading = ref(false);
const releaseTable = ref<WafReleaseItem[]>([]);
const releasePagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

const jobQuery = reactive({
  status: '' as '' | WafJobStatus,
  action: ''
});

const jobLoading = ref(false);
const jobTable = ref<WafJobItem[]>([]);
const jobPagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

const uploadModalVisible = ref(false);
const uploadSubmitting = ref(false);
const uploadFormRef = ref<FormInst | null>(null);
const uploadForm = reactive({
  kind: 'crs' as WafKind,
  version: '',
  checksum: '',
  activateNow: false,
  file: null as File | null
});

const uploadRules: FormRules = {
  kind: { required: true, message: '请选择规则类型', trigger: 'change' },
  version: { required: true, message: '请输入版本号', trigger: 'blur' },
  file: {
    validator() {
      if (!uploadForm.file) {
        return new Error('请选择待上传规则包');
      }
      return true;
    },
    trigger: 'change'
  }
};

const rollbackModalVisible = ref(false);
const rollbackSubmitting = ref(false);
const rollbackFormRef = ref<FormInst | null>(null);
const rollbackForm = reactive({
  target: 'last_good' as 'last_good' | 'version',
  version: ''
});

const rollbackRules: FormRules = {
  target: { required: true, message: '请选择回滚目标', trigger: 'change' },
  version: {
    validator() {
      if (rollbackForm.target === 'version' && !rollbackForm.version.trim()) {
        return new Error('指定版本回滚时必须填写版本号');
      }
      return true;
    },
    trigger: 'blur'
  }
};

const sourceColumns: DataTableColumns<WafSourceItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '名称', key: 'name', minWidth: 140 },
  {
    title: '类型',
    key: 'kind',
    width: 130,
    render(row) {
      return h(NTag, { type: row.kind === 'crs' ? 'success' : 'warning', bordered: false }, { default: () => row.kind });
    }
  },
  {
    title: '模式',
    key: 'mode',
    width: 110,
    render(row) {
      return h(NTag, { type: row.mode === 'remote' ? 'info' : 'default', bordered: false }, { default: () => row.mode });
    }
  },
  {
    title: '地址',
    key: 'url',
    minWidth: 260,
    ellipsis: { tooltip: true },
    render(row) {
      return row.url || '-';
    }
  },
  {
    title: '代理',
    key: 'proxyUrl',
    minWidth: 180,
    ellipsis: { tooltip: true },
    render(row) {
      return row.proxyUrl || '-';
    }
  },
  { title: '调度', key: 'schedule', width: 160, ellipsis: { tooltip: true }, render: row => row.schedule || '-' },
  {
    title: '开关',
    key: 'switches',
    minWidth: 200,
    render(row) {
      const labels = [
        row.enabled ? '启用' : '禁用',
        row.autoCheck ? '自动检查' : '手动检查',
        row.autoDownload ? '自动下载' : '手动下载',
        row.autoActivate ? '自动激活' : '手动激活'
      ];
      return h(
        NSpace,
        { size: 4, wrapItem: true },
        {
          default: () => labels.map(label => h(NTag, { size: 'small', bordered: false }, { default: () => label }))
        }
      );
    }
  },
  { title: '最近版本', key: 'lastRelease', width: 140, render: row => row.lastRelease || '-' },
  {
    title: '最近错误',
    key: 'lastError',
    minWidth: 220,
    ellipsis: { tooltip: true },
    render(row) {
      if (!row.lastError) return '-';
      return h(NTag, { type: 'error', bordered: false }, { default: () => row.lastError });
    }
  },
  { title: '更新时间', key: 'updatedAt', width: 180 },
  {
    title: '操作',
    key: 'action',
    width: 320,
    fixed: 'right',
    render(row) {
      return h(
        NSpace,
        { size: 4 },
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                secondary: true,
                onClick: () => handleCheckSource(row)
              },
              { default: () => '检查' }
            ),
            h(
              NButton,
              {
                size: 'small',
                type: 'primary',
                secondary: true,
                onClick: () => handleSyncSource(row, false)
              },
              { default: () => '同步' }
            ),
            h(
              NButton,
              {
                size: 'small',
                type: 'success',
                secondary: true,
                onClick: () => handleSyncSource(row, true)
              },
              { default: () => '同步并激活' }
            ),
            h(
              NButton,
              {
                size: 'small',
                onClick: () => handleEditSource(row)
              },
              { default: () => '编辑' }
            ),
            h(
              NPopconfirm,
              { onPositiveClick: () => handleDeleteSource(row) },
              {
                trigger: () => h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
                default: () => '删除后不可恢复，确认继续吗？'
              }
            )
          ]
        }
      );
    }
  }
];

const releaseColumns: DataTableColumns<WafReleaseItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '来源ID', key: 'sourceId', width: 90 },
  {
    title: '类型',
    key: 'kind',
    width: 120,
    render(row) {
      return h(NTag, { type: row.kind === 'crs' ? 'success' : 'warning', bordered: false }, { default: () => row.kind });
    }
  },
  { title: '版本', key: 'version', minWidth: 180, ellipsis: { tooltip: true } },
  { title: '包类型', key: 'artifactType', width: 110 },
  {
    title: '大小',
    key: 'sizeBytes',
    width: 120,
    render(row) {
      return formatBytes(row.sizeBytes);
    }
  },
  { title: '校验值', key: 'checksum', minWidth: 220, ellipsis: { tooltip: true }, render: row => row.checksum || '-' },
  {
    title: '状态',
    key: 'status',
    width: 120,
    render(row) {
      return h(NTag, { type: mapReleaseStatusType(row.status), bordered: false }, { default: () => row.status });
    }
  },
  {
    title: '路径',
    key: 'storagePath',
    minWidth: 260,
    ellipsis: { tooltip: true }
  },
  { title: '更新时间', key: 'updatedAt', width: 180 },
  {
    title: '操作',
    key: 'action',
    width: 180,
    fixed: 'right',
    render(row) {
      return h(
        NSpace,
        { size: 4 },
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                type: 'primary',
                secondary: true,
                disabled: row.status === 'active',
                onClick: () => handleActivateRelease(row)
              },
              { default: () => '激活' }
            )
          ]
        }
      );
    }
  }
];

const jobColumns: DataTableColumns<WafJobItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '来源 ID', key: 'sourceId', width: 90 },
  { title: '版本 ID', key: 'releaseId', width: 90 },
  { title: '动作', key: 'action', width: 120, render: row => mapJobActionLabel(row.action) },
  { title: '触发方式', key: 'triggerMode', width: 120, render: row => mapJobTriggerModeLabel(row.triggerMode) },
  {
    title: '状态',
    key: 'status',
    width: 110,
    render(row) {
      return h(NTag, { type: mapJobStatusType(row.status), bordered: false }, { default: () => mapJobStatusLabel(row.status) });
    }
  },
  { title: '操作人', key: 'operator', width: 120, render: row => row.operator || '-' },
  { title: '开始时间', key: 'startedAt', width: 180, render: row => row.startedAt || '-' },
  { title: '结束时间', key: 'finishedAt', width: 180, render: row => row.finishedAt || '-' },
  {
    title: '消息',
    key: 'message',
    minWidth: 320,
    ellipsis: { tooltip: true },
    render(row) {
      return mapJobMessage(row.message);
    }
  }
];

function mapReleaseStatusType(status: WafReleaseStatus) {
  switch (status) {
    case 'active':
      return 'success';
    case 'verified':
      return 'info';
    case 'failed':
      return 'error';
    case 'rolled_back':
      return 'warning';
    default:
      return 'default';
  }
}

function mapJobStatusType(status: WafJobStatus) {
  switch (status) {
    case 'success':
      return 'success';
    case 'failed':
      return 'error';
    default:
      return 'warning';
  }
}

function mapJobStatusLabel(status: string) {
  switch (status) {
    case 'running':
      return '执行中';
    case 'success':
      return '成功';
    case 'failed':
      return '失败';
    default:
      return status || '-';
  }
}

function mapJobActionLabel(action: string) {
  switch (action) {
    case 'check':
      return '检查';
    case 'download':
      return '下载';
    case 'verify':
      return '校验';
    case 'activate':
      return '激活';
    case 'rollback':
      return '回滚';
    case 'engine_check':
      return '引擎检查';
    default:
      return action || '-';
  }
}

function mapJobTriggerModeLabel(triggerMode: string) {
  switch (triggerMode) {
    case 'manual':
      return '手动';
    case 'upload':
      return '上传';
    case 'schedule':
      return '定时';
    case 'auto':
      return '自动';
    case 'system':
      return '系统';
    default:
      return triggerMode || '-';
  }
}

function mapJobMessage(rawMessage: string) {
  const messageText = String(rawMessage || '').trim();
  if (!messageText) {
    return '-';
  }

  const exactMap: Record<string, string> = {
    'check success': '检查成功',
    'sync success': '同步成功',
    'upload success': '上传成功',
    'activate success': '激活成功',
    'rollback success': '回滚成功',
    'engine source check success': '引擎源检查成功'
  };

  if (exactMap[messageText]) {
    return exactMap[messageText];
  }

  const replacementRules: Array<[RegExp, string]> = [
    [/context deadline exceeded/gi, '请求超时'],
    [/i\/o timeout/gi, '网络超时'],
    [/invalid proxy url:/gi, '代理地址不合法：'],
    [/invalid url:/gi, '无效地址：'],
    [/only https url is allowed/gi, '仅支持 HTTPS 地址'],
    [/only https scheme is allowed/gi, '仅允许 HTTPS 协议'],
    [/proxy url scheme must be http or https/gi, '代理地址协议仅支持 http/https'],
    [/source not found/gi, '未找到更新源'],
    [/source is disabled/gi, '更新源已禁用'],
    [/source mode is not remote/gi, '更新源模式不是 remote'],
    [/source url is empty/gi, '更新源地址为空'],
    [/move package failed:/gi, '移动安装包失败：'],
    [/create release dir failed:/gi, '创建版本目录失败：'],
    [/create release failed:/gi, '创建版本记录失败：'],
    [/fetch failed:/gi, '下载失败：'],
    [/host not allowed:/gi, '源域名不在允许列表：'],
    [/unexpected status code:/gi, '下载返回异常状态码：'],
    [/write temp file failed:/gi, '写入临时文件失败：'],
    [/close temp file failed:/gi, '关闭临时文件失败：'],
    [/move temp file failed:/gi, '移动临时文件失败：'],
    [/prepare waf store failed:/gi, '准备 Waf 存储目录失败：']
  ];

  let localizedMessage = messageText;
  for (const [pattern, replacement] of replacementRules) {
    localizedMessage = localizedMessage.replace(pattern, replacement);
  }

  return localizedMessage;
}

function formatBytes(size: number) {
  if (!size || size <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = size;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unitIndex]}`;
}

function displayEngineValue(value: unknown) {
  if (value === undefined || value === null || value === '') {
    return '-';
  }
  return String(value);
}

async function fetchEngineStatus() {
  if (engineUnavailable.value) {
    return;
  }

  engineLoading.value = true;
  try {
    const { data, error } = await fetchWafEngineStatus();
    if (!error && data) {
      engineStatus.value = data;
      engineUnavailable.value = false;
      return;
    }

    if (error) {
      const status = Number((error as any)?.response?.status || 0);
      if (status === 404 || status === 405) {
        engineUnavailable.value = true;
      }
    }
  } finally {
    engineLoading.value = false;
  }
}

function handleRefreshEngineStatus() {
  fetchEngineStatus();
}

async function handleCheckEngine() {
  if (engineUnavailable.value) {
    message.warning('后端接口尚未开放，当前仅展示占位状态');
    return;
  }

  engineChecking.value = true;
  try {
    const { error } = await checkWafEngine();
    if (!error) {
      message.success('引擎检查任务已提交');
      fetchEngineStatus();
      if (activeTab.value === 'job') {
        fetchJobs();
      }
      return;
    }

    const status = Number((error as any)?.response?.status || 0);
    if (status === 404 || status === 405) {
      engineUnavailable.value = true;
      message.warning('后端接口尚未开放，已切换占位模式');
      return;
    }
  } finally {
    engineChecking.value = false;
  }
}

async function fetchSources() {
  sourceLoading.value = true;
  try {
    const queryKind = sourceQuery.kind || routeKind.value;
    const normalizedKind = queryKind === 'crs' || queryKind === 'coraza_engine' ? queryKind : undefined;
    const { data, error } = await fetchWafSourceList({
      page: sourcePagination.page as number,
      pageSize: sourcePagination.pageSize as number,
      kind: normalizedKind,
      name: sourceQuery.name.trim() || undefined
    });
    if (!error && data) {
      const list = data.list || [];
      const total = data.total || 0;

      if (!sourceQuery.name.trim() && !normalizedKind && total > 0 && list.length === 0 && (sourcePagination.page as number) > 1) {
        sourcePagination.page = 1;
        await fetchSources();
        return;
      }

      sourceTable.value = list;
      sourcePagination.itemCount = total;
    }
  } finally {
    sourceLoading.value = false;
  }
}

function resetSourceQuery() {
  sourceQuery.name = '';
  sourceQuery.kind = '';
  sourcePagination.page = 1;
  fetchSources();
}

function handleSourcePageChange(page: number) {
  sourcePagination.page = page;
  fetchSources();
}

function handleSourcePageSizeChange(pageSize: number) {
  sourcePagination.pageSize = pageSize;
  sourcePagination.page = 1;
  fetchSources();
}

function resetSourceForm() {
  sourceForm.id = 0;
  sourceForm.name = '';
  sourceForm.kind = routeKind.value || 'crs';
  sourceForm.mode = 'remote';
  sourceForm.url = '';
  sourceForm.checksumUrl = '';
  sourceForm.proxyUrl = '';
  sourceForm.authType = 'none';
  sourceForm.authSecret = '';
  sourceForm.schedule = '';
  sourceForm.enabled = true;
  sourceForm.autoCheck = true;
  sourceForm.autoDownload = true;
  sourceForm.autoActivate = false;
  sourceForm.meta = '';
}

function handleAddSource() {
  sourceModalMode.value = 'add';
  resetSourceForm();
  applyDefaultSource('crs');
  sourceModalVisible.value = true;
}

function buildAvailableSourceName(baseName: string) {
  const normalized = baseName.trim();
  if (!normalized) return baseName;

  const names = new Set(sourceTable.value.map(item => item.name));
  if (!names.has(normalized)) {
    return normalized;
  }

  let index = 2;
  let candidate = `${normalized}-${index}`;
  while (names.has(candidate)) {
    index += 1;
    candidate = `${normalized}-${index}`;
  }
  return candidate;
}

function applyDefaultSource(kind: WafKind) {
  sourceForm.kind = kind;
  sourceForm.mode = 'remote';
  sourceForm.authType = 'none';
  sourceForm.authSecret = '';
  sourceForm.enabled = true;
  sourceForm.autoCheck = true;
  sourceForm.autoDownload = kind === 'crs';
  sourceForm.autoActivate = false;

  if (kind === 'crs') {
    sourceForm.name = buildAvailableSourceName('default-crs');
    sourceForm.url = 'https://codeload.github.com/coreruleset/coreruleset/tar.gz/refs/heads/main';
    sourceForm.checksumUrl = '';
    sourceForm.schedule = '0 0 */6 * * *';
    sourceForm.meta = '{"default":true,"official":true,"repo":"https://github.com/coreruleset/coreruleset"}';
    return;
  }

  sourceForm.name = buildAvailableSourceName('official-coraza-engine');
  sourceForm.url = 'https://codeload.github.com/corazawaf/coraza-caddy/tar.gz/refs/heads/main';
  sourceForm.checksumUrl = '';
  sourceForm.schedule = '0 0 0 * * *';
  sourceForm.meta = '{"official":true,"repo":"https://github.com/corazawaf/coraza-caddy"}';
}

function handleEditSource(row: WafSourceItem) {
  sourceModalMode.value = 'edit';
  sourceForm.id = row.id;
  sourceForm.name = row.name;
  sourceForm.kind = row.kind;
  sourceForm.mode = row.mode;
  sourceForm.url = row.url;
  sourceForm.checksumUrl = row.checksumUrl;
  sourceForm.proxyUrl = row.proxyUrl || '';
  sourceForm.authType = row.authType;
  sourceForm.authSecret = '';
  sourceForm.schedule = row.schedule;
  sourceForm.enabled = row.enabled;
  sourceForm.autoCheck = row.autoCheck;
  sourceForm.autoDownload = row.autoDownload;
  sourceForm.autoActivate = row.autoActivate;
  sourceForm.meta = '';
  sourceModalVisible.value = true;
}

async function handleSubmitSource() {
  await sourceFormRef.value?.validate();
  sourceSubmitting.value = true;
  try {
    const payload = {
      name: sourceForm.name.trim(),
      kind: sourceForm.kind,
      mode: sourceForm.mode,
      url: sourceForm.url.trim(),
      checksumUrl: sourceForm.checksumUrl.trim(),
      proxyUrl: sourceForm.proxyUrl.trim(),
      authType: sourceForm.authType,
      authSecret: sourceForm.authSecret.trim(),
      schedule: sourceForm.schedule.trim(),
      enabled: sourceForm.enabled,
      autoCheck: sourceForm.autoCheck,
      autoDownload: sourceForm.autoDownload,
      autoActivate: sourceForm.autoActivate,
      meta: sourceForm.meta.trim()
    };

    const request =
      sourceModalMode.value === 'add'
        ? createWafSource(payload)
        : updateWafSource(sourceForm.id, payload);

    const { error } = await request;
    if (!error) {
      message.success(sourceModalMode.value === 'add' ? '新增更新源成功' : '更新更新源成功');
      sourceModalVisible.value = false;
      fetchSources();
    }
  } finally {
    sourceSubmitting.value = false;
  }
}

function handleDeleteSource(row: WafSourceItem) {
  deleteWafSource(row.id).then(({ error }) => {
    if (!error) {
      message.success('删除成功');
      fetchSources();
    }
  });
}

function handleCheckSource(row: WafSourceItem) {
  checkWafSource(row.id).then(({ error }) => {
    if (!error) {
      message.success('检查任务已提交');
      fetchSources();
      if (activeTab.value === 'job') {
        fetchJobs();
      }
    }
  });
}

function handleSyncSource(row: WafSourceItem, activateNow: boolean) {
  const content = activateNow ? '将下载、校验并立即激活该源对应版本，确认继续？' : '将下载并校验该源对应版本，确认继续？';

  dialog.warning({
    title: activateNow ? '同步并激活确认' : '同步确认',
    content,
    positiveText: '确认',
    negativeText: '取消',
    async onPositiveClick() {
      const { error } = await syncWafSource(row.id, activateNow);
      if (!error) {
        message.success(activateNow ? '同步并激活成功' : '同步成功');
        fetchSources();
        fetchReleases();
        if (activeTab.value === 'job') {
          fetchJobs();
        }
      } else {
        const backendMsg = (error as any)?.response?.data?.msg;
        const rawMessage = String(backendMsg || error.message || '');
        if (rawMessage.includes('context deadline exceeded')) {
          message.error('同步超时：请配置代理后重试，或稍后再试');
        }
      }
    }
  });
}

async function fetchReleases() {
  releaseLoading.value = true;
  try {
    const queryKind = releaseQuery.kind || routeKind.value;
    const { data, error } = await fetchWafReleaseList({
      page: releasePagination.page as number,
      pageSize: releasePagination.pageSize as number,
      kind: queryKind,
      status: releaseQuery.status
    });
    if (!error && data) {
      releaseTable.value = data.list || [];
      releasePagination.itemCount = data.total || 0;
    }
  } finally {
    releaseLoading.value = false;
  }
}

function resetReleaseQuery() {
  releaseQuery.kind = '';
  releaseQuery.status = '';
  releasePagination.page = 1;
  fetchReleases();
}

function handleReleasePageChange(page: number) {
  releasePagination.page = page;
  fetchReleases();
}

function handleReleasePageSizeChange(pageSize: number) {
  releasePagination.pageSize = pageSize;
  releasePagination.page = 1;
  fetchReleases();
}

function handleActivateRelease(row: WafReleaseItem) {
  dialog.warning({
    title: '激活确认',
    content: `确认激活版本 ${row.version} 吗？`,
    positiveText: '确认',
    negativeText: '取消',
    async onPositiveClick() {
      const { error } = await activateWafRelease(row.id);
      if (!error) {
        message.success('激活已提交');
        fetchReleases();
        fetchJobs();
      }
    }
  });
}

function openRollbackModal() {
  rollbackForm.target = 'last_good';
  rollbackForm.version = '';
  rollbackModalVisible.value = true;
}

async function handleSubmitRollback() {
  await rollbackFormRef.value?.validate();
  rollbackSubmitting.value = true;
  try {
    const payload =
      rollbackForm.target === 'version'
        ? { target: 'version' as const, version: rollbackForm.version.trim() }
        : { target: 'last_good' as const };

    const { error } = await rollbackWafRelease(payload);
    if (!error) {
      message.success('回滚任务已提交');
      rollbackModalVisible.value = false;
      fetchReleases();
      fetchJobs();
    }
  } finally {
    rollbackSubmitting.value = false;
  }
}

function openUploadModal() {
  uploadForm.kind = routeKind.value || 'crs';
  uploadForm.version = '';
  uploadForm.checksum = '';
  uploadForm.activateNow = false;
  uploadForm.file = null;
  uploadModalVisible.value = true;
}

watch(
  routeKind,
  value => {
    if (value) {
      sourceForm.kind = value;
      uploadForm.kind = value;
    }
  },
  { immediate: true }
);

function handleBeforeUpload(data: { file: UploadFileInfo }) {
  const raw = data.file.file;
  if (!raw) return false;

  const name = raw.name.toLowerCase();
  if (!(name.endsWith('.zip') || name.endsWith('.tar.gz'))) {
    message.error('仅支持 .zip 或 .tar.gz 文件');
    return false;
  }

  uploadForm.file = raw;
  return false;
}

function handleRemoveUpload() {
  uploadForm.file = null;
  return true;
}

async function handleSubmitUpload() {
  await uploadFormRef.value?.validate();
  if (!uploadForm.file) {
    message.error('请先选择上传文件');
    return;
  }

  uploadSubmitting.value = true;
  try {
    const formData = new FormData();
    formData.append('kind', uploadForm.kind);
    formData.append('version', uploadForm.version.trim());
    if (uploadForm.checksum.trim()) {
      formData.append('checksum', uploadForm.checksum.trim());
    }
    formData.append('activateNow', String(uploadForm.activateNow));
    formData.append('file', uploadForm.file);

    const { error } = await uploadWafPackage(formData);
    if (!error) {
      message.success('上传成功，规则包已入库');
      uploadModalVisible.value = false;
      fetchReleases();
      fetchJobs();
    }
  } finally {
    uploadSubmitting.value = false;
  }
}

async function fetchJobs() {
  jobLoading.value = true;
  try {
    const { data, error } = await fetchWafJobList({
      page: jobPagination.page as number,
      pageSize: jobPagination.pageSize as number,
      status: jobQuery.status,
      action: jobQuery.action || undefined
    });
    if (!error && data) {
      jobTable.value = data.list || [];
      jobPagination.itemCount = data.total || 0;
    }
  } finally {
    jobLoading.value = false;
  }
}

function resetJobQuery() {
  jobQuery.status = '';
  jobQuery.action = '';
  jobPagination.page = 1;
  fetchJobs();
}

function handleJobPageChange(page: number) {
  jobPagination.page = page;
  fetchJobs();
}

function handleJobPageSizeChange(pageSize: number) {
  jobPagination.pageSize = pageSize;
  jobPagination.page = 1;
  fetchJobs();
}

function refreshCurrentTab() {
  if (activeTab.value === 'source') {
    fetchSources();
    return;
  }
  if (activeTab.value === 'release') {
    fetchReleases();
    return;
  }
  fetchJobs();
}

watch(activeTab, value => {
  if (value === 'source') {
    fetchSources();
  } else if (value === 'release') {
    fetchReleases();
  } else {
    fetchJobs();
  }
});

watch(routeKind, () => {
  sourceQuery.kind = '';
  releaseQuery.kind = '';
  sourcePagination.page = 1;
  releasePagination.page = 1;
  if (activeTab.value === 'source') {
    fetchSources();
  } else if (activeTab.value === 'release') {
    fetchReleases();
  }
});

onMounted(() => {
  fetchEngineStatus();
  fetchSources();
});
</script>

<style scoped>
:deep(.n-data-table .n-data-table-th__title) {
  white-space: nowrap;
}
</style>
