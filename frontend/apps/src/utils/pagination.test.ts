import * as fc from 'fast-check';
import { describe, expect, it } from 'vitest';

import { fromPageResult, toPageParams } from './pagination';

/** 合法页码 / 页大小：≥1 的有限整数 */
const positivePageIntArb = fc.integer({ min: 1, max: 10_000 });

/** 合法非负整数 total */
const nonNegativeTotalArb = fc.integer({ min: 0, max: 1_000_000 });

/** 非法 total：非「非负整数 number」 */
const illegalTotalArb = fc.oneof(
  fc.constant(undefined),
  fc.constant(null),
  fc.constant(Number.NaN),
  fc.constant(Number.POSITIVE_INFINITY),
  fc.constant(Number.NEGATIVE_INFINITY),
  fc.double({ min: Number.MIN_VALUE, max: 1e6, noNaN: true }).filter(
    (n) => !Number.isInteger(n),
  ),
  fc.integer({ min: -10_000, max: -1 }),
  fc.string(),
  fc.boolean(),
  fc.constant({}),
  fc.constant([]),
);

/** 小于 1 或非有限的 page/pageSize 输入（应归一化为 1） */
const denormalPageInputArb = fc.oneof(
  fc.integer({ min: -10_000, max: 0 }),
  fc.constant(Number.NaN),
  fc.constant(Number.POSITIVE_INFINITY),
  fc.constant(Number.NEGATIVE_INFINITY),
  fc.double({ min: -1000, max: 0.999, noNaN: true }),
);

describe('pagination', () => {
  it('Property 2: 分页映射往返与边界 — 合法往返与总页数语义', () => {
    fc.assert(
      fc.property(
        positivePageIntArb,
        positivePageIntArb,
        nonNegativeTotalArb,
        (page, pageSize, total) => {
          // UI → 请求参数：合法整数保持不变
          const params = toPageParams({ page, pageSize });
          expect(params.page).toBe(page);
          expect(params.pageSize).toBe(pageSize);

          // 响应 total → UI total：合法非负整数保持不变
          const ui = fromPageResult({ total });
          expect(ui.total).toBe(total);

          // 总页数语义：ceil(total / pageSize)（total=0 时为 0）
          const totalPages =
            ui.total === 0 ? 0 : Math.ceil(ui.total / params.pageSize);
          expect(totalPages).toBe(
            total === 0 ? 0 : Math.ceil(total / pageSize),
          );
          // 若有数据，当前页不应超出总页数语义上的上界（仅校验可计算性）
          if (totalPages > 0) {
            expect(params.page).toBeGreaterThanOrEqual(1);
            expect(params.pageSize).toBeGreaterThanOrEqual(1);
          }
        },
      ),
      { numRuns: 100 },
    );
  });

  it('Property 2: 分页映射往返与边界 — 非法 total → 0', () => {
    fc.assert(
      fc.property(illegalTotalArb, (total) => {
        // 直接传非法 total
        expect(fromPageResult({ total })).toEqual({ total: 0 });
        // 缺失 total / null / undefined 响应体
        expect(fromPageResult(null)).toEqual({ total: 0 });
        expect(fromPageResult(undefined)).toEqual({ total: 0 });
        expect(fromPageResult({})).toEqual({ total: 0 });
      }),
      { numRuns: 100 },
    );
  });

  it('Property 2: 分页映射往返与边界 — page/pageSize 归一化为 1', () => {
    fc.assert(
      fc.property(
        denormalPageInputArb,
        denormalPageInputArb,
        positivePageIntArb,
        positivePageIntArb,
        (badPage, badPageSize, goodPage, goodPageSize) => {
          // 双侧均非法 → 均归一化为 1
          expect(toPageParams({ page: badPage, pageSize: badPageSize })).toEqual(
            { page: 1, pageSize: 1 },
          );

          // 仅 page 非法
          expect(
            toPageParams({ page: badPage, pageSize: goodPageSize }),
          ).toEqual({ page: 1, pageSize: goodPageSize });

          // 仅 pageSize 非法
          expect(
            toPageParams({ page: goodPage, pageSize: badPageSize }),
          ).toEqual({ page: goodPage, pageSize: 1 });
        },
      ),
      { numRuns: 100 },
    );
  });

  // 合法 + 边界样例

  it('合法输入：toPageParams 保持 page/pageSize，fromPageResult 保持 total', () => {
    expect(toPageParams({ page: 2, pageSize: 20 })).toEqual({
      page: 2,
      pageSize: 20,
    });
    expect(fromPageResult({ total: 100 })).toEqual({ total: 100 });
    // total=0 为合法非负整数，应原样返回
    expect(fromPageResult({ total: 0 })).toEqual({ total: 0 });
  });

  it('边界/非法：page/pageSize < 1 或非有限 → 归一化为 1；非法 total → 0', () => {
    expect(toPageParams({ page: 0, pageSize: -5 })).toEqual({
      page: 1,
      pageSize: 1,
    });
    expect(toPageParams({ page: Number.NaN, pageSize: 10 })).toEqual({
      page: 1,
      pageSize: 10,
    });
    expect(toPageParams({ page: 3.9, pageSize: 15.2 })).toEqual({
      page: 3,
      pageSize: 15,
    });
    expect(fromPageResult({ total: undefined })).toEqual({ total: 0 });
    expect(fromPageResult({ total: -1 })).toEqual({ total: 0 });
    expect(fromPageResult({ total: 1.5 })).toEqual({ total: 0 });
    expect(fromPageResult(null)).toEqual({ total: 0 });
    expect(fromPageResult(undefined)).toEqual({ total: 0 });
  });
});
