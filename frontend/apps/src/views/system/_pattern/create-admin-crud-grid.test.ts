import { describe, expect, it, vi } from 'vitest';

import {
  createAdminCrudGridOptions,
  resolveProxyPageParams,
  toVxeProxyResult,
} from './create-admin-crud-grid';

describe('toVxeProxyResult', () => {
  it('maps list/total happy path', () => {
    expect(
      toVxeProxyResult({
        list: [{ id: 1 }, { id: 2 }],
        total: 2,
      }),
    ).toEqual({
      items: [{ id: 1 }, { id: 2 }],
      total: 2,
    });
  });

  it('treats missing/illegal total as 0 and non-array list as empty', () => {
    expect(toVxeProxyResult({ list: null, total: -1 })).toEqual({
      items: [],
      total: 0,
    });
    expect(toVxeProxyResult(undefined as never)).toEqual({
      items: [],
      total: 0,
    });
    expect(toVxeProxyResult({ list: 'x' as never, total: 1.5 })).toEqual({
      items: [],
      total: 0,
    });
  });
});

describe('resolveProxyPageParams', () => {
  it('reads currentPage/pageSize (1-based)', () => {
    expect(
      resolveProxyPageParams({ currentPage: 3, pageSize: 50 }),
    ).toEqual({ page: 3, pageSize: 50 });
  });

  it('normalizes invalid page/pageSize to 1 and defaults pageSize to 20', () => {
    expect(resolveProxyPageParams({})).toEqual({ page: 1, pageSize: 20 });
    expect(
      resolveProxyPageParams({ currentPage: 0, pageSize: -5 }),
    ).toEqual({ page: 1, pageSize: 1 });
  });
});

describe('createAdminCrudGridOptions', () => {
  it('wires query schema, pager, and proxy query via Pagination_Helper', async () => {
    const query = vi.fn(async () => ({
      items: [{ id: 1 }],
      total: 1,
    }));

    const options = createAdminCrudGridOptions({
      tableTitle: '用户管理',
      columns: [{ field: 'id', title: 'ID' }],
      querySchema: [
        {
          component: 'Input',
          fieldName: 'username',
          label: '用户名',
        },
      ],
      pageSize: 10,
      query,
    });

    expect(options.tableTitle).toBe('用户管理');
    expect(options.showSearchForm).toBe(true);
    expect(options.formOptions?.schema).toHaveLength(1);
    expect(options.formOptions?.submitButtonOptions?.content).toBe('查询');
    expect(options.gridOptions?.pagerConfig?.pageSize).toBe(10);
    expect(options.gridOptions?.toolbarConfig?.search).toBe(true);
    expect(options.gridOptions?.rowConfig?.keyField).toBe('id');

    const proxyQuery = options.gridOptions?.proxyConfig?.ajax?.query as
      | ((...args: any[]) => Promise<unknown>)
      | undefined;
    expect(typeof proxyQuery).toBe('function');

    await proxyQuery?.(
      {
        page: { currentPage: 2, pageSize: 10 },
      },
      { username: 'alice' },
    );

    expect(query).toHaveBeenCalledWith({
      page: 2,
      pageSize: 10,
      formValues: { username: 'alice' },
    });
  });

  it('keeps factory proxy.query when gridOptions.proxyConfig is overridden', async () => {
    const query = vi.fn(async () => ({ items: [], total: 0 }));
    const options = createAdminCrudGridOptions({
      columns: [{ field: 'id', title: 'ID' }],
      query,
      gridOptions: {
        proxyConfig: {
          autoLoad: false,
          ajax: {
            // 业务误传的 query 不应覆盖工厂绑定
            query: vi.fn(),
          } as never,
        },
      },
    });

    expect(options.gridOptions?.proxyConfig?.autoLoad).toBe(false);
    const proxyQuery = options.gridOptions?.proxyConfig?.ajax?.query as
      | ((...args: any[]) => Promise<unknown>)
      | undefined;
    await proxyQuery?.(
      { page: { currentPage: 1, pageSize: 20 } },
      {},
    );
    expect(query).toHaveBeenCalledTimes(1);
  });

  it('hides search form when querySchema is empty', () => {
    const options = createAdminCrudGridOptions({
      columns: [{ field: 'id', title: 'ID' }],
      query: async () => ({ items: [], total: 0 }),
    });
    expect(options.showSearchForm).toBe(false);
    expect(options.formOptions).toBeUndefined();
    expect(options.gridOptions?.toolbarConfig?.search).toBe(false);
  });
});
