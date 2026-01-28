<script setup lang="ts">
import { $t } from '@/locales';
</script>

<template>
  <div class="h-full overflow-hidden">
    <NCard :title="$t('route.caddy_log')" class="h-full sm:flex-1-hidden card-wrapper">
      <div class="h-full flex-col">
          Caddy Log Page
      </div>
    </NCard>
  </div>
</template>

<style scoped></style>
