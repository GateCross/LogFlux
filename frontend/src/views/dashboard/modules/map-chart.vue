<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import * as echarts from 'echarts/core';
import type { MapSeriesOption } from 'echarts/charts';
import { useEcharts } from '@/hooks/common/echarts';
import type { ECOption } from '@/hooks/common/echarts';
import { registerChinaMap, registerWorldMap } from '@/utils/map-register';

registerChinaMap();
registerWorldMap();

interface GeoItem {
  name: string;
  value: number;
}

interface Props {
  chinaData: GeoItem[];
  worldData: GeoItem[];
}

const props = defineProps<Props>();

const mode = ref<'china' | 'world'>('china');

const activeData = computed(() => {
  const raw = mode.value === 'china' ? props.chinaData : props.worldData;
  return raw.map(item => ({ name: String(item.name), value: Number(item.value) || 0 }));
});

const { domRef, chart } = useEcharts(
  (): ECOption => ({
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
    series: [buildMapSeries()]
  }),
  {
    onUpdated(instance) {
      instance.hideLoading();
      // 数据到达时 chart 可能已渲染但 series 为空，补刷一次
      if (activeData.value.length > 0) {
        applyData(instance);
      }
    }
  }
);

const visualMax = computed(() => {
  const values = activeData.value.map(item => item.value);
  return Math.max(10, ...values);
});

function applyData(instance?: echarts.ECharts | null) {
  const target = instance ?? chart.value;
  if (!target) return;
  target.setOption({
    visualMap: { max: visualMax.value },
    series: [buildMapSeries()]
  });
}

function buildMapSeries(): MapSeriesOption {
  return {
    name: '访问来源',
    type: 'map',
    map: mode.value,
    roam: true,
    emphasis: {
      label: { show: true },
      itemStyle: { areaColor: '#38bdf8' }
    },
    data: activeData.value
  };
}

function getFullOptions(): ECOption {
  return {
    tooltip: { show: true, formatter: '{b}: {c}' },
    visualMap: {
      min: 0,
      max: visualMax.value,
      left: 'left',
      top: 'bottom',
      text: ['高', '低'],
      calculable: true,
      inRange: { color: ['#e0f2fe', '#0ea5e9'] }
    },
    series: [buildMapSeries()]
  };
}

// 数据变化时更新（chart 已存在则增量更新，否则等 onUpdated 补刷）
watch(
  () => [props.chinaData, props.worldData],
  () => applyData(),
  { deep: true }
);

// 地图类型切换：销毁旧实例，重新初始化
watch(mode, () => {
  nextTick(() => {
    if (chart.value) {
      chart.value.dispose();
    }
    if (!domRef.value) return;
    chart.value = echarts.init(domRef.value);
    chart.value.setOption({ ...getFullOptions(), backgroundColor: 'transparent' });
  });
});
</script>

<template>
  <NCard title="访问地理分布" class="h-full rounded-2xl border-none shadow-sm">
    <template #header-extra>
      <NButtonGroup size="tiny">
        <NButton :type="mode === 'china' ? 'primary' : 'default'" @click="mode = 'china'"> 国内 </NButton>
        <NButton :type="mode === 'world' ? 'primary' : 'default'" @click="mode = 'world'"> 国际 </NButton>
      </NButtonGroup>
    </template>
    <div ref="domRef" class="h-400px w-full"></div>
    <div class="absolute bottom-4 right-4 border border-gray-100 rounded-xl bg-white/80 p-4 backdrop-blur-sm">
      <div class="mb-2 text-sm font-bold">Top 区域</div>
      <div class="flex flex-col gap-2">
        <div v-for="item in activeData" :key="item.name" class="flex items-center justify-between gap-8">
          <span class="flex items-center gap-2">
            <span class="h-2 w-2 rounded-full bg-primary"></span>
            {{ item.name }}
          </span>
          <span class="font-bold">{{ item.value }}</span>
        </div>
        <div v-if="activeData.length === 0" class="text-xs text-gray-400">暂无数据</div>
      </div>
    </div>
  </NCard>
</template>
