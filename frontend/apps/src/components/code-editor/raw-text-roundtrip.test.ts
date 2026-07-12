import * as fc from 'fast-check';
import { describe, expect, it } from 'vitest';

import { assertRawTextRoundTrip, rawTextRoundTrip } from './raw-text-roundtrip';

describe('Raw text round-trip', () => {
  it('无编辑时读出与载入字符串相等', () => {
    fc.assert(
      fc.property(fc.string({ maxLength: 2000 }), (s) => {
        expect(rawTextRoundTrip(s)).toBe(s);
        expect(assertRawTextRoundTrip(s)).toBe(true);
      }),
      { numRuns: 100 },
    );
  });

  it('合法输入：含换行与空白的 Caddyfile 片段保持原样', () => {
    const sample = 'example.com {\n  reverse_proxy localhost:8080\n}\n';
    expect(rawTextRoundTrip(sample)).toBe(sample);
  });

  it('边界：空串与仅空白', () => {
    expect(rawTextRoundTrip('')).toBe('');
    expect(rawTextRoundTrip(' \t\n')).toBe(' \t\n');
  });
});
