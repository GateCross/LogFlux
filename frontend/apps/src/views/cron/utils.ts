/** Cron 页纯展示/校验（无 UI 副作用） */

export function formatScriptMode(mode: string) {
  return mode === 'file' ? '上传脚本' : '手写脚本';
}

export function formatStatus(status: number) {
  switch (status) {
    case 0:
      return '运行中';
    case 1:
      return '成功';
    case 2:
      return '失败';
    case 3:
      return '超时';
    default:
      return '未知';
  }
}

export function taskStatusText(status: number) {
  return status === 1 ? '启用' : '禁用';
}

export function statusColor(status: number) {
  switch (status) {
    case 0:
      return 'blue';
    case 1:
      return 'green';
    case 2:
      return 'red';
    case 3:
      return 'orange';
    default:
      return 'default';
  }
}

export function formatTriggerMode(mode: string) {
  if (mode === 'schedule') return '定时触发';
  if (mode === 'manual') return '手动触发';
  return mode || '-';
}

export function formatBytes(bytes: number) {
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = bytes;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

export function shortenHash(hash: string) {
  if (!hash) return '-';
  return hash.length <= 16 ? hash : `${hash.slice(0, 8)}...${hash.slice(-8)}`;
}

/** 校验脚本文件：≤1MiB 且 .sh。失败返回中文错误文案，成功返回 null。 */
export function getScriptFileValidationError(file: File): string | null {
  if (file.size > 1024 * 1024) {
    return '脚本文件不能超过 1 MiB';
  }
  if (!file.name.toLowerCase().endsWith('.sh')) {
    return '仅支持上传 .sh 脚本文件';
  }
  return null;
}
