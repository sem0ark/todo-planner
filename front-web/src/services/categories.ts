const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export interface Category {
  id: number;
  name: string;
  color: string;
  created_at: string;
  updated_at: string;
}

export interface CategoryInput {
  name: string;
  color: string;
}

export async function getCategories(token: string): Promise<Category[]> {
  const response = await fetch(`${API_URL}/categories`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch categories');
  }

  const data = await response.json();
  return data.categories;
}

export async function createCategory(token: string, input: CategoryInput): Promise<Category> {
  const response = await fetch(`${API_URL}/categories`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error('Failed to create category');
  }

  return response.json();
}

export async function updateCategory(token: string, id: number, input: CategoryInput): Promise<Category> {
  const response = await fetch(`${API_URL}/categories/${id}`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error('Failed to update category');
  }

  return response.json();
}

export async function deleteCategory(token: string, id: number): Promise<void> {
  const response = await fetch(`${API_URL}/categories/${id}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to delete category');
  }
}
