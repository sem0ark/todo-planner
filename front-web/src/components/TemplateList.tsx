import { useEffect } from "react";
import { useAuthStore } from "../store/authStore";
import { useTemplateStore } from "../store/templateStore";
import { getTemplates, deleteTemplate } from "../services/templates";

interface TemplateListProps {
  onEdit: (id: number) => void;
  onCreate: () => void;
}

export default function TemplateList({ onEdit, onCreate }: TemplateListProps) {
  const { token } = useAuthStore();
  const { templates, setTemplates, removeTemplate } = useTemplateStore();

  useEffect(() => {
    if (token) {
      loadTemplates();
    }
  }, [token]);

  const loadTemplates = async () => {
    if (!token) return;
    try {
      const data = await getTemplates(token);
      setTemplates(data);
    } catch (err) {
      console.error("Failed to load templates:", err);
    }
  };

  const handleDelete = async (id: number) => {
    if (!token) return;
    if (
      !confirm(
        "Delete this template? It will be removed from view but historical data will remain.",
      )
    )
      return;
    try {
      await deleteTemplate(token, id);
      removeTemplate(id);
    } catch (err) {
      console.error("Failed to delete template:", err);
    }
  };

  return (
    <div className="w-full max-w-3xl">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-snow">Templates</h2>
        <button
          onClick={onCreate}
          className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud"
        >
          New Template
        </button>
      </div>

      <div className="space-y-2">
        {templates.map((template) => (
          <div
            key={template.id}
            className="flex items-center gap-4 p-4 bg-navy border border-slate-grey/20 rounded-lg hover:bg-slate-blue/10"
          >
            <div className="flex-1">
              <h3 className="text-snow font-medium">{template.name}</h3>
              <p className="text-sm text-cloud">
                {template.planned_blocks.length} blocks
              </p>
            </div>
            <button
              onClick={() => onEdit(template.id)}
              className="px-3 py-1 text-sm text-cloud hover:text-snow transition-colors duration-micro"
            >
              Edit
            </button>
            <button
              onClick={() => handleDelete(template.id)}
              className="px-3 py-1 text-sm text-error hover:text-error/80 transition-colors duration-micro"
            >
              Delete
            </button>
          </div>
        ))}
        {templates.length === 0 && (
          <p className="text-center text-cloud py-8">
            No templates yet. Create one to get started.
          </p>
        )}
      </div>
    </div>
  );
}
