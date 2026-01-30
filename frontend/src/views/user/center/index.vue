<template>
  <div class="h-full">
    <n-card :bordered="false" class="h-full rounded-16px shadow-sm">
      <n-tabs type="line" animated>
        <n-tab-pane name="profile" tab="Profile">
          <!-- Basic Profile Info -->
          <n-form
            ref="formRef"
            :label-width="80"
            :model="authStore.userInfo"
            label-placement="left"
          >
            <n-form-item label="Username">
              <n-input v-model:value="authStore.userInfo.username" disabled />
            </n-form-item>
            <n-form-item label="Roles">
              <n-tag v-for="role in authStore.userInfo.roles" :key="role" type="primary" class="mr-2">
                {{ role }}
              </n-tag>
            </n-form-item>
          </n-form>
        </n-tab-pane>
        <n-tab-pane name="preferences" tab="Preferences">
          <!-- Notification Settings -->
          <n-divider title-placement="left">Notification Settings</n-divider>
          <n-form
            ref="prefFormRef"
            :label-width="120"
            :model="preferences"
            label-placement="left"
          >
            <n-form-item label="In-App Notification Level">
              <n-select
                v-model:value="preferences.minLevel"
                :options="levelOptions"
                placeholder="Select Minimum Level"
              />
            </n-form-item>
            <n-form-item>
              <n-button type="primary" :loading="loading" @click="handleSavePreferences">
                Save Preferences
              </n-button>
            </n-form-item>
            <n-form-item>
               <n-alert title="Note" type="info" :bordered="false">
                 Only notifications with a level equal to or higher than the selected level will be shown in the global header.
               </n-alert>
            </n-form-item>
          </n-form>
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue';
import { useAuthStore } from '@/store/modules/auth';
import { fetchUpdateUserPreferences } from '@/service/api/auth';
import { useMessage } from 'naive-ui';

const authStore = useAuthStore();
const message = useMessage();
const loading = ref(false);

interface UserPreferences {
  minLevel: string;
}

const preferences = reactive<UserPreferences>({
  minLevel: 'info' // Default
});

const levelOptions = [
  { label: 'Info', value: 'info' },
  { label: 'Warning', value: 'warning' },
  { label: 'Error', value: 'error' },
  { label: 'Critical', value: 'critical' }
];

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
      message.success('Preferences saved successfully');
      // Update store
      authStore.userInfo.preferences = prefsStr;
    } else {
      message.error('Failed to save preferences');
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
