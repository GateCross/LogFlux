import { computed, reactive, ref } from 'vue';
import type { FormInst, FormRules, PaginationProps } from 'naive-ui';
import {
  createWafSource,
  deleteWafSource,
  fetchWafSourceList,
  syncWafSource,
  updateWafSource,
  type WafAuthType,
  type WafKind,
  type WafMode,
  type WafSourceItem
} from '@/service/api/caddy-source';

type MessageApi = {
  success: (content: string) => void;
  error: (content: string) => void;
};

type DialogApi = {
  warning: (options: {
    title?: string;
    content?: string;
    positiveText?: string;
    negativeText?: string;
    onPositiveClick?: () => void | Promise<void>;
  }) => void;
};

interface UseWafSourceOptions {
  message: MessageApi;
  dialog: DialogApi;
  mergeJobSourceNameMap?: (list: WafSourceItem[]) => void;
  onSyncSuccess?: (activateNow: boolean) => void;
}

export function useWafSource(options: UseWafSourceOptions) {
  const { message, dialog, mergeJobSourceNameMap, onSyncSuccess } = options;

  const sourceQuery = reactive({
    name: ''
  });

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

  const sourceRules: FormRules = {
    name: { required: true, message: '请输入更新源名称', trigger: 'blur' },
    mode: { required: true, message: '请选择更新模式', trigger: 'change' },
    url: {
      validator() {
        if (sourceForm.mode !== 'remote') return true;
        if (!sourceForm.url.trim()) {
          return new Error('远程模式下请输入源地址');
        }
        return true;
      },
      trigger: ['blur', 'change']
    },
    schedule: {
      validator() {
        if (!sourceForm.enabled) return true;
        if (!sourceForm.schedule.trim()) {
          return new Error('启用状态下请填写 cron 表达式');
        }
        return true;
      },
      trigger: ['blur', 'change']
    }
  };

  async function fetchSources() {
    sourceLoading.value = true;
    try {
      const { data, error } = await fetchWafSourceList({
        page: sourcePagination.page as number,
        pageSize: sourcePagination.pageSize as number,
        kind: 'crs',
        name: sourceQuery.name.trim() || undefined
      });
      if (!error && data) {
        const list = data.list || [];
        const total = data.total || 0;

        if (!sourceQuery.name.trim() && total > 0 && list.length === 0 && (sourcePagination.page as number) > 1) {
          sourcePagination.page = 1;
          await fetchSources();
          return;
        }

        sourceTable.value = list;
        mergeJobSourceNameMap?.(list);
        sourcePagination.itemCount = total;
      }
    } finally {
      sourceLoading.value = false;
    }
  }

  function resetSourceQuery() {
    sourceQuery.name = '';
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
    sourceForm.kind = 'crs';
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

  function applyDefaultSource() {
    sourceForm.kind = 'crs';
    sourceForm.mode = 'remote';
    sourceForm.authType = 'none';
    sourceForm.authSecret = '';
    sourceForm.enabled = true;
    sourceForm.autoCheck = true;
    sourceForm.autoDownload = true;
    sourceForm.autoActivate = false;

    sourceForm.name = buildAvailableSourceName('default-crs');
    sourceForm.url = 'https://codeload.github.com/coreruleset/coreruleset/tar.gz/refs/heads/main';
    sourceForm.checksumUrl = '';
    sourceForm.proxyUrl = '';
    sourceForm.schedule = '0 0 */6 * * *';
    sourceForm.meta = '{"default":true,"official":true,"repo":"https://github.com/coreruleset/coreruleset"}';
  }

  function handleAddSource() {
    sourceModalMode.value = 'add';
    resetSourceForm();
    applyDefaultSource();
    sourceModalVisible.value = true;
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

      const request = sourceModalMode.value === 'add' ? createWafSource(payload) : updateWafSource(sourceForm.id, payload);
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
          onSyncSuccess?.(activateNow);
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

  return {
    sourceQuery,
    sourceLoading,
    sourceTable,
    sourcePagination,
    sourceModalVisible,
    sourceModalMode,
    sourceSubmitting,
    sourceFormRef,
    sourceForm,
    sourceModalTitle,
    sourceRules,
    fetchSources,
    resetSourceQuery,
    handleSourcePageChange,
    handleSourcePageSizeChange,
    handleAddSource,
    handleEditSource,
    handleSubmitSource,
    handleDeleteSource,
    handleSyncSource,
    applyDefaultSource
  };
}
