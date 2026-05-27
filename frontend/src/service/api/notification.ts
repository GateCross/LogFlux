import { request } from '../request';

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

  /** logs 维度: 0=pending,1=sending,2=success,3=failed */
  status: number;
  error: string;
  retryCount: number;
  sentAt: string;
  createdAt: string;

  /** jobs 维度: queued/processing/succeeded/failed */
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

export function getChannelList() {
  return request<ChannelListResp>({ url: '/api/notification/channel', method: 'get' });
}

export function createChannel(data: Omit<ChannelItem, 'id' | 'createdAt' | 'updatedAt'>) {
  return request<void>({ url: '/api/notification/channel', method: 'post', data });
}

export function updateChannel(id: number, data: Partial<Omit<ChannelItem, 'id' | 'createdAt' | 'updatedAt'>>) {
  return request<void>({ url: `/api/notification/channel/${id}`, method: 'put', data });
}

export function deleteChannel(id: number) {
  return request<void>({ url: `/api/notification/channel/${id}`, method: 'delete' });
}

export function testChannel(data: TestChannelPayload) {
  return request<void>({ url: '/api/notification/channel/test', method: 'post', data });
}

export function getRuleList() {
  return request<RuleListResp>({ url: '/api/notification/rule', method: 'get' });
}

export function createRule(data: Omit<RuleItem, 'id' | 'createdAt' | 'updatedAt'>) {
  return request<void>({ url: '/api/notification/rule', method: 'post', data });
}

export function updateRule(id: number, data: Partial<Omit<RuleItem, 'id' | 'createdAt' | 'updatedAt'>>) {
  return request<void>({ url: `/api/notification/rule/${id}`, method: 'put', data });
}

export function deleteRule(id: number) {
  return request<void>({ url: `/api/notification/rule/${id}`, method: 'delete' });
}

export function getTemplateList() {
  return request<TemplateListResp>({ url: '/api/notification/template', method: 'get' });
}

export function createTemplate(data: Omit<TemplateItem, 'id' | 'createdAt' | 'updatedAt'>) {
  return request<void>({ url: '/api/notification/template', method: 'post', data });
}

export function updateTemplate(id: number, data: Partial<Omit<TemplateItem, 'id' | 'createdAt' | 'updatedAt'>>) {
  return request<void>({ url: `/api/notification/template/${id}`, method: 'put', data });
}

export function deleteTemplate(id: number) {
  return request<void>({ url: `/api/notification/template/${id}`, method: 'delete' });
}

export interface PreviewTemplateResp {
  content: string;
}

export function previewTemplate(
  data: Pick<TemplateItem, 'format' | 'content'> & { type?: string; data?: Record<string, any> }
) {
  return request<PreviewTemplateResp>({ url: '/api/notification/template/preview', method: 'post', data });
}

export function getLogList(params: { page: number; pageSize: number; [key: string]: any }) {
  return request<LogListResp>({ url: '/api/notification/log', method: 'get', params });
}

export function deleteNotificationLog(id: number) {
  return request<void>({ url: `/api/notification/log/${id}`, method: 'delete' });
}

export function batchDeleteNotificationLogs(ids: number[]) {
  return request<void>({ url: '/api/notification/log/batch-delete', method: 'post', data: { ids } });
}

export function clearNotificationLogs() {
  return request<void>({ url: '/api/notification/log/clear', method: 'post' });
}

export function getUnreadNotifications() {
  return request<UnreadNotificationResp>({ url: '/api/notification/unread', method: 'get' });
}

export function readNotification(id: number) {
  return request<void>({ url: `/api/notification/read/${id}`, method: 'post' });
}

export function readAllNotifications() {
  return request<void>({ url: '/api/notification/read/all', method: 'post' });
}
