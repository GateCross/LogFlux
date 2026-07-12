import * as fc from 'fast-check';
import { describe, expect, it } from 'vitest';

import { apiErrorMessage } from './api-error-message';

/** 非空可展示字符串（trim 后 length > 0） */
const nonEmptyStringArb = fc
  .string({ minLength: 1, maxLength: 40 })
  .filter((s) => s.trim().length > 0)
  .map((s) => s.trim());

/** 空字段候选：undefined / 非字符串 / 空白串 */
const emptyFieldArb = fc.oneof(
  fc.constant(undefined),
  fc.constant(null),
  fc.constant(0),
  fc.constant(true),
  fc.constant({}),
  fc.constant([]),
  fc.constantFrom('', '   ', '\t', '\n', '  \n  '),
);

type PayloadFields = {
  message?: unknown;
  msg?: unknown;
  error?: unknown;
};

function buildError(opts: {
  responseData?: PayloadFields | null;
  data?: PayloadFields | null;
  message?: unknown;
}): unknown {
  const err: Record<string, unknown> = {};
  if (opts.responseData !== undefined) {
    err.response = { data: opts.responseData };
  }
  if (opts.data !== undefined) {
    err.data = opts.data;
  }
  if (opts.message !== undefined) {
    err.message = opts.message;
  }
  return err;
}

/**
 * 与实现一致的期望优先级：
 * response.data.message → response.data.msg → response.data.error
 * → data.message → data.msg → data.error
 * → error.message → fallback
 */
function expectedMessage(
  responseData: PayloadFields | null | undefined,
  data: PayloadFields | null | undefined,
  message: unknown,
  fallback: string,
): string {
  const pick = (value: unknown): string | undefined => {
    if (typeof value !== 'string') return undefined;
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  };
  const fromPayload = (payload: PayloadFields | null | undefined) => {
    if (!payload || typeof payload !== 'object') return undefined;
    return pick(payload.message) ?? pick(payload.msg) ?? pick(payload.error);
  };
  return (
    fromPayload(responseData) ??
    fromPayload(data) ??
    pick(message) ??
    fallback
  );
}

describe('apiErrorMessage', () => {
  it('Property 1: 错误文案提取优先级与幂等', () => {
    fc.assert(
      fc.property(
        fc.record({
          responseMessage: fc.option(fc.oneof(nonEmptyStringArb, emptyFieldArb), {
            nil: undefined,
          }),
          responseMsg: fc.option(fc.oneof(nonEmptyStringArb, emptyFieldArb), {
            nil: undefined,
          }),
          responseError: fc.option(fc.oneof(nonEmptyStringArb, emptyFieldArb), {
            nil: undefined,
          }),
          dataMessage: fc.option(fc.oneof(nonEmptyStringArb, emptyFieldArb), {
            nil: undefined,
          }),
          dataMsg: fc.option(fc.oneof(nonEmptyStringArb, emptyFieldArb), {
            nil: undefined,
          }),
          dataError: fc.option(fc.oneof(nonEmptyStringArb, emptyFieldArb), {
            nil: undefined,
          }),
          errorMessage: fc.option(fc.oneof(nonEmptyStringArb, emptyFieldArb), {
            nil: undefined,
          }),
          fallback: nonEmptyStringArb,
        }),
        (fields) => {
          const responseData: PayloadFields = {
            message: fields.responseMessage,
            msg: fields.responseMsg,
            error: fields.responseError,
          };
          const data: PayloadFields = {
            message: fields.dataMessage,
            msg: fields.dataMsg,
            error: fields.dataError,
          };
          const error = buildError({
            responseData,
            data,
            message: fields.errorMessage,
          });

          const expected = expectedMessage(
            responseData,
            data,
            fields.errorMessage,
            fields.fallback,
          );
          const first = apiErrorMessage(error, fields.fallback);
          const second = apiErrorMessage(error, fields.fallback);
          const third = apiErrorMessage(error, fields.fallback);

          expect(first).toBe(expected);
          // 幂等：同一输入多次调用结果相同
          expect(second).toBe(first);
          expect(third).toBe(first);
        },
      ),
      { numRuns: 100 },
    );
  });

  it('合法优先级样例：response.data.message 优先于更低字段', () => {
    const error = buildError({
      responseData: {
        message: '  from-response-message  ',
        msg: 'from-response-msg',
        error: 'from-response-error',
      },
      data: { message: 'from-data-message' },
      message: 'from-error-message',
    });
    expect(apiErrorMessage(error, 'fallback')).toBe('from-response-message');
  });

  it('合法优先级样例：response.data.msg 次于 message，高于 error', () => {
    const error = buildError({
      responseData: {
        message: '   ',
        msg: 'from-response-msg',
        error: 'from-response-error',
      },
      data: { message: 'from-data-message' },
      message: 'from-error-message',
    });
    expect(apiErrorMessage(error, 'fallback')).toBe('from-response-msg');
  });

  it('合法优先级样例：data 级字段在 response.data 全空时生效', () => {
    const error = buildError({
      responseData: { message: '', msg: null, error: undefined },
      data: { message: '', msg: 'from-data-msg', error: 'from-data-error' },
      message: 'from-error-message',
    });
    expect(apiErrorMessage(error, 'fallback')).toBe('from-data-msg');
  });

  it('合法优先级样例：error.message 在载荷全空时生效', () => {
    const error = buildError({
      responseData: {},
      data: {},
      message: '  network timeout  ',
    });
    expect(apiErrorMessage(error, 'fallback')).toBe('network timeout');
  });

  it('全空字段回退 fallback', () => {
    const cases: unknown[] = [
      null,
      undefined,
      {},
      buildError({
        responseData: { message: '', msg: '  ', error: null },
        data: { message: undefined, msg: '', error: '   ' },
        message: '',
      }),
      buildError({
        responseData: null,
        data: null,
        message: '   ',
      }),
    ];
    for (const error of cases) {
      expect(apiErrorMessage(error, '操作失败')).toBe('操作失败');
    }
  });

  it('同一输入多次调用结果相同（幂等示例）', () => {
    const error = buildError({
      responseData: { msg: '  stable  ' },
      message: 'ignored',
    });
    const a = apiErrorMessage(error, 'fallback');
    const b = apiErrorMessage(error, 'fallback');
    const c = apiErrorMessage(error, 'fallback');
    expect(a).toBe('stable');
    expect(b).toBe(a);
    expect(c).toBe(a);
  });

  // 合法 + 边界样例

  it('合法输入：response.data.error 在 message/msg 空时生效', () => {
    const error = {
      response: { data: { message: '', msg: '  ', error: '  服务不可用  ' } },
      data: { message: 'should-not-win' },
      message: 'should-not-win',
    };
    expect(apiErrorMessage(error, '操作失败')).toBe('服务不可用');
  });

  it('边界/非法：非对象错误与空白 message 回退 fallback', () => {
    expect(apiErrorMessage('plain-string-error', '默认错误')).toBe('默认错误');
    expect(apiErrorMessage(42, '默认错误')).toBe('默认错误');
    expect(apiErrorMessage({ message: '   ' }, '默认错误')).toBe('默认错误');
    expect(
      apiErrorMessage(
        { response: { data: 'not-an-object' }, message: null },
        '默认错误',
      ),
    ).toBe('默认错误');
  });
});
