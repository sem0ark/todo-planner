export interface PomodoroConfig {
  workDuration: number;
  restDuration: number;
}

export interface Category {
  id: number;
  name: string;
  color: string;
  pomodoroConfig: PomodoroConfig | null;
}

export interface PlannedBlock {
  id: number;
  categoryId: number;
  startTime: string;
  durationMinutes: number;
}

export interface ActualBlock {
  id: number;
  categoryId: number | null;
  blockType: string;
  startTime: string;
  durationMinutes: number;
}

export interface DayRecord {
  id: number;
  calendarDate: string;
  reviewStatus: string;
  actualBlocks: ActualBlock[];
}

export interface DayEvent {
  eventType: "confirmation" | "transition";
  categoryId: number;
  occurredAt: number;
}

export interface DayEventsResponse {
  actualBlocks: ActualBlock[];
}

export interface Repository {
  fetchCategories(): Promise<Category[]>;
  fetchPlannedBlocks(): Promise<PlannedBlock[]>;
  fetchOrCreateDayRecord(date: string): Promise<DayRecord>;
  submitEvents(id: number, events: DayEvent[]): Promise<DayEventsResponse>;
}
