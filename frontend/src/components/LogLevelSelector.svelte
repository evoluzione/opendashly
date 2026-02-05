<script lang="ts">
    import { servicesState, selectLogLevel } from "../lib/stores/query";

    let isOpen = false;

    const levels = [
        {
            value: "Tutti",
            label: "Tutti i livelli",
            color: "#0f172a",
            bg: "#f8fafc",
        },
        { value: "TRACE", label: "TRACE", color: "#64748b", bg: "#f1f5f9" },
        { value: "DEBUG", label: "DEBUG", color: "#a855f7", bg: "#f3e8ff" },
        { value: "INFO", label: "INFO", color: "#3b82f6", bg: "#dbeafe" },
        { value: "WARN", label: "WARN", color: "#f59e0b", bg: "#fef3c7" },
        { value: "ERROR", label: "ERROR", color: "#ef4444", bg: "#fee2e2" },
        { value: "FATAL", label: "FATAL", color: "#dc2626", bg: "#fef2f2" },
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
    <span class="label-text">Livello Log</span>
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
            {selectedLevel.label}
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
                        {level.label}
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
        color: #64748b;
    }

    .trigger {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 10px 14px;
        border-radius: 10px;
        border: 1px solid #e2e8f0;
        background: white;
        cursor: pointer;
        transition: all 0.2s ease;
        min-width: 160px;
    }

    .trigger:hover {
        border-color: #cbd5e1;
    }

    .trigger.open {
        border-color: #6366f1;
        box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
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
        color: #64748b;
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
        border: 1px solid #e2e8f0;
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
        background: #f8fafc;
    }

    .option.selected {
        background: #f1f5f9;
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
        color: #6366f1;
        flex-shrink: 0;
    }
</style>
