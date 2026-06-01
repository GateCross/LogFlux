/* eslint-disable */
/**
 * 一次性迁移脚本：将旧 Vue 版嵌套 langs（frontend/src/locales/langs/*.ts）
 * 扁平化为 react-intl / Umi locale 所需的「点分扁平键」文案，并生成
 * frontend-next/src/locales/zh-CN.ts 与 en-US.ts。
 *
 * - 扁平化分隔符使用 `.`，保持与旧 $t('common.add') 调用一致的键字符串。
 * - 以 zh-CN 的键顺序为基准对齐 en-US，保证两套文案键集合一致（Property 9）。
 * - 对含字面量花括号的值（JSON 示例占位提示）按 ICU 规则转义，避免 react-intl 解析报错，
 *   同时保持渲染文本与旧版完全一致。
 *
 * 用法：node scripts/gen-locales.cjs
 * 生成完成后该脚本可删除（已纳入任务 5.1 的一次性迁移工序）。
 */
const fs = require('fs');
const path = require('path');

const OLD_DIR = path.resolve(__dirname, '../../frontend/src/locales/langs');
const OUT_DIR = path.resolve(__dirname, '../src/locales');

function loadLang(file) {
  let content = fs.readFileSync(file, 'utf8');
  content = content.replace(/const local\s*:\s*App\.I18n\.Schema\s*=/, 'module.exports =');
  content = content.replace(/export default local;?\s*$/, '');
  const tmp = file + '.gen.cjs';
  fs.writeFileSync(tmp, content);
  delete require.cache[require.resolve(tmp)];
  const mod = require(tmp);
  fs.unlinkSync(tmp);
  return mod;
}

function flatten(obj, prefix, out) {
  out = out || {};
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      flatten(v, key, out);
    } else {
      out[key] = v;
    }
  }
  return out;
}

/**
 * ICU 字面量花括号转义：仅对包含「花括号 + 空白」（即 JSON 示例，而非 {name} 占位符）的值，
 * 将所有 `{`/`}` 转义为 `'{'`/`'}'`，使 react-intl 输出与原文一致且不抛解析错误。
 */
function escapeBraces(val) {
  if (typeof val === 'string' && /\{\s/.test(val)) {
    return val.replace(/\{/g, "'{'").replace(/\}/g, "'}'");
  }
  return val;
}

function emitBody(flat, order) {
  return order
    .map(k => {
      const val = escapeBraces(flat[k]);
      return `  ${JSON.stringify(k)}: ${JSON.stringify(val)},`;
    })
    .join('\n');
}

function emitZh(flat, order, header) {
  return (
    `${header}` +
    `/** zh-CN 原始文案（基准语言 / 回退目标，Req 5.1 / 5.7）。 */\n` +
    `const messages = {\n${emitBody(flat, order)}\n};\n\n` +
    `export default messages;\n`
  );
}

function emitEn(flat, order, header) {
  return (
    `${header}` +
    `import zhCN from './zh-CN';\n\n` +
    `/** en-US 原始文案（未合并回退基准；键集合一致性测试以此为准，Property 9）。 */\n` +
    `export const messages = {\n${emitBody(flat, order)}\n};\n\n` +
    `/**\n` +
    ` * 默认导出：以 zh-CN 作为回退基准合并（fallbackLocale=zh-CN，Req 5.7）。\n` +
    ` * en-US 缺失的键将回退渲染 zh-CN 对应文案，而非渲染键名或空串。\n` +
    ` */\n` +
    `export default { ...zhCN, ...messages };\n`
  );
}

const zh = flatten(loadLang(path.join(OLD_DIR, 'zh-cn.ts')));
const en = flatten(loadLang(path.join(OLD_DIR, 'en-us.ts')));

const zhKeys = Object.keys(zh);
const enKeys = Object.keys(en);
const zhSet = new Set(zhKeys);
const enSet = new Set(enKeys);

const onlyZh = zhKeys.filter(k => !enSet.has(k));
const onlyEn = enKeys.filter(k => !zhSet.has(k));

if (onlyZh.length || onlyEn.length) {
  console.error('KEY MISMATCH detected:');
  if (onlyZh.length) console.error('  only in zh-CN:', onlyZh);
  if (onlyEn.length) console.error('  only in en-US:', onlyEn);
} else {
  console.log(`OK: both locales cover ${zhKeys.length} keys.`);
}

const zhHeader = `/**\n * 简体中文文案（zh-CN）。\n *\n * 由旧 Vue 版 \`frontend/src/locales/langs/zh-cn.ts\` 迁移并扁平化为 react-intl 所需的\n * 「点分扁平键」结构（Req 5.1）。键字符串与旧 \`$t('common.add')\` 调用保持一致。\n *\n * 维护约定：\n *  - zh-CN 为基准语言（缺键回退目标，Req 5.7），新增键须同时补充 en-US（Property 9）。\n *  - 含字面量花括号的 JSON 示例值按 ICU 规则以 \`'{'\`/\`'}'\` 转义，渲染结果与原文一致。\n */\n`;
const enHeader = `/**\n * English locale (en-US).\n *\n * Migrated from the legacy Vue \`frontend/src/locales/langs/en-us.ts\` and flattened into the\n * dot-notation keys required by react-intl (Req 5.1). Key strings match the legacy\n * \`$t('common.add')\` call sites.\n *\n * Conventions:\n *  - The named \`messages\` export is the RAW en-US set, kept in the SAME key order as zh-CN so both\n *    locales share an identical key set (Property 9 asserts against the raw sets).\n *  - The default export merges zh-CN as the fallback base so en-US-missing keys fall back to zh-CN\n *    (fallbackLocale=zh-CN, Req 5.7).\n *  - JSON-example values containing literal braces are ICU-escaped as \`'{'\`/\`'}'\`.\n */\n`;

fs.writeFileSync(path.join(OUT_DIR, 'zh-CN.ts'), emitZh(zh, zhKeys, zhHeader), 'utf8');
// en-US 以 zh-CN 的键顺序输出，保证两文件键顺序一致、便于对照维护。
fs.writeFileSync(path.join(OUT_DIR, 'en-US.ts'), emitEn(en, zhKeys, enHeader), 'utf8');

console.log('Generated:');
console.log('  ', path.join(OUT_DIR, 'zh-CN.ts'));
console.log('  ', path.join(OUT_DIR, 'en-US.ts'));
