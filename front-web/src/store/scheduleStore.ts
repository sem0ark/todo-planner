import { create } from 'zustand';
import type { WeeklyScheduleSlot, ScheduleOverride } from '../services/schedule';

interface ScheduleState {
  weeklySchedule: WeeklyScheduleSlot[];
  overrides: ScheduleOverride[];
  isLoading: boolean;
  error: string | null;
  setWeeklySchedule: (schedule: WeeklyScheduleSlot[]) => void;
  setOverrides: (overrides: ScheduleOverride[]) => void;
  updateSlot: (dayOfWeek: number, templateId: number | null) => void;
  addOrUpdateOverride: (override: ScheduleOverride) => void;
  removeOverride: (date: string) => void;
  setLoading: (isLoading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useScheduleStore = create<ScheduleState>((set) => ({
  weeklySchedule: [],
  overrides: [],
  isLoading: false,
  error: null,
  setWeeklySchedule: (schedule) => set({ weeklySchedule: schedule }),
  setOverrides: (overrides) => set({ overrides }),
  updateSlot: (dayOfWeek, templateId) =>
    set((state) => ({
      weeklySchedule: state.weeklySchedule.map((slot) =>
        slot.day_of_week === dayOfWeek
          ? { ...slot, day_template_id: templateId }
          : slot
      ),
    })),
  addOrUpdateOverride: (override) =>
    set((state) => {
      const existing = state.overrides.find(
        (o) => o.calendar_date === override.calendar_date
      );
      if (existing) {
        return {
          overrides: state.overrides.map((o) =>
            o.calendar_date === override.calendar_date ? override : o
          ),
        };
      }
      return { overrides: [...state.overrides, override] };
    }),
  removeOverride: (date) =>
    set((state) => ({
      overrides: state.overrides.filter((o) => o.calendar_date !== date),
    })),
  setLoading: (isLoading) => set({ isLoading }),
  setError: (error) => set({ error }),
}));
