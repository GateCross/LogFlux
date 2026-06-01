/**
 * 测试框架冒烟测试（任务 1.3）。
 *
 * 目的：验证 vitest + fast-check + @testing-library/react + jsdom 测试链路可用，
 * 且 `@/*` 路径别名在测试中与 UmiJS 应用一致地解析。
 * 这是最小化的“证明工具链可用”的测试，不针对业务逻辑。
 */
import { render, screen } from '@testing-library/react';
import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
// 经由 `@/*` 别名导入，验证别名解析与 Umi 运行时一致。
import access from '@/access';
import { PBT_ASSERT_OPTIONS, PBT_RUNS } from '@/test/pbt';

describe('测试框架冒烟', () => {
  it('vitest 基础断言可用', () => {
    expect(1 + 1).toBe(2);
  });

  // Feature: frontend-umijs-max-migration, Property 0 (smoke): 加法交换律
  it('fast-check 属性测试可用（≥100 次迭代）', () => {
    expect(PBT_RUNS).toBeGreaterThanOrEqual(100);
    fc.assert(
      fc.property(fc.integer(), fc.integer(), (a, b) => {
        return a + b === b + a;
      }),
      PBT_ASSERT_OPTIONS,
    );
  });

  it('@/ 路径别名解析与 Umi 一致', () => {
    expect(typeof access).toBe('function');
    expect(access({ currentUser: { roles: ['R_SUPER'] } })).toEqual({ isSuperRole: true });
  });

  it('@testing-library/react + jsdom 渲染可用', () => {
    render(<button type="button">登录</button>);
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument();
  });
});
