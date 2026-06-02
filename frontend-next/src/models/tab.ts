/**
 * Multi-Tab Model (Task 8.1 / Req 6.4 / Property 11).
 *
 * Migrated from the legacy Vue frontend's tab store
 * (`frontend/src/store/modules/tab/index.ts`).
 *
 * Responsibilities:
 *  - Maintain the ordered list of open tabs and the active tab ID.
 *  - Open / switch / close tabs with proper neighbor-selection invariants.
 *  - Bulk close operations (closeOtherTabs, closeLeftTabs, closeRightTabs, closeAllTabs)
 *    that respect **fixed** tabs (they are never removed by bulk operations).
 *  - Pin / unpin (fix / unfix) tabs.
 *  - Persist and restore tab state via localStorage (keys prefixed by the storage util).
 *
 * Invariants (Req 6.4 / Property 11):
 *  - No duplicate IDs across the tab list.
 *  - At most one active tab (the active ID appears at most once).
 *  - After every close operation the active tab ID always points to an existing tab
 *    (or is null when no tabs remain).
 *  - Fixed tabs cannot be removed by closeAll / closeLeft / closeRight / closeOther.
 *
 * Storage key: `globalTabs` (matches the key used in `src/models/auth.ts` checkTabClear).
 */
import { useState, useCallback } from 'react';
import { getStorage, setStorage } from '@/utils/storage';
import { ROUTE_HOME } from '@/constants/app';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Single tab descriptor. */
export interface TabItem {
  /** Unique identifier for the tab (typically derived from the route name or path). */
  id: string;
  /** Human-readable display label. */
  label: string;
  /** The URL path this tab represents. */
  path: string;
  /** Optional i18n key for the label (used when the label should be localised). */
  i18nKey?: string;
  /** Whether the tab is pinned / fixed (immune to bulk close operations). */
  fixed?: boolean;
}

// ---------------------------------------------------------------------------
// Storage helpers
// ---------------------------------------------------------------------------

const STORAGE_KEY = 'globalTabs';

interface StoredTabData {
  tabs: TabItem[];
  activeTabId: string | null;
}

// ---------------------------------------------------------------------------
// Utility functions (exported per specification)
// ---------------------------------------------------------------------------

/**
 * Return a shallow copy of the entire tab list.
 *
 * Useful for consumers that need a stable snapshot (e.g. for iteration in
 * effects or selectors) without depending on the hook's internal state reference.
 */
export function getAllTabs(tabs: TabItem[]): TabItem[] {
  return [...tabs];
}

/**
 * Find a tab by its unique ID. Returns `undefined` when not found.
 */
export function getTabById(tabs: TabItem[], id: string): TabItem | undefined {
  return tabs.find((t) => t.id === id);
}

// ---------------------------------------------------------------------------
// Tab Model Hook (default export, registered as Umi model `tab`)
// ---------------------------------------------------------------------------

export default function useTabModel() {
  const [tabs, setTabs] = useState<TabItem[]>([]);
  const [activeTabId, setActiveTabId] = useState<string | null>(null);

  // -------------------------------------------------------------------------
  // openTab
  // -------------------------------------------------------------------------

  /**
   * Open a new tab or switch to an existing one.
   *
   * Resolution order:
   *  1. If a tab with the same `id` already exists, switch to it (no duplicate).
   *  2. If no ID match but a tab with the same `path` exists, switch to that tab
   *     (avoids duplicate routes under different IDs).
   *  3. Otherwise append a new tab and make it active.
   */
  const openTab = useCallback((tab: TabItem) => {
    setTabs((prev) => {
      // 1. Exact ID match -> just switch, no mutation needed.
      const existingById = prev.find((t) => t.id === tab.id);
      if (existingById) {
        setActiveTabId(existingById.id);
        return prev;
      }

      // 2. Path match -> switch to existing tab, avoiding duplicate paths.
      const existingByPath = prev.find((t) => t.path === tab.path);
      if (existingByPath) {
        setActiveTabId(existingByPath.id);
        return prev;
      }

      // 3. New tab -> append and activate.
      setActiveTabId(tab.id);
      return [...prev, { ...tab }];
    });
  }, []);

  // -------------------------------------------------------------------------
  // switchTab
  // -------------------------------------------------------------------------

  /**
   * Set the active tab by ID. No-op if the ID does not correspond to an open tab.
   */
  const switchTab = useCallback(
    (id: string) => {
      if (tabs.some((t) => t.id === id)) {
        setActiveTabId(id);
      }
    },
    [tabs],
  );

  // -------------------------------------------------------------------------
  // closeTab
  // -------------------------------------------------------------------------

  /**
   * Close a single tab.
   *
   * When the closed tab is the currently active one the next valid tab is
   * activated: right neighbour first, then left neighbour. If no tabs remain
   * the active ID becomes `null`.
   *
   * Fixed tabs **can** be closed by this direct operation (only bulk operations
   * respect the fixed guard).
   */
  const closeTab = useCallback(
    (id: string) => {
      const idx = tabs.findIndex((t) => t.id === id);
      if (idx === -1) return;

      const newTabs = tabs.filter((t) => t.id !== id);
      setTabs(newTabs);

      if (activeTabId === id) {
        if (newTabs.length === 0) {
          setActiveTabId(null);
        } else if (idx < newTabs.length) {
          // Right neighbour (the element that was at idx+1 now sits at idx).
          setActiveTabId(newTabs[idx].id);
        } else {
          // No right neighbour -> fall back to left neighbour.
          setActiveTabId(newTabs[idx - 1].id);
        }
      }
    },
    [tabs, activeTabId],
  );

  // -------------------------------------------------------------------------
  // closeOtherTabs
  // -------------------------------------------------------------------------

  /**
   * Close every tab **except** the specified one and any fixed tabs.
   * The surviving tab (or the first remaining fixed tab) becomes active.
   */
  const closeOtherTabs = useCallback(
    (id: string) => {
      const newTabs = tabs.filter((t) => t.id === id || t.fixed);
      setTabs(newTabs);

      if (newTabs.some((t) => t.id === id)) {
        setActiveTabId(id);
      } else if (newTabs.length > 0) {
        setActiveTabId(newTabs[0].id);
      } else {
        setActiveTabId(null);
      }
    },
    [tabs],
  );

  // -------------------------------------------------------------------------
  // closeLeftTabs
  // -------------------------------------------------------------------------

  /**
   * Close all tabs to the **left** of the specified tab, preserving fixed tabs.
   */
  const closeLeftTabs = useCallback(
    (id: string) => {
      const idx = tabs.findIndex((t) => t.id === id);
      if (idx <= 0) return; // nothing to the left, or tab not found

      const kept = tabs.filter((t, i) => i >= idx || t.fixed);
      setTabs(kept);

      // If the currently active tab was removed, activate the reference tab.
      if (!kept.some((t) => t.id === activeTabId)) {
        setActiveTabId(id);
      }
    },
    [tabs, activeTabId],
  );

  // -------------------------------------------------------------------------
  // closeRightTabs
  // -------------------------------------------------------------------------

  /**
   * Close all tabs to the **right** of the specified tab, preserving fixed tabs.
   */
  const closeRightTabs = useCallback(
    (id: string) => {
      const idx = tabs.findIndex((t) => t.id === id);
      if (idx === -1) return;

      const kept = tabs.filter((t, i) => i <= idx || t.fixed);
      setTabs(kept);

      // If the currently active tab was removed, activate the reference tab.
      if (!kept.some((t) => t.id === activeTabId)) {
        setActiveTabId(id);
      }
    },
    [tabs, activeTabId],
  );

  // -------------------------------------------------------------------------
  // closeAllTabs
  // -------------------------------------------------------------------------

  /**
   * Close all non-fixed tabs, then activate the home tab (first remaining tab).
   *
   * Fixed tabs are preserved. If no fixed tabs exist the tab list becomes empty
   * and the active ID is set to `null` (the layout should call `initHomeTab`
   * or navigate to the home route to re-create the home tab).
   */
  const closeAllTabs = useCallback(() => {
    const fixedTabs = tabs.filter((t) => t.fixed);
    setTabs(fixedTabs);

    if (fixedTabs.length > 0) {
      setActiveTabId(fixedTabs[0].id);
    } else {
      setActiveTabId(null);
    }
  }, [tabs]);

  // -------------------------------------------------------------------------
  // fixTab / unfixTab
  // -------------------------------------------------------------------------

  /** Pin (fix) a tab so it is immune to bulk close operations. */
  const fixTab = useCallback((id: string) => {
    setTabs((prev) => prev.map((t) => (t.id === id ? { ...t, fixed: true } : t)));
  }, []);

  /** Unpin (unfix) a tab. */
  const unfixTab = useCallback((id: string) => {
    setTabs((prev) => prev.map((t) => (t.id === id ? { ...t, fixed: false } : t)));
  }, []);

  // -------------------------------------------------------------------------
  // initHomeTab
  // -------------------------------------------------------------------------

  /**
   * Initialise with the home tab.
   *
   * - If cached tabs exist in localStorage **and** contain at least one entry,
   *   restore them (via `loadTabs`).
   * - Otherwise create the home tab (fixed, non-duplicate) and activate it.
   *
   * Call this once during layout mount to guarantee a valid initial state.
   */
  const initHomeTab = useCallback(() => {
    const cached = getStorage<StoredTabData>(STORAGE_KEY);
    if (cached && Array.isArray(cached.tabs) && cached.tabs.length > 0) {
      setTabs(cached.tabs);
      // Validate activeTabId against restored tabs.
      if (cached.activeTabId && cached.tabs.some((t) => t.id === cached.activeTabId)) {
        setActiveTabId(cached.activeTabId);
      } else {
        setActiveTabId(cached.tabs[0].id);
      }
      return;
    }

    // No valid cache -> create the home tab.
    const homeTab: TabItem = {
      id: ROUTE_HOME,
      label: 'Home',
      path: `/${ROUTE_HOME}`,
      i18nKey: 'route.home',
      fixed: true,
    };
    setTabs([homeTab]);
    setActiveTabId(homeTab.id);
  }, []);

  // -------------------------------------------------------------------------
  // cacheTabs / loadTabs
  // -------------------------------------------------------------------------

  /** Persist the current tab state to localStorage. */
  const cacheTabs = useCallback(() => {
    setStorage<StoredTabData>(STORAGE_KEY, { tabs, activeTabId });
  }, [tabs, activeTabId]);

  /**
   * Load tab state from localStorage.
   *
   * If the stored data is missing, malformed, or empty the current state is left
   * untouched (the caller should fall back to `initHomeTab` if needed).
   */
  const loadTabs = useCallback(() => {
    const cached = getStorage<StoredTabData>(STORAGE_KEY);
    if (cached && Array.isArray(cached.tabs) && cached.tabs.length > 0) {
      setTabs(cached.tabs);
      if (cached.activeTabId && cached.tabs.some((t) => t.id === cached.activeTabId)) {
        setActiveTabId(cached.activeTabId);
      } else {
        setActiveTabId(cached.tabs[0].id);
      }
    }
  }, []);

  // -------------------------------------------------------------------------
  // Public API
  // -------------------------------------------------------------------------

  return {
    // State
    tabs,
    activeTabId,

    // Single-tab operations
    openTab,
    switchTab,
    closeTab,

    // Bulk close operations
    closeOtherTabs,
    closeLeftTabs,
    closeRightTabs,
    closeAllTabs,

    // Pin / unpin
    fixTab,
    unfixTab,

    // Lifecycle
    initHomeTab,
    cacheTabs,
    loadTabs,
  };
}
