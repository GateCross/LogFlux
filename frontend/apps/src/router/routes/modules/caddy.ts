import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'mdi:server-network',
      order: 2,
      title: '服务',
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
        name: 'CaddyCatalog',
        path: '/caddy/catalog',
        component: () => import('#/views/caddy/catalog/index.vue'),
        meta: {
          icon: 'mdi:view-grid-outline',
          title: '服务目录',
        },
      },
      {
        name: 'CaddyCertificates',
        path: '/caddy/certificates',
        component: () => import('#/views/caddy/certificates/index.vue'),
        meta: {
          icon: 'mdi:certificate-outline',
          title: '证书中心',
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
          icon: 'mdi:source-branch',
          title: 'WAF 来源',
        },
      },
    ],
  },
];

export default routes;
