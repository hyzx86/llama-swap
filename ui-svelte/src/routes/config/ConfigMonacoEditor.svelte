<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import loader from "@monaco-editor/loader";
  import type { editor } from "monaco-editor";

  let {
    value = $bindable(""),
    language = "yaml",
    onChange,
  }: {
    value?: string;
    language?: string;
    onChange?: (value: string) => void;
  } = $props();

  let container: HTMLDivElement;
  let monacoEditor: editor.IStandaloneCodeEditor | null = null;
  let editorLoaded = $state(false);

  onMount(async () => {
    try {
      // Configure Monaco to load from CDN
      loader.config({
        paths: {
          vs: "https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs",
        },
      });

      const monaco = await loader.init();
      
      monacoEditor = monaco.editor.create(container!, {
        value: value,
        language: language,
        theme: "vs-dark",
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        fontSize: 13,
        lineHeight: 20,
        fontFamily: "JetBrains Mono, Fira Code, Consolas, monospace",
        wordWrap: "on",
        automaticLayout: true,
        padding: { top: 12, bottom: 12 },
        bracketsColorization: true,
        guide: {
          indentation: true,
        },
      });

      editorLoaded = true;

      // Listen for changes
      monacoEditor!.onDidChangeModelContent(() => {
        const newValue = monacoEditor!.getValue();
        if (newValue !== value) {
          onChange?.(newValue);
        }
      });
    } catch (err) {
      console.error("Failed to load Monaco Editor:", err);
    }
  });

  onDestroy(() => {
    monacoEditor?.dispose();
    monacoEditor = null;
  });

  // Update editor value when external value changes
  $effect(() => {
    if (monacoEditor !== null && value !== monacoEditor.getValue()) {
      monacoEditor.setValue(value);
    }
  });
</script>

<div class="relative h-full min-h-0">
  <div
    bind:this={container}
    class="h-full min-h-0 w-full rounded-md border"
    class:loading={!editorLoaded}
  ></div>
  {#if !editorLoaded}
    <div class="absolute inset-0 flex items-center justify-center bg-background/80">
      <span class="text-sm text-muted-foreground">Loading editor...</span>
    </div>
  {/if}
</div>

<style>
  :global(.monaco-editor) {
    background: transparent !important;
  }
</style>
