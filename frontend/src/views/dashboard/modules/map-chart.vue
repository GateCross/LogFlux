<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import * as echarts from 'echarts/core';
import type { MapSeriesOption } from 'echarts/charts';
import type { TooltipComponentOption } from 'echarts/components';
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

type TooltipFormatterParams = Parameters<
  Extract<NonNullable<TooltipComponentOption['formatter']>, (...args: any[]) => unknown>
>[0];

const props = defineProps<Props>();

const mode = ref<'china' | 'world'>('china');

const CHINA_REGION_NAME_MAP: Record<string, string> = {
  北京: '北京市',
  天津: '天津市',
  上海: '上海市',
  重庆: '重庆市',
  河北: '河北省',
  山西: '山西省',
  辽宁: '辽宁省',
  吉林: '吉林省',
  黑龙江: '黑龙江省',
  江苏: '江苏省',
  浙江: '浙江省',
  安徽: '安徽省',
  福建: '福建省',
  江西: '江西省',
  山东: '山东省',
  河南: '河南省',
  湖北: '湖北省',
  湖南: '湖南省',
  广东: '广东省',
  海南: '海南省',
  四川: '四川省',
  贵州: '贵州省',
  云南: '云南省',
  陕西: '陕西省',
  甘肃: '甘肃省',
  青海: '青海省',
  台湾: '台湾省',
  内蒙古: '内蒙古自治区',
  广西: '广西壮族自治区',
  西藏: '西藏自治区',
  宁夏: '宁夏回族自治区',
  新疆: '新疆维吾尔自治区',
  香港: '香港特别行政区',
  澳门: '澳门特别行政区'
};

const activeData = computed(() => {
  const raw = mode.value === 'china' ? props.chinaData : props.worldData;
  const regionValueMap = new Map<string, number>();

  raw.forEach(item => {
    const name = normalizeRegionName(String(item.name));
    const value = Number(item.value);
    regionValueMap.set(name, (regionValueMap.get(name) ?? 0) + (Number.isFinite(value) ? value : 0));
  });

  return Array.from(regionValueMap, ([name, value]) => ({ name, value }));
});

const { domRef, chart } = useEcharts(
  (): ECOption => ({
    tooltip: {
      show: true,
      formatter: formatTooltip
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

function normalizeRegionName(name: string) {
  if (mode.value !== 'china') {
    return name;
  }
  return CHINA_REGION_NAME_MAP[name] ?? name;
}

function formatTooltip(params: TooltipFormatterParams) {
  const item = Array.isArray(params) ? params[0] : params;
  const value = Number(item?.value);
  return `${item?.name ?? ''}: ${Number.isFinite(value) ? value : 0}`;
}

function getFullOptions(): ECOption {
  return {
    tooltip: { show: true, formatter: formatTooltip },
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
