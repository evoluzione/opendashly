<script lang="ts">
    import { onMount } from "svelte";
    import { servicesState, loadServices } from "../lib/stores/query";
    import {
        dashboardState,
        selectDashboardService,
    } from "../lib/stores/dashboard";

    onMount(() => {
        void loadServices();
    });

    function handleChange(event: Event) {
        const value = (event.target as HTMLSelectElement).value;
        const serviceName = value === "Tutti" ? null : value;
        selectDashboardService(serviceName);
    }
</script>

<div class="dashboard-service-filter">
    <label for="dashboard-service-select">Servizio</label>
    <select
        id="dashboard-service-select"
        on:change={handleChange}
        value={$dashboardState.selectedService || "Tutti"}
    >
        <option value="Tutti">Tutti i servizi</option>
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
        color: #64748b;
    }

    select {
        padding: 10px 14px;
        border-radius: 10px;
        border: 1px solid #e2e8f0;
        background: white;
        font-size: 13px;
        font-weight: 500;
        color: #0f172a;
        cursor: pointer;
        transition: all 0.2s ease;
        appearance: none;
        background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%2364748b' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
        background-repeat: no-repeat;
        background-position: right 12px center;
        padding-right: 40px;
    }

    select:hover {
        border-color: #cbd5e1;
        background-color: #f8fafc;
    }

    select:focus {
        outline: none;
        border-color: #2563eb;
        box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
    }
</style>
