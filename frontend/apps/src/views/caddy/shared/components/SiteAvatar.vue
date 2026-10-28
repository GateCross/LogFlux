<script lang="ts" setup>
/**
 * 站点头像：优先显示本地已缓存的 favicon，没有就显示首字母。
 *
 * 抓取时机只有两处，渲染本身从不发请求：
 *   1. 建站成功时抓一次（SiteCreateModal）
 *   2. 用户点击头像时手动抓
 * 这样进目录页不会对每个站点发一轮外网请求，站点再多也不会拖慢首屏。
 *
 * 图标由后端代抓（/caddy/site-icon）：面板常以 http 暴露而站点是 https，
 * 浏览器直连会被混合内容拦下，入口块 CSP 也只放行 img-src self/data:。
 */
import { computed, ref, watch } from 'vue';

import { Tooltip } from 'antdv-next';

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

/** 只读本地缓存；没有就是没有，不在这里发请求 */
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
  if (iconUrl.value && !failed.value) return '点击重新抓取站点图标';
  return '点击抓取站点图标';
});

/**
 * 手动抓取：渲染不发请求，只有这里会主动拉一次。
 * 建站那次没抓到（DNS 未生效、站点刚起）时，用户点一下就能补上。
 */
async function handleClick() {
  if (!props.clickable || !host.value) return;
  if (fetching.value) return;

  fetching.value = true;
  failed.value = false;
  try {
    const icon = await fetchAndStoreSiteIcon(host.value);
    if (icon) {
      iconUrl.value = icon;
    }
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
