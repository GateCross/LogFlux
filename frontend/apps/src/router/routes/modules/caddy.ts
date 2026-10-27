import type { RouteRecordRaw } from 'vue-router';

/**
 * 站点域路由（原「服务」分组收敛）：
 * 6 项收敛为 5 个一级入口——站点 / 证书中心 / 访问控制 / 全局设置，
 * 配置管理隐藏（路由保留可直达，入口收进全局设置「高级配置」链接），
 * WAF 来源并入全局设置卡片后删除路由项；
 * 安全事件由日志组提升为一级（见 logs.ts）。
 * 旧 /caddy 分组深链重定向到站点页。
 */
const routes: RouteRecordRaw[] = [
  {
    /** 旧「服务」分组入口：仅做深链兼容重定向，不出现在菜单 */
    meta: {
      hideInMenu: true,
      title: '站点',
    },
    name: 'Caddy',
    path: '/caddy',
    redirect: '/caddy/catalog',
  },
  {
    name: 'CaddyCatalog',
    path: '/caddy/catalog',
    component: () => import('#/views/caddy/catalog/index.vue'),
    meta: {
      icon: 'mdi:view-grid-outline',
      order: 2,
      title: '站点',
    },
  },
  {
    /** 站点详情整页（对齐雷池，替代原抽屉）：query 携带 host，从站点卡片进入 */
    name: 'CaddySiteDetail',
    path: '/caddy/catalog/site',
    component: () => import('#/views/caddy/sites/SiteDetail.vue'),
    meta: {
      hideInMenu: true,
      title: '站点详情',
    },
  },
  {
    /** 配置管理：菜单隐藏，路由保留（全局设置「高级配置」链接与服务目录深链直达） */
    name: 'CaddyConfig',
    path: '/caddy/config',
    component: () => import('#/views/caddy/config/index.vue'),
    meta: {
      hideInMenu: true,
      icon: 'mdi:cog-outline',
      title: '配置管理',
    },
  },
  {
    name: 'CaddyCertificates',
    path: '/caddy/certificates',
    component: () => import('#/views/caddy/certificates/index.vue'),
    meta: {
      icon: 'mdi:certificate-outline',
      order: 4,
      title: '证书中心',
    },
  },
  {
    name: 'CaddyAccess',
    path: '/caddy/access',
    component: () => import('#/views/caddy/access/index.vue'),
    meta: {
      icon: 'mdi:shield-lock-outline',
      order: 5,
      title: '访问控制',
    },
  },
  {
    name: 'CaddySettings',
    path: '/caddy/settings',
    component: () => import('#/views/caddy/settings/index.vue'),
    meta: {
      icon: 'mdi:tune-vertical',
      order: 6,
      title: '全局设置',
    },
  },
];

export default routes;
