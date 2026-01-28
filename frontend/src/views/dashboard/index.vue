<script setup lang="ts">
import { computed } from 'vue';
import { useAppStore } from '@/store/modules/app';
import HeaderBanner from './modules/header-banner.vue';
import StatCard from './modules/stat-card.vue';
import TrendChart from './modules/trend-chart.vue';
import MapChart from './modules/map-chart.vue';
import { getStatCards, getErrorStats } from './data';

const appStore = useAppStore();

const gap = computed(() => (appStore.isMobile ? 0 : 16));
const statCards = getStatCards();
const errorStats = getErrorStats();
</script>

<template>
  <NSpace vertical :size="16">
    <HeaderBanner />

    <NGrid :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem v-for="item in statCards" :key="item.id" span="24 s:12 m:8 l:4">
        <StatCard :data="item" />
      </NGridItem>
    </NGrid>

    <NGrid :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem v-for="item in errorStats" :key="item.title" span="24 s:12 m:8">
        <NCard :border="false" class="rounded-2xl shadow-sm h-full">
           <div class="flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <span class="text-gray-500">{{ item.title }}</span>
              <div :class="['i-carbon:warning-filled', item.type === 'error' ? 'text-red-500' : 'text-orange-500']"></div>
            </div>
            <div class="flex items-end gap-2">
              <span class="text-2xl font-bold">{{ item.value }}</span>
              <span class="text-xs text-red-500 flex items-center bg-red-50 px-1 rounded">
                {{ item.rate }}
                <div class="i-carbon:arrow-up text-xs"></div>
              </span>
            </div>
           </div>
        </NCard>
      </NGridItem>
    </NGrid>

    <NGrid :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem span="24 l:16">
        <MapChart />
      </NGridItem>
      <NGridItem span="24 l:8">
         <NSpace vertical :size="16" class="h-full">
            <TrendChart />
            <NCard title="实时日志" class="flex-1 rounded-2xl shadow-sm">
               <div class="flex flex-col gap-3 text-xs">
                 <div v-for="i in 5" :key="i" class="flex justify-between items-center border-b border-gray-100 pb-2">
                   <div class="flex gap-2 items-center">
                     <span class="bg-green-100 text-green-600 px-1 rounded">GET</span>
                     <span class="text-gray-600">/api/v1/user/{{i}}</span>
                   </div>
                   <span class="text-gray-400">200ms</span>
                 </div>
               </div>
            </NCard>
         </NSpace>
      </NGridItem>
    </NGrid>
  </NSpace>
</template>

<style scoped></style>
