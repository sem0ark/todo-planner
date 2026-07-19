import { create } from 'zustand';
import type { UserSettings } from '../services/settings';

interface SettingsStore {
  settings: UserSettings | null;
  setSettings: (settings: UserSettings) => void;
  clearSettings: () => void;
}

export const useSettingsStore = create<SettingsStore>((set) => ({
  settings: null,
  setSettings: (settings) => set({ settings }),
  clearSettings: () => set({ settings: null }),
}));
