<script lang="ts">
  import TreeNode from "./TreeNode.svelte";
  import type { ConfigNode } from "$lib/config/yamlTree";

  let {
    nodes,
    selectedPath,
    onSelect,
  }: {
    nodes: ConfigNode[];
    selectedPath: string[];
    onSelect: (path: string[]) => void;
  } = $props();
</script>

<div class="flex flex-col gap-0.5 p-2">
  {#if nodes.length === 0}
    <p class="p-2 text-sm text-muted-foreground">No top-level keys.</p>
  {:else}
    {#each nodes as node (node.path.join("\u0000"))}
      <TreeNode {node} {selectedPath} {onSelect} depth={0} />
    {/each}
  {/if}
</div>
