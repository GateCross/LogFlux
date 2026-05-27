import * as echarts from 'echarts/core';
import chinaJson from 'echarts-china-map/lib/china.json';

let registered = false;

export function registerChinaMap() {
  if (registered) return;
  if (!echarts.getMap('china')) {
    echarts.registerMap('china', chinaJson as any);
  }
  registered = true;
}
