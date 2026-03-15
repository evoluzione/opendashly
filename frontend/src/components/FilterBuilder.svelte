<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import Autocomplete from "./common/Autocomplete.svelte";
    import { getLogAttributes } from "../services/query";
    import type { FilterItem } from "../services/query";
    import { locale, t } from "../lib/i18n";

    export let filters: FilterItem[] = [];

    const dispatch = createEventDispatcher();
    let attributeWarnings: boolean[] = [];

    $: if (attributeWarnings.length !== filters.length) {
        attributeWarnings = filters.map((_, index) => attributeWarnings[index] ?? false);
    }

    function addFilter() {
        filters = [
            ...filters,
            { connector: "AND", key: "", operator: "=", value: "" },
        ];
        attributeWarnings = [...attributeWarnings, false];
        dispatch("change", filters);
    }

    function removeFilter(index: number) {
        filters = filters.filter((_, i) => i !== index);
        attributeWarnings = attributeWarnings.filter((_, i) => i !== index);
        dispatch("change", filters);
    }

    function handleChange() {
        dispatch("change", filters);
    }

    function clearAttributeWarning(index: number) {
        attributeWarnings[index] = false;
        attributeWarnings = [...attributeWarnings];
    }

    function handleAttributeResolved(
        index: number,
        event: CustomEvent<{ query: string; options: string[] }> | Event
    ) {
        const detail = (event as CustomEvent<{ query: string; options: string[] }>).detail;
        const query = String(detail?.query ?? "").trim();
        if (!query) {
            clearAttributeWarning(index);
            return;
        }
        const options = Array.isArray(detail?.options)
            ? detail.options
            : [];
        const exists = options.some(
            (option) => option.toLowerCase() === query.toLowerCase()
        );
        attributeWarnings[index] = !exists;
        attributeWarnings = [...attributeWarnings];
    }

    $: operators = [
        { label: t($locale, "filterBuilder.operatorEqual"), value: "=", icon: "=" },
        { label: t($locale, "filterBuilder.operatorNotEqual"), value: "!=", icon: "≠" },
        { label: t($locale, "filterBuilder.operatorContains"), value: "contains", icon: "∋" },
    ];
</script>

<div class="filter-builder">
    {#if filters.length === 0}
        <div class="empty-state">
            <div class="empty-icon">
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="48"
                    height="48"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                >
                    <polygon
                        points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"
                    ></polygon>
                </svg>
            </div>
            <h4>{t($locale, "filterBuilder.emptyTitle")}</h4>
            <p>
                {t($locale, "filterBuilder.emptyDescription")}
            </p>
        </div>
    {:else}
        <div class="filter-list">
            {#each filters as filter, i}
                <div class="filter-row" class:first-row={i === 0}>
                    <div class="filter-card">
                        {#if i > 0}
                            <div class="connector">
                                <select
                                    bind:value={filter.connector}
                                    on:change={handleChange}
                                    class:and={filter.connector === "AND"}
                                    class:or={filter.connector === "OR"}
                                >
                                    <option value="AND">AND</option>
                                    <option value="OR">OR</option>
                                </select>
                            </div>
                        {:else}
                            <div class="connector-placeholder">
                                <span class="where-badge">{t($locale, "filterBuilder.where")}</span>
                            </div>
                        {/if}

                        <div class="filter-fields">
                            <div class="field-key">
                                <span class="field-label">{t($locale, "filterBuilder.attribute")}</span>
                                <div class="attribute-input-wrap">
                                    <Autocomplete
                                        bind:value={filter.key}
                                        placeholder={t($locale, "filterBuilder.attributePlaceholder")}
                                        fetchOptions={getLogAttributes}
                                        on:select={() => {
                                            clearAttributeWarning(i);
                                            handleChange();
                                        }}
                                        on:input={() => {
                                            clearAttributeWarning(i);
                                            handleChange();
                                        }}
                                        on:resolved={(event) =>
                                            handleAttributeResolved(i, event)}
                                    />
                                    {#if attributeWarnings[i]}
                                        <span
                                            class="attribute-warning"
                                            title={t($locale, "filterBuilder.attributeMissingTitle")}
                                            aria-label={t($locale, "filterBuilder.attributeMissingAria")}
                                        >
                                            <svg
                                                xmlns="http://www.w3.org/2000/svg"
                                                width="14"
                                                height="14"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                stroke-width="2"
                                                stroke-linecap="round"
                                                stroke-linejoin="round"
                                            >
                                                <path
                                                    d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
                                                ></path>
                                                <line x1="12" y1="9" x2="12" y2="13"></line>
                                                <line x1="12" y1="17" x2="12.01" y2="17"></line>
                                            </svg>
                                        </span>
                                    {/if}
                                </div>
                            </div>

                            <div class="operator">
                                <label for="operator-{i}">{t($locale, "filterBuilder.operator")}</label>
                                <select
                                    id="operator-{i}"
                                    bind:value={filter.operator}
                                    on:change={handleChange}
                                >
                                    {#each operators as op}
                                        <option value={op.value}
                                            >{op.icon} {op.label}</option
                                        >
                                    {/each}
                                </select>
                            </div>

                            <div class="field-value">
                                <label for="value-{i}">{t($locale, "filterBuilder.value")}</label>
                                <input
                                    id="value-{i}"
                                    type="text"
                                    bind:value={filter.value}
                                    placeholder={t($locale, "filterBuilder.valuePlaceholder")}
                                    on:input={handleChange}
                                />
                            </div>
                        </div>

                        <button
                            class="remove-btn"
                            on:click={() => removeFilter(i)}
                            title={t($locale, "filterBuilder.removeFilter")}
                            aria-label={t($locale, "filterBuilder.removeFilter")}
                        >
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                width="18"
                                height="18"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                            >
                                <circle cx="12" cy="12" r="10"></circle>
                                <line x1="15" y1="9" x2="9" y2="15"></line>
                                <line x1="9" y1="9" x2="15" y2="15"></line>
                            </svg>
                        </button>
                    </div>
                </div>
            {/each}
        </div>
    {/if}

    <button class="add-btn" on:click={addFilter}>
        <span class="add-icon">
            <svg
                xmlns="http://www.w3.org/2000/svg"
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
            >
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="8" x2="12" y2="16"></line>
                <line x1="8" y1="12" x2="16" y2="12"></line>
            </svg>
        </span>
        {t($locale, "filterBuilder.addFilter")}
    </button>
</div>

<style>
    .filter-builder {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    /* Empty State */
    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 40px 24px;
        text-align: center;
        background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
        border-radius: 16px;
        border: 2px dashed #cbd5e1;
    }

    .empty-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 80px;
        height: 80px;
        background: linear-gradient(135deg, #e0e7ff 0%, #c7d2fe 100%);
        border-radius: 50%;
        margin-bottom: 16px;
        color: #6366f1;
    }

    .empty-state h4 {
        margin: 0 0 8px 0;
        font-size: 16px;
        font-weight: 600;
        color: #1e293b;
    }

    .empty-state p {
        margin: 0;
        font-size: 14px;
        color: #64748b;
        max-width: 280px;
    }

    /* Filter List */
    .filter-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .filter-row {
        position: relative;
    }

    .filter-row:not(.first-row)::before {
        content: "";
        position: absolute;
        left: 35px;
        top: -6px;
        width: 2px;
        height: 6px;
        background: linear-gradient(to bottom, #6366f1, #8b5cf6);
        border-radius: 1px;
    }

    .filter-card {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        padding: 16px;
        background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
        border-radius: 12px;
        border: 1px solid #e2e8f0;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
        transition: all 0.2s ease;
    }

    .filter-card:hover {
        border-color: #c7d2fe;
        box-shadow: 0 4px 12px rgba(99, 102, 241, 0.1);
    }

    /* Connector */
    .connector,
    .connector-placeholder {
        width: 58px;
        flex-shrink: 0;
        padding-top: 22px;
    }

    .connector select {
        width: 100%;
        padding: 8px 6px;
        border-radius: 8px;
        font-size: 11px;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        border: none;
        cursor: pointer;
        transition: all 0.2s ease;
        text-align: center;
    }

    .connector select.and {
        background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
        color: #1d4ed8;
    }

    .connector select.or {
        background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
        color: #b45309;
    }

    .where-badge {
        display: inline-block;
        padding: 8px 8px;
        background: linear-gradient(135deg, #e0e7ff 0%, #c7d2fe 100%);
        color: #4338ca;
        font-size: 10px;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        border-radius: 8px;
    }

    /* Filter Fields */
    .filter-fields {
        flex: 1;
        display: grid;
        grid-template-columns: 1fr 140px 1fr;
        gap: 12px;
        min-width: 0;
    }

    .filter-fields label,
    .field-label {
        display: block;
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: #64748b;
        margin-bottom: 6px;
    }

    .field-key,
    .field-value {
        min-width: 0;
    }

    .attribute-input-wrap {
        position: relative;
    }

    .attribute-input-wrap :global(input) {
        padding-right: 34px;
    }

    .attribute-warning {
        position: absolute;
        top: 50%;
        right: 10px;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        color: #f59e0b;
        transform: translateY(-50%);
        cursor: help;
    }

    .operator select {
        width: 100%;
        padding: 10px 12px;
        border-radius: 8px;
        border: 1px solid #e2e8f0;
        font-size: 13px;
        font-weight: 500;
        background: white;
        color: #334155;
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .operator select:hover {
        border-color: #6366f1;
    }

    .operator select:focus {
        outline: none;
        border-color: #6366f1;
        box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
    }

    .field-value input {
        width: 100%;
        padding: 10px 12px;
        border: 1px solid #e2e8f0;
        border-radius: 8px;
        font-size: 13px;
        background: white;
        color: #0f172a;
        transition: all 0.2s ease;
    }

    .field-value input::placeholder {
        color: #94a3b8;
    }

    .field-value input:hover {
        border-color: #6366f1;
    }

    .field-value input:focus {
        outline: none;
        border-color: #6366f1;
        box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
    }

    /* Remove Button */
    .remove-btn {
        padding: 8px;
        color: #94a3b8;
        background: transparent;
        border: none;
        cursor: pointer;
        border-radius: 8px;
        transition: all 0.2s ease;
        margin-top: 20px;
    }

    .remove-btn:hover {
        color: #ef4444;
        background: #fef2f2;
        transform: scale(1.1);
    }

    /* Add Button */
    .add-btn {
        align-self: center;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        padding: 12px 24px;
        font-size: 14px;
        font-weight: 600;
        color: #6366f1;
        background: linear-gradient(135deg, #eef2ff 0%, #e0e7ff 100%);
        border: 2px dashed #a5b4fc;
        border-radius: 12px;
        cursor: pointer;
        transition: all 0.3s ease;
        width: 100%;
        max-width: 300px;
    }

    .add-btn:hover {
        background: linear-gradient(135deg, #e0e7ff 0%, #c7d2fe 100%);
        border-color: #6366f1;
        color: #4f46e5;
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(99, 102, 241, 0.25);
    }

    .add-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        transition: transform 0.3s ease;
    }

    .add-btn:hover .add-icon {
        transform: rotate(90deg);
    }

    /* Responsive */
    @media (max-width: 700px) {
        .filter-fields {
            grid-template-columns: 1fr;
        }

        .filter-card {
            flex-direction: column;
            align-items: stretch;
        }

        .connector,
        .connector-placeholder {
            width: 100%;
            padding-top: 0;
            margin-bottom: 8px;
        }

        .remove-btn {
            align-self: flex-end;
            margin-top: 8px;
        }
    }
</style>
