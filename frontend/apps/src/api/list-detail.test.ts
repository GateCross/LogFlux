/** list-detail 辅助函数单测 */

import { describe, expect, it } from 'vitest';

import {
  isListDetailEmpty,
  listDetailErrorMessage,
  suppressGlobalErrorToast,
  toListDetailErrorMessage,
  withListDetailErrorMode,
} from './list-detail';

describe('suppressGlobalErrorToast / withListDetailErrorMode', () => {
  it('exports suppress API (errorMessageMode: none)', () => {
    expect(suppressGlobalErrorToast).toEqual({ errorMessageMode: 'none' });
  });

  it('withListDetailErrorMode forces errorMessageMode none without other fields', () => {
    const cfg = withListDetailErrorMode();
    expect(cfg.errorMessageMode).toBe('none');
  });

  it('withListDetailErrorMode merges params and always overrides mode to none', () => {
    const cfg = withListDetailErrorMode({
      params: { page: 1 },
      // 强制 none
      errorMessageMode: 'message' as 'none',
    });
    expect(cfg.errorMessageMode).toBe('none');
    expect(cfg.params).toEqual({ page: 1 });
  });

  it('does not invent fictional suppress fields (only errorMessageMode)', () => {
    const keys = Object.keys(suppressGlobalErrorToast);
    expect(keys).toEqual(['errorMessageMode']);
    expect('showErrorMessage' in suppressGlobalErrorToast).toBe(false);
    expect('silent' in suppressGlobalErrorToast).toBe(false);
  });
});

describe('listDetailErrorMessage / toListDetailErrorMessage', () => {
  it('extracts response.data.message for inline Alert (happy)', () => {
    const msg = listDetailErrorMessage(
      { response: { data: { message: '后端拒绝' } } },
      '加载失败',
    );
    expect(msg).toBe('后端拒绝');
  });

  it('falls back when no extractable message (boundary)', () => {
    expect(listDetailErrorMessage({}, '加载失败')).toBe('加载失败');
    expect(listDetailErrorMessage(null, '加载失败')).toBe('加载失败');
  });

  it('toListDetailErrorMessage returns null when no error (no false Alert)', () => {
    expect(toListDetailErrorMessage(null, '加载失败')).toBeNull();
    expect(toListDetailErrorMessage(undefined, '加载失败')).toBeNull();
  });

  it('toListDetailErrorMessage maps error to Chinese string when present', () => {
    expect(
      toListDetailErrorMessage(
        { response: { data: { msg: '限流' } } },
        '加载失败',
      ),
    ).toBe('限流');
  });
});

describe('isListDetailEmpty', () => {
  it('treats null/undefined/empty array as empty', () => {
    expect(isListDetailEmpty(null)).toBe(true);
    expect(isListDetailEmpty(undefined)).toBe(true);
    expect(isListDetailEmpty([])).toBe(true);
  });

  it('treats non-empty array as not empty', () => {
    expect(isListDetailEmpty([{ id: 1 }])).toBe(false);
  });

  it('supports { list } envelope used by paginated APIs', () => {
    expect(isListDetailEmpty({ list: [], total: 0 })).toBe(true);
    expect(isListDetailEmpty({ list: [{ id: 1 }], total: 1 })).toBe(false);
    expect(isListDetailEmpty({ list: null })).toBe(true);
  });
});

describe('no double-toast contract', () => {
  it('list-detail helpers never export message/toast', async () => {
    const mod = await import('./list-detail');
    expect(mod.suppressGlobalErrorToast.errorMessageMode).toBe('none');
    expect(typeof mod.listDetailErrorMessage).toBe('function');
    expect(typeof mod.withListDetailErrorMode).toBe('function');
    // 无 toast API 导出
    expect((mod as Record<string, unknown>).message).toBeUndefined();
    expect((mod as Record<string, unknown>).toast).toBeUndefined();
  });
});
