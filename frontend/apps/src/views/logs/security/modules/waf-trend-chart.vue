<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import { Card, Tag } from 'antdv-next';
import * as echarts from 'echarts/core';
import { LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';

echarts.use([LineChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer]);

interface Props {
  times: string[];
  totals: number[];
  blockeds: number[];
}

const props = defineProps<Props>();

const chartRef = ref<HTMLElement | null>(null);
let chartInstance: echarts.ECharts | null = null;

function buildOption(): echarts.EChartsCoreOption {
  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'line' },
    },
    legend: {
      data: ['总命中', '拦截'],
      top: 0,
      right: 0,
      icon: 'roundRect',
      itemWidth: 12,
      itemHeight: 4,
      textStyle: { color: 'rgba(128, 128, 128, 0.95)' },
    },
    grid: {
      left: '2%',
      right: '2%',
      top: '14%',
      bottom: '2%',
      // ECharts 6: containLabel 已弃用，等价迁移到 outerBounds
      outerBoundsMode: 'same',
      outerBoundsContain: 'axisLabel',
    },
    xAxis: {
      type: 'category',
      data: props.times,
      boundaryGap: false,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: 'rgba(128, 128, 128, 0.9)' },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: 'rgba(128, 128, 128, 0.9)' },
      splitLine: {
        show: true,
        lineStyle: { type: 'dashed', color: 'rgba(128, 128, 128, 0.25)' },
      },
    },
    series: [
      {
        name: '总命中',
        type: 'line',
        smooth: true,
        showSymbol: false,
        itemStyle: { color: '#f59e0b' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(245, 158, 11, 0.35)' },
              { offset: 1, color: 'rgba(245, 158, 11, 0.05)' },
            ],
          },
        },
        data: props.totals,
      },
      {
        name: '拦截',
        type: 'line',
        smooth: true,
        showSymbol: false,
        itemStyle: { color: '#ef4444' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(239, 68, 68, 0.35)' },
              { offset: 1, color: 'rgba(239, 68, 68, 0.05)' },
            ],
          },
        },
        data: props.blockeds,
      },
    ],
  };
}

function syncChart() {
  if (!chartInstance) return;
  chartInstance.setOption(buildOption());
}

watch(() => [props.times, props.totals, props.blockeds], syncChart, {
  deep: true,
});

onMounted(() => {
  if (chartRef.value) {
    chartInstance = echarts.init(chartRef.value);
    chartInstance.setOption(buildOption());
  }
});

onUnmounted(() => {
  chartInstance?.dispose();
  chartInstance = null;
});
</script>

<template>
  <Card class="h-full rounded-2xl shadow-sm">
    <template #title>
      <div class="flex items-center gap-2">
        <span>WAF 命中趋势</span>
        <Tag color="red">Security</Tag>
      </div>
    </template>
    <div ref="chartRef" class="h-[360px]" />
  </Card>
</template>
