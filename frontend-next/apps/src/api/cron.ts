import { requestClient } from '#/api/request';

export namespace CronApi {
  /** Cron task entity */
  export interface Task {
    createdAt: string;
    cronExpression: string;
    description: string;
    enabled: boolean;
    id: string;
    name: string;
    scriptContent: string;
    scriptFileId: string;
    scriptType: 'file' | 'inline';
    timeout: number;
    updatedAt: string;
    [key: string]: any;
  }

  /** Parameters for creating a cron task */
  export interface CreateTaskParams {
    cronExpression: string;
    description?: string;
    enabled?: boolean;
    name: string;
    /** Inline script content (used when scriptType is 'inline') */
    scriptContent?: string;
    scriptType: 'file' | 'inline';
    timeout?: number;
  }

  /** Parameters for updating a cron task */
  export interface UpdateTaskParams {
    cronExpression?: string;
    description?: string;
    enabled?: boolean;
    name?: string;
    scriptContent?: string;
    scriptType?: 'file' | 'inline';
    timeout?: number;
  }

  /** Script file version entry */
  export interface ScriptFileVersion {
    activatedAt: string;
    createdAt: string;
    fileName: string;
    fileSize: number;
    id: string;
    isActive: boolean;
    taskId: string;
    version: number;
    [key: string]: any;
  }

  /** Cron execution log entry */
  export interface Log {
    createdAt: string;
    duration: number;
    endTime: string;
    errorMessage: string;
    id: string;
    output: string;
    startTime: string;
    status: 'failed' | 'running' | 'success';
    taskId: string;
    taskName: string;
    triggerType: 'auto' | 'manual';
    [key: string]: any;
  }

  /** Paginated list query parameters for cron logs */
  export interface LogListParams {
    page?: number;
    pageSize?: number;
    status?: string;
    taskId?: string;
  }

  /** Paginated response wrapper */
  export interface PaginatedResult<T> {
    list: T[];
    page: number;
    pageSize: number;
    total: number;
  }
}

// ────────────────────────────────────────────
// Cron Task CRUD
// ────────────────────────────────────────────

/** List all cron tasks — GET /cron/task */
export async function getCronTaskListApi() {
  return requestClient.get<CronApi.Task[]>('/cron/task');
}

/** Create a cron task — POST /cron/task */
export async function createCronTaskApi(data: CronApi.CreateTaskParams) {
  return requestClient.post<CronApi.Task>('/cron/task', data);
}

/** Update a cron task — PUT /cron/task/:id */
export async function updateCronTaskApi(
  id: string,
  data: CronApi.UpdateTaskParams,
) {
  return requestClient.put<CronApi.Task>(`/cron/task/${id}`, data);
}

/** Delete a cron task — DELETE /cron/task/:id */
export async function deleteCronTaskApi(id: string) {
  return requestClient.delete<void>(`/cron/task/${id}`);
}

/** Manually trigger a cron task — POST /cron/task/:id/trigger */
export async function triggerCronTaskApi(id: string) {
  return requestClient.post<void>(`/cron/task/${id}/trigger`);
}

// ────────────────────────────────────────────
// Script file management
// ────────────────────────────────────────────

/** Get script file version history — GET /cron/task/:taskId/script/history */
export async function getCronScriptHistoryApi(taskId: string) {
  return requestClient.get<CronApi.ScriptFileVersion[]>(
    `/cron/task/${taskId}/script/history`,
  );
}

/** Upload a script file for a task — POST /cron/task/:taskId/script */
export async function uploadCronScriptApi(taskId: string, file: File) {
  const formData = new FormData();
  formData.append('file', file);
  return requestClient.post<CronApi.ScriptFileVersion>(
    `/cron/task/${taskId}/script`,
    formData,
  );
}

/** Activate a specific script version — POST /cron/task/:taskId/script/:fileId/activate */
export async function activateCronScriptApi(taskId: string, fileId: string) {
  return requestClient.post<void>(
    `/cron/task/${taskId}/script/${fileId}/activate`,
  );
}

// ────────────────────────────────────────────
// Cron execution logs
// ────────────────────────────────────────────

/** List cron execution logs (paginated) — GET /cron/log */
export async function getCronLogListApi(params?: CronApi.LogListParams) {
  return requestClient.get<CronApi.PaginatedResult<CronApi.Log>>('/cron/log', {
    params,
  });
}

/** Get cron log detail — GET /cron/log/:id */
export async function getCronLogDetailApi(id: string) {
  return requestClient.get<CronApi.Log>(`/cron/log/${id}`);
}
