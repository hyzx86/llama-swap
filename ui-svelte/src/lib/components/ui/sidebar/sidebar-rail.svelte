<script lang="ts">
	import { cn, type WithElementRef } from "$lib/utils.js";
	import type { HTMLAttributes } from "svelte/elements";
	import { useSidebar } from "./context.svelte.js";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLButtonElement>, HTMLButtonElement> = $props();

	const sidebar = useSidebar();

// Drag-to-resize support. A plain click still toggles the sidebar, but
// dragging the rail horizontally resizes it instead.
let dragStartX = 0;
let dragStartWidth = 0;
let dragging = false;
let moved = false;

const MIN_WIDTH = 256; // 16rem
const MAX_WIDTH_RATIO = 0.5; // at most 50% of the viewport width

function onPointerDown(e: PointerEvent) {
	if (e.button !== 0 || sidebar.isMobile) return;
	e.preventDefault();
	dragging = true;
	moved = false;
	dragStartX = e.clientX;

	// If the sidebar is collapsed (icon mode), expand it first so the width
	// can be adjusted. Start from the persisted width if there is one, so
	// expand restores the last dragged position.
	if (sidebar.state === "collapsed") {
		sidebar.toggle();
		dragStartWidth =
			sidebar.sidebarWidth > 0
				? sidebar.sidebarWidth
				: Math.min(480, Math.round(window.innerWidth * MAX_WIDTH_RATIO));
	} else {
		const el = document.querySelector(
			'[data-slot="sidebar-container"]'
		) as HTMLElement | null;
		dragStartWidth = el ? el.getBoundingClientRect().width : 320;
	}

	sidebar.beginResize();
	document.body.style.userSelect = "none";
	document.body.style.cursor = "col-resize";
	document.addEventListener("pointermove", onPointerMove);
	document.addEventListener("pointerup", onPointerUp);
	document.addEventListener("pointercancel", onPointerUp);
}

function onPointerMove(e: PointerEvent) {
	if (!dragging) return;
	const delta = e.clientX - dragStartX;
	if (!moved && Math.abs(delta) > 3) moved = true;
	const maxWidth = Math.max(MIN_WIDTH, Math.round(window.innerWidth * MAX_WIDTH_RATIO));
	const next = Math.min(maxWidth, Math.max(MIN_WIDTH, dragStartWidth + delta));
	sidebar.setSidebarWidth(next);
}

function onPointerUp() {
	if (!dragging) return;
	dragging = false;
	sidebar.endResize();
	document.body.style.userSelect = "";
	document.body.style.cursor = "";
	document.removeEventListener("pointermove", onPointerMove);
	document.removeEventListener("pointerup", onPointerUp);
	document.removeEventListener("pointercancel", onPointerUp);
}

function handleClick() {
	if (moved) {
		moved = false;
		return;
	}
	sidebar.toggle();
}
</script>

<button
	bind:this={ref}
	data-sidebar="rail"
	data-slot="sidebar-rail"
	aria-label="Toggle Sidebar"
	tabindex={-1}
	onclick={handleClick}
	onpointerdown={onPointerDown}
	title="Toggle Sidebar"
	class={cn(
		"hover:after:bg-sidebar-border absolute inset-y-0 z-20 hidden w-4 -translate-x-1/2 transition-all ease-linear group-data-[side=left]:-right-4 group-data-[side=right]:left-0 after:absolute after:inset-y-0 after:left-1/2 after:w-[2px] sm:flex",
		"in-data-[side=left]:cursor-w-resize in-data-[side=right]:cursor-e-resize",
		"[[data-side=left][data-state=collapsed]_&]:cursor-e-resize [[data-side=right][data-state=collapsed]_&]:cursor-w-resize",
		"hover:group-data-[collapsible=offcanvas]:bg-sidebar group-data-[collapsible=offcanvas]:translate-x-0 group-data-[collapsible=offcanvas]:after:left-full",
		"[[data-side=left][data-collapsible=offcanvas]_&]:-right-2",
		"[[data-side=right][data-collapsible=offcanvas]_&]:-left-2",
		className
	)}
	{...restProps}
>
	{@render children?.()}
</button>
