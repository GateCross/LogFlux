import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import {
  validateWafPolicyFields,
  POLICY_ENGINE_MODE_VALUES,
  POLICY_AUDIT_ENGINE_VALUES,
  POLICY_AUDIT_LOG_FORMAT_VALUES,
  type WafPolicyFieldName,
  type WafPolicyFieldsInput,
} from './security-validators';

/**
 * Property 12：WAF 策略输入校验（任务 11.6）。
 *
 * 被测纯逻辑：`validateWafPolicyFields`（`security-validators.ts`），返回「非法字段集合」。
 *
 * 测试思路（避免自我循环 oracle）：为每个字段构造「按构造已知合法/非法」的取值，
 * 覆盖枚举内/外、数值范围内/外、必填缺失（undefined/null/空白）与合法组合；
 * 期望非法字段集合 = 所有 `valid === false` 的字段集合，再断言：
 *  - 分类：实际「存在非法字段」当且仅当期望「存在非法字段」；
 *  - 精确性：实际非法字段集合恰好等于期望非法字段集合；
 *  - 全合法时非法字段集合为空。
 */

/** 请求体限制上限：1 GiB（与实现 `1024 * 1024 * 1024` 一致）。 */
const REQUEST_BODY_LIMIT_MAX = 1024 * 1024 * 1024;

/** 单字段取值及其（按构造已知的）合法性。 */
interface FieldCase {
  value: unknown;
  valid: boolean;
}

const valid = (value: unknown): FieldCase => ({ value, valid: true });
const invalid = (value: unknown): FieldCase => ({ value, valid: false });

const ENGINE_MODES = POLICY_ENGINE_MODE_VALUES as readonly string[];
const AUDIT_ENGINES = POLICY_AUDIT_ENGINE_VALUES as readonly string[];
const AUDIT_LOG_FORMATS = POLICY_AUDIT_LOG_FORMAT_VALUES as readonly string[];

/** name：必填非空（trim 后非空）。 */
const nameArb: fc.Arbitrary<FieldCase> = fc.oneof(
  fc
    .string({ minLength: 1 })
    .filter((s) => s.trim().length > 0)
    .map(valid),
  // 必填缺失 / 空 / 纯空白
  fc.constantFrom(undefined, null, '', '   ', '\t', '\n  ').map(invalid),
);

/** engineMode：必选且取值须落在枚举范围内。 */
const engineModeArb: fc.Arbitrary<FieldCase> = fc.oneof(
  fc.constantFrom(...ENGINE_MODES).map(valid),
  fc.oneof(
    fc.string().filter((s) => !ENGINE_MODES.includes(s)),
    // 必填缺失 + 越界取值（含大小写不匹配 / 非字符串）
    fc.constantFrom(undefined, null, 0, 1, true, '', 'ON', 'On', 'detection'),
  ).map(invalid),
);

/** auditEngine：必选且取值须落在枚举范围内。 */
const auditEngineArb: fc.Arbitrary<FieldCase> = fc.oneof(
  fc.constantFrom(...AUDIT_ENGINES).map(valid),
  fc.oneof(
    fc.string().filter((s) => !AUDIT_ENGINES.includes(s)),
    fc.constantFrom(undefined, null, 0, true, 'RelevantOnly', 'foo'),
  ).map(invalid),
);

/** auditLogFormat：必选且取值须落在枚举范围内。 */
const auditLogFormatArb: fc.Arbitrary<FieldCase> = fc.oneof(
  fc.constantFrom(...AUDIT_LOG_FORMATS).map(valid),
  fc.oneof(
    fc.string().filter((s) => !AUDIT_LOG_FORMATS.includes(s)),
    fc.constantFrom(undefined, null, 0, 'JSON', 'xml'),
  ).map(invalid),
);

/** auditRelevantStatus：非空（trim 后）且为合法正则。 */
const auditRelevantStatusArb: fc.Arbitrary<FieldCase> = fc.oneof(
  fc.constantFrom('200', '^200$', '4\\d\\d', '.*', '[0-9]+', '5\\d{2}', 'a|b').map(valid),
  fc
    .oneof(
      // 必填缺失 / 空 / 纯空白 → trim 后为空
      fc.constantFrom(undefined, null, '', '   ', '\t'),
      // 非空但为非法正则
      fc.constantFrom('[', '(', '*', '\\', '(?<', '[a-', '+', '?'),
    )
    .map(invalid),
);

/** requestBodyLimit / requestBodyNoFilesLimit：有限数值，> 0 且 <= 1 GiB。 */
const bodyLimitArb: fc.Arbitrary<FieldCase> = fc.oneof(
  // 合法：(0, MAX]
  fc.integer({ min: 1, max: REQUEST_BODY_LIMIT_MAX }).map(valid),
  // 非法：<= 0
  fc.integer({ min: -1_000_000, max: 0 }).map(invalid),
  // 非法：> MAX
  fc.integer({ min: REQUEST_BODY_LIMIT_MAX + 1, max: REQUEST_BODY_LIMIT_MAX + 1_000_000 }).map(invalid),
  // 非法：非数值 / 非有限 / 必填缺失
  fc.constantFrom(undefined, null, '', 'abc', NaN, Infinity, -Infinity).map(invalid),
);

/** config：空值视为通过；非空时须为合法 JSON。 */
const configArb: fc.Arbitrary<FieldCase> = fc.oneof(
  // 合法：空 / 纯空白
  fc.constantFrom(undefined, null, '', '   ').map(valid),
  // 合法：非空且为合法 JSON
  fc.constantFrom('{}', '[]', '"x"', '123', 'true', 'false', 'null', '{"a":1}', '[1,2,3]').map(valid),
  // 非法：非空且非合法 JSON
  fc.constantFrom('{', 'not json', '{a:1}', '[1,2', "'x'", '{"a":}', 'undefined').map(invalid),
);

type PolicyCases = Record<WafPolicyFieldName, FieldCase>;

const policyCaseArb: fc.Arbitrary<PolicyCases> = fc.record({
  name: nameArb,
  engineMode: engineModeArb,
  auditEngine: auditEngineArb,
  auditLogFormat: auditLogFormatArb,
  auditRelevantStatus: auditRelevantStatusArb,
  requestBodyLimit: bodyLimitArb,
  requestBodyNoFilesLimit: bodyLimitArb,
  config: configArb,
});

describe('security-validators WAF 策略输入校验（Property 12）', () => {
  // Feature: frontend-umijs-max-migration, Property 12: WAF 策略输入校验
  // Validates: Requirements 13.3, 13.4
  it('非法字段集合恰好等于实际违规字段集合；全合法时为空', () => {
    fc.assert(
      fc.property(policyCaseArb, (cases) => {
        const fields = Object.keys(cases) as WafPolicyFieldName[];

        const input: WafPolicyFieldsInput = {
          name: cases.name.value,
          engineMode: cases.engineMode.value,
          auditEngine: cases.auditEngine.value,
          auditLogFormat: cases.auditLogFormat.value,
          auditRelevantStatus: cases.auditRelevantStatus.value,
          requestBodyLimit: cases.requestBodyLimit.value,
          requestBodyNoFilesLimit: cases.requestBodyNoFilesLimit.value,
          config: cases.config.value,
        };

        const expectedInvalid = new Set<WafPolicyFieldName>(fields.filter((field) => !cases[field].valid));
        const actualInvalid = validateWafPolicyFields(input);

        // 分类：存在非法字段当且仅当期望存在非法字段。
        expect(actualInvalid.size > 0).toBe(expectedInvalid.size > 0);

        // 精确性：非法字段集合恰好等于实际违规字段集合。
        expect([...actualInvalid].sort()).toEqual([...expectedInvalid].sort());

        // 全合法时非法字段集合为空。
        if (expectedInvalid.size === 0) {
          expect(actualInvalid.size).toBe(0);
        }
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
