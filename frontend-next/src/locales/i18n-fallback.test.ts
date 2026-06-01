/**
 * i18n 缺键回退至 zh-CN 属性测试（任务 5.4）。
 *
 * 对应 design.md 的 Correctness Property 8：
 *   *For any* 已注册文案键，当所选语言（en-US）缺少该键时，文案解析应回退渲染该键在
 *   zh-CN 中对应的文案，而非渲染空串或键名。
 *
 * 被测产品行为（见 `src/locales/en-US.ts`）：
 *  - en-US 的**默认导出**为 `{ ...zhCN, ...messages }`——把 zh-CN 合并作为缺键回退基准
 *    （fallbackLocale=zh-CN，Req 5.7）。因此「以 en-US 默认导出解析某键」等价于：en-US 原始
 *    `messages` 含该键则取 en-US 文案，否则回退取 zh-CN 文案；二者皆为非空文案，绝不会是
 *    空串或键名。
 *
 * 测试思路（保证非空泛且忠实于产品合并逻辑）：
 *  - 「已注册文案键」取 zh-CN 默认导出的键集合（zh-CN 为基准/回退目标语言，Req 5.1/5.7）。
 *  - 当前 zh-CN 与 en-US 原始键集合相等（Property 9），现实中 en-US 并无真实缺键；若直接断言
 *    「真实缺键」会得到空采样空间而使属性退化。故对任意已注册键 K，**模拟** en-US 缺失 K：
 *    从 en-US 原始 `messages` 的副本中删除 K，再以与产品**完全相同**的合并表达式
 *    `{ ...zhCN, ...rawEnUSMissing }` 重建解析表，断言解析 K 回退为 zh-CN[K]（非空、非键名）。
 *  - 同时对**实际产品默认导出**断言：任意已注册键解析结果恒为非空且不等于键名，直接覆盖
 *    Property 8「而非渲染空串或键名」的全域不变式。
 */
import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import zhCNMessages from './zh-CN';
// 取 en-US 的**默认导出**（已合并 zh-CN 回退基准的解析表）与**具名导出**（原始 en-US 文案）。
import enUSDefault, { messages as rawEnUSMessages } from './en-US';

/** zh-CN 文案表（基准/回退目标语言）。 */
const zhCN = zhCNMessages as Record<string, string>;
/** en-US 原始文案表（未合并回退基准）。 */
const rawEnUS = rawEnUSMessages as Record<string, string>;
/** en-US 解析表（产品默认导出：`{ ...zhCN, ...rawEnUS }`）。 */
const mergedEnUS = enUSDefault as Record<string, string>;

/** 已注册文案键集合（以 zh-CN 基准键集合为准，Req 5.1/5.7）。 */
const registeredKeys = Object.keys(zhCN);

/** 从已注册键集合中采样任意一个键。 */
const arbRegisteredKey: fc.Arbitrary<string> = fc.constantFrom(...registeredKeys);

/**
 * 以与 `src/locales/en-US.ts` **完全相同**的合并表达式重建 en-US 解析表，
 * 用于模拟「en-US 缺失某键」时的回退解析。
 */
function buildMergedEnUS(rawEnUSPartial: Record<string, string>): Record<string, string> {
  return { ...zhCN, ...rawEnUSPartial };
}

describe('i18n — 缺键回退至 zh-CN（Property 8）', () => {
  // Feature: frontend-umijs-max-migration, Property 8: i18n 缺键回退至 zh-CN
  // Validates: Requirements 5.7
  it('en-US 缺失某已注册键时解析回退为 zh-CN 文案（非空、非键名），且默认导出对任意键恒非空非键名', () => {
    fc.assert(
      fc.property(arbRegisteredKey, key => {
        const zhValue = zhCN[key];

        // 前置不变式：基准语言文案为非空字符串且不等于键名（回退目标恒有效）。
        expect(typeof zhValue).toBe('string');
        expect(zhValue.length).toBeGreaterThan(0);
        expect(zhValue).not.toBe(key);

        // 模拟 en-US 缺失该键，再以产品同款合并表达式重建解析表。
        const rawEnUSMissing: Record<string, string> = { ...rawEnUS };
        delete rawEnUSMissing[key];
        const resolved = buildMergedEnUS(rawEnUSMissing)[key];

        // 核心：en-US 缺键 → 回退渲染 zh-CN 对应文案，而非空串或键名。
        expect(resolved).toBe(zhValue);
        expect(resolved).not.toBe('');
        expect(resolved).not.toBe(key);

        // 全域不变式：实际产品默认导出对任意已注册键的解析结果恒为非空且不等于键名
        // （en-US 含该键取 en-US 文案，否则回退 zh-CN；二者皆有效，绝不空串/键名）。
        const actual = mergedEnUS[key];
        expect(typeof actual).toBe('string');
        expect(actual.length).toBeGreaterThan(0);
        expect(actual).not.toBe(key);
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
