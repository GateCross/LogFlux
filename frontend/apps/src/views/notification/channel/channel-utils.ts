import type { NotificationApi } from '#/api/notification';

/** Webhook 正文字段来源 */
export type WebhookBodySource =
  | 'content'
  | 'custom'
  | 'data'
  | 'level'
  | 'message'
  | 'timestamp'
  | 'title'
  | 'type';

export interface ChannelFormState {
  config: string;
  description: string;
  enabled: boolean;
  events: string;
  name: string;
  type: string;
}

export interface WebhookBodyFieldItem {
  customValue: string;
  key: string;
  source: WebhookBodySource;
}

export interface WebhookHeaderItem {
  key: string;
  value: string;
}

export interface WebhookConfigForm {
  body_fields: WebhookBodyFieldItem[];
  headers: WebhookHeaderItem[];
  method: string;
  url: string;
}

export const channelTypeOptions = [
  { label: 'Webhook', value: 'webhook' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'Slack', value: 'slack' },
  { label: '企业微信机器人', value: 'wecom' },
  { label: '企业微信应用消息', value: 'wechat_mp' },
  { label: 'Discord', value: 'discord' },
  { label: '邮件', value: 'email' },
  { label: '站内通知', value: 'in_app' },
];

export const methodOptions = [
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'GET', value: 'GET' },
];

export const bodySourceOptions = [
  { label: '标题', value: 'title' },
  { label: '内容', value: 'content' },
  { label: '消息', value: 'message' },
  { label: '级别', value: 'level' },
  { label: '类型', value: 'type' },
  { label: '时间', value: 'timestamp' },
  { label: '事件数据', value: 'data' },
  { label: '自定义', value: 'custom' },
];

export const eventOptions = [
  { label: '全部事件', value: '*' },
  { label: '系统事件', value: 'system.*' },
  { label: 'Caddy 事件', value: 'caddy.*' },
  { label: '安全事件', value: 'security.*' },
  { label: '任务事件', value: 'task.*' },
  { label: '归档事件', value: 'archive.*' },
];

export function channelTypeLabel(type: string) {
  return channelTypeOptions.find((item) => item.value === type)?.label ?? type;
}

export function createDefaultFormState(): ChannelFormState {
  return {
    config: '{}',
    description: '',
    enabled: true,
    events: '["*"]',
    name: '',
    type: 'webhook',
  };
}

export function createHeaderItem(): WebhookHeaderItem {
  return { key: '', value: '' };
}

export function createBodyFieldItem(): WebhookBodyFieldItem {
  return { customValue: '', key: '', source: 'custom' };
}

export function createDefaultWebhookForm(): WebhookConfigForm {
  return {
    body_fields: [
      { customValue: '', key: 'title', source: 'title' },
      { customValue: '', key: 'content', source: 'content' },
    ],
    headers: [
      { key: 'Content-Type', value: 'application/json' },
      { key: 'apiKey', value: '' },
    ],
    method: 'POST',
    url: '',
  };
}

export function parseConfig(value: Record<string, any> | string | undefined) {
  if (!value) return {};
  if (typeof value !== 'string') return value;
  try {
    return JSON.parse(value);
  } catch {
    return {};
  }
}

export function stringifyConfig(
  value: Record<string, any> | string | undefined,
) {
  if (!value) return '{}';
  if (typeof value === 'string') return value || '{}';
  return JSON.stringify(value, null, 2);
}

export function buildWebhookConfig(webhookForm: WebhookConfigForm) {
  const headers = webhookForm.headers.reduce<Record<string, string>>(
    (acc, item) => {
      const key = item.key.trim();
      if (key) acc[key] = item.value.trim();
      return acc;
    },
    {},
  );

  const bodyFields = webhookForm.body_fields.reduce<Record<string, string>>(
    (acc, item) => {
      const key = item.key.trim();
      if (!key) return acc;
      acc[key] = item.source === 'custom' ? item.customValue : item.source;
      return acc;
    },
    {},
  );

  return {
    body_fields: bodyFields,
    headers,
    method: webhookForm.method,
    payload_mode: 'message_api',
    url: webhookForm.url.trim(),
  };
}

export function applyWebhookConfig(
  config: Record<string, any> | string | undefined,
): WebhookConfigForm {
  const next = createDefaultWebhookForm();
  const parsed = parseConfig(config);
  const knownSources = new Set<WebhookBodySource>([
    'content',
    'data',
    'level',
    'message',
    'timestamp',
    'title',
    'type',
  ]);

  const headers =
    parsed.headers && typeof parsed.headers === 'object'
      ? Object.entries(parsed.headers as Record<string, any>).map(
          ([key, value]) => ({
            key,
            value: String(value ?? ''),
          }),
        )
      : next.headers;

  const bodyFields =
    parsed.body_fields && typeof parsed.body_fields === 'object'
      ? Object.entries(parsed.body_fields as Record<string, any>).map(
          ([key, value]) => {
            const source = String(value ?? '');
            const isKnownSource = knownSources.has(source as WebhookBodySource);
            return {
              customValue: isKnownSource ? '' : source,
              key,
              source: isKnownSource ? (source as WebhookBodySource) : 'custom',
            };
          },
        )
      : next.body_fields;

  return {
    ...next,
    body_fields: bodyFields.length > 0 ? bodyFields : next.body_fields,
    headers: headers.length > 0 ? headers : next.headers,
    method: typeof parsed.method === 'string' ? parsed.method : next.method,
    url: typeof parsed.url === 'string' ? parsed.url : next.url,
  };
}

export function applyEventTags(eventsText: string | undefined): string[] {
  if (!eventsText?.trim()) {
    return ['*'];
  }

  try {
    const parsed = JSON.parse(eventsText);
    if (Array.isArray(parsed)) {
      const tags = parsed.map((item) => String(item)).filter(Boolean);
      return tags.length > 0 ? tags : ['*'];
    }
  } catch {
    // fallback below
  }

  return ['*'];
}

export function channelEndpoint(record: Partial<NotificationApi.Channel>) {
  const config = parseConfig(record.config);
  return config.url || config.endpoint || config.webhook || config.address || '-';
}
