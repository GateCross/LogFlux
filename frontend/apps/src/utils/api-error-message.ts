/**
 * 从 API 错误提取展示文案；优先级：response.data.message/msg/error → data.* → error.message → fallback
 */
function pickNonEmptyString(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined;
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : undefined;
}

function pickFromPayload(payload: unknown): string | undefined {
  if (!payload || typeof payload !== 'object') return undefined;
  const record = payload as Record<string, unknown>;
  return (
    pickNonEmptyString(record.message) ??
    pickNonEmptyString(record.msg) ??
    pickNonEmptyString(record.error)
  );
}

export function apiErrorMessage(error: unknown, fallback: string): string {
  const err = error as {
    response?: { data?: unknown };
    data?: unknown;
    message?: unknown;
  } | null;

  const fromResponseData = pickFromPayload(err?.response?.data);
  if (fromResponseData) return fromResponseData;

  const fromData = pickFromPayload(err?.data);
  if (fromData) return fromData;

  const fromMessage = pickNonEmptyString(err?.message);
  if (fromMessage) return fromMessage;

  return fallback;
}
