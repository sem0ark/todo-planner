import type { Category, PlannedBlock, ActualBlock } from "./models";
import {
  secondsOfDay,
  parseTime,
  findCurrentBlock,
  findNextBlock,
  findCurrentActualBlock,
} from "./time";
export type Phase = "initializing" | "prompted" | "active";
export type PomPhase = "work" | "rest";
export interface WidgetState {
  phase: Phase;
  categories: Category[];
  plannedBlocks: PlannedBlock[];
  dayRecordId: number | null;
  actualBlocks: ActualBlock[];
  currentCategoryId: number | null;
  lastEventTime: number;
  offsetMinutes: number;
  pomPhase: PomPhase;
  pomElapsed: number;
  lastPromptedBlockId: number | null;
}
export const INITIAL_STATE: WidgetState = {
  phase: "initializing",
  categories: [],
  plannedBlocks: [],
  dayRecordId: null,
  actualBlocks: [],
  currentCategoryId: null,
  lastEventTime: Date.now(),
  offsetMinutes: 0,
  pomPhase: "work",
  pomElapsed: 0,
  lastPromptedBlockId: null,
};

export type Action =
  | {
      type: "INITIALIZED";
      categories: Category[];
      plannedBlocks: PlannedBlock[];
      dayRecordId: number;
      actualBlocks: ActualBlock[];
      now: number;
    }
  | { type: "CONFIRM" | "RETURN"; now: number }
  | { type: "SELECT"; categoryId: number; now: number }
  | { type: "OFFSET"; minutes: number; now: number }
  | { type: "TICK"; now: number }
  | { type: "SYNC_BLOCKS"; blocks: ActualBlock[] };

export type Effect =
  | { type: "LOG_TRANSITION"; categoryId: number; occurredAt: number | null }
  | { type: "LOG_CONFIRMATION"; categoryId: number }
  | { type: "NOTIFY_BOUNDARY"; categoryName: string }
  | { type: "NOTIFY_POMODORO" }
  | { type: "HAPTIC"; pattern: "success" | "tap" | "prompt" | "pomodoro" };

export interface Step {
  state: WidgetState;
  effects: Effect[];
}

const category = (s: WidgetState, id: number | null) =>
  id == null ? null : (s.categories.find((c) => c.id === id) ?? null);

const plannedCategory = (s: WidgetState, now: number) =>
  (
    findCurrentBlock(s.plannedBlocks, secondsOfDay(now)) ??
    findNextBlock(s.plannedBlocks, secondsOfDay(now))
  )?.categoryId ?? null;

const promptBlock = (s: WidgetState, now: number) => {
  const block = findCurrentBlock(s.plannedBlocks, secondsOfDay(now));
  return block &&
    block.id !== s.lastPromptedBlockId &&
    secondsOfDay(now) - parseTime(block.startTime) < 60
    ? block
    : null;
};

const transition = (s: WidgetState, id: number, now: number): Step => ({
  state: {
    ...s,
    phase: "active",
    currentCategoryId: id,
    lastEventTime: now,
    offsetMinutes: 0,
    pomPhase: "work",
    pomElapsed: 0,
  },
  effects: [
    { type: "LOG_TRANSITION", categoryId: id, occurredAt: null },
    { type: "HAPTIC", pattern: "success" },
  ],
});

export function reduce(s: WidgetState, a: Action): Step {
  if (a.type === "INITIALIZED") {
    const actual = findCurrentActualBlock(a.actualBlocks, secondsOfDay(a.now));
    return {
      state: {
        ...s,
        phase: "active",
        categories: a.categories,
        plannedBlocks: a.plannedBlocks,
        dayRecordId: a.dayRecordId,
        actualBlocks: a.actualBlocks,
        currentCategoryId:
          actual?.categoryId ??
          plannedCategory(
            { ...s, categories: a.categories, plannedBlocks: a.plannedBlocks },
            a.now,
          ),
        lastEventTime: a.now,
      },
      effects: [],
    };
  }

  if (a.type === "SYNC_BLOCKS")
    return { state: { ...s, actualBlocks: a.blocks }, effects: [] };

  if (a.type === "SELECT") return transition(s, a.categoryId, a.now);
  if (
    a.type === "OFFSET" &&
    s.phase === "active" &&
    s.currentCategoryId != null
  ) {
    const adjusted = s.lastEventTime - a.minutes * 60000;
    return {
      state: {
        ...s,
        offsetMinutes: s.offsetMinutes + a.minutes,
        lastEventTime: adjusted,
      },
      effects: [
        {
          type: "LOG_TRANSITION",
          categoryId: s.currentCategoryId,
          occurredAt: adjusted,
        },
      ],
    };
  }

  if (a.type === "RETURN" && s.phase === "active") {
    const id = plannedCategory(s, a.now);
    return id != null && id !== s.currentCategoryId
      ? transition(s, id, a.now)
      : { state: s, effects: [] };
  }

  if (a.type === "CONFIRM") {
    if (s.phase === "prompted") {
      const id = plannedCategory(s, a.now) ?? s.currentCategoryId;
      return id == null
        ? { state: s, effects: [] }
        : {
            state: {
              ...s,
              phase: "active",
              currentCategoryId: id,
              lastEventTime: a.now,
              pomElapsed: 0,
            },
            effects: [
              { type: "LOG_CONFIRMATION", categoryId: id },
              { type: "HAPTIC", pattern: "tap" },
            ],
          };
    }
    const cat = category(s, s.currentCategoryId);
    if (
      s.phase === "active" &&
      cat?.pomodoroConfig &&
      (s.pomPhase === "rest" || s.pomElapsed >= cat.pomodoroConfig.workDuration)
    )
      return {
        state: {
          ...s,
          pomPhase: s.pomPhase === "rest" ? "work" : "rest",
          pomElapsed: 0,
        },
        effects: [{ type: "HAPTIC", pattern: "pomodoro" }],
      };
  }

  if (a.type === "TICK" && s.phase === "active") {
    const prompt = promptBlock(s, a.now);
    if (prompt) {
      const name = category(s, prompt.categoryId)?.name ?? "Next";
      return {
        state: { ...s, phase: "prompted", lastPromptedBlockId: prompt.id },
        effects: [
          { type: "NOTIFY_BOUNDARY", categoryName: name },
          { type: "HAPTIC", pattern: "prompt" },
        ],
      };
    }
    const cat = category(s, s.currentCategoryId);
    if (!cat?.pomodoroConfig) return { state: s, effects: [] };
    const limit =
      s.pomPhase === "work"
        ? cat.pomodoroConfig.workDuration
        : cat.pomodoroConfig.restDuration;
    const elapsed = s.pomElapsed + 1;
    return {
      state:
        s.pomPhase === "rest" && elapsed > limit * 1.5
          ? { ...s, pomPhase: "work", pomElapsed: 0 }
          : { ...s, pomElapsed: elapsed },
      effects:
        s.pomElapsed < limit && elapsed >= limit
          ? [
              { type: "NOTIFY_POMODORO" },
              { type: "HAPTIC", pattern: "pomodoro" },
            ]
          : [],
    };
  }
  return { state: s, effects: [] };
}
