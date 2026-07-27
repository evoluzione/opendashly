<script lang="ts">
  import { onMount } from "svelte";

  export let text: string;
  export let title: string | undefined = undefined;
  let className = "";
  export { className as class };

  let container: HTMLElement;
  let inner: HTMLElement;
  let overflowPx = 0;
  let hovering = false;

  function measure() {
    if (!container || !inner) return;
    overflowPx = Math.max(0, inner.scrollWidth - container.clientWidth);
  }

  function handleEnter() {
    measure();
    if (overflowPx > 0) hovering = true;
  }

  function handleLeave() {
    hovering = false;
  }

  onMount(() => {
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(container);
    return () => ro.disconnect();
  });

  // Roughly constant scroll speed regardless of how much text overflows.
  $: scrollDuration = Math.max(0.5, overflowPx / 45);
</script>

<span
  class="marquee {className}"
  bind:this={container}
  title={title ?? text}
  on:mouseenter={handleEnter}
  on:mouseleave={handleLeave}
><span
    class="marquee-inner"
    bind:this={inner}
    style:transform={hovering ? `translateX(-${overflowPx}px)` : "translateX(0)"}
    style:transition-duration={hovering ? `${scrollDuration}s` : "0.25s"}
  >{text}</span
  ></span
>

<style>
  .marquee {
    display: block;
    overflow: hidden;
    white-space: nowrap;
  }

  .marquee-inner {
    display: inline-block;
    white-space: nowrap;
    transition-property: transform;
    transition-timing-function: linear;
  }
</style>
