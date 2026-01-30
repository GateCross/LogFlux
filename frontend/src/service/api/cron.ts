import { request } from '../request';

export interface CronTask {
    id: number;
    name: string;
    schedule: string;
    script: string;
    status: number; // 1: enable, 0: disable
    timeout: number;
    nextRun: string;
    createdAt: string;
    updatedAt: string;
}

export interface CronTaskLog {
    id: number;
    taskId: number;
    taskName: string;
    startTime: string;
    endTime: string;
    status: number; // 0: Running, 1: Success, 2: Failed, 3: Timeout
    exitCode: number;
    output: string;
    error: string;
    duration: number;
}

export interface CronTaskListResp {
    list: CronTask[];
    total: number;
}

export interface CronLogListResp {
    list: CronTaskLog[];
    total: number;
}

export function fetchCronTaskList(params?: any) {
    return request<CronTaskListResp>({
        url: '/api/cron/task',
        method: 'get',
        params
    });
}

export function createCronTask(data: any) {
    return request<void>({
        url: '/api/cron/task',
        method: 'post',
        data
    });
}

export function updateCronTask(id: number, data: any) {
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

export function fetchCronLogList(params?: any) {
    return request<CronLogListResp>({
        url: '/api/cron/log',
        method: 'get',
        params
    });
}
