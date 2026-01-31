<template>
  <div class="h-full">
    <n-card :bordered="false" class="h-full rounded-16px shadow-sm">
      <n-tabs type="line" animated>
        <n-tab-pane name="profile" :tab="$t('page.userCenter.profile')">
          <!-- Basic Profile Info -->
          <n-form
            ref="formRef"
            :label-width="80"
            :model="authStore.userInfo"
            label-placement="left"
          >
            <n-form-item :label="$t('page.userCenter.username')">
              <n-input v-model:value="authStore.userInfo.username" disabled />
            </n-form-item>
            <n-form-item :label="$t('page.userCenter.roles')">
              <n-tag v-for="role in authStore.userInfo.roles" :key="role" type="primary" class="mr-2">
                {{ role }}
              </n-tag>
            </n-form-item>
          </n-form>
        </n-tab-pane>
        <n-tab-pane name="preferences" :tab="$t('page.userCenter.preferences')">
          <!-- Notification Settings -->
          <n-divider title-placement="left">{{ $t('page.userCenter.notificationSettings') }}</n-divider>
          <n-form
            ref="prefFormRef"
            :label-width="120"
            :model="preferences"
            label-placement="left"
          >
            <n-form-item :label="$t('page.userCenter.inAppNotificationLevel')">
              <n-select
                v-model:value="preferences.minLevel"
                :options="levelOptions"
                :placeholder="$t('page.userCenter.selectMinLevel')"
              />
            </n-form-item>
            <n-form-item>
              <n-button type="primary" :loading="loading" @click="handleSavePreferences">
                {{ $t('page.userCenter.savePreferences') }}
              </n-button>
            </n-form-item>
            <n-form-item>
               <n-alert :title="$t('page.userCenter.note')" type="info" :bordered="false">
                 {{ $t('page.userCenter.noteContent') }}
               </n-alert>
            </n-form-item>
          </n-form>
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue';
import { useAuthStore } from '@/store/modules/auth';
import { fetchUpdateUserPreferences } from '@/service/api/auth';
import { useMessage } from 'naive-ui';
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

<style scoped></style>
