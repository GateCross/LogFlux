/**
 * 实时 QPS 趋势图（复刻旧 Vue frontend trend-chart.vue）。
 * 使用 echarts-for-react，配置对齐旧版。
 */
import React, { useMemo } from 'react';
import ReactEChartsCore from 'echarts-for-react/lib/core';
import * as echarts from 'echarts/core';
import { LineChart } from 'echarts/charts';
import {
  GridComponent,
  TooltipComponent,
} from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';

echarts.use([LineChart, GridComponent, TooltipComponent, CanvasRenderer]);

interface TrendChartProps {
  times: string[];
  values: number[];
}

const TrendChart: React.FC<TrendChartProps> = ({ times, values }) => {
  const option = useMemo(
    () => ({
      tooltip: {
        trigger: 'axis' as const,
        axisPointer: { type: 'line' as const },
      },
      grid: {
        left: '2%',
        right: '2%',
        top: '10%',
        bottom: '2%',
        containLabel: true,
      },
      xAxis: {
        type: 'category' as const,
        data: times,
        boundaryGap: false,
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { show: false },
      },
      yAxis: {
        type: 'value' as const,
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: {
          show: true,
          lineStyle: { type: 'dashed' as const, color: '#eeeeee' },
        },
      },
      series: [
        {
          name: 'QPS',
          type: 'line' as const,
          smooth: true,
          showSymbol: false,
          itemStyle: { color: '#06b6d4' },
          areaStyle: {
            color: {
              type: 'linear' as const,
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
          data: values,
        },
      ],
    }),
    [times, values],
  );

  return (
    <ReactEChartsCore
      echarts={echarts}
      option={option}
      style={{ height: 300, width: '100%' }}
      notMerge
      lazyUpdate
    />
  );
};

export default TrendChart;
