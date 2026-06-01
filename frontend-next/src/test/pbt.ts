/**
 * 属性测试（Property-Based Testing）公共约定（任务 1.3 / Req 17.2）。
 *
 * 设计文档 Testing Strategy 规定：
 * - 每个 Correctness Property 用「单个」属性测试实现，使用 `fast-check` + `vitest`。
 * - 每个属性测试至少运行 100 次迭代（`{ numRuns: 100 }`）。
 * - 每个属性测试以注释标注来源设计属性，格式：
 *     // Feature: frontend-umijs-max-migration, Property {number}: {property_text}
 *
 * 用法示例：
 *
 *   import fc from 'fast-check';
 *   import { it } from 'vitest';
 *   import { PBT_RUNS } from '@/test/pbt';
 *
 *   // Feature: frontend-umijs-max-migration, Property 7: 语言归一化与回退
 *   // Validates: Requirements 5.6
 *   it('normalizeLang 输出恒属于 {zh-CN, en-US}', () => {
 *     fc.assert(
 *       fc.property(fc.string(), (input) => {
 *         expect(['zh-CN', 'en-US']).toContain(normalizeLang(input));
 *       }),
 *       { numRuns: PBT_RUNS },
 *     );
 *   });
 */

/** 属性测试默认迭代次数（设计约定的下限）。 */
export const PBT_RUNS = 100;

/**
 * fast-check 断言参数的默认配置。
 * 在调用 `fc.assert(prop, PBT_ASSERT_OPTIONS)` 时直接复用，确保统一的迭代次数。
 */
export const PBT_ASSERT_OPTIONS = { numRuns: PBT_RUNS } as const;
