<script lang="ts" setup>
/**
 * 站点头像：优先用站点自带的 favicon，取不到回落首字母。
 *
 * 图标由后端代抓（/caddy/site-icon）：面板常以 http 暴露而站点是 https，
 * 浏览器直连会被混合内容拦下，入口块 CSP 也只放行 img-src self/data:。
 */
import { computed, ref, watch } from 'vue';

import { useQuery } from '@tanstack/vue-query';

import { getSiteIconApi } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';

const props = withDefaults(
  defineProps<{
    /** 站点主域名；为空时不发请求，直接显示字母 */
    host?: string;
    /** 站点名，字母头像取首字符 */
    name?: string;
    /** 头像边长（px） */
    size?: number;
  }>(),
  { size: 32, host: '', name: '' },
);

const failed = ref(false);

const host = computed(() => props.host.trim().toLowerCase());

// 同一个组件复用到别的站点时要重新给一次机会，否则失败标记会传染
watch(host, () => {
  failed.value = false;
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

const { data: icon } = useQuery({
  queryKey: computed(() => qk.caddy.siteIcon(host.value)),
  queryFn: () => getSiteIconApi(host.value, withListDetailErrorMode()),
  // 空 host 不发请求；host 变化自动按新 key 重取
  enabled: computed(() => host.value.length > 0),
  // 后端已按 host 缓存 6 小时，这里再留一段即可，避免切页重复拉
  staleTime: 30 * 60_000,
  retry: false,
});

const iconUrl = computed(() => (failed.value ? '' : (icon.value ?? '')));
</script>

<template>
  <img
    v-if="iconUrl"
    class="site-avatar image"
    :src="iconUrl"
    alt=""
    :style="boxStyle"
    @error="failed = true"
  />
  <span v-else class="site-avatar" :style="boxStyle">{{ letter }}</span>
</template>

<style scoped>
.site-avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  color: hsl(var(--primary));
  background: hsl(var(--primary) / 10%);
  border-radius: 9px;
}

.site-avatar.image {
  object-fit: contain;
  padding: 2px;
  background: hsl(var(--background));
  border: 1px solid hsl(var(--border));
}
</style>
