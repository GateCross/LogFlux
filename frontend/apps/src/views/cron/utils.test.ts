import { describe, expect, it } from 'vitest';

import {
  formatBytes,
  formatScriptMode,
  formatStatus,
  formatTriggerMode,
  getScriptFileValidationError,
  shortenHash,
  statusColor,
  taskStatusText,
} from './utils';

describe('cron/utils', () => {
  it('formatScriptMode maps file and inline', () => {
    expect(formatScriptMode('file')).toBe('上传脚本');
    expect(formatScriptMode('inline')).toBe('手写脚本');
    expect(formatScriptMode('other')).toBe('手写脚本');
  });

  it('formatStatus / taskStatusText / statusColor cover known codes', () => {
    expect(formatStatus(0)).toBe('运行中');
    expect(formatStatus(1)).toBe('成功');
    expect(formatStatus(2)).toBe('失败');
    expect(formatStatus(3)).toBe('超时');
    expect(formatStatus(99)).toBe('未知');
    expect(taskStatusText(1)).toBe('启用');
    expect(taskStatusText(0)).toBe('禁用');
    expect(statusColor(1)).toBe('green');
    expect(statusColor(99)).toBe('default');
  });

  it('formatTriggerMode and formatBytes edges', () => {
    expect(formatTriggerMode('schedule')).toBe('定时触发');
    expect(formatTriggerMode('manual')).toBe('手动触发');
    expect(formatTriggerMode('')).toBe('-');
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(1024)).toBe('1.0 KB');
  });

  it('shortenHash and getScriptFileValidationError boundaries', () => {
    expect(shortenHash('')).toBe('-');
    expect(shortenHash('abc')).toBe('abc');
    expect(shortenHash('0123456789abcdef0123')).toBe('01234567...cdef0123');

    const ok = new File(['echo hi'], 'run.sh', { type: 'text/plain' });
    expect(getScriptFileValidationError(ok)).toBeNull();

    const notSh = new File(['x'], 'run.txt', { type: 'text/plain' });
    expect(getScriptFileValidationError(notSh)).toBe('仅支持上传 .sh 脚本文件');

    const big = new File([new Uint8Array(1024 * 1024 + 1)], 'big.sh', {
      type: 'text/plain',
    });
    expect(getScriptFileValidationError(big)).toBe('脚本文件不能超过 1 MiB');
  });
});
