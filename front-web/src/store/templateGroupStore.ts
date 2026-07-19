import { create } from 'zustand';
import type { TemplateGroup } from '../services/templateGroups';

interface TemplateGroupState {
  groups: TemplateGroup[];
  isLoading: boolean;
  error: string | null;
  setGroups: (groups: TemplateGroup[]) => void;
  addGroup: (group: TemplateGroup) => void;
  updateGroup: (id: number, group: TemplateGroup) => void;
  removeGroup: (id: number) => void;
  setLoading: (isLoading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useTemplateGroupStore = create<TemplateGroupState>((set) => ({
  groups: [],
  isLoading: false,
  error: null,
  setGroups: (groups) => set({ groups }),
  addGroup: (group) => set((state) => ({ groups: [...state.groups, group] })),
  updateGroup: (id, group) =>
    set((state) => ({
      groups: state.groups.map((g) => (g.id === id ? group : g)),
    })),
  removeGroup: (id) =>
    set((state) => ({ groups: state.groups.filter((g) => g.id !== id) })),
  setLoading: (isLoading) => set({ isLoading }),
  setError: (error) => set({ error }),
}));
