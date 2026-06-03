import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'mdi:account-cog-outline',
      order: 5,
      title: '系统管理',
    },
    name: 'Manage',
    path: '/manage',
    children: [
      {
        name: 'ManageUser',
        path: '/manage/user',
        component: () => import('#/views/manage/user/index.vue'),
        meta: { icon: 'mdi:account-outline', title: '用户管理' },
      },
      {
        name: 'ManageRole',
        path: '/manage/role',
        component: () => import('#/views/manage/role/index.vue'),
        meta: { icon: 'mdi:shield-account-outline', title: '角色管理' },
      },
      {
        name: 'ManageMenu',
        path: '/manage/menu',
        component: () => import('#/views/manage/menu/index.vue'),
        meta: { icon: 'mdi:menu', title: '菜单管理' },
      },
    ],
  },
];

export default routes;
