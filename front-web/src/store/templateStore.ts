import { create } from 'zustand';
import type { Template } from '../services/templates';

interface TemplateState {
  templates: Template[];
  currentTemplate: Template | null;
  isLoading: boolean;
  error: string | null;
  setTemplates: (templates: Template[]) => void;
  setCurrentTemplate: (template: Template | null) => void;
  addTemplate: (template: Template) => void;
  updateTemplate: (id: number, template: Template) => void;
  removeTemplate: (id: number) => void;
  setLoading: (isLoading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useTemplateStore = create<TemplateState>((set) => ({
  templates: [],
  currentTemplate: null,
  isLoading: false,
  error: null,
  setTemplates: (templates) => set({ templates }),
  setCurrentTemplate: (template) => set({ currentTemplate: template }),
  addTemplate: (template) =>
    set((state) => ({ templates: [...state.templates, template] })),
  updateTemplate: (id, template) =>
    set((state) => ({
      templates: state.templates.map((t) => (t.id === id ? template : t)),
      currentTemplate:
        state.currentTemplate?.id === id ? template : state.currentTemplate,
    })),
  removeTemplate: (id) =>
    set((state) => ({
      templates: state.templates.filter((t) => t.id !== id),
      currentTemplate:
        state.currentTemplate?.id === id ? null : state.currentTemplate,
    })),
  setLoading: (isLoading) => set({ isLoading }),
  setError: (error) => set({ error }),
}));
