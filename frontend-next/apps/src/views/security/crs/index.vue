<script lang="ts" setup>
import { Page } from '@vben/common-ui';

import { Card, Result } from 'ant-design-vue';

defineOptions({ name: 'SecurityCrs' });
</script>

<template>
  <Page title="CRS Management" description="Manage OWASP Core Rule Set (CRS) for the WAF engine.">
    <Card>
      <Result
        status="info"
        title="CRS Management"
        sub-title="Configure and manage OWASP Core Rule Set versions, rule updates, and custom CRS overrides for the WAF engine."
      />
    </Card>
  </Page>
</template>
