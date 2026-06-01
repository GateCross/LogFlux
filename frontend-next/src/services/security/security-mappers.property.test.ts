import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import {
  mapReleaseStatusType,
  mapPolicyEngineModeType,
  mapPolicyRevisionStatusType,
  mapJobStatusType,
  mapJobStatusLabel,
  mapJobActionLabel,
  mapJobTriggerModeLabel,
} from './security-mappers';
import {
  mapPolicyEngineModeLabel,
  mapCrsTemplateLabel,
  mapScopeTypeLabel,
  mapPolicyRevisionStatusLabel,
} from './security-policy-utils';

/**
 * Property 13：Security 字段映射等价性（高优先级，任务 11.4）。
 *
 * 被测纯逻辑：移植后的 WAF 字段映射函数
 *  - 标签映射（`security-policy-utils.ts`）：mapPolicyEngineModeLabel / mapCrsTemplateLabel /
 *    mapScopeTypeLabel / mapPolicyRevisionStatusLabel
 *  - 标签 / 标签类型映射（`security-mappers.ts`）：mapJobStatusLabel / mapJobActionLabel /
 *    mapJobTriggerModeLabel / mapReleaseStatusType / mapPolicyEngineModeType /
 *    mapPolicyRevisionStatusType / mapJobStatusType
 *
 * 等价性基线：旧 Vue 版 `frontend/src/views/security/{security-mappers,security-policy-utils}.ts`
 * 的映射逻辑（移植后逐字保留语义，仅改类型导入路径）。下方 oracle 独立编码「已知枚举键 →
 * 既定标签 / 既定标签类型」与「未知输入 → 既定兜底值（`-`、原值或 'default'/'warning'）」，
 * 对相同输入断言移植函数输出与基线一致。
 *
 * 实现要点（影响断言）：
 *  - 标签映射对未知键回退为 `value || '-'`（空串回退为 '-'）。
 *  - 标签类型映射多数对未知键回退为 'default'，但 mapJobStatusType 的兜底为 'warning'。
 *  - 各 switch 基于原始取值（不做 trim/lowercase），故对枚举外脏字符串按 default 分支处理。
 */

// 各函数的已知枚举键 → 既定标签 / 既定标签类型（独立 oracle，对齐旧 Vue 版）。
const engineModeLabelMap: Record<string, string> = {
  on: 'On（阻断）',
  detectiononly: 'DetectionOnly（仅检测）',
  off: 'Off（关闭）',
};
const crsTemplateLabelMap: Record<string, string> = {
  low_fp: '低误报',
  balanced: '平衡',
  high_blocking: '高拦截',
  custom: '自定义',
};
const scopeTypeLabelMap: Record<string, string> = {
  global: '全局',
  site: '站点',
  route: '路由',
};
const revisionStatusLabelMap: Record<string, string> = {
  draft: '草稿',
  published: '已发布',
  rolled_back: '已回滚',
};
const jobStatusLabelMap: Record<string, string> = {
  running: '执行中',
  success: '成功',
  failed: '失败',
};
const jobActionLabelMap: Record<string, string> = {
  check: '检查',
  download: '下载',
  verify: '校验',
  activate: '激活',
  rollback: '回滚',
  engine_check: '引擎检查',
};
const jobTriggerModeLabelMap: Record<string, string> = {
  manual: '手动',
  upload: '上传',
  schedule: '定时',
  auto: '自动',
  system: '系统',
};
const releaseStatusTypeMap: Record<string, string> = {
  active: 'success',
  verified: 'info',
  failed: 'error',
  rolled_back: 'warning',
};
const engineModeTypeMap: Record<string, string> = {
  on: 'error',
  detectiononly: 'warning',
  off: 'default',
};
const revisionStatusTypeMap: Record<string, string> = {
  published: 'success',
  rolled_back: 'warning',
};
const jobStatusTypeMap: Record<string, string> = {
  success: 'success',
  failed: 'error',
};

/** 仅判定自有键，避免命中 `toString` 等原型属性（被测函数用 switch，故仅匹配自有枚举键）。 */
const hasKey = (map: Record<string, string>, value: string) =>
  Object.prototype.hasOwnProperty.call(map, value);

/** 标签映射 oracle：已知键取既定标签，未知键回退 `value || '-'`。 */
const labelOracle = (map: Record<string, string>) => (value: string) =>
  hasKey(map, value) ? map[value] : value || '-';

/** 标签类型映射 oracle：已知键取既定类型，未知键回退给定兜底值。 */
const typeOracle = (map: Record<string, string>, fallback: string) => (value: string) =>
  hasKey(map, value) ? map[value] : fallback;

// 被测函数与其独立 oracle 的配对（同一输入应产生相同结果）。
const cases: Array<{ name: string; fn: (value: string) => string; oracle: (value: string) => string }> = [
  { name: 'mapPolicyEngineModeLabel', fn: (v) => mapPolicyEngineModeLabel(v as never), oracle: labelOracle(engineModeLabelMap) },
  { name: 'mapCrsTemplateLabel', fn: (v) => mapCrsTemplateLabel(v), oracle: labelOracle(crsTemplateLabelMap) },
  { name: 'mapScopeTypeLabel', fn: (v) => mapScopeTypeLabel(v), oracle: labelOracle(scopeTypeLabelMap) },
  { name: 'mapPolicyRevisionStatusLabel', fn: (v) => mapPolicyRevisionStatusLabel(v as never), oracle: labelOracle(revisionStatusLabelMap) },
  { name: 'mapJobStatusLabel', fn: (v) => mapJobStatusLabel(v), oracle: labelOracle(jobStatusLabelMap) },
  { name: 'mapJobActionLabel', fn: (v) => mapJobActionLabel(v), oracle: labelOracle(jobActionLabelMap) },
  { name: 'mapJobTriggerModeLabel', fn: (v) => mapJobTriggerModeLabel(v), oracle: labelOracle(jobTriggerModeLabelMap) },
  { name: 'mapReleaseStatusType', fn: (v) => mapReleaseStatusType(v as never), oracle: typeOracle(releaseStatusTypeMap, 'default') },
  { name: 'mapPolicyEngineModeType', fn: (v) => mapPolicyEngineModeType(v as never), oracle: typeOracle(engineModeTypeMap, 'default') },
  { name: 'mapPolicyRevisionStatusType', fn: (v) => mapPolicyRevisionStatusType(v as never), oracle: typeOracle(revisionStatusTypeMap, 'default') },
  { name: 'mapJobStatusType', fn: (v) => mapJobStatusType(v as never), oracle: typeOracle(jobStatusTypeMap, 'warning') },
];

// 所有函数的已知枚举键集合，用于生成「枚举内」输入。
const knownKeys = Array.from(
  new Set([
    ...Object.keys(engineModeLabelMap),
    ...Object.keys(crsTemplateLabelMap),
    ...Object.keys(scopeTypeLabelMap),
    ...Object.keys(revisionStatusLabelMap),
    ...Object.keys(jobStatusLabelMap),
    ...Object.keys(jobActionLabelMap),
    ...Object.keys(jobTriggerModeLabelMap),
    ...Object.keys(releaseStatusTypeMap),
    ...Object.keys(engineModeTypeMap),
    ...Object.keys(revisionStatusTypeMap),
    ...Object.keys(jobStatusTypeMap),
  ]),
);

/** 输入空间：枚举内键 + 枚举外脏字符串（含空串、Unicode、特殊字符、大小写变体）。 */
const valueArb: fc.Arbitrary<string> = fc.oneof(
  fc.constantFrom(...knownKeys),
  // 大小写 / 首尾空白变体：因映射不做 trim/lowercase，应落入未知分支
  fc.constantFrom(...knownKeys).map((k) => k.toUpperCase()),
  fc.constantFrom(...knownKeys).map((k) => ` ${k} `),
  fc.constant(''),
  fc.string(),
  fc.fullUnicodeString(),
);

describe('Security 字段映射等价性（Property 13）', () => {
  // Feature: frontend-umijs-max-migration, Property 13: Security 字段映射等价性
  // Validates: Requirements 13.7, 17.1
  it('移植后映射函数对枚举内/外取值的输出与既定标签/兜底基线一致', () => {
    fc.assert(
      fc.property(valueArb, (value) => {
        for (const { fn, oracle } of cases) {
          expect(fn(value)).toBe(oracle(value));
        }
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
