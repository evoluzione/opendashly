<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { onMount, tick } from "svelte";

    export let value = "";
    export let placeholder = "";
    export let fetchOptions: (search: string) => Promise<string[]>;

    const dispatch = createEventDispatcher();

    let options: string[] = [];
    let showOptions = false;
    let loading = false;
    let focusedIndex = -1;
    let inputRef: HTMLInputElement;
    let debounceTimer: ReturnType<typeof setTimeout>;

    async function handleInput() {
        dispatch("input", value);
        showOptions = true;
        focusedIndex = -1;

        if (debounceTimer) clearTimeout(debounceTimer);
        debounceTimer = setTimeout(async () => {
            loading = true;
            try {
                options = await fetchOptions(value);
            } catch (e) {
                console.error("Failed to fetch options", e);
            } finally {
                loading = false;
            }
        }, 300);
    }

    function selectOption(option: string) {
        value = option;
        showOptions = false;
        dispatch("select", option);
        dispatch("input", value);
    }

    function handleKeydown(event: KeyboardEvent) {
        if (!showOptions) {
            if (event.key === "ArrowDown") {
                showOptions = true;
                handleInput();
            }
            return;
        }

        if (event.key === "ArrowDown") {
            focusedIndex = (focusedIndex + 1) % options.length;
            event.preventDefault();
        } else if (event.key === "ArrowUp") {
            focusedIndex = (focusedIndex - 1 + options.length) % options.length;
            event.preventDefault();
        } else if (event.key === "Enter") {
            if (focusedIndex >= 0 && options[focusedIndex]) {
                selectOption(options[focusedIndex]);
                event.preventDefault();
            } else {
                showOptions = false;
            }
        } else if (event.key === "Escape") {
            showOptions = false;
        }
    }

    function blur(event: FocusEvent) {
        // Delay hiding to allow click to register
        setTimeout(() => {
            showOptions = false;
        }, 200);
    }
</script>

<div class="autocomplete">
    <input
        bind:this={inputRef}
        type="text"
        bind:value
        {placeholder}
        on:input={handleInput}
        on:focus={() => {
            showOptions = true;
            if (!options.length) handleInput();
        }}
        on:blur={blur}
        on:keydown={handleKeydown}
    />

    {#if showOptions && (options.length > 0 || loading)}
        <ul class="options">
            {#if loading}
                <li class="loading">Caricamento...</li>
            {:else}
                {#each options as option, i}
                    <li
                        class:active={i === focusedIndex}
                        on:click={() => selectOption(option)}
                        on:mouseenter={() => (focusedIndex = i)}
                    >
                        {option}
                    </li>
                {/each}
            {/if}
        </ul>
    {/if}
</div>

<style>
    .autocomplete {
        position: relative;
        width: 100%;
    }

    input {
        width: 100%;
        padding: 8px 12px;
        border: 1px solid #e2e8f0;
        border-radius: 6px;
        font-size: 13px;
        color: #0f172a;
        background: #ffffff;
        transition: all 0.2s;
    }

    input:focus {
        outline: none;
        border-color: #3b82f6;
        box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
    }

    .options {
        position: absolute;
        top: 100%;
        left: 0;
        right: 0;
        margin-top: 4px;
        background: white;
        border: 1px solid #e2e8f0;
        border-radius: 6px;
        box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        z-index: 50;
        max-height: 200px;
        overflow-y: auto;
        list-style: none;
        padding: 0;
    }

    li {
        padding: 8px 12px;
        font-size: 13px;
        cursor: pointer;
        color: #334155;
    }

    li:hover,
    li.active {
        background: #f1f5f9;
        color: #0f172a;
    }

    .loading {
        color: #94a3b8;
        cursor: default;
    }
</style>
