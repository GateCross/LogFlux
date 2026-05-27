<script setup lang="ts">
import { computed } from 'vue';
import { useAppStore } from '@/store/modules/app';
import { useAuthStore } from '@/store/modules/auth';

const appStore = useAppStore();
const authStore = useAuthStore();

const gap = computed(() => (appStore.isMobile ? 0 : 16));

interface StatisticData {
  id: number;
  label: string;
  value: string | number;
}

interface Props {
  rangeText: string;
  stats: StatisticData[];
}

const props = withDefaults(defineProps<Props>(), {
  rangeText: '',
  stats: () => []
});
</script>

<template>
  <NCard :bordered="false" class="card-wrapper">
    <NGrid :x-gap="gap" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem span="24 s:24 m:18">
        <div class="flex-y-center">
          <div class="h-72px w-72px flex-center shrink-0 rd-50% bg-primary:10">
            <div class="i-carbon:user-avatar text-40px text-primary"></div>
          </div>
          <div class="pl-12px">
            <h3 class="text-18px font-semibold">欢迎回来，{{ authStore.userInfo.username }} 👋</h3>
            <p class="text-[#999] leading-30px">统计范围：{{ props.rangeText }}</p>
          </div>
        </div>
      </NGridItem>
      <NGridItem span="24 s:24 m:6">
        <NSpace :size="24" justify="end">
          <div v-for="item in props.stats" :key="item.id" class="flex flex-col items-center">
            <span class="text-[#999]">{{ item.label }}</span>
            <span class="text-20px">{{ item.value }}</span>
          </div>
        </NSpace>
      </NGridItem>
    </NGrid>
  </NCard>
</template>

<style scoped></style>
