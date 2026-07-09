<script lang="ts">
    import { servicesState, selectLogLevel } from "../lib/stores/query";
    import { locale, t } from "../lib/i18n";

    let isOpen = false;

    const levels = [
        {
            value: "",
            label: "",
            color: "var(--color-slate-950)",
            bg: "var(--color-slate-50)",
        },
        { value: "TRACE", label: "TRACE", color: "var(--color-slate-500)", bg: "var(--color-slate-100)" },
        { value: "DEBUG", label: "DEBUG", color: "var(--color-primary-400)", bg: "var(--color-primary-25)" },
        { value: "INFO", label: "INFO", color: "var(--color-info-500)", bg: "var(--color-info-100)" },
        { value: "WARN", label: "WARN", color: "var(--color-warning-500)", bg: "var(--color-warning-100)" },
        { value: "ERROR", label: "ERROR", color: "var(--color-danger-500)", bg: "var(--color-danger-100)" },
        { value: "FATAL", label: "FATAL", color: "var(--color-danger-600)", bg: "var(--color-danger-50)" },
    ];

    function getLevel(value: string) {
        return levels.find((l) => l.value === value) ?? levels[0];
    }

    function handleSelect(value: string) {
        selectLogLevel(value);
        isOpen = false;
    }

    function handleClickOutside(event: MouseEvent) {
        const target = event.target as HTMLElement;
        if (!target.closest(".level-selector")) {
            isOpen = false;
        }
    }

    function handleKeydown(event: KeyboardEvent) {
        if (event.key === "Escape") {
            isOpen = false;
        }
    }

    $: selectedLevel = getLevel($servicesState.selectedLogLevel);
</script>

<svelte:window on:click={handleClickOutside} on:keydown={handleKeydown} />

<div class="level-selector">
    <span class="label-text">{t($locale, "logLevel.label")}</span>
    <button
        type="button"
        class="trigger"
        class:open={isOpen}
        on:click={() => (isOpen = !isOpen)}
        style="--level-color: {selectedLevel.color}; --level-bg: {selectedLevel.bg}"
        aria-haspopup="listbox"
        aria-expanded={isOpen}
    >
        <span
            class="badge"
            style="background: {selectedLevel.bg}; color: {selectedLevel.color}"
        >
            {selectedLevel.label || t($locale, "query.allLevels")}
        </span>
        <svg
            class="chevron"
            class:rotated={isOpen}
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
        >
            <polyline points="6 9 12 15 18 9"></polyline>
        </svg>
    </button>

    {#if isOpen}
        <div class="dropdown">
            {#each levels as level}
                <button
                    type="button"
                    class="option"
                    class:selected={level.value ===
                        $servicesState.selectedLogLevel}
                    on:click={() => handleSelect(level.value)}
                >
                    <span
                        class="option-badge"
                        style="background: {level.bg}; color: {level.color}"
                    >
                        {level.label || t($locale, "query.allLevels")}
                    </span>
                    {#if level.value === $servicesState.selectedLogLevel}
                        <svg
                            width="16"
                            height="16"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                        >
                            <polyline points="20 6 9 17 4 12"></polyline>
                        </svg>
                    {/if}
                </button>
            {/each}
        </div>
    {/if}
</div>

<style>
    .level-selector {
        position: relative;
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .label-text {
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--color-slate-500);
    }

    .trigger {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 10px 14px;
        border-radius: 10px;
        border: 1px solid var(--color-slate-200);
        background: white;
        cursor: pointer;
        transition: all 0.2s ease;
        min-width: 160px;
    }

    .trigger:hover {
        border-color: var(--color-slate-300);
    }

    .trigger.open {
        border-color: var(--color-primary-600);
        box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.1);
    }

    .badge {
        display: inline-flex;
        align-items: center;
        padding: 4px 10px;
        border-radius: 6px;
        font-size: 12px;
        font-weight: 600;
    }

    .chevron {
        color: var(--color-slate-500);
        transition: transform 0.2s ease;
        flex-shrink: 0;
    }

    .chevron.rotated {
        transform: rotate(180deg);
    }

    .dropdown {
        position: absolute;
        top: calc(100% + 6px);
        left: 0;
        right: 0;
        background: white;
        border: 1px solid var(--color-slate-200);
        border-radius: 12px;
        box-shadow:
            0 10px 40px rgba(0, 0, 0, 0.12),
            0 2px 6px rgba(0, 0, 0, 0.04);
        z-index: 100;
        overflow: hidden;
        animation: dropdownIn 0.15s ease-out;
    }

    @keyframes dropdownIn {
        from {
            opacity: 0;
            transform: translateY(-8px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .option {
        display: flex;
        align-items: center;
        justify-content: space-between;
        width: 100%;
        padding: 10px 14px;
        border: none;
        background: transparent;
        cursor: pointer;
        transition: background 0.15s ease;
        text-align: left;
    }

    .option:hover {
        background: var(--color-slate-50);
    }

    .option.selected {
        background: var(--color-slate-100);
    }

    .option-badge {
        display: inline-flex;
        align-items: center;
        padding: 4px 10px;
        border-radius: 6px;
        font-size: 12px;
        font-weight: 600;
    }

    .option svg {
        color: var(--color-primary-600);
        flex-shrink: 0;
    }
</style>
