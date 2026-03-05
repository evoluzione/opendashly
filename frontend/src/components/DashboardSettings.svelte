<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import type { DashboardChartSetting } from '../services/dashboard_settings';
  import {
    DASHBOARD_GRID_COLUMNS,
    dashboardSettingsState,
    getDefaultDashboardSettings,
    loadDashboardSettings,
    mergeWithDefaultDashboardSettings,
    saveDashboardSettings
  } from '../lib/stores/dashboard_settings';

  type ChartDefinition = {
    key: string;
    label: string;
    description: string;
  };

  type DragState = {
    key: string;
    source: 'grid' | 'disabled';
    mode: 'move' | 'resizeWidth';
    pointerId: number;
    startClientX: number;
    startClientY: number;
    baseW: number;
    didDrag: boolean;
    targetKey: string | null;
    targetMode: 'card' | 'slot' | null;
    overCanvas: boolean;
    overDisabledSidebar: boolean;
    captureEl: HTMLElement | null;
  };

  const GRID_GAP = 10;
  const ROW_HEIGHT = 64;
  const FIXED_H = 2;
  const MIN_W = 1;
  const MAX_W = DASHBOARD_GRID_COLUMNS;

  const chartCatalog: ChartDefinition[] = [
    {
      key: 'apdex_gauge',
      label: 'Gauge APDEX',
      description: 'Indicatore sintetico di soddisfazione basato sui tempi risposta.'
    },
    {
      key: 'error_rate_gauge',
      label: 'Gauge Error Rate',
      description: 'Percentuale complessiva di richieste in errore.'
    },
    {
      key: 'throughput_gauge',
      label: 'Gauge Throughput',
      description: 'Riepilogo richieste totali e velocita media.'
    },
    {
      key: 'latency_distribution',
      label: 'Distribuzione Latenza',
      description: 'Istogramma dei bucket di latenza.'
    },
    {
      key: 'latency_percentiles',
      label: 'Percentili Latenza',
      description: 'Serie temporali P50/P95/P99.'
    },
    {
      key: 'throughput_timeseries',
      label: 'Throughput nel Tempo',
      description: 'Serie temporale richieste ed errori.'
    },
    {
      key: 'error_rate_timeseries',
      label: 'Error Rate nel Tempo',
      description: 'Serie temporale percentuale errori.'
    },
    {
      key: 'slo_compliance',
      label: 'SLO Compliance',
      description: 'Percentuale finestre conformi a target P95.'
    },
    {
      key: 'error_budget_burn',
      label: 'Error Budget Burn',
      description: 'Consumo budget errori su finestre 1h/6h.'
    },
    {
      key: 'service_latency_rank',
      label: 'Latenza per Servizio',
      description: 'Ranking servizi con P95 peggiore.'
    },
    {
      key: 'service_throughput',
      label: 'Throughput per Servizio',
      description: 'Volume richieste aggregato per servizio.'
    },
    {
      key: 'availability_trend',
      label: 'Disponibilita nel Tempo',
      description: 'Trend availability = 100% - error rate.'
    },
    {
      key: 'slowest_endpoints',
      label: 'Endpoint Piu Lenti',
      description: 'Top endpoint per P95.'
    },
    {
      key: 'top_endpoints_throughput',
      label: 'Top Endpoint per Throughput',
      description: 'Endpoint ordinati per numero di richieste.'
    },
    {
      key: 'error_hotspots',
      label: 'Hotspot Errori',
      description: 'Endpoint con piu errori o error rate elevato.'
    }
  ];

  const chartByKey = new Map(chartCatalog.map((chart) => [chart.key, chart]));
  const defaultByKey = new Map(getDefaultDashboardSettings().map((setting) => [setting.key, setting]));

  let localSettings: DashboardChartSetting[] = [];
  let savedSnapshot: DashboardChartSetting[] = [];
  let enabledSettings: DashboardChartSetting[] = [];
  let disabledSettings: DashboardChartSetting[] = [];
  let editorGridRows = 1;

  let isDirty = false;
  let saving = false;
  let viewportWidth = 1200;
  let canEdit = true;

  let editorEl: HTMLDivElement | null = null;
  let controlsEl: HTMLDivElement | null = null;

  let dragState: DragState | null = null;
  let hasLoadedInitialSettings = false;

  function cloneSettings(settings: DashboardChartSetting[]): DashboardChartSetting[] {
    return settings.map((setting) => ({ ...setting }));
  }

  function serializeSettings(settings: DashboardChartSetting[]): string {
    return JSON.stringify(
      [...settings]
        .sort((a, b) => a.key.localeCompare(b.key))
        .map((setting) => ({
          key: setting.key,
          enabled: setting.enabled,
          order: setting.order,
          x: setting.x,
          y: setting.y,
          w: setting.w,
          h: setting.h
        }))
    );
  }

  function syncDirty() {
    isDirty = serializeSettings(localSettings) !== serializeSettings(savedSnapshot);
  }

  function normalizedW(value: number | undefined, fallbackW: number): number {
    if (!Number.isFinite(value)) return fallbackW;
    return Math.max(MIN_W, Math.min(MAX_W, Number(value)));
  }

  function getSettingWidth(key: string, settingMap: Map<string, DashboardChartSetting>): number {
    const fallback = defaultByKey.get(key);
    const current = settingMap.get(key);
    return normalizedW(current?.w, fallback?.w ?? 3);
  }

  function applyNormalized(settings: DashboardChartSetting[]) {
    localSettings = mergeWithDefaultDashboardSettings(settings).map((setting) => ({
      ...setting,
      w: normalizedW(setting.w, defaultByKey.get(setting.key)?.w ?? 3),
      h: FIXED_H
    }));
    syncDirty();
  }

  function initLocalSettings(settings: DashboardChartSetting[]) {
    localSettings = mergeWithDefaultDashboardSettings(settings).map((setting) => ({
      ...setting,
      w: normalizedW(setting.w, defaultByKey.get(setting.key)?.w ?? 3),
      h: FIXED_H
    }));
    savedSnapshot = cloneSettings(localSettings);
    isDirty = false;
  }

  function getEnabledOrderKeys(source?: DashboardChartSetting[]): string[] {
    return [...(source ?? localSettings)]
      .filter((setting) => setting.enabled && chartByKey.has(setting.key))
      .sort((a, b) => {
        if ((a.y ?? 0) !== (b.y ?? 0)) return (a.y ?? 0) - (b.y ?? 0);
        if ((a.x ?? 0) !== (b.x ?? 0)) return (a.x ?? 0) - (b.x ?? 0);
        return (a.order ?? 0) - (b.order ?? 0);
      })
      .map((setting) => setting.key);
  }

  function packEnabledOrder(enabledKeys: string[], settingMap: Map<string, DashboardChartSetting>) {
    const out = new Map<string, { x: number; y: number; w: number; h: number }>();
    let row = 0;
    let col = 0;

    // Keep the exact order chosen by drag&drop; only wrap rows when needed.
    for (const key of enabledKeys) {
      const width = getSettingWidth(key, settingMap);

      if (col + width > DASHBOARD_GRID_COLUMNS) {
        row += 1;
        col = 0;
      }

      out.set(key, {
        x: col,
        y: row * FIXED_H,
        w: width,
        h: FIXED_H
      });
      col += width;
    }

    return out;
  }

  function commitEnabledOrder(enabledKeys: string[], patchByKey?: Map<string, Partial<DashboardChartSetting>>) {
    const base = mergeWithDefaultDashboardSettings(localSettings);
    const map = new Map(base.map((setting) => [setting.key, setting]));

    if (patchByKey) {
      for (const [key, patch] of patchByKey.entries()) {
        const current = map.get(key);
        if (!current) continue;
        map.set(key, { ...current, ...patch });
      }
    }

    const packed = packEnabledOrder(enabledKeys, map);
    if (!packed) return;

    const enabledSet = new Set(enabledKeys);
    const orderMap = new Map(enabledKeys.map((key, index) => [key, (index + 1) * 10]));

    let disabledOrder = (enabledKeys.length + 1) * 10;
    const next = base.map((setting) => {
      const current = map.get(setting.key) ?? setting;
      if (enabledSet.has(setting.key)) {
        const pos = packed.get(setting.key);
        if (!pos) return current;
        return {
          ...current,
          enabled: true,
          order: orderMap.get(setting.key) ?? current.order,
          x: pos.x,
          y: pos.y,
          w: pos.w,
          h: FIXED_H
        };
      }

      const out = { ...current, enabled: false, order: disabledOrder, h: FIXED_H };
      disabledOrder += 10;
      return out;
    });

    applyNormalized(next);
  }

  function disableWidget(key: string) {
    const enabledKeys = getEnabledOrderKeys().filter((item) => item !== key);
    commitEnabledOrder(enabledKeys);
  }

  function activateWidgetByDrop(key: string, targetKey: string | null) {
    const enabledKeys = getEnabledOrderKeys();
    if (enabledKeys.includes(key)) return;

    let targetIndex = enabledKeys.length;
    if (targetKey) {
      const idx = enabledKeys.indexOf(targetKey);
      if (idx >= 0) targetIndex = idx;
    }

    enabledKeys.splice(targetIndex, 0, key);
    commitEnabledOrder(enabledKeys);
  }

  function reorderEnabledWidget(
    key: string,
    targetKey: string | null,
    targetMode: 'card' | 'slot' | null,
    overCanvas: boolean
  ) {
    const currentEnabledKeys = getEnabledOrderKeys();
    const fromIndex = currentEnabledKeys.indexOf(key);
    if (fromIndex < 0) return;

    let patchByKey: Map<string, Partial<DashboardChartSetting>> | undefined;
    if (targetMode === 'card' && targetKey && targetKey !== key) {
      const sourceSetting = localSettings.find((item) => item.key === key);
      const targetSetting = localSettings.find((item) => item.key === targetKey);
      if (sourceSetting?.enabled && targetSetting?.enabled) {
        const sourceW = normalizedW(sourceSetting.w, defaultByKey.get(sourceSetting.key)?.w ?? 3);
        const targetW = normalizedW(targetSetting.w, defaultByKey.get(targetSetting.key)?.w ?? 3);
        patchByKey = new Map<string, Partial<DashboardChartSetting>>();
        patchByKey.set(key, { w: targetW, h: FIXED_H });
        patchByKey.set(targetKey, { w: sourceW, h: FIXED_H });
      }
    }

    if (!overCanvas) {
      return;
    }

    if (targetMode === 'card' && targetKey && targetKey !== key) {
      const targetIndex = currentEnabledKeys.indexOf(targetKey);
      if (targetIndex >= 0) {
        // Exact index swap: only source/target exchange slots.
        const swapped = [...currentEnabledKeys];
        const targetValue = swapped[targetIndex];
        swapped[targetIndex] = swapped[fromIndex];
        swapped[fromIndex] = targetValue;
        commitEnabledOrder(swapped, patchByKey);
        return;
      }
    }

    if (targetMode === 'slot' && targetKey && targetKey !== key) {
      const targetIndex = currentEnabledKeys.indexOf(targetKey);
      if (targetIndex >= 0) {
        const reordered = [...currentEnabledKeys];
        reordered.splice(fromIndex, 1);
        const adjustedTarget = fromIndex < targetIndex ? targetIndex - 1 : targetIndex;
        reordered.splice(adjustedTarget, 0, key);
        commitEnabledOrder(reordered);
        return;
      }
    }

    commitEnabledOrder(currentEnabledKeys, patchByKey);
  }

  function resetLayout() {
    applyNormalized(getDefaultDashboardSettings());
  }

  async function handleSave() {
    saving = true;
    try {
      const snapshot = cloneSettings(localSettings);
      await saveDashboardSettings(localSettings);
      savedSnapshot = snapshot;
      isDirty = false;
      await loadDashboardSettings();
    } finally {
      saving = false;
    }
  }

  function cancelChanges() {
    localSettings = cloneSettings(savedSnapshot);
    isDirty = false;
  }

  function getCanvasStyle(setting: DashboardChartSetting): string {
    return `grid-column:${(setting.x ?? 0) + 1} / span ${setting.w ?? 3};grid-row:${(setting.y ?? 0) + 1} / span ${FIXED_H};`;
  }

  function pointInsideRect(x: number, y: number, rect: DOMRect): boolean {
    return x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom;
  }

  function beginDrag(
    event: PointerEvent,
    key: string,
    source: 'grid' | 'disabled',
    mode: 'move' | 'resizeWidth' = 'move'
  ) {
    if (!canEdit) return;
    const current = localSettings.find((item) => item.key === key);
    dragState = {
      key,
      source,
      mode,
      pointerId: event.pointerId,
      startClientX: event.clientX,
      startClientY: event.clientY,
      baseW: current?.w ?? 3,
      didDrag: false,
      targetKey: null,
      targetMode: null,
      overCanvas: false,
      overDisabledSidebar: false,
      captureEl: event.currentTarget as HTMLElement | null
    };

    if (dragState.captureEl && typeof dragState.captureEl.setPointerCapture === 'function') {
      dragState.captureEl.setPointerCapture(event.pointerId);
    }

    window.addEventListener('pointermove', onPointerMove);
    window.addEventListener('pointerup', onPointerUp);
    window.addEventListener('pointercancel', onPointerUp);
    event.preventDefault();
  }

  function updateDropTarget(clientX: number, clientY: number) {
    if (!dragState) return;

    dragState.overCanvas = !!(editorEl && pointInsideRect(clientX, clientY, editorEl.getBoundingClientRect()));
    if (controlsEl) {
      const controlsRect = controlsEl.getBoundingClientRect();
      const relaxedX = clientX >= controlsRect.left - 12;
      const relaxedY = clientY >= controlsRect.top - 20 && clientY <= controlsRect.bottom + 20;
      dragState.overDisabledSidebar = relaxedX && relaxedY;
    } else {
      dragState.overDisabledSidebar = false;
    }

    dragState.targetKey = null;
    dragState.targetMode = null;
    if (dragState.overCanvas && editorEl) {
      const cards = editorEl.querySelectorAll<HTMLElement>('[data-widget-key]');
      const candidates: Array<{ key: string; rect: DOMRect }> = [];
      for (const card of cards) {
        const key = card.dataset.widgetKey;
        if (!key || (dragState.source === 'grid' && key === dragState.key)) continue;
        const rect = card.getBoundingClientRect();
        candidates.push({ key, rect });
        if (pointInsideRect(clientX, clientY, rect)) {
          dragState.targetKey = key;
          dragState.targetMode = 'card';
          break;
        }
      }

      if (!dragState.targetKey && candidates.length > 0) {
        candidates.sort((a, b) => {
          const topDelta = a.rect.top - b.rect.top;
          if (Math.abs(topDelta) > 8) return topDelta;
          return a.rect.left - b.rect.left;
        });

        const slot = candidates.find(({ rect }) => {
          const beforeRow = clientY < rect.top;
          const sameRow = clientY >= rect.top && clientY <= rect.bottom;
          const beforeColInRow = sameRow && clientX < rect.left + rect.width / 2;
          return beforeRow || beforeColInRow;
        });

        if (slot) {
          dragState.targetKey = slot.key;
          dragState.targetMode = 'slot';
        }
      }
    }
  }

  function onPointerMove(event: PointerEvent) {
    if (!dragState || dragState.pointerId !== event.pointerId) return;
    const dx = Math.abs(event.clientX - dragState.startClientX);
    const dy = Math.abs(event.clientY - dragState.startClientY);
    if (!dragState.didDrag && (dx > 4 || dy > 4)) {
      dragState.didDrag = true;
    }
    if (!dragState.didDrag) return;

    if (dragState.mode === 'resizeWidth' && dragState.source === 'grid') {
      const gridWidth = editorEl?.getBoundingClientRect().width ?? 0;
      if (gridWidth <= 0) return;
      const colWidth = (gridWidth - GRID_GAP * (DASHBOARD_GRID_COLUMNS - 1)) / DASHBOARD_GRID_COLUMNS;
      const step = colWidth + GRID_GAP;
      const deltaCols = Math.round((event.clientX - dragState.startClientX) / step);
      const nextW = Math.max(MIN_W, Math.min(MAX_W, dragState.baseW + deltaCols));

      const patch = new Map<string, Partial<DashboardChartSetting>>();
      patch.set(dragState.key, { w: nextW, h: FIXED_H });
      commitEnabledOrder(getEnabledOrderKeys(), patch);
      return;
    }

    updateDropTarget(event.clientX, event.clientY);
  }

  function onPointerUp(event: PointerEvent) {
    if (!dragState || dragState.pointerId !== event.pointerId) return;
    if (!dragState.didDrag) {
      if (dragState.captureEl && typeof dragState.captureEl.releasePointerCapture === 'function') {
        dragState.captureEl.releasePointerCapture(event.pointerId);
      }
      dragState = null;
      window.removeEventListener('pointermove', onPointerMove);
      window.removeEventListener('pointerup', onPointerUp);
      window.removeEventListener('pointercancel', onPointerUp);
      return;
    }
    updateDropTarget(event.clientX, event.clientY);

    if (dragState.mode === 'move' && dragState.source === 'grid') {
      if (dragState.overDisabledSidebar) {
        disableWidget(dragState.key);
      } else {
        reorderEnabledWidget(dragState.key, dragState.targetKey, dragState.targetMode, dragState.overCanvas);
      }
    } else if (dragState.mode === 'move' && dragState.source === 'disabled' && dragState.overCanvas) {
      activateWidgetByDrop(dragState.key, dragState.targetKey);
    }

    if (dragState.captureEl && typeof dragState.captureEl.releasePointerCapture === 'function') {
      dragState.captureEl.releasePointerCapture(event.pointerId);
    }

    dragState = null;
    window.removeEventListener('pointermove', onPointerMove);
    window.removeEventListener('pointerup', onPointerUp);
    window.removeEventListener('pointercancel', onPointerUp);
  }

  onMount(async () => {
    await loadDashboardSettings();
    initLocalSettings($dashboardSettingsState.settings ?? []);
    hasLoadedInitialSettings = true;
  });

  onDestroy(() => {
    window.removeEventListener('pointermove', onPointerMove);
    window.removeEventListener('pointerup', onPointerUp);
    window.removeEventListener('pointercancel', onPointerUp);
  });

  $: canEdit = viewportWidth >= 900;

  $: if (
    !hasLoadedInitialSettings &&
    !isDirty &&
    $dashboardSettingsState.settings &&
    $dashboardSettingsState.settings.length > 0
  ) {
    initLocalSettings($dashboardSettingsState.settings);
    hasLoadedInitialSettings = true;
  }

  $: enabledSettings = [...localSettings]
    .filter((setting) => setting.enabled && chartByKey.has(setting.key))
    .sort((a, b) => {
      if ((a.y ?? 0) !== (b.y ?? 0)) return (a.y ?? 0) - (b.y ?? 0);
      if ((a.x ?? 0) !== (b.x ?? 0)) return (a.x ?? 0) - (b.x ?? 0);
      return (a.order ?? 0) - (b.order ?? 0);
    });

  $: disabledSettings = [...localSettings]
    .filter((setting) => !setting.enabled && chartByKey.has(setting.key))
    .sort((a, b) => (chartByKey.get(a.key)?.label ?? a.key).localeCompare(chartByKey.get(b.key)?.label ?? b.key));

  $: editorGridRows =
    enabledSettings.reduce((max, setting) => Math.max(max, (setting.y ?? 0) + FIXED_H), 0) + 1;
</script>

<svelte:window bind:innerWidth={viewportWidth} />

<section class="dashboard-settings">
  <header>
    <h2>Designer dashboard</h2>
    <p>
      Griglia fissa a 6 colonne e altezza card fissa. La larghezza e modificabile, la disposizione si compatta in
      automatico.
    </p>
  </header>

  {#if $dashboardSettingsState.error}
    <div class="status error">{$dashboardSettingsState.error}</div>
  {/if}

  <div class="toolbar">
    <button class="btn ghost" type="button" on:click={resetLayout} disabled={saving}>Reset layout</button>
    <button class="btn ghost" type="button" on:click={cancelChanges} disabled={!isDirty || saving}>Annulla</button>
    <button class="btn primary" type="button" on:click={handleSave} disabled={!isDirty || saving}>
      {saving ? 'Salvataggio...' : 'Salva'}
    </button>
  </div>

  {#if !canEdit}
    <div class="status">Editor disabilitato su mobile: usa desktop/tablet per modificare il layout.</div>
  {/if}

  <div class="designer-layout">
    <div class="canvas-wrapper" bind:this={editorEl}>
      <div
        class="canvas"
        class:drop-active={dragState?.source === 'disabled' && dragState?.overCanvas}
        style={`--columns:${DASHBOARD_GRID_COLUMNS};--row-height:${ROW_HEIGHT}px;--gap:${GRID_GAP}px;--rows:${editorGridRows};`}
      >
        {#each enabledSettings as setting (setting.key)}
          <article
            class="widget-card"
            class:dragging={dragState?.source === 'grid' && dragState?.key === setting.key}
            class:drop-target={dragState?.targetKey === setting.key}
            data-widget-key={setting.key}
            style={getCanvasStyle(setting)}
          >
            <header class="widget-header" on:pointerdown={(event) => beginDrag(event, setting.key, 'grid')}>
              <span class="drag-handle" title="Trascina">⋮⋮</span>
              <div>
                <h3>{chartByKey.get(setting.key)?.label}</h3>
                <p>{chartByKey.get(setting.key)?.description}</p>
              </div>
              <span class="chip-size" aria-label={`Larghezza ${setting.w}/6`}>
                {setting.w}/6
              </span>
            </header>

            <div class="widget-preview">
              <div class="mock-line"></div>
              <div class="mock-bars">
                <span style="height: 36%"></span>
                <span style="height: 62%"></span>
                <span style="height: 48%"></span>
                <span style="height: 78%"></span>
                <span style="height: 54%"></span>
              </div>
            </div>
            {#if canEdit}
              <button
                type="button"
                class="resize-width-handle"
                aria-label={`Ridimensiona larghezza ${chartByKey.get(setting.key)?.label}`}
                on:pointerdown={(event) => beginDrag(event, setting.key, 'grid', 'resizeWidth')}
              ></button>
            {/if}
          </article>
        {/each}
      </div>
    </div>

    <aside
      class="controls"
      bind:this={controlsEl}
      class:drop-active={dragState?.source === 'grid' && dragState?.overDisabledSidebar}
    >
      <h3>Widget disattivati</h3>
      <p class="controls-subtitle">Trascina nella griglia per attivare, trascina qui per disattivare.</p>

      {#if disabledSettings.length === 0}
        <div class="empty-state">Nessun widget disattivato.</div>
      {:else}
        {#each disabledSettings as setting (setting.key)}
          <button
            type="button"
            class="disabled-card"
            class:dragging={dragState?.source === 'disabled' && dragState?.key === setting.key}
            on:pointerdown={(event) => beginDrag(event, setting.key, 'disabled')}
          >
            <span class="disabled-drag">⋮⋮</span>
            <span>{chartByKey.get(setting.key)?.label}</span>
          </button>
        {/each}
      {/if}
    </aside>
  </div>
</section>

<style>
  .dashboard-settings {
    --brand-indigo: #6366f1;
    --brand-violet: #8b5cf6;
    --text-strong: #0f172a;
    --text-muted: #64748b;
    --panel-bg: #ffffff;
    --panel-border: rgba(15, 23, 42, 0.06);
    --panel-shadow:
      0 1px 3px rgba(15, 23, 42, 0.08),
      0 4px 12px rgba(15, 23, 42, 0.04);
  }

  .dashboard-settings {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }

  header h2 {
    font-size: 22px;
    margin: 0 0 8px;
    color: var(--text-strong);
  }

  header p {
    margin: 0;
    color: var(--text-muted);
    font-size: 14px;
    max-width: 900px;
  }

  .status {
    padding: 12px;
    border-radius: 12px;
    background: #f8fafc;
    color: var(--text-muted);
    font-size: 14px;
    border: 1px solid var(--panel-border);
  }

  .status.error {
    background: rgba(239, 68, 68, 0.05);
    color: #ef4444;
    border-color: rgba(239, 68, 68, 0.15);
  }

  .toolbar {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .btn {
    border: 1px solid #cbd5e1;
    background: #fff;
    color: #1e293b;
    border-radius: 8px;
    height: 34px;
    padding: 0 14px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition:
      background 0.15s ease,
      border-color 0.15s ease,
      color 0.15s ease,
      transform 0.15s ease;
  }

  .btn.primary {
    border: none;
    background: linear-gradient(135deg, var(--brand-indigo) 0%, var(--brand-violet) 100%);
    color: #fff;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }

  .btn:hover:not(:disabled) {
    background: #f8fafc;
    border-color: #94a3b8;
    color: #334155;
  }

  .btn.primary:hover:not(:disabled) {
    background: linear-gradient(135deg, #5558ee 0%, #7c56ee 100%);
    border-color: transparent;
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(99, 102, 241, 0.4);
    color: #fff;
  }

  .btn:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }

  .designer-layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 320px;
    gap: 14px;
    align-items: stretch;
  }

  .canvas-wrapper {
    border: 1px solid var(--panel-border);
    border-radius: 16px;
    background: var(--panel-bg);
    box-shadow: var(--panel-shadow);
    padding: 12px;
    height: auto;
    overflow: visible;
  }

  .canvas {
    width: 100%;
    display: grid;
    gap: var(--gap);
    grid-template-columns: repeat(var(--columns), minmax(0, 1fr));
    grid-auto-rows: var(--row-height);
    grid-template-rows: repeat(var(--rows), var(--row-height));
    min-height: 520px;
    position: relative;
    transition: box-shadow 0.15s ease;
  }

  .canvas.drop-active {
    box-shadow: inset 0 0 0 2px rgba(99, 102, 241, 0.45);
    border-radius: 10px;
  }

  .widget-card {
    border-radius: 12px;
    border: 1px solid var(--panel-border);
    background: #fff;
    box-shadow:
      0 1px 3px rgba(15, 23, 42, 0.08),
      0 3px 10px rgba(15, 23, 42, 0.04);
    display: flex;
    flex-direction: column;
    position: relative;
    overflow: hidden;
    min-width: 0;
    transition: transform 0.15s ease, box-shadow 0.15s ease;
  }

  .widget-card.drop-target {
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.35);
    transform: translateY(-2px);
  }

  .widget-card.dragging {
    opacity: 0.55;
    transform: scale(0.98);
    border-style: dashed;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  .widget-header {
    display: flex;
    gap: 8px;
    align-items: flex-start;
    padding: 10px;
    border-bottom: 1px solid #e2e8f0;
    cursor: grab;
    user-select: none;
    touch-action: none;
  }

  .widget-header:active {
    cursor: grabbing;
  }

  .drag-handle {
    width: 22px;
    height: 22px;
    border-radius: 6px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    color: var(--text-muted);
    flex: 0 0 22px;
  }

  .widget-header h3 {
    margin: 0;
    font-size: 12px;
    color: var(--text-strong);
  }

  .widget-header p {
    margin: 2px 0 0;
    font-size: 11px;
    color: var(--text-muted);
    line-height: 1.3;
  }

  .chip-size {
    margin-left: auto;
    border: 1px solid #c7d2fe;
    background: #eef2ff;
    color: #4338ca;
    border-radius: 999px;
    font-size: 11px;
    height: 24px;
    min-width: 44px;
    padding: 0 10px;
    cursor: default;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
  }

  .widget-preview {
    flex: 1;
    min-height: 0;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .mock-line {
    height: 8px;
    border-radius: 999px;
    background: linear-gradient(90deg, #e2e8f0 0%, #cbd5e1 48%, #e2e8f0 100%);
  }

  .mock-bars {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    align-items: end;
    gap: 6px;
    flex: 1;
  }

  .mock-bars span {
    display: block;
    width: 100%;
    border-radius: 6px 6px 2px 2px;
    background: linear-gradient(180deg, #a5b4fc 0%, #6366f1 100%);
  }

  .controls {
    border: 1px solid var(--panel-border);
    border-radius: 16px;
    background: #fff;
    box-shadow: var(--panel-shadow);
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    overflow: visible;
    height: auto;
    transition: box-shadow 0.15s ease, border-color 0.15s ease;
  }

  .controls.drop-active {
    border-color: var(--brand-indigo);
    box-shadow: inset 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  .controls h3 {
    margin: 0;
    font-size: 13px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
  }

  .controls-subtitle {
    margin: 0;
    font-size: 12px;
    color: var(--text-muted);
  }

  .empty-state {
    border: 1px dashed #cbd5e1;
    border-radius: 10px;
    padding: 12px;
    font-size: 12px;
    color: var(--text-muted);
    background: #f8fafc;
  }

  .disabled-card {
    width: 100%;
    border: 1px solid var(--panel-border);
    border-radius: 10px;
    background: #f8fafc;
    color: var(--text-strong);
    padding: 10px;
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: grab;
    font-size: 12px;
    font-weight: 600;
    text-align: left;
    touch-action: none;
  }

  .disabled-card:active {
    cursor: grabbing;
  }

  .disabled-card.dragging {
    opacity: 0.55;
    border-style: dashed;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  .disabled-drag {
    width: 18px;
    height: 18px;
    border-radius: 6px;
    border: 1px solid #e2e8f0;
    background: #fff;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: var(--text-muted);
    font-size: 11px;
    flex: 0 0 18px;
  }

  .resize-width-handle {
    position: absolute;
    right: 0;
    bottom: 0;
    width: 24px;
    height: 24px;
    border: none;
    border-top: 1px solid #c7d2fe;
    border-left: 1px solid #c7d2fe;
    border-top-left-radius: 12px;
    background: linear-gradient(135deg, #e0e7ff 0%, #a5b4fc 100%);
    cursor: ew-resize;
    touch-action: none;
    opacity: 0.9;
    transition: opacity 0.15s ease, transform 0.15s ease;
  }

  .resize-width-handle:hover {
    opacity: 1;
    transform: scale(1.05);
  }

  @media (max-width: 1200px) {
    .designer-layout {
      grid-template-columns: 1fr;
    }

    .controls {
      height: auto;
    }
  }

  @media (max-width: 900px) {
    .canvas {
      grid-template-columns: 1fr;
      grid-template-rows: none;
      grid-auto-rows: auto;
      min-height: 0;
    }

    .widget-card {
      grid-column: 1 / -1 !important;
      grid-row: auto !important;
      min-height: 180px;
    }
  }
</style>
