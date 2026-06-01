/**
 * Security 校验器的 AntD `Form` rules 适配器（task 11.2 / Req 13.3、13.4）。
 *
 * 职责：将 `security-validators.ts` 中**框架无关的纯函数**（返回 `string | null`）
 * 包装为 Ant Design `Form` 所需的 `Rule`（`{ validator }` 形态），替代旧 Vue 版
 * 对 `naive-ui` `FormItemRule` + `{ validator, trigger }` 的耦合。
 *
 * 约定：
 *  - 纯函数返回 `null` 视为合法 → 适配器 `validator` resolve。
 *  - 纯函数返回非空字符串视为非法 → 适配器 `validator` reject 该错误文案，
 *    由 AntD `Form` 展示为字段级错误（对应 Req 13.4「对每个非法字段展示具体错误提示」）。
 *  - 本文件仅做形态适配，不改变任何校验语义；校验真值表完全由纯函数决定。
 */
import type { Rule } from 'antd/es/form';

import {
  DEFAULT_DATE_TIME_MESSAGE,
  DEFAULT_METHOD_MESSAGE,
  DEFAULT_STATUS_CODE_MESSAGE,
  validateAuditRelevantStatus,
  validateDateTime,
  validateMethod,
  validatePolicyAuditEngine,
  validatePolicyAuditLogFormat,
  validatePolicyConfig,
  validatePolicyEngineMode,
  validatePolicyName,
  validateRequestBodyLimit,
  validateRequestBodyNoFilesLimit,
  validateStatusCode
} from './security-validators';

/**
 * 将「纯校验函数（value → string | null）」包装为 AntD `Rule`。
 *
 * @param validate 纯校验函数，返回 `null` 合法、非空字符串为错误文案。
 */
export function toAntdRule(validate: (value: unknown) => string | null): Rule {
  return {
    validator(_rule, value) {
      const error = validate(value);
      if (error) {
        return Promise.reject(new Error(error));
      }
      return Promise.resolve();
    }
  };
}

/** 截止时间校验规则（对应旧 `createDateTimeValidator`）。 */
export function createDateTimeRule(message: string = DEFAULT_DATE_TIME_MESSAGE): Rule {
  return toAntdRule(value => validateDateTime(value, message));
}

/** 状态码校验规则（对应旧 `createStatusCodeValidator`）。 */
export function createStatusCodeRule(message: string = DEFAULT_STATUS_CODE_MESSAGE): Rule {
  return toAntdRule(value => validateStatusCode(value, message));
}

/** HTTP Method 校验规则（对应旧 `createMethodValidator`）。 */
export function createMethodRule(allowedMethods: string[], message: string = DEFAULT_METHOD_MESSAGE): Rule {
  return toAntdRule(value => validateMethod(value, allowedMethods, message));
}

/**
 * WAF 策略表单的 AntD `rules` 映射（对应旧 `index.vue` 的 `policyRules`）。
 *
 * 各字段规则均由对应纯函数包装而成，保证与 Vue 版同输入同判定。
 */
export const wafPolicyAntdRules: Record<string, Rule[]> = {
  name: [toAntdRule(validatePolicyName)],
  engineMode: [toAntdRule(validatePolicyEngineMode)],
  auditEngine: [toAntdRule(validatePolicyAuditEngine)],
  auditLogFormat: [toAntdRule(validatePolicyAuditLogFormat)],
  auditRelevantStatus: [toAntdRule(validateAuditRelevantStatus)],
  requestBodyLimit: [toAntdRule(validateRequestBodyLimit)],
  requestBodyNoFilesLimit: [toAntdRule(validateRequestBodyNoFilesLimit)],
  config: [toAntdRule(validatePolicyConfig)]
};
