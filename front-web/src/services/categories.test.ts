import { describe, it, expect, beforeEach, vi } from 'vitest';
import { getCategories, createCategory, updateCategory, deleteCategory } from './categories';

vi.stubGlobal('fetch', vi.fn());

describe('categories service', () => {
  const token = 'test-token';
  const mockCategory = {
    id: 1,
    name: 'Work',
    color: '#003448',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getCategories', () => {
    it('should fetch categories successfully', async () => {
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ categories: [mockCategory] }),
      });

      const result = await getCategories(token);

      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/categories', {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(result).toEqual([mockCategory]);
    });

    it('should throw error on failed request', async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(getCategories(token)).rejects.toThrow('Failed to fetch categories');
    });
  });

  describe('createCategory', () => {
    it('should create category successfully', async () => {
      const input = { name: 'Work', color: '#003448' };
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockCategory,
      });

      const result = await createCategory(token, input);

      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/categories', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(input),
      });
      expect(result).toEqual(mockCategory);
    });

    it('should throw error on failed request', async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(createCategory(token, { name: 'Work', color: '#003448' })).rejects.toThrow(
        'Failed to create category'
      );
    });
  });

  describe('updateCategory', () => {
    it('should update category successfully', async () => {
      const input = { name: 'Updated Work', color: '#003448' };
      (fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ ...mockCategory, ...input }),
      });

      const result = await updateCategory(token, 1, input);

      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/categories/1', {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(input),
      });
      expect(result).toEqual({ ...mockCategory, ...input });
    });

    it('should throw error on failed request', async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(updateCategory(token, 1, { name: 'Work', color: '#003448' })).rejects.toThrow(
        'Failed to update category'
      );
    });
  });

  describe('deleteCategory', () => {
    it('should delete category successfully', async () => {
      (fetch as any).mockResolvedValueOnce({ ok: true });

      await deleteCategory(token, 1);

      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/categories/1', {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
    });

    it('should throw error on failed request', async () => {
      (fetch as any).mockResolvedValueOnce({ ok: false });

      await expect(deleteCategory(token, 1)).rejects.toThrow('Failed to delete category');
    });
  });
});
