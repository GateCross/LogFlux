import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      hideInMenu: true,
      icon: 'mdi:shield-check-outline',
      order: 2,
      title: '安全',
    },
    name: 'Security',
    path: '/security',
    children: [
      {
        name: 'SecuritySource',
        path: '/security/source',
        component: () => import('#/views/security/source/index.vue'),
        meta: { icon: 'mdi:source-branch', title: '来源管理', hideInMenu: true },
      },
      {
        name: 'SecurityPolicy',
        path: '/security/policy',
        component: () => import('#/views/security/policy/index.vue'),
        meta: { hideInMenu: true, icon: 'mdi:shield-check', title: '策略管理' },
      },
      {
        name: 'SecurityObserve',
        path: '/security/observe',
        component: () => import('#/views/security/observe/index.vue'),
        meta: { hideInMenu: true, icon: 'mdi:eye-outline', title: '观测日志' },
      },
      {
        name: 'SecurityOps',
        path: '/security/ops',
        component: () => import('#/views/security/ops/index.vue'),
        meta: { icon: 'mdi:wrench-outline', title: '运维操作', hideInMenu: true },
      },
      {
        name: 'SecurityRuntime',
        path: '/security/runtime',
        component: () => import('#/views/security/runtime/index.vue'),
        meta: { icon: 'mdi:clock-outline', title: '运行时', hideInMenu: true },
      },
      {
        name: 'SecurityCrs',
        path: '/security/crs',
        component: () => import('#/views/security/crs/index.vue'),
        meta: { icon: 'mdi:tune-vertical', title: 'CRS 调优', hideInMenu: true },
      },
      {
        name: 'SecurityExclusion',
        path: '/security/exclusion',
        component: () => import('#/views/security/exclusion/index.vue'),
        meta: { icon: 'mdi:cancel', title: '排除规则', hideInMenu: true },
      },
      {
        name: 'SecurityBinding',
        path: '/security/binding',
        component: () => import('#/views/security/binding/index.vue'),
        meta: { icon: 'mdi:link-variant', title: '绑定管理', hideInMenu: true },
      },
      {
        name: 'SecurityRelease',
        path: '/security/release',
        component: () => import('#/views/security/release/index.vue'),
        meta: { icon: 'mdi:rocket-launch-outline', title: '发布管理', hideInMenu: true },
      },
      {
        name: 'SecurityJob',
        path: '/security/job',
        component: () => import('#/views/security/job/index.vue'),
        meta: { icon: 'mdi:timer-outline', title: '任务管理', hideInMenu: true },
      },
    ],
  },
];

export default routes;
