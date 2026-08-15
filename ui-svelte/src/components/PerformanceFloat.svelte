<script lang="ts">
  import { fetchPerformance } from "../stores/api";
  import type { SysStat, GpuStat } from "../lib/types";
  import { floatVisible } from "../stores/performanceFloat";

  let sysData = $state<SysStat[]>([]);
  let gpuData = $state<GpuStat[]>([]);
  let x = $state(20);
  let y = $state(80);
  let dragging = false;
  let startX = 0;
  let startY = 0;
  let startXPos = 0;
  let startYPos = 0;

  const floatStats = $derived.by(() => {
    const sys = sysData[sysData.length - 1];
    // API returns stats oldest→newest, so the newest sample is the last element.
    if (!sys) return null;
    const gpu = gpuData[gpuData.length - 1] ?? null;
    return {
      memUsedMb: sys.mem_used_mb,
      memTotalMb: sys.mem_total_mb,
      gpuTemp: gpu?.temp_c ?? 0,
      gpuVramUsedMb: gpu?.mem_used_mb ?? 0,
      gpuVramTotalMb: gpu?.mem_total_mb ?? 0,
      gpuName: gpu?.name ?? "",
    };
  });

  const memPercent = $derived(
    floatStats && floatStats.memTotalMb > 0
      ? Math.min(100, (floatStats.memUsedMb / floatStats.memTotalMb) * 100)
      : 0
  );
  const vramPercent = $derived(
    floatStats && floatStats.gpuVramTotalMb > 0
      ? Math.min(100, (floatStats.gpuVramUsedMb / floatStats.gpuVramTotalMb) * 100)
      : 0
  );

  // "Traffic light" color based on load percentage (per metric).
  function loadColor(pct: number): string {
    if (pct >= 85) return "#ef4444";
    if (pct >= 60) return "#f59e0b";
    return "#22c55e";
  }

  // GPU temperature color: 30-40 green, 40-50 yellow, 50+ red.
  function tempColor(t: number): string {
    if (t >= 50) return "#ef4444";
    if (t >= 40) return "#f59e0b";
    return "#22c55e";
  }

  async function tick() {
    const lastTs = sysData.length > 0 ? sysData[sysData.length - 1].timestamp : undefined;
    const resp = await fetchPerformance(lastTs);
    if (resp) {
      const newSys = resp.sys_stats ?? [];
      const newGpu = resp.gpu_stats ?? [];
      if (newSys.length > 0) sysData = [...sysData, ...newSys].slice(-500);
      if (newGpu.length > 0) gpuData = [...gpuData, ...newGpu].slice(-500);
    }
  }

  $effect(() => {
    void tick();
    const interval = setInterval(tick, 2000);
    return () => clearInterval(interval);
  });

  function formatMb(mb: number): string {
    if (mb >= 1024) return (mb / 1024).toFixed(1) + " GB";
    return Math.round(mb).toFixed(0) + " MB";
  }

  function formatPct(used: number, total: number): string {
    if (!total) return "—";
    return ((used / total) * 100).toFixed(0) + "%";
  }

  function onDragStart(e: MouseEvent) {
    if ((e.target as HTMLElement).closest(".perf-float-btn, .perf-compact-close")) return;
    dragging = true;
    startX = e.clientX;
    startY = e.clientY;
    startXPos = x;
    startYPos = y;
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
  }

  function onMouseMove(e: MouseEvent) {
    if (!dragging) return;
    x = startXPos + (e.clientX - startX);
    y = startYPos + (e.clientY - startY);
  }

  function onMouseUp() {
    dragging = false;
    document.removeEventListener("mousemove", onMouseMove);
    document.removeEventListener("mouseup", onMouseUp);
  }

  function toggleVisible() {
    $floatVisible = false;
  }
</script>

{#if $floatVisible}
  <div
    class="perf-float-container"
    style="left: {x}px; top: {y}px;"
  >
    <div
      class="perf-float perf-float-compact"
      role="button"
      tabindex="-1"
      aria-label="Drag to move"
      title="Drag to move"
      onmousedown={onDragStart}
    >
      {#if floatStats}
        <span class="perf-compact-line">
          <span style="color: {loadColor(memPercent)};">RAM {formatMb(floatStats.memUsedMb)}/{formatMb(floatStats.memTotalMb)} ({formatPct(floatStats.memUsedMb, floatStats.memTotalMb)})</span>
          {#if floatStats.gpuVramTotalMb > 0} <span style="color: {loadColor(vramPercent)};">· VRAM {formatMb(floatStats.gpuVramUsedMb)}/{formatMb(floatStats.gpuVramTotalMb)} ({formatPct(floatStats.gpuVramUsedMb, floatStats.gpuVramTotalMb)})</span>{/if}
          {#if floatStats.gpuTemp > 0} <span style="color: {tempColor(floatStats.gpuTemp)};">· {Math.round(floatStats.gpuTemp)}°C</span>{/if}
        </span>
      {:else}
        <span class="perf-compact-line">…</span>
      {/if}
      <button class="perf-compact-close" title="Close" onclick={toggleVisible}>✕</button>
    </div>
  </div>
{/if}

<style>
  .perf-float-container {
    position: fixed;
    z-index: 9999;
    pointer-events: none;
  }

  .perf-float {
    pointer-events: auto;
    background: var(--background);
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
    min-width: 420px;
    max-width: 720px;
    overflow: hidden;
    user-select: none;
    cursor: grab;
  }

  .perf-float-compact {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    max-width: none;
    padding: 4px 6px 4px 14px;
    border-radius: 999px;
    border-width: 2px;
    border-style: solid;
    border-color: var(--foreground);
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.4);
    white-space: nowrap;
    font-weight: 600;
  }

  .perf-compact-line {
    font-size: 12px;
    white-space: nowrap;
  }

  .perf-compact-close {
    background: none;
    border: none;
    color: inherit;
    cursor: pointer;
    font-size: 12px;
    line-height: 1;
    padding: 2px 4px;
    border-radius: 999px;
    opacity: 0.7;
    flex-shrink: 0;
  }

  .perf-compact-close:hover {
    opacity: 1;
    background: rgba(127, 127, 127, 0.25);
  }

  .perf-float:active {
    cursor: grabbing;
  }
</style>
