/**
 * 错误消息去重属性测试（任务 2.4 / Property 2 / Req 2.7）。
 *
 * 设计依据：design.md「Correctness Properties」之 Property 2（错误消息去重）。
 *
 * 被测纯逻辑：`showErrorMsg` + `errMsgStack`（`err-msg.ts`）。
 *
 * 属性陈述（单一属性，覆盖任意「展示 / 关闭」操作序列）：
 *  - 任意时刻 `errMsgStack` 中不含重复消息；
 *  - 某消息的提示「未关闭」前，对同名消息的再次 `showErrorMsg` 会被抑制
 *    （不重复入栈、不重复经显示器弹出）；
 *  - 该提示「关闭」后，同名消息方可再次被入栈/弹出（可复用）。
 *
 * 测试手法：用「受控显示器」捕获每条消息的 `onClose`（不自动关闭），由测试按生成的
 * 操作序列显式触发关闭，从而精确区分「提示未关闭 / 已关闭」两种状态。使用 fake timers
 * 钳制 `onClose` 内部的兜底清栈 `setTimeout`（默认 5000ms）——本属性聚焦去重不变式，
 * 兜底清栈的时序行为由 `err-msg.test.ts` 的示例单测覆盖；此处不推进定时器，确保
 * `errMsgStack` 恒等于「当前未关闭消息集合」。
 */
import fc from 'fast-check';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import { type ErrMsgState, type ErrorMessageDisplayer, resetErrorMessageDisplayer, showErrorMsg } from './err-msg';

/** 操作序列中的单步：展示某消息，或关闭某消息的提示。 */
type Op = { type: 'show'; message: string } | { type: 'close'; message: string };

/**
 * 消息生成器：以「小消息池」为主、辅以任意 Unicode 串，
 * 使同名消息频繁碰撞，从而真正覆盖去重与「关闭后复用」路径，
 * 同时覆盖空串、特殊/Unicode 字符等边界取值。
 */
const messageArb = fc.oneof(
  { weight: 4, arbitrary: fc.constantFrom('网络错误', '保存失败', 'A', 'B', '') },
  { weight: 1, arbitrary: fc.fullUnicodeString() },
);

const showOpArb: fc.Arbitrary<Op> = messageArb.map((message) => ({ type: 'show', message }));
const closeOpArb: fc.Arbitrary<Op> = messageArb.map((message) => ({ type: 'close', message }));
const opsArb = fc.array(fc.oneof(showOpArb, closeOpArb), { minLength: 1, maxLength: 50 });

describe('err-msg 错误消息去重（Property 2）', () => {
  beforeEach(() => {
    // fake timers 钳制兜底清栈定时器；reset 全局显示器避免跨用例污染（本测试始终显式注入 display）。
    vi.useFakeTimers();
    resetErrorMessageDisplayer();
  });

  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
    resetErrorMessageDisplayer();
  });

  // Feature: frontend-umijs-max-migration, Property 2: 错误消息去重
  // Validates: Requirements 2.7
  it('任意展示/关闭序列下：栈无重复、提示未关闭前去重、关闭后可复用', () => {
    fc.assert(
      fc.property(opsArb, (ops) => {
        const state: ErrMsgState = { errMsgStack: [] };

        // 受控显示器：记录每条消息的总弹出次数，并暂存其 onClose 回调（不自动关闭）。
        const displayCount = new Map<string, number>();
        const closers = new Map<string, () => void>();
        const displayer: ErrorMessageDisplayer = (message, onClose) => {
          displayCount.set(message, (displayCount.get(message) ?? 0) + 1);
          closers.set(message, onClose);
        };

        // 模型：当前「提示未关闭」的消息集合（应与 errMsgStack 内容一一对应）。
        const open = new Set<string>();

        const assertInvariants = () => {
          // 不变式①：栈内无重复消息。
          expect(new Set(state.errMsgStack).size).toBe(state.errMsgStack.length);
          // 不变式②：栈内容恰为「当前未关闭消息集合」。
          expect(new Set(state.errMsgStack)).toEqual(open);
        };

        assertInvariants();

        for (const op of ops) {
          if (op.type === 'show') {
            const wasOpen = open.has(op.message);
            const prevCount = displayCount.get(op.message) ?? 0;

            showErrorMsg(state, op.message, { display: displayer });

            if (wasOpen) {
              // 去重：提示未关闭前，同名消息既不再入栈也不再弹出。
              expect(displayCount.get(op.message) ?? 0).toBe(prevCount);
              expect(state.errMsgStack.filter((m) => m === op.message).length).toBe(1);
            } else {
              // 新消息（含「关闭后复用」）：入栈一次并弹出一次。
              open.add(op.message);
              expect(displayCount.get(op.message) ?? 0).toBe(prevCount + 1);
              expect(state.errMsgStack).toContain(op.message);
            }
          } else if (open.has(op.message)) {
            // 关闭一条当前未关闭的提示：触发其 onClose，使该消息移出栈、可再次被提示。
            closers.get(op.message)?.();
            open.delete(op.message);
          }
          // 关闭一条本就未打开的消息：无对应在途提示，视为 no-op（与 Request_Layer 实际调用语义一致）。

          assertInvariants();
        }
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
