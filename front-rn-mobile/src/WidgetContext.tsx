import React, {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { AppState } from "react-native";
import type { Category, PlannedBlock, Repository } from "./models";
import {
  INITIAL_STATE,
  reduce,
  type WidgetState,
  type Action,
  type Effect,
} from "./machine";
import {
  blockProgress,
  findCurrentBlock,
  findNextBlock,
  secondsOfDay,
  todayString,
} from "./time";
import { haptic } from "./haptics";
import { fireImmediateNotification } from "./notifications";
export interface Widget {
  state: WidgetState;
  currentCategory: Category | null;
  plannedCategory: Category | null;
  plannedBlock: PlannedBlock | null;
  isOnSchedule: boolean;
  progress: number;
  pomodoroProgress: number;
  pomodoroActive: boolean;
  confirm(): void;
  select(id: number): void;
  offset(minutes: number): void;
  returnToPlan(): void;
}
const Context = createContext<Widget | null>(null);
export function useWidget() {
  const value = useContext(Context);
  if (!value) throw new Error("useWidget must be inside WidgetProvider");
  return value;
}
export function WidgetProvider({
  repo,
  children,
}: {
  repo: Repository;
  children: React.ReactNode;
}) {
  const [state, setState] = useState(INITIAL_STATE);
  const stateRef = useRef(INITIAL_STATE);
  const [, rerender] = useState(0);
  const apply = async (step: { state: WidgetState; effects: Effect[] }) => {
    stateRef.current = step.state;
    setState(step.state);
    for (const effect of step.effects) {
      const follow = await execute(effect, repo, step.state);
      if (follow) {
        const next = reduce(stateRef.current, follow);
        stateRef.current = next.state;
        setState(next.state);
      }
    }
  };
  const dispatch = (action: Action) =>
    void apply(reduce(stateRef.current, action));
  useEffect(() => {
    Promise.all([
      repo.fetchCategories(),
      repo.fetchPlannedBlocks(),
      repo.fetchOrCreateDayRecord(todayString()),
    ]).then(([categories, plannedBlocks, day]) =>
      dispatch({
        type: "INITIALIZED",
        categories,
        plannedBlocks,
        dayRecordId: day.id,
        actualBlocks: day.actualBlocks,
        now: Date.now(),
      }),
    );
  }, [repo]);
  useEffect(() => {
    const timer = setInterval(() => {
      rerender((value) => value + 1);
      apply(reduce(stateRef.current, { type: "TICK", now: Date.now() }));
    }, 1000);
    const sub = AppState.addEventListener("change", () =>
      rerender((value) => value + 1),
    );
    return () => {
      clearInterval(timer);
      sub.remove();
    };
  }, []);
  const nowSec = secondsOfDay(Date.now());
  const plannedBlock =
    findCurrentBlock(state.plannedBlocks, nowSec) ??
    findNextBlock(state.plannedBlocks, nowSec);
  const get = (id: number | null) =>
    state.categories.find((category) => category.id === id) ?? null;
  const currentCategory = get(state.currentCategoryId);
  const plannedCategory = get(plannedBlock?.categoryId ?? null);
  const pomodoro = currentCategory?.pomodoroConfig;
  return (
    <Context.Provider
      value={{
        state,
        currentCategory,
        plannedCategory,
        plannedBlock,
        isOnSchedule:
          !currentCategory ||
          !plannedCategory ||
          currentCategory.id === plannedCategory.id,
        progress: plannedBlock ? blockProgress(plannedBlock, nowSec) : 0,
        pomodoroProgress: pomodoro
          ? Math.min(
              state.pomElapsed /
                (state.pomPhase === "work"
                  ? pomodoro.workDuration
                  : pomodoro.restDuration),
              1,
            )
          : 0,
        pomodoroActive: state.phase === "active" && !!pomodoro,
        confirm: () => dispatch({ type: "CONFIRM", now: Date.now() }),
        select: (id) =>
          dispatch({ type: "SELECT", categoryId: id, now: Date.now() }),
        offset: (minutes) =>
          dispatch({ type: "OFFSET", minutes, now: Date.now() }),
        returnToPlan: () => dispatch({ type: "RETURN", now: Date.now() }),
      }}
    >
      {children}
    </Context.Provider>
  );
}
async function execute(
  effect: Effect,
  repo: Repository,
  state: WidgetState,
): Promise<Action | null> {
  if (effect.type === "HAPTIC") {
    await haptic(effect.pattern);
    return null;
  }
  if (effect.type === "NOTIFY_BOUNDARY") {
    await fireImmediateNotification(
      `-> ${effect.categoryName}`,
      "Tap to confirm or switch.",
    );
    return null;
  }
  if (effect.type === "NOTIFY_POMODORO") {
    await fireImmediateNotification("Pomodoro", "Phase complete.");
    return null;
  }
  if (state.dayRecordId == null) return null;
  const response = await repo.submitEvents(state.dayRecordId, [
    {
      eventType:
        effect.type === "LOG_CONFIRMATION" ? "confirmation" : "transition",
      categoryId: effect.categoryId,
      occurredAt: effect.occurredAt ?? Date.now(),
    },
  ]);
  return { type: "SYNC_BLOCKS", blocks: response.actualBlocks };
}
