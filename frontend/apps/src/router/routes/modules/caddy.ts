import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'mdi:server-network',
      order: 1,
      title: 'Caddy 配置',
    },
    name: 'Caddy',
    path: '/caddy',
    children: [
      {
        name: 'CaddyConfig',
        path: '/caddy/config',
        component: () => import('#/views/caddy/config/index.vue'),
        meta: {
          icon: 'mdi:cog-outline',
          title: '配置管理',
        },
      },
      {
        name: 'CaddyAccess',
        path: '/caddy/access',
        component: () => import('#/views/caddy/access/index.vue'),
        meta: {
          icon: 'mdi:shield-lock-outline',
          title: '访问控制',
        },
      },
      {
        name: 'CaddySource',
        path: '/caddy/source',
        component: () => import('#/views/caddy/source/index.vue'),
        meta: {
          hideInMenu: true,
          icon: 'mdi:source-branch',
          title: '来源管理',
        },
      },
      {
        name: 'CaddyLog',
        path: '/caddy/log',
        component: () => import('#/views/caddy/log/index.vue'),
        meta: {
          icon: 'mdi:text-box-outline',
          title: '访问日志',
        },
      },
      {
        name: 'CaddySystemLog',
        path: '/caddy/system-log',
        component: () => import('#/views/caddy/system-log/index.vue'),
        meta: {
          icon: 'mdi:file-document-outline',
          title: '系统日志',
        },
      },
    ],
  },
];

export default routes;
