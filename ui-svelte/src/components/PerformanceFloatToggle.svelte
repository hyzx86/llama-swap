<script lang="ts">
  import { floatVisible } from "../stores/performanceFloat";
  import { Activity } from "@lucide/svelte";

  function toggle() {
    $floatVisible = !$floatVisible;
  }
</script>

{#if !$floatVisible}
  <button
    class="perf-float-toggle"
    onclick={toggle}
    title="Show Performance Monitor"
  >
    <Activity size={16} />
  </button>
{/if}

<style>
  .perf-float-toggle {
    position: fixed;
    top: 12px;
    right: 12px;
    z-index: 9998;
    background: var(--primary);
    color: var(--primary-foreground);
    border: none;
    border-radius: 50%;
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
    opacity: 0.85;
    transition: opacity 0.2s;
  }

  .perf-float-toggle:hover {
    opacity: 1;
  }
</style>
