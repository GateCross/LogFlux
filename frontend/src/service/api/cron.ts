import { request } from '../request';

export type CronScriptMode = 'inline' | 'file';
export type CronTriggerMode = 'manual' | 'schedule' | string;

export interface CronTaskItem {
  id: number;
  name: string;
  schedule: string;
  script: string;
  scriptMode: CronScriptMode;
  currentFileId: number;
  currentFileName: string;
  currentFileVersion: number;
  currentFilePath: string;
  currentFileSha256: string;
  status: number;
  timeout: number;
  nextRun: string;
  createdAt: string;
  updatedAt: string;
}

export interface CronTaskFileItem {
  id: number;
  taskId: number;
  version: number;
  originalName: string;
  storedName: string;
  filePath: string;
  sizeBytes: number;
  sha256: string;
  isCurrent: boolean;
  createdAt: string;
}

export interface CronLogItem {
  id: number;
  taskId: number;
  taskName: string;
  startTime: string;
  endTime: string;
  status: number;
  exitCode: number;
  output: string;
  error: string;
  duration: number;
  triggerMode: CronTriggerMode;
  scriptMode: CronScriptMode;
  scriptSnapshot: string;
  scriptFileId: number;
  scriptFileVersion: number;
  scriptFileName: string;
  scriptFilePath: string;
  scriptFileSha256: string;
}

export interface CronTaskListResp {
  list: CronTaskItem[];
  total: number;
}

export interface CronTaskFileListResp {
  list: CronTaskFileItem[];
  total: number;
}

export interface CronLogListResp {
  list: CronLogItem[];
  total: number;
}

export interface CronTaskFormPayload {
  name: string;
  schedule: string;
  scriptMode?: CronScriptMode;
  script?: string;
  status: number;
  timeout: number;
}

export interface CronTaskUpdatePayload extends Partial<CronTaskFormPayload> {
  status?: number;
  timeout?: number;
}

export function fetchCronTaskList(params?: { page?: number; pageSize?: number; name?: string }) {
  return request<CronTaskListResp>({
    url: '/api/cron/task',
    method: 'get',
    params
  });
}

export function createCronTask(data: CronTaskFormPayload) {
  return request<void>({
    url: '/api/cron/task',
    method: 'post',
    data
  });
}

export function updateCronTask(id: number, data: CronTaskUpdatePayload) {
  return request<void>({
    url: `/api/cron/task/${id}`,
    method: 'put',
    data
  });
}

export function deleteCronTask(id: number) {
  return request<void>({
    url: `/api/cron/task/${id}`,
    method: 'delete'
  });
}

export function triggerCronTask(id: number) {
  return request<void>({
    url: `/api/cron/task/${id}/trigger`,
    method: 'post'
  });
}

export function fetchCronTaskScriptHistory(taskId: number, params?: { page?: number; pageSize?: number }) {
  return request<CronTaskFileListResp>({
    url: `/api/cron/task/${taskId}/script/history`,
    method: 'get',
    params
  });
}

export function uploadCronTaskScript(taskId: number, data: FormData) {
  return request<void>({
    url: `/api/cron/task/${taskId}/script`,
    method: 'post',
    data
  });
}

export function activateCronTaskScript(taskId: number, fileId: number) {
  return request<void>({
    url: `/api/cron/task/${taskId}/script/${fileId}/activate`,
    method: 'post'
  });
}

export function fetchCronLogList(params?: { page?: number; pageSize?: number; taskId?: number; status?: number }) {
  return request<CronLogListResp>({
    url: '/api/cron/log',
    method: 'get',
    params
  });
}

export function fetchCronLogDetail(id: number) {
  return request<CronLogItem>({
    url: `/api/cron/log/${id}`,
    method: 'get'
  });
}
