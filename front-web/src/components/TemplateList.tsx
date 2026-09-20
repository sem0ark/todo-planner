import { useEffect, useState } from "react";
import { useAuthStore } from "../store/authStore";
import { useTemplateStore } from "../store/templateStore";
import { useCategoryStore } from "../store/categoryStore";
import {
  getTemplates,
  deleteTemplate,
  createTemplate,
  type Template,
} from "../services/templates";
import { getCategories } from "../services/categories";
import TemplateSummary from "./TemplateSummary";

interface TemplateListProps {
  onEdit: (id: number) => void;
  onCreate: () => void;
}

interface TemplateListEntryProps {
  template: Template;
  onEdit: (id: number) => void;
}

function TemplateListEntry({ template, onEdit }: TemplateListEntryProps) {
  const { token } = useAuthStore();
  const { removeTemplate, addTemplate } = useTemplateStore();
  const { categories } = useCategoryStore();
  const [copied, setCopied] = useState(false);

  const handleDelete = async () => {
    if (!token) return;
    if (
      !confirm(
        "Delete this template? It will be removed from view but historical data will remain.",
      )
    )
      return;
    try {
      await deleteTemplate(token, template.id);
      removeTemplate(template.id);
    } catch (err) {
      console.error("Failed to delete template:", err);
    }
  };

  const handleDuplicate = async () => {
    if (!token) return;
    try {
      const newTemplate = await createTemplate(token, {
        name: `${template.name} (Copy)`,
        template_group_id: template.template_group_id,
        plan: template.plan,
      });
      addTemplate(newTemplate);
    } catch (err) {
      console.error("Failed to duplicate template:", err);
    }
  };

  const handleCopySchedule = () => {
    const sortedPlan = [...template.plan].sort((firstBlock, secondBlock) =>
      firstBlock.start_time.localeCompare(secondBlock.start_time),
    );
    const scheduleParts = sortedPlan.map(
      (block) => `${block.start_time} ${block.category_id}`,
    );
    const scheduleString = `SCHEDULE=${scheduleParts.join(", ")}`;
    navigator.clipboard.writeText(scheduleString).catch((err) => {
      console.error("Failed to copy schedule to clipboard:", err);
    });

    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="group flex flex-col p-4 bg-navy border border-slate-grey/20 rounded-lg hover:bg-slate-blue/10">
      <div>
        <h3 className="text-snow font-medium">{template.name}</h3>
        <TemplateSummary plan={template.plan} categories={categories} />
      </div>
      <div className="flex items-center gap-4 mt-3 pt-2 border-t border-slate-grey/10">
        <button
          onClick={handleCopySchedule}
          className="text-sm text-cloud hover:text-snow underline underline-offset-2 transition-colors duration-micro"
        >
          {copied ? "Copied!" : "Copy to Clipboard"}
        </button>
        <button
          onClick={handleDuplicate}
          className="text-sm text-cloud hover:text-snow underline underline-offset-2 transition-colors duration-micro"
        >
          Make a Copy
        </button>
        <button
          onClick={() => onEdit(template.id)}
          className="text-sm text-cloud hover:text-snow underline underline-offset-2 transition-colors duration-micro"
        >
          Edit
        </button>
        <button
          onClick={handleDelete}
          className="text-sm text-error hover:text-error/80 underline underline-offset-2 transition-colors duration-micro"
        >
          Delete
        </button>
      </div>
    </div>
  );
}

export default function TemplateList({ onEdit, onCreate }: TemplateListProps) {
  const { token } = useAuthStore();
  const { templates, setTemplates } = useTemplateStore();
  const { categories, setCategories } = useCategoryStore();

  useEffect(() => {
    if (token) {
      loadTemplates();
    }
  }, [token]);

  useEffect(() => {
    if (!token || categories.length > 0) return;

    getCategories(token)
      .then(setCategories)
      .catch((err) => console.error("Failed to load categories:", err));
  }, [token, categories.length, setCategories]);

  const loadTemplates = async () => {
    if (!token) return;
    try {
      const data = await getTemplates(token);
      setTemplates(data);
    } catch (err) {
      console.error("Failed to load templates:", err);
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
          <TemplateListEntry
            key={template.id}
            template={template}
            onEdit={onEdit}
          />
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
