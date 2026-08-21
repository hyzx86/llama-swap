import { persistentStore } from "./persistent";

/**
 * Favorited model ids. Persisted to localStorage so favorites survive
 * reloads. Model ids are unique per model and stable across reconnects.
 */
export const favorites = persistentStore<string[]>("favorite-models", []);

/** Toggle the favorite state of a model id. */
export function toggleFavorite(id: string): void {
  favorites.update((favs) => {
    if (favs.includes(id)) {
      return favs.filter((fav) => fav !== id);
    }
    return [...favs, id];
  });
}
