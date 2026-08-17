<script lang="ts">
  import { onMount } from "svelte";
  import { Save, RefreshCw, FileWarning, FileCheck2 } from "@lucide/svelte";
  import * as Button from "$lib/components/ui/button/index.js";
  import { parseDocument } from "yaml";
  import type { Document } from "yaml";
  import { buildTree } from "$lib/config/yamlTree";
  import { fetchConfig, saveConfig } from "../stores/api";
  import type { ConfigFile } from "../stores/api";
  import ConfigTree from "./config/ConfigTree.svelte";
  import ConfigMonacoEditor from "./config/ConfigMonacoEditor.svelte";

  let doc = $state<Document | null>(null);
  let config = $state<ConfigFile | null>(null);
  let raw = $state("");
  let loadedText = $state("");
  let selectedPath = $state<string[]>([]);
  let loading = $state(false);
  let saving = $state(false);
  let error = $state<string | null>(null);
  let notice = $state<string | null>(null);
  let unavailable = $state(false);
  let rawError = $state<string | null>(null);

  let rawTimer: ReturnType<typeof setTimeout> | null = null;

  const dirty = $derived(!!config && raw !== loadedText);
  const tree = $derived.by(() => doc ? buildTree(doc) : []);

  async function load() {
    console.log('[ConfigEditor] load() called');
    loading = true;
    error = null;
    notice = null;
    try {
      console.log('[ConfigEditor] fetching config...');
      config = await fetchConfig();
      console.log('[ConfigEditor] config fetched:', config);
      loadedText = config.content;
      raw = config.content;
      console.log('[ConfigEditor] parsing document...');
      doc = parseDocument(config.content);
      console.log('[ConfigEditor] doc parsed:', doc);
      rawError = null;
    } catch (e) {
      console.error('[ConfigEditor] error loading config:', e);
      if (String(e).includes("unavailable")) {
        unavailable = true;
      } else {
        error = e instanceof Error ? e.message : String(e);
      }
    } finally {
      loading = false;
      console.log('[ConfigEditor] load() finished, loading=false');
    }
  }

  onMount(load);

  // Raw edits are re-parsed (debounced) into the document so the tree stays in sync.
  // Invalid YAML is shown but not applied until fixed.
  function onRawInput(text: string) {
    raw = text;
    if (rawTimer) clearTimeout(rawTimer);
    rawTimer = setTimeout(() => {
      const next = parseDocument(text);
      if (next.errors.length > 0) {
        rawError = next.errors[0].message;
      } else {
        doc = next;
        rawError = null;
      }
    }, 300);
  }

  function selectNode(path: string[]) {
    selectedPath = path;
  }

  async function save() {
    if (!config) return;
    saving = true;
    error = null;
    notice = null;
    try {
      await saveConfig(raw);
      loadedText = raw;
      doc = parseDocument(raw);
      rawError = null;
      notice = "Saved. The service reloads the config automatically.";
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      saving = false;
    }
  }
</script>

<div class="flex h-full min-h-0 flex-col">
  <div class="flex items-center gap-3 border-b px-4 py-2.5">
    <h2 class="text-base font-semibold">Config</h2>
    {#if config}
      <span class="truncate font-mono text-xs text-muted-foreground" title={config.path}>
        {config.path}
      </span>
    {/if}
    {#if dirty}
      <span class="flex items-center gap-1 rounded-full bg-warning/15 px-2 py-0.5 text-xs font-medium text-warning">
        <span class="size-1.5 rounded-full bg-warning"></span> Unsaved
      </span>
    {/if}

    <div class="ml-auto flex items-center gap-2">
      <Button.Root variant="ghost" size="sm" onclick={load} disabled={loading}>
        <RefreshCw class={loading ? "animate-spin" : ""} /> Reload
      </Button.Root>
      <Button.Root size="sm" onclick={save} disabled={!dirty || saving || !!rawError}>
        <Save class={saving ? "animate-pulse" : ""} /> {saving ? "Saving…" : "Save"}
      </Button.Root>
    </div>
  </div>

  {#if error}
    <div class="flex items-center gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-sm text-destructive">
      <FileWarning class="size-4 shrink-0" />
      <span class="min-w-0 flex-1">{error}</span>
    </div>
  {:else if notice}
    <div class="flex items-center gap-2 border-b border-success/30 bg-success/10 px-4 py-2 text-sm text-success">
      <FileCheck2 class="size-4 shrink-0" />
      <span class="min-w-0 flex-1">{notice}</span>
    </div>
  {/if}

  {#if unavailable}
    <div class="flex flex-1 items-center justify-center p-8 text-sm text-muted-foreground">
      Config editing is unavailable — the server was started without a <code class="font-mono">--config</code> file.
    </div>
  {:else if !doc}
    <div class="flex flex-1 items-center justify-center p-8 text-sm text-muted-foreground">
      {loading ? "Loading config…" : "No config loaded."}
    </div>
  {:else}
    <div class="flex min-h-0 flex-1">
      <div class="w-64 shrink-0 overflow-y-auto border-r">
        <ConfigTree nodes={tree} selectedPath={selectedPath} onSelect={selectNode} />
      </div>
      <div class="min-h-0 flex-1 p-3">
        {#if rawError}
          <div class="mb-2 flex items-center gap-2 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-1.5 text-xs text-destructive">
            <FileWarning class="size-3.5 shrink-0" /> {rawError}
          </div>
        {/if}
        <ConfigMonacoEditor bind:value={raw} language="yaml" onChange={onRawInput} />
      </div>
    </div>
  {/if}
</div>
