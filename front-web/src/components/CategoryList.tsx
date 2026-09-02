import { useEffect, useState } from "react";
import { useAuthStore } from "../store/authStore";
import { useCategoryStore } from "../store/categoryStore";
import {
  getCategories,
  createCategory,
  updateCategory,
  deleteCategory,
} from "../services/categories";
import { DEFAULT_PASTEL_COLORS } from "../utils/colors";
import type { Category, CategoryInput } from "../services/categories";

export default function CategoryList() {
  const { token } = useAuthStore();
  const {
    categories,
    setCategories,
    addCategory,
    updateCategory: updateCategoryStore,
    removeCategory,
  } = useCategoryStore();
  const [isCreating, setIsCreating] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [name, setName] = useState("");
  const [color, setColor] = useState(DEFAULT_PASTEL_COLORS[0]);
  const [pomodoroEnabled, setPomodoroEnabled] = useState(false);
  const [workDuration, setWorkDuration] = useState("25");
  const [restDuration, setRestDuration] = useState("5");
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
      setError(
        err instanceof Error ? err.message : "Failed to load categories",
      );
    }
  };

  const handleCreate = async () => {
    if (!token || !name.trim()) return;
    try {
      const input = buildCategoryInput();
      const category = await createCategory(token, input);
      addCategory(category);
      resetForm();
      setIsCreating(false);
      setError(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create category",
      );
    }
  };

  const handleUpdate = async (id: number) => {
    if (!token || !name.trim()) return;
    try {
      const input = buildCategoryInput();
      const category = await updateCategory(token, id, input);
      updateCategoryStore(id, category);
      resetForm();
      setEditingId(null);
      setError(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to update category",
      );
    }
  };

  const handleDelete = async (id: number) => {
    if (!token) return;
    if (
      !confirm(
        "Delete this category? It will be removed from view but historical data will remain.",
      )
    )
      return;
    try {
      await deleteCategory(token, id);
      removeCategory(id);
      setError(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete category",
      );
    }
  };

  const startEdit = (
    id: number,
    currentName: string,
    currentColor: string,
    pomodoroConfig: Category["pomodoro_config"],
  ) => {
    setEditingId(id);
    setName(currentName);
    setColor(currentColor);
    setPomodoroEnabled(pomodoroConfig !== null);
    setWorkDuration(
      pomodoroConfig ? String(pomodoroConfig.work_duration / 60) : "25",
    );
    setRestDuration(
      pomodoroConfig ? String(pomodoroConfig.rest_duration / 60) : "5",
    );
    setIsCreating(false);
  };

  const cancelEdit = () => {
    setEditingId(null);
    setIsCreating(false);
    resetForm();
    setError(null);
  };

  const resetForm = () => {
    setName("");
    setColor(DEFAULT_PASTEL_COLORS[0]);
    setPomodoroEnabled(false);
    setWorkDuration("25");
    setRestDuration("5");
  };

  const buildCategoryInput = (): CategoryInput => ({
    name: name.trim(),
    color,
    pomodoro_config: pomodoroEnabled
      ? {
          work_duration: Number(workDuration) * 60,
          rest_duration: Number(restDuration) * 60,
        }
      : null,
  });

  return (
    <div className="w-full max-w-3xl">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-snow">Categories</h2>
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
        <div className="mb-4 p-4 bg-navy border border-slate-grey/20 rounded-lg">
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-cloud mb-2">
                Name
              </label>
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
              <label className="block text-sm font-medium text-cloud mb-2">
                Color
              </label>
              <div className="flex flex-wrap gap-2 mb-3">
                {DEFAULT_PASTEL_COLORS.map((pastellColor) => (
                  <button
                    key={pastellColor}
                    type="button"
                    onClick={() => setColor(pastellColor)}
                    className={`w-10 h-10 rounded-lg border-2 transition-all duration-micro ${
                      color === pastellColor
                        ? "border-cloud scale-110"
                        : "border-slate-grey hover:border-cloud/50"
                    }`}
                    style={{ backgroundColor: pastellColor }}
                    title={pastellColor}
                  />
                ))}
              </div>
              <input
                type="color"
                value={color}
                onChange={(e) => setColor(e.target.value)}
                className="w-10 h-10 bg-navy/60 border-2 border-slate-grey rounded-lg cursor-pointer"
                title="Custom color"
              />
            </div>
            <div className="border-t border-slate-grey/60 pt-4">
              <label className="flex min-h-12 items-center gap-3 text-sm font-medium text-cloud">
                <input
                  type="checkbox"
                  checked={pomodoroEnabled}
                  onChange={(e) => setPomodoroEnabled(e.target.checked)}
                  className="h-5 w-5 accent-slate-blue"
                />
                Enable Pomodoro timer
              </label>
              {pomodoroEnabled && (
                <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <label className="text-sm text-cloud">
                    Work duration
                    <span className="relative mt-1 block">
                      <input
                        type="number"
                        min="1"
                        step="1"
                        value={workDuration}
                        onChange={(e) => setWorkDuration(e.target.value)}
                        className="w-full rounded-lg border-2 border-slate-grey bg-navy/60 px-3 py-2 pr-16 text-right font-mono tabular-nums text-snow outline-none focus:border-cloud"
                        aria-label="Work duration in minutes"
                      />
                      <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm text-cloud">
                        min
                      </span>
                    </span>
                  </label>
                  <label className="text-sm text-cloud">
                    Rest duration
                    <span className="relative mt-1 block">
                      <input
                        type="number"
                        min="1"
                        step="1"
                        value={restDuration}
                        onChange={(e) => setRestDuration(e.target.value)}
                        className="w-full rounded-lg border-2 border-slate-grey bg-navy/60 px-3 py-2 pr-16 text-right font-mono tabular-nums text-snow outline-none focus:border-cloud"
                        aria-label="Rest duration in minutes"
                      />
                      <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm text-cloud">
                        min
                      </span>
                    </span>
                  </label>
                </div>
              )}
            </div>
            <div className="flex gap-2">
              <button
                onClick={() =>
                  editingId !== null ? handleUpdate(editingId) : handleCreate()
                }
                className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud"
              >
                {editingId !== null ? "Update" : "Create"}
              </button>
              <button
                onClick={cancelEdit}
                className="h-9 px-4 text-sm font-semibold text-cloud bg-transparent border border-slate-grey rounded-lg transition-all duration-micro hover:bg-slate-blue/10"
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
            className="flex items-center gap-4 p-4 bg-navy border border-slate-grey/20 rounded-lg hover:bg-slate-blue/10"
          >
            <div
              className="w-8 h-8 rounded-lg border-2 border-slate-grey"
              style={{ backgroundColor: category.color }}
            />
            <div className="min-w-0 flex-1">
              <span className="block truncate text-snow font-medium">
                {category.name}
              </span>
              {category.pomodoro_config && (
                <span className="mt-1 block font-mono text-sm tabular-nums text-cloud">
                  Pomodoro {category.pomodoro_config.work_duration / 60}/
                  {category.pomodoro_config.rest_duration / 60} min
                </span>
              )}
            </div>
            <button
              onClick={() =>
                startEdit(
                  category.id,
                  category.name,
                  category.color,
                  category.pomodoro_config,
                )
              }
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
          <p className="text-center text-cloud py-8">
            No categories yet. Create one to get started.
          </p>
        )}
      </div>
    </div>
  );
}
