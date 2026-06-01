/**
 * 中英文案键集合一致属性测试（任务 5.3）。
 *
 * 对应 design.md 的 Correctness Property 9：
 *   *For all* 已注册文案键，zh-CN 与 en-US 两套文案的键集合相等（对称差为空），
 *   即不存在仅出现在一套语言中的键。
 *
 * 重要说明（键来源的选择）：
 *  - `en-US.ts` 的**默认导出**为 `{ ...zhCN, ...messages }`——它把 zh-CN 合并作为缺键回退基准
 *    （fallbackLocale=zh-CN，Req 5.7）。若以默认导出参与比较，其键集合恒为 zh-CN ∪ en-US，
 *    会**掩盖** en-US 真实缺失的键，使本属性失去意义。
 *  - 因此本测试以 en-US 的**具名导出 `messages`**（未合并回退基准的原始 en-US 文案）参与比较，
 *    与 zh-CN 默认导出（其原始文案）对齐，方能真实检验「两套原始键集合相等」。
 *
 * 测试以「从两套键集合的并集中采样任意键」的方式生成输入，断言该键同时存在于
 * zh-CN 与 en-US 两套原始文案中（即并集中的每个键都属于交集 → 对称差为空）。
 * 由于并集是有限集合，`fc.constantFrom` 在 100 次迭代中可充分覆盖其元素。
 */
import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import zhCNMessages from './zh-CN';
// 注意：取 en-US 的**具名导出**（原始 en-US 键集合），而非合并 zh-CN 回退后的默认导出。
import { messages as enUSMessages } from './en-US';

/** zh-CN 原始文案键集合。 */
const zhKeys = Object.keys(zhCNMessages);
/** en-US 原始文案键集合（未合并回退基准）。 */
const enKeys = Object.keys(enUSMessages);

const zhKeySet = new Set(zhKeys);
const enKeySet = new Set(enKeys);

/** 两套语言键的并集（去重），作为属性的采样输入空间。 */
const unionKeys = [...new Set([...zhKeys, ...enKeys])];

/** 从并集中采样任意一个已注册文案键。 */
const arbUnionKey: fc.Arbitrary<string> = fc.constantFrom(...unionKeys);

describe('i18n — 中英文案键集合一致（Property 9）', () => {
  // Feature: frontend-umijs-max-migration, Property 9: 中英文案键集合一致
  // Validates: Requirements 5.1
  it('并集中的任意已注册键同时存在于 zh-CN 与 en-US（对称差为空）', () => {
    fc.assert(
      fc.property(arbUnionKey, key => {
        // 并集中的每个键都必须同时属于两套语言 → 不存在仅出现在一套语言中的键。
        expect(zhKeySet.has(key)).toBe(true);
        expect(enKeySet.has(key)).toBe(true);
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
