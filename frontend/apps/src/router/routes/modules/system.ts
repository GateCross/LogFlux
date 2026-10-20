import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'mdi:cog-sync-outline',
      order: 6,
      title: '系统',
    },
    name: 'System',
    path: '/system',
    children: [
      {
        name: 'SystemUser',
        path: '/system/user',
        component: () => import('#/views/system/user/index.vue'),
        meta: {
          icon: 'mdi:account-outline',
          title: '用户管理',
        },
      },
    ],
  },
];

export default routes;
