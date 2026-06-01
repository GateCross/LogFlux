/**
 * 偏好回退至默认 —— 属性测试（任务 3.3，Property 16）。
 *
 * 对应 design.md 的 Correctness Property 16，验证 `loadPreferences` 这一「框架无关纯逻辑」满足：
 * 对任意「无法产出合法偏好」的持久化输入，恒回退至默认偏好、返回 `usedDefault=true`，
 * 且永不抛出异常（不中断应用初始化，Req 15.4 / 15.5 / 6.7）。
 *
 * 说明：本属性与任务 3.2 的 Property 15（往返一致）互补，分置独立文件以避免对
 * `preferences.test.ts` 的并发写入冲突。
 *
 * 生成器覆盖（设计要求的四类「无法产出合法偏好」的输入，均「构造即非法」）：
 *  1. invalid-json    —— 随机非 JSON 字符串（`JSON.parse` 必然抛错）。
 *  2. truncated-json  —— 合法序列化结果被截断后的残缺字符串（语法非法）。
 *  3. wrong-shape     —— 合法 JSON 但非纯对象（基本类型 / 数组），永不可能是合法偏好。
 *  4. invalid-object  —— 可反序列化为纯对象，但缺失必需字段 / 取值越界（可叠加多余字段，
 *                        验证多余字段不能「挽救」非法对象）。
 *
 * 期望由「生成它的分区」独立决定，与被测实现内部细节解耦；分支自检用与实现无关的
 * `JSON.parse` / 纯对象判定确认确实命中 Req 15.4（语法非法）或 Req 15.5（可解析但非法）路径。
 */
import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import {
  createDefaultPreferences,
  DEFAULT_PREFERENCES,
  LANGS,
  LAYOUT_MODES,
  loadPreferences,
  MAX_WATERMARK_TEXT_LENGTH,
  THEME_SCHEMES,
  type UserPreferences,
} from './preferences';

/** 测试内部使用的可变对象别名（仅用于在克隆体上施加非法变更）。 */
type Loose = Record<string, any>;

/** 独立判定：`raw` 经 `JSON.parse` 是否抛错（与被测实现无关的语法合法性预言）。 */
function jsonParseThrows(raw: string): boolean {
  try {
    JSON.parse(raw);
    return false;
  } catch {
    return true;
  }
}

/** 独立判定：是否为「纯对象」（非 null、非数组）。 */
function isPlainObject(value: unknown): boolean {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

// ---------------------------------------------------------------------------
// 合法偏好基线生成器（仅用于派生「截断」与「单点非法」的输入；其自身保证合法，
// 从而确保派生输入的非法性唯一来自我们施加的变更）。
// ---------------------------------------------------------------------------

/** 合法主色样本（hex / rgb / rgba，均能通过 isValidThemeColor）。 */
const VALID_COLORS = ['#fff', '#ffff', '#ffffff', '#11223344', 'rgb(0,0,0)', 'rgba(255, 255, 255, 0.3)', '#646cff'];

const arbValidPreferences: fc.Arbitrary<UserPreferences> = fc.record({
  theme: fc.record({
    themeScheme: fc.constantFrom(...THEME_SCHEMES),
    themeColor: fc.constantFrom(...VALID_COLORS),
    grayscale: fc.boolean(),
    colourWeakness: fc.boolean(),
  }),
  lang: fc.constantFrom(...LANGS),
  layout: fc.record({
    mode: fc.constantFrom(...LAYOUT_MODES),
    siderCollapse: fc.boolean(),
  }),
  watermark: fc.record({
    visible: fc.boolean(),
    text: fc.string({ maxLength: MAX_WATERMARK_TEXT_LENGTH }),
  }),
});

// ---------------------------------------------------------------------------
// 分区 1：随机非 JSON 字符串（JSON.parse 抛错）。
// ---------------------------------------------------------------------------

const arbInvalidJsonString = fc
  .oneof(
    fc.string(),
    fc.constantFrom('{', '[', '{"a":', '{not valid json', 'undefined', "{'a':1}", '{,}', '[1,2', 'NaN', '}{', 'null,'),
  )
  .filter(jsonParseThrows)
  .map(raw => ({ kind: 'invalid-json' as const, raw }));

// ---------------------------------------------------------------------------
// 分区 2：截断的 JSON（合法偏好序列化后取前缀，过滤出语法非法者）。
// ---------------------------------------------------------------------------

const arbTruncatedJson = arbValidPreferences
  .chain(p => {
    const full = JSON.stringify(p);
    return fc.integer({ min: 1, max: full.length - 1 }).map(n => full.slice(0, n));
  })
  .filter(jsonParseThrows)
  .map(raw => ({ kind: 'truncated-json' as const, raw }));

// ---------------------------------------------------------------------------
// 分区 3：合法 JSON 但非纯对象（基本类型 / 数组），永不可能是合法偏好。
// ---------------------------------------------------------------------------

const arbWrongShape = fc
  .oneof(
    fc.integer(),
    fc.double({ min: -1_000_000, max: 1_000_000, noNaN: true }),
    fc.boolean(),
    fc.constant(null),
    fc.string(),
    fc.array(fc.oneof(fc.integer(), fc.string(), fc.boolean(), fc.constant(null)), { maxLength: 5 }),
  )
  .map(value => ({ kind: 'wrong-shape' as const, raw: JSON.stringify(value) }));

// ---------------------------------------------------------------------------
// 分区 4：可解析为纯对象但缺字段 / 取值越界（可叠加多余字段）。
// ---------------------------------------------------------------------------

const INVALID_COLORS = ['', 'red', 'not-a-color', '#12', '#xyzxyz', 'rgb(256,0,0)', 'rgba(0,0,0,2)', 'hsl(0,0,0)'];
const INVALID_LANGS = ['', 'fr-FR', 'EN', 'zh', 'en', 'jp', 'zh_CN'];
const INVALID_MODES = ['', 'unknown', 'grid', 'mixed', 'Vertical'];
const INVALID_SCHEMES = ['', 'bright', 'system', 'Light', 'DARK'];
const NON_BOOLEANS: unknown[] = [0, 1, 2, 'true', 'false', null, 'yes'];
const NON_OBJECTS: unknown[] = [0, 42, 'str', null, true, [] as unknown, [1, 2] as unknown];

/** 缺失必需字段（删除顶层或嵌套的某个必需键）。 */
const arbMissing = fc.constantFrom<(o: Loose) => void>(
  o => {
    delete o.theme;
  },
  o => {
    delete o.lang;
  },
  o => {
    delete o.layout;
  },
  o => {
    delete o.watermark;
  },
  o => {
    delete o.theme.themeScheme;
  },
  o => {
    delete o.theme.themeColor;
  },
  o => {
    delete o.theme.grayscale;
  },
  o => {
    delete o.theme.colourWeakness;
  },
  o => {
    delete o.layout.mode;
  },
  o => {
    delete o.layout.siderCollapse;
  },
  o => {
    delete o.watermark.visible;
  },
);

/** 取值越界 / 类型错误（单点替换为确定非法的值）。 */
const arbOutOfRange = fc.oneof(
  fc.constantFrom(...INVALID_COLORS).map(v => (o: Loose) => {
    o.theme.themeColor = v;
  }),
  fc.constantFrom(...INVALID_LANGS).map(v => (o: Loose) => {
    o.lang = v;
  }),
  fc.constantFrom(...INVALID_MODES).map(v => (o: Loose) => {
    o.layout.mode = v;
  }),
  fc.constantFrom(...INVALID_SCHEMES).map(v => (o: Loose) => {
    o.theme.themeScheme = v;
  }),
  fc.constantFrom(...NON_BOOLEANS).map(v => (o: Loose) => {
    o.theme.grayscale = v;
  }),
  fc.constantFrom(...NON_BOOLEANS).map(v => (o: Loose) => {
    o.theme.colourWeakness = v;
  }),
  fc.constantFrom(...NON_BOOLEANS).map(v => (o: Loose) => {
    o.layout.siderCollapse = v;
  }),
  fc.constantFrom(...NON_BOOLEANS).map(v => (o: Loose) => {
    o.watermark.visible = v;
  }),
  // 超长水印文案（含特殊 / Unicode 字符），长度必然 > 上限。
  fc.string({ minLength: 0, maxLength: 5 }).map(prefix => (o: Loose) => {
    o.watermark.text = prefix.padEnd(MAX_WATERMARK_TEXT_LENGTH + 1, 'x★');
  }),
);

/** 子对象类型错误（theme / layout / watermark 被替换为非纯对象）。 */
const arbBadSubObject = fc.oneof(
  fc.constantFrom(...NON_OBJECTS).map(v => (o: Loose) => {
    o.theme = v;
  }),
  fc.constantFrom(...NON_OBJECTS).map(v => (o: Loose) => {
    o.layout = v;
  }),
  fc.constantFrom(...NON_OBJECTS).map(v => (o: Loose) => {
    o.watermark = v;
  }),
);

const arbMutation: fc.Arbitrary<(o: Loose) => void> = fc.oneof(arbMissing, arbOutOfRange, arbBadSubObject);

/** 多余字段（键名均不与必需键冲突，因此不会「挽救」已非法的对象）。 */
const EXTRA_KEYS = ['foo', 'bar', '_extra', 'meta', 'version', 'unknownField'];
const arbExtraKeys = fc.dictionary(fc.constantFrom(...EXTRA_KEYS), fc.jsonValue(), { maxKeys: 4 });

const arbInvalidObject = fc
  .tuple(arbValidPreferences, arbMutation, fc.option(arbExtraKeys, { nil: undefined }))
  .map(([base, mutate, extra]) => {
    const obj = JSON.parse(JSON.stringify(base)) as Loose;
    mutate(obj);
    if (extra) {
      Object.assign(obj, extra);
    }
    return { kind: 'invalid-object' as const, raw: JSON.stringify(obj) };
  });

/** 四个分区的并集：任意「无法产出合法偏好」的持久化输入。 */
const arbFallbackCase = fc.oneof(arbInvalidJsonString, arbTruncatedJson, arbWrongShape, arbInvalidObject);

describe('preferences — 偏好回退至默认（Property 16）', () => {
  // Feature: frontend-umijs-max-migration, Property 16: 偏好回退至默认
  // Validates: Requirements 15.4, 15.5, 6.7
  it('对任意无法产出合法偏好的输入，loadPreferences 回退默认、usedDefault=true 且不抛异常', () => {
    fc.assert(
      fc.property(arbFallbackCase, ({ kind, raw }) => {
        const run = () => loadPreferences(raw);

        // 永不抛出 —— 不中断应用初始化（Req 15.4 / 15.5 / 6.7）。
        expect(run).not.toThrow();

        const result = run();

        // 回退至默认偏好，并向调用方返回「已使用默认」的指示。
        expect(result.usedDefault).toBe(true);
        expect(result.preferences).toEqual(DEFAULT_PREFERENCES);
        expect(result.preferences).toEqual(createDefaultPreferences());

        // 分支自检（与实现解耦）：确认输入确实命中预期的回退路径。
        if (kind === 'invalid-json' || kind === 'truncated-json') {
          // Req 15.4：语法非法、无法反序列化。
          expect(jsonParseThrows(raw)).toBe(true);
        } else {
          // Req 15.5：可反序列化，但不是合法偏好对象。
          expect(jsonParseThrows(raw)).toBe(false);
          const parsed = JSON.parse(raw);
          // wrong-shape：非纯对象；invalid-object：纯对象但缺字段 / 越界。
          expect(isPlainObject(parsed)).toBe(kind === 'invalid-object');
        }
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
