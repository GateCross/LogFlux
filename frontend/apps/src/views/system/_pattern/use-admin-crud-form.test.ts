
import { describe, expect, it, vi } from 'vitest';

import { useAdminCrudForm } from './use-admin-crud-form';

describe('useAdminCrudForm', () => {
  it('openCreate applies defaults and openEdit maps record', () => {
    const form = useAdminCrudForm<{ name: string }>({
      createDefaults: () => ({ name: '' }),
      mapRecordToValues: (record) => ({
        name: (record as { name: string }).name,
      }),
      submit: async () => {},
    });

    form.openCreate();
    expect(form.open.value).toBe(true);
    expect(form.mode.value).toBe('create');
    expect(form.initialValues.value).toEqual({ name: '' });
    expect(form.editingRecord.value).toBeNull();

    form.openEdit({ name: 'bob', id: 2 });
    expect(form.mode.value).toBe('edit');
    expect(form.initialValues.value).toEqual({ name: 'bob' });
    expect(form.editingRecord.value).toEqual({ name: 'bob', id: 2 });
  });

  it('handleSubmit success closes modal; failure keeps open with errorMessage', async () => {
    const submit = vi
      .fn()
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce({ message: '网络错误' });

    const form = useAdminCrudForm<{ name: string }>({
      createDefaults: () => ({ name: 'n' }),
      mapRecordToValues: () => ({ name: 'n' }),
      submit,
      errorFallback: '保存失败',
    });

    form.openCreate();
    const ok = await form.handleSubmit({ name: 'n' });
    expect(ok).toBe(true);
    expect(form.open.value).toBe(false);
    expect(form.submitLoading.value).toBe(false);

    form.openCreate();
    const fail = await form.handleSubmit({ name: 'n' });
    expect(fail).toBe(false);
    expect(form.open.value).toBe(true);
    expect(form.errorMessage.value).toBe('网络错误');
    expect(form.submitLoading.value).toBe(false);
  });
});
