<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { Button, Card } from 'ant-design-vue';
import * as echarts from 'echarts/core';
import type { MapSeriesOption } from 'echarts/charts';
import type { TooltipComponentOption } from 'echarts/components';
import { MapChart } from 'echarts/charts';
import { TooltipComponent, VisualMapComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { registerChinaMap, registerWorldMap } from '#/utils/map-register';

echarts.use([MapChart, TooltipComponent, VisualMapComponent, CanvasRenderer]);

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
  澳门: '澳门特别行政区',
};

const WORLD_COUNTRY_NAME_MAP: Record<string, string> = {
  'United States': '美国',
  'United States of America': '美国',
  'United Kingdom': '英国',
  'South Korea': '韩国',
  'North Korea': '朝鲜',
  Russia: '俄罗斯',
  Japan: '日本',
  Germany: '德国',
  France: '法国',
  Italy: '意大利',
  Spain: '西班牙',
  Portugal: '葡萄牙',
  Netherlands: '荷兰',
  Belgium: '比利时',
  Switzerland: '瑞士',
  Austria: '奥地利',
  Sweden: '瑞典',
  Norway: '挪威',
  Denmark: '丹麦',
  Finland: '芬兰',
  Poland: '波兰',
  'Czech Republic': '捷克',
  Czechia: '捷克',
  Hungary: '匈牙利',
  Greece: '希腊',
  Turkey: '土耳其',
  Australia: '澳大利亚',
  'New Zealand': '新西兰',
  Canada: '加拿大',
  Brazil: '巴西',
  Argentina: '阿根廷',
  Mexico: '墨西哥',
  India: '印度',
  Indonesia: '印度尼西亚',
  Thailand: '泰国',
  Vietnam: '越南',
  Philippines: '菲律宾',
  Malaysia: '马来西亚',
  Singapore: '新加坡',
  Egypt: '埃及',
  'South Africa': '南非',
  Nigeria: '尼日利亚',
  Kenya: '肯尼亚',
  'Saudi Arabia': '沙特阿拉伯',
  Iran: '伊朗',
  Iraq: '伊拉克',
  Israel: '以色列',
  Pakistan: '巴基斯坦',
  Bangladesh: '孟加拉国',
  'Sri Lanka': '斯里兰卡',
  Myanmar: '缅甸',
  Cambodia: '柬埔寨',
  Laos: '老挝',
  Mongolia: '蒙古',
  Korea: '韩国',
  Taiwan: '台湾',
  'Hong Kong': '香港',
  Macau: '澳门',
  Ireland: '爱尔兰',
  Iceland: '冰岛',
  Ukraine: '乌克兰',
  Belarus: '白俄罗斯',
  Romania: '罗马尼亚',
  Bulgaria: '保加利亚',
  Croatia: '克罗地亚',
  Serbia: '塞尔维亚',
  Colombia: '哥伦比亚',
  Chile: '智利',
  Peru: '秘鲁',
  Venezuela: '委内瑞拉',
  Cuba: '古巴',
  Jamaica: '牙买加',
};

const activeData = computed(() => {
  const raw = mode.value === 'china' ? props.chinaData : props.worldData;
  const regionValueMap = new Map<string, number>();

  raw.forEach((item) => {
    const name = normalizeRegionName(String(item.name));
    const value = Number(item.value);
    regionValueMap.set(
      name,
      (regionValueMap.get(name) ?? 0) + (Number.isFinite(value) ? value : 0),
    );
  });

  return Array.from(regionValueMap, ([name, value]) => ({ name, value }));
});

const visualMax = computed(() => {
  const values = activeData.value.map((item) => item.value);
  return Math.max(10, ...values);
});

function normalizeRegionName(name: string) {
  if (mode.value !== 'china') {
    return WORLD_COUNTRY_NAME_MAP[name] ?? name;
  }
  return CHINA_REGION_NAME_MAP[name] ?? name;
}

function formatTooltip(params: TooltipFormatterParams) {
  const item = Array.isArray(params) ? params[0] : params;
  const value = Number(item?.value);
  return `${item?.name ?? ''}: ${Number.isFinite(value) ? value : 0}`;
}

function buildMapSeries(): MapSeriesOption {
  return {
    name: '访问来源',
    type: 'map',
    map: mode.value,
    roam: true,
    emphasis: {
      label: { show: true },
      itemStyle: { areaColor: '#38bdf8' },
    },
    data: activeData.value,
  };
}

const chartRef = ref<HTMLElement | null>(null);
let chartInstance: echarts.ECharts | null = null;

function getFullOptions(): echarts.EChartsOption {
  return {
    tooltip: { show: true, formatter: formatTooltip },
    visualMap: {
      min: 0,
      max: visualMax.value,
      left: 'left',
      top: 'bottom',
      text: ['高', '低'],
      calculable: true,
      inRange: { color: ['#e0f2fe', '#0ea5e9'] },
    },
    series: [buildMapSeries()],
  };
}

function applyData() {
  if (!chartInstance) return;
  chartInstance.setOption({
    visualMap: { max: visualMax.value },
    series: [buildMapSeries()],
  });
}

watch(() => [props.chinaData, props.worldData], applyData, { deep: true });

watch(mode, () => {
  nextTick(() => {
    if (chartInstance) {
      chartInstance.dispose();
    }
    if (!chartRef.value) return;
    chartInstance = echarts.init(chartRef.value);
    chartInstance.setOption({
      ...getFullOptions(),
      backgroundColor: 'transparent',
    });
  });
});

onMounted(() => {
  if (chartRef.value) {
    chartInstance = echarts.init(chartRef.value);
    chartInstance.setOption({ ...getFullOptions(), backgroundColor: 'transparent' });
  }
});

onUnmounted(() => {
  chartInstance?.dispose();
  chartInstance = null;
});
</script>

<template>
  <Card class="rounded-2xl shadow-sm" :body-style="{ position: 'relative' }">
    <template #title>
      <div class="flex items-center justify-between">
        <span>访问地理分布</span>
        <div>
          <Button.Group size="small">
            <Button :type="mode === 'china' ? 'primary' : 'default'" @click="mode = 'china'">国内</Button>
            <Button :type="mode === 'world' ? 'primary' : 'default'" @click="mode = 'world'">国际</Button>
          </Button.Group>
        </div>
      </div>
    </template>
    <div ref="chartRef" class="h-[400px] w-full" />
    <div class="absolute bottom-4 right-4 rounded-xl border border-gray-100 bg-white/80 p-4 backdrop-blur-sm">
      <div class="mb-2 text-sm font-bold">Top 区域</div>
      <div class="flex flex-col gap-2">
        <div v-for="item in activeData" :key="item.name" class="flex items-center justify-between gap-8">
          <span class="flex items-center gap-2">
            <span class="h-2 w-2 rounded-full bg-blue-500" />
            {{ item.name }}
          </span>
          <span class="font-bold">{{ item.value }}</span>
        </div>
        <div v-if="activeData.length === 0" class="text-xs text-gray-400">暂无数据</div>
      </div>
    </div>
  </Card>
</template>
