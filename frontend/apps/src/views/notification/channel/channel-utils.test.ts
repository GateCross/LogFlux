import { describe, expect, it } from 'vitest';

import {
  applyEventTags,
  applyWebhookConfig,
  buildWebhookConfig,
  channelEndpoint,
  channelTypeLabel,
  createDefaultWebhookForm,
  parseConfig,
  stringifyConfig,
} from './channel-utils';

describe('channel-utils', () => {
  it('channelTypeLabel returns Chinese labels and falls back to raw type', () => {
    expect(channelTypeLabel('webhook')).toBe('Webhook');
    expect(channelTypeLabel('wecom')).toBe('企业微信机器人');
    expect(channelTypeLabel('unknown_type')).toBe('unknown_type');
  });

  it('parseConfig handles object, valid JSON, invalid JSON and empty', () => {
    expect(parseConfig({ url: 'https://x' })).toEqual({ url: 'https://x' });
    expect(parseConfig('{"a":1}')).toEqual({ a: 1 });
    expect(parseConfig('{bad')).toEqual({});
    expect(parseConfig(undefined)).toEqual({});
  });

  it('stringifyConfig round-trips objects and keeps non-empty strings', () => {
    expect(stringifyConfig({ a: 1 })).toBe(JSON.stringify({ a: 1 }, null, 2));
    expect(stringifyConfig('{"a":1}')).toBe('{"a":1}');
    expect(stringifyConfig('')).toBe('{}');
    expect(stringifyConfig(undefined)).toBe('{}');
  });

  it('buildWebhookConfig trims keys/values and maps body sources', () => {
    const form = createDefaultWebhookForm();
    form.url = '  https://hook.example/  ';
    form.method = 'PUT';
    form.headers = [
      { key: ' Content-Type ', value: ' application/json ' },
      { key: '  ', value: 'skip-me' },
    ];
    form.body_fields = [
      { key: ' title ', source: 'title', customValue: '' },
      { key: 'note', source: 'custom', customValue: 'hello' },
      { key: '  ', source: 'content', customValue: '' },
    ];

    expect(buildWebhookConfig(form)).toEqual({
      body_fields: { title: 'title', note: 'hello' },
      headers: { 'Content-Type': 'application/json' },
      method: 'PUT',
      payload_mode: 'message_api',
      url: 'https://hook.example/',
    });
  });

  it('applyWebhookConfig restores known sources and custom values', () => {
    const applied = applyWebhookConfig({
      url: 'https://hook',
      method: 'PATCH',
      headers: { Authorization: 'Bearer x' },
      body_fields: { title: 'title', custom: 'literal' },
    });

    expect(applied.url).toBe('https://hook');
    expect(applied.method).toBe('PATCH');
    expect(applied.headers).toEqual([{ key: 'Authorization', value: 'Bearer x' }]);
    expect(applied.body_fields).toEqual([
      { key: 'title', source: 'title', customValue: '' },
      { key: 'custom', source: 'custom', customValue: 'literal' },
    ]);
  });

  it('applyEventTags parses arrays and falls back to *', () => {
    expect(applyEventTags('["system.*","caddy.*"]')).toEqual([
      'system.*',
      'caddy.*',
    ]);
    expect(applyEventTags('[]')).toEqual(['*']);
    expect(applyEventTags('')).toEqual(['*']);
    expect(applyEventTags('not-json')).toEqual(['*']);
  });

  it('channelEndpoint prefers url then endpoint/webhook/address', () => {
    expect(channelEndpoint({ config: { url: 'u' } })).toBe('u');
    expect(channelEndpoint({ config: { endpoint: 'e' } })).toBe('e');
    expect(channelEndpoint({ config: { webhook: 'w' } })).toBe('w');
    expect(channelEndpoint({ config: { address: 'a' } })).toBe('a');
    expect(channelEndpoint({ config: {} })).toBe('-');
  });
});
