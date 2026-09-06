const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export interface SnapshotBlock {
  id?: number;
  snapshot_id?: number;
  category_id: number;
  start_time: string;
  duration_minutes: number;
}

export interface TemplateSnapshot {
  id: number;
  day_template_id: number;
  user_id: number;
  snapshot_blocks: SnapshotBlock[];
  snapshotted_at: string;
}

export interface Template {
  id: number;
  name: string;
  template_group_id: number | null;
  current_snapshot: TemplateSnapshot | null;
  created_at: string;
  updated_at: string;
}

export interface TemplateInput {
  name: string;
  template_group_id: number | null;
  snapshot_blocks: Omit<SnapshotBlock, "id" | "snapshot_id">[];
}

function normalizeSnapshotBlockStartTime(startTime: string): string {
  const timeParts = startTime.split(":");
  const seconds = (timeParts[2] ?? "00").split(".")[0];
  return `${timeParts[0]}:${timeParts[1]}:${seconds}`;
}

function normalizeTemplateInput(input: TemplateInput): TemplateInput {
  return {
    ...input,
    snapshot_blocks: input.snapshot_blocks.map((block) => ({
      ...block,
      start_time: normalizeSnapshotBlockStartTime(block.start_time),
    })),
  };
}

export async function getTemplates(token: string): Promise<Template[]> {
  const response = await fetch(`${API_URL}/templates`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error("Failed to fetch templates");
  }

  const data = await response.json();
  return data.templates;
}

export async function createTemplate(
  token: string,
  input: TemplateInput,
): Promise<Template> {
  const normalizedInput = normalizeTemplateInput(input);
  const response = await fetch(`${API_URL}/templates`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(normalizedInput),
  });

  if (!response.ok) {
    throw new Error("Failed to create template");
  }

  return response.json();
}

export async function updateTemplate(
  token: string,
  id: number,
  input: TemplateInput,
): Promise<Template> {
  const normalizedInput = normalizeTemplateInput(input);
  const response = await fetch(`${API_URL}/templates/${id}`, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(normalizedInput),
  });

  if (!response.ok) {
    throw new Error("Failed to update template");
  }

  return response.json();
}

export async function deleteTemplate(token: string, id: number): Promise<void> {
  const response = await fetch(`${API_URL}/templates/${id}`, {
    method: "DELETE",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error("Failed to delete template");
  }
}
