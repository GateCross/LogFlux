<script lang="ts" setup>
import { Page } from '@vben/common-ui';

import { Card, Result } from 'ant-design-vue';

defineOptions({ name: 'SecurityRuntime' });
</script>

<template>
  <Page title="WAF Runtime" description="Runtime status and real-time monitoring for the WAF engine.">
    <Card>
      <Result
        status="info"
        title="WAF Runtime"
        sub-title="Monitor the WAF engine runtime status, active configurations, and real-time request processing metrics."
      />
    </Card>
  </Page>
</template>
