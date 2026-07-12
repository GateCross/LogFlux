/**
 * 离线注册 Iconify 图标集。
 * 使用 @iconify-json/* 本地数据，运行时不再请求 api.unisvg.com / api.iconify.design。
 */
import { icons as antDesign } from '@iconify-json/ant-design';
import { icons as carbon } from '@iconify-json/carbon';
import { icons as ep } from '@iconify-json/ep';
import { icons as fluentMdl2 } from '@iconify-json/fluent-mdl2';
import { icons as ic } from '@iconify-json/ic';
import { icons as lucide } from '@iconify-json/lucide';
import { icons as materialSymbols } from '@iconify-json/material-symbols';
import { icons as mdi } from '@iconify-json/mdi';
import { addCollection, registerIconNames } from '@vben-core/icons';

let loaded = false;

function registerCollection(collection: Parameters<typeof addCollection>[0]) {
  addCollection(collection);
  const prefix = collection.prefix;
  const names = Object.keys(collection.icons ?? {});
  if (collection.aliases) {
    names.push(...Object.keys(collection.aliases));
  }
  registerIconNames(prefix, names);
}

/** 注册项目使用到的全部离线图标集（幂等） */
export function setupIconifyOffline() {
  if (loaded) {
    return;
  }
  loaded = true;

  // 后端菜单 + 前端路由/组件会用到的图标前缀
  registerCollection(antDesign);
  registerCollection(carbon);
  registerCollection(ep);
  registerCollection(fluentMdl2);
  registerCollection(ic);
  registerCollection(lucide);
  registerCollection(materialSymbols);
  registerCollection(mdi);
}
