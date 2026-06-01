/**
 * Preferences_Store —— 用户偏好序列化 / 反序列化 / 校验与默认偏好（任务 3.1）。
 *
 * 本模块为「框架无关的纯逻辑」，不依赖 React / Umi / localStorage（本地存储封装见任务 3.4
 * 的 `src/utils/storage.ts`，偏好远端同步事务见 `src/models/app.ts`）。
 *
 * 设计依据：
 *  - design.md「Components and Interfaces ▸ Preferences_Store」与「Data Models ▸ UserPreferences」。
 *  - 字段集合迁移自旧 Vue 版 `App.Theme.ThemeSetting`（`frontend/src/theme/settings.ts`），
 *    保持结构 `{ theme, lang, layout }`，并补充自由文本水印字段以覆盖
 *    Req 15.3 列举的「空可选字段 / 最长允许字符串 / 含特殊 Unicode 字符」等边界情形。
 *
 * 支撑的需求 / 属性：
 *  - Req 15.1 序列化为字符串；Req 15.2 反序列化为对象。
 *  - Property 15（往返一致）：对任意「合法偏好对象」先 `serializePreferences` 再
 *    `deserializePreferences` 应在全部字段（含嵌套）上深度相等。本模块通过将偏好限制为
 *    「仅含 JSON 安全的枚举 / 布尔 / 字符串值」来保证 JSON 往返天然深度相等。
 *  - Req 15.4 / 15.5 与 Property 16（回退默认）：`loadPreferences` 对语法非法字符串或
 *    可解析但不合法（缺必需字段 / 取值越界）的对象，回退默认偏好并返回 `usedDefault=true`，
 *    且不抛出异常。
 */

/** 主题明暗方案（迁移自 `UnionKey.ThemeScheme`）。 */
export type ThemeScheme = 'light' | 'dark' | 'auto';

/** 界面语言（Req 5）。 */
export type LangType = 'zh-CN' | 'en-US';

/** 布局模式（迁移自 `UnionKey.ThemeLayoutMode`）。 */
export type LayoutMode =
  | 'vertical'
  | 'horizontal'
  | 'vertical-mix'
  | 'vertical-hybrid-header-first'
  | 'top-hybrid-sidebar-first'
  | 'top-hybrid-header-first';

/**
 * 用户偏好模型（Preferences_Store 的核心数据结构，design.md「Data Models」）。
 *
 * 所有字段取值均为 JSON 安全类型（枚举字符串 / 布尔 / 受限字符串），不含 `undefined`
 * 之外的不可序列化值，从而保证 `serialize → deserialize` 的深度相等往返（Property 15）。
 */
export interface UserPreferences {
  /** 主题偏好 */
  theme: {
    /** 明暗方案 */
    themeScheme: ThemeScheme;
    /** 主题主色（必须为合法颜色值，见 `isValidThemeColor`，Req 6.3 / 15.5） */
    themeColor: string;
    /** 灰度模式 */
    grayscale: boolean;
    /** 色弱模式 */
    colourWeakness: boolean;
  };
  /** 界面语言 */
  lang: LangType;
  /** 布局偏好 */
  layout: {
    /** 布局模式 */
    mode: LayoutMode;
    /** 侧边栏是否收起 */
    siderCollapse: boolean;
  };
  /** 水印偏好（自由文本，覆盖空 / 最长 / Unicode 边界情形） */
  watermark: {
    /** 是否显示水印 */
    visible: boolean;
    /**
     * 水印文案（可选）。允许空字符串与含特殊 / Unicode 字符的文本；
     * 长度上限为 {@link MAX_WATERMARK_TEXT_LENGTH}。缺省（未提供）视为合法。
     */
    text?: string;
  };
}

/** 允许的明暗方案集合。 */
export const THEME_SCHEMES: readonly ThemeScheme[] = ['light', 'dark', 'auto'];

/** 允许的语言集合（Req 5.1）。 */
export const LANGS: readonly LangType[] = ['zh-CN', 'en-US'];

/** 允许的布局模式集合。 */
export const LAYOUT_MODES: readonly LayoutMode[] = [
  'vertical',
  'horizontal',
  'vertical-mix',
  'vertical-hybrid-header-first',
  'top-hybrid-sidebar-first',
  'top-hybrid-header-first',
];

/** 水印文案最大长度（用于「最长允许字符串」边界校验）。 */
export const MAX_WATERMARK_TEXT_LENGTH = 64;

/**
 * 默认用户偏好（Req 6.7 / 15.4 / 15.5 的回退目标）。
 *
 * 取值迁移自旧 Vue 版默认主题设置（`frontend/src/theme/settings.ts`）与默认语言（zh-CN，Req 5.6）。
 * 以 `createDefaultPreferences()` 生成全新对象，避免调用方意外修改共享引用。
 */
export function createDefaultPreferences(): UserPreferences {
  return {
    theme: {
      themeScheme: 'light',
      themeColor: '#646cff',
      grayscale: false,
      colourWeakness: false,
    },
    lang: 'zh-CN',
    layout: {
      mode: 'vertical',
      siderCollapse: false,
    },
    watermark: {
      visible: false,
      text: 'LogFlux',
    },
  };
}

/**
 * 默认用户偏好（只读快照）。
 *
 * 需要可变副本时请使用 {@link createDefaultPreferences}；`loadPreferences` 在回退时
 * 返回的即是全新副本，调用方可安全修改。
 */
export const DEFAULT_PREFERENCES: Readonly<UserPreferences> = Object.freeze(createDefaultPreferences());

/** 判断是否为「纯对象」（非 null、非数组）。 */
function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** 判断是否为合法明暗方案。 */
function isThemeScheme(value: unknown): value is ThemeScheme {
  return typeof value === 'string' && (THEME_SCHEMES as readonly string[]).includes(value);
}

/** 判断是否为合法语言。 */
function isLang(value: unknown): value is LangType {
  return typeof value === 'string' && (LANGS as readonly string[]).includes(value);
}

/** 判断是否为合法布局模式。 */
function isLayoutMode(value: unknown): value is LayoutMode {
  return typeof value === 'string' && (LAYOUT_MODES as readonly string[]).includes(value);
}

/** 十六进制颜色：`#rgb` / `#rgba` / `#rrggbb` / `#rrggbbaa`。 */
const HEX_COLOR_RE = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/;

/** 判断是否为 0~255 的整数通道值。 */
function isByteChannel(token: string): boolean {
  if (!/^\d{1,3}$/.test(token)) {
    return false;
  }
  const n = Number(token);
  return Number.isInteger(n) && n >= 0 && n <= 255;
}

/** 判断是否为 0~1 的 alpha 值（允许整数或小数）。 */
function isAlphaChannel(token: string): boolean {
  if (!/^(0|1|0?\.\d+)$/.test(token)) {
    return false;
  }
  const n = Number(token);
  return Number.isFinite(n) && n >= 0 && n <= 1;
}

/** 判断是否为合法 `rgb()` / `rgba()` 颜色。 */
function isValidRgbColor(value: string): boolean {
  const match = /^rgba?\(([^)]*)\)$/.exec(value);
  if (!match) {
    return false;
  }
  const parts = match[1].split(',').map(part => part.trim());
  if (parts.some(part => part.length === 0)) {
    return false;
  }
  // rgb(r,g,b) 接受 3 段；rgba(r,g,b,a) 接受 4 段。
  if (parts.length !== 3 && parts.length !== 4) {
    return false;
  }
  const [r, g, b, a] = parts;
  if (!isByteChannel(r) || !isByteChannel(g) || !isByteChannel(b)) {
    return false;
  }
  if (parts.length === 4 && !isAlphaChannel(a)) {
    return false;
  }
  return true;
}

/**
 * 校验主题主色是否合法（Req 6.3 / 15.5）。
 *
 * 接受十六进制（`#rgb`/`#rgba`/`#rrggbb`/`#rrggbbaa`）与 `rgb()`/`rgba()` 两种格式；
 * 其余字符串（含空串、命名颜色、Unicode 等）一律判定为非法。
 */
export function isValidThemeColor(value: unknown): value is string {
  if (typeof value !== 'string') {
    return false;
  }
  if (HEX_COLOR_RE.test(value)) {
    return true;
  }
  return isValidRgbColor(value);
}

/**
 * 显式 schema 校验：判断任意值是否为「合法用户偏好对象」。
 *
 * 校验项（Req 15.5）：
 *  - 必需字段齐全：`theme.{themeScheme,themeColor,grayscale,colourWeakness}`、`lang`、
 *    `layout.{mode,siderCollapse}`、`watermark.visible`。
 *  - 取值落在允许范围 / 枚举：明暗方案、语言、布局模式枚举；主色为合法颜色；
 *    布尔字段为布尔；可选水印文案为长度 ≤ 上限的字符串（缺省合法）。
 */
export function isValidPreferences(value: unknown): value is UserPreferences {
  if (!isPlainObject(value)) {
    return false;
  }

  const { theme, lang, layout, watermark } = value;

  // theme
  if (!isPlainObject(theme)) {
    return false;
  }
  if (!isThemeScheme(theme.themeScheme)) {
    return false;
  }
  if (!isValidThemeColor(theme.themeColor)) {
    return false;
  }
  if (typeof theme.grayscale !== 'boolean') {
    return false;
  }
  if (typeof theme.colourWeakness !== 'boolean') {
    return false;
  }

  // lang
  if (!isLang(lang)) {
    return false;
  }

  // layout
  if (!isPlainObject(layout)) {
    return false;
  }
  if (!isLayoutMode(layout.mode)) {
    return false;
  }
  if (typeof layout.siderCollapse !== 'boolean') {
    return false;
  }

  // watermark
  if (!isPlainObject(watermark)) {
    return false;
  }
  if (typeof watermark.visible !== 'boolean') {
    return false;
  }
  if (watermark.text !== undefined) {
    if (typeof watermark.text !== 'string' || watermark.text.length > MAX_WATERMARK_TEXT_LENGTH) {
      return false;
    }
  }

  return true;
}

/**
 * 从已校验的合法偏好中重建「规范化」副本。
 *
 * 仅拷贝已知字段，剔除多余键，确保 `loadPreferences` 返回的对象结构确定、与模型一致，
 * 且为调用方可安全修改的全新对象。可选的 `watermark.text` 仅在存在时保留。
 */
function normalizeValidPreferences(value: UserPreferences): UserPreferences {
  const result: UserPreferences = {
    theme: {
      themeScheme: value.theme.themeScheme,
      themeColor: value.theme.themeColor,
      grayscale: value.theme.grayscale,
      colourWeakness: value.theme.colourWeakness,
    },
    lang: value.lang,
    layout: {
      mode: value.layout.mode,
      siderCollapse: value.layout.siderCollapse,
    },
    watermark: {
      visible: value.watermark.visible,
    },
  };

  if (value.watermark.text !== undefined) {
    result.watermark.text = value.watermark.text;
  }

  return result;
}

/**
 * 将偏好对象序列化为字符串以写入本地存储（Req 15.1）。
 */
export function serializePreferences(preferences: UserPreferences): string {
  return JSON.stringify(preferences);
}

/**
 * 将偏好字符串反序列化为偏好对象（Req 15.2，输入合法时）。
 *
 * 本函数面向「已知合法」的输入（如 {@link serializePreferences} 的输出），不做 schema 校验，
 * 字符串语法非法时会抛出 `SyntaxError`。处理不可信输入请使用 {@link loadPreferences}。
 */
export function deserializePreferences(raw: string): UserPreferences {
  return JSON.parse(raw) as UserPreferences;
}

/** `loadPreferences` 的返回结构：偏好对象及是否回退到默认偏好的指示。 */
export interface LoadPreferencesResult {
  /** 最终生效的偏好对象（合法时为持久化值的规范化副本，否则为默认偏好副本） */
  preferences: UserPreferences;
  /** 是否因输入缺失 / 非法而使用了默认偏好 */
  usedDefault: boolean;
}

/**
 * 读取并校验持久化偏好字符串，必要时回退默认偏好（Req 15.4 / 15.5 / 6.7，Property 16）。
 *
 * 回退（返回 `usedDefault=true`）的情形，且任何情形均不抛出异常：
 *  - 输入为 `null` / `undefined` / 空串（无持久化偏好）。
 *  - 输入字符串语法非法、无法被 `JSON.parse`（Req 15.4）。
 *  - 输入可被反序列化但不是合法偏好对象（缺必需字段 / 取值越界，Req 15.5）。
 *
 * 合法时返回持久化偏好的规范化副本，`usedDefault=false`。
 */
export function loadPreferences(raw: string | null | undefined): LoadPreferencesResult {
  if (raw == null || raw === '') {
    return { preferences: createDefaultPreferences(), usedDefault: true };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    // Req 15.4：语法非法字符串 → 回退默认，不中断初始化。
    return { preferences: createDefaultPreferences(), usedDefault: true };
  }

  if (!isValidPreferences(parsed)) {
    // Req 15.5：可解析但非合法偏好对象 → 回退默认。
    return { preferences: createDefaultPreferences(), usedDefault: true };
  }

  return { preferences: normalizeValidPreferences(parsed), usedDefault: false };
}
