<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { fade, scale } from "svelte/transition";
  import { locale, t } from "../../lib/i18n";

  export let open = false;
  export let title = "";
  export let message = "";
  export let confirmLabel = "";
  export let cancelLabel = "";
  export let variant: "danger" | "warning" | "info" = "warning";

  const dispatch = createEventDispatcher();

  function confirm() {
    dispatch("confirm");
  }

  function cancel() {
    dispatch("cancel");
  }

  function handleBackdropKeydown(event: KeyboardEvent) {
    if (event.key === "Enter" || event.key === " " || event.key === "Escape") {
      event.preventDefault();
      cancel();
    }
  }

  const variantStyles = {
    danger: {
      gradient: "linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%)",
      iconBg: "rgba(255, 255, 255, 0.2)",
    },
    warning: {
      gradient: "linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%)",
      iconBg: "rgba(255, 255, 255, 0.2)",
    },
    info: {
      gradient: "linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%)",
      iconBg: "rgba(255, 255, 255, 0.2)",
    },
  };

  $: style = variantStyles[variant] || variantStyles.warning;
  $: resolvedTitle = title || t($locale, "assistant.resetConversationTitle");
  $: resolvedConfirmLabel = confirmLabel || t($locale, "common.apply");
  $: resolvedCancelLabel = cancelLabel || t($locale, "common.cancel");
</script>

{#if open}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    aria-label={resolvedTitle}
    on:click|self={cancel}
    on:keydown={handleBackdropKeydown}
    transition:fade={{ duration: 200 }}
  >
    <div
      class="modal"
      role="dialog"
      aria-modal="true"
      transition:scale={{ duration: 200, start: 0.95 }}
    >
      <header style="background: {style.gradient}">
        <div class="header-content">
          <div class="header-icon" style="background: {style.iconBg}">
            {#if variant === "danger"}
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
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="15" y1="9" x2="9" y2="15"></line>
                <line x1="9" y1="9" x2="15" y2="15"></line>
              </svg>
            {:else if variant === "warning"}
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
                <path
                  d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
                ></path>
                <line x1="12" y1="9" x2="12" y2="13"></line>
                <line x1="12" y1="17" x2="12.01" y2="17"></line>
              </svg>
            {:else}
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
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="16" x2="12" y2="12"></line>
                <line x1="12" y1="8" x2="12.01" y2="8"></line>
              </svg>
            {/if}
          </div>
          <h3>{resolvedTitle}</h3>
        </div>
        <button
          type="button"
          class="close-btn"
          on:click={cancel}
          aria-label={t($locale, "common.close")}
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
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </header>
      <div class="modal-content">
        <p>{message}</p>
      </div>
      <div class="actions">
        <button type="button" class="secondary" on:click={cancel}>
          {resolvedCancelLabel}
        </button>
        <button type="button" class="primary {variant}" on:click={confirm}>
          {resolvedConfirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(var(--rgb-slate-950), 0.7);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    z-index: 1300;
  }

  .modal {
    width: min(460px, 94vw);
    background: white;
    border-radius: 16px;
    box-shadow:
      0 25px 50px -12px rgba(0, 0, 0, 0.25),
      0 0 0 1px rgba(255, 255, 255, 0.1);
    overflow: hidden;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
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
    border-radius: 10px;
  }

  h3 {
    margin: 0;
    font-size: 16px;
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
    padding: 24px 20px;
    background: linear-gradient(180deg, var(--color-slate-50) 0%, var(--color-white) 100%);
  }

  p {
    margin: 0;
    font-size: 14px;
    color: var(--color-slate-700);
    line-height: 1.6;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding: 16px 20px;
    background: var(--color-slate-50);
    border-top: 1px solid var(--color-slate-200);
  }

  button {
    border-radius: 10px;
    padding: 10px 18px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .secondary {
    background: white;
    color: var(--color-slate-600);
    border: 1px solid var(--color-slate-200);
  }

  .secondary:hover {
    background: var(--color-slate-100);
    border-color: var(--color-slate-300);
  }

  .primary {
    color: white;
    border: none;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  }

  .primary.danger {
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
  }

  .primary.danger:hover {
    box-shadow: 0 6px 20px rgba(var(--rgb-primary-600), 0.4);
    transform: translateY(-2px);
  }

  .primary.warning {
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
  }

  .primary.warning:hover {
    box-shadow: 0 6px 20px rgba(var(--rgb-primary-600), 0.4);
    transform: translateY(-2px);
  }

  .primary.info {
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
  }

  .primary.info:hover {
    box-shadow: 0 6px 20px rgba(var(--rgb-primary-600), 0.4);
    transform: translateY(-2px);
  }
</style>
