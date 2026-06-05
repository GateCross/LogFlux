import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace CaddyWafPolicyApi {
  export interface WafPolicy {
    [key: string]: any;
  }

  export interface WafPolicyListResult {
    list: WafPolicy[];
    total: number;
  }

  export interface PolicyRevision {
    [key: string]: any;
  }

  export interface PolicyRevisionListResult {
    list: PolicyRevision[];
    total: number;
  }

  export interface PolicyDirectivePreview {
    [key: string]: any;
  }

  export interface RuleExclusion {
    [key: string]: any;
  }

  export interface RuleExclusionListResult {
    list: RuleExclusion[];
    total: number;
  }

  export interface PolicyBinding {
    [key: string]: any;
  }

  export interface PolicyBindingListResult {
    list: PolicyBinding[];
    total: number;
  }

  export interface RollbackParams {
    revisionId: number;
  }
}

// ─── WAF Policy CRUD ───

/**
 * 获取 WAF 策略列表 — GET /caddy/waf/policy
 */
export async function getWafPolicyListApi() {
  const resp = await requestClient.get<CaddyWafPolicyApi.WafPolicyListResult>(
    '/caddy/waf/policy',
  );
  return listOf(resp);
}

/**
 * 创建 WAF 策略 — POST /caddy/waf/policy
 */
export async function createWafPolicyApi(data: CaddyWafPolicyApi.WafPolicy) {
  return requestClient.post<void>('/caddy/waf/policy', data);
}

/**
 * 更新 WAF 策略 — PUT /caddy/waf/policy/:id
 */
export async function updateWafPolicyApi(
  id: number,
  data: CaddyWafPolicyApi.WafPolicy,
) {
  return requestClient.put<void>(`/caddy/waf/policy/${id}`, data);
}

/**
 * 删除 WAF 策略 — DELETE /caddy/waf/policy/:id
 */
export async function deleteWafPolicyApi(id: number) {
  return requestClient.delete<void>(`/caddy/waf/policy/${id}`);
}

// ─── Policy Operations ───

/**
 * 预览策略生成的指令 — POST /caddy/waf/policy/:id/preview
 */
export async function previewWafPolicyApi(id: number) {
  return requestClient.post<CaddyWafPolicyApi.PolicyDirectivePreview>(
    `/caddy/waf/policy/${id}/preview`,
  );
}

/**
 * 校验 WAF 策略 — POST /caddy/waf/policy/:id/validate
 */
export async function validateWafPolicyApi(id: number) {
  return requestClient.post<void>(`/caddy/waf/policy/${id}/validate`);
}

/**
 * 发布策略版本 — POST /caddy/waf/policy/:id/publish
 */
export async function publishWafPolicyApi(id: number) {
  return requestClient.post<void>(`/caddy/waf/policy/${id}/publish`);
}

/**
 * 回滚到指定策略版本 — POST /caddy/waf/policy/rollback
 */
export async function rollbackWafPolicyApi(
  data: CaddyWafPolicyApi.RollbackParams,
) {
  return requestClient.post<void>('/caddy/waf/policy/rollback', data);
}

/**
 * 获取策略版本列表 — GET /caddy/waf/policy/revision
 */
export async function getWafPolicyRevisionListApi() {
  const resp = await requestClient.get<CaddyWafPolicyApi.PolicyRevisionListResult>(
    '/caddy/waf/policy/revision',
  );
  return listOf(resp);
}

// ─── Rule Exclusions ───

/**
 * 获取规则排除列表 — GET /caddy/waf/policy/exclusion
 */
export async function getWafPolicyExclusionListApi() {
  const resp = await requestClient.get<CaddyWafPolicyApi.RuleExclusionListResult>(
    '/caddy/waf/policy/exclusion',
  );
  return listOf(resp);
}

/**
 * 创建规则排除 — POST /caddy/waf/policy/exclusion
 */
export async function createWafPolicyExclusionApi(
  data: CaddyWafPolicyApi.RuleExclusion,
) {
  return requestClient.post<void>('/caddy/waf/policy/exclusion', data);
}

/**
 * 更新规则排除 — PUT /caddy/waf/policy/exclusion/:id
 */
export async function updateWafPolicyExclusionApi(
  id: number,
  data: CaddyWafPolicyApi.RuleExclusion,
) {
  return requestClient.put<void>(`/caddy/waf/policy/exclusion/${id}`, data);
}

/**
 * 删除规则排除 — DELETE /caddy/waf/policy/exclusion/:id
 */
export async function deleteWafPolicyExclusionApi(id: number) {
  return requestClient.delete<void>(`/caddy/waf/policy/exclusion/${id}`);
}

// ─── Policy Bindings ───

/**
 * 获取策略绑定列表 — GET /caddy/waf/policy/binding
 */
export async function getWafPolicyBindingListApi() {
  const resp = await requestClient.get<CaddyWafPolicyApi.PolicyBindingListResult>(
    '/caddy/waf/policy/binding',
  );
  return listOf(resp);
}

/**
 * 创建策略绑定 — POST /caddy/waf/policy/binding
 */
export async function createWafPolicyBindingApi(
  data: CaddyWafPolicyApi.PolicyBinding,
) {
  return requestClient.post<void>('/caddy/waf/policy/binding', data);
}

/**
 * 更新策略绑定 — PUT /caddy/waf/policy/binding/:id
 */
export async function updateWafPolicyBindingApi(
  id: number,
  data: CaddyWafPolicyApi.PolicyBinding,
) {
  return requestClient.put<void>(`/caddy/waf/policy/binding/${id}`, data);
}

/**
 * 删除策略绑定 — DELETE /caddy/waf/policy/binding/:id
 */
export async function deleteWafPolicyBindingApi(id: number) {
  return requestClient.delete<void>(`/caddy/waf/policy/binding/${id}`);
}
