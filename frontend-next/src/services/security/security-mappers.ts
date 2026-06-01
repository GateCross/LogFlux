/**
 * Security 字段映射纯逻辑（移植自旧 Vue 版 `src/views/security/security-mappers.ts`）。
 *
 * 移植说明（task 11.1 / Req 13.7、17.1）：
 *  - 框架无关纯函数，无 Vue / Naive UI / 运行时耦合，逐字保留原有逻辑语义。
 *  - 仅将类型导入从旧 `@/service/api/*` 改为本目录共享类型 `./types`。
 *  - 标签字符串、tag 类型取值与兜底值（`-` 或原值）保持不变，保证相同取值产生与
 *    Vue 版相同的展示标签 / 标签类型（支撑 Property 13：Security 字段映射等价性）。
 *  - 标签类型返回值（'default' | 'warning' | 'success' | 'info' | 'error'）沿用旧版
 *    的 Naive UI 语义字符串，作为框架无关的取值保留；上层 UI 适配层负责映射到组件库。
 */
import type { WafPolicyEngineMode, WafPolicyRevisionStatus } from './types';
import type { WafPolicyFalsePositiveFeedbackItem } from './types';
import type { WafJobStatus, WafReleaseStatus } from './types';

export function mapPolicyFeedbackStatusLabel(status: string) {
  switch (
    String(status || '')
      .trim()
      .toLowerCase()
  ) {
    case 'confirmed':
      return '已确认';
    case 'resolved':
      return '已处理';
    default:
      return '待确认';
  }
}

export function mapPolicyFeedbackStatusTagType(status: string): 'default' | 'warning' | 'success' {
  switch (
    String(status || '')
      .trim()
      .toLowerCase()
  ) {
    case 'confirmed':
      return 'warning';
    case 'resolved':
      return 'success';
    default:
      return 'default';
  }
}

export function mapPolicyFeedbackSLAStatusLabel(row: WafPolicyFalsePositiveFeedbackItem) {
  if ((row.feedbackStatus || '') === 'resolved') {
    return '已解决';
  }
  return row.isOverdue ? '已超时' : '正常';
}

export function mapPolicyFeedbackSLAStatusTagType(
  row: WafPolicyFalsePositiveFeedbackItem
): 'default' | 'warning' | 'success' {
  if ((row.feedbackStatus || '') === 'resolved') {
    return 'success';
  }
  return row.isOverdue ? 'warning' : 'default';
}

export function mapReleaseStatusType(status: WafReleaseStatus) {
  switch (status) {
    case 'active':
      return 'success';
    case 'verified':
      return 'info';
    case 'failed':
      return 'error';
    case 'rolled_back':
      return 'warning';
    default:
      return 'default';
  }
}

export function mapPolicyEngineModeType(mode: WafPolicyEngineMode) {
  switch (mode) {
    case 'on':
      return 'error';
    case 'detectiononly':
      return 'warning';
    case 'off':
      return 'default';
    default:
      return 'default';
  }
}

export function mapPolicyRevisionStatusType(status: WafPolicyRevisionStatus) {
  switch (status) {
    case 'published':
      return 'success';
    case 'rolled_back':
      return 'warning';
    default:
      return 'default';
  }
}

export function mapJobStatusType(status: WafJobStatus) {
  switch (status) {
    case 'success':
      return 'success';
    case 'failed':
      return 'error';
    default:
      return 'warning';
  }
}

export function mapJobStatusLabel(status: string) {
  switch (status) {
    case 'running':
      return '执行中';
    case 'success':
      return '成功';
    case 'failed':
      return '失败';
    default:
      return status || '-';
  }
}

export function mapJobActionLabel(action: string) {
  switch (action) {
    case 'check':
      return '检查';
    case 'download':
      return '下载';
    case 'verify':
      return '校验';
    case 'activate':
      return '激活';
    case 'rollback':
      return '回滚';
    case 'engine_check':
      return '引擎检查';
    default:
      return action || '-';
  }
}

export function mapJobTriggerModeLabel(triggerMode: string) {
  switch (triggerMode) {
    case 'manual':
      return '手动';
    case 'upload':
      return '上传';
    case 'schedule':
      return '定时';
    case 'auto':
      return '自动';
    case 'system':
      return '系统';
    default:
      return triggerMode || '-';
  }
}

export function mapJobMessage(rawMessage: string) {
  const messageText = String(rawMessage || '').trim();
  if (!messageText) {
    return '-';
  }

  const exactMap: Record<string, string> = {
    'check success': '检查成功',
    'sync success': '同步成功',
    'upload success': '上传成功',
    'activate success': '激活成功',
    'rollback success': '回滚成功',
    'engine source check success': '引擎源检查成功'
  };

  if (exactMap[messageText]) {
    return exactMap[messageText];
  }

  const replacementRules: Array<[RegExp, string]> = [
    [/context deadline exceeded/gi, '请求超时'],
    [/i\/o timeout/gi, '网络超时'],
    [/invalid proxy url:/gi, '代理地址不合法：'],
    [/invalid url:/gi, '无效地址：'],
    [/only https url is allowed/gi, '仅支持 HTTPS 地址'],
    [/only https scheme is allowed/gi, '仅允许 HTTPS 协议'],
    [/proxy url scheme must be http or https/gi, '代理地址协议仅支持 http/https'],
    [/source not found/gi, '未找到更新源'],
    [/source is disabled/gi, '更新源已禁用'],
    [/source mode is not remote/gi, '更新源模式不是 remote'],
    [/source url is empty/gi, '更新源地址为空'],
    [/move package failed:/gi, '移动安装包失败：'],
    [/create release dir failed:/gi, '创建版本目录失败：'],
    [/create release failed:/gi, '创建版本记录失败：'],
    [/fetch failed:/gi, '下载失败：'],
    [/host not allowed:/gi, '源域名不在允许列表：'],
    [/unexpected status code:/gi, '下载返回异常状态码：'],
    [/write temp file failed:/gi, '写入临时文件失败：'],
    [/close temp file failed:/gi, '关闭临时文件失败：'],
    [/move temp file failed:/gi, '移动临时文件失败：'],
    [/prepare waf store failed:/gi, '准备 Waf 存储目录失败：']
  ];

  let localizedMessage = messageText;
  for (const [pattern, replacement] of replacementRules) {
    localizedMessage = localizedMessage.replace(pattern, replacement);
  }

  return localizedMessage;
}

export function formatRatePercent(value: number) {
  const numeric = Number(value || 0);
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return '0%';
  }
  return `${(numeric * 100).toFixed(2)}%`;
}

export function formatDateTime(date: Date) {
  const pad = (num: number) => String(num).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}
