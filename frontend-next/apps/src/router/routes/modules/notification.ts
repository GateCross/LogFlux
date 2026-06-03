import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'mdi:bell-outline',
      order: 4,
      title: '通知管理',
    },
    name: 'Notification',
    path: '/notification',
    children: [
      {
        name: 'NotificationChannel',
        path: '/notification/channel',
        component: () => import('#/views/notification/channel/index.vue'),
        meta: { icon: 'mdi:broadcast', title: '通知渠道' },
      },
      {
        name: 'NotificationRule',
        path: '/notification/rule',
        component: () => import('#/views/notification/rule/index.vue'),
        meta: { icon: 'mdi:script-text-outline', title: '通知规则' },
      },
      {
        name: 'NotificationTemplate',
        path: '/notification/template',
        component: () => import('#/views/notification/template/index.vue'),
        meta: { icon: 'mdi:file-outline', title: '通知模板' },
      },
      {
        name: 'NotificationLog',
        path: '/notification/log',
        component: () => import('#/views/notification/log/index.vue'),
        meta: { icon: 'mdi:text-box-outline', title: '发送日志' },
      },
    ],
  },
];

export default routes;
