<script lang="ts" setup>
/**
 * 站点头像：优先从本地存储读取已缓存的 favicon，未获取时回落字母。
 * 遵循用户要求：建站时自动获取一次，平时不自动请求；用户点击头像可手动获取/刷新。
 */
import { computed, ref, watch } from 'vue';

import { message, Tooltip } from 'antdv-next';

import {
  fetchAndStoreSiteIcon,
  getStoredSiteIcon,
} from './site-icon-storage';

const props = withDefaults(
  defineProps<{
    /** 是否允许手动点击抓取，默认 true */
    clickable?: boolean;
    /** 站点主域名 */
    host?: string;
    /** 站点名，字母头像取首字符 */
    name?: string;
    /** 头像边长（px） */
    size?: number;
  }>(),
  { size: 32, host: '', name: '', clickable: true },
);

const host = computed(() => props.host.trim().toLowerCase());
const fetching = ref(false);
const failed = ref(false);

// 初始化时仅从本地存储读取（建站时或此前手动抓取已存入）
const iconUrl = ref(getStoredSiteIcon(host.value));

watch(host, (next) => {
  failed.value = false;
  iconUrl.value = getStoredSiteIcon(next);
});

const letter = computed(() => {
  const source = (props.name || props.host || '?').trim();
  return (source[0] ?? '?').toUpperCase();
});

const boxStyle = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`,
  fontSize: `${Math.round(props.size * 0.45)}px`,
}));

const tooltipText = computed(() => {
  if (!props.clickable) return '';
  if (fetching.value) return '正在抓取站点图标...';
  if (iconUrl.value && !failed.value) return '点击刷新站点图标';
  return '点击获取站点图标';
});

async function handleClick() {
  if (!props.clickable) return;
  if (!host.value) {
    message.warning('站点未配置有效域名，无法获取图标');
    return;
  }
  if (fetching.value) return;

  fetching.value = true;
  failed.value = false;
  try {
    const icon = await fetchAndStoreSiteIcon(host.value);
    if (icon) {
      iconUrl.value = icon;
      message.success('站点图标已获取');
    } else {
      message.info('未获取到该站点的图标');
    }
  } catch {
    message.error('获取站点图标失败');
  } finally {
    fetching.value = false;
  }
}
</script>

<template>
  <Tooltip :title="tooltipText" :mouse-enter-delay="0.3">
    <div
      class="site-avatar-wrap"
      :class="{ clickable: props.clickable, loading: fetching }"
      :style="boxStyle"
      @click.stop="handleClick"
    >
      <img
        v-if="iconUrl && !failed"
        class="site-avatar-img"
        :src="iconUrl"
        alt=""
        @error="failed = true"
      />
      <span v-else class="site-avatar-letter">{{ letter }}</span>
      <div v-if="fetching" class="site-avatar-spinner"></div>
    </div>
  </Tooltip>
</template>

<style scoped>
.site-avatar-wrap {
  position: relative;
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  user-select: none;
}

.site-avatar-wrap.clickable {
  cursor: pointer;
}

.site-avatar-wrap.clickable:hover {
  filter: brightness(0.95);
  box-shadow: 0 0 0 2px hsl(var(--primary) / 30%);
}

.site-avatar-letter {
  display: flex;
  width: 100%;
  height: 100%;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  color: hsl(var(--primary));
  background: hsl(var(--primary) / 10%);
  border-radius: 9px;
}

.site-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  padding: 2px;
  background: hsl(var(--background));
  border: 1px solid hsl(var(--border));
  border-radius: 9px;
}

.site-avatar-spinner {
  position: absolute;
  inset: 0;
  border-radius: 9px;
  background: hsl(var(--background) / 60%);
  display: flex;
  align-items: center;
  justify-content: center;
}

.site-avatar-spinner::after {
  content: '';
  width: 16px;
  height: 16px;
  border: 2px solid hsl(var(--primary) / 20%);
  border-top-color: hsl(var(--primary));
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
