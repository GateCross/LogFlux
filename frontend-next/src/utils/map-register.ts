/**
 * ECharts 地图注册工具。
 * 复刻旧 Vue frontend 的 map-register.ts，保持 idempotent。
 */
import * as echarts from 'echarts/core';
import chinaJson from 'echarts-china-map/lib/china.json';
import worldJson from '@surbowl/world-geo-json-zh/world.zh.json';

let chinaRegistered = false;
let worldRegistered = false;

export function registerChinaMap() {
  if (chinaRegistered) return;
  if (!echarts.getMap('china')) {
    echarts.registerMap('china', chinaJson as any);
  }
  chinaRegistered = true;
}

export function registerWorldMap() {
  if (worldRegistered) return;
  if (!echarts.getMap('world')) {
    echarts.registerMap('world', worldJson as any);
  }
  worldRegistered = true;
}
