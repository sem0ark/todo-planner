import type { SnapshotBlock } from "./templates";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export type ActualBlockType = "actual" | "blank" | "untracked";

export interface ActualBlock {
  category_id: number | null;
  block_type: ActualBlockType;
  start_time: string;
  duration_minutes: number;
  is_open: boolean;
}

export interface DaySnapshot {
  snapshot_id: number;
  snapshotted_at: string;
  blocks: SnapshotBlock[];
}

export interface DayRecord {
  calendar_date: string;
  day_template_id: number | null;
  snapshot: DaySnapshot | null;
  actual_blocks: ActualBlock[];
  created_at: string;
  updated_at: string;
}

async function request<T>(
  token: string,
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });

  if (!response.ok) {
    throw new Error(`Day record request failed (${response.status})`);
  }
  return response.json();
}

export async function getDayRecords(
  token: string,
  from: string,
  to: string,
): Promise<DayRecord[]> {
  const data = await request<{ day_records: DayRecord[] }>(
    token,
    `/days?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
  );
  return data.day_records;
}

export function getDayRecord(token: string, date: string): Promise<DayRecord> {
  return request<DayRecord>(token, `/days/${encodeURIComponent(date)}`);
}

export function createDayRecord(
  token: string,
  date: string,
): Promise<DayRecord> {
  return request<DayRecord>(token, `/days/${encodeURIComponent(date)}`, {
    method: "POST",
  });
}

export interface ActualBlockInput {
  category_id: number | null;
  block_type: "actual" | "blank";
  start_time: string;
  duration_minutes: number;
}

export interface UpdateDayBlocksInput {
  actual_blocks: ActualBlockInput[];
}

export function updateDayBlocks(
  token: string,
  date: string,
  input: UpdateDayBlocksInput,
): Promise<DayRecord> {
  return request<DayRecord>(token, `/days/${encodeURIComponent(date)}/blocks`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function updateDayRecordTemplate(
  token: string,
  date: string,
  dayTemplateId: number | null,
): Promise<DayRecord> {
  return request<DayRecord>(
    token,
    `/days/${encodeURIComponent(date)}/template`,
    {
      method: "PUT",
      body: JSON.stringify({ day_template_id: dayTemplateId }),
    },
  );
}

export type DayEventType = "transition" | "confirmation" | "amendment";

export interface DayEventInput {
  client_event_id: string;
  event_type: DayEventType;
  category_id: number | null;
  occurred_at: string;
  target_client_event_id?: string;
  corrected_at?: string;
}

export interface AppendDayEventsInput {
  device_id: number;
  events: DayEventInput[];
}

export interface AppendDayEventsResponse extends DayRecord {
  accepted_events: Array<DayEventInput & { occurred_at: string }>;
  duplicate_client_event_ids: string[];
}

export function appendDayEvents(
  token: string,
  date: string,
  input: AppendDayEventsInput,
): Promise<AppendDayEventsResponse> {
  return request<AppendDayEventsResponse>(
    token,
    `/days/${encodeURIComponent(date)}/events`,
    {
      method: "POST",
      body: JSON.stringify(input),
    },
  );
}
