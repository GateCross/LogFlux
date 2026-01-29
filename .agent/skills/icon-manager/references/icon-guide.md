# LogFlux Icon Guide

## Allowed Icon Sets
We strictly use **Iconify** with offline bundling. Do NOT use online URLs.

**Pre-installed Icon Sets** (found in `package.json`):
- `mdi` (Material Design Icons)
- `carbon` (Carbon Design System)
- `ant-design`
- `heroicons`
- `ic` (Google Material Icons)
- `line-md`
- `majesticons`
- `material-symbols`
- `ph` (Phosphor)

## Import Convention
All icons must be registered in `frontend/src/plugins/iconify.ts`.

### format
```typescript
// 1. Import
import IconName from '@iconify/icons-<set>/<icon-name>';

// 2. Register
addIcon('<set>:<icon-name>', IconName);
```

### Example
To use `mdi:home`:
```typescript
import Home from '@iconify/icons-mdi/home';

// ... inside setupIconifyOffline function
addIcon('mdi:home', Home);
```

## Strict Rules
1. **No Mixed Styles**: Stick to the existing sets.
2. **Offline Only**: Use `@iconify/icons-*` packages.
3. **Naming**: Variable name should be PascalCase of the icon name (e.g., `account-box` -> `AccountBox`).
