<script lang="ts">
  import { fetchPerformance } from "../stores/api";
  import type { SysStat, GpuStat } from "../lib/types";

  let { class: className = "" }: { class?: string } = $props();

  let sysData = $state<SysStat[]>([]);
  let gpuData = $state<GpuStat[]>([]);

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
</script>

<div class="perf-float perf-float-compact {className}" role="status" aria-label="Performance monitor">
  {#if floatStats}
    <span class="perf-compact-line">
      <span style="color: {loadColor(memPercent)};">RAM {formatMb(floatStats.memUsedMb)}/{formatMb(floatStats.memTotalMb)} ({formatPct(floatStats.memUsedMb, floatStats.memTotalMb)})</span>
      {#if floatStats.gpuVramTotalMb > 0} <span style="color: {loadColor(vramPercent)};">· VRAM {formatMb(floatStats.gpuVramUsedMb)}/{formatMb(floatStats.gpuVramTotalMb)} ({formatPct(floatStats.gpuVramUsedMb, floatStats.gpuVramTotalMb)})</span>{/if}
      {#if floatStats.gpuTemp > 0} <span style="color: {tempColor(floatStats.gpuTemp)};">· {Math.round(floatStats.gpuTemp)}°C</span>{/if}
    </span>
  {:else}
    <span class="perf-compact-line">…</span>
  {/if}
</div>

<style>
  .perf-float {
    pointer-events: auto;
    user-select: none;
  }

  .perf-float-compact {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    max-width: none;
    padding: 2px 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .perf-compact-line {
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Docked at the right end of the content header (title bar). Pushing to
     the right is handled by Tailwind's ml-auto on the caller; this rule only
     needs to handle the narrow-screen wrap behaviour. */
  :global(.perf-float-dock) {
    flex-shrink: 0;
  }

  /* On narrow / mobile screens the header wraps, so the monitor drops onto
     its own full-width row directly beneath the title and stays pinned to
     the top (the header itself is sticky). It stays right-aligned. */
  @media (max-width: 767px) {
    :global(.perf-float-dock) {
      order: 5;
      margin-left: auto;
      flex-basis: 100%;
      justify-content: flex-end;
    }

    .perf-float-compact {
      max-width: 100%;
    }
  }
</style>
