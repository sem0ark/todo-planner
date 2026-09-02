import { describe, it, expect, beforeEach } from "vitest";
import { useCategoryStore } from "./categoryStore";
import type { Category } from "../services/categories";

describe("categoryStore", () => {
  const mockCategory: Category = {
    id: 1,
    name: "Work",
    color: "#003448",
    pomodoro_config: null,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };

  beforeEach(() => {
    useCategoryStore.setState({
      categories: [],
      isLoading: false,
      error: null,
    });
  });

  it("should initialize with empty state", () => {
    const state = useCategoryStore.getState();
    expect(state.categories).toEqual([]);
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
  });

  it("should set categories", () => {
    const { setCategories } = useCategoryStore.getState();
    setCategories([mockCategory]);
    expect(useCategoryStore.getState().categories).toEqual([mockCategory]);
  });

  it("should add category", () => {
    const { addCategory } = useCategoryStore.getState();
    addCategory(mockCategory);
    expect(useCategoryStore.getState().categories).toHaveLength(1);
    expect(useCategoryStore.getState().categories[0]).toEqual(mockCategory);
  });

  it("should update category", () => {
    const { setCategories, updateCategory } = useCategoryStore.getState();
    setCategories([mockCategory]);

    const updated = { ...mockCategory, name: "Updated Work" };
    updateCategory(1, updated);

    const state = useCategoryStore.getState();
    expect(state.categories[0].name).toBe("Updated Work");
  });

  it("should remove category", () => {
    const { setCategories, removeCategory } = useCategoryStore.getState();
    setCategories([mockCategory]);

    removeCategory(1);

    expect(useCategoryStore.getState().categories).toHaveLength(0);
  });

  it("should set loading state", () => {
    const { setLoading } = useCategoryStore.getState();
    setLoading(true);
    expect(useCategoryStore.getState().isLoading).toBe(true);

    setLoading(false);
    expect(useCategoryStore.getState().isLoading).toBe(false);
  });

  it("should set error state", () => {
    const { setError } = useCategoryStore.getState();
    setError("Test error");
    expect(useCategoryStore.getState().error).toBe("Test error");

    setError(null);
    expect(useCategoryStore.getState().error).toBeNull();
  });
});
