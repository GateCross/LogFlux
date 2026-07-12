# LogFlux Icon Guide

## Allowed Icon Sets
We strictly use **Iconify offline**. Do NOT use online CDN/API.

**Installed Icon Sets** (`packages/icons/package.json`):
- `mdi`
- `carbon`
- `ant-design`
- `ic`
- `ep`
- `fluent-mdl2`
- `lucide`
- `material-symbols`

## Registration
All collections are registered in:

`frontend/packages/icons/src/iconify/load.ts`

```typescript
import { icons as mdi } from '@iconify-json/mdi';
import { addCollection, registerIconNames } from '@vben-core/icons';

addCollection(mdi);
registerIconNames(mdi.prefix, Object.keys(mdi.icons));
```

## Usage
```vue
<script setup>
import { IconifyIcon } from '@vben/icons';
// 或
import { Icon } from '@iconify/vue/offline';
</script>

<template>
  <IconifyIcon icon="mdi:home" />
</template>
```

## Strict Rules
1. **Offline Only**: never import default `@iconify/vue` Icon (online).
2. **Register First**: new prefixes must be added to `load.ts`.
3. **Naming**: use `prefix:name` (e.g. `carbon:settings`).
