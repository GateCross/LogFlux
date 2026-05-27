<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useMessage } from 'naive-ui';
import { fetchUpdateUserPreferences } from '@/service/api/auth';
import { useAuthStore } from '@/store/modules/auth';
import { $t } from '@/locales';

const authStore = useAuthStore();
const message = useMessage();
const loading = ref(false);

interface UserPreferences {
  minLevel: string;
}

const preferences = reactive<UserPreferences>({
  minLevel: 'info' // Default
});

const levelOptions = computed(() => [
  { label: $t('page.userCenter.levels.info'), value: 'info' },
  { label: $t('page.userCenter.levels.warning'), value: 'warning' },
  { label: $t('page.userCenter.levels.error'), value: 'error' },
  { label: $t('page.userCenter.levels.critical'), value: 'critical' }
]);

function initPreferences() {
  if (authStore.userInfo.preferences) {
    try {
      const prefs = JSON.parse(authStore.userInfo.preferences);
      if (prefs.minLevel) {
        preferences.minLevel = prefs.minLevel;
      }
    } catch (e) {
      console.error('Failed to parse user preferences', e);
    }
  }
}

async function handleSavePreferences() {
  loading.value = true;
  try {
    const prefsStr = JSON.stringify(preferences);
    const { error } = await fetchUpdateUserPreferences(prefsStr);
    if (!error) {
      message.success($t('page.userCenter.saveSuccess'));
      // Update store
      authStore.userInfo.preferences = prefsStr;
    } else {
      message.error($t('page.userCenter.saveFailed'));
    }
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  initPreferences();
});
</script>

<template>
  <div class="h-full">
    <NCard :bordered="false" class="h-full rounded-16px shadow-sm">
      <NTabs type="line" animated>
        <NTabPane name="profile" :tab="$t('page.userCenter.profile')">
          <!-- Basic Profile Info -->
          <NForm ref="formRef" :label-width="80" :model="authStore.userInfo" label-placement="left">
            <NFormItem :label="$t('page.userCenter.username')">
              <NInput v-model:value="authStore.userInfo.username" disabled />
            </NFormItem>
            <NFormItem :label="$t('page.userCenter.roles')">
              <NTag v-for="role in authStore.userInfo.roles" :key="role" type="primary" class="mr-2">
                {{ role }}
              </NTag>
            </NFormItem>
          </NForm>
        </NTabPane>
        <NTabPane name="preferences" :tab="$t('page.userCenter.preferences')">
          <!-- Notification Settings -->
          <NDivider title-placement="left">{{ $t('page.userCenter.notificationSettings') }}</NDivider>
          <NForm ref="prefFormRef" :label-width="120" :model="preferences" label-placement="left">
            <NFormItem :label="$t('page.userCenter.inAppNotificationLevel')">
              <NSelect
                v-model:value="preferences.minLevel"
                :options="levelOptions"
                :placeholder="$t('page.userCenter.selectMinLevel')"
              />
            </NFormItem>
            <NFormItem>
              <NButton type="primary" :loading="loading" @click="handleSavePreferences">
                {{ $t('page.userCenter.savePreferences') }}
              </NButton>
            </NFormItem>
            <NFormItem>
              <NAlert :title="$t('page.userCenter.note')" type="info" :bordered="false">
                {{ $t('page.userCenter.noteContent') }}
              </NAlert>
            </NFormItem>
          </NForm>
        </NTabPane>
      </NTabs>
    </NCard>
  </div>
</template>

<style scoped></style>
