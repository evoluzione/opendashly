<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { fade, scale } from "svelte/transition";

    export let open = false;
    export let title = "";

    const dispatch = createEventDispatcher();

    function close() {
        dispatch("close");
    }

    function handleKeydown(event: KeyboardEvent) {
        if (event.key === "Escape") {
            close();
        }
    }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
    <div
        class="modal-backdrop"
        role="button"
        tabindex="0"
        on:click|self={close}
        on:keydown={(e) => e.key === "Enter" && close()}
        transition:fade={{ duration: 200 }}
    >
        <div
            class="modal-container"
            role="dialog"
            aria-modal="true"
            aria-label={title}
            transition:scale={{ duration: 200, start: 0.95 }}
        >
            <header class="modal-header">
                <div class="header-content">
                    <div class="header-icon">
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            width="20"
                            height="20"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <polygon
                                points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"
                            ></polygon>
                        </svg>
                    </div>
                    <h3>{title}</h3>
                </div>
                <button class="close-btn" on:click={close} aria-label="Chiudi">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="20"
                        height="20"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        ><line x1="18" y1="6" x2="6" y2="18"></line><line
                            x1="6"
                            y1="6"
                            x2="18"
                            y2="18"
                        ></line></svg
                    >
                </button>
            </header>
            <div class="modal-content">
                <slot />
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(15, 23, 42, 0.7);
        backdrop-filter: blur(8px);
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 16px;
        z-index: 100;
    }

    .modal-container {
        background: white;
        border-radius: 16px;
        box-shadow:
            0 25px 50px -12px rgba(0, 0, 0, 0.25),
            0 0 0 1px rgba(255, 255, 255, 0.1);
        width: 100%;
        max-width: 700px;
        max-height: 90vh;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .modal-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 20px 24px;
        background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
        color: white;
    }

    .header-content {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .header-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 36px;
        height: 36px;
        background: rgba(255, 255, 255, 0.2);
        border-radius: 10px;
    }

    h3 {
        margin: 0;
        font-size: 18px;
        font-weight: 600;
        color: white;
    }

    .close-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(255, 255, 255, 0.15);
        border: none;
        color: white;
        cursor: pointer;
        padding: 8px;
        border-radius: 8px;
        transition: all 0.2s ease;
    }

    .close-btn:hover {
        background: rgba(255, 255, 255, 0.25);
        transform: scale(1.05);
    }

    .modal-content {
        padding: 24px;
        overflow-y: auto;
        background: linear-gradient(180deg, #f8fafc 0%, #ffffff 100%);
    }
</style>
