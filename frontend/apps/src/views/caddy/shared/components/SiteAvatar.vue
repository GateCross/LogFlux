<script lang="ts" setup>
/**
 * 站点头像：优先显示本地已缓存的 favicon，没有就显示首字母。
 *
 * 抓取时机只有两处，渲染本身从不发请求：
 *   1. 建站成功时抓一次（SiteCreateModal）
 *   2. 用户点击头像时手动抓
 * 这样进目录页不会对每个站点发一轮请求，站点再多也不会拖慢首屏。
 *
 * 图标由后端代抓（/caddy/site-icon）：前端只传 host，后端自动反查该站点
 * 配置的 reverse_proxy 上游并直连探测，既避免前端受混合内容与 CSP 限制，
 * 也避免在 URL Query 中明文传递内网 IP 触发 WAF 的 SSRF（403）拦截。
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
    /** 兼容上游字段（内部统一按 host 索引） */
    upstream?: string;
  }>(),
  { size: 32, host: '', upstream: '', name: '', clickable: true },
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

.site-avatar-img {
  width: 100%;
  height: 100%;
  border-radius: inherit;
  object-fit: contain;
}

.site-avatar-letter {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  color: #fff;
  font-weight: 600;
  border-radius: inherit;
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
}

.site-avatar-spinner {
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: rgb(0 0 0 / 25%);
  animation: site-avatar-pulse 1.2s ease-in-out infinite;
}

@keyframes site-avatar-pulse {
  0%,
  100% {
    opacity: 0.3;
  }
  50% {
    opacity: 0.7;
  }
}
</style>
