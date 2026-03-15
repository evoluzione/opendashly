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
  import { locale, t } from '../lib/i18n';

  type ChartDefinition = {
    key: string;
    labelKey: string;
    descriptionKey: string;
  };

  type PreviewKind = 'gauge' | 'timeseries' | 'distribution' | 'table' | 'kpi';

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
      labelKey: 'dashboardSettings.chart.apdex_gauge.label',
      descriptionKey: 'dashboardSettings.chart.apdex_gauge.description'
    },
    {
      key: 'error_rate_gauge',
      labelKey: 'dashboardSettings.chart.error_rate_gauge.label',
      descriptionKey: 'dashboardSettings.chart.error_rate_gauge.description'
    },
    {
      key: 'throughput_gauge',
      labelKey: 'dashboardSettings.chart.throughput_gauge.label',
      descriptionKey: 'dashboardSettings.chart.throughput_gauge.description'
    },
    {
      key: 'latency_distribution',
      labelKey: 'dashboardSettings.chart.latency_distribution.label',
      descriptionKey: 'dashboardSettings.chart.latency_distribution.description'
    },
    {
      key: 'latency_percentiles',
      labelKey: 'dashboardSettings.chart.latency_percentiles.label',
      descriptionKey: 'dashboardSettings.chart.latency_percentiles.description'
    },
    {
      key: 'throughput_timeseries',
      labelKey: 'dashboardSettings.chart.throughput_timeseries.label',
      descriptionKey: 'dashboardSettings.chart.throughput_timeseries.description'
    },
    {
      key: 'error_rate_timeseries',
      labelKey: 'dashboardSettings.chart.error_rate_timeseries.label',
      descriptionKey: 'dashboardSettings.chart.error_rate_timeseries.description'
    },
    {
      key: 'slo_compliance',
      labelKey: 'dashboardSettings.chart.slo_compliance.label',
      descriptionKey: 'dashboardSettings.chart.slo_compliance.description'
    },
    {
      key: 'error_budget_burn',
      labelKey: 'dashboardSettings.chart.error_budget_burn.label',
      descriptionKey: 'dashboardSettings.chart.error_budget_burn.description'
    },
    {
      key: 'service_latency_rank',
      labelKey: 'dashboardSettings.chart.service_latency_rank.label',
      descriptionKey: 'dashboardSettings.chart.service_latency_rank.description'
    },
    {
      key: 'service_throughput',
      labelKey: 'dashboardSettings.chart.service_throughput.label',
      descriptionKey: 'dashboardSettings.chart.service_throughput.description'
    },
    {
      key: 'availability_trend',
      labelKey: 'dashboardSettings.chart.availability_trend.label',
      descriptionKey: 'dashboardSettings.chart.availability_trend.description'
    },
    {
      key: 'slowest_endpoints',
      labelKey: 'dashboardSettings.chart.slowest_endpoints.label',
      descriptionKey: 'dashboardSettings.chart.slowest_endpoints.description'
    },
    {
      key: 'top_endpoints_throughput',
      labelKey: 'dashboardSettings.chart.top_endpoints_throughput.label',
      descriptionKey: 'dashboardSettings.chart.top_endpoints_throughput.description'
    },
    {
      key: 'error_hotspots',
      labelKey: 'dashboardSettings.chart.error_hotspots.label',
      descriptionKey: 'dashboardSettings.chart.error_hotspots.description'
    }
  ];

  const chartByKey = new Map(chartCatalog.map((chart) => [chart.key, chart]));
  const defaultByKey = new Map(getDefaultDashboardSettings().map((setting) => [setting.key, setting]));

  const previewKindByKey: Record<string, PreviewKind> = {
    apdex_gauge: 'gauge',
    error_rate_gauge: 'gauge',
    throughput_gauge: 'gauge',
    latency_distribution: 'distribution',
    latency_percentiles: 'timeseries',
    throughput_timeseries: 'timeseries',
    error_rate_timeseries: 'timeseries',
    slo_compliance: 'kpi',
    error_budget_burn: 'kpi',
    service_latency_rank: 'table',
    service_throughput: 'table',
    availability_trend: 'timeseries',
    slowest_endpoints: 'table',
    top_endpoints_throughput: 'table',
    error_hotspots: 'table'
  };

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
  let controlsEl: HTMLElement | null = null;

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

  function getPreviewKind(key: string): PreviewKind {
    return previewKindByKey[key] ?? 'timeseries';
  }

  function getChartLabel(key: string): string {
    const chart = chartByKey.get(key);
    if (!chart) return key;
    return t($locale, chart.labelKey);
  }

  function getChartDescription(key: string): string {
    const chart = chartByKey.get(key);
    if (!chart) return '';
    return t($locale, chart.descriptionKey);
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
    .sort((a, b) => getChartLabel(a.key).localeCompare(getChartLabel(b.key)));

  $: editorGridRows =
    enabledSettings.reduce((max, setting) => Math.max(max, (setting.y ?? 0) + FIXED_H), 0) + 1;
</script>

<svelte:window bind:innerWidth={viewportWidth} />

<section class="dashboard-settings">
  <header>
    <h2>{t($locale, 'dashboardSettings.title')}</h2>
    <p>
      {t($locale, 'dashboardSettings.description')}
    </p>
  </header>

  {#if $dashboardSettingsState.error}
    <div class="status error">{$dashboardSettingsState.error}</div>
  {/if}

  <div class="toolbar">
    <button class="btn ghost" type="button" on:click={resetLayout} disabled={saving}>{t($locale, 'dashboardSettings.resetLayout')}</button>
    <button class="btn ghost" type="button" on:click={cancelChanges} disabled={!isDirty || saving}>{t($locale, 'dashboardSettings.cancel')}</button>
    <button class="btn primary" type="button" on:click={handleSave} disabled={!isDirty || saving}>
      {saving ? t($locale, 'dashboardSettings.saving') : t($locale, 'dashboardSettings.save')}
    </button>
  </div>

  {#if !canEdit}
    <div class="status">{t($locale, 'dashboardSettings.mobileDisabled')}</div>
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
              <span class="drag-handle" title={t($locale, 'dashboardSettings.drag')}>⋮⋮</span>
              <div class="widget-title-wrap">
                <h3>{getChartLabel(setting.key)}</h3>
                <div class="info-tooltip">
                  <button
                    type="button"
                    class="info-trigger"
                    aria-label={t($locale, 'dashboardSettings.infoOn', { label: getChartLabel(setting.key) })}
                    on:pointerdown|stopPropagation
                    on:click|stopPropagation
                  >
                    i
                  </button>
                  <span class="info-bubble" role="tooltip">
                    {getChartDescription(setting.key)}
                  </span>
                </div>
              </div>
              <span class="chip-size" aria-label={t($locale, 'dashboardSettings.width', { value: String(setting.w) })}>
                {setting.w}/6
              </span>
            </header>

            <div class="widget-preview">
              {#if getPreviewKind(setting.key) === 'gauge'}
                <div class="preview-gauge-wrap">
                  <div class="preview-gauge">
                    <span class="preview-gauge-value">94%</span>
                  </div>
                  <div class="preview-gauge-track"></div>
                </div>
              {:else if getPreviewKind(setting.key) === 'timeseries'}
                <div class="preview-cartesian-wrap">
                  <div class="preview-cartesian" aria-hidden="true">
                    <svg viewBox="0 0 120 56" preserveAspectRatio="none">
                      <line x1="8" y1="48" x2="114" y2="48" class="axis" />
                      <line x1="8" y1="8" x2="8" y2="48" class="axis" />
                      <polyline
                        points="10,38 28,34 44,36 62,24 78,28 94,20 112,16"
                        class="series-a"
                      />
                      <polyline
                        points="10,40 28,39 44,33 62,30 78,22 94,26 112,21"
                        class="series-b"
                      />
                    </svg>
                  </div>
                  <div class="preview-legend">
                    <span class="dot blue"></span>
                    <span class="dot orange"></span>
                    <span class="dot red"></span>
                  </div>
                </div>
              {:else if getPreviewKind(setting.key) === 'distribution'}
                <div class="preview-histogram">
                  <span style="height: 26%"></span>
                  <span style="height: 46%"></span>
                  <span style="height: 74%"></span>
                  <span style="height: 58%"></span>
                  <span style="height: 34%"></span>
                </div>
              {:else if getPreviewKind(setting.key) === 'table'}
                <div class="preview-table">
                  <div class="preview-table-header">
                    <span class="cell cell-title"></span>
                    <span class="cell cell-trend"></span>
                    <span class="cell cell-value"></span>
                  </div>
                  <div class="preview-table-body">
                    <div class="preview-table-line">
                      <span class="cell cell-title"></span>
                      <span class="cell cell-trend"></span>
                      <span class="cell cell-value"></span>
                    </div>
                    <div class="preview-table-line">
                      <span class="cell cell-title"></span>
                      <span class="cell cell-trend"></span>
                      <span class="cell cell-value"></span>
                    </div>
                    <div class="preview-table-line">
                      <span class="cell cell-title"></span>
                      <span class="cell cell-trend"></span>
                      <span class="cell cell-value"></span>
                    </div>
                  </div>
                </div>
              {:else}
                <div class="preview-kpi-grid">
                  <div class="kpi-pill">99.9%</div>
                  <div class="kpi-pill">+12%</div>
                  <div class="kpi-pill">1.2x</div>
                </div>
              {/if}
            </div>
            {#if canEdit}
              <button
                type="button"
                class="resize-width-handle"
                aria-label={t($locale, 'dashboardSettings.resizeWidth', { label: getChartLabel(setting.key) })}
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
      <h3>{t($locale, 'dashboardSettings.disabledWidgets')}</h3>
      <p class="controls-subtitle">{t($locale, 'dashboardSettings.disabledWidgetsSubtitle')}</p>

      {#if disabledSettings.length === 0}
        <div class="empty-state">{t($locale, 'dashboardSettings.noDisabledWidgets')}</div>
      {:else}
        {#each disabledSettings as setting (setting.key)}
          <button
            type="button"
            class="disabled-card"
            class:dragging={dragState?.source === 'disabled' && dragState?.key === setting.key}
            on:pointerdown={(event) => beginDrag(event, setting.key, 'disabled')}
          >
            <span class="disabled-drag">⋮⋮</span>
            <span>{getChartLabel(setting.key)}</span>
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
    align-items: center;
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

  .widget-title-wrap {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .info-tooltip {
    position: relative;
    display: inline-flex;
    align-items: center;
    flex: 0 0 auto;
  }

  .info-trigger {
    width: 18px;
    height: 18px;
    border-radius: 999px;
    border: 1px solid #c7d2fe;
    background: #eef2ff;
    color: #4f46e5;
    font-size: 11px;
    font-weight: 700;
    line-height: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    cursor: help;
  }

  .info-trigger:focus-visible {
    outline: 2px solid #6366f1;
    outline-offset: 2px;
  }

  .info-bubble {
    position: absolute;
    left: 50%;
    top: calc(100% + 8px);
    transform: translateX(-50%) translateY(-2px);
    min-width: 170px;
    max-width: 240px;
    padding: 6px 8px;
    border-radius: 8px;
    border: 1px solid #c7d2fe;
    background: #ffffff;
    color: #334155;
    font-size: 11px;
    line-height: 1.35;
    box-shadow: 0 8px 22px rgba(15, 23, 42, 0.16);
    opacity: 0;
    visibility: hidden;
    pointer-events: none;
    z-index: 20;
    transition: opacity 0.14s ease, transform 0.14s ease, visibility 0.14s ease;
  }

  .info-tooltip:hover .info-bubble,
  .info-tooltip:focus-within .info-bubble {
    opacity: 1;
    visibility: visible;
    transform: translateX(-50%) translateY(0);
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
    --preview-accent: #6366f1;
    --preview-accent-soft: #a5b4fc;
    --preview-accent-2: #7c3aed;
    --preview-bg: #f8fafc;
    --preview-bg-2: #f1f5f9;
    --preview-text: #4338ca;
    --preview-track: #dbe5f3;
    flex: 1;
    min-height: 0;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    justify-content: center;
    background: linear-gradient(180deg, var(--preview-bg) 0%, var(--preview-bg-2) 100%);
    border-top: 1px solid #eef2f7;
  }

  .preview-gauge-wrap {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  .preview-gauge {
    width: 64px;
    height: 64px;
    border-radius: 999px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: conic-gradient(var(--preview-accent) 0 72%, var(--preview-track) 72% 100%);
    position: relative;
  }

  .preview-gauge::after {
    content: '';
    width: 46px;
    height: 46px;
    border-radius: 999px;
    background: #ffffff;
    position: absolute;
  }

  .preview-gauge-value {
    position: relative;
    z-index: 1;
    font-size: 11px;
    font-weight: 700;
    color: var(--preview-text);
  }

  .preview-gauge-track {
    width: 80%;
    height: 8px;
    border-radius: 999px;
    background: linear-gradient(90deg, var(--preview-accent-2) 0%, var(--preview-accent) 58%, var(--preview-track) 58%);
  }

  .preview-cartesian,
  .preview-histogram {
    flex: 1;
  }

  .preview-cartesian-wrap {
    position: relative;
    flex: 1;
    min-height: 0;
  }

  .preview-cartesian {
    position: absolute;
    inset: 0;
    border: 1px solid #dbe2f0;
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.72);
    padding: 6px;
  }

  .preview-cartesian svg {
    width: 100%;
    height: 100%;
    display: block;
  }

  .preview-cartesian .axis {
    stroke: #cbd5e1;
    stroke-width: 1;
  }

  .preview-cartesian .series-a {
    fill: none;
    stroke: var(--preview-accent);
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .preview-cartesian .series-b {
    fill: none;
    stroke: var(--preview-accent-2);
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
    opacity: 0.9;
  }

  .preview-histogram {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    align-items: end;
    gap: 6px;
  }

  .preview-histogram span {
    display: block;
    width: 100%;
    border-radius: 4px 4px 2px 2px;
    background: linear-gradient(180deg, var(--preview-accent-soft) 0%, var(--preview-accent-2) 100%);
  }

  .preview-legend {
    position: absolute;
    right: 6px;
    bottom: 6px;
    display: flex;
    align-items: center;
    gap: 6px;
    justify-content: flex-end;
    padding: 3px 5px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.82);
    border: 1px solid #e2e8f0;
  }

  .dot {
    width: 7px;
    height: 7px;
    border-radius: 999px;
    display: inline-block;
  }

  .dot.blue {
    background: var(--preview-accent);
  }

  .dot.orange {
    background: var(--preview-accent-2);
  }

  .dot.red {
    background: var(--preview-track);
  }

  .preview-table {
    height: 100%;
    border: 1px solid #dbe2f0;
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.78);
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .preview-table-header {
    display: grid;
    grid-template-columns: 1.35fr 0.7fr 0.9fr;
    gap: 6px;
    align-items: center;
    padding: 6px 8px;
    border-bottom: 1px solid #e2e8f0;
    background: linear-gradient(90deg, rgba(99, 102, 241, 0.08) 0%, rgba(139, 92, 246, 0.1) 100%);
  }

  .preview-table-body {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 6px 8px;
    flex: 1;
  }

  .preview-table-line {
    display: grid;
    grid-template-columns: 1.35fr 0.7fr 0.9fr;
    gap: 6px;
    align-items: center;
    padding: 3px 0;
    border-bottom: 1px dashed #e2e8f0;
  }

  .preview-table-line:last-child {
    border-bottom: 0;
  }

  .preview-table .cell {
    display: block;
    height: 9px;
    border-radius: 999px;
    background: linear-gradient(90deg, rgba(99, 102, 241, 0.22) 0%, rgba(139, 92, 246, 0.36) 100%);
  }

  .preview-table .cell-title {
    width: 84%;
  }

  .preview-table .cell-trend {
    width: 62%;
  }

  .preview-table .cell-value {
    width: 74%;
    justify-self: end;
  }

  .preview-kpi-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 6px;
  }

  .kpi-pill {
    height: 30px;
    border-radius: 8px;
    background: linear-gradient(135deg, var(--preview-bg) 0%, var(--preview-bg-2) 100%);
    border: 1px solid var(--preview-track);
    color: var(--preview-text);
    font-size: 10px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
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
