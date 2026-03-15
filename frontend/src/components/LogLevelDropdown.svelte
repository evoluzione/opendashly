<script lang="ts">
    import { servicesState, selectLogLevel } from "../lib/stores/query";
    import { locale, t } from "../lib/i18n";

    const levels = [
        "Tutti",
        "TRACE",
        "DEBUG",
        "INFO",
        "WARN",
        "ERROR",
        "FATAL",
    ];

    function getLevelColor(level: string): string {
        switch (level) {
            case "FATAL":
                return "#ef4444"; // Red 500
            case "ERROR":
                return "#ef4444"; // Red 500
            case "WARN":
                return "#f59e0b"; // Amber 500
            case "INFO":
                return "#3b82f6"; // Blue 500
            case "DEBUG":
                return "#a855f7"; // Purple 500
            case "TRACE":
                return "#64748b"; // Slate 500
            default:
                return "#0f172a"; // Slate 900
        }
    }

    function getLevelBg(level: string): string {
        switch (level) {
            case "FATAL":
                return "#fee2e2"; // Red 100
            case "ERROR":
                return "#fee2e2"; // Red 100
            case "WARN":
                return "#fef3c7"; // Amber 100
            case "INFO":
                return "#dbeafe"; // Blue 100
            case "DEBUG":
                return "#f3e8ff"; // Purple 100
            case "TRACE":
                return "#f1f5f9"; // Slate 100
            default:
                return "white";
        }
    }

    function handleChange(event: Event) {
        const value = (event.target as HTMLSelectElement).value;
        selectLogLevel(value);
    }
</script>

<div class="loglevel-dropdown">
    <label for="level-select">{t($locale, "logLevel.label")}</label>
    <div class="select-wrapper">
        <select id="level-select" on:change={handleChange}>
            {#each levels as level}
                <option
                    value={level}
                    selected={level === $servicesState.selectedLogLevel}
                    style="color: {getLevelColor(level)}">{level}</option
                >
            {/each}
        </select>
        <!-- Overlay for coloring the selected value (simple trick since select styling is limited) -->
        <div
            class="selected-value"
            style="background-color: {getLevelBg(
                $servicesState.selectedLogLevel,
            )}; color: {getLevelColor($servicesState.selectedLogLevel)}"
        >
            {$servicesState.selectedLogLevel}
        </div>
        <svg
            class="chevron"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
        >
            <polyline points="6 9 12 15 18 9"></polyline>
        </svg>
    </div>
</div>

<style>
    .loglevel-dropdown {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    label {
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: #64748b;
    }

    .select-wrapper {
        position: relative;
        width: 100%;
        height: 42px; /* Match height of other inputs */
    }

    select {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        opacity: 0; /* Hide the real select but keep it interactive */
        z-index: 2;
        cursor: pointer;
    }

    .selected-value {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        padding: 0 16px;
        border-radius: 10px;
        border: 1px solid #e2e8f0;
        font-size: 14px;
        font-weight: 600;
        pointer-events: none;
        z-index: 1;
        transition: all 0.2s ease;
    }

    .chevron {
        position: absolute;
        right: 12px;
        top: 50%;
        transform: translateY(-50%);
        color: #64748b;
        pointer-events: none;
        z-index: 1;
    }

    .select-wrapper:hover .selected-value {
        border-color: #cbd5e1;
    }
</style>
