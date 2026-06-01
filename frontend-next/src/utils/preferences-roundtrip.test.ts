/**
 * 用户偏好序列化往返一致 —— 属性测试（任务 3.2，Property 15，最高优先级）。
 *
 * 对应 design.md 的 Correctness Property 15，验证 `serializePreferences` /
 * `deserializePreferences` 这一「框架无关纯逻辑」满足：对任意「合法用户偏好对象」，
 * 先序列化为字符串（Req 15.1）再反序列化为对象（Req 15.2），所得对象在全部字段
 * （含嵌套字段）上与原对象逐一深度相等（Req 15.3）。
 *
 * 生成器覆盖设计要求的边界情形：
 *  - 主题：明暗方案 / 布局模式 / 语言枚举全集；合法主色（hex 3/4/6/8 位、rgb、rgba）。
 *  - 布尔字段：grayscale / colourWeakness / siderCollapse / watermark.visible 全取值。
 *  - 可选水印文案：
 *      · 省略（空可选字段，键真实缺席）；
 *      · 空字符串；
 *      · 最长允许字符串（长度恰为 MAX_WATERMARK_TEXT_LENGTH）；
 *      · 含特殊 / Unicode 字符（含 JSON 元字符、控制字符、emoji、CJK、行分隔符等）。
 *
 * 为确保生成的偏好确实落在「合法输入空间」（Req 15.3 限定为「合法偏好对象」），
 * 测试内以与往返实现解耦的 `isValidPreferences` 作前置自检。
 */
import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { PBT_ASSERT_OPTIONS } from '@/test/pbt';
import {
  deserializePreferences,
  isValidPreferences,
  LANGS,
  LAYOUT_MODES,
  MAX_WATERMARK_TEXT_LENGTH,
  serializePreferences,
  THEME_SCHEMES,
  type UserPreferences,
} from './preferences';

// ---------------------------------------------------------------------------
// 合法主色生成器（hex / rgb / rgba，均能通过 isValidThemeColor）。
// ---------------------------------------------------------------------------

const HEX_DIGITS = '0123456789abcdefABCDEF'.split('');

/** 合法十六进制颜色：`#rgb` / `#rgba` / `#rrggbb` / `#rrggbbaa`。 */
const arbHexColor = fc
  .tuple(
    fc.constantFrom(3, 4, 6, 8),
    fc.array(fc.constantFrom(...HEX_DIGITS), { minLength: 8, maxLength: 8 }),
  )
  .map(([len, digits]) => `#${digits.slice(0, len).join('')}`);

/** 0~255 的整数通道。 */
const arbByte = fc.integer({ min: 0, max: 255 });

/** 0~1 的合法 alpha（覆盖整数与小数形态，均匹配 `^(0|1|0?\.\d+)$`）。 */
const arbAlpha = fc.constantFrom('0', '1', '0.5', '0.25', '0.999', '.5', '0.0');

/** 合法 `rgb(r,g,b)`。 */
const arbRgbColor = fc
  .tuple(arbByte, arbByte, arbByte)
  .map(([r, g, b]) => `rgb(${r},${g},${b})`);

/** 合法 `rgba(r,g,b,a)`。 */
const arbRgbaColor = fc
  .tuple(arbByte, arbByte, arbByte, arbAlpha)
  .map(([r, g, b, a]) => `rgba(${r}, ${g}, ${b}, ${a})`);

const arbValidColor = fc.oneof(arbHexColor, arbRgbColor, arbRgbaColor);

// ---------------------------------------------------------------------------
// 合法水印文案生成器（覆盖空 / 最长 / 特殊 + Unicode 字符边界）。
// ---------------------------------------------------------------------------

/** JSON 元字符、控制字符、行分隔符与典型 Unicode 取值，用于压测序列化转义往返。 */
const SPECIAL_UNITS = [
  '"',
  '\\',
  '/',
  '\n',
  '\r',
  '\t',
  '\b',
  '\f',
  '\u0000',
  '\u001f',
  '\u2028',
  '\u2029',
  '<',
  '>',
  '&',
  "'",
  '🌊',
  '日志',
  '★',
  'é',
  '𝕏',
];

/** 单个「字符单元」：特殊单元 / 任意 Unicode 码点 / 任意 BMP 字符。 */
const arbTextUnit = fc.oneof(
  fc.constantFrom(...SPECIAL_UNITS),
  fc.fullUnicode(),
  fc.char(),
);

/** 任意长度 ≤ 上限的水印文案（含特殊 / Unicode 字符；按 UTF-16 码元切到上限内）。 */
const arbBoundedText = fc
  .array(arbTextUnit, { maxLength: MAX_WATERMARK_TEXT_LENGTH })
  .map(units => units.join('').slice(0, MAX_WATERMARK_TEXT_LENGTH));

/** 恰为「最长允许字符串」（长度严格等于上限）的水印文案。 */
const arbMaxLengthText = fc
  .array(arbTextUnit, { minLength: MAX_WATERMARK_TEXT_LENGTH, maxLength: MAX_WATERMARK_TEXT_LENGTH * 2 })
  .map(units => units.join('').slice(0, MAX_WATERMARK_TEXT_LENGTH));

const arbWatermarkText = fc.oneof(fc.constant(''), arbBoundedText, arbMaxLengthText);

/** 水印偏好：覆盖「省略可选 text（键缺席）」与「含 text」两类。 */
const arbWatermark: fc.Arbitrary<UserPreferences['watermark']> = fc.oneof(
  fc.record({ visible: fc.boolean() }),
  fc.record({ visible: fc.boolean(), text: arbWatermarkText }),
);

// ---------------------------------------------------------------------------
// 合法用户偏好生成器（涵盖主题 / 语言 / 布局 / 水印的全部允许取值与边界）。
// ---------------------------------------------------------------------------

const arbValidPreferences: fc.Arbitrary<UserPreferences> = fc.record({
  theme: fc.record({
    themeScheme: fc.constantFrom(...THEME_SCHEMES),
    themeColor: arbValidColor,
    grayscale: fc.boolean(),
    colourWeakness: fc.boolean(),
  }),
  lang: fc.constantFrom(...LANGS),
  layout: fc.record({
    mode: fc.constantFrom(...LAYOUT_MODES),
    siderCollapse: fc.boolean(),
  }),
  watermark: arbWatermark,
});

describe('preferences — 用户偏好序列化往返一致（Property 15）', () => {
  // Feature: frontend-umijs-max-migration, Property 15: 用户偏好序列化往返一致
  // Validates: Requirements 15.1, 15.2, 15.3
  it('对任意合法偏好，serialize→deserialize 在全部字段（含嵌套）上深度相等', () => {
    fc.assert(
      fc.property(arbValidPreferences, original => {
        // 前置自检（与往返实现解耦）：确认生成的偏好确属「合法偏好对象」（Req 15.3 限定的输入空间）。
        expect(isValidPreferences(original)).toBe(true);

        // Req 15.1：序列化产物为字符串。
        const serialized = serializePreferences(original);
        expect(typeof serialized).toBe('string');

        // Req 15.2 / 15.3：反序列化还原为对象，并在全部字段（含嵌套）上逐一深度相等。
        const roundTripped = deserializePreferences(serialized);
        expect(roundTripped).toStrictEqual(original);

        // 可选字段的「存在 / 缺席」也须保持一致（空可选字段边界）。
        expect('text' in roundTripped.watermark).toBe('text' in original.watermark);
      }),
      PBT_ASSERT_OPTIONS,
    );
  });
});
