<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { useMessage, useDialog } from 'naive-ui';
import { VueMonacoEditor, loader } from '@guolao/vue-monaco-editor';
import { fetchCaddyServers, fetchCaddyConfig, updateCaddyConfig, addCaddyServer, updateCaddyServer, deleteCaddyServer } from '@/service/api/caddy';

// Configure Monaco Editor loader to use npmmirror for better performance in China
loader.config({
  paths: {
    vs: 'https://registry.npmmirror.com/monaco-editor/0.44.0/files/min/vs',
  },
});

// Defines
interface CaddyServer { 
  id: number;
  name: string;
  url: string;
  type: string;
  token?: string;
}

const message = useMessage();
const dialog = useDialog();
const loading = ref(false);
const saving = ref(false);
const servers = ref<CaddyServer[]>([]);
const currentServerId = ref<number | null>(null);
const mode = ref<'preview' | 'edit'>('preview');
const configContent = ref('');

// Server Management Modal
const showServerModal = ref(false);
const serverModalType = ref<'add' | 'edit'>('add');
const serverFormModel = ref<Omit<CaddyServer, 'id'> & { id?: number }>({
  name: '',
  url: '',
  type: 'local',
  token: ''
});

// Computed
const isPreview = computed(() => mode.value === 'preview');
const serverOptions = computed(() => 
  servers.value.map(s => ({ label: s.name, value: s.id }))
);

// Methods
async function getServers() {
  const { data, error } = await fetchCaddyServers();
  if (error) {
    message.error('获取服务器列表失败');
    return;
  }
  if (data?.list) {
    servers.value = data.list;
    // 自动选择第一个
    if (servers.value.length > 0) {
      if (!currentServerId.value || !servers.value.find(s => s.id === currentServerId.value)) {
        currentServerId.value = servers.value[0].id;
      }
    } else {
      currentServerId.value = null;
      configContent.value = '';
    }
  }
}

async function getConfig() {
  if (!currentServerId.value) return;
  
  loading.value = true;
  const { data, error } = await fetchCaddyConfig(currentServerId.value);
  loading.value = false;

  if (error) {
    message.error('获取配置失败');
    return;
  }
  if (data) {
     // Backend now returns Caddyfile from DB (or default text)
     // No need to parse JSON.
     configContent.value = data.config || '';
  }
}

async function handleSaveConfig() {
  if (!currentServerId.value) return;

  saving.value = true;
  const { error } = await updateCaddyConfig(currentServerId.value, configContent.value);
  saving.value = false;

  if (error) {
    message.error('保存配置失败');
    return;
  }
  message.success('配置已保存');
  mode.value = 'preview';
}

// Server Management Methods
function openAddServerModal() {
  serverModalType.value = 'add';
  serverFormModel.value = { name: '', url: 'http://localhost:2019', type: 'local', token: '' };
  showServerModal.value = true;
}

function openEditServerModal() {
  const server = servers.value.find(s => s.id === currentServerId.value);
  if (!server) return;
  serverModalType.value = 'edit';
  serverFormModel.value = { ...server };
  showServerModal.value = true;
}

async function handleDeleteServer() {
  if (!currentServerId.value) return;
  
  dialog.warning({
    title: '确认删除',
    content: '确定要删除此服务器吗？',
    positiveText: '确认',
    negativeText: '取消',
    onPositiveClick: async () => {
      const { error } = await deleteCaddyServer(currentServerId.value!);
      if (error) {
        message.error('删除服务器失败');
        return;
      }
      message.success('服务器已删除');
      await getServers();
    }
  });
}

async function handleSaveServer() {
  let error;
  if (serverModalType.value === 'add') {
    const res = await addCaddyServer(serverFormModel.value);
    error = res.error;
  } else {
    const res = await updateCaddyServer(serverFormModel.value);
    error = res.error;
  }

  if (error) {
    message.error('保存服务器失败');
    return;
  }
  message.success(serverModalType.value === 'add' ? '添加成功' : '更新成功');
  showServerModal.value = false;
  await getServers();
}

// Watchers
watch(currentServerId, () => {
  if (currentServerId.value) {
    getConfig();
  }
});

onMounted(() => {
  getServers();
});
</script>

<template>
  <div class="h-full overflow-hidden flex flex-col">
    <NCard 
      title="Caddy配置" 
      class="h-full card-wrapper" 
      :content-style="{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }"
    >
      <template #header-extra>
        <div class="flex items-center gap-2">
           <NSelect
              v-model:value="currentServerId"
              :options="serverOptions"
              placeholder="选择服务器"
              class="w-48"
              size="small"
           />
           <div class="flex items-center gap-2">
             <NButton size="small" @click="openAddServerModal">
               <div class="flex items-center gap-1">
                 <span class="i-carbon-add" />
                 <span>新增</span>
               </div>
             </NButton>
             <NButton size="small" :disabled="!currentServerId" @click="openEditServerModal">
               <div class="flex items-center gap-1">
                 <span class="i-carbon-edit" />
                 <span>编辑</span>
               </div>
             </NButton>
             <NButton size="small" :disabled="!currentServerId" @click="handleDeleteServer" type="error" ghost>
               <div class="flex items-center gap-1">
                 <span class="i-carbon-trash-can" />
                 <span>删除</span>
               </div>
             </NButton>
           </div>
           
           <div class="w-1px h-4 bg-gray-200 mx-2"></div>

           <NRadioGroup v-model:value="mode" size="small">
             <NRadioButton value="preview">预览</NRadioButton>
             <NRadioButton value="edit">编辑</NRadioButton>
           </NRadioGroup>
           <NButton 
             v-if="!isPreview" 
             type="primary" 
             size="small" 
             :loading="saving"
             :disabled="!currentServerId"
             @click="handleSaveConfig"
           >
             保存配置
           </NButton>
        </div>
      </template>
      
      <div class="flex-1 min-h-0 flex flex-col gap-4">
        <div v-if="servers.length === 0" class="flex flex-col items-center justify-center p-8 text-gray-400 h-full">
           <div class="text-lg">未找到 Caddy 服务器</div>
           <div class="text-sm mt-2">请点击上方“+”按钮添加服务器</div>
        </div>
        
        <NSpin :show="loading" class="h-full" content-class="h-full" v-else>
           <div class="h-full relative">
              <VueMonacoEditor
                v-model:value="configContent"
                language="shell"
                theme="vs"
                :options="{
                  automaticLayout: true,
                  fixedOverflowWidgets: true,
                  readOnly: isPreview,
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  wordWrap: 'on'
                }"
                class="absolute inset-0"
              />
           </div>
        </NSpin>
      </div>
    </NCard>

    <!-- Server Management Modal -->
    <!-- ... modal content ... -->
    <NModal v-model:show="showServerModal" preset="card" :title="serverModalType === 'add' ? '添加服务器' : '编辑服务器'" class="w-500px">
      <!-- ... form content unrelated to layout ... -->
      <NForm label-placement="left" label-width="80">
        <NFormItem label="名称" path="name">
          <NInput v-model:value="serverFormModel.name" placeholder="服务器名称" />
        </NFormItem>
        <NFormItem label="地址" path="url">
          <NInput v-model:value="serverFormModel.url" placeholder="http://localhost:2019" />
        </NFormItem>
        <NFormItem label="类型" path="type">
          <NRadioGroup v-model:value="serverFormModel.type">
            <NRadioButton value="local">本地</NRadioButton>
            <NRadioButton value="remote">远程</NRadioButton>
          </NRadioGroup>
        </NFormItem>
        <NFormItem label="凭证" path="token" v-if="serverFormModel.type === 'remote'">
          <NInput v-model:value="serverFormModel.token" placeholder="可选认证凭证" />
        </NFormItem>
        <div class="flex justify-end gap-2">
          <NButton @click="showServerModal = false">取消</NButton>
          <NButton type="primary" @click="handleSaveServer">保存</NButton>
        </div>
      </NForm>
    </NModal>
  </div>
</template>

<style scoped>
:deep(.n-card__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
}
:deep(.n-spin-content) {
  height: 100%;
}
:deep(.n-input),
:deep(.n-input-wrapper),
:deep(.n-input__textarea),
:deep(.n-input__textarea-el) {
  height: 100% !important;
}

/* Ensure Monaco widgets (like search) are on top */
:deep(.monaco-editor-overlay) {
  z-index: 1000 !important;
}
</style>
