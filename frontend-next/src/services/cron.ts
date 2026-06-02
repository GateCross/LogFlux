/**
 * Service_API: Cron task management (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/cron.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

// ─── Types ───────────────────────────────────────────────────────────────────

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

// ─── Functions ───────────────────────────────────────────────────────────────

/** GET /api/cron/task - Fetch cron task list. */
export function fetchCronTaskList(params?: {
  page?: number;
  pageSize?: number;
  name?: string;
}): Promise<FlatResponse<CronTaskListResp>> {
  return request<CronTaskListResp>({
    url: '/api/cron/task',
    method: 'get',
    params,
  });
}

/** POST /api/cron/task - Create a new cron task (JSON body). */
export function createCronTask(data: CronTaskFormPayload): Promise<FlatResponse<void>> {
  return request<void>({
    url: '/api/cron/task',
    method: 'post',
    data,
  });
}

/** POST /api/cron/task - Create a new cron task with file upload (FormData). */
export function createCronTaskWithFile(data: FormData): Promise<FlatResponse<void>> {
  return request<void>({
    url: '/api/cron/task',
    method: 'post',
    data,
  });
}

/** PUT /api/cron/task/:id - Update an existing cron task. */
export function updateCronTask(id: number, data: CronTaskUpdatePayload): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/cron/task/${id}`,
    method: 'put',
    data,
  });
}

/** DELETE /api/cron/task/:id - Delete a cron task. */
export function deleteCronTask(id: number): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/cron/task/${id}`,
    method: 'delete',
  });
}

/** POST /api/cron/task/:id/trigger - Manually trigger a cron task. */
export function triggerCronTask(id: number): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/cron/task/${id}/trigger`,
    method: 'post',
  });
}

/** GET /api/cron/task/:taskId/script/history - Fetch script file history for a task. */
export function fetchCronTaskScriptHistory(
  taskId: number,
  params?: { page?: number; pageSize?: number },
): Promise<FlatResponse<CronTaskFileListResp>> {
  return request<CronTaskFileListResp>({
    url: `/api/cron/task/${taskId}/script/history`,
    method: 'get',
    params,
  });
}

/** POST /api/cron/task/:taskId/script - Upload a script file for a task (FormData). */
export function uploadCronTaskScript(taskId: number, data: FormData): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/cron/task/${taskId}/script`,
    method: 'post',
    data,
  });
}

/** POST /api/cron/task/:taskId/script/:fileId/activate - Activate a script file version. */
export function activateCronTaskScript(taskId: number, fileId: number): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/cron/task/${taskId}/script/${fileId}/activate`,
    method: 'post',
  });
}

/** GET /api/cron/log - Fetch cron execution log list. */
export function fetchCronLogList(params?: {
  page?: number;
  pageSize?: number;
  taskId?: number;
  status?: number;
}): Promise<FlatResponse<CronLogListResp>> {
  return request<CronLogListResp>({
    url: '/api/cron/log',
    method: 'get',
    params,
  });
}

/** GET /api/cron/log/:id - Fetch cron log detail. */
export function fetchCronLogDetail(id: number): Promise<FlatResponse<CronLogItem>> {
  return request<CronLogItem>({
    url: `/api/cron/log/${id}`,
    method: 'get',
  });
}
