<script lang="ts">
  import { link } from "svelte-spa-router";
  import { FerrisWheel, Boxes, Activity, ScrollText, Gauge, Cpu, Sun, Moon, Monitor, ChevronRight, Settings, FileCog, Star, Copy, ExternalLink } from "@lucide/svelte";
  import { Portal } from "bits-ui";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import * as Collapsible from "$lib/components/ui/collapsible/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { toggleTheme, themeMode, appTitle } from "../stores/theme";
  import { currentRoute } from "../stores/route";
  import { playgroundActivity } from "../stores/playgroundActivity";
  import { performanceEnabled, models } from "../stores/api";
  import { showUnlistedModels } from "../stores/modelDisplay";
  import { modelsMenuOpen } from "../stores/sidebar";
  import { favorites, toggleFavorite } from "../stores/favorites";
  import type { Model } from "../lib/types";
  import ConnectionStatus from "./ConnectionStatus.svelte";

  function handleTitleChange(newTitle: string): void {
    const sanitized = newTitle.replace(/\n/g, "").trim().substring(0, 64) || "llama-swap";
    appTitle.set(sanitized);
  }

  function handleKeyDown(e: KeyboardEvent): void {
    if (e.key === "Enter") {
      e.preventDefault();
      const target = e.currentTarget as HTMLElement;
      handleTitleChange(target.textContent || "(set title)");
      target.blur();
    }
  }

  function handleBlur(e: FocusEvent): void {
    const target = e.currentTarget as HTMLElement;
    handleTitleChange(target.textContent || "(set title)");
  }

  function isActive(path: string, current: string): boolean {
    return path === "/" ? current === "/" : current.startsWith(path);
  }

  let visibleLocalModels = $derived(
    $models.filter((model) => !model.peerID && ($showUnlistedModels || !model.unlisted)),
  );
  let visiblePeerModels = $derived(
    $models.filter((model) => model.peerID && ($showUnlistedModels || !model.unlisted)),
  );

  // Favorited models are sorted to the top of each list. The subscription to
  // $favorites keeps the sort reactive when favorites change.
  let localModelsSorted = $derived.by(() => {
    const favs = new Set($favorites);
    return [...visibleLocalModels].sort((a, b) => Number(favs.has(b.id)) - Number(favs.has(a.id)));
  });
  let peerModelsSorted = $derived.by(() => {
    const favs = new Set($favorites);
    return [...visiblePeerModels].sort((a, b) => Number(favs.has(b.id)) - Number(favs.has(a.id)));
  });

  function copyModelId(id: string): void {
    void navigator.clipboard.writeText(id);
  }

  // Right-click context menu for model rows. It is rendered through a portal
  // into <body> so it escapes the sidebar's stacking context (the sidebar rail
  // would otherwise cover it, regardless of z-index) and is always shown at a
  // fixed spot on the left of the viewport, with its height sized to content.
  let modelMenu = $state<{ model: Model; y: number } | null>(null);
  let modelMenuRef = $state<HTMLDivElement | null>(null);

  function openModelMenu(e: MouseEvent, model: Model): void {
    e.preventDefault();
    e.stopPropagation();
    // Clamp so the menu stays fully visible vertically.
    const y = Math.min(Math.max(e.clientY, 8), window.innerHeight - 120);
    modelMenu = { model, y };
  }

  function closeModelMenu(): void {
    modelMenu = null;
  }

  function onDocPointerDown(e: PointerEvent): void {
    if (modelMenuRef && !modelMenuRef.contains(e.target as Node)) {
      closeModelMenu();
    }
  }

  function onDocKeyDown(e: KeyboardEvent): void {
    if (e.key === "Escape") closeModelMenu();
  }

  type DotColor = "grey" | "yellow" | "green";
  function statusDotColor(model: Model): DotColor {
    if (model.state === "ready") return "green";
    if (model.state === "starting" || model.state === "stopping") return "yellow";
    return "grey";
  }

  const dotClass: Record<DotColor, string> = {
    grey: "bg-muted-foreground/40",
    yellow: "bg-warning",
    green: "bg-success",
  };
</script>

{#snippet modelMenuItem(model: Model)}
  {@const fav = $favorites.includes(model.id)}
  <Sidebar.MenuSubItem>
    <div
      class="group/model-row relative flex items-center"
      oncontextmenu={(e) => openModelMenu(e, model)}
    >
      <Sidebar.MenuSubButton
        isActive={$currentRoute === `/models/${encodeURIComponent(model.id)}`}
      >
        {#snippet child({ props })}
          <a href="/models/{encodeURIComponent(model.id)}" use:link {...props}>
            <span class={`size-2 shrink-0 rounded-full ${dotClass[statusDotColor(model)]}`}></span>
            <span class="flex-1 truncate pr-8">{model.id}</span>
          </a>
        {/snippet}
      </Sidebar.MenuSubButton>
      <div class="absolute right-1 flex items-center">
        <button
          class="flex size-5 shrink-0 items-center justify-center rounded-md transition-colors {fav ? 'text-warning' : 'text-sidebar-foreground/40 hover:text-sidebar-foreground'}"
          title={fav ? "Remove from favorites" : "Add to favorites"}
          aria-label={fav ? "Remove from favorites" : "Add to favorites"}
          onclick={(e) => {
            e.stopPropagation();
            toggleFavorite(model.id);
          }}
        >
          <Star class="size-3.5" fill={fav ? "currentColor" : "none"} />
        </button>
      </div>
    </div>
  </Sidebar.MenuSubItem>
{/snippet}

<Sidebar.Root collapsible="icon">
  <Sidebar.Header>
    <div class="flex items-center gap-2 px-2 py-1.5">
      <div class="flex shrink-0 items-center justify-center">
        <ConnectionStatus />
      </div>
      <h1
        contenteditable="true"
        class="truncate pb-0 text-base font-semibold outline-none rounded-md px-1 hover:bg-sidebar-accent group-data-[collapsible=icon]:hidden"
        onblur={handleBlur}
        onkeydown={handleKeyDown}
      >
        {$appTitle}
      </h1>
    </div>
  </Sidebar.Header>

  <Sidebar.Content>
    <Sidebar.Group>
      <Sidebar.GroupContent>
        <Sidebar.Menu class="gap-1">
          <Sidebar.MenuItem>
            <Sidebar.MenuButton isActive={$currentRoute === "/" || isActive("/activity", $currentRoute)} tooltipContent="Activity">
              {#snippet child({ props })}
                <a href="/" use:link {...props}>
                  <Activity />
                  <span>Activity</span>
                </a>
              {/snippet}
            </Sidebar.MenuButton>
          </Sidebar.MenuItem>

          <Sidebar.MenuItem>
            <Sidebar.MenuButton isActive={isActive("/playground", $currentRoute)} tooltipContent="Playground">
              {#snippet child({ props })}
                <a href="/playground" use:link {...props}>
                  <FerrisWheel />
                  <span class={$playgroundActivity ? "activity-link" : ""}>Playground</span>
                </a>
              {/snippet}
            </Sidebar.MenuButton>
          </Sidebar.MenuItem>

          <Sidebar.MenuItem>
            <Sidebar.MenuButton isActive={isActive("/config", $currentRoute)} tooltipContent="Config">
              {#snippet child({ props })}
                <a href="/config" use:link {...props}>
                  <FileCog />
                  <span>Config</span>
                </a>
              {/snippet}
            </Sidebar.MenuButton>
          </Sidebar.MenuItem>

          <Sidebar.MenuItem>
            <Collapsible.Root
              open={$modelsMenuOpen}
              onOpenChange={(v) => modelsMenuOpen.set(v)}
              class="gap-0"
            >
              <Sidebar.MenuButton
                isActive={$currentRoute.startsWith("/models")}
                tooltipContent="Models"
              >
                {#snippet child({ props })}
                  <a href="/models" use:link {...props}>
                    <Boxes />
                    <span>Models</span>
                    <span
                      class="ml-auto transition-transform duration-200 {$modelsMenuOpen ? 'rotate-90' : ''}"
                      role="button"
                      tabindex="0"
                      aria-label="Toggle models section"
                      onclick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        modelsMenuOpen.update((v) => !v);
                      }}
                      onkeydown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          e.stopPropagation();
                          modelsMenuOpen.update((v) => !v);
                        }
                      }}
                    >
                      <ChevronRight />
                    </span>
                  </a>
                {/snippet}
              </Sidebar.MenuButton>
              <Collapsible.Content>
                <Sidebar.MenuSub>
                  {#each localModelsSorted as model (model.id)}
                    {@render modelMenuItem(model)}
                  {/each}
                  {#if peerModelsSorted.length > 0}
                    <li class="text-sidebar-foreground/70 px-2 pt-2 pb-1 text-xs font-medium">
                      Peers
                    </li>
                    {#each peerModelsSorted as model (model.id)}
                      {@render modelMenuItem(model)}
                    {/each}
                  {/if}
                </Sidebar.MenuSub>
              </Collapsible.Content>
            </Collapsible.Root>
          </Sidebar.MenuItem>

          <Sidebar.MenuItem>
            <Sidebar.MenuButton isActive={isActive("/logs", $currentRoute)} tooltipContent="Logs">
              {#snippet child({ props })}
                <a href="/logs" use:link {...props}>
                  <ScrollText />
                  <span>Logs</span>
                </a>
              {/snippet}
            </Sidebar.MenuButton>
          </Sidebar.MenuItem>

          {#if $performanceEnabled}
            <Sidebar.MenuItem>
              <Sidebar.MenuButton isActive={isActive("/performance", $currentRoute)} tooltipContent="Performance">
                {#snippet child({ props })}
                  <a href="/performance" use:link {...props}>
                    <Gauge />
                    <span>Performance</span>
                  </a>
                {/snippet}
              </Sidebar.MenuButton>
            </Sidebar.MenuItem>
          {/if}

          <Sidebar.MenuItem>
            <Sidebar.MenuButton isActive={isActive("/hardware", $currentRoute)} tooltipContent="Hardware">
              {#snippet child({ props })}
                <a href="/hardware" use:link {...props}>
                  <Cpu />
                  <span>Hardware</span>
                </a>
              {/snippet}
            </Sidebar.MenuButton>
          </Sidebar.MenuItem>
        </Sidebar.Menu>
      </Sidebar.GroupContent>
    </Sidebar.Group>
  </Sidebar.Content>

  <Sidebar.Footer>
    <div
      class="flex items-center justify-between gap-2 px-1 group-data-[collapsible=icon]:flex-col-reverse"
    >
      <Sidebar.MenuButton
        isActive={isActive("/settings", $currentRoute)}
        tooltipContent="Settings"
      >
        {#snippet child({ props })}
          <a href="/settings" use:link {...props}>
            <Settings />
            <span>Settings</span>
          </a>
        {/snippet}
      </Sidebar.MenuButton>
      <Button
        variant="ghost"
        size="icon"
        onclick={toggleTheme}
        title="Toggle theme (current: {$themeMode})"
      >
        {#if $themeMode === "system"}
          <Monitor />
        {:else if $themeMode === "light"}
          <Sun />
        {:else}
          <Moon />
        {/if}
        <span class="sr-only">Toggle theme</span>
      </Button>
    </div>
  </Sidebar.Footer>
  <Sidebar.Rail />
</Sidebar.Root>

<svelte:window onpointerdown={onDocPointerDown} onkeydown={onDocKeyDown} />

{#if modelMenu}
  <Portal>
    <div
      bind:this={modelMenuRef}
      role="menu"
      style="left: 86px; top: {modelMenu.y}px;"
      class="ring-foreground/10 bg-popover text-popover-foreground animate-in fade-in-0 zoom-in-95 fixed z-50 min-w-44 rounded-lg p-1 shadow-md ring-1 outline-none"
      oncontextmenu={(e) => e.preventDefault()}
    >
      <div
        role="menuitem"
        tabindex="0"
        class="hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground relative flex cursor-default items-center gap-1.5 rounded-md px-1.5 py-1 text-sm outline-hidden select-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4"
        onclick={() => {
          toggleFavorite(modelMenu.model.id);
          closeModelMenu();
        }}
        onkeydown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            toggleFavorite(modelMenu.model.id);
            closeModelMenu();
          }
        }}
      >
        <Star class={$favorites.includes(modelMenu.model.id) ? "fill-current" : ""} />
        {$favorites.includes(modelMenu.model.id) ? "Remove from favorites" : "Add to favorites"}
      </div>
      <div
        role="menuitem"
        tabindex="0"
        class="hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground relative flex cursor-default items-center gap-1.5 rounded-md px-1.5 py-1 text-sm outline-hidden select-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4"
        onclick={() => {
          copyModelId(modelMenu.model.id);
          closeModelMenu();
        }}
        onkeydown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            copyModelId(modelMenu.model.id);
            closeModelMenu();
          }
        }}
      >
        <Copy />
        Copy ID
      </div>
      <div
        role="menuitem"
        tabindex="0"
        class="hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground relative flex cursor-default items-center gap-1.5 rounded-md px-1.5 py-1 text-sm outline-hidden select-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4"
        onkeydown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            closeModelMenu();
            window.location.hash = `/models/${encodeURIComponent(modelMenu.model.id)}`;
          }
        }}
      >
        <a
          href="/models/{encodeURIComponent(modelMenu.model.id)}"
          use:link
          class="flex items-center gap-1.5"
          onclick={(e) => {
            e.stopPropagation();
            setTimeout(closeModelMenu, 0);
          }}
        >
          <ExternalLink />
          Open
        </a>
      </div>
    </div>
  </Portal>
{/if}
