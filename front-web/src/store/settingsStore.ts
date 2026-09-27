import { create } from "zustand";
import { getSettings, type UserSettings } from "../services/settings";

interface SettingsStore {
  settings: UserSettings | null;
  setSettings: (settings: UserSettings) => void;
  clearSettings: () => void;
  loadSettings: (token: string) => Promise<void>;
}

export const useSettingsStore = create<SettingsStore>((set) => ({
  settings: null,
  setSettings: (settings) => set({ settings }),
  clearSettings: () => set({ settings: null }),
  loadSettings: async (token) => {
    set({ settings: await getSettings(token) });
  },
}));
