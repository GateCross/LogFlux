import type {
  AdminCrudFormMode,
  AdminCrudFormOptions,
  AdminCrudFormState,
} from './types';

import { readonly, ref } from 'vue';

import { apiErrorMessage } from '#/utils/api-error-message';

const DEFAULT_ERROR_FALLBACK = '提交失败';

export function useAdminCrudForm<TValues extends Record<string, unknown>>(
  options: AdminCrudFormOptions<TValues>,
): AdminCrudFormState<TValues> {
  const {
    createDefaults,
    mapRecordToValues,
    submit,
    errorFallback = DEFAULT_ERROR_FALLBACK,
  } = options;

  const open = ref(false);
  const mode = ref<AdminCrudFormMode>('create');
  const submitLoading = ref(false);
  const editingRecord = ref<unknown | null>(null);
  const errorMessage = ref<string | null>(null);
  const initialValues = ref(createDefaults()) as { value: TValues };

  function openCreate() {
    mode.value = 'create';
    editingRecord.value = null;
    errorMessage.value = null;
    initialValues.value = createDefaults();
    open.value = true;
  }

  function openEdit(record: unknown) {
    mode.value = 'edit';
    editingRecord.value = record;
    errorMessage.value = null;
    initialValues.value = mapRecordToValues(record);
    open.value = true;
  }

  function close() {
    open.value = false;
    submitLoading.value = false;
    errorMessage.value = null;
  }

  async function handleSubmit(values: TValues): Promise<boolean> {
    submitLoading.value = true;
    errorMessage.value = null;
    try {
      await submit({
        mode: mode.value,
        values,
        record: editingRecord.value,
      });
      open.value = false;
      return true;
    } catch (error) {
      errorMessage.value = apiErrorMessage(error, errorFallback);
      return false;
    } finally {
      submitLoading.value = false;
    }
  }

  return {
    open: readonly(open),
    mode: readonly(mode),
    submitLoading: readonly(submitLoading),
    editingRecord: readonly(editingRecord),
    errorMessage: readonly(errorMessage),
    openCreate,
    openEdit,
    close,
    handleSubmit,
    initialValues: readonly(initialValues) as AdminCrudFormState<TValues>['initialValues'],
  };
}
