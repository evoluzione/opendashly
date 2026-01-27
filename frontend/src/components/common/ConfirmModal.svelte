<script lang="ts">
  import { createEventDispatcher } from "svelte";

  export let open = false;
  export let title = "Conferma";
  export let message = "";
  export let confirmLabel = "Conferma";
  export let cancelLabel = "Annulla";

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
</script>

{#if open}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    aria-label={title}
    on:click|self={cancel}
    on:keydown={handleBackdropKeydown}
  >
    <div class="modal" role="dialog" aria-modal="true" on:click|stopPropagation>
      <header>
        <h3>{title}</h3>
      </header>
      <p>{message}</p>
      <div class="actions">
        <button type="button" class="secondary" on:click={cancel}>
          {cancelLabel}
        </button>
        <button type="button" class="primary" on:click={confirm}>
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(15, 23, 42, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    z-index: 80;
  }

  .modal {
    width: min(420px, 92vw);
    background: white;
    border-radius: 14px;
    padding: 20px;
    box-shadow: 0 20px 40px rgba(15, 23, 42, 0.2);
    display: grid;
    gap: 12px;
  }

  h3 {
    margin: 0;
    font-size: 16px;
    color: #0f172a;
  }

  p {
    margin: 0;
    font-size: 14px;
    color: #334155;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  button {
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    padding: 8px 14px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
  }

  .secondary {
    background: #f8fafc;
    color: #475569;
  }

  .primary {
    background: #1d4ed8;
    color: white;
    border-color: #1d4ed8;
  }
</style>
