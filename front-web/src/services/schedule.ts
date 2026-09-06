const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export interface WeeklyScheduleSlot {
  id?: number | null;
  day_of_week: number;
  day_template_id: number | null;
  updated_at?: string | null;
}

export interface ScheduleOverride {
  id: number;
  calendar_date: string;
  day_template_id: number | null;
  created_at: string;
}

export interface Schedule {
  weekly_schedule: WeeklyScheduleSlot[];
  overrides: ScheduleOverride[];
}

export interface UpdateWeeklyScheduleInput {
  weekly_schedule: Array<{
    day_of_week: number;
    day_template_id: number | null;
  }>;
}

export interface UpdateScheduleOverrideInput {
  day_template_id: number;
}

export async function getSchedule(token: string): Promise<Schedule> {
  const response = await fetch(`${API_URL}/schedule`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error("Failed to fetch schedule");
  }

  return response.json();
}

export async function updateWeeklySchedule(
  token: string,
  input: UpdateWeeklyScheduleInput,
): Promise<Schedule> {
  const response = await fetch(`${API_URL}/schedule/weekly`, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error("Failed to update weekly schedule");
  }

  const data = await response.json();
  return {
    weekly_schedule: data.weekly_schedule,
    overrides: [],
  };
}

export async function updateScheduleOverride(
  token: string,
  date: string,
  input: UpdateScheduleOverrideInput,
): Promise<ScheduleOverride> {
  const response = await fetch(`${API_URL}/schedule/overrides/${date}`, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error("Failed to update schedule override");
  }

  return response.json();
}

export async function deleteScheduleOverride(
  token: string,
  date: string,
): Promise<void> {
  const response = await fetch(`${API_URL}/schedule/overrides/${date}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error("Failed to delete schedule override");
}
