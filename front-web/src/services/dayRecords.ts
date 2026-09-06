const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export type ReviewStatus = "Unreviewed" | "Reviewed" | "Ignored";
export type ActualBlockType = "actual" | "blank" | "untracked";

export interface SnapshotBlock {
  id: number;
  snapshot_id: number;
  category_id: number;
  start_time: string;
  duration_minutes: number;
}

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
  snapshot_id: number | null;
  calendar_date: string;
  review_status: ReviewStatus;
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

export function updateDayRecordStatus(
  token: string,
  id: number,
  status: Exclude<ReviewStatus, "Unreviewed">,
): Promise<DayRecord> {
  return request<DayRecord>(token, `/day-records/${id}/status`, {
    method: "PUT",
    body: JSON.stringify({ review_status: status }),
  });
}
