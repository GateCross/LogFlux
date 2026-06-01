/**
 * Security 既有单元测试移植（任务 11.3，Req 17.2 / 17.5）。
 *
 * 来源：旧 Vue 版 `frontend/src/views/security/security-policy-utils.test.ts`
 * （原以 `node:test` + `node:assert/strict` 通过 `tsx --test` 运行）。
 * 改写为 vitest（`describe`/`it`/`expect`），断言逻辑与覆盖范围保持不变。
 */
import { describe, expect, it } from 'vitest';

import {
  buildPolicyWorkspaceActions,
  formatBytes,
  mapCrsTemplateLabel,
  mapPolicyEngineModeLabel,
  mapPolicyRevisionStatusLabel,
  mapScopeTypeLabel,
} from './security-policy-utils';

describe('security-policy-utils', () => {
  it('security policy mapping helpers return expected labels', () => {
    expect(mapPolicyEngineModeLabel('on')).toBe('On（阻断）');
    expect(mapCrsTemplateLabel('balanced')).toBe('平衡');
    expect(mapScopeTypeLabel('route')).toBe('路由');
    expect(mapPolicyRevisionStatusLabel('rolled_back')).toBe('已回滚');
  });

  it('formatBytes formats boundary values', () => {
    expect(formatBytes(0)).toBe('-');
    expect(formatBytes(1024)).toBe('1.00 KB');
    expect(formatBytes(10 * 1024 * 1024)).toBe('10 MB');
  });

  it('buildPolicyWorkspaceActions exposes section-specific guidance', () => {
    const crsActions = buildPolicyWorkspaceActions({
      activeSection: 'crs',
      hasPendingCrsTuningChanges: true,
      bindingConflictCount: 0,
      selectedPolicyName: 'default-policy',
    });
    expect(crsActions.some(item => item.includes('未保存改动'))).toBe(true);

    const bindingActions = buildPolicyWorkspaceActions({
      activeSection: 'binding',
      hasPendingCrsTuningChanges: false,
      bindingConflictCount: 2,
      selectedPolicyName: 'default-policy',
    });
    expect(bindingActions.some(item => item.includes('2 组绑定冲突'))).toBe(true);
  });
});
