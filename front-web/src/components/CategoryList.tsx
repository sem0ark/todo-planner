import { useEffect, useState } from 'react';
import { useAuthStore } from '../store/authStore';
import { useCategoryStore } from '../store/categoryStore';
import { getCategories, createCategory, updateCategory, deleteCategory } from '../services/categories';

export default function CategoryList() {
  const { token } = useAuthStore();
  const { categories, setCategories, addCategory, updateCategory: updateCategoryStore, removeCategory } = useCategoryStore();
  const [isCreating, setIsCreating] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [name, setName] = useState('');
  const [color, setColor] = useState('#003448');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (token) {
      loadCategories();
    }
  }, [token]);

  const loadCategories = async () => {
    if (!token) return;
    try {
      const data = await getCategories(token);
      setCategories(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load categories');
    }
  };

  const handleCreate = async () => {
    if (!token || !name.trim()) return;
    try {
      const category = await createCategory(token, { name: name.trim(), color });
      addCategory(category);
      setName('');
      setColor('#003448');
      setIsCreating(false);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create category');
    }
  };

  const handleUpdate = async (id: number) => {
    if (!token || !name.trim()) return;
    try {
      const category = await updateCategory(token, id, { name: name.trim(), color });
      updateCategoryStore(id, category);
      setEditingId(null);
      setName('');
      setColor('#003448');
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update category');
    }
  };

  const handleDelete = async (id: number) => {
    if (!token) return;
    if (!confirm('Delete this category? It will be removed from view but historical data will remain.')) return;
    try {
      await deleteCategory(token, id);
      removeCategory(id);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete category');
    }
  };

  const startEdit = (id: number, currentName: string, currentColor: string) => {
    setEditingId(id);
    setName(currentName);
    setColor(currentColor);
    setIsCreating(false);
  };

  const cancelEdit = () => {
    setEditingId(null);
    setIsCreating(false);
    setName('');
    setColor('#003448');
    setError(null);
  };

  return (
    <div className="w-full max-w-3xl">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-semibold text-snow">Categories</h2>
        <button
          onClick={() => setIsCreating(true)}
          className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud"
          disabled={isCreating || editingId !== null}
        >
          New Category
        </button>
      </div>

      {error && (
        <div className="mb-4 px-4 py-3 bg-error/10 border border-error rounded-lg text-error text-sm">
          {error}
        </div>
      )}

      {(isCreating || editingId !== null) && (
        <div className="mb-4 p-4 bg-slate-blue/10 border border-slate-grey rounded-lg">
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-cloud mb-2">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full px-4 py-2 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
                placeholder="Category name"
                autoFocus
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-cloud mb-2">Color</label>
              <input
                type="color"
                value={color}
                onChange={(e) => setColor(e.target.value)}
                className="w-20 h-10 bg-navy/60 border-2 border-slate-grey rounded-lg cursor-pointer"
              />
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => (editingId !== null ? handleUpdate(editingId) : handleCreate())}
                className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud"
              >
                {editingId !== null ? 'Update' : 'Create'}
              </button>
              <button
                onClick={cancelEdit}
                className="px-4 py-2 text-sm font-semibold text-snow bg-navy/60 border border-slate-grey rounded-lg transition-all duration-micro hover:bg-slate-blue/20"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="space-y-2">
        {categories.map((category) => (
          <div
            key={category.id}
            className="flex items-center gap-4 p-4 bg-slate-blue/10 border border-slate-grey rounded-lg"
          >
            <div
              className="w-8 h-8 rounded-lg border-2 border-slate-grey"
              style={{ backgroundColor: category.color }}
            />
            <span className="flex-1 text-snow font-medium">{category.name}</span>
            <button
              onClick={() => startEdit(category.id, category.name, category.color)}
              className="px-3 py-1 text-sm text-cloud hover:text-snow transition-colors duration-micro"
              disabled={isCreating || editingId !== null}
            >
              Edit
            </button>
            <button
              onClick={() => handleDelete(category.id)}
              className="px-3 py-1 text-sm text-error hover:text-error/80 transition-colors duration-micro"
              disabled={isCreating || editingId !== null}
            >
              Delete
            </button>
          </div>
        ))}
        {categories.length === 0 && !isCreating && (
          <p className="text-center text-cloud py-8">No categories yet. Create one to get started.</p>
        )}
      </div>
    </div>
  );
}
