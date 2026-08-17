<script lang="ts">
  import hljs from "highlight.js/lib/core";
  import yaml from "highlight.js/lib/languages/yaml";

  hljs.registerLanguage("yaml", yaml);

  let {
    value = $bindable(""),
    onInput,
  }: {
    value?: string;
    onInput?: (text: string) => void;
  } = $props();

  let gutter: HTMLDivElement;
  let highlightedHtml = $state("");
  let lineNumbers = $derived(value.split("\n").map((_, i) => i + 1));

  // Keep highlight overlay aligned with textarea scroll.
  function syncScroll(e: Event) {
    const ta = e.currentTarget as HTMLTextAreaElement;
    if (gutter) gutter.scrollTop = ta.scrollTop;
  }

  // Re-highlight whenever the text changes.
  $effect(() => {
    const text = value;
    const result = hljs.highlight(text, { language: "yaml" });
    // Replace newlines with <br> for display in pre/code, and escape HTML entities.
    highlightedHtml = result.value.replace(/\n$/, "");
  });
</script>

<div class="relative flex h-full min-h-0 overflow-hidden rounded-md border">
  <!-- Line number gutter -->
  <div
    bind:this={gutter}
    class="w-12 shrink-0 select-none overflow-hidden border-r bg-muted/40 py-2 text-right font-mono text-xs leading-6 text-muted-foreground"
    aria-hidden="true"
  >
    {#each lineNumbers as n (n)}
      <div class="pr-2">{n}</div>
    {/each}
  </div>

  <!-- Highlight overlay (pointer-events-none sits behind the textarea) -->
  <pre
    class="pointer-events-none absolute inset-0 m-0 min-h-0 flex-1 overflow-hidden rounded-l-none border-0 bg-transparent p-2 font-mono text-xs leading-6"
    aria-hidden="true"
  ><code>{@html highlightedHtml}</code></pre>

  <!-- Editable textarea -->
  <textarea
    bind:value
    oninput={(e) => onInput?.((e.currentTarget as HTMLTextAreaElement).value)}
    onscroll={syncScroll}
    spellcheck="false"
    class="absolute inset-0 min-h-0 flex-1 resize-none overflow-auto rounded-l-none border-0 bg-transparent p-2 font-mono text-xs leading-6 outline-none text-transparent"
    aria-label="YAML editor"
  ></textarea>
</div>

<style>
  :global(.dark) pre.hljs {
    background: transparent;
  }
</style>
