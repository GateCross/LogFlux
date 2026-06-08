<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import { Card, Tag } from 'ant-design-vue';
import * as echarts from 'echarts/core';
import { LineChart } from 'echarts/charts';
import { GridComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';

echarts.use([LineChart, GridComponent, TooltipComponent, CanvasRenderer]);

interface Props {
  times: string[];
  values: number[];
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
    grid: {
      left: '2%',
      right: '2%',
      top: '10%',
      bottom: '2%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: props.times,
      boundaryGap: false,
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: {
        show: true,
        lineStyle: { type: 'dashed', color: '#eeeeee' },
      },
    },
    series: [
      {
        name: 'QPS',
        type: 'line',
        smooth: true,
        showSymbol: false,
        itemStyle: { color: '#06b6d4' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(6, 182, 212, 0.4)' },
              { offset: 1, color: 'rgba(6, 182, 212, 0.05)' },
            ],
          },
        },
        data: props.values,
      },
    ],
  };
}

function syncChart() {
  if (!chartInstance) return;
  chartInstance.setOption(buildOption());
}

watch(() => [props.times, props.values], syncChart, { deep: true });

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
        <span>实时 QPS 趋势</span>
        <Tag color="blue">Live</Tag>
      </div>
    </template>
    <div ref="chartRef" class="h-[400px]" />
  </Card>
</template>
