<script setup lang="ts">
import GlobalContent from '../modules/global-content/index.vue';

defineOptions({
  name: 'BlankLayout'
});
</script>

<template>
  <div class="h-full">
    <GlobalContent :show-padding="false" />
  </div>
</template>

<style scoped></style>
