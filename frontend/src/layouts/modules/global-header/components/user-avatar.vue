<script setup lang="ts">
import { computed, ref, reactive } from 'vue';
import type { VNode } from 'vue';
import { useMessage } from 'naive-ui';
import { useAuthStore } from '@/store/modules/auth';
import { useRouterPush } from '@/hooks/common/router';
import { useSvgIcon } from '@/hooks/common/icon';
import { $t } from '@/locales';
import { request } from '@/service/request';
import { encrypt } from '@/utils/crypto';

defineOptions({
  name: 'UserAvatar'
});

const authStore = useAuthStore();
const { routerPushByKey, toLogin } = useRouterPush();
const { SvgIconVNode } = useSvgIcon();
const message = useMessage();

function loginOrRegister() {
  toLogin();
}

type DropdownKey = 'user-center' | 'logout' | 'changePassword';

type DropdownOption =
  | {
      key: DropdownKey;
      label: string;
      icon?: () => VNode;
    }
  | {
      type: 'divider';
      key: string;
    };

const options = computed(() => {
  const opts: DropdownOption[] = [
    {
      label: $t('route.user_center'),
      key: 'user-center',
      icon: SvgIconVNode({ icon: 'ph:user', fontSize: 18 })
    },
    {
      label: $t('common.changePassword'),
      key: 'changePassword',
      icon: SvgIconVNode({ icon: 'carbon:password', fontSize: 18 })
    },
    {
      type: 'divider',
      key: 'divider1'
    },
    {
      label: $t('common.logout'),
      key: 'logout',
      icon: SvgIconVNode({ icon: 'ph:sign-out', fontSize: 18 })
    }
  ];

  return opts;
});

function logout() {
  window.$dialog?.info({
    title: $t('common.tip'),
    content: $t('common.logoutConfirm'),
    positiveText: $t('common.confirm'),
    negativeText: $t('common.cancel'),
    onPositiveClick: () => {
      authStore.resetStore();
    }
  });
}

// Password change modal
const showModal = ref(false);
const formRef = ref<any>(null);
const submitLoading = ref(false);
const formModel = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
});

const rules: import('naive-ui').FormRules = {
  oldPassword: { required: true, message: '请输入旧密码', trigger: 'blur' },
  newPassword: { required: true, message: '请输入新密码', trigger: 'blur' },
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (rule, value) => {
        return value === formModel.newPassword;
      },
      message: '两次输入密码不一致',
      trigger: 'blur'
    }
  ]
};

async function handlePasswordSubmit() {
  formRef.value?.validate(async (errors: any) => {
    if (!errors) {
      submitLoading.value = true;
      try {
        const { error } = await request({
          url: '/api/user/change_password',
          method: 'post',
          data: {
            oldPassword: encrypt(formModel.oldPassword),
            newPassword: encrypt(formModel.newPassword)
          }
        });
        if (!error) {
          message.success('密码修改成功，请重新登录');
          showModal.value = false;
          // Logout after 1s
          setTimeout(() => {
            authStore.resetStore();
          }, 1000);
        }
      } finally {
        submitLoading.value = false;
      }
    }
  });
}

function handleDropdown(key: DropdownKey) {
  console.log('handleDropdown key:', key);
  if (key === 'logout') {
    logout();
  } else if (key === 'changePassword') {
    formModel.oldPassword = '';
    formModel.newPassword = '';
    formModel.confirmPassword = '';
    showModal.value = true;
  } else if (key === 'user-center') {
    console.log('Navigating to user_center');
    routerPushByKey('user_center');
  } else {
    // If your other options are jumps from other routes, they will be directly supported here
    routerPushByKey(key);
  }
}
</script>

<template>
  <NButton v-if="!authStore.isLogin" quaternary @click="loginOrRegister">
    {{ $t('page.login.common.loginOrRegister') }}
  </NButton>
  <NDropdown v-else placement="bottom" trigger="click" :options="options" @select="handleDropdown">
    <div>
      <ButtonIcon>
        <SvgIcon icon="ph:user-circle" class="text-icon-large" />
        <span class="text-16px font-medium">{{ authStore.userInfo.username }}</span>
      </ButtonIcon>
    </div>
  </NDropdown>

  <n-modal v-model:show="showModal" :title="$t('common.changePassword')" preset="card" class="w-400px">
    <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="80">
      <n-form-item :label="$t('common.oldPassword')" path="oldPassword">
        <n-input v-model:value="formModel.oldPassword" type="password" show-password-on="mousedown" placeholder="请输入旧密码" />
      </n-form-item>
      <n-form-item :label="$t('common.newPassword')" path="newPassword">
        <n-input v-model:value="formModel.newPassword" type="password" show-password-on="mousedown" placeholder="请输入新密码" />
      </n-form-item>
      <n-form-item :label="$t('common.confirmPassword')" path="confirmPassword">
        <n-input v-model:value="formModel.confirmPassword" type="password" show-password-on="mousedown" placeholder="请再次输入新密码" />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="showModal = false">取消</n-button>
        <n-button type="primary" :loading="submitLoading" @click="handlePasswordSubmit">确定</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped></style>
