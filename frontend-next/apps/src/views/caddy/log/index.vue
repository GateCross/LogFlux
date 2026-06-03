<script lang="ts" setup>
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';

// This view is an alias / redirect to the access logs page.
const router = useRouter();

onMounted(() => {
  router.replace('/caddy/access');
});
</script>

<template>
  <div />
</template>
