import type { SnapshotBlock } from "./templates";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export type ActualBlockType = "actual" | "blank" | "untracked";

export interface ActualBlock {
  id: number;
  day_record_id: number;
  category_id: number | null;
  block_type: ActualBlockType;
  start_time: string;
  duration_minutes: number;
  updated_at: string;
}

export interface DayRecord {
  id: number;
  user_id: number;
  day_template_id: number | null;
  snapshot_id: number | null;
  calendar_date: string;
  snapshot_blocks: SnapshotBlock[];
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
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });

  if (!response.ok)
    throw new Error(`Day record request failed (${response.status})`);
  return response.json();
}

export async function getDayRecords(
  token: string,
  from: string,
  to: string,
): Promise<DayRecord[]> {
  const data = await request<{ day_records: DayRecord[] }>(
    token,
    `/day-records?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
  );
  return data.day_records;
}

export function createDayRecord(
  token: string,
  date: string,
): Promise<DayRecord> {
  return request<DayRecord>(token, "/day-records", {
    method: "POST",
    body: JSON.stringify({ calendar_date: date }),
  });
}

export interface ActualBlockInput {
  category_id: number | null;
  block_type: ActualBlockType;
  start_time: string;
  duration_minutes: number;
}

export interface UpdateDayRecordInput {
  actual_blocks: ActualBlockInput[];
}

export interface UpdateDayRecordResponse {
  actual_blocks: ActualBlock[];
  updated_at: string;
}

export function updateDayRecord(
  token: string,
  id: number,
  input: UpdateDayRecordInput,
): Promise<UpdateDayRecordResponse> {
  return request<UpdateDayRecordResponse>(token, `/day-records/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function updateDayRecordTemplate(
  token: string,
  id: number,
  dayTemplateId: number | null,
): Promise<DayRecord> {
  return request<DayRecord>(token, `/day-records/${id}/template`, {
    method: "PUT",
    body: JSON.stringify({ day_template_id: dayTemplateId }),
  });
}
