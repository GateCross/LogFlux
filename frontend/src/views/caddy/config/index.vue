<script setup lang="ts">
import { $t } from '@/locales';
</script>

<template>
  <div class="h-full overflow-hidden">
    <NCard :title="$t('route.caddy_config')" class="h-full sm:flex-1-hidden card-wrapper">
      <div class="h-full flex-col">
          Caddy Config Page
      </div>
    </NCard>
  </div>
</template>

<style scoped></style>
