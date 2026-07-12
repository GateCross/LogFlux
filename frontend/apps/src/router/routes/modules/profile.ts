import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    name: 'User',
    path: '/user/center',
    component: () => import('#/views/user/center/index.vue'),
    meta: {
      icon: 'lucide:user',
      hideInMenu: true,
      title: '个人中心',
    },
  },
];

export default routes;
