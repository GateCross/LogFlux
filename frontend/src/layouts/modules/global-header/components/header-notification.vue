<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { NPopover, NList, NListItem, NThing, NAvatar, NText, NButton, NEmpty, useMessage } from 'naive-ui';
import { getUnreadNotifications, readNotification } from '@/service/api/notification';

defineOptions({
  name: 'HeaderNotification'
});

const { t } = useI18n();
const router = useRouter();
const message = useMessage();

const loading = ref(false);
const show = ref(false);
const list = ref<any[]>([]);

async function fetchUnread() {
  loading.value = true;
  try {
    const { data, error } = await getUnreadNotifications();
    if (!error && data) {
      list.value = data.list || [];
    }
  } finally {
    loading.value = false;
  }
}

async function handleRead(id: number) {
  const { error } = await readNotification(id);
  if (!error) {
    list.value = list.value.filter(item => item.id !== id);
  }
}

async function handleReadAll() {
  // TODO: Add mark all read API
  for (const item of list.value) {
    await handleRead(item.id);
  }
}

function handleViewAll() {
  show.value = false;
  router.push('/notification/log');
}

onMounted(() => {
  fetchUnread();
  // Poll every minute
  setInterval(fetchUnread, 60000);
});
</script>

<template>
  <NPopover v-model:show="show" trigger="click" placement="bottom-end" :width="320">
    <template #trigger>
      <div class="flex-center h-full w-40px cursor-pointer hover:bg-gray-100 dark:hover:bg-white/10 rounded-4px">
        <n-badge :value="list.length" :max="99" :show="list.length > 0">
           <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24"><path fill="currentColor" d="M12 22q-.825 0-1.412-.587T10 20h4q0 .825-.587 1.413T12 22m6-6v-5q0-3.075-1.9-5.462T11.5 3.05V3q-2.3.175-3.9 1.9T6 9v7H4v2h16v-2zm-2 0H8v-5q0-1.65 1.175-2.825T12 6t2.825 1.175T16 11z"/></svg>
        </n-badge>
      </div>
    </template>
    
    <div class="flex flex-col max-h-400px">
      <div class="flex justify-between items-center p-3 border-b">
         <span class="font-bold">{{ t('page.notification.log.title') }}</span>
         <NButton text type="primary" size="tiny" @click="handleReadAll" v-if="list.length > 0">
           {{ t('common.confirm') }} (Mark All Read)
         </NButton>
      </div>
      
      <div class="overflow-auto flex-1">
        <NList hoverable clickable>
           <NListItem v-for="item in list" :key="item.id" @click="handleRead(item.id)">
              <NThing :title="item.title" content-style="margin-top: 4px;">
                 <template #description>
                    <NText depth="3" class="text-xs">{{ item.createdAt }}</NText>
                 </template>
                 <div class="text-xs line-clamp-2">{{ item.message }}</div>
              </NThing>
           </NListItem>
           <div v-if="list.length === 0" class="p-4 text-center">
              <NEmpty description="No unread notifications" size="small" />
           </div>
        </NList>
      </div>

      <div class="p-2 border-t text-center">
         <NButton text block @click="handleViewAll">
            View History
         </NButton>
      </div>
    </div>
  </NPopover>
</template>

<style scoped>
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
