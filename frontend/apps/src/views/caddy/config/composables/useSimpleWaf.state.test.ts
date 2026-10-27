import { nextTick, ref } from 'vue';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import { createDefaultWafForm, useSimpleWaf, WAF_REQUEST_BODY_MAX_MB } from './useSimpleWaf';

const fetchQuery = vi.fn();

vi.mock('@tanstack/vue-query', () => ({
  useQueryClient: () => ({
    fetchQuery,
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock('#/api/caddy/simple-waf', () => ({
  applySimpleWafConfigApi: vi.fn(),
  getSimpleWafConfigApi: vi.fn(),
  previewSimpleWafConfigApi: vi.fn(),
  updateSimpleWafConfigApi: vi.fn(),
}));

vi.mock('#/api/list-detail', () => ({
  withListDetailErrorMode: () => ({}),
}));

vi.mock('#/api/list-detail-mutation', () => ({
  invalidateListDetailQueries: vi.fn(),
  invalidateListDetailQueryKeys: vi.fn(),
}));

vi.mock('#/api/query-keys', () => ({
  qk: { caddy: { config: () => ['caddy', 'config'] } },
}));

vi.mock('#/api/request', () => ({
  suppressGlobalErrorToast: {},
  requestClient: {},
}));

vi.mock('#/utils/api-error-message', () => ({
  apiErrorMessage: (_error: unknown, fallback: string) => fallback,
}));

vi.mock('antdv-next', () => ({
  message: { error: vi.fn(), success: vi.fn() },
}));

function serverConfig(suffix = '1', overrides: Record<string, unknown> = {}) {
  return {
    audit: 'relevantonly',
    availableSites: [`s${suffix}.example.com`],
    importedSites: [`s${suffix}.example.com`],
    corazaVersion: '3.2.1',
    crsVersion: '4.0.0',
    enabled: true,
    integrated: true,
    mode: 'on',
    requestBodyAccess: true,
    requestBodyLimit: 10 * 1024 * 1024,
    requestBodyNoFilesLimit: 1024 * 1024,
    siteAddresses: [`s${suffix}.example.com`],
    strength: 'balanced',
    ...overrides,
  };
}

function createWaf() {
  return useSimpleWaf({});
}

describe('useSimpleWaf state machine', () => {
  beforeEach(() => {
    fetchQuery.mockReset();
  });

  it('ready: 成功读取后允许写操作', async () => {
    fetchQuery.mockResolvedValue(serverConfig('1'));
    const waf = createWaf();

    await waf.fetchWafConfig();

    expect(waf.wafLoadState.value).toBe('ready');
    expect(waf.wafLoaded.value).toBe(true);
    expect(waf.wafActionsDisabled.value).toBe(false);
    expect(waf.wafForm.enabled).toBe(true);
  });

  it('error: 读失败清空表单并禁用写操作，避免用默认值覆盖线上策略', async () => {
    fetchQuery.mockResolvedValueOnce(serverConfig('1'));
    const waf = createWaf();
    await waf.fetchWafConfig();
    expect(waf.wafForm.enabled).toBe(true);

    fetchQuery.mockRejectedValueOnce(new Error('boom'));
    await waf.fetchWafConfig();

    expect(waf.wafLoadState.value).toBe('error');
    expect(waf.wafStatus.value).toBeNull();
    expect(waf.wafLoaded.value).toBe(false);
    expect(waf.wafForm).toEqual(createDefaultWafForm());
    expect(waf.wafActionsDisabled.value).toBe(true);
  });

  it('stale: 重置后旧状态不再可用', async () => {
    fetchQuery.mockResolvedValue(serverConfig('1'));
    const waf = createWaf();
    await waf.fetchWafConfig();

    waf.resetWafState();
    await nextTick();

    expect(waf.isWafStale()).toBe(true);
    expect(waf.wafActionsDisabled.value).toBe(true);
  });

  it('reset: 作废在途请求，旧响应不会覆盖新状态', async () => {
    let resolveFirst: (value: unknown) => void = () => {};
    fetchQuery.mockImplementationOnce(
      () => new Promise((resolve) => {
        resolveFirst = resolve;
      }),
    );
    const waf = createWaf();

    const pending = waf.fetchWafConfig();
    waf.resetWafState();
    resolveFirst(serverConfig('1'));
    await pending;

    expect(waf.wafStatus.value).toBeNull();
    expect(waf.wafLoaded.value).toBe(false);
    expect(waf.wafLoadState.value).toBe('idle');
  });

  it('sync: 站点为空时保持空选择（表示跟随全部），不固化当前站点快照', async () => {
    fetchQuery.mockResolvedValue(serverConfig('1', { siteAddresses: [] }));
    const waf = createWaf();

    await waf.fetchWafConfig();

    expect(waf.wafForm.siteAddresses).toEqual([]);
    expect(waf.wafAvailableSites.value).toEqual(['s1.example.com']);
  });

  it('dirty: 表单改动后置脏，改回原值后恢复干净', async () => {
    fetchQuery.mockResolvedValue(serverConfig('1'));
    const waf = createWaf();
    await waf.fetchWafConfig();
    expect(waf.wafDirty.value).toBe(false);

    waf.wafForm.strength = 'high_blocking';
    expect(waf.wafDirty.value).toBe(true);

    waf.wafForm.strength = 'balanced';
    expect(waf.wafDirty.value).toBe(false);
  });

  it('dirty: 站点集合顺序变化不算改动', async () => {
    fetchQuery.mockResolvedValue(
      serverConfig('1', {
        availableSites: ['a.example.com', 'b.example.com'],
        siteAddresses: ['a.example.com', 'b.example.com'],
      }),
    );
    const waf = createWaf();
    await waf.fetchWafConfig();

    waf.wafForm.siteAddresses = ['b.example.com', 'a.example.com'];

    expect(waf.wafDirty.value).toBe(false);
  });

  it('dirty: 未加载成功时永远不算脏，避免误弹确认', () => {
    const waf = createWaf();
    waf.wafForm.strength = 'high_blocking';

    expect(waf.wafDirty.value).toBe(false);
  });

  it('effective: 站点选择为空时回退到全部可用站点', async () => {
    fetchQuery.mockResolvedValue(
      serverConfig('1', {
        availableSites: ['a.example.com', 'b.example.com'],
        importedSites: ['a.example.com', 'b.example.com'],
        siteAddresses: [],
      }),
    );
    const waf = createWaf();
    await waf.fetchWafConfig();

    expect(waf.wafSiteStates.value.map((item) => item.site)).toEqual([
      'a.example.com',
      'b.example.com',
    ]);
    expect(waf.wafSiteStates.value.every((item) => item.state === 'active')).toBe(true);

    waf.wafForm.siteAddresses = ['a.example.com'];
    expect(waf.wafSiteStates.value).toEqual([
      { site: 'a.example.com', state: 'active' },
      { site: 'b.example.com', state: 'pending' },
    ]);
  });

  it('site states: 关闭 WAF 时已挂载站点标为将移除', async () => {
    fetchQuery.mockResolvedValue(
      serverConfig('1', {
        availableSites: ['a.example.com', 'b.example.com'],
        importedSites: ['a.example.com'],
        siteAddresses: ['a.example.com'],
      }),
    );
    const waf = createWaf();
    await waf.fetchWafConfig();

    waf.wafForm.enabled = false;

    expect(waf.wafSiteStates.value).toEqual([
      { site: 'a.example.com', state: 'remove' },
      { site: 'b.example.com', state: 'pending' },
    ]);
  });

  describe('validateWafForm', () => {
    it('happy path: 合法区间返回 null', async () => {
      fetchQuery.mockResolvedValue(serverConfig('1'));
      const waf = createWaf();
      await waf.fetchWafConfig();

      expect(waf.validateWafForm()).toBeNull();
    });

    it('boundary: 超过后端 1GB 上限被拒绝', async () => {
      fetchQuery.mockResolvedValue(serverConfig('1'));
      const waf = createWaf();
      await waf.fetchWafConfig();

      waf.wafForm.requestBodyLimitMB = WAF_REQUEST_BODY_MAX_MB + 1;
      expect(waf.validateWafForm()).toContain(`${WAF_REQUEST_BODY_MAX_MB}`);
    });

    it('boundary: 无文件上限大于总上限被拒绝', async () => {
      fetchQuery.mockResolvedValue(serverConfig('1'));
      const waf = createWaf();
      await waf.fetchWafConfig();

      waf.wafForm.requestBodyLimitMB = 5;
      waf.wafForm.requestBodyNoFilesLimitMB = 10;
      expect(waf.validateWafForm()).toContain('不能大于');
    });
  });

  it('apply payload: 携带 baseHash 用于乐观并发', async () => {
    fetchQuery.mockResolvedValue(serverConfig('1'));
    const baseHash = ref('abc123');
    const waf = useSimpleWaf({ baseHash });
    await waf.fetchWafConfig();

    expect(waf.buildWafPayload().baseHash).toBe('abc123');
  });
});
