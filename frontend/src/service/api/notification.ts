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
    status: number;
    error: string;
    retryCount: number;
    sentAt: string;
    createdAt: string;
}

export function getChannelList() {
    return request<any>({ url: '/api/notification/channel', method: 'get' });
}

export function createChannel(data: any) {
    return request<any>({ url: '/api/notification/channel', method: 'post', data });
}

export function updateChannel(id: number, data: any) {
    return request<any>({ url: `/api/notification/channel/${id}`, method: 'put', data });
}

export function deleteChannel(id: number) {
    return request<any>({ url: `/api/notification/channel/${id}`, method: 'delete' });
}

export function testChannel(id: number) {
    return request<any>({ url: '/api/notification/channel/test', method: 'post', data: { id } });
}

export function getRuleList() {
    return request<any>({ url: '/api/notification/rule', method: 'get' });
}

export function createRule(data: any) {
    return request<any>({ url: '/api/notification/rule', method: 'post', data });
}

export function updateRule(id: number, data: any) {
    return request<any>({ url: `/api/notification/rule/${id}`, method: 'put', data });
}

export function deleteRule(id: number) {
    return request<any>({ url: `/api/notification/rule/${id}`, method: 'delete' });
}

export function getTemplateList() {
    return request<any>({ url: '/api/notification/template', method: 'get' });
}

export function createTemplate(data: any) {
    return request<any>({ url: '/api/notification/template', method: 'post', data });
}

export function updateTemplate(id: number, data: any) {
    return request<any>({ url: `/api/notification/template/${id}`, method: 'put', data });
}

export function deleteTemplate(id: number) {
    return request<any>({ url: `/api/notification/template/${id}`, method: 'delete' });
}

export function previewTemplate(data: any) {
    return request<any>({ url: '/api/notification/template/preview', method: 'post', data });
}

export function getLogList(params: any) {
    return request<any>({ url: '/api/notification/log', method: 'get', params });
}

export function getUnreadNotifications() {
    return request<any>({ url: '/api/notification/unread', method: 'get' });
}

export function readNotification(id: number) {
    return request<any>({ url: `/api/notification/read/${id}`, method: 'post' });
}
