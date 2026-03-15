<script lang="ts">
    import { onMount } from "svelte";
    import { servicesState, loadServices } from "../lib/stores/query";
    import {
        dashboardState,
        selectDashboardService,
    } from "../lib/stores/dashboard";
    import { locale, t } from "../lib/i18n";

    onMount(() => {
        void loadServices();
    });

    function handleChange(event: Event) {
        const value = (event.target as HTMLSelectElement).value;
        const serviceName = value === "" ? null : value;
        selectDashboardService(serviceName);
    }
</script>

<div class="dashboard-service-filter">
    <label for="dashboard-service-select">{t($locale, "home.service")}</label>
    <select
        id="dashboard-service-select"
        on:change={handleChange}
        value={$dashboardState.selectedService || ""}
    >
        <option value="">{t($locale, "home.allServices")}</option>
        {#each $servicesState.services as service}
            <option value={service}>{service}</option>
        {/each}
    </select>
</div>

<style>
    .dashboard-service-filter {
        display: flex;
        flex-direction: column;
        gap: 6px;
        min-width: 180px;
    }

    label {
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--color-slate-500);
    }

    select {
        padding: 10px 14px;
        border-radius: 10px;
        border: 1px solid var(--color-slate-200);
        background: white;
        font-size: 13px;
        font-weight: 500;
        color: var(--color-slate-950);
        cursor: pointer;
        transition: all 0.2s ease;
        appearance: none;
        background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%2364748b' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
        background-repeat: no-repeat;
        background-position: right 12px center;
        padding-right: 40px;
    }

    select:hover {
        border-color: var(--color-slate-300);
        background-color: var(--color-slate-50);
    }

    select:focus {
        outline: none;
        border-color: var(--color-info-600);
        box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
    }
</style>
