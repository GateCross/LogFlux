import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildPolicyWorkspaceActions,
  formatBytes,
  mapCrsTemplateLabel,
  mapPolicyEngineModeLabel,
  mapPolicyRevisionStatusLabel,
  mapScopeTypeLabel
} from './security-policy-utils';

test('security policy mapping helpers return expected labels', () => {
  assert.equal(mapPolicyEngineModeLabel('on'), 'On（阻断）');
  assert.equal(mapCrsTemplateLabel('balanced'), '平衡');
  assert.equal(mapScopeTypeLabel('route'), '路由');
  assert.equal(mapPolicyRevisionStatusLabel('rolled_back'), '已回滚');
});

test('formatBytes formats boundary values', () => {
  assert.equal(formatBytes(0), '-');
  assert.equal(formatBytes(1024), '1.00 KB');
  assert.equal(formatBytes(10 * 1024 * 1024), '10 MB');
});

test('buildPolicyWorkspaceActions exposes section-specific guidance', () => {
  const crsActions = buildPolicyWorkspaceActions({
    activeSection: 'crs',
    hasPendingCrsTuningChanges: true,
    bindingConflictCount: 0,
    selectedPolicyName: 'default-policy'
  });
  assert.ok(crsActions.some(item => item.includes('未保存改动')));

  const bindingActions = buildPolicyWorkspaceActions({
    activeSection: 'binding',
    hasPendingCrsTuningChanges: false,
    bindingConflictCount: 2,
    selectedPolicyName: 'default-policy'
  });
  assert.ok(bindingActions.some(item => item.includes('2 组绑定冲突')));
});
