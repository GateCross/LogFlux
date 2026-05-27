<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { NAvatar, NButton, NEmpty, NList, NListItem, NPopover, NText, NThing, useMessage } from 'naive-ui';
import { useI18n } from 'vue-i18n';
import { getUnreadNotifications, readAllNotifications, readNotification } from '@/service/api/notification';

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
  const { error } = await readAllNotifications();
  if (!error) {
    list.value = [];
    message.success(t('common.success'));
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
      <div class="h-full w-40px flex-center cursor-pointer rounded-4px hover:bg-gray-100 dark:hover:bg-white/10">
        <NBadge :value="list.length" :max="99" :show="list.length > 0">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24">
            <path
              fill="currentColor"
              d="M12 22q-.825 0-1.412-.587T10 20h4q0 .825-.587 1.413T12 22m6-6v-5q0-3.075-1.9-5.462T11.5 3.05V3q-2.3.175-3.9 1.9T6 9v7H4v2h16v-2zm-2 0H8v-5q0-1.65 1.175-2.825T12 6t2.825 1.175T16 11z"
            />
          </svg>
        </NBadge>
      </div>
    </template>

    <div class="max-h-400px flex flex-col">
      <div class="flex items-center justify-between border-b p-3">
        <span class="font-bold">{{ t('page.notification.log.title') }}</span>
        <NButton v-if="list.length > 0" text type="primary" size="tiny" @click="handleReadAll">
          {{ t('common.confirm') }} (Mark All Read)
        </NButton>
      </div>

      <div class="flex-1 overflow-auto">
        <NList hoverable clickable>
          <NListItem v-for="item in list" :key="item.id" @click="handleRead(item.id)">
            <NThing :title="item.title" content-style="margin-top: 4px;">
              <template #description>
                <NText depth="3" class="text-xs">{{ item.createdAt }}</NText>
              </template>
              <div class="line-clamp-2 text-xs">{{ item.message }}</div>
            </NThing>
          </NListItem>
          <div v-if="list.length === 0" class="p-4 text-center">
            <NEmpty description="No unread notifications" size="small" />
          </div>
        </NList>
      </div>

      <div class="border-t p-2 text-center">
        <NButton text block @click="handleViewAll">View History</NButton>
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
