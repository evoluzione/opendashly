<script lang="ts">
  export let message: string;
</script>

<div class="loading-state">
  <span class="loading-spinner" aria-hidden="true"></span>
  <p class="loading-message">{message}</p>
</div>

<style>
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 18px;
    min-height: 200px;
    padding: 32px;
  }

  .loading-spinner {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    border: 3px solid var(--color-slate-100);
    border-top-color: var(--color-primary-600);
    border-right-color: var(--color-primary-500);
    animation: loading-spin 0.7s cubic-bezier(0.5, 0.1, 0.5, 0.9) infinite;
  }

  .loading-message {
    margin: 0;
    color: var(--color-slate-500);
    font-size: 14px;
    font-weight: 500;
    letter-spacing: 0.01em;
    animation: loading-pulse 1.6s ease-in-out infinite;
  }

  @keyframes loading-spin {
    to {
      transform: rotate(360deg);
    }
  }

  @keyframes loading-pulse {
    0%,
    100% {
      opacity: 0.55;
    }
    50% {
      opacity: 1;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .loading-spinner,
    .loading-message {
      animation: none;
    }
  }
</style>
