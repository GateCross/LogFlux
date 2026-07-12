import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

import type { ListResult } from './_utils';

import { listOf, totalOf } from './_utils';

export namespace CronApi {
  export type ScriptMode = 'file' | 'inline';
  export type TriggerMode = 'manual' | 'schedule' | string;

  export interface Task {
    createdAt: string;
    currentFileId: number;
    currentFileName: string;
    currentFilePath: string;
    currentFileSha256: string;
    currentFileVersion: number;
    id: number;
    name: string;
    nextRun: string;
    schedule: string;
    script: string;
    scriptMode: ScriptMode;
    status: number;
    timeout: number;
    updatedAt: string;
  }

  export interface TaskPayload {
    name: string;
    schedule: string;
    script?: string;
    scriptMode: ScriptMode;
    status: number;
    timeout: number;
  }

  export interface TaskListParams {
    name?: string;
    page?: number;
    pageSize?: number;
  }

  export interface ScriptFile {
    createdAt: string;
    filePath: string;
    id: number;
    isCurrent: boolean;
    originalName: string;
    sha256: string;
    sizeBytes: number;
    storedName: string;
    taskId: number;
    version: number;
  }

  export interface Log {
    duration: number;
    endTime: string;
    error: string;
    exitCode: number;
    id: number;
    output: string;
    scriptFileId: number;
    scriptFileName: string;
    scriptFilePath: string;
    scriptFileSha256: string;
    scriptFileVersion: number;
    scriptMode: ScriptMode;
    scriptSnapshot: string;
    startTime: string;
    status: number;
    taskId: number;
    taskName: string;
    triggerMode: TriggerMode;
  }

  export interface ListParams {
    page?: number;
    pageSize?: number;
  }

  export interface LogListParams extends ListParams {
    status?: number;
    taskId?: number;
  }

  export interface ListResult<T> {
    list: T[];
    total: number;
  }
}

function toListResult<T>(resp: ListResult<T> | T[]): CronApi.ListResult<T> {
  return {
    list: listOf<T>(resp),
    total: totalOf(resp),
  };
}

export async function getCronTaskListApi(
  params?: CronApi.TaskListParams,
  config?: RequestClientConfig,
) {
  const resp = await requestClient.get<CronApi.ListResult<CronApi.Task>>(
    '/cron/task',
    { params, ...config },
  );
  return toListResult<CronApi.Task>(resp);
}

export async function createCronTaskApi(data: CronApi.TaskPayload) {
  return requestClient.post<void>('/cron/task', data);
}

export async function createCronTaskWithFileApi(
  data: CronApi.TaskPayload,
  file: File,
) {
  const formData = new FormData();
  formData.append('name', data.name);
  formData.append('schedule', data.schedule);
  formData.append('scriptMode', data.scriptMode);
  formData.append('script', data.script ?? '');
  formData.append('status', String(data.status));
  formData.append('timeout', String(data.timeout));
  formData.append('file', file);
  return requestClient.post<void>('/cron/task', formData);
}

export async function updateCronTaskApi(
  id: number,
  data: CronApi.TaskPayload,
) {
  return requestClient.put<void>(`/cron/task/${id}`, data);
}

export async function deleteCronTaskApi(id: number) {
  return requestClient.delete<void>(`/cron/task/${id}`);
}

export async function triggerCronTaskApi(id: number) {
  return requestClient.post<void>(`/cron/task/${id}/trigger`);
}

export async function getCronScriptHistoryApi(
  taskId: number,
  params?: CronApi.ListParams,
  config?: RequestClientConfig,
) {
  const resp = await requestClient.get<CronApi.ListResult<CronApi.ScriptFile>>(
    `/cron/task/${taskId}/script/history`,
    { params, ...config },
  );
  return toListResult<CronApi.ScriptFile>(resp);
}

export async function uploadCronScriptApi(taskId: number, file: File) {
  const formData = new FormData();
  formData.append('file', file);
  return requestClient.post<void>(`/cron/task/${taskId}/script`, formData);
}

export async function activateCronScriptApi(taskId: number, fileId: number) {
  return requestClient.post<void>(
    `/cron/task/${taskId}/script/${fileId}/activate`,
  );
}

export async function getCronLogListApi(
  params?: CronApi.LogListParams,
  config?: RequestClientConfig,
) {
  const resp = await requestClient.get<CronApi.ListResult<CronApi.Log>>(
    '/cron/log',
    { params, ...config },
  );
  return toListResult<CronApi.Log>(resp);
}

export async function getCronLogDetailApi(
  id: number,
  config?: RequestClientConfig,
) {
  return requestClient.get<CronApi.Log>(`/cron/log/${id}`, config);
}
