import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  getSchedule,
  updateWeeklySchedule,
  updateScheduleOverride,
} from "./schedule";

vi.stubGlobal("fetch", vi.fn());

describe("schedule service", () => {
  const token = "test-token";
  const mockSchedule = {
    weekly_schedule: [
      {
        id: 1,
        day_of_week: 0,
        day_template_id: 1,
        updated_at: "2024-01-01T00:00:00Z",
      },
      {
        id: 2,
        day_of_week: 1,
        day_template_id: null,
        updated_at: "2024-01-01T00:00:00Z",
      },
    ],
    overrides: [
      {
        id: 1,
        calendar_date: "2024-12-25",
        day_template_id: 2,
        created_at: "2024-01-01T00:00:00Z",
      },
    ],
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getSchedule", () => {
    it("should fetch schedule successfully", async () => {
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockSchedule,
      });

      const result = await getSchedule(token);

      expect(fetch).toHaveBeenCalledWith("http://localhost:8080/schedule", {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(result).toEqual(mockSchedule);
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(getSchedule(token)).rejects.toThrow(
        "Failed to fetch schedule",
      );
    });
  });

  describe("updateWeeklySchedule", () => {
    it("should update weekly schedule successfully", async () => {
      const input = {
        weekly_schedule: [
          { day_of_week: 0, day_template_id: 1 },
          { day_of_week: 1, day_template_id: 2 },
        ],
      };
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ weekly_schedule: mockSchedule.weekly_schedule }),
      });

      const result = await updateWeeklySchedule(token, input);

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/schedule/weekly",
        {
          method: "PUT",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify(input),
        },
      );
      expect(result.weekly_schedule).toEqual(mockSchedule.weekly_schedule);
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(
        updateWeeklySchedule(token, { weekly_schedule: [] }),
      ).rejects.toThrow("Failed to update weekly schedule");
    });
  });

  describe("updateScheduleOverride", () => {
    it("should create or update override successfully", async () => {
      const date = "2024-12-25";
      const input = { day_template_id: 1 };
      const mockOverride = {
        id: 1,
        calendar_date: date,
        day_template_id: 1,
        created_at: "2024-01-01T00:00:00Z",
      };
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockOverride,
      });

      const result = await updateScheduleOverride(token, date, input);

      expect(fetch).toHaveBeenCalledWith(
        `http://localhost:8080/schedule/overrides/${date}`,
        {
          method: "PUT",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify(input),
        },
      );
      expect(result).toEqual(mockOverride);
    });

    it("should throw error on failed request", async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(
        updateScheduleOverride(token, "2024-12-25", { day_template_id: 1 }),
      ).rejects.toThrow("Failed to update schedule override");
    });
  });
});
