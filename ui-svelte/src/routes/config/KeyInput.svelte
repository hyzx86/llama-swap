<script lang="ts">
  import { Input } from "$lib/components/ui/input/index.js";

  let {
    keyText,
    onCommit,
  }: {
    keyText: string;
    onCommit: (text: string) => void;
  } = $props();

  let draft = $state(keyText);
  $effect(() => {
    draft = keyText;
  });
</script>

<Input
  bind:value={draft}
  class="w-full font-mono text-xs"
  onblur={() => onCommit(draft)}
  onkeydown={(e) => e.key === "Enter" && onCommit(draft)}
/>
