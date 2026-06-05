<script lang="ts" setup>
import type { UploadProps } from 'ant-design-vue';

import { computed, reactive, watch } from 'vue';

import { useUserStore } from '@vben/stores';

import {
  Avatar,
  Button,
  Card,
  Col,
  Descriptions,
  DescriptionsItem,
  Form,
  FormItem,
  Input,
  Row,
  Select,
  Space,
  Tag,
  Textarea,
  Upload,
  message,
} from 'ant-design-vue';

import { updateUserPreferencesApi } from '#/api/core/user';

defineOptions({ name: 'UserCenter' });

interface UserProfile {
  avatar?: string;
  displayName?: string;
  email?: string;
  homePath?: string;
  introduction?: string;
  language?: string;
  preferences?: string;
  realName?: string;
  roles?: string[];
  userId?: string;
  username?: string;
}

const userStore = useUserStore();

const defaultAvatars = [
  {
    color: '#1677ff',
    label: '蓝',
    value: avatarDataUrl('#1677ff', '#e6f4ff', 'LF'),
  },
  {
    color: '#13a8a8',
    label: '青',
    value: avatarDataUrl('#13a8a8', '#e6fffb', 'LF'),
  },
  {
    color: '#722ed1',
    label: '紫',
    value: avatarDataUrl('#722ed1', '#f9f0ff', 'LF'),
  },
  {
    color: '#d4380d',
    label: '橙',
    value: avatarDataUrl('#d4380d', '#fff2e8', 'LF'),
  },
];

const preferencesLoading = reactive({
  saving: false,
  uploading: false,
});

const prefs = reactive({
  avatar: defaultAvatars[0]?.value ?? '',
  displayName: '',
  email: '',
  language: 'zh-CN',
  introduction: '',
});

const userInfo = computed<UserProfile>(() => userStore.userInfo ?? {});
const currentAvatar = computed(() => prefs.avatar || userInfo.value.avatar);
const currentDisplayName = computed(
  () => prefs.displayName || userInfo.value.realName || userInfo.value.username || 'LogFlux 用户',
);
const roleLabels = computed(() => userInfo.value.roles ?? []);

function avatarDataUrl(primary: string, secondary: string, text: string) {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="160" height="160" viewBox="0 0 160 160"><rect width="160" height="160" rx="32" fill="${secondary}"/><circle cx="80" cy="64" r="34" fill="${primary}"/><path d="M32 142c8-30 25-46 48-46s40 16 48 46" fill="${primary}" opacity=".9"/><text x="80" y="74" text-anchor="middle" font-family="Arial, sans-serif" font-size="28" font-weight="700" fill="white">${text}</text></svg>`;
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`;
}

function parsePreferences(value?: string) {
  if (!value) return {};
  try {
    const parsed = JSON.parse(value);
    return parsed && typeof parsed === 'object' ? parsed as Record<string, any> : {};
  } catch {
    return {};
  }
}

function syncPreferences() {
  const saved = parsePreferences(userInfo.value.preferences);
  prefs.avatar = saved.avatar || userInfo.value.avatar || defaultAvatars[0]?.value || '';
  prefs.displayName = saved.displayName || userInfo.value.realName || userInfo.value.username || '';
  prefs.email = saved.email || userInfo.value.email || '';
  prefs.language = saved.language || userInfo.value.language || 'zh-CN';
  prefs.introduction = saved.introduction || userInfo.value.introduction || '';
}

function updateLocalUserInfo() {
  userStore.setUserInfo({
    ...(userStore.userInfo ?? {}),
    avatar: prefs.avatar,
    displayName: prefs.displayName,
    email: prefs.email,
    introduction: prefs.introduction,
    language: prefs.language,
    realName: prefs.displayName || userInfo.value.username || 'LogFlux 用户',
    preferences: JSON.stringify(buildPreferencesPayload()),
  } as any);
}

function buildPreferencesPayload() {
  return {
    avatar: prefs.avatar,
    displayName: prefs.displayName,
    email: prefs.email,
    introduction: prefs.introduction,
    language: prefs.language,
  };
}

async function handleSavePreferences() {
  preferencesLoading.saving = true;
  try {
    await updateUserPreferencesApi(JSON.stringify(buildPreferencesPayload()));
    updateLocalUserInfo();
    message.success('个人资料已保存');
  } catch {
    message.error('保存个人资料失败');
  } finally {
    preferencesLoading.saving = false;
  }
}

function handleSelectAvatar(avatar: string) {
  prefs.avatar = avatar;
}

function compressImage(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener('error', () => reject(new Error('读取图片失败')));
    reader.addEventListener('load', () => {
      const image = new Image();
      image.addEventListener('error', () => reject(new Error('图片解析失败')));
      image.addEventListener('load', () => {
        const canvas = document.createElement('canvas');
        const size = 160;
        canvas.width = size;
        canvas.height = size;
        const context = canvas.getContext('2d');
        if (!context) {
          reject(new Error('浏览器不支持图片处理'));
          return;
        }

        const sourceSize = Math.min(image.width, image.height);
        const sourceX = (image.width - sourceSize) / 2;
        const sourceY = (image.height - sourceSize) / 2;
        context.clearRect(0, 0, size, size);
        context.drawImage(image, sourceX, sourceY, sourceSize, sourceSize, 0, 0, size, size);
        resolve(canvas.toDataURL('image/png'));
      });
      image.src = String(reader.result);
    });
    reader.readAsDataURL(file);
  });
}

const beforeAvatarUpload: UploadProps['beforeUpload'] = async (file) => {
  if (!file.type.startsWith('image/')) {
    message.warning('请选择图片文件');
    return Upload.LIST_IGNORE;
  }
  if (file.size > 2 * 1024 * 1024) {
    message.warning('头像图片不能超过 2 MiB');
    return Upload.LIST_IGNORE;
  }

  preferencesLoading.uploading = true;
  try {
    prefs.avatar = await compressImage(file);
    message.success('头像已选择，保存后生效');
  } catch {
    message.error('头像处理失败');
  } finally {
    preferencesLoading.uploading = false;
  }
  return Upload.LIST_IGNORE;
};

watch(
  () => userStore.userInfo,
  () => syncPreferences(),
  { immediate: true },
);
</script>

<template>
  <div class="user-center-page">
    <Row :gutter="[16, 16]">
      <Col :xs="24" :lg="8">
        <Card :bordered="false">
          <div class="profile-summary">
            <Avatar :size="96" :src="currentAvatar">
              {{ currentDisplayName.slice(0, 1) }}
            </Avatar>
            <h2>{{ currentDisplayName }}</h2>
            <p>{{ userInfo.username || '-' }}</p>
            <Space class="role-tags" wrap>
              <Tag v-for="role in roleLabels" :key="role" color="blue">
                {{ role === 'admin' ? '管理员' : role === 'analyst' ? '分析员' : role }}
              </Tag>
            </Space>
          </div>

          <Descriptions :column="1" size="small" bordered class="profile-descriptions">
            <DescriptionsItem label="用户名">
              {{ userInfo.username || '-' }}
            </DescriptionsItem>
            <DescriptionsItem label="邮箱">
              {{ prefs.email || '-' }}
            </DescriptionsItem>
            <DescriptionsItem label="角色">
              {{ roleLabels.join('，') || '-' }}
            </DescriptionsItem>
          </Descriptions>
        </Card>
      </Col>

      <Col :xs="24" :lg="16">
        <Card :bordered="false" title="个人资料">
          <Form
            :model="prefs"
            :label-col="{ span: 5 }"
            :wrapper-col="{ span: 17 }"
          >
            <FormItem label="个人头像">
              <div class="avatar-editor">
                <Avatar :size="72" :src="currentAvatar">
                  {{ currentDisplayName.slice(0, 1) }}
                </Avatar>
                <div class="avatar-actions">
                  <div class="avatar-options">
                    <button
                      v-for="item in defaultAvatars"
                      :key="item.label"
                      class="avatar-option"
                      :class="{ active: prefs.avatar === item.value }"
                      type="button"
                      :title="`使用${item.label}色默认头像`"
                      @click="handleSelectAvatar(item.value)"
                    >
                      <span :style="{ backgroundColor: item.color }"></span>
                    </button>
                  </div>
                  <Upload
                    :before-upload="beforeAvatarUpload"
                    :max-count="1"
                    accept="image/*"
                    :show-upload-list="false"
                  >
                    <Button :loading="preferencesLoading.uploading">
                      上传头像
                    </Button>
                  </Upload>
                </div>
              </div>
            </FormItem>

            <FormItem label="显示名称">
              <Input v-model:value="prefs.displayName" placeholder="请输入显示名称" />
            </FormItem>

            <FormItem label="邮箱">
              <Input v-model:value="prefs.email" placeholder="请输入邮箱地址" />
            </FormItem>

            <FormItem label="语言">
              <Select v-model:value="prefs.language">
                <Select.Option value="zh-CN">简体中文</Select.Option>
                <Select.Option value="en-US">English</Select.Option>
              </Select>
            </FormItem>

            <FormItem label="个人简介">
              <Textarea
                v-model:value="prefs.introduction"
                :rows="4"
                placeholder="写一点个人介绍，方便团队成员识别你"
              />
            </FormItem>

            <FormItem :wrapper-col="{ offset: 5, span: 17 }">
              <Space>
                <Button type="primary" :loading="preferencesLoading.saving" @click="handleSavePreferences">
                  保存资料
                </Button>
                <Button @click="syncPreferences">
                  恢复当前资料
                </Button>
              </Space>
            </FormItem>
          </Form>
        </Card>
      </Col>
    </Row>
  </div>
</template>

<style scoped>
.user-center-page {
  padding: 20px;
}

.profile-summary {
  display: flex;
  align-items: center;
  flex-direction: column;
  padding: 16px 0 20px;
  text-align: center;
}

.profile-summary h2 {
  margin: 16px 0 4px;
  font-size: 18px;
  font-weight: 600;
}

.profile-summary p {
  margin: 0;
  color: #667085;
}

.role-tags {
  justify-content: center;
  margin-top: 12px;
}

.profile-descriptions {
  margin-top: 8px;
}

.avatar-editor {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
}

.avatar-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.avatar-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.avatar-option {
  width: 30px;
  height: 30px;
  padding: 2px;
  cursor: pointer;
  background: transparent;
  border: 1px solid #d0d5dd;
  border-radius: 50%;
}

.avatar-option span {
  display: block;
  width: 100%;
  height: 100%;
  border-radius: 50%;
}

.avatar-option.active {
  border-color: #1677ff;
  box-shadow: 0 0 0 2px rgba(22, 119, 255, 0.15);
}
</style>
