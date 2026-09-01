const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export interface TemplateGroup {
  id: number;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface TemplateGroupInput {
  name: string;
}

export async function getTemplateGroups(
  token: string,
): Promise<TemplateGroup[]> {
  const response = await fetch(`${API_URL}/template-groups`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error("Failed to fetch template groups");
  }

  const data = await response.json();
  return data.groups;
}

export async function createTemplateGroup(
  token: string,
  input: TemplateGroupInput,
): Promise<TemplateGroup> {
  const response = await fetch(`${API_URL}/template-groups`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error("Failed to create template group");
  }

  return response.json();
}

export async function updateTemplateGroup(
  token: string,
  id: number,
  input: TemplateGroupInput,
): Promise<TemplateGroup> {
  const response = await fetch(`${API_URL}/template-groups/${id}`, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error("Failed to update template group");
  }

  return response.json();
}

export async function deleteTemplateGroup(
  token: string,
  id: number,
): Promise<void> {
  const response = await fetch(`${API_URL}/template-groups/${id}`, {
    method: "DELETE",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error("Failed to delete template group");
  }
}
