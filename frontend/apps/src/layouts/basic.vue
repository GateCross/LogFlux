<script lang="ts" setup>
import type { NotificationItem } from '@vben/layouts';

import type { NotificationApi } from '#/api/notification';

import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';

import { AuthenticationLoginExpiredModal } from '@vben/common-ui';
import { VBEN_DOC_URL, VBEN_GITHUB_URL } from '@vben/constants';
import { useWatermark } from '@vben/hooks';
import { BookOpenText, CircleHelp, SvgGithubIcon } from '@vben/icons';
import {
  BasicLayout,
  LockScreen,
  Notification,
  UserDropdown,
} from '@vben/layouts';
import { preferences, usePreferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { openWindow } from '@vben/utils';

import {
  getUnreadNotificationsApi,
  markAllNotificationsReadApi,
  markNotificationReadApi,
} from '#/api/notification';
import { $t } from '#/locales';
import { useAuthStore } from '#/store';
import LoginForm from '#/views/_core/authentication/login.vue';

const NOTIFICATION_POLL_INTERVAL = 60_000;

const notifications = ref<NotificationItem[]>([]);
let notificationTimer: number | undefined;

const router = useRouter();
const userStore = useUserStore();
const authStore = useAuthStore();
const accessStore = useAccessStore();
const { destroyWatermark, updateWatermark } = useWatermark();
const { isDark } = usePreferences();
const showDot = computed(() =>
  notifications.value.some((item) => !item.isRead),
);

function toNotificationItem(
  item: NotificationApi.UnreadNotification,
): NotificationItem {
  const message = item.message || item.content || '暂无通知内容';
  const title = item.title || item.eventType || item.type || '系统通知';

  return {
    ...item,
    avatar: preferences.logo.source || preferences.app.defaultAvatar,
    date: item.createdAt || item.sentAt || '',
    isRead: item.read ?? false,
    link: '/notification/log',
    message,
    title,
  };
}

async function loadUnreadNotifications() {
  try {
    const list = await getUnreadNotificationsApi();
    notifications.value = list.map((item) => toNotificationItem(item));
  } catch {
    notifications.value = [];
  }
}

const menus = computed(() => [
  {
    handler: () => {
      router.push({ name: 'Profile' });
    },
    icon: 'lucide:user',
    text: $t('page.auth.profile'),
  },
  {
    handler: () => {
      openWindow(VBEN_DOC_URL, {
        target: '_blank',
      });
    },
    icon: BookOpenText,
    text: $t('ui.widgets.document'),
  },
  {
    handler: () => {
      openWindow(VBEN_GITHUB_URL, {
        target: '_blank',
      });
    },
    icon: SvgGithubIcon,
    text: 'GitHub',
  },
  {
    handler: () => {
      openWindow(`${VBEN_GITHUB_URL}/issues`, {
        target: '_blank',
      });
    },
    icon: CircleHelp,
    text: $t('ui.widgets.qa'),
  },
]);

const avatar = computed(() => {
  return userStore.userInfo?.avatar || '';
});

async function handleLogout() {
  await authStore.logout(false);
}

async function handleNoticeClear() {
  const previous = notifications.value;
  notifications.value = [];

  try {
    await markAllNotificationsReadApi();
  } catch {
    notifications.value = previous;
  }
}

async function markRead(id: number | string) {
  const item = notifications.value.find((item) => item.id === id);
  if (!item || item.isRead) {
    return;
  }

  item.isRead = true;
  try {
    await markNotificationReadApi(String(id));
  } catch {
    item.isRead = false;
  }
}

function remove(id: number | string) {
  notifications.value = notifications.value.filter((item) => item.id !== id);
}

async function handleMakeAll() {
  const unreadItems = notifications.value.filter((item) => !item.isRead);
  if (unreadItems.length === 0) {
    return;
  }

  unreadItems.forEach((item) => (item.isRead = true));
  try {
    await markAllNotificationsReadApi();
  } catch {
    unreadItems.forEach((item) => (item.isRead = false));
  }
}

const viewAll = () => {
  router.push({ name: 'NotificationLog' });
};

const handleClick = async (item: NotificationItem) => {
  if (item.id) {
    await markRead(item.id);
  }

  // 如果通知项有链接，点击时跳转
  if (item.link) {
    navigateTo(item.link, item.query, item.state);
  }
};

function navigateTo(
  link: string,
  query?: Record<string, any>,
  state?: Record<string, any>,
) {
  if (link.startsWith('http://') || link.startsWith('https://')) {
    // 外部链接，在新标签页打开
    window.open(link, '_blank');
  } else {
    // 内部路由链接，支持 query 参数和 state
    router.push({
      path: link,
      query: query || {},
      state,
    });
  }
}

watch(
  () => ({
    enable: preferences.app.watermark,
    content: preferences.app.watermarkContent,
    isDark: isDark.value,
  }),
  async ({ enable, content, isDark: isDarkValue }) => {
    if (enable) {
      const watermarkColor = isDarkValue
        ? 'rgba(255, 255, 255, 0.12)'
        : 'rgba(0, 0, 0, 0.12)';

      await updateWatermark({
        advancedStyle: {
          colorStops: [
            {
              color: watermarkColor,
              offset: 0,
            },
            {
              color: watermarkColor,
              offset: 1,
            },
          ],
          type: 'linear',
        },
        content:
          content ||
          `${userStore.userInfo?.username} - ${userStore.userInfo?.realName}`,
      });
    } else {
      destroyWatermark();
    }
  },
  {
    immediate: true,
  },
);

onMounted(() => {
  void loadUnreadNotifications();
  notificationTimer = window.setInterval(() => {
    void loadUnreadNotifications();
  }, NOTIFICATION_POLL_INTERVAL);
});

onUnmounted(() => {
  if (notificationTimer) {
    window.clearInterval(notificationTimer);
  }
});
</script>

<template>
  <BasicLayout @clear-preferences-and-logout="handleLogout">
    <template #user-dropdown>
      <UserDropdown
        :avatar
        :menus
        :text="userStore.userInfo?.realName"
        :description="userStore.userInfo?.username"
        tag-text="LogFlux"
        @logout="handleLogout"
        @clear-preferences-and-logout="handleLogout"
      />
    </template>
    <template #notification>
      <Notification
        :dot="showDot"
        :notifications="notifications"
        @clear="handleNoticeClear"
        @read="(item) => item.id && markRead(item.id)"
        @remove="(item) => item.id && remove(item.id)"
        @make-all="handleMakeAll"
        @on-click="handleClick"
        @view-all="viewAll"
      />
    </template>
    <template #extra>
      <AuthenticationLoginExpiredModal
        v-model:open="accessStore.loginExpired"
        :avatar
      >
        <LoginForm />
      </AuthenticationLoginExpiredModal>
    </template>
    <template #lock-screen>
      <LockScreen :avatar @to-login="handleLogout" />
    </template>
  </BasicLayout>
</template>
