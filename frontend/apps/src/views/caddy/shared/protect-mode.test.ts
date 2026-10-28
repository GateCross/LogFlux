import { describe, expect, it } from 'vitest';

import { PROTECT_MODE_OPTIONS, protectModeLabel, protectModeOption } from './protect-mode';

describe('站点防护模式', () => {
  it('四种模式与顺序固定：防护 → 观察 → 维护 → 关闭', () => {
    expect(PROTECT_MODE_OPTIONS.map((item) => item.value)).toEqual([
      'protect',
      'detect',
      'maintenance',
      'off',
    ]);
    expect(PROTECT_MODE_OPTIONS.map((item) => item.label)).toEqual([
      '防护模式',
      '观察模式',
      '维护模式',
      '关闭',
    ]);
  });

  it('每种模式都写清后果与配色', () => {
    expect(protectModeOption('protect')).toMatchObject({
      label: '防护模式',
      tone: 'success',
      description: '发现攻击后将自动拦截，并记录攻击事件',
    });
    expect(protectModeOption('detect')).toMatchObject({
      label: '观察模式',
      tone: 'warning',
      description: '发现攻击后仅记录攻击事件，不拦截',
    });
    expect(protectModeOption('maintenance')).toMatchObject({
      label: '维护模式',
      tone: 'danger',
      description: '展示维护页面，任何人都将无法访问您的应用',
    });
    expect(protectModeOption('off')).toMatchObject({ label: '关闭', tone: 'muted' });
  });

  it('未知值与空值回落到未防护，不抛错也不猜测', () => {
    for (const value of ['blocking', '', null, undefined]) {
      expect(protectModeOption(value)).toMatchObject({ value: 'off', label: '未防护' });
    }
  });

  it('label 取选中项名称', () => {
    expect(protectModeLabel('protect')).toBe('防护模式');
    expect(protectModeLabel('detect')).toBe('观察模式');
    expect(protectModeLabel('maintenance')).toBe('维护模式');
    expect(protectModeLabel('off')).toBe('关闭');
  });
});
