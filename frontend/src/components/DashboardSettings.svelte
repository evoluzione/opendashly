<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import type { DashboardChartSetting } from '../services/dashboard_settings';
  import {
    dashboardSettingsState,
    loadDashboardSettings,
    saveDashboardSettings
  } from '../lib/stores/dashboard_settings';

  type ChartDefinition = {
    key: string;
    label: string;
    description: string;
  };

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
      key: 'status_codes',
      label: 'Status Code',
      description: 'Distribuzione OK/Error/Unset/Other.'
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
    },
    {
      key: 'log_volume',
      label: 'Log nel Tempo',
      description: 'Volume dei log nel tempo.'
    },
    {
      key: 'log_levels',
      label: 'Distribuzione Log',
      description: 'Ripartizione per livello di severita.'
    }
  ];

  let localSettings: DashboardChartSetting[] = [];
  let isDirty = false;
  let saveTimer: ReturnType<typeof setTimeout> | null = null;

  const defaultOrderMap = new Map<string, number>(
    chartCatalog.map((chart, index) => [chart.key, (index + 1) * 10])
  );

  function applySettings(settings: DashboardChartSetting[]) {
    const stored = new Map(settings.map((setting) => [setting.key, setting]));
    localSettings = chartCatalog.map((chart, index) => {
      const fallbackOrder = (index + 1) * 10;
      const existing = stored.get(chart.key);
      return {
        key: chart.key,
        enabled: existing?.enabled ?? true,
        order: existing?.order ?? fallbackOrder
      };
    });
  }

  function getLocalSetting(key: string): DashboardChartSetting {
    const existing = localSettings.find((setting) => setting.key === key);
    if (existing) return existing;
    return { key, enabled: true, order: defaultOrderMap.get(key) ?? 0 };
  }

  function toggleSetting(key: string) {
    const current = getLocalSetting(key);
    const updated = { ...current, enabled: !current.enabled };
    localSettings = localSettings.filter((setting) => setting.key !== key);
    localSettings = [...localSettings, updated];
    isDirty = true;
  }

  let draggingKey: string | null = null;
  let dragOverKey: string | null = null;
  let autoScrollInterval: ReturnType<typeof setInterval> | null = null;
  let lastDragY = 0;

  const SCROLL_ZONE = 80;
  const SCROLL_SPEED = 12;

  function startAutoScroll() {
    if (autoScrollInterval) return;
    autoScrollInterval = setInterval(() => {
      const viewportHeight = window.innerHeight;
      if (lastDragY < SCROLL_ZONE) {
        const intensity = 1 - lastDragY / SCROLL_ZONE;
        window.scrollBy(0, -SCROLL_SPEED * intensity);
      } else if (lastDragY > viewportHeight - SCROLL_ZONE) {
        const intensity = 1 - (viewportHeight - lastDragY) / SCROLL_ZONE;
        window.scrollBy(0, SCROLL_SPEED * intensity);
      }
    }, 16);
  }

  function stopAutoScroll() {
    if (autoScrollInterval) {
      clearInterval(autoScrollInterval);
      autoScrollInterval = null;
    }
  }

  function handleWindowDragOver(event: DragEvent) {
    lastDragY = event.clientY;
  }

  function reorderCharts(sourceKey: string, targetKey: string) {
    const orderedKeys = sortedCatalog.map((chart) => chart.key);
    const fromIndex = orderedKeys.indexOf(sourceKey);
    const toIndex = orderedKeys.indexOf(targetKey);
    if (fromIndex < 0 || toIndex < 0 || fromIndex === toIndex) return;

    orderedKeys.splice(fromIndex, 1);
    orderedKeys.splice(toIndex, 0, sourceKey);

    localSettings = orderedKeys.map((key, index) => {
      const current = getLocalSetting(key);
      return { ...current, order: (index + 1) * 10 };
    });
    isDirty = true;
  }

  function handleDragStart(event: DragEvent, key: string) {
    draggingKey = key;
    if (event.dataTransfer) {
      event.dataTransfer.setData('text/plain', key);
      event.dataTransfer.effectAllowed = 'move';
    }
    lastDragY = event.clientY;
    window.addEventListener('dragover', handleWindowDragOver);
    startAutoScroll();
  }

  function handleDragOver(event: DragEvent, key: string) {
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'move';
    }
    if (!draggingKey || draggingKey === key) {
      dragOverKey = null;
      return;
    }
    dragOverKey = key;
  }

  function handleDrop(event: DragEvent, key: string) {
    event.preventDefault();
    const sourceKey = event.dataTransfer?.getData('text/plain') || draggingKey;
    if (sourceKey) {
      reorderCharts(sourceKey, key);
    }
    draggingKey = null;
    dragOverKey = null;
  }

  function handleDragEnd() {
    draggingKey = null;
    dragOverKey = null;
    stopAutoScroll();
    window.removeEventListener('dragover', handleWindowDragOver);
  }

  function handleDragLeave(event: DragEvent) {
    const relatedTarget = event.relatedTarget as HTMLElement | null;
    if (!relatedTarget?.closest('.setting-card')) {
      dragOverKey = null;
    }
  }

  function scheduleAutoSave() {
    if (!isDirty) return;
    if (saveTimer) {
      clearTimeout(saveTimer);
    }
    saveTimer = setTimeout(async () => {
      await saveDashboardSettings(localSettings);
      isDirty = false;
      applySettings($dashboardSettingsState.settings ?? []);
    }, 400);
  }

  onMount(async () => {
    await loadDashboardSettings();
    applySettings($dashboardSettingsState.settings ?? []);
  });

  onDestroy(() => {
    stopAutoScroll();
    window.removeEventListener('dragover', handleWindowDragOver);
  });

  $: if (!isDirty && $dashboardSettingsState.settings && $dashboardSettingsState.settings.length > 0) {
    applySettings($dashboardSettingsState.settings);
  }

  $: if (isDirty) {
    scheduleAutoSave();
  }

  $: sortedCatalog = [...chartCatalog].sort((a, b) => {
    const left = localSettings.find((s) => s.key === a.key)?.order ?? defaultOrderMap.get(a.key) ?? 0;
    const right = localSettings.find((s) => s.key === b.key)?.order ?? defaultOrderMap.get(b.key) ?? 0;
    return left - right;
  });
</script>

<section class="dashboard-settings">
  <header>
    <h2>Visibilita grafici dashboard</h2>
    <p>Seleziona quali grafici mostrare nella dashboard per tutti gli utenti.</p>
  </header>

  {#if $dashboardSettingsState.error}
    <div class="status error">{$dashboardSettingsState.error}</div>
  {/if}

  <div class="settings-grid">
    {#each sortedCatalog as chart (chart.key)}
      <div
        class="setting-card"
        class:dragging={draggingKey === chart.key}
        class:drag-over={dragOverKey === chart.key && draggingKey !== chart.key}
        on:dragover={(event) => handleDragOver(event, chart.key)}
        on:dragleave={handleDragLeave}
        on:drop={(event) => handleDrop(event, chart.key)}
      >
        <span
          class="drag-handle"
          draggable="true"
          on:dragstart={(event) => handleDragStart(event, chart.key)}
          on:dragend={handleDragEnd}
          title="Trascina per riordinare"
        >
          ⋮⋮
        </span>
        <div class="setting-info">
          <h3>{chart.label}</h3>
          <p>{chart.description}</p>
        </div>
        <div class="setting-actions">
          <label class="toggle">
            <input
              type="checkbox"
              checked={getLocalSetting(chart.key).enabled}
              on:change={() => toggleSetting(chart.key)}
            />
            <span class="slider"></span>
          </label>
        </div>
      </div>
    {/each}
  </div>

</section>

<style>
  .dashboard-settings {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  header h2 {
    font-size: 20px;
    margin-bottom: 6px;
  }

  header p {
    color: #64748b;
    margin: 0;
  }

  .status {
    padding: 10px 12px;
    border-radius: 10px;
    background: #f8fafc;
    color: #475569;
  }

  .status.error {
    background: #fee2e2;
    color: #b91c1c;
  }

  .settings-grid {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .settings-grid .setting-card {
    transition: transform 0.15s ease, box-shadow 0.15s ease, border-color 0.15s ease;
  }

  .setting-card {
    background: white;
    border-radius: 14px;
    padding: 16px;
    border: 1px solid rgba(15, 23, 42, 0.08);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .setting-info {
    flex: 1;
    min-width: 0;
  }

  .setting-card h3 {
    font-size: 14px;
    margin: 0 0 6px;
  }

  .setting-card p {
    font-size: 12px;
    color: #64748b;
    margin: 0;
  }

  .setting-actions {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .drag-handle {
    width: 28px;
    height: 28px;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    color: #0f172a;
    font-weight: 700;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    cursor: grab;
    user-select: none;
  }

  .drag-handle:active {
    cursor: grabbing;
  }

  .setting-card.dragging {
    opacity: 0.5;
    border-style: dashed;
    border-color: #94a3b8;
    transform: scale(0.98);
  }

  .setting-card.drag-over {
    border-color: #2563eb;
    border-width: 2px;
    box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.15);
    transform: translateY(-2px);
  }

  .toggle {
    position: relative;
    width: 44px;
    height: 24px;
    display: inline-block;
    flex: 0 0 44px;
    box-sizing: border-box;
  }

  .toggle input {
    opacity: 0;
    width: 0;
    height: 0;
  }

  .slider {
    position: absolute;
    cursor: pointer;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: #e2e8f0;
    border-radius: 999px;
    transition: 0.2s;
    box-sizing: border-box;
    overflow: hidden;
  }

  .slider:before {
    position: absolute;
    content: '';
    height: 18px;
    width: 18px;
    left: 3px;
    top: 3px;
    background-color: white;
    border-radius: 50%;
    transition: 0.2s;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.2);
  }

  .toggle input:checked + .slider {
    background-color: #2563eb;
  }

  .toggle input:checked + .slider:before {
    transform: translateX(20px);
  }

</style>
