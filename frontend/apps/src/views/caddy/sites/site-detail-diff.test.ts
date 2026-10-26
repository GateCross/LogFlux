import { describe, expect, it } from 'vitest';

import { buildLineDiff } from '../config/caddy-config-diff';

/**
 * 站点详情保存前的变更预览语义。
 * 这里锁定的是「弹窗展示什么」：只展示真正变动的行，
 * 而不是把整个新块当成全新增。
 */
describe('站点块变更预览', () => {
  const previous = ['a.example.com {', '\treverse_proxy localhost:8080', '}'].join(
    '\n',
  );

  it('改动上游只产生一行 修改，其余行保持 same', () => {
    const next = ['a.example.com {', '\treverse_proxy localhost:9090', '}'].join('\n');
    const rows = buildLineDiff(previous, next);

    const changed = rows.filter((row) => row.type !== 'same');
    expect(changed).toHaveLength(1);
    expect(changed[0]).toMatchObject({
      type: 'changed',
      left: '\treverse_proxy localhost:8080',
      right: '\treverse_proxy localhost:9090',
    });
  });

  it('删掉一行（如关闭 WAF 挂载）必须出现 removed 行', () => {
    const previousWithWaf = [
      'a.example.com {',
      '\timport waf_protect',
      '\treverse_proxy localhost:8080',
      '}',
    ].join('\n');
    const rows = buildLineDiff(previousWithWaf, previous);

    const removed = rows.filter((row) => row.type === 'removed');
    expect(removed).toHaveLength(1);
    expect(removed[0]?.left).toBe('\timport waf_protect');
  });

  it('新增一行（如打开压缩）必须出现 added 行', () => {
    const next = [
      'a.example.com {',
      '\timport compression',
      '\treverse_proxy localhost:8080',
      '}',
    ].join('\n');
    const rows = buildLineDiff(previous, next);

    const added = rows.filter((row) => row.type === 'added');
    expect(added).toHaveLength(1);
    expect(added[0]?.right).toBe('\timport compression');
  });

  it('无改动时没有任何非 same 行', () => {
    const rows = buildLineDiff(previous, previous);
    expect(rows.filter((row) => row.type !== 'same')).toHaveLength(0);
  });
});
