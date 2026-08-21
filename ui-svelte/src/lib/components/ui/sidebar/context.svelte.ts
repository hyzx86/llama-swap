import { IsMobile } from "$lib/hooks/is-mobile.svelte.js";
import { getContext, setContext } from "svelte";
import { SIDEBAR_KEYBOARD_SHORTCUT, SIDEBAR_WIDTH_STORAGE_KEY } from "./constants.js";

/** Reads the persisted sidebar width (px) from localStorage, or `0` if none. */
function loadSidebarWidth(): number {
	try {
		const raw = localStorage.getItem(SIDEBAR_WIDTH_STORAGE_KEY);
		if (!raw) return 0;
		const n = Number.parseInt(raw, 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	} catch {
		return 0;
	}
}

/** Persists the sidebar width (px) to localStorage so it survives reloads. */
function saveSidebarWidth(width: number): void {
	try {
		localStorage.setItem(SIDEBAR_WIDTH_STORAGE_KEY, String(Math.round(width)));
	} catch {
		// Ignore storage failures (e.g. private mode).
	}
}

type Getter<T> = () => T;

export type SidebarStateProps = {
	/**
	 * A getter function that returns the current open state of the sidebar.
	 * We use a getter function here to support `bind:open` on the `Sidebar.Provider`
	 * component.
	 */
	open: Getter<boolean>;

	/**
	 * A function that sets the open state of the sidebar. To support `bind:open`, we need
	 * a source of truth for changing the open state to ensure it will be synced throughout
	 * the sub-components and any `bind:` references.
	 */
	setOpen: (open: boolean) => void;
};

class SidebarState {
	readonly props: SidebarStateProps;
	open = $derived.by(() => this.props.open());
	openMobile = $state(false);
	setOpen: SidebarStateProps["setOpen"];
	#isMobile: IsMobile;
	state = $derived.by(() => (this.open ? "expanded" : "collapsed"));

	/**
	 * Current sidebar width in pixels while the user is resizing it.
	 * `0` means "not resized yet, use the default `--sidebar-width`".
	 * Initialized from localStorage so the last dragged width survives reloads
	 * and collapse/expand cycles.
	 */
	sidebarWidth = $state(loadSidebarWidth());
	/** Whether a drag resize is currently in progress. */
	resizing = $state(false);

	/** Sets the sidebar width (in px) from a drag-resize gesture and persists it. */
	setSidebarWidth = (width: number) => {
		this.sidebarWidth = width;
		saveSidebarWidth(width);
	};

	beginResize = () => {
		this.resizing = true;
	};

	endResize = () => {
		this.resizing = false;
	};

	constructor(props: SidebarStateProps) {
		this.setOpen = props.setOpen;
		this.#isMobile = new IsMobile();
		this.props = props;
	}

	// Convenience getter for checking if the sidebar is mobile
	// without this, we would need to use `sidebar.isMobile.current` everywhere
	get isMobile() {
		return this.#isMobile.current;
	}

	// Event handler to apply to the `<svelte:window>`
	handleShortcutKeydown = (e: KeyboardEvent) => {
		if (e.key === SIDEBAR_KEYBOARD_SHORTCUT && (e.metaKey || e.ctrlKey)) {
			e.preventDefault();
			this.toggle();
		}
	};

	setOpenMobile = (value: boolean) => {
		this.openMobile = value;
	};

	toggle = () => {
		return this.#isMobile.current
			? (this.openMobile = !this.openMobile)
			: this.setOpen(!this.open);
	};
}

const SYMBOL_KEY = "scn-sidebar";

/**
 * Instantiates a new `SidebarState` instance and sets it in the context.
 *
 * @param props The constructor props for the `SidebarState` class.
 * @returns  The `SidebarState` instance.
 */
export function setSidebar(props: SidebarStateProps): SidebarState {
	return setContext(Symbol.for(SYMBOL_KEY), new SidebarState(props));
}

/**
 * Retrieves the `SidebarState` instance from the context. This is a class instance,
 * so you cannot destructure it.
 * @returns The `SidebarState` instance.
 */
export function useSidebar(): SidebarState {
	return getContext(Symbol.for(SYMBOL_KEY));
}
