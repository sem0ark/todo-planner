import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  getTemplateGroups,
  createTemplateGroup,
  updateTemplateGroup,
  deleteTemplateGroup,
} from "./templateGroups";

vi.stubGlobal("fetch", vi.fn());

describe("templateGroups service", () => {
  const token = "test-token";
  const mockGroup = {
    id: 1,
    name: "Work Templates",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getTemplateGroups", () => {
    it("should fetch template groups successfully", async () => {
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ groups: [mockGroup] }),
      });

      const result = await getTemplateGroups(token);

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/template-groups",
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      expect(result).toEqual([mockGroup]);
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(getTemplateGroups(token)).rejects.toThrow(
        "Failed to fetch template groups",
      );
    });
  });

  describe("createTemplateGroup", () => {
    it("should create template group successfully", async () => {
      const input = { name: "Work Templates" };
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockGroup,
      });

      const result = await createTemplateGroup(token, input);

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/template-groups",
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify(input),
        },
      );
      expect(result).toEqual(mockGroup);
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(
        createTemplateGroup(token, { name: "Test" }),
      ).rejects.toThrow("Failed to create template group");
    });
  });

  describe("updateTemplateGroup", () => {
    it("should update template group successfully", async () => {
      const input = { name: "Updated Templates" };
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ ...mockGroup, ...input }),
      });

      const result = await updateTemplateGroup(token, 1, input);

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/template-groups/1",
        {
          method: "PUT",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify(input),
        },
      );
      expect(result).toEqual({ ...mockGroup, ...input });
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(
        updateTemplateGroup(token, 1, { name: "Test" }),
      ).rejects.toThrow("Failed to update template group");
    });
  });

  describe("deleteTemplateGroup", () => {
    it("should delete template group successfully", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: true });

      await deleteTemplateGroup(token, 1);

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/template-groups/1",
        {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
        },
      );
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(deleteTemplateGroup(token, 1)).rejects.toThrow(
        "Failed to delete template group",
      );
    });
  });
});
