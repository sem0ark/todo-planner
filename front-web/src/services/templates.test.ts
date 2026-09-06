import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  getTemplates,
  createTemplate,
  updateTemplate,
  deleteTemplate,
} from "./templates";

vi.stubGlobal("fetch", vi.fn());

describe("templates service", () => {
  const token = "test-token";
  const mockTemplate = {
    id: 1,
    name: "Weekday Schedule",
    template_group_id: null,
    current_snapshot: {
      id: 1,
      day_template_id: 1,
      user_id: 1,
      snapshot_blocks: [
        { id: 1, category_id: 1, start_time: "09:00:00", duration_minutes: 60 },
      ],
      snapshotted_at: "2024-01-01T00:00:00Z",
    },
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getTemplates", () => {
    it("should fetch templates successfully", async () => {
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ templates: [mockTemplate] }),
      });

      const result = await getTemplates(token);

      expect(fetch).toHaveBeenCalledWith("http://localhost:8080/templates", {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(result).toEqual([mockTemplate]);
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(getTemplates(token)).rejects.toThrow(
        "Failed to fetch templates",
      );
    });
  });

  describe("createTemplate", () => {
    it("normalizes block start times to HH:MM:SS", async () => {
      const input = {
        name: "Weekday Schedule",
        template_group_id: null,
        snapshot_blocks: [
          {
            category_id: 1,
            start_time: "06:00:00.000000",
            duration_minutes: 120,
          },
        ],
      };
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplate,
      });

      await createTemplate(token, input);

      expect(JSON.parse((fetch as any).mock.calls[0][1].body)).toEqual({
        ...input,
        snapshot_blocks: [
          { ...input.snapshot_blocks[0], start_time: "06:00:00" },
        ],
      });
    });

    it("should create template successfully", async () => {
      const input = {
        name: "Weekday Schedule",
        template_group_id: null,
        snapshot_blocks: [
          { category_id: 1, start_time: "09:00:00", duration_minutes: 60 },
        ],
      };
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockTemplate,
      });

      const result = await createTemplate(token, input);

      expect(fetch).toHaveBeenCalledWith("http://localhost:8080/templates", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(input),
      });
      expect(result).toEqual(mockTemplate);
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(
        createTemplate(token, {
          name: "Test",
          template_group_id: null,
          snapshot_blocks: [],
        }),
      ).rejects.toThrow("Failed to create template");
    });
  });

  describe("updateTemplate", () => {
    it("should update template successfully", async () => {
      const input = {
        name: "Updated Schedule",
        template_group_id: null,
        snapshot_blocks: [
          { category_id: 1, start_time: "09:00:00", duration_minutes: 90 },
        ],
      };
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ ...mockTemplate, ...input }),
      });

      const result = await updateTemplate(token, 1, input);

      expect(fetch).toHaveBeenCalledWith("http://localhost:8080/templates/1", {
        method: "PUT",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(input),
      });
      expect(result.name).toBe(input.name);
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(
        updateTemplate(token, 1, {
          name: "Test",
          template_group_id: null,
          snapshot_blocks: [],
        }),
      ).rejects.toThrow("Failed to update template");
    });
  });

  describe("deleteTemplate", () => {
    it("should delete template successfully", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: true });

      await deleteTemplate(token, 1);

      expect(fetch).toHaveBeenCalledWith("http://localhost:8080/templates/1", {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(deleteTemplate(token, 1)).rejects.toThrow(
        "Failed to delete template",
      );
    });
  });
});
