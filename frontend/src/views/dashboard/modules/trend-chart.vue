<script setup lang="ts">
import { ref, computed } from 'vue';
import { useEcharts } from '@/hooks/common/echarts';
import { getTrendData } from '../data';

const { domRef, updateOptions } = useEcharts(() => ({
  tooltip: {
    trigger: 'axis',
    axisPointer: { type: 'line' }
  },
  grid: {
    left: '2%',
    right: '2%',
    top: '10%',
    bottom: '2%',
    containLabel: true
  },
  xAxis: {
    type: 'category',
    data: [],
    boundaryGap: false,
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: { show: false }
  },
  yAxis: {
    type: 'value',
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: { 
      show: true, 
      lineStyle: { type: 'dashed', color: '#eeeeee' } 
    }
  },
  series: []
}));

const data = getTrendData();

updateOptions(opts => {
  opts.xAxis = { ...opts.xAxis, data: data.times };
  opts.series = [
    {
      name: 'QPS',
      type: 'line',
      smooth: true,
      showSymbol: false,
      itemStyle: { color: '#06b6d4' },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(6, 182, 212, 0.4)' },
            { offset: 1, color: 'rgba(6, 182, 212, 0.05)' }
          ]
        }
      },
      data: data.qps
    }
  ];
  return opts;
});
</script>

<template>
  <NCard title="实时 QPS 趋势" class="h-full rounded-2xl border-none shadow-sm">
    <template #header-extra>
      <div class="flex items-center gap-2">
        <NTag size="small" type="primary" round>Live</NTag>
      </div>
    </template>
    <div ref="domRef" class="h-300px"></div>
  </NCard>
</template>
