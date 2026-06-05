export interface ListResult<T> {
  list: T[];
  total?: number;
}

export function listOf<T>(value: ListResult<T> | T[] | null | undefined): T[] {
  if (Array.isArray(value)) {
    return value;
  }
  return Array.isArray(value?.list) ? value.list : [];
}

export function totalOf<T>(value: ListResult<T> | T[] | null | undefined) {
  return Array.isArray(value) ? value.length : (value?.total ?? 0);
}
