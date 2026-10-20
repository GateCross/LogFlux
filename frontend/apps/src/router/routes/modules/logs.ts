import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'mdi:text-box-search-outline',
      order: 3,
      title: '日志',
    },
    name: 'Logs',
    path: '/logs',
    children: [
      {
        name: 'LogsAccess',
        path: '/logs/access',
        component: () => import('#/views/logs/access/index.vue'),
        meta: {
          icon: 'mdi:text-box-outline',
          title: '访问日志',
        },
      },
      {
        name: 'LogsSystem',
        path: '/logs/system',
        component: () => import('#/views/logs/system/index.vue'),
        meta: {
          icon: 'mdi:file-document-outline',
          title: '系统日志',
        },
      },
    ],
  },
];

export default routes;
