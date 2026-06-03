import { requestClient } from '#/api/request';

export namespace NotificationApi {
  // ─── Channel ───────────────────────────────────────────────

  /** 通知渠道类型 */
  export type ChannelType = 'dingtalk' | 'email' | 'slack' | 'webhook' | 'wechat';

  /** 通知渠道状态 */
  export type ChannelStatus = 'disabled' | 'enabled';

  /** 通知渠道 */
  export interface Channel {
    config: Record<string, any>;
    createdAt: string;
    id: string;
    name: string;
    status: ChannelStatus;
    type: ChannelType;
    updatedAt: string;
  }

  /** 创建/更新渠道参数 */
  export interface ChannelParams {
    config: Record<string, any>;
    name: string;
    status?: ChannelStatus;
    type: ChannelType;
  }

  /** 测试渠道参数 */
  export interface ChannelTestParams {
    channelId: string;
    message?: string;
  }

  // ─── Rule ──────────────────────────────────────────────────

  /** 通知规则 */
  export interface Rule {
    channelId: string;
    conditions: Record<string, any>;
    createdAt: string;
    enabled: boolean;
    id: string;
    level: string;
    name: string;
    templateId?: string;
    updatedAt: string;
  }

  /** 创建/更新规则参数 */
  export interface RuleParams {
    channelId: string;
    conditions: Record<string, any>;
    enabled?: boolean;
    level: string;
    name: string;
    templateId?: string;
  }

  // ─── Template ──────────────────────────────────────────────

  /** 通知模板 */
  export interface Template {
    content: string;
    createdAt: string;
    id: string;
    name: string;
    type: string;
    updatedAt: string;
  }

  /** 创建/更新模板参数 */
  export interface TemplateParams {
    content: string;
    name: string;
    type: string;
  }

  /** 模板预览参数 */
  export interface TemplatePreviewParams {
    content: string;
    variables?: Record<string, any>;
  }

  /** 模板预览结果 */
  export interface TemplatePreviewResult {
    rendered: string;
  }

  // ─── Log ───────────────────────────────────────────────────

  /** 通知日志 */
  export interface NotificationLog {
    channelId: string;
    channelName: string;
    content: string;
    createdAt: string;
    errorMessage?: string;
    id: string;
    ruleId?: string;
    ruleName?: string;
    status: 'failed' | 'pending' | 'sent';
  }

  /** 通知日志分页查询参数 */
  export interface LogQueryParams {
    channelId?: string;
    current?: number;
    endTime?: string;
    pageSize?: number;
    startTime?: string;
    status?: string;
  }

  /** 批量删除日志参数 */
  export interface BatchDeleteLogParams {
    ids: string[];
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

// ─── Channel ─────────────────────────────────────────────────

/**
 * 获取通知渠道列表 — GET /notification/channel
 */
export async function getNotificationChannelsApi() {
  return requestClient.get<NotificationApi.Channel[]>('/notification/channel');
}

/**
 * 创建通知渠道 — POST /notification/channel
 */
export async function createNotificationChannelApi(
  data: NotificationApi.ChannelParams,
) {
  return requestClient.post<NotificationApi.Channel>(
    '/notification/channel',
    data,
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
    data,
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
  return requestClient.post<void>('/notification/channel/test', data);
}

// ─── Rule ────────────────────────────────────────────────────

/**
 * 获取通知规则列表 — GET /notification/rule
 */
export async function getNotificationRulesApi() {
  return requestClient.get<NotificationApi.Rule[]>('/notification/rule');
}

/**
 * 创建通知规则 — POST /notification/rule
 */
export async function createNotificationRuleApi(
  data: NotificationApi.RuleParams,
) {
  return requestClient.post<NotificationApi.Rule>(
    '/notification/rule',
    data,
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
    data,
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
  return requestClient.get<NotificationApi.Template[]>(
    '/notification/template',
  );
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
    data,
  );
}

// ─── Log ─────────────────────────────────────────────────────

/**
 * 获取通知日志列表（分页） — GET /notification/log
 */
export async function getNotificationLogsApi(
  params?: NotificationApi.LogQueryParams,
) {
  return requestClient.get<NotificationApi.NotificationLog[]>(
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
  return requestClient.get<NotificationApi.UnreadNotification[]>(
    '/notification/unread',
  );
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
