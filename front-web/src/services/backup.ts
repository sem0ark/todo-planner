const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export async function downloadBackup(token: string): Promise<Blob> {
  const response = await fetch(`${API_URL}/backup`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok)
    throw new Error(`Backup export failed (${response.status})`);
  return response.blob();
}

export async function importBackup(token: string, file: File): Promise<void> {
  const response = await fetch(`${API_URL}/backup`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: await file.text(),
  });
  if (!response.ok)
    throw new Error(`Backup import failed (${response.status})`);
}
