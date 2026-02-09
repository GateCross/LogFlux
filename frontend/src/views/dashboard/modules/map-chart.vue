<script setup lang="ts">
import { computed, watch } from 'vue';
import { useEcharts } from '@/hooks/common/echarts';
import type { ECOption } from '@/hooks/common/echarts';

interface GeoItem {
  name: string;
  value: number;
}

interface Props {
  data: GeoItem[];
}

const props = defineProps<Props>();

const { domRef, updateOptions } = useEcharts((): ECOption => ({
  tooltip: {
    show: true,
    formatter: '{b}: {c}'
  },
  visualMap: {
    min: 0,
    max: 200,
    left: 'left',
    top: 'bottom',
    text: ['高', '低'],
    calculable: true,
    inRange: {
      color: ['#e0f2fe', '#0ea5e9']
    }
  },
  series: []
}));

// Mock map registration or use empty one if asset missing
import * as echarts from 'echarts';
// Try to register a simple box for demo if real map json is missing in assets
if (!echarts.getMap('china')) {
  // Just a placeholder rectangle for visual check if no map data
   const geoJson = {
    "type": "FeatureCollection",
    "features": []
  };
  echarts.registerMap('china', geoJson as any);
}

const visualMax = computed(() => {
  const values = props.data.map(item => item.value);
  return Math.max(10, ...values);
});

const syncChart = () => {
  updateOptions(opts => {
    opts.visualMap = { ...opts.visualMap, max: visualMax.value };
    opts.series = [
      {
        name: '访问来源',
        type: 'map',
        map: 'china',
        roam: true,
        emphasis: {
          label: { show: true },
          itemStyle: { areaColor: '#38bdf8' }
        },
        data: props.data
      }
    ];
    return opts;
  });
};

watch(
  () => props.data,
  () => syncChart(),
  { immediate: true, deep: true }
);
</script>

<template>
  <NCard title="访问地理分布" class="h-full rounded-2xl border-none shadow-sm">
    <div ref="domRef" class="h-400px w-full"></div>
    <div class="absolute bottom-4 right-4 bg-white/80 p-4 rounded-xl backdrop-blur-sm border border-gray-100">
      <div class="text-sm font-bold mb-2">Top 区域</div>
      <div class="flex flex-col gap-2">
        <div v-for="item in data" :key="item.name" class="flex items-center justify-between gap-8">
          <span class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full bg-primary"></span>
            {{ item.name }}
          </span>
          <span class="font-bold">{{ item.value }}</span>
        </div>
        <div v-if="data.length === 0" class="text-xs text-gray-400">暂无数据</div>
      </div>
    </div>
  </NCard>
</template>
