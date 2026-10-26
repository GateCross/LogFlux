/**
 * 站点日志的参数模型与 snippet 渲染（纯函数，无 API 依赖）。
 *
 * LogFlux 只托管一份参数化 snippet：可变项（文件名 / 大小上限 / 级别）走
 * import 的位置参数传值，站点块文本仍是唯一事实来源，这里只负责把三开关
 * 翻译成 Caddyfile 文本、以及从文本里读回来，不做任何 IO。
 *
 * Caddy 事实：站点块内 log 只管访问日志，全局 log 才管运行时日志；log 的
 * level 是最低级别而非白名单，所以「开访问 + 关错误」这种组合无法表达。
 */

/** 站点日志设置：访问日志 / 错误日志 / 单文件大小上限 */
export interface SiteLogSettings {
  /** 访问日志：关掉后站点块内的 log 不再记录 access 日志 */
  access: boolean;
  /** 错误日志：由 log 的 level 控制，关掉后只留访问日志 */
  error: boolean;
  /** 单文件大小上限（MB），对应 roll_size；精度 1MB，滚动归档而非截断覆盖 */
  sizeMb: number;
}

/** 默认全开、单文件 10MB */
export const DEFAULT_SITE_LOG_SETTINGS: SiteLogSettings = {
  access: true,
  error: true,
  sizeMb: 10,
};

/** LogFlux 托管的日志 snippet 名，站点侧以 import 引用 */
export const SITE_LOG_SNIPPET_NAME = 'subdomain-log';

export const SITE_LOG_SIZE_MIN_MB = 1;
export const SITE_LOG_SIZE_MAX_MB = 1024;

/** 「开访问 + 关错误」的提示文案：该组合 Caddy 无法表达 */
export const SITE_LOG_FORCED_ERROR_MESSAGE =
  'Caddy 无法单独关闭错误日志：开启访问日志会同时记录错误日志';

/** log 指令的 level：最低级别；OFF 是前端哨兵值，绝不写进 Caddyfile */
export type SiteLogLevel = 'INFO' | 'ERROR' | 'OFF';

/** 三开关翻译成 level；OFF 表示两开关全关、整条 import 都不该写 */
export function settingsToLevel(settings: SiteLogSettings): SiteLogLevel {
  if (!settings.access && !settings.error) return 'OFF';
  if (settings.access) {
    // access=true, error=false 同样落到 INFO：level 是最低级别而非白名单，
    // 记录访问日志（INFO）必然连带记录错误日志，该组合无法表达，只能按全开处理。
    return 'INFO';
  }
  return 'ERROR';
}

/**
 * level 反向翻译成三开关；大小写不敏感，认不出的值按全开兜底。
 *
 * DEBUG / NOTICE 比 INFO 更宽，按 INFO 处理会悄悄收紧日志量；其余无法一一
 * 对应的值（WARN / FATAL / PANIC 等）同样兜底全开，宁可多记不可漏记。
 */
export function levelToSettings(level: string, sizeMb: number): SiteLogSettings {
  const size = normalizeSiteLogSizeMb(sizeMb);
  const key = (level ?? '').trim().toUpperCase();
  if (key === 'ERROR') return { access: false, error: true, sizeMb: size };
  if (key === 'OFF') return { access: false, error: false, sizeMb: size };
  return { access: true, error: true, sizeMb: size };
}

/** 归一到 1..1024 的整数 MB；非数值回落到默认 10MB */
export function normalizeSiteLogSizeMb(value: number): number {
  if (!Number.isFinite(value)) return DEFAULT_SITE_LOG_SETTINGS.sizeMb;
  // Caddy 的 roll_size 精度 1MB 且小数向上取整，这里保持同一口径
  const rounded = Math.ceil(value);
  return Math.min(
    SITE_LOG_SIZE_MAX_MB,
    Math.max(SITE_LOG_SIZE_MIN_MB, rounded),
  );
}

/**
 * 渲染 LogFlux 托管的 snippet 定义（2 空格缩进、LF 换行）。
 *
 * 骨架与具体站点无关：文件名 / 大小 / 级别是三个位置参数，按站点侧 import
 * 的出现顺序取 {args[0..2]}，一份定义被所有站点共享。入参不影响输出文本；
 * 大小上限的 clamp 在站点侧 import 行渲染时完成。
 * roll_keep 固定 1：1 个活动文件 + 1 个归档文件最省磁盘（0 会被 Caddy 当未设置）。
 */
export function renderLogSnippet(_settings: SiteLogSettings): string {
  return [
    `(${SITE_LOG_SNIPPET_NAME}) {`,
    '  log {',
    '    output file /var/log/caddy/{args[0]}.log {',
    '      roll_size {args[1]}mb',
    '      roll_keep 1',
    '    }',
    '    level {args[2]}',
    '    format json',
    '  }',
    '}',
  ].join('\n');
}

/**
 * 渲染站点侧的 import 行，返回不带缩进的行数组（缩进由写回时按站点块补齐）。
 *
 * 位置参数单行：文件名、大小上限、级别按 snippet 定义的取参顺序依次传值。
 * 两开关全关时返回空数组：level OFF 不是合法 Caddy 指令，正确做法是整条
 * import 都不写，让站点块里彻底没有 log 相关字节。
 */
export function renderSiteImportBlock(
  settings: SiteLogSettings,
  name: string,
): string[] {
  const level = settingsToLevel(settings);
  if (level === 'OFF') return [];
  const logName = (name ?? '').trim().split(/\s+/)[0] || 'site';
  return [
    `import ${SITE_LOG_SNIPPET_NAME} ${logName} ${normalizeSiteLogSizeMb(settings.sizeMb)} ${level}`,
  ];
}

/** import subdomain-log 指令的行首匹配；紧跟字符不能是名字的一部分（subdomain-logger） */
export const SITE_LOG_IMPORT_HEAD_RE = new RegExp(
  `^[ \\t]*import[ \\t]+${SITE_LOG_SNIPPET_NAME}(?![\\w-])`,
);

/**
 * 从站点块读出三开关与文件名：块里没有 import 视为两开关全关。
 *
 * 关闭态的 size 仍取默认值——没有 import 就没有可读的大小上限，UI 回填默认值
 * 保证用户重新打开日志时不必再填一遍。
 */
export function readSiteLogSettings(
  siteBlock: string,
  fallbackSizeMb: number = DEFAULT_SITE_LOG_SETTINGS.sizeMb,
): ParsedSiteLogImport {
  const parsed = parseSiteLogImport(siteBlock);
  if (parsed) return parsed;
  return {
    name: '',
    settings: {
      access: false,
      error: false,
      sizeMb: normalizeSiteLogSizeMb(fallbackSizeMb),
    },
  };
}

export interface ParsedSiteLogImport {
  /** 日志文件名；块写法缺 name 时为空串，由调用侧回落到主域名前缀 */
  name: string;
  settings: SiteLogSettings;
}

/**
 * 解析站点块里已有的 import subdomain-log。
 *
 * 兼容三种写法：位置参数单行（import subdomain-log myapp 20 ERROR）依次读
 * 文件名 / 大小 / 级别；旧的单行写法（import subdomain-log myapp）只有文件名，
 * 其余按默认值；块写法读 name / size / level 三键。找不到返回 null。
 */
export function parseSiteLogImport(
  siteBlock: string,
): null | ParsedSiteLogImport {
  const lines = (siteBlock ?? '')
    .split('\n')
    .map((line) => line.replace(/\r$/, ''));

  for (let idx = 0; idx < lines.length; idx += 1) {
    const line = lines[idx] ?? '';
    const head = line.match(SITE_LOG_IMPORT_HEAD_RE);
    if (!head) continue;

    const rest = line.slice(head[0].length).trim();
    // 块写法：参数以 name/size/level 键值对写在花括号里
    if (rest.includes('{')) {
      const args = readSiteLogImportArgs(lines, idx, rest);
      const rawSize = (args.size ?? '').trim();
      const size = rawSize === '' ? DEFAULT_SITE_LOG_SETTINGS.sizeMb : Number(rawSize);
      return {
        name: (args.name ?? '').trim(),
        settings: levelToSettings(
          args.level ?? '',
          Number.isFinite(size) ? size : DEFAULT_SITE_LOG_SETTINGS.sizeMb,
        ),
      };
    }

    const tokens = rest.split(/\s+/).filter(Boolean);
    // 位置参数单行：文件名、大小、级别依次传值
    if (tokens.length >= 3) {
      const size = Number(tokens[1]);
      return {
        name: tokens[0] ?? '',
        settings: levelToSettings(
          tokens[2] ?? '',
          Number.isFinite(size) ? size : DEFAULT_SITE_LOG_SETTINGS.sizeMb,
        ),
      };
    }
    // 旧的单行写法：只剩文件名一个参数，设置按全开兜底
    return { name: rest, settings: { ...DEFAULT_SITE_LOG_SETTINGS } };
  }
  return null;
}

/**
 * 读取块写法的键值参数：按空白切成 token 后顺序两两配对。
 *
 * Caddy 的块参数就是顺序的 key value 序列，换行不影响语义，因此先数花括号
 * 截出块内文本再统一切词，避免逐行配对在同行括号 / 块内注释时错位。
 */
function readSiteLogImportArgs(
  lines: string[],
  headIdx: number,
  headTail: string,
): Record<string, string> {
  const args: Record<string, string> = {};
  // 块参数不含嵌套花括号，从块头行的 { 起数到配对的 } 为止
  const source = [headTail, ...lines.slice(headIdx + 1)].join('\n');
  const opened = source.indexOf('{');
  if (opened < 0) return args;

  let depth = 0;
  let closed = source.length;
  for (let idx = opened; idx < source.length; idx += 1) {
    const char = source[idx];
    if (char === '{') {
      depth += 1;
    } else if (char === '}') {
      depth -= 1;
      if (depth === 0) {
        closed = idx;
        break;
      }
    }
  }

  const tokens = source
    .slice(opened + 1, closed)
    .split('\n')
    // 整行注释既不贡献参数，也不会让键值配对错位
    .filter((line) => !line.trim().startsWith('#'))
    .join(' ')
    .split(/\s+/)
    .filter(Boolean);

  for (let idx = 0; idx + 1 < tokens.length; idx += 2) {
    const key = tokens[idx] ?? '';
    if (key) args[key] = tokens[idx + 1] ?? '';
  }
  return args;
}

/** 校验三开关：大小上限必须是 1..1024 的整数，且不允许「开访问 + 关错误」 */
export function validateSiteLogSettings(settings: SiteLogSettings): string[] {
  const errors: string[] = [];
  const size = settings.sizeMb;
  if (
    !Number.isInteger(size) ||
    size < SITE_LOG_SIZE_MIN_MB ||
    size > SITE_LOG_SIZE_MAX_MB
  ) {
    errors.push(
      `单文件大小上限需为 ${SITE_LOG_SIZE_MIN_MB} 至 ${SITE_LOG_SIZE_MAX_MB} 之间的整数（MB）`,
    );
  }
  // level 是最低级别：访问日志（INFO）无法排除错误日志
  if (settings.access && !settings.error) {
    errors.push(SITE_LOG_FORCED_ERROR_MESSAGE);
  }
  return errors;
}

/** 三开关比较：只看语义，不看参数书写顺序 */
export function isSameSiteLogSettings(
  a: SiteLogSettings,
  b: SiteLogSettings,
): boolean {
  return (
    a.access === b.access &&
    a.error === b.error &&
    normalizeSiteLogSizeMb(a.sizeMb) === normalizeSiteLogSizeMb(b.sizeMb)
  );
}

/** 三开关摘要文案（只读展示生效档位，如「INFO：访问日志 + 错误日志」） */
export function describeSiteLogLevel(settings: SiteLogSettings): string {
  switch (settingsToLevel(settings)) {
    case 'ERROR': {
      return 'ERROR：仅错误日志';
    }
    case 'OFF': {
      return 'OFF：不记录日志';
    }
    default: {
      return 'INFO：访问日志 + 错误日志';
    }
  }
}
