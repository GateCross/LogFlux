/**
 * 访问地理分布地图（复刻旧 Vue frontend map-chart.vue）。
 * 支持国内/国际切换，使用 echarts choropleth map。
 */
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button, Space } from 'antd';
import * as echarts from 'echarts/core';
import { MapChart as MapChartImpl } from 'echarts/charts';
import {
  TooltipComponent,
  VisualMapComponent,
} from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { registerChinaMap, registerWorldMap } from '@/utils/map-register';

echarts.use([MapChartImpl, TooltipComponent, VisualMapComponent, CanvasRenderer]);

// 注册地图（idempotent）
registerChinaMap();
registerWorldMap();

interface GeoItem {
  name: string;
  value: number;
}

interface MapChartProps {
  chinaData: GeoItem[];
  worldData: GeoItem[];
}

// ─── 名称归一化映射（对齐旧版） ──────────────────────────────────────────────

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

function normalizeRegionName(name: string, mode: 'china' | 'world'): string {
  if (mode !== 'china') {
    return WORLD_COUNTRY_NAME_MAP[name] ?? name;
  }
  return CHINA_REGION_NAME_MAP[name] ?? name;
}

function aggregateData(raw: GeoItem[], mode: 'china' | 'world'): GeoItem[] {
  const regionValueMap = new Map<string, number>();
  raw.forEach((item) => {
    const name = normalizeRegionName(String(item.name), mode);
    const value = Number(item.value);
    regionValueMap.set(
      name,
      (regionValueMap.get(name) ?? 0) + (Number.isFinite(value) ? value : 0),
    );
  });
  return Array.from(regionValueMap, ([name, value]) => ({ name, value }));
}

// ─── 组件 ────────────────────────────────────────────────────────────────────

const MapChart: React.FC<MapChartProps> = ({ chinaData, worldData }) => {
  const [mode, setMode] = useState<'china' | 'world'>('china');
  const chartRef = useRef<HTMLDivElement>(null);
  const instanceRef = useRef<echarts.ECharts | null>(null);

  const activeData = useMemo(
    () => aggregateData(mode === 'china' ? chinaData : worldData, mode),
    [mode, chinaData, worldData],
  );

  const visualMax = useMemo(
    () => Math.max(10, ...activeData.map((item) => item.value)),
    [activeData],
  );

  const buildOption = useCallback(
    (data: GeoItem[], mapMode: 'china' | 'world', max: number) => ({
      tooltip: {
        show: true,
        formatter: (params: any) => {
          const item = Array.isArray(params) ? params[0] : params;
          const value = Number(item?.value);
          return `${item?.name ?? ''}: ${Number.isFinite(value) ? value : 0}`;
        },
      },
      visualMap: {
        min: 0,
        max,
        left: 'left',
        top: 'bottom',
        text: ['高', '低'],
        calculable: true,
        inRange: {
          color: ['#e0f2fe', '#0ea5e9'],
        },
      },
      series: [
        {
          name: '访问来源',
          type: 'map',
          map: mapMode,
          roam: true,
          emphasis: {
            label: { show: true },
            itemStyle: { areaColor: '#38bdf8' },
          },
          data,
        },
      ],
    }),
    [],
  );

  // 初始化图表
  useEffect(() => {
    if (!chartRef.current) return;

    const instance = echarts.init(chartRef.current);
    instanceRef.current = instance;
    instance.setOption({
      ...buildOption(activeData, mode, visualMax),
      backgroundColor: 'transparent',
    });

    const handleResize = () => instance.resize();
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      instance.dispose();
      instanceRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 数据变化时更新
  useEffect(() => {
    const instance = instanceRef.current;
    if (!instance) return;
    instance.setOption({
      visualMap: { max: visualMax },
      series: [{ data: activeData }],
    });
  }, [activeData, visualMax]);

  // 模式切换时 dispose + 重新 init
  useEffect(() => {
    if (!chartRef.current) return;
    const instance = instanceRef.current;
    if (instance) {
      instance.dispose();
    }
    const newInstance = echarts.init(chartRef.current);
    instanceRef.current = newInstance;
    newInstance.setOption({
      ...buildOption(activeData, mode, visualMax),
      backgroundColor: 'transparent',
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode]);

  return (
    <div style={{ position: 'relative' }}>
      <div style={{ marginBottom: 8, textAlign: 'right' }}>
        <Space size="small">
          <Button
            size="small"
            type={mode === 'china' ? 'primary' : 'default'}
            onClick={() => setMode('china')}
          >
            国内
          </Button>
          <Button
            size="small"
            type={mode === 'world' ? 'primary' : 'default'}
            onClick={() => setMode('world')}
          >
            国际
          </Button>
        </Space>
      </div>
      <div ref={chartRef} style={{ height: 400, width: '100%' }} />
      <div
        style={{
          position: 'absolute',
          bottom: 16,
          right: 16,
          border: '1px solid #f0f0f0',
          borderRadius: 12,
          background: 'rgba(255,255,255,0.8)',
          padding: 16,
          backdropFilter: 'blur(8px)',
          maxWidth: 200,
        }}
      >
        <div style={{ marginBottom: 8, fontWeight: 'bold', fontSize: 13 }}>
          Top 区域
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {activeData.length > 0 ? (
            activeData.slice(0, 10).map((item) => (
              <div
                key={item.name}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 16,
                }}
              >
                <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: '50%',
                      background: '#0ea5e9',
                    }}
                  />
                  {item.name}
                </span>
                <span style={{ fontWeight: 'bold' }}>{item.value}</span>
              </div>
            ))
          ) : (
            <span style={{ fontSize: 12, color: '#999' }}>暂无数据</span>
          )}
        </div>
      </div>
    </div>
  );
};

export default MapChart;
