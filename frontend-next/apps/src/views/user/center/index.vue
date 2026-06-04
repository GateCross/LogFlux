<script lang="ts" setup>
import { computed, reactive, ref } from 'vue';

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
  message,
} from 'ant-design-vue';

defineOptions({ name: 'UserCenter' });

const userStore = useUserStore();

interface UserProfile {
  avatar?: string;
  email?: string;
  introduction?: string;
  language?: string;
  notifyEmail?: boolean;
  realName?: string;
  roles?: string[];
  username?: string;
}

const userInfo = computed<UserProfile>(() => userStore.userInfo ?? {});

// ── Preferences form ─────────────────────────────────────────

const preferencesLoading = ref(false);

const prefs = reactive({
  displayName: userInfo.value?.realName ?? '',
  email: userInfo.value?.email ?? '',
  language: userInfo.value?.language ?? 'zh-CN',
  introduction: userInfo.value?.introduction ?? '',
  notifyEmail: userInfo.value?.notifyEmail ?? true,
});

function handleSavePreferences() {
  preferencesLoading.value = true;
  // TODO: wire to real preferences update API when available
  setTimeout(() => {
    preferencesLoading.value = false;
    message.success('Preferences saved');
  }, 500);
}
</script>

<template>
  <div class="p-5">
    <Row :gutter="[16, 16]">
      <!-- ── Profile card ─────────────────────────────────── -->
      <Col :xs="24" :lg="8">
        <Card>
          <div class="flex flex-col items-center py-4">
            <Avatar
              :size="96"
              :src="userInfo?.avatar"
            >
              {{ userInfo?.realName?.[0] ?? 'U' }}
            </Avatar>
            <h2 class="mt-4 mb-1 text-lg font-semibold">
              {{ userInfo?.realName ?? userInfo?.username ?? 'Unknown' }}
            </h2>
            <p class="text-gray-500">
              {{ userInfo?.username ?? '' }}
            </p>
            <Space class="mt-2" wrap>
              <Tag
                v-for="role in userInfo?.roles ?? []"
                :key="role"
                color="blue"
              >
                {{ role }}
              </Tag>
            </Space>
          </div>

          <Descriptions :column="1" size="small" bordered class="mt-2">
            <DescriptionsItem label="Username">
              {{ userInfo?.username ?? '-' }}
            </DescriptionsItem>
            <DescriptionsItem label="Email">
              {{ userInfo?.email ?? '-' }}
            </DescriptionsItem>
            <DescriptionsItem label="Roles">
              {{ (userInfo?.roles ?? []).join(', ') || '-' }}
            </DescriptionsItem>
          </Descriptions>
        </Card>
      </Col>

      <!-- ── Preferences form ─────────────────────────────── -->
      <Col :xs="24" :lg="16">
        <Card title="Account Preferences">
          <Form
            :model="prefs"
            :label-col="{ span: 6 }"
            :wrapper-col="{ span: 16 }"
          >
            <FormItem label="Display Name">
              <Input
                v-model:value="prefs.displayName"
                placeholder="Your display name"
              />
            </FormItem>

            <FormItem label="Email">
              <Input
                v-model:value="prefs.email"
                placeholder="you@example.com"
              />
            </FormItem>

            <FormItem label="Language">
              <Select v-model:value="prefs.language">
                <Select.Option value="zh-CN">Chinese (Simplified)</Select.Option>
                <Select.Option value="en-US">English (US)</Select.Option>
              </Select>
            </FormItem>

            <FormItem label="Introduction">
              <Textarea
                v-model:value="prefs.introduction"
                :rows="4"
                placeholder="A brief introduction about yourself"
              />
            </FormItem>

            <FormItem :wrapper-col="{ offset: 6, span: 16 }">
              <Button
                type="primary"
                :loading="preferencesLoading"
                @click="handleSavePreferences"
              >
                Save Preferences
              </Button>
            </FormItem>
          </Form>
        </Card>
      </Col>
    </Row>
  </div>
</template>
