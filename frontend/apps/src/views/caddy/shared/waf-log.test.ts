import { describe, expect, it } from 'vitest';

import { parseWafLog } from './waf-log';

describe('parseWafLog', () => {
  it('非 WAF 日志返回 null', () => {
    expect(parseWafLog(undefined)).toBeNull();
    expect(parseWafLog('')).toBeNull();
    expect(parseWafLog('{}')).toBeNull();
    expect(parseWafLog('{"source":"access"}')).toBeNull();
    expect(parseWafLog('not json')).toBeNull();
  });

  it('解析拦截规则与等级', () => {
    const extra = JSON.stringify({
      source: 'waf',
      interrupted: true,
      highest_severity: 'CRITICAL',
      matched_rules: [{ id: 942100, msg: 'SQL Injection Attack Detected' }],
    });
    expect(parseWafLog(extra)).toEqual({
      interrupted: true,
      ruleId: 942100,
      ruleMsg: 'SQL Injection Attack Detected',
      severity: 'CRITICAL',
    });
  });

  it('无命中规则时仅返回处置状态', () => {
    expect(parseWafLog('{"source":"waf","interrupted":false}')).toEqual({
      interrupted: false,
      ruleId: undefined,
      ruleMsg: undefined,
      severity: undefined,
    });
  });
});
