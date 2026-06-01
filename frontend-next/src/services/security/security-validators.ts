/**
 * Security 校验器纯逻辑（移植自旧 Vue 版 `src/views/security/security-validators.ts`
 * 以及 `src/views/security/index.vue` 中内联的 `policyRules` / `policyFeedbackRules`）。
 *
 * 移植说明（task 11.2 / Req 13.3、13.4、17.1）：
 *  - 旧版校验器返回 Naive UI 的 `{ validator(rule, value), trigger }` 形态，强耦合
 *    `naive-ui` 的 `FormItemRule` 类型。此处将校验「核心」抽为框架无关纯函数，
 *    统一返回 `string | null`（`null` 表示通过，字符串为错误信息），不依赖任何
 *    Vue / Naive UI / React 运行时；由 `./security-validators-antd.ts` 适配器单独
 *    包装为 AntD `Form` 的 `Rule`。
 *  - `validateDateTime` / `validateStatusCode` / `validateMethod` 与旧版校验器
 *    逐字保留判定语义（同输入 → 同 通过/不通过 决定）。
 *  - `validateWafPolicyFields` 移植旧 `index.vue` 的 `policyRules`：对每个字段执行
 *    与旧版等价的判定，并返回「非法字段集合」。按 Req 13.3 要求，引擎模式 /
 *    审计模式 / 审计日志格式等枚举字段在「必填非空」基础上额外校验取值落在后端
 *    约定枚举范围内（枚举成员校验天然蕴含非空校验，与旧版 `required` 等价且更严格，
 *    满足 Req 13.3「取值落在枚举范围内」）。其余字段（auditRelevantStatus /
 *    requestBodyLimit / requestBodyNoFilesLimit / config）逐字保留旧版自定义校验逻辑。
 *  - 返回的非法字段集合恰好等于实际违规字段集合（支撑 Property 12：WAF 策略输入校验）。
 */
import type { WafPolicyEngineMode } from './types';

/** 截止时间格式：YYYY-MM-DD HH:mm:ss（与旧版正则逐字一致）。 */
const DATE_TIME_PATTERN = /^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}$/;

/** 截止时间校验默认错误信息（与旧 `createDateTimeValidator` 默认值一致）。 */
export const DEFAULT_DATE_TIME_MESSAGE = '截止时间格式应为 YYYY-MM-DD HH:mm:ss';

/** 状态码校验默认错误信息（与旧 `createStatusCodeValidator` 默认值一致）。 */
export const DEFAULT_STATUS_CODE_MESSAGE = '状态码必须在 100-599 之间';

/** Method 校验默认错误信息（与旧 `createMethodValidator` 默认值一致）。 */
export const DEFAULT_METHOD_MESSAGE = 'Method 不合法';

/** 请求体限制上限：1 GiB（与旧版 `1024 * 1024 * 1024` 一致）。 */
const REQUEST_BODY_LIMIT_MAX = 1024 * 1024 * 1024;

/** 引擎模式允许取值（对应旧 `policyEngineModeOptions` / `WafPolicyEngineMode`）。 */
export const POLICY_ENGINE_MODE_VALUES: readonly WafPolicyEngineMode[] = ['on', 'off', 'detectiononly'];

/** 审计模式允许取值（对应旧 `policyAuditEngineOptions`）。 */
export const POLICY_AUDIT_ENGINE_VALUES: readonly string[] = ['relevantonly', 'on', 'off'];

/** 审计日志格式允许取值（对应旧 `policyAuditLogFormatOptions`）。 */
export const POLICY_AUDIT_LOG_FORMAT_VALUES: readonly string[] = ['json', 'native'];

/**
 * 校验截止时间格式。空值视为通过（与旧 `createDateTimeValidator` 一致）。
 *
 * @returns 通过返回 `null`，否则返回错误信息。
 */
export function validateDateTime(value: unknown, message = DEFAULT_DATE_TIME_MESSAGE): string | null {
  const text = String(value ?? '').trim();
  if (!text) {
    return null;
  }
  if (!DATE_TIME_PATTERN.test(text)) {
    return message;
  }
  return null;
}

/**
 * 校验 HTTP 状态码，须为 100-599 之间的有限数值（与旧 `createStatusCodeValidator` 一致，
 * 空值经 `Number('')` → 0 判定为非法）。
 *
 * @returns 通过返回 `null`，否则返回错误信息。
 */
export function validateStatusCode(value: unknown, message = DEFAULT_STATUS_CODE_MESSAGE): string | null {
  const num = Number(value);
  if (!Number.isFinite(num) || num < 100 || num > 599) {
    return message;
  }
  return null;
}

/**
 * 校验 HTTP Method 是否在允许集合内。空值视为通过（与旧 `createMethodValidator` 一致），
 * 大小写不敏感（统一转大写后比对）。
 *
 * @returns 通过返回 `null`，否则返回错误信息。
 */
export function validateMethod(value: unknown, allowedMethods: string[], message = DEFAULT_METHOD_MESSAGE): string | null {
  const normalized = String(value ?? '')
    .trim()
    .toUpperCase();
  if (!normalized) {
    return null;
  }
  if (!allowedMethods.includes(normalized)) {
    return message;
  }
  return null;
}

/** 校验策略名称（必填非空，对应旧 `policyRules.name`）。 */
export function validatePolicyName(value: unknown, message = '请输入策略名称'): string | null {
  return String(value ?? '').trim() ? null : message;
}

/** 校验引擎模式（必选且须落在枚举范围内，对应旧 `policyRules.engineMode`）。 */
export function validatePolicyEngineMode(value: unknown, message = '请选择引擎模式'): string | null {
  return POLICY_ENGINE_MODE_VALUES.includes(value as WafPolicyEngineMode) ? null : message;
}

/** 校验审计模式（必选且须落在枚举范围内，对应旧 `policyRules.auditEngine`）。 */
export function validatePolicyAuditEngine(value: unknown, message = '请选择审计模式'): string | null {
  return POLICY_AUDIT_ENGINE_VALUES.includes(value as string) ? null : message;
}

/** 校验审计日志格式（必选且须落在枚举范围内，对应旧 `policyRules.auditLogFormat`）。 */
export function validatePolicyAuditLogFormat(value: unknown, message = '请选择审计日志格式'): string | null {
  return POLICY_AUDIT_LOG_FORMAT_VALUES.includes(value as string) ? null : message;
}

/**
 * 校验审计状态匹配表达式（对应旧 `policyRules.auditRelevantStatus`）：
 * 非空且须为合法正则表达式。
 */
export function validateAuditRelevantStatus(value: unknown): string | null {
  const raw = String(value ?? '').trim();
  if (!raw) {
    return '请输入审计状态匹配表达式';
  }
  try {
    // eslint-disable-next-line no-new
    new RegExp(raw);
    return null;
  } catch {
    return '审计状态匹配表达式格式不合法';
  }
}

/**
 * 校验请求体限制（对应旧 `policyRules.requestBodyLimit`）：
 * 须为大于 0 的有限数值，且不超过 1 GiB。
 */
export function validateRequestBodyLimit(value: unknown): string | null {
  const num = Number(value);
  if (!Number.isFinite(num) || num <= 0) {
    return '请求体限制必须大于 0';
  }
  if (num > REQUEST_BODY_LIMIT_MAX) {
    return '请求体限制不能超过 1 GiB';
  }
  return null;
}

/**
 * 校验无文件请求体限制（对应旧 `policyRules.requestBodyNoFilesLimit`）：
 * 须为大于 0 的有限数值，且不超过 1 GiB。
 */
export function validateRequestBodyNoFilesLimit(value: unknown): string | null {
  const num = Number(value);
  if (!Number.isFinite(num) || num <= 0) {
    return '无文件请求体限制必须大于 0';
  }
  if (num > REQUEST_BODY_LIMIT_MAX) {
    return '无文件请求体限制不能超过 1 GiB';
  }
  return null;
}

/**
 * 校验扩展配置（对应旧 `policyRules.config`）：空值视为通过，非空时须为合法 JSON。
 */
export function validatePolicyConfig(value: unknown): string | null {
  const raw = String(value ?? '').trim();
  if (!raw) {
    return null;
  }
  try {
    JSON.parse(raw);
    return null;
  } catch {
    return '扩展配置必须是合法 JSON';
  }
}

/** WAF 策略表单待校验字段名（与旧 `policyRules` 字段集合一致）。 */
export type WafPolicyFieldName =
  | 'name'
  | 'engineMode'
  | 'auditEngine'
  | 'auditLogFormat'
  | 'auditRelevantStatus'
  | 'requestBodyLimit'
  | 'requestBodyNoFilesLimit'
  | 'config';

/** WAF 策略字段校验输入（各字段取值不限定类型，由各校验器内部归一化）。 */
export interface WafPolicyFieldsInput {
  name?: unknown;
  engineMode?: unknown;
  auditEngine?: unknown;
  auditLogFormat?: unknown;
  auditRelevantStatus?: unknown;
  requestBodyLimit?: unknown;
  requestBodyNoFilesLimit?: unknown;
  config?: unknown;
}

/**
 * 逐字段校验 WAF 策略输入，返回「字段名 → 错误信息」映射（仅包含非法字段）。
 *
 * 供 `validateWafPolicyFields`（取键集合）与 AntD 适配器（取错误信息）共用，
 * 保证两者判定完全一致。
 */
export function collectWafPolicyFieldErrors(input: WafPolicyFieldsInput): Partial<Record<WafPolicyFieldName, string>> {
  const errors: Partial<Record<WafPolicyFieldName, string>> = {};

  const checks: Array<[WafPolicyFieldName, string | null]> = [
    ['name', validatePolicyName(input.name)],
    ['engineMode', validatePolicyEngineMode(input.engineMode)],
    ['auditEngine', validatePolicyAuditEngine(input.auditEngine)],
    ['auditLogFormat', validatePolicyAuditLogFormat(input.auditLogFormat)],
    ['auditRelevantStatus', validateAuditRelevantStatus(input.auditRelevantStatus)],
    ['requestBodyLimit', validateRequestBodyLimit(input.requestBodyLimit)],
    ['requestBodyNoFilesLimit', validateRequestBodyNoFilesLimit(input.requestBodyNoFilesLimit)],
    ['config', validatePolicyConfig(input.config)]
  ];

  for (const [field, error] of checks) {
    if (error) {
      errors[field] = error;
    }
  }

  return errors;
}

/**
 * 校验 WAF 策略输入并返回「非法字段集合」。
 *
 * 当所有字段合法时返回空集合；否则返回的集合恰好等于实际违规字段集合
 * （支撑 Property 12：WAF 策略输入校验，Req 13.3/13.4）。
 */
export function validateWafPolicyFields(input: WafPolicyFieldsInput): Set<WafPolicyFieldName> {
  return new Set(Object.keys(collectWafPolicyFieldErrors(input)) as WafPolicyFieldName[]);
}
