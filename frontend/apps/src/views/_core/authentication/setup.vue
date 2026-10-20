<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';

import { computed } from 'vue';

import { AuthenticationLogin, z } from '@vben/common-ui';

import { message } from 'antdv-next';

import { useAuthStore } from '#/store';

defineOptions({ name: 'Setup' });

const authStore = useAuthStore();

/** 密码规则与后端 common/password 保持一致：8 位以上，且含字母和数字 */
const PASSWORD_PATTERN = /^(?=.*[a-z])(?=.*\d).{8,}$/i;

const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入管理员用户名',
      },
      fieldName: 'username',
      label: '用户名',
      rules: z.string().min(1, { message: '请输入用户名' }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: '至少 8 位，需包含字母和数字',
      },
      fieldName: 'password',
      label: '密码',
      rules: z
        .string()
        .min(8, { message: '密码至少 8 位' })
        .regex(PASSWORD_PATTERN, { message: '密码需同时包含字母和数字' }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: '请再次输入密码',
      },
      fieldName: 'confirmPassword',
      label: '确认密码',
      rules: z.string().min(1, { message: '请再次输入密码' }),
    },
  ];
});

/** 校验两次输入一致后再提交，沿用登录接口的 loading 状态 */
async function handleSubmit(values: Record<string, any>) {
  if (values.password !== values.confirmPassword) {
    message.error('两次输入的密码不一致');
    return;
  }
  await authStore.setupAdmin({
    password: values.password,
    username: values.username,
  });
}
</script>

<template>
  <AuthenticationLogin
    :form-schema="formSchema"
    :loading="authStore.loginLoading"
    :show-code-login="false"
    :show-forget-password="false"
    :show-qrcode-login="false"
    :show-register="false"
    :show-third-party-login="false"
    sub-title="首次使用，请创建管理员账号"
    title="初始化 LogFlux"
    @submit="handleSubmit"
  />
</template>
