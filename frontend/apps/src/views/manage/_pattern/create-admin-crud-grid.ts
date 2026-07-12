/** 管理端 CRUD 列表 grid 工厂（vxe proxy + 分页） */

import type { VbenFormProps } from '#/adapter/form';
import type { VxeGridProps } from '#/adapter/vxe-table';

import type {
  AdminCrudQueryValues,
  CreateAdminCrudGridOptionsInput,
} from './types';

import { fromPageResult, toPageParams } from '#/utils/pagination';

const DEFAULT_PAGE_SIZE = 20;

/** list/total → vxe proxy 的 { items, total } */
export function toVxeProxyResult<T>(resp: {
  list?: T[] | null;
  total?: unknown;
}): { items: T[]; total: number } {
  const items = Array.isArray(resp?.list) ? resp.list : [];
  const { total } = fromPageResult({ total: resp?.total });
  return { items, total };
}

/** 从 vxe proxy page 入参解析分页 */
export function resolveProxyPageParams(page: {
  currentPage?: number;
  pageSize?: number;
}): { page: number; pageSize: number } {
  return toPageParams({
    page: page?.currentPage ?? 1,
    pageSize: page?.pageSize ?? DEFAULT_PAGE_SIZE,
  });
}

/** 构建 useVbenVxeGrid 选项 */
export function createAdminCrudGridOptions<T extends Record<string, any>>(
  input: CreateAdminCrudGridOptionsInput<T>,
): VxeGridProps<T> {
  const {
    columns,
    querySchema,
    rowKey = 'id',
    tableTitle,
    pageSize = DEFAULT_PAGE_SIZE,
    query,
    gridOptions: gridOverrides,
  } = input;

  const formOptions: VbenFormProps | undefined = querySchema?.length
    ? {
        schema: querySchema,
        // 查询区紧凑布局；文案由业务 schema 的 label 提供（中文）
        commonConfig: {
          componentProps: {
            class: 'w-full',
          },
        },
        wrapperClass: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
        submitButtonOptions: {
          content: '查询',
        },
        resetButtonOptions: {
          content: '重置',
        },
      }
    : undefined;

  const {
    pagerConfig: pagerOverride,
    proxyConfig: proxyOverride,
    toolbarConfig: toolbarOverride,
    rowConfig: rowOverride,
    ...restGridOverrides
  } = gridOverrides ?? {};

  const base = {
    tableTitle,
    showSearchForm: Boolean(formOptions),
    formOptions,
    gridOptions: {
      columns,
      keepSource: true,
      ...restGridOverrides,
      // 嵌套配置后合并，避免浅 spread 冲掉本工厂注入的 pager / proxy.query / toolbar
      pagerConfig: {
        enabled: true,
        pageSize: pageSize >= 1 ? pageSize : DEFAULT_PAGE_SIZE,
        pageSizes: [10, 20, 50, 100],
        ...pagerOverride,
      },
      proxyConfig: {
        autoLoad: true,
        ...proxyOverride,
        ajax: {
          ...proxyOverride?.ajax,
          query: async (
            { page }: { page: { currentPage?: number; pageSize?: number } },
            formValues: AdminCrudQueryValues = {},
          ) => {
            const pageParams = resolveProxyPageParams(page ?? {});
            return query({
              page: pageParams.page,
              pageSize: pageParams.pageSize,
              formValues: formValues ?? {},
            });
          },
        },
      },
      rowConfig: {
        keyField: rowKey,
        ...rowOverride,
      },
      toolbarConfig: {
        // 有查询表单时展示切换搜索区按钮
        search: Boolean(formOptions),
        refresh: true,
        zoom: true,
        custom: true,
        ...toolbarOverride,
      },
    },
  } as VxeGridProps<T>;

  return base;
}
