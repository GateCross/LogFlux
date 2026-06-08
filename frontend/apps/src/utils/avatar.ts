/**
 * 生成默认头像的 SVG data URL
 * 当用户未设置头像时使用
 */

const AVATAR_COLORS: Array<[string, string]> = [
  ['#1677ff', '#e6f4ff'],
  ['#13a8a8', '#e6fffb'],
  ['#722ed1', '#f9f0ff'],
  ['#d4380d', '#fff2e8'],
];

/**
 * 根据用户名生成一致的默认头像
 * 同一用户名始终得到相同颜色和文字
 */
export function getDefaultAvatarDataUrl(username?: string): string {
  const text = (username || 'LF').slice(0, 2).toUpperCase();
  // 用用户名的 charCode 之和选择颜色，保证同一用户始终同一颜色
  const hash = [...text].reduce((acc, ch) => acc + ch.charCodeAt(0), 0);
  const [primary, secondary] = AVATAR_COLORS[hash % AVATAR_COLORS.length]!;

  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="160" height="160" viewBox="0 0 160 160"><rect width="160" height="160" rx="32" fill="${secondary}"/><circle cx="80" cy="64" r="34" fill="${primary}"/><path d="M32 142c8-30 25-46 48-46s40 16 48 46" fill="${primary}" opacity=".9"/><text x="80" y="74" text-anchor="middle" font-family="Arial, sans-serif" font-size="28" font-weight="700" fill="white">${text}</text></svg>`;
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`;
}

/**
 * 获取用户头像（带默认值回退）
 * 优先级：用户选择的头像 > 服务器头像 > 默认生成头像
 */
export function resolveAvatar(
  userAvatar: string | undefined | null,
  username?: string,
): string {
  return userAvatar || getDefaultAvatarDataUrl(username);
}
