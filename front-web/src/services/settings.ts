const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export interface UserSettings {
  id: number;
  user_id: number;
  day_boundary_time: string;
  created_at: string;
  updated_at: string;
}

export interface UserSettingsInput {
  day_boundary_time: string;
}

export async function getSettings(token: string): Promise<UserSettings> {
  const response = await fetch(`${API_URL}/settings`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch settings');
  }

  return response.json();
}

export async function updateSettings(token: string, input: UserSettingsInput): Promise<UserSettings> {
  const response = await fetch(`${API_URL}/settings`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error('Failed to update settings');
  }

  return response.json();
}
