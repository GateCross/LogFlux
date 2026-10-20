import type { DeepReadonly, Ref } from 'vue';

import type { VbenFormSchema } from '#/adapter/form';
import type { VxeTableGridColumns, VxeTableGridOptions } from '#/adapter/vxe-table';

/** 列表查询表单值（字段由业务 schema 决定） */
export type AdminCrudQueryValues = Record<string, unknown>;

/** 只读 ref 视图（composable 对外不暴露写权限） */
export type AdminCrudReadonlyRef<T> = DeepReadonly<Ref<T>>;

/** 新建/编辑表单模式 */
export type AdminCrudFormMode = 'create' | 'edit';

export interface AdminCrudFormOptions<TValues extends Record<string, unknown>> {
  /** 打开新建时的默认值工厂 */
  createDefaults: () => TValues;
  /** 打开编辑时从行数据映射到表单值 */
  mapRecordToValues: (record: unknown) => TValues;
  /** 提交：create 或 edit（edit 时 record 为打开编辑时的源对象） */
  submit: (payload: {
    mode: AdminCrudFormMode;
    values: TValues;
    record: unknown | null;
  }) => Promise<void>;
  /** 提交失败中文 fallback */
  errorFallback?: string;
}

export interface AdminCrudFormState<TValues extends Record<string, unknown>> {
  open: AdminCrudReadonlyRef<boolean>;
  mode: AdminCrudReadonlyRef<AdminCrudFormMode>;
  submitLoading: AdminCrudReadonlyRef<boolean>;
  /** 当前编辑源记录（create 时为 null） */
  editingRecord: AdminCrudReadonlyRef<unknown | null>;
  errorMessage: AdminCrudReadonlyRef<string | null>;
  openCreate: () => void;
  openEdit: (record: unknown) => void;
  close: () => void;
  /**
   * 校验通过后的提交入口。
   * 调用方通常在 useVbenForm 的 handleSubmit 里调用。
   */
  handleSubmit: (values: TValues) => Promise<boolean>;
  /** 打开弹层时应用的初始值（供 setValues） */
  initialValues: AdminCrudReadonlyRef<TValues>;
}

/** 统一列表 Grid 工厂入参 */
export interface CreateAdminCrudGridOptionsInput<T extends Record<string, any>> {
  columns: VxeTableGridColumns<T>;
  /** 查询区 schema；不传则不挂搜索表单 */
  querySchema?: VbenFormSchema[];
  /** 行主键字段，默认 id */
  rowKey?: string;
  /** 表格标题 */
  tableTitle?: string;
  /** 默认 pageSize，默认 20 */
  pageSize?: number;
  /**
   * proxy ajax.query：接收分页 + 表单值，返回 { items, total }
   * （与 adapter/vxe-table proxyConfig.response 约定一致）
   */
  query: (params: {
    page: number;
    pageSize: number;
    formValues: AdminCrudQueryValues;
  }) => Promise<{ items: T[]; total: number }>;
  /** 额外 gridOptions 覆盖 */
  gridOptions?: DeepPartialGridOptions<T>;
}

/** 避免直接依赖 DeepPartial 路径；与 vxe gridOptions 兼容的浅覆盖 */
export type DeepPartialGridOptions<T> = Partial<
  Omit<
    VxeTableGridOptions<T>,
    'columns' | 'proxyConfig' | 'pagerConfig' | 'toolbarConfig' | 'rowConfig'
  >
> & {
  proxyConfig?: Partial<NonNullable<VxeTableGridOptions<T>['proxyConfig']>> & {
    ajax?: Partial<
      NonNullable<NonNullable<VxeTableGridOptions<T>['proxyConfig']>['ajax']>
    >;
  };
  pagerConfig?: Partial<NonNullable<VxeTableGridOptions<T>['pagerConfig']>>;
  toolbarConfig?: Partial<
    NonNullable<VxeTableGridOptions<T>['toolbarConfig']>
  >;
  rowConfig?: Partial<NonNullable<VxeTableGridOptions<T>['rowConfig']>>;
};
