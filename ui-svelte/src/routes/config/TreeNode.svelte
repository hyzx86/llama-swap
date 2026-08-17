<script lang="ts">
  import {
    ChevronRight,
    Boxes,
    Network,
    GitBranch,
    Hash,
    Layers,
    Tags,
    Settings2,
    Braces,
    List,
    FileCog,
  } from "@lucide/svelte";
  import type { ConfigNode } from "$lib/config/yamlTree";
  import TreeNode from "./TreeNode.svelte";

  let {
    node,
    selectedPath,
    onSelect,
    depth = 0,
  }: {
    node: ConfigNode;
    selectedPath: string[];
    onSelect: (path: string[]) => void;
    depth?: number;
  } = $props();

  const hasChildren = $derived(node.kind !== "scalar");
  // Top-level entries start expanded so the main sections are visible.
  let expanded = $state(depth === 0);

  const selected = $derived(selectedPath.length === node.path.length && selectedPath.every((seg, i) => seg === node.path[i]));

  function toggle() {
    if (hasChildren) expanded = !expanded;
    onSelect(node.path);
  }

  // Known top-level sections get a distinct icon; everything else falls back
  // to a generic collection/scalar glyph.
  const TOP_ICONS: Record<string, string> = {
    models: "models",
    peers: "peers",
    selectors: "selectors",
    macros: "macros",
    routing: "routing",
    profiles: "profiles",
    targets: "targets",
  };

  const icon = $derived.by(() => {
    if (depth === 0 && TOP_ICONS[node.key]) return TOP_ICONS[node.key];
    if (node.kind === "seq") return "seq";
    if (node.kind === "scalar") return "scalar";
    return "map";
  });
</script>

<div>
  <button
    type="button"
    class="flex w-full items-center gap-1 rounded-md py-1 pr-2 text-left text-sm hover:bg-muted/70
      {selected ? 'bg-accent text-accent-foreground' : 'text-foreground'}"
    style="padding-left: {8 + depth * 16}px"
    onclick={toggle}
    title={node.path.join(" > ") || "root"}
  >
    {#if hasChildren}
      <span class="shrink-0 transition-transform duration-150 {expanded ? 'rotate-90' : ''}">
        <ChevronRight class="size-3.5" />
      </span>
    {:else}
      <span class="w-3.5 shrink-0"></span>
    {/if}

    {#if icon === "models"}
      <Boxes class="size-4 shrink-0 text-primary" />
    {:else if icon === "peers"}
      <Network class="size-4 shrink-0 text-primary" />
    {:else if icon === "selectors"}
      <GitBranch class="size-4 shrink-0 text-primary" />
    {:else if icon === "macros"}
      <Hash class="size-4 shrink-0 text-primary" />
    {:else if icon === "routing"}
      <Layers class="size-4 shrink-0 text-primary" />
    {:else if icon === "profiles"}
      <Tags class="size-4 shrink-0 text-primary" />
    {:else if icon === "targets"}
      <FileCog class="size-4 shrink-0 text-primary" />
    {:else if icon === "map"}
      <Braces class="size-4 shrink-0 text-muted-foreground" />
    {:else if icon === "seq"}
      <List class="size-4 shrink-0 text-muted-foreground" />
    {:else if icon === "scalar"}
      <span class="mx-1.5 size-1.5 shrink-0 rounded-full bg-muted-foreground/50"></span>
    {:else}
      <Settings2 class="size-4 shrink-0 text-primary" />
    {/if}

    <span class="truncate font-medium">{node.key || "(empty)"}</span>
    {#if node.kind === "scalar" && !node.multiline}
      <span class="ml-1 truncate text-xs text-muted-foreground">: {node.value || "∅"}</span>
    {:else if node.kind === "scalar" && node.multiline}
      <span class="ml-1 text-xs text-muted-foreground">¶</span>
    {/if}
  </button>

  {#if hasChildren && expanded}
    <div>
      {#each node.children as child (child.path.join("\u0000"))}
        <TreeNode node={child} {selectedPath} {onSelect} depth={depth + 1} />
      {/each}
    </div>
  {/if}
</div>
