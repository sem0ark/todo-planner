import { describe, it, expect, beforeEach } from "vitest";
import { useScheduleStore } from "./scheduleStore";
import type {
  WeeklyScheduleSlot,
  ScheduleOverride,
} from "../services/schedule";

describe("scheduleStore", () => {
  const mockSlot: WeeklyScheduleSlot = {
    id: 1,
    day_of_week: 0,
    day_template_id: 1,
    updated_at: "2024-01-01T00:00:00Z",
  };

  const mockOverride: ScheduleOverride = {
    id: 1,
    calendar_date: "2024-12-25",
    day_template_id: 2,
    created_at: "2024-01-01T00:00:00Z",
  };

  beforeEach(() => {
    useScheduleStore.setState({
      weeklySchedule: [],
      overrides: [],
      isLoading: false,
      error: null,
    });
  });

  it("should initialize with empty state", () => {
    const state = useScheduleStore.getState();
    expect(state.weeklySchedule).toEqual([]);
    expect(state.overrides).toEqual([]);
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
  });

  it("should set weekly schedule", () => {
    const { setWeeklySchedule } = useScheduleStore.getState();
    setWeeklySchedule([mockSlot]);
    expect(useScheduleStore.getState().weeklySchedule).toEqual([mockSlot]);
  });

  it("should set overrides", () => {
    const { setOverrides } = useScheduleStore.getState();
    setOverrides([mockOverride]);
    expect(useScheduleStore.getState().overrides).toEqual([mockOverride]);
  });

  it("should update slot by day of week", () => {
    const { setWeeklySchedule, updateSlot } = useScheduleStore.getState();
    setWeeklySchedule([mockSlot]);

    updateSlot(0, 5);

    const state = useScheduleStore.getState();
    expect(state.weeklySchedule[0].day_template_id).toBe(5);
  });

  it("should not update other slots", () => {
    const slot2 = { ...mockSlot, id: 2, day_of_week: 1, day_template_id: 3 };
    const { setWeeklySchedule, updateSlot } = useScheduleStore.getState();
    setWeeklySchedule([mockSlot, slot2]);

    updateSlot(0, 5);

    const state = useScheduleStore.getState();
    expect(state.weeklySchedule[0].day_template_id).toBe(5);
    expect(state.weeklySchedule[1].day_template_id).toBe(3);
  });

  it("should add new override", () => {
    const { addOrUpdateOverride } = useScheduleStore.getState();
    addOrUpdateOverride(mockOverride);

    const state = useScheduleStore.getState();
    expect(state.overrides).toHaveLength(1);
    expect(state.overrides[0]).toEqual(mockOverride);
  });

  it("should update existing override", () => {
    const { setOverrides, addOrUpdateOverride } = useScheduleStore.getState();
    setOverrides([mockOverride]);

    const updated = { ...mockOverride, day_template_id: 5 };
    addOrUpdateOverride(updated);

    const state = useScheduleStore.getState();
    expect(state.overrides).toHaveLength(1);
    expect(state.overrides[0].day_template_id).toBe(5);
  });

  it("should remove override by date", () => {
    const { setOverrides, removeOverride } = useScheduleStore.getState();
    setOverrides([mockOverride]);

    removeOverride("2024-12-25");

    expect(useScheduleStore.getState().overrides).toHaveLength(0);
  });

  it("should not remove non-matching overrides", () => {
    const { setOverrides, removeOverride } = useScheduleStore.getState();
    setOverrides([mockOverride]);

    removeOverride("2024-12-26");

    expect(useScheduleStore.getState().overrides).toHaveLength(1);
  });

  it("should set loading state", () => {
    const { setLoading } = useScheduleStore.getState();
    setLoading(true);
    expect(useScheduleStore.getState().isLoading).toBe(true);

    setLoading(false);
    expect(useScheduleStore.getState().isLoading).toBe(false);
  });

  it("should set error state", () => {
    const { setError } = useScheduleStore.getState();
    setError("Test error");
    expect(useScheduleStore.getState().error).toBe("Test error");

    setError(null);
    expect(useScheduleStore.getState().error).toBeNull();
  });
});
