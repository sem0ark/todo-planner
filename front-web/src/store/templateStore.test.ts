import { describe, it, expect, beforeEach } from "vitest";
import { useTemplateStore } from "./templateStore";
import type { Template } from "../services/templates";

describe("templateStore", () => {
  const mockTemplate: Template = {
    id: 1,
    name: "Weekday Schedule",
    template_group_id: null,
    planned_blocks: [
      { id: 1, category_id: 1, start_time: "09:00:00", duration_minutes: 60 },
    ],
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };

  beforeEach(() => {
    useTemplateStore.setState({
      templates: [],
      currentTemplate: null,
      isLoading: false,
      error: null,
    });
  });

  it("should initialize with empty state", () => {
    const state = useTemplateStore.getState();
    expect(state.templates).toEqual([]);
    expect(state.currentTemplate).toBeNull();
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
  });

  it("should set templates", () => {
    const { setTemplates } = useTemplateStore.getState();
    setTemplates([mockTemplate]);
    expect(useTemplateStore.getState().templates).toEqual([mockTemplate]);
  });

  it("should set current template", () => {
    const { setCurrentTemplate } = useTemplateStore.getState();
    setCurrentTemplate(mockTemplate);
    expect(useTemplateStore.getState().currentTemplate).toEqual(mockTemplate);
  });

  it("should add template", () => {
    const { addTemplate } = useTemplateStore.getState();
    addTemplate(mockTemplate);
    expect(useTemplateStore.getState().templates).toHaveLength(1);
    expect(useTemplateStore.getState().templates[0]).toEqual(mockTemplate);
  });

  it("should update template in list", () => {
    const { setTemplates, updateTemplate } = useTemplateStore.getState();
    setTemplates([mockTemplate]);

    const updated = { ...mockTemplate, name: "Updated Schedule" };
    updateTemplate(1, updated);

    const state = useTemplateStore.getState();
    expect(state.templates[0].name).toBe("Updated Schedule");
  });

  it("should update current template when matching id", () => {
    const { setCurrentTemplate, updateTemplate } = useTemplateStore.getState();
    setCurrentTemplate(mockTemplate);

    const updated = { ...mockTemplate, name: "Updated Schedule" };
    updateTemplate(1, updated);

    const state = useTemplateStore.getState();
    expect(state.currentTemplate?.name).toBe("Updated Schedule");
  });

  it("should not update current template when id does not match", () => {
    const { setCurrentTemplate, updateTemplate } = useTemplateStore.getState();
    setCurrentTemplate(mockTemplate);

    const updated = { ...mockTemplate, id: 2, name: "Other Schedule" };
    updateTemplate(2, updated);

    const state = useTemplateStore.getState();
    expect(state.currentTemplate?.name).toBe("Weekday Schedule");
  });

  it("should remove template", () => {
    const { setTemplates, removeTemplate } = useTemplateStore.getState();
    setTemplates([mockTemplate]);

    removeTemplate(1);

    expect(useTemplateStore.getState().templates).toHaveLength(0);
  });

  it("should clear current template when removed", () => {
    const { setTemplates, setCurrentTemplate, removeTemplate } =
      useTemplateStore.getState();
    setTemplates([mockTemplate]);
    setCurrentTemplate(mockTemplate);

    removeTemplate(1);

    expect(useTemplateStore.getState().currentTemplate).toBeNull();
  });

  it("should not clear current template when different id removed", () => {
    const { setCurrentTemplate, removeTemplate } = useTemplateStore.getState();
    setCurrentTemplate(mockTemplate);

    removeTemplate(2);

    expect(useTemplateStore.getState().currentTemplate).toEqual(mockTemplate);
  });

  it("should set loading state", () => {
    const { setLoading } = useTemplateStore.getState();
    setLoading(true);
    expect(useTemplateStore.getState().isLoading).toBe(true);

    setLoading(false);
    expect(useTemplateStore.getState().isLoading).toBe(false);
  });

  it("should set error state", () => {
    const { setError } = useTemplateStore.getState();
    setError("Test error");
    expect(useTemplateStore.getState().error).toBe("Test error");

    setError(null);
    expect(useTemplateStore.getState().error).toBeNull();
  });
});
