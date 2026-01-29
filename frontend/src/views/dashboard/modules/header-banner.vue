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
  value: string;
}

const statisticData: StatisticData[] = [
  {
    id: 0,
    label: '项目数',
    value: '25'
  },
  {
    id: 1,
    label: '待办',
    value: '4/16'
  },
  {
    id: 2,
    label: '消息',
    value: '12'
  }
];
</script>

<template>
  <NCard :bordered="false" class="card-wrapper">
    <NGrid :x-gap="gap" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem span="24 s:24 m:18">
        <div class="flex-y-center">
          <div class="shrink-0 w-72px h-72px rd-50% flex-center bg-primary:10">
            <div class="i-carbon:user-avatar text-40px text-primary"></div>
          </div>
          <div class="pl-12px">
            <h3 class="text-18px font-semibold">
              早安，{{ authStore.userInfo.username }}，今天又是充满活力的一天！
            </h3>
            <p class="leading-30px text-[#999]">今日多云转晴，20℃ - 25℃！</p>
          </div>
        </div>
      </NGridItem>
      <NGridItem span="24 s:24 m:6">
        <NSpace :size="24" justify="end">
          <div v-for="item in statisticData" :key="item.id" class="flex flex-col items-center">
            <span class="text-[#999]">{{ item.label }}</span>
            <span class="text-20px">{{ item.value }}</span>
          </div>
        </NSpace>
      </NGridItem>
    </NGrid>
  </NCard>
</template>

<style scoped></style>
