/**
 * 站点防护模式的文案与配色唯一来源（弹窗卡片、徽章、状态文案共用）。
 *
 * 四种模式都是站点级的，各自对应站点块里的一处定点改写：
 * - 防护模式：import waf_protect（拦截）
 * - 观察模式：import waf_protect_detect（只记录）
 * - 维护模式：写 @maintenance + respond 503（整站不可访问）
 * - 关闭：以上都不挂，请求直接放行
 *
 * 选项顺序对齐雷池：能拦 → 只看 → 维护 → 关。
 */
export type SiteProtectMode = 'detect' | 'maintenance' | 'off' | 'protect';

export type ProtectModeTone = 'danger' | 'muted' | 'success' | 'warning';

export interface SiteProtectModeOption {
  value: SiteProtectMode;
  label: string;
  /** 一句行为后果，作为弹窗卡片正文与 Tooltip */
  description: string;
  tone: ProtectModeTone;
}

export const PROTECT_MODE_OPTIONS: SiteProtectModeOption[] = [
  {
    value: 'protect',
    label: '防护模式',
    description: '发现攻击后将自动拦截，并记录攻击事件',
    tone: 'success',
  },
  {
    value: 'detect',
    label: '观察模式',
    description: '发现攻击后仅记录攻击事件，不拦截',
    tone: 'warning',
  },
  {
    value: 'maintenance',
    label: '维护模式',
    description: '展示维护页面，任何人都将无法访问您的应用',
    tone: 'danger',
  },
  {
    value: 'off',
    label: '关闭',
    description: '不检测也不拦截，站点请求直接放行',
    tone: 'muted',
  },
];

const FALLBACK_OPTION: SiteProtectModeOption = {
  value: 'off',
  label: '未防护',
  description: '当前站点未挂载任何 WAF 模式',
  tone: 'muted',
};

/** 按值取选项；未知值返回中性兜底，不做模糊匹配 */
export function protectModeOption(mode?: null | string): SiteProtectModeOption {
  return PROTECT_MODE_OPTIONS.find((item) => item.value === mode) ?? FALLBACK_OPTION;
}

/** 模式的中文名（状态标签、只读展示用） */
export function protectModeLabel(mode?: null | string): string {
  return protectModeOption(mode).label;
}
