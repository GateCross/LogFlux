/** 从访问日志的 extraData 中解析 WAF 审计信息；非 WAF 日志返回 null。 */
export interface WafLogInfo {
  interrupted: boolean;
  ruleId?: number;
  ruleMsg?: string;
  severity?: string;
}

export function parseWafLog(extraData?: string): WafLogInfo | null {
  if (!extraData) return null;
  let extra: Record<string, any>;
  try {
    extra = JSON.parse(extraData);
  } catch {
    return null;
  }
  if (extra?.source !== 'waf') return null;

  const first = Array.isArray(extra.matched_rules) ? extra.matched_rules[0] : undefined;
  return {
    interrupted: Boolean(extra.interrupted),
    ruleId: first?.id != null ? Number(first.id) : undefined,
    ruleMsg: first?.msg || first?.message || undefined,
    severity: extra.highest_severity || first?.severity || undefined,
  };
}
