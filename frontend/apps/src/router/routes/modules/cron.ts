import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    name: 'Cron',
    path: '/cron',
    component: () => import('#/views/cron/index.vue'),
    meta: {
      icon: 'mdi:clock-outline',
      order: 8,
      title: '定时任务',
    },
  },
];

export default routes;
