<script lang="ts">
  export let text: string;
  export let position: 'top' | 'bottom' = 'top';
  export let align: 'left' | 'center' | 'right' = 'center';
</script>

<span class="info-tooltip">
  <svg class="info-icon" viewBox="0 0 20 20" fill="currentColor">
    <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z" clip-rule="evenodd" />
  </svg>
  <span 
    class="tooltip-text" 
    class:bottom={position === 'bottom'}
    class:align-left={align === 'left'}
    class:align-right={align === 'right'}
  >{text}</span>
</span>

<style>
  .info-tooltip {
    position: relative;
    display: inline-flex;
    align-items: center;
    cursor: help;
    margin-left: 6px;
  }

  .info-icon {
    width: 14px;
    height: 14px;
    color: #94a3b8;
    transition: color 0.15s ease;
  }

  .info-tooltip:hover .info-icon {
    color: #64748b;
  }

  .tooltip-text {
    visibility: hidden;
    opacity: 0;
    position: absolute;
    z-index: 9999;
    bottom: calc(100% + 8px);
    left: 50%;
    transform: translateX(-50%);
    background: #0f172a;
    color: white;
    padding: 10px 14px;
    border-radius: 8px;
    font-size: 12px;
    font-weight: 400;
    line-height: 1.5;
    white-space: normal;
    width: max-content;
    max-width: 280px;
    text-transform: none;
    letter-spacing: normal;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    transition: opacity 0.2s ease, visibility 0.2s ease;
    pointer-events: none;
  }

  .tooltip-text.bottom {
    bottom: auto;
    top: calc(100% + 8px);
  }

  .tooltip-text.align-left {
    left: 0;
    transform: translateX(0);
  }

  .tooltip-text.align-right {
    left: auto;
    right: 0;
    transform: translateX(0);
  }

  .tooltip-text::after {
    content: '';
    position: absolute;
    top: 100%;
    left: 50%;
    transform: translateX(-50%);
    border: 6px solid transparent;
    border-top-color: #0f172a;
  }

  .tooltip-text.bottom::after {
    top: auto;
    bottom: 100%;
    border-top-color: transparent;
    border-bottom-color: #0f172a;
  }

  .tooltip-text.align-left::after {
    left: 16px;
  }

  .tooltip-text.align-right::after {
    left: auto;
    right: 16px;
  }

  .info-tooltip:hover .tooltip-text {
    visibility: visible;
    opacity: 1;
  }
</style>
