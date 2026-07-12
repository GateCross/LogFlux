/** list-detail mutation invalidate 单测 */

import { QueryClient, type QueryKey } from '@tanstack/vue-query';
import * as fc from 'fast-check';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  invalidateListDetailQueries,
  invalidateListDetailQueryKeys,
} from './list-detail-mutation';
import { qk } from './query-keys';

// helper 契约（mock client）

function mockQueryClient() {
  return {
    invalidateQueries: vi.fn(async () => undefined),
  };
}

describe('invalidateListDetailQueries', () => {
  it('invalidates with qk factory key (prefix path)', async () => {
    const client = mockQueryClient();
    const key = qk.system.users({ page: 1, pageSize: 20 });

    await invalidateListDetailQueries(client as never, key);

    expect(client.invalidateQueries).toHaveBeenCalledTimes(1);
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: key });
  });
});

describe('invalidateListDetailQueryKeys', () => {
  it('invalidates multiple related keys in parallel', async () => {
    const client = mockQueryClient();
    const keys = [qk.cron.list({}), qk.cron.detail(1)];

    await invalidateListDetailQueryKeys(client as never, keys);

    expect(client.invalidateQueries).toHaveBeenCalledTimes(2);
    expect(client.invalidateQueries).toHaveBeenNthCalledWith(1, {
      queryKey: keys[0],
    });
    expect(client.invalidateQueries).toHaveBeenNthCalledWith(2, {
      queryKey: keys[1],
    });
  });
});

// ---------------------------------------------------------------------------
// Property 7: mutation 后列表缓存与后端一致
// ---------------------------------------------------------------------------

type UserRow = { id: number; name: string };
type ListPayload = { list: UserRow[]; total: number };

/** 可模型化的内存后端（模拟 List_Data_In_Scope_Page 列表） */
class InMemoryUserBackend {
  private nextId = 1;
  private rows: UserRow[] = [];

  seed(items: Array<{ name: string }>): void {
    this.rows = items.map((item) => ({
      id: this.nextId++,
      name: item.name,
    }));
  }

  list(): ListPayload {
    return {
      list: this.rows.map((r) => ({ ...r })),
      total: this.rows.length,
    };
  }

  /** 与「重新进入页面拉取」同一权威源 */
  snapshot(): ListPayload {
    return this.list();
  }

  create(name: string): UserRow {
    const row = { id: this.nextId++, name };
    this.rows.push(row);
    return { ...row };
  }

  update(id: number, name: string): UserRow | null {
    const idx = this.rows.findIndex((r) => r.id === id);
    if (idx < 0) return null;
    this.rows[idx] = { id, name };
    return { ...this.rows[idx]! };
  }

  delete(id: number): boolean {
    const before = this.rows.length;
    this.rows = this.rows.filter((r) => r.id !== id);
    return this.rows.length < before;
  }
}

/** QueryClient 测试默认：staleTime 30s */
function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        gcTime: 60_000,
      },
    },
  });
}

/**
 * 成功 mutation 后 invalidate 列表前缀
 */
async function successfulMutationThenInvalidate(
  queryClient: QueryClient,
  listKeyPrefix: QueryKey,
  mutate: () => void,
): Promise<void> {
  mutate();
  await invalidateListDetailQueries(queryClient, listKeyPrefix);
}

/** invalidate 后再拉取 */
async function reFetchAfterInvalidate<T>(
  queryClient: QueryClient,
  queryKey: QueryKey,
  queryFn: () => Promise<T>,
): Promise<T> {
  return queryClient.fetchQuery({ queryKey, queryFn });
}

describe('mutation 后列表缓存与后端一致', () => {
  let queryClient: QueryClient;
  let backend: InMemoryUserBackend;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    backend = new InMemoryUserBackend();
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('example: create 成功 + invalidate 后 list cache 与后端 / 重拉等价', async () => {
    backend.seed([{ name: 'alice' }]);
    const listKey = qk.system.users({ page: 1, pageSize: 20 });
    const prefix: QueryKey = ['system', 'users'];
    const queryFn = async () => backend.list();

    await queryClient.prefetchQuery({ queryKey: listKey, queryFn });

    const stale = queryClient.getQueryData(listKey) as ListPayload;
    expect(stale.total).toBe(1);
    expect(stale.list.map((r) => r.name)).toEqual(['alice']);

    await successfulMutationThenInvalidate(queryClient, prefix, () => {
      backend.create('bob');
    });
    // 允许中间 loading；重拉后与后端一致
    const after = await reFetchAfterInvalidate(queryClient, listKey, queryFn);
    const reFetch = backend.snapshot();

    expect(after).toEqual(reFetch);
    expect(queryClient.getQueryData(listKey)).toEqual(reFetch);
    expect(after.total).toBe(2);
    expect(after.list.map((r) => r.name).sort()).toEqual(['alice', 'bob']);
  });

  it('example: update 成功 + invalidate 后 list cache 与后端一致', async () => {
    backend.seed([{ name: 'alice' }, { name: 'bob' }]);
    const listKey = qk.system.users({ page: 1, pageSize: 20 });
    const prefix: QueryKey = ['system', 'users'];
    const queryFn = async () => backend.list();

    await queryClient.prefetchQuery({ queryKey: listKey, queryFn });
    const targetId = backend.snapshot().list[0]!.id;

    await successfulMutationThenInvalidate(queryClient, prefix, () => {
      backend.update(targetId, 'alice-renamed');
    });
    const after = await reFetchAfterInvalidate(queryClient, listKey, queryFn);

    expect(after).toEqual(backend.snapshot());
    expect(after.list.find((r) => r.id === targetId)?.name).toBe(
      'alice-renamed',
    );
  });

  it('example: delete 成功 + invalidate 后 list cache 与后端一致', async () => {
    backend.seed([{ name: 'alice' }, { name: 'bob' }]);
    const listKey = qk.system.users({ page: 1, pageSize: 20 });
    const prefix: QueryKey = ['system', 'users'];
    const queryFn = async () => backend.list();

    await queryClient.prefetchQuery({ queryKey: listKey, queryFn });
    const targetId = backend.snapshot().list[0]!.id;

    await successfulMutationThenInvalidate(queryClient, prefix, () => {
      backend.delete(targetId);
    });
    const after = await reFetchAfterInvalidate(queryClient, listKey, queryFn);

    expect(after).toEqual(backend.snapshot());
    expect(after.total).toBe(1);
    expect(after.list.some((r) => r.id === targetId)).toBe(false);
  });

  it('example: 前缀 invalidate 使多页 list cache 均与后端一致', async () => {
    backend.seed([{ name: 'a' }, { name: 'b' }]);
    const page1 = qk.system.users({ page: 1, pageSize: 10 });
    const page2 = qk.system.users({ page: 2, pageSize: 10 });
    const prefix: QueryKey = ['system', 'users'];
    const queryFn = async () => backend.list();

    await queryClient.prefetchQuery({ queryKey: page1, queryFn });
    await queryClient.prefetchQuery({ queryKey: page2, queryFn });

    await successfulMutationThenInvalidate(queryClient, prefix, () => {
      backend.create('c');
    });

    const snap = backend.snapshot();
    expect(await reFetchAfterInvalidate(queryClient, page1, queryFn)).toEqual(
      snap,
    );
    expect(await reFetchAfterInvalidate(queryClient, page2, queryFn)).toEqual(
      snap,
    );
    expect(queryClient.getQueryData(page1)).toEqual(snap);
    expect(queryClient.getQueryData(page2)).toEqual(snap);
  });

  it('example: invalidateListDetailQueryKeys 同时刷新 list + detail', async () => {
    backend.seed([{ name: 'task-1' }]);
    const listKey = qk.cron.list({ resource: 'tasks' });
    const detailId = 1;
    const detailKey = qk.cron.detail(detailId);

    let detailName = 'task-1';
    const listFn = async () => backend.list();
    const detailFn = async () => ({ id: detailId, name: detailName });

    await queryClient.prefetchQuery({ queryKey: listKey, queryFn: listFn });
    await queryClient.prefetchQuery({ queryKey: detailKey, queryFn: detailFn });

    backend.create('task-2');
    detailName = 'task-1-updated';

    await invalidateListDetailQueryKeys(queryClient, [listKey, detailKey]);

    expect(await reFetchAfterInvalidate(queryClient, listKey, listFn)).toEqual(
      backend.snapshot(),
    );
    expect(
      await reFetchAfterInvalidate(queryClient, detailKey, detailFn),
    ).toEqual({ id: detailId, name: 'task-1-updated' });
  });

  it('example: 未 invalidate 时 staleTime 内仍陈旧（对照：证明 invalidate 必要性）', async () => {
    backend.seed([{ name: 'alice' }]);
    const listKey = qk.system.users({ page: 1, pageSize: 20 });
    const queryFn = async () => backend.list();

    await queryClient.prefetchQuery({ queryKey: listKey, queryFn });

    backend.create('bob');
    // 故意不 invalidate：staleTime 内 fetchQuery 命中缓存，与后端不一致
    const stillCached = await queryClient.fetchQuery({
      queryKey: listKey,
      queryFn,
    });
    expect(stillCached).toEqual({
      list: [{ id: 1, name: 'alice' }],
      total: 1,
    });
    expect(stillCached).not.toEqual(backend.snapshot());
    expect(queryClient.getQueryData(listKey)).not.toEqual(backend.snapshot());
  });

  it('Property 7: mutation 后列表缓存与后端一致 — 任意成功写后与重拉等价', async () => {
    const nameArb = fc
      .string({ minLength: 1, maxLength: 16 })
      .filter((s) => s.trim().length > 0);

    type MutationOp =
      | { type: 'create'; name: string }
      | { type: 'update'; index: number; name: string }
      | { type: 'delete'; index: number };

    const mutationArb: fc.Arbitrary<MutationOp> = fc.oneof(
      nameArb.map((name) => ({ type: 'create' as const, name })),
      fc.record({
        type: fc.constant('update' as const),
        index: fc.nat({ max: 20 }),
        name: nameArb,
      }),
      fc.record({
        type: fc.constant('delete' as const),
        index: fc.nat({ max: 20 }),
      }),
    );

    await fc.assert(
      fc.asyncProperty(
        fc.array(nameArb, { minLength: 0, maxLength: 8 }),
        fc.array(mutationArb, { minLength: 1, maxLength: 6 }),
        async (initialNames, ops) => {
          const client = createTestQueryClient();
          const store = new InMemoryUserBackend();
          store.seed(initialNames.map((name) => ({ name })));

          const listKey = qk.system.users({ page: 1, pageSize: 50 });
          const prefix: QueryKey = ['system', 'users'];
          const queryFn = async () => store.list();

          await client.prefetchQuery({ queryKey: listKey, queryFn });

          for (const op of ops) {
            const snap = store.snapshot().list;
            if (op.type === 'create') {
              store.create(op.name);
            } else if (op.type === 'update') {
              if (snap.length === 0) continue;
              const id = snap[op.index % snap.length]!.id;
              store.update(id, op.name);
            } else {
              if (snap.length === 0) continue;
              const id = snap[op.index % snap.length]!.id;
              store.delete(id);
            }
          }

          // 成功 mutation 序列后统一 invalidate（与 onSuccess 语义一致）
          await invalidateListDetailQueries(client, prefix);
          const cached = await reFetchAfterInvalidate(client, listKey, queryFn);
          const reFetch = store.snapshot();
          expect(cached).toEqual(reFetch);
          expect(client.getQueryData(listKey)).toEqual(reFetch);

          client.clear();
        },
      ),
      { numRuns: 100 },
    );
  });

  it('Property 7 unit: qk.caddy.servers 前缀 invalidate 后与后端一致', async () => {
    type Server = { id: number; name: string };
    let servers: Server[] = [
      { id: 1, name: 's1' },
      { id: 2, name: 's2' },
    ];
    const key = qk.caddy.servers();
    const queryFn = async () => servers.map((s) => ({ ...s }));

    await queryClient.prefetchQuery({ queryKey: key, queryFn });

    servers = servers.filter((s) => s.id !== 1);
    await invalidateListDetailQueries(queryClient, key);
    const after = await reFetchAfterInvalidate(queryClient, key, queryFn);

    expect(after).toEqual(servers);
    expect(after.map((s) => s.id)).toEqual([2]);
    expect(queryClient.getQueryData(key)).toEqual(servers);
  });
});
