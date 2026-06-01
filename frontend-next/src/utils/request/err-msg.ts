/**
 * 错误消息去重栈（任务 2.3 / Req 2.7）。
 *
 * 设计依据：design.md「Request_Layer」之「错误消息去重（showErrorMsg + errMsgStack）」，
 * 以及 Correctness Property 2（错误消息去重）。
 *
 * 行为对齐旧 Vue 版 `frontend/src/service/request/shared.ts` 的 `showErrorMsg`：
 *  1. 同一消息在其提示「未关闭」前不重复入栈、不重复弹出（去重）；
 *  2. 提示关闭后，先将该消息移出栈，使其可再次被提示；
 *  3. 关闭后延时（默认 5000ms）清空整个错误消息栈（兜底清理）。
 *
 * 与旧实现的关键差异：剥离对 Naive UI `window.$message` 的耦合。
 *  - 本模块为「框架无关纯逻辑」：仅维护字符串去重栈 `errMsgStack` 并按上述规则调度，
 *    「如何把消息显示给用户」通过可注入的 {@link ErrorMessageDisplayer} 抽象出去。
 *  - 真正的 UI 显示（AntD `message.error`）由 axios 装配层（任务 2.7）通过
 *    {@link setErrorMessageDisplayer} 注入 {@link createAntdMessageDisplayer} 适配器接线；
 *    本模块不 import `antd`，从而可被 Property 2 的属性测试（任务 2.4）直接覆盖，
 *    无需渲染任何组件、无需加载组件库。
 *
 * 不变式（Property 2）：在任意「展示 → 关闭 → 再展示」的操作序列下，
 * `errMsgStack` 任意时刻都不含重复消息；某消息提示未关闭前不会被重复入栈/弹出。
 */

/**
 * 错误消息去重栈所依附的最小状态结构。
 *
 * 与 design.md 的 `RequestInstanceState`（`{ errMsgStack, refreshTokenPromise }`）结构兼容：
 * `showErrorMsg` 仅依赖 `errMsgStack` 字段，因此可直接接收完整的请求实例状态，
 * 也可在测试中传入一个仅含 `errMsgStack` 的轻量对象。
 */
export interface ErrMsgState {
  /** 当前「提示未关闭」的错误消息栈（不含重复项）。 */
  errMsgStack: string[];
}

/**
 * 错误消息显示器（可注入的副作用边界）。
 *
 * 实现方负责将 `message` 展示给用户，并在该提示「关闭/离场」时调用一次 `onClose` 回调。
 * 去重栈的清理逻辑全部由本模块在 `onClose` 中驱动，显示器无需了解栈的存在。
 *
 * @param message 要展示的错误消息文本。
 * @param onClose 提示关闭时由显示器回调一次（对应旧 Naive UI 的 `onLeave`、AntD 的 `onClose`）。
 */
export type ErrorMessageDisplayer = (message: string, onClose: () => void) => void;

/** `showErrorMsg` 的可选项（用于显示器与时序注入，便于测试与装配解耦）。 */
export interface ShowErrorMsgOptions {
  /** 本次调用使用的显示器；缺省时回退到经 {@link setErrorMessageDisplayer} 配置的全局显示器。 */
  display?: ErrorMessageDisplayer;
  /** 提示关闭后清空整个错误消息栈的延时（毫秒）；缺省 {@link DEFAULT_CLEAR_DELAY}。 */
  clearDelay?: number;
}

/** 提示关闭后清空整个错误消息栈的默认延时（毫秒），与旧 Vue 版一致。 */
export const DEFAULT_CLEAR_DELAY = 5000;

/**
 * 全局默认错误消息状态（请求层单例使用）。
 *
 * 对应旧实现中挂在 axios 实例上的 `request.state.errMsgStack`。
 * 默认 `showErrorMsg`（不显式传入 `state`）即作用于此单例。
 */
export const defaultErrMsgState: ErrMsgState = { errMsgStack: [] };

/**
 * 未配置显示器时的兜底实现：仅维护去重栈、立即「关闭」（不实际展示任何 UI）。
 *
 * 之所以选择「立即 onClose」而非永不关闭：避免在显示器尚未接线（如单元测试或
 * 装配前的早期阶段）时，消息被永久滞留在栈中而再也无法被同名消息复用。
 */
const noopDisplayer: ErrorMessageDisplayer = (_message, onClose) => {
  onClose();
};

/** 当前全局显示器（由装配层注入；默认 no-op）。 */
let globalDisplayer: ErrorMessageDisplayer = noopDisplayer;

/**
 * 配置全局错误消息显示器（由 axios 装配层 / app 运行时调用，任务 2.7）。
 *
 * 例：`setErrorMessageDisplayer(createAntdMessageDisplayer(message));`
 */
export function setErrorMessageDisplayer(displayer: ErrorMessageDisplayer): void {
  globalDisplayer = displayer;
}

/** 重置全局显示器为兜底实现（主要用于测试隔离）。 */
export function resetErrorMessageDisplayer(): void {
  globalDisplayer = noopDisplayer;
}

/**
 * 展示一条去重的错误消息（Req 2.7）。
 *
 * 规则（与旧 `frontend/src/service/request/shared.ts` 一致）：
 *  - 若 `message` 已在 `state.errMsgStack` 中（提示尚未关闭）→ 抑制：不入栈、不再次弹出。
 *  - 否则 → 入栈并经显示器弹出；当该提示关闭时：
 *      1. 将该消息从栈中移除（使其可再次被提示）；
 *      2. 延时 `clearDelay` 毫秒后清空整个栈（兜底清理）。
 *
 * 纯逻辑保证：仅在 `!includes(message)` 时入栈，故 `errMsgStack` 任意时刻不含重复项。
 *
 * @param state 去重栈所在状态；缺省使用 {@link defaultErrMsgState} 单例。
 * @param message 错误消息文本。
 * @param options 显示器与时序注入项。
 */
export function showErrorMsg(message: string, options?: ShowErrorMsgOptions): void;
export function showErrorMsg(state: ErrMsgState, message: string, options?: ShowErrorMsgOptions): void;
export function showErrorMsg(
  stateOrMessage: ErrMsgState | string,
  messageOrOptions?: string | ShowErrorMsgOptions,
  maybeOptions?: ShowErrorMsgOptions,
): void {
  // 重载归一：支持 `showErrorMsg(message)` 与 `showErrorMsg(state, message, options)` 两种调用形态。
  let state: ErrMsgState;
  let message: string;
  let options: ShowErrorMsgOptions;

  if (typeof stateOrMessage === 'string') {
    state = defaultErrMsgState;
    message = stateOrMessage;
    options = (messageOrOptions as ShowErrorMsgOptions | undefined) ?? {};
  } else {
    state = stateOrMessage;
    message = messageOrOptions as string;
    options = maybeOptions ?? {};
  }

  const display = options.display ?? globalDisplayer;
  const clearDelay = options.clearDelay ?? DEFAULT_CLEAR_DELAY;

  // 防御：state 可能在外部被置空（与旧实现一致的健壮性处理）。
  if (!Array.isArray(state.errMsgStack)) {
    state.errMsgStack = [];
  }

  // 去重：同一消息提示未关闭前不重复入栈/弹出。
  if (state.errMsgStack.includes(message)) {
    return;
  }

  state.errMsgStack.push(message);

  display(message, () => {
    // 提示关闭：先移除该消息，使其可再次被提示。
    state.errMsgStack = state.errMsgStack.filter(msg => msg !== message);

    // 兜底：延时清空整个栈（关闭后延时清栈）。
    setTimeout(() => {
      state.errMsgStack = [];
    }, clearDelay);
  });
}

/**
 * AntD `message` 显示器适配器（由装配层 / app 运行时使用）。
 *
 * 不在本模块直接 import `antd`：通过结构化类型 {@link MessageApiLike} 接收 AntD 的
 * 静态 `message` 或 `App.useApp().message` 实例，保持本模块对 UI 库零耦合。
 *
 * AntD `message.error(content, duration?, onClose?)`：当提示在 `duration` 后自动关闭时
 * 回调 `onClose`，恰好对应本模块需要的「关闭后清栈」时机（等价旧 Naive UI 的 `onLeave`）。
 *
 * @param api AntD message 实例（静态 `message` 或 hook 版 `messageApi`）。
 * @param duration 提示持续秒数（缺省由 AntD 决定，默认 3s）。
 */
export interface MessageApiLike {
  error(content: string, duration?: number, onClose?: () => void): unknown;
}

export function createAntdMessageDisplayer(api: MessageApiLike, duration?: number): ErrorMessageDisplayer {
  return (message, onClose) => {
    api.error(message, duration, onClose);
  };
}
