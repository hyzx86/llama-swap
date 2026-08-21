import { get } from "svelte/store";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { favorites, toggleFavorite } from "./favorites";

function mockLocalStorage(initial: string | null = null): Storage {
  let data = initial;
  return {
    getItem: vi.fn((key: string) => (key === "favorite-models" ? data : null)),
    setItem: vi.fn((key: string, value: string) => {
      if (key === "favorite-models") data = value;
    }),
    removeItem: vi.fn((key: string) => {
      if (key === "favorite-models") data = null;
    }),
    clear: vi.fn(),
    key: vi.fn(() => null),
    length: 0,
  } as unknown as Storage;
}

function stubBrowserGlobals(ls: Storage) {
  // persistentStore guards persistence behind `typeof window !== "undefined"`,
  // so make window available and point the bare `localStorage` global at the
  // stub to exercise the read/write paths in the Node test environment.
  vi.stubGlobal("window", globalThis);
  vi.stubGlobal("localStorage", ls);
}

describe("favorites store", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    favorites.set([]);
  });

  it("toggles a model on and off", () => {
    stubBrowserGlobals(mockLocalStorage());
    expect(get(favorites)).toEqual([]);

    toggleFavorite("model-a");
    expect(get(favorites)).toEqual(["model-a"]);

    toggleFavorite("model-b");
    expect(get(favorites)).toEqual(["model-a", "model-b"]);

    toggleFavorite("model-a");
    expect(get(favorites)).toEqual(["model-b"]);
  });

  it("persists favorites to localStorage", () => {
    const ls = mockLocalStorage();
    stubBrowserGlobals(ls);

    toggleFavorite("persisted-model");
    expect(ls.setItem).toHaveBeenCalledWith(
      "favorite-models",
      JSON.stringify(["persisted-model"]),
    );
  });

  it("restores favorites from localStorage on creation", async () => {
    const ls = mockLocalStorage(JSON.stringify(["saved-a", "saved-b"]));
    stubBrowserGlobals(ls);

    // Re-create the module so persistentStore reads the stubbed localStorage.
    vi.resetModules();
    const fresh = await import("./favorites");
    expect(get(fresh.favorites)).toEqual(["saved-a", "saved-b"]);
  });
});
