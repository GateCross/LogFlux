/**
 * Service_API: Notification channels, rules, templates & logs (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/notification.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

// ─── Types ───────────────────────────────────────────────────────────────────

export interface ChannelItem {
  id: number;
  name: string;
  type: string;
  enabled: boolean;
  config: string;
  events: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface WebhookHeaderItem {
  key: string;
  value: string;
}

export interface WebhookBodyFieldItem {
  key: string;
  source: 'title' | 'content' | 'message' | 'level' | 'type' | 'timestamp' | 'data' | 'custom';
  customValue: string;
}

export interface WebhookConfigForm {
  url: string;
  method: string;
  payload_mode: 'default' | 'message_api';
  api_key: string;
  api_key_header: string;
  title_field: string;
  content_field: string;
  headers: WebhookHeaderItem[];
  body_fields: WebhookBodyFieldItem[];
}

export interface TestChannelPayload {
  id: number;
  title?: string;
  content?: string;
}

export interface RuleItem {
  id: number;
  name: string;
  enabled: boolean;
  ruleType: string;
  eventType: string;
  condition: string;
  channelIds: number[];
  template: string;
  silenceDuration: number;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface TemplateItem {
  id: number;
  name: string;
  format: string;
  content: string;
  type: string;
  createdAt: string;
  updatedAt: string;
}

export interface LogItem {
  id: number;
  eventId: string;
  eventType: string;
  title: string;
  message: string;
  level: string;
  channelId: number;
  ruleId: number;

  /** logs dimension: 0=pending, 1=sending, 2=success, 3=failed */
  status: number;
  error: string;
  retryCount: number;
  sentAt: string;
  createdAt: string;

  /** jobs dimension: queued/processing/succeeded/failed */
  jobStatus: string;
  jobRetryCount: number;
  nextRunAt: string;
  lastError: string;
}

export interface ChannelListResp {
  list: ChannelItem[];
}

export interface RuleListResp {
  list: RuleItem[];
}

export interface TemplateListResp {
  list: TemplateItem[];
}

export interface LogListResp {
  list: LogItem[];
  total: number;
}

export interface UnreadNotificationItem {
  id: number;
  title: string;
  content: string;
  level: string;
  createdAt: string;
}

export interface UnreadNotificationResp {
  list: UnreadNotificationItem[];
  total: number;
}

export interface PreviewTemplateResp {
  content: string;
}

// ─── Functions: Channels ─────────────────────────────────────────────────────

/** GET /api/notification/channel - Fetch notification channel list. */
export function getChannelList(): Promise<FlatResponse<ChannelListResp>> {
  return request<ChannelListResp>({ url: '/api/notification/channel', method: 'get' });
}

/** POST /api/notification/channel - Create a notification channel. */
export function createChannel(
  data: Omit<ChannelItem, 'id' | 'createdAt' | 'updatedAt'>,
): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/notification/channel', method: 'post', data });
}

/** PUT /api/notification/channel/:id - Update a notification channel. */
export function updateChannel(
  id: number,
  data: Partial<Omit<ChannelItem, 'id' | 'createdAt' | 'updatedAt'>>,
): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/notification/channel/${id}`, method: 'put', data });
}

/** DELETE /api/notification/channel/:id - Delete a notification channel. */
export function deleteChannel(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/notification/channel/${id}`, method: 'delete' });
}

/** POST /api/notification/channel/test - Test a notification channel. */
export function testChannel(data: TestChannelPayload): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/notification/channel/test', method: 'post', data });
}

// ─── Functions: Rules ────────────────────────────────────────────────────────

/** GET /api/notification/rule - Fetch notification rule list. */
export function getRuleList(): Promise<FlatResponse<RuleListResp>> {
  return request<RuleListResp>({ url: '/api/notification/rule', method: 'get' });
}

/** POST /api/notification/rule - Create a notification rule. */
export function createRule(
  data: Omit<RuleItem, 'id' | 'createdAt' | 'updatedAt'>,
): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/notification/rule', method: 'post', data });
}

/** PUT /api/notification/rule/:id - Update a notification rule. */
export function updateRule(
  id: number,
  data: Partial<Omit<RuleItem, 'id' | 'createdAt' | 'updatedAt'>>,
): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/notification/rule/${id}`, method: 'put', data });
}

/** DELETE /api/notification/rule/:id - Delete a notification rule. */
export function deleteRule(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/notification/rule/${id}`, method: 'delete' });
}

// ─── Functions: Templates ────────────────────────────────────────────────────

/** GET /api/notification/template - Fetch notification template list. */
export function getTemplateList(): Promise<FlatResponse<TemplateListResp>> {
  return request<TemplateListResp>({ url: '/api/notification/template', method: 'get' });
}

/** POST /api/notification/template - Create a notification template. */
export function createTemplate(
  data: Omit<TemplateItem, 'id' | 'createdAt' | 'updatedAt'>,
): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/notification/template', method: 'post', data });
}

/** PUT /api/notification/template/:id - Update a notification template. */
export function updateTemplate(
  id: number,
  data: Partial<Omit<TemplateItem, 'id' | 'createdAt' | 'updatedAt'>>,
): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/notification/template/${id}`, method: 'put', data });
}

/** DELETE /api/notification/template/:id - Delete a notification template. */
export function deleteTemplate(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/notification/template/${id}`, method: 'delete' });
}

/** POST /api/notification/template/preview - Preview a notification template. */
export function previewTemplate(
  data: Pick<TemplateItem, 'format' | 'content'> & { type?: string; data?: Record<string, any> },
): Promise<FlatResponse<PreviewTemplateResp>> {
  return request<PreviewTemplateResp>({
    url: '/api/notification/template/preview',
    method: 'post',
    data,
  });
}

// ─── Functions: Logs ─────────────────────────────────────────────────────────

/** GET /api/notification/log - Fetch notification log list. */
export function getLogList(params: {
  page: number;
  pageSize: number;
  [key: string]: any;
}): Promise<FlatResponse<LogListResp>> {
  return request<LogListResp>({ url: '/api/notification/log', method: 'get', params });
}

/** DELETE /api/notification/log/:id - Delete a notification log. */
export function deleteNotificationLog(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/notification/log/${id}`, method: 'delete' });
}

/** POST /api/notification/log/batch-delete - Batch delete notification logs. */
export function batchDeleteNotificationLogs(ids: number[]): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/notification/log/batch-delete', method: 'post', data: { ids } });
}

/** POST /api/notification/log/clear - Clear all notification logs. */
export function clearNotificationLogs(): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/notification/log/clear', method: 'post' });
}

// ─── Functions: Unread / Read ────────────────────────────────────────────────

/** GET /api/notification/unread - Fetch unread notifications. */
export function getUnreadNotifications(): Promise<FlatResponse<UnreadNotificationResp>> {
  return request<UnreadNotificationResp>({ url: '/api/notification/unread', method: 'get' });
}

/** POST /api/notification/read/:id - Mark a notification as read. */
export function readNotification(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/notification/read/${id}`, method: 'post' });
}

/** POST /api/notification/read/all - Mark all notifications as read. */
export function readAllNotifications(): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/notification/read/all', method: 'post' });
}
