/**
 * 分页参数编解码：UI 页码从 1 起；非法 page/pageSize 归一为 1；非法 total 为 0
 */

export interface PageUiState {
  page: number;
  pageSize: number;
}

export interface PageRequestParams {
  page: number;
  pageSize: number;
}

export interface PageResultTotal {
  total: number;
}

function normalizePositiveInt(value: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return 1;
  }
  const n = Math.floor(value);
  return n < 1 ? 1 : n;
}

function parseNonNegativeInt(value: unknown): number {
  if (typeof value !== 'number' || !Number.isInteger(value) || value < 0) {
    return 0;
  }
  return value;
}

/** UI 分页 → 请求参数 */
export function toPageParams(ui: PageUiState): PageRequestParams {
  return {
    page: normalizePositiveInt(ui?.page),
    pageSize: normalizePositiveInt(ui?.pageSize),
  };
}

/** 列表响应 → UI total */
export function fromPageResult(
  resp: { total?: unknown } | null | undefined,
): PageResultTotal {
  return {
    total: parseNonNegativeInt(resp?.total),
  };
}
