import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";
import { storage as appStorage } from "../lib/storage";
import type { RecentProvider } from "../types/api";

const MAX_RECENT = 10;

const storage = createJSONStorage(() => ({
  getItem: (key: string) => appStorage.getItem(key),
  setItem: (key: string, value: string) => appStorage.setItem(key, value),
  removeItem: (key: string) => appStorage.deleteItem(key),
}));

interface AppState {
  recentProviders: RecentProvider[];
  addRecent: (provider: { slug: string; name: string; type: string }) => void;
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      recentProviders: [],
      addRecent: (provider) =>
        set((state) => {
          const filtered = state.recentProviders.filter(
            (r) => r.slug !== provider.slug
          );
          const entry: RecentProvider = {
            ...provider,
            visited_at: new Date().toISOString(),
          };
          return {
            recentProviders: [entry, ...filtered].slice(0, MAX_RECENT),
          };
        }),
    }),
    {
      name: "turnero-app-store",
      storage,
    }
  )
);
