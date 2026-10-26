/**
 * HTTP 状态码到 Tag 颜色的共享映射，访问日志页与站点日志面板共用。
 */

/**
 * 状态码分档着色：2xx 绿、3xx 蓝、4xx 橙、5xx 红。
 * 空值与 0 保持中性灰；1xx 极少出现在访问日志里，与 2xx 同按绿处理。
 */
export function statusTagColor(status?: number): string {
  if (status == null || status === 0) return 'default';
  if (status >= 500) return 'red';
  if (status >= 400) return 'orange';
  if (status >= 300) return 'blue';
  return 'green';
}
