import { requestClient } from '#/api/request';

import { listOf } from './_utils';

export namespace NotificationApi {
  // ─── Channel ───────────────────────────────────────────────

  /** 通知渠道类型 */
  export type ChannelType = string;

  /** 通知渠道状态 */
  export type ChannelStatus = 'disabled' | 'enabled';

  /** 通知渠道 */
  export interface Channel {
    description?: string;
    config: Record<string, any>;
    createdAt: string;
    enabled?: boolean;
    events?: string;
    id: string;
    name: string;
    status: ChannelStatus;
    type: ChannelType;
    updatedAt: string;
  }

  /** 创建/更新渠道参数 */
  export interface ChannelParams {
    config: Record<string, any> | string;
    description?: string;
    enabled?: boolean;
    events?: string;
    name: string;
    status?: ChannelStatus;
    type: ChannelType;
  }

  /** 测试渠道参数 */
  export interface ChannelTestParams {
    channelId: string;
    content?: string;
    message?: string;
    title?: string;
  }

  // ─── Rule ──────────────────────────────────────────────────

  /** 通知规则 */
  export interface Rule {
    channelId: string;
    channelIds?: number[];
    conditions: Record<string, any>;
    condition?: string;
    createdAt: string;
    description?: string;
    enabled: boolean;
    eventType?: string;
    id: string;
    level: string;
    name: string;
    ruleType?: string;
    silenceDuration?: number;
    template?: string;
    templateId?: string;
    updatedAt: string;
  }

  /** 创建/更新规则参数 */
  export interface RuleParams {
    channelId?: string;
    channelIds?: number[];
    condition?: string;
    conditions?: Record<string, any>;
    description?: string;
    enabled?: boolean;
    eventType?: string;
    level?: string;
    name: string;
    ruleType?: string;
    silenceDuration?: number;
    template?: string;
    templateId?: string;
  }

  // ─── Template ──────────────────────────────────────────────

  /** 通知模板 */
  export interface Template {
    content: string;
    createdAt: string;
    format?: string;
    id: string;
    name: string;
    type: string;
    updatedAt: string;
  }

  /** 创建/更新模板参数 */
  export interface TemplateParams {
    content: string;
    format?: string;
    name: string;
    type: string;
  }

  /** 模板预览参数 */
  export interface TemplatePreviewParams {
    content: string;
    data?: Record<string, any>;
    format?: string;
    variables?: Record<string, any>;
  }

  /** 模板预览结果 */
  export interface TemplatePreviewResult {
    content?: string;
    rendered: string;
  }

  // ─── Log ───────────────────────────────────────────────────

  /** 通知日志 */
  export interface NotificationLog {
    channelId: number | string;
    channelName?: string;
    content?: string;
    createdAt: string;
    errorMessage?: string;
    error?: string;
    eventType?: string;
    id: number | string;
    jobRetryCount?: number;
    jobStatus?: string;
    lastError?: string;
    level?: string;
    message?: string;
    nextRunAt?: string;
    retryCount?: number;
    ruleId?: number | string;
    ruleName?: string;
    sentAt?: string;
    status: 'failed' | 'pending' | 'sending' | 'sent' | 'success' | number;
    title?: string;
  }

  /** 通知日志分页查询参数 */
  export interface LogQueryParams {
    channelId?: string;
    current?: number;
    endTime?: string;
    page?: number;
    pageSize?: number;
    startTime?: string;
    status?: string;
  }

  /** 批量删除日志参数 */
  export interface BatchDeleteLogParams {
    ids: Array<number | string>;
  }

  // ─── Unread / Read ─────────────────────────────────────────

  /** 未读通知 */
  export interface UnreadNotification {
    content: string;
    createdAt: string;
    id: string;
    read: boolean;
    title: string;
    type: string;
  }
}

interface ListResp<T> {
  list: T[];
  total?: number;
}

function parseConfig(value: Record<string, any> | string | undefined) {
  if (!value) return {};
  if (typeof value !== 'string') return value;
  try {
    return JSON.parse(value);
  } catch {
    return {};
  }
}

function normalizeChannel(item: any): NotificationApi.Channel {
  return {
    ...item,
    config: parseConfig(item.config),
    status: item.status ?? (item.enabled ? 'enabled' : 'disabled'),
  };
}

function channelPayload(data: NotificationApi.ChannelParams) {
  const config =
    typeof data.config === 'string'
      ? data.config
      : JSON.stringify(data.config ?? {});

  return {
    config,
    description: data.description ?? '',
    enabled: data.enabled ?? data.status !== 'disabled',
    events: data.events ?? JSON.stringify(['*']),
    name: data.name,
    type: data.type,
  };
}

function parseJsonRecord(value: Record<string, any> | string | undefined) {
  if (!value) return {};
  if (typeof value !== 'string') return value;
  try {
    return JSON.parse(value);
  } catch {
    return {};
  }
}

function normalizeRule(item: any): NotificationApi.Rule {
  const channelIds = Array.isArray(item.channelIds) ? item.channelIds : [];
  const firstChannelId = item.channelId ?? channelIds[0];
  return {
    ...item,
    channelId: firstChannelId === undefined ? '' : String(firstChannelId),
    conditions: parseJsonRecord(item.conditions ?? item.condition),
    enabled: item.enabled ?? true,
    level: item.level ?? item.eventType ?? 'error',
    templateId: item.templateId ?? item.template,
  };
}

function rulePayload(data: NotificationApi.RuleParams) {
  const condition = data.condition ?? JSON.stringify(data.conditions ?? {});
  const channelIds =
    data.channelIds ??
    (data.channelId ? [Number(data.channelId)].filter((id) => Number.isFinite(id)) : []);
  return {
    channelIds,
    condition,
    description: data.description ?? '',
    enabled: data.enabled ?? true,
    eventType: data.eventType ?? data.level ?? 'error',
    name: data.name,
    ruleType: data.ruleType ?? 'threshold',
    silenceDuration: data.silenceDuration ?? 300,
    template: data.template ?? data.templateId ?? '',
  };
}

// ─── Channel ─────────────────────────────────────────────────

/**
 * 获取通知渠道列表 — GET /notification/channel
 */
export async function getNotificationChannelsApi() {
  const resp = await requestClient.get<ListResp<any>>('/notification/channel');
  return listOf(resp).map(normalizeChannel);
}

/**
 * 创建通知渠道 — POST /notification/channel
 */
export async function createNotificationChannelApi(
  data: NotificationApi.ChannelParams,
) {
  return requestClient.post<NotificationApi.Channel>(
    '/notification/channel',
    channelPayload(data),
  );
}

/**
 * 更新通知渠道 — PUT /notification/channel/:id
 */
export async function updateNotificationChannelApi(
  id: string,
  data: NotificationApi.ChannelParams,
) {
  return requestClient.put<NotificationApi.Channel>(
    `/notification/channel/${id}`,
    channelPayload(data),
  );
}

/**
 * 删除通知渠道 — DELETE /notification/channel/:id
 */
export async function deleteNotificationChannelApi(id: string) {
  return requestClient.delete<void>(`/notification/channel/${id}`);
}

/**
 * 发送测试通知 — POST /notification/channel/test
 */
export async function testNotificationChannelApi(
  data: NotificationApi.ChannelTestParams,
) {
  return requestClient.post<void>('/notification/channel/test', {
    content: data.content ?? data.message,
    id: Number(data.channelId),
    title: data.title,
  });
}

// ─── Rule ────────────────────────────────────────────────────

/**
 * 获取通知规则列表 — GET /notification/rule
 */
export async function getNotificationRulesApi() {
  const resp = await requestClient.get<ListResp<NotificationApi.Rule>>(
    '/notification/rule',
  );
  return listOf(resp).map(normalizeRule);
}

/**
 * 创建通知规则 — POST /notification/rule
 */
export async function createNotificationRuleApi(
  data: NotificationApi.RuleParams,
) {
  return requestClient.post<NotificationApi.Rule>(
    '/notification/rule',
    rulePayload(data),
  );
}

/**
 * 更新通知规则 — PUT /notification/rule/:id
 */
export async function updateNotificationRuleApi(
  id: string,
  data: NotificationApi.RuleParams,
) {
  return requestClient.put<NotificationApi.Rule>(
    `/notification/rule/${id}`,
    rulePayload(data),
  );
}

/**
 * 删除通知规则 — DELETE /notification/rule/:id
 */
export async function deleteNotificationRuleApi(id: string) {
  return requestClient.delete<void>(`/notification/rule/${id}`);
}

// ─── Template ────────────────────────────────────────────────

/**
 * 获取通知模板列表 — GET /notification/template
 */
export async function getNotificationTemplatesApi() {
  const resp = await requestClient.get<ListResp<NotificationApi.Template>>(
    '/notification/template',
  );
  return listOf(resp);
}

/**
 * 创建通知模板 — POST /notification/template
 */
export async function createNotificationTemplateApi(
  data: NotificationApi.TemplateParams,
) {
  return requestClient.post<NotificationApi.Template>(
    '/notification/template',
    data,
  );
}

/**
 * 更新通知模板 — PUT /notification/template/:id
 */
export async function updateNotificationTemplateApi(
  id: string,
  data: NotificationApi.TemplateParams,
) {
  return requestClient.put<NotificationApi.Template>(
    `/notification/template/${id}`,
    data,
  );
}

/**
 * 删除通知模板 — DELETE /notification/template/:id
 */
export async function deleteNotificationTemplateApi(id: string) {
  return requestClient.delete<void>(`/notification/template/${id}`);
}

/**
 * 预览模板渲染结果 — POST /notification/template/preview
 */
export async function previewNotificationTemplateApi(
  data: NotificationApi.TemplatePreviewParams,
) {
  return requestClient.post<NotificationApi.TemplatePreviewResult>(
    '/notification/template/preview',
    {
      content: data.content,
      data: JSON.stringify(data.variables ?? data.data ?? {}),
      format: data.format ?? 'text',
    },
  );
}

// ─── Log ─────────────────────────────────────────────────────

/**
 * 获取通知日志列表（分页） — GET /notification/log
 */
export async function getNotificationLogsApi(
  params?: NotificationApi.LogQueryParams,
) {
  return requestClient.get<ListResp<NotificationApi.NotificationLog>>(
    '/notification/log',
    { params },
  );
}

/**
 * 删除单条通知日志 — DELETE /notification/log/:id
 */
export async function deleteNotificationLogApi(id: string) {
  return requestClient.delete<void>(`/notification/log/${id}`);
}

/**
 * 批量删除通知日志 — POST /notification/log/batch-delete
 */
export async function batchDeleteNotificationLogsApi(
  data: NotificationApi.BatchDeleteLogParams,
) {
  return requestClient.post<void>('/notification/log/batch-delete', data);
}

/**
 * 清空所有通知日志 — POST /notification/log/clear
 */
export async function clearNotificationLogsApi() {
  return requestClient.post<void>('/notification/log/clear');
}

// ─── Unread / Read ───────────────────────────────────────────

/**
 * 获取未读通知 — GET /notification/unread
 */
export async function getUnreadNotificationsApi() {
  const resp = await requestClient.get<ListResp<NotificationApi.UnreadNotification>>(
    '/notification/unread',
  );
  return listOf(resp);
}

/**
 * 标记单条通知为已读 — POST /notification/read/:id
 */
export async function markNotificationReadApi(id: string) {
  return requestClient.post<void>(`/notification/read/${id}`);
}

/**
 * 标记所有通知为已读 — POST /notification/read/all
 */
export async function markAllNotificationsReadApi() {
  return requestClient.post<void>('/notification/read/all');
}
