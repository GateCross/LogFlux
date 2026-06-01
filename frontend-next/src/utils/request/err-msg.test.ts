/**
 * 错误消息去重栈单元测试（任务 2.3 / Req 2.7）。
 *
 * 针对 `showErrorMsg` + `errMsgStack` 的具体示例与边界场景：去重抑制、关闭后可复用、
 * 关闭后延时清栈、显示器注入与 AntD 适配器回调时序。Property 2 的「对所有序列普适」
 * 不变式由任务 2.4 的属性测试覆盖，此处只补充示例/副作用维度的验证。
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  DEFAULT_CLEAR_DELAY,
  type ErrMsgState,
  type ErrorMessageDisplayer,
  createAntdMessageDisplayer,
  defaultErrMsgState,
  resetErrorMessageDisplayer,
  setErrorMessageDisplayer,
  showErrorMsg,
} from './err-msg';

/**
 * 受控显示器：不立即关闭提示，而是把每条消息的 `onClose` 暂存起来，
 * 由测试显式触发，从而精确模拟「提示未关闭 / 提示关闭」两种状态。
 */
function createControlledDisplayer() {
  const shown: string[] = [];
  const closers = new Map<string, () => void>();
  const displayer: ErrorMessageDisplayer = (message, onClose) => {
    shown.push(message);
    closers.set(message, onClose);
  };
  /** 触发某条消息的关闭回调。 */
  const close = (message: string) => {
    closers.get(message)?.();
  };
  return { displayer, shown, close };
}

describe('err-msg — showErrorMsg / errMsgStack 去重', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    // 隔离全局单例状态，避免跨用例污染。
    defaultErrMsgState.errMsgStack = [];
    resetErrorMessageDisplayer();
  });

  afterEach(() => {
    vi.useRealTimers();
    resetErrorMessageDisplayer();
    defaultErrMsgState.errMsgStack = [];
  });

  it('未关闭前对同一消息去重：只入栈一次、只弹出一次', () => {
    const state: ErrMsgState = { errMsgStack: [] };
    const { displayer, shown } = createControlledDisplayer();

    showErrorMsg(state, '网络错误', { display: displayer });
    showErrorMsg(state, '网络错误', { display: displayer });
    showErrorMsg(state, '网络错误', { display: displayer });

    expect(shown).toEqual(['网络错误']);
    expect(state.errMsgStack).toEqual(['网络错误']);
  });

  it('不同消息分别入栈且各自弹出，栈中无重复', () => {
    const state: ErrMsgState = { errMsgStack: [] };
    const { displayer, shown } = createControlledDisplayer();

    showErrorMsg(state, 'A', { display: displayer });
    showErrorMsg(state, 'B', { display: displayer });
    showErrorMsg(state, 'A', { display: displayer });

    expect(shown).toEqual(['A', 'B']);
    expect(state.errMsgStack).toEqual(['A', 'B']);
    // 不变式：无重复。
    expect(new Set(state.errMsgStack).size).toBe(state.errMsgStack.length);
  });

  it('提示关闭后该消息移出栈，可再次被提示', () => {
    const state: ErrMsgState = { errMsgStack: [] };
    const { displayer, shown, close } = createControlledDisplayer();

    showErrorMsg(state, '保存失败', { display: displayer });
    expect(state.errMsgStack).toEqual(['保存失败']);

    // 关闭提示：消息立即移出栈。
    close('保存失败');
    expect(state.errMsgStack).toEqual([]);

    // 关闭后同名消息可再次入栈/弹出。
    showErrorMsg(state, '保存失败', { display: displayer });
    expect(shown).toEqual(['保存失败', '保存失败']);
    expect(state.errMsgStack).toEqual(['保存失败']);
  });

  it('提示关闭后延时清空整个栈（兜底清理）', () => {
    const state: ErrMsgState = { errMsgStack: [] };
    const { displayer, close } = createControlledDisplayer();

    showErrorMsg(state, 'A', { display: displayer });
    showErrorMsg(state, 'B', { display: displayer });
    expect(state.errMsgStack).toEqual(['A', 'B']);

    // 关闭 A：A 移出栈，B 仍在；清栈定时器尚未到点。
    close('A');
    expect(state.errMsgStack).toEqual(['B']);

    // 未到延时：栈保持不变。
    vi.advanceTimersByTime(DEFAULT_CLEAR_DELAY - 1);
    expect(state.errMsgStack).toEqual(['B']);

    // 到达默认延时：整个栈被清空。
    vi.advanceTimersByTime(1);
    expect(state.errMsgStack).toEqual([]);
  });

  it('支持自定义清栈延时', () => {
    const state: ErrMsgState = { errMsgStack: [] };
    const { displayer, close } = createControlledDisplayer();

    showErrorMsg(state, 'X', { display: displayer, clearDelay: 1000 });
    close('X');
    expect(state.errMsgStack).toEqual([]);

    showErrorMsg(state, 'Y', { display: displayer, clearDelay: 1000 });
    expect(state.errMsgStack).toEqual(['Y']);
    close('Y');
    vi.advanceTimersByTime(1000);
    expect(state.errMsgStack).toEqual([]);
  });

  it('省略 state 时作用于全局单例 defaultErrMsgState', () => {
    const { displayer, shown } = createControlledDisplayer();

    showErrorMsg('全局错误', { display: displayer });
    showErrorMsg('全局错误', { display: displayer });

    expect(shown).toEqual(['全局错误']);
    expect(defaultErrMsgState.errMsgStack).toEqual(['全局错误']);
  });

  it('使用经 setErrorMessageDisplayer 注入的全局显示器', () => {
    const { displayer, shown } = createControlledDisplayer();
    setErrorMessageDisplayer(displayer);

    const state: ErrMsgState = { errMsgStack: [] };
    showErrorMsg(state, '注入显示器');

    expect(shown).toEqual(['注入显示器']);
    expect(state.errMsgStack).toEqual(['注入显示器']);
  });

  it('默认 no-op 显示器立即关闭，消息不会永久滞留栈中', () => {
    const state: ErrMsgState = { errMsgStack: [] };
    // 未配置显示器：默认 no-op 立即 onClose → 消息随即移出栈。
    showErrorMsg(state, '兜底');
    expect(state.errMsgStack).toEqual([]);

    // 因已移出，同名消息可再次「提示」（仍立即关闭）。
    showErrorMsg(state, '兜底');
    expect(state.errMsgStack).toEqual([]);
  });

  it('errMsgStack 被外部置空时具备健壮性', () => {
    const state = { errMsgStack: undefined } as unknown as ErrMsgState;
    const { displayer, shown } = createControlledDisplayer();

    showErrorMsg(state, '健壮性', { display: displayer });
    expect(shown).toEqual(['健壮性']);
    expect(state.errMsgStack).toEqual(['健壮性']);
  });

  describe('createAntdMessageDisplayer 适配器', () => {
    it('转调 AntD message.error，并在自动关闭时回调 onClose 驱动清栈', () => {
      // 模拟 AntD message：error(content, duration?, onClose?)，此处立即触发 onClose。
      const error = vi.fn((_content: string, _duration?: number, onClose?: () => void) => {
        onClose?.();
        return undefined;
      });
      const displayer = createAntdMessageDisplayer({ error }, 3);

      const state: ErrMsgState = { errMsgStack: [] };
      showErrorMsg(state, 'antd 错误', { display: displayer });

      expect(error).toHaveBeenCalledWith('antd 错误', 3, expect.any(Function));
      // onClose 已触发 → 消息移出栈。
      expect(state.errMsgStack).toEqual([]);
    });
  });
});
