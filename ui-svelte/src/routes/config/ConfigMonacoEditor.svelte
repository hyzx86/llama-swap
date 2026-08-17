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

  // Persist the editor's view state (scroll position + cursor) in the browser
  // so reopening the editor restores the last scroll position.
  const VIEW_STATE_KEY = "llama-swap-config-editor-view-state";
  let scrollSaveTimer: ReturnType<typeof setTimeout> | null = null;

  function saveViewState(): void {
    if (!monacoEditor) return;
    try {
      const state = monacoEditor.saveViewState();
      if (state) localStorage.setItem(VIEW_STATE_KEY, JSON.stringify(state));
    } catch (e) {
      console.error("Failed to save editor view state:", e);
    }
  }

  function restoreViewState(): void {
    if (!monacoEditor) return;
    try {
      const saved = localStorage.getItem(VIEW_STATE_KEY);
      if (!saved) return;
      const state = JSON.parse(saved);
      // Defer so the freshly set content is laid out before restoring scroll.
      requestAnimationFrame(() => {
        monacoEditor?.restoreViewState(state);
      });
    } catch (e) {
      console.error("Failed to restore editor view state:", e);
    }
  }

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

      // Persist the scroll position (debounced) as the user scrolls.
      monacoEditor!.onDidScrollChange(() => {
        if (scrollSaveTimer) clearTimeout(scrollSaveTimer);
        scrollSaveTimer = setTimeout(saveViewState, 200);
      });

      // Restore the last scroll position for this editor.
      restoreViewState();
    } catch (err) {
      console.error("Failed to load Monaco Editor:", err);
    }
  });

  onDestroy(() => {
    if (scrollSaveTimer) clearTimeout(scrollSaveTimer);
    saveViewState(); // capture the final position before disposal
    monacoEditor?.dispose();
    monacoEditor = null;
  });

  // Update editor value when external value changes
  $effect(() => {
    if (monacoEditor !== null && value !== monacoEditor.getValue()) {
      monacoEditor.setValue(value);
      restoreViewState();
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
