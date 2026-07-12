<script lang="ts" setup>
import { computed, reactive, ref, watch } from 'vue';

import {
  Alert,
  Button,
  Checkbox,
  Empty,
  Input,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  message,
} from 'antdv-next';

import {
  DOCKER_DISCOVERY_LABEL_HELP,
  applyDiscoveryScanToSession,
  buildDraftsFromSelectedCandidates,
  clearCandidateSelection,
  createEmptyDiscoverySession,
  discoveryAutoLoadEnabled,
  selectAllValidCandidates,
  toggleCandidateSelection,
  validateDiscoverySelection,
  type DockerDiscoveryCandidate,
  type DockerDiscoverySessionState,
} from './docker-discovery-utils';
import type { QuickSiteDraft } from './quick-config-utils';

const props = defineProps<{
  open: boolean;
  scanning?: boolean;
  /** 父组件注入的最近扫描结果（会话级，不落库） */
  scanResult?: {
    list: DockerDiscoveryCandidate[];
    scannedAt: string;
    message: string;
  } | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  /** 请求后端扫描（只读，不 /load） */
  scan: [];
  /** 仅写入会话草稿，不调用 /load */
  'commit-drafts': [drafts: QuickSiteDraft[]];
  /** dry-run Preview：合并草稿后走既有 /adapt */
  preview: [drafts: QuickSiteDraft[]];
  /** 用户确认后走既有 Apply_Path（Preview 确认弹窗） */
  apply: [drafts: QuickSiteDraft[]];
}>();

const session = reactive<DockerDiscoverySessionState>(createEmptyDiscoverySession());
const showHelp = ref(false);
const filterText = ref('');

watch(
  () => props.open,
  (open) => {
    if (open && props.scanResult) {
      Object.assign(session, applyDiscoveryScanToSession(session, props.scanResult));
    }
  },
);

watch(
  () => props.scanResult,
  (result) => {
    if (!result) return;
    Object.assign(session, applyDiscoveryScanToSession(session, result));
  },
  { deep: true },
);

const filteredCandidates = computed(() => {
  const q = filterText.value.trim().toLowerCase();
  if (!q) return session.candidates;
  return session.candidates.filter((c) => {
    const blob = [
      c.name,
      c.containerName,
      c.upstream,
      ...(c.domains ?? []),
      c.candidateId,
    ]
      .join(' ')
      .toLowerCase();
    return blob.includes(q);
  });
});

const selectedCount = computed(() => session.selectedIds.length);
const validCount = computed(() => session.candidates.filter((c) => c.valid).length);

const columns = [
  { title: '', key: 'select', width: 48 },
  { title: '站点', key: 'name', dataIndex: 'name' },
  { title: '域名', key: 'domains' },
  { title: '上游', key: 'upstream', dataIndex: 'upstream' },
  { title: '容器', key: 'container' },
  { title: '状态', key: 'valid', width: 120 },
];

function close() {
  emit('update:open', false);
}

function isSelected(id: string) {
  return session.selectedIds.includes(id);
}

function onToggle(id: string, checked: boolean) {
  Object.assign(session, toggleCandidateSelection(session, id, checked));
}

function onSelectAllValid() {
  Object.assign(session, selectAllValidCandidates(session));
}

function onClearSelection() {
  Object.assign(session, clearCandidateSelection(session));
}

function ensureDrafts(): QuickSiteDraft[] | null {
  const errors = validateDiscoverySelection(session);
  if (errors.length) {
    message.warning(errors[0]);
    return null;
  }
  return buildDraftsFromSelectedCandidates(session);
}

function handleCommitDrafts() {
  const drafts = ensureDrafts();
  if (!drafts) return;
  emit('commit-drafts', drafts);
}

function handlePreview() {
  const drafts = ensureDrafts();
  if (!drafts) return;
  emit('preview', drafts);
}

function handleApply() {
  const drafts = ensureDrafts();
  if (!drafts) return;
  emit('apply', drafts);
}
</script>

<template>
  <Modal
    :open="open"
    title="Docker 发现 → 会话候选草稿"
    width="980px"
    :footer="null"
    destroy-on-hidden
    @cancel="close"
  >
    <div class="docker-discovery">
      <Alert
        class="mb-3"
        type="info"
        show-icon
        message="扫描容器 logflux.* 标签生成会话候选；不会写入平行 discovery 数据库，也不会自动热加载（/load）。路径：dry-run Preview → 您确认 → 既有 Apply_Path。"
      />

      <div class="toolbar mb-3">
        <Space wrap>
          <Button type="primary" :loading="scanning" @click="emit('scan')">
            扫描 Docker
          </Button>
          <Button size="small" @click="onSelectAllValid">全选有效</Button>
          <Button size="small" @click="onClearSelection">清空勾选</Button>
          <Button size="small" type="link" @click="showHelp = !showHelp">
            {{ showHelp ? '收起标签说明' : '标签约定' }}
          </Button>
          <span v-if="session.scannedAt" class="hint-text">
            最近扫描：{{ session.scannedAt }} · 有效 {{ validCount }} · 已选 {{ selectedCount }}
          </span>
        </Space>
        <Input
          v-model:value="filterText"
          allow-clear
          placeholder="过滤名称 / 域名 / 上游"
          class="filter-input"
        />
      </div>

      <Alert
        v-if="showHelp"
        class="mb-3"
        type="warning"
        show-icon
        message="Docker labels"
      >
        <template #description>
          <ul class="help-list">
            <li v-for="line in DOCKER_DISCOVERY_LABEL_HELP" :key="line">{{ line }}</li>
          </ul>
        </template>
      </Alert>

      <Alert
        v-if="session.message"
        class="mb-3"
        :type="session.candidates.length ? 'success' : 'warning'"
        show-icon
        :message="session.message"
      />

      <Spin :spinning="Boolean(scanning)">
        <div v-if="session.candidates.length === 0" class="empty-wrap">
          <Empty description="暂无会话候选。请先扫描，或确认容器已打 logflux.enable 标签且 Docker 可达。" />
        </div>
        <Table
          v-else
          size="small"
          row-key="candidateId"
          :columns="columns"
          :data-source="filteredCandidates"
          :pagination="false"
          :scroll="{ y: 360 }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'select'">
              <Checkbox
                :checked="isSelected(record.candidateId)"
                :disabled="!record.valid"
                @change="(e: any) => onToggle(record.candidateId, Boolean(e?.target?.checked))"
              />
            </template>
            <template v-else-if="column.key === 'domains'">
              <span>{{ (record.domains || []).join(', ') || '—' }}</span>
            </template>
            <template v-else-if="column.key === 'container'">
              <div class="container-cell">
                <span>{{ record.containerName || '—' }}</span>
                <span class="hint-text">{{ record.status }}</span>
              </div>
            </template>
            <template v-else-if="column.key === 'valid'">
              <Tag
                v-if="record.valid"
                color="success"
              >
                有效
              </Tag>
              <template v-else>
                <Tag color="error" :title="record.reason || '无效'">无效</Tag>
                <div v-if="record.reason" class="reason-text">{{ record.reason }}</div>
              </template>
            </template>
            <template v-else-if="column.key === 'name'">
              <div>
                <div>{{ record.name }}</div>
                <div class="hint-text">
                  TLS {{ record.tlsMode || 'auto' }}
                  · {{ record.lbPolicy || 'round_robin' }}
                  <template v-if="record.healthPath"> · health {{ record.healthPath }}</template>
                </div>
              </div>
            </template>
          </template>
        </Table>
      </Spin>

      <div class="footer-hint hint-text mt-2">
        自动热加载：{{ discoveryAutoLoadEnabled() ? '是' : '否（默认）' }} · 候选仅存浏览器会话
      </div>

      <div class="wizard-footer">
        <Space wrap>
          <Button type="primary" @click="handleCommitDrafts">写入会话草稿</Button>
          <Button :disabled="!selectedCount" @click="handlePreview">预览校验（/adapt）</Button>
          <Button type="primary" danger :disabled="!selectedCount" @click="handleApply">
            预览并应用…
          </Button>
          <Button @click="close">关闭</Button>
        </Space>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.docker-discovery {
  min-height: 320px;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
}

.filter-input {
  max-width: 260px;
}

.help-list {
  margin: 0;
  padding-left: 18px;
}

.container-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.reason-text {
  margin-top: 2px;
  color: #b42318;
  font-size: 12px;
  max-width: 160px;
}

.hint-text {
  color: #667085;
  font-size: 12px;
}

.empty-wrap {
  padding: 24px 0;
}

.wizard-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
  margin-top: 12px;
  border-top: 1px solid #eef0f4;
}

.mb-3 {
  margin-bottom: 12px;
}

.mt-2 {
  margin-top: 8px;
}
</style>
