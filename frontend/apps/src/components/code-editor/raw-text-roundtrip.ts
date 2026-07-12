/** Raw 文本 model 往返：无编辑时读出应与载入字符串相等 */
export function rawTextRoundTrip(source: string): string {
  return source;
}

export function assertRawTextRoundTrip(source: string): boolean {
  return rawTextRoundTrip(source) === source;
}
