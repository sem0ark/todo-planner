import type {
  Category,
  PlannedBlock,
  ActualBlock,
  DayRecord,
  Repository,
  DayEvent,
} from "./models";
import { todayString, formatTime, minutesBetween } from "./time";
const categories: Category[] = [
  {
    id: 1,
    name: "Working",
    color: "#2563eb",
    pomodoroConfig: { workDuration: 2700, restDuration: 300 },
  },
  { id: 2, name: "Exercise", color: "#dc2626", pomodoroConfig: null },
  { id: 3, name: "Rest", color: "#0891b2", pomodoroConfig: null },
  {
    id: 4,
    name: "Learning",
    color: "#16a34a",
    pomodoroConfig: { workDuration: 1500, restDuration: 300 },
  },
  { id: 5, name: "Housework", color: "#e9a663", pomodoroConfig: null },
];
const schedule = [
  ["00:00:00", 3],
  ["06:00:00", 2],
  ["07:00:00", 3],
  ["08:00:00", 1],
  ["12:00:00", 3],
  ["13:00:00", 1],
  ["17:00:00", 4],
  ["18:30:00", 5],
  ["19:30:00", 3],
] as const;
let nextId = 5000;
function plannedBlocks(): PlannedBlock[] {
  return schedule.map(([time, categoryId], index) => ({
    id: 100 + index,
    categoryId,
    startTime: time,
    durationMinutes: minutesBetween(
      time,
      schedule[index + 1]?.[0] ?? "24:00:00",
    ),
  }));
}
export function createMockRepository(): Repository {
  let day: DayRecord = {
    id: nextId++,
    calendarDate: todayString(),
    reviewStatus: "unreviewed",
    actualBlocks: [],
  };
  return {
    async fetchCategories() {
      return categories;
    },
    async fetchPlannedBlocks() {
      return plannedBlocks();
    },
    async fetchOrCreateDayRecord(date) {
      if (day.calendarDate !== date)
        day = {
          id: nextId++,
          calendarDate: date,
          reviewStatus: "unreviewed",
          actualBlocks: [],
        };
      return day;
    },
    async submitEvents(_id, events: DayEvent[]) {
      const blocks = [...day.actualBlocks];
      for (const event of events) {
        if (event.eventType !== "transition") continue;
        const time = formatTime(event.occurredAt);
        const last = blocks.at(-1);
        if (last?.durationMinutes === 0)
          blocks[blocks.length - 1] = {
            ...last,
            durationMinutes: minutesBetween(last.startTime, time),
          };
        blocks.push({
          id: nextId++,
          categoryId: event.categoryId,
          blockType: "actual",
          startTime: time,
          durationMinutes: 0,
        });
      }
      day = { ...day, actualBlocks: blocks };
      return { actualBlocks: blocks };
    },
  };
}
