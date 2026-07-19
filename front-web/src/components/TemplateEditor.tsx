import { useEffect, useState } from 'react';
import { useAuthStore } from '../store/authStore';
import { useCategoryStore } from '../store/categoryStore';
import { useTemplateStore } from '../store/templateStore';
import { createTemplate, updateTemplate, getTemplates } from '../services/templates';
import type { PlannedBlock } from '../services/templates';
import { getCategories } from '../services/categories';
import TimelineEditor from './TimelineEditor';

interface TemplateEditorProps {
  templateId: number | null;
  onClose: () => void;
}

export default function TemplateEditor({ templateId, onClose }: TemplateEditorProps) {
  const { token } = useAuthStore();
  const { categories, setCategories } = useCategoryStore();
  const { templates, addTemplate, updateTemplate: updateTemplateStore, setTemplates } = useTemplateStore();

  const [name, setName] = useState('');
  const [blocks, setBlocks] = useState<PlannedBlock[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (token) {
      loadData();
    }
  }, [token]);

  useEffect(() => {
    if (templateId && templates.length > 0) {
      const template = templates.find((t) => t.id === templateId);
      if (template) {
        setName(template.name);
        setBlocks(template.planned_blocks);
      }
    } else {
      setName('');
      setBlocks([]);
    }
  }, [templateId, templates]);

  const loadData = async () => {
    if (!token) return;
    try {
      const [categoriesData, templatesData] = await Promise.all([
        getCategories(token),
        getTemplates(token),
      ]);
      setCategories(categoriesData);
      setTemplates(templatesData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    }
  };

  const handleSave = async () => {
    if (!token || !name.trim()) {
      setError('Template name is required');
      return;
    }

    setIsSaving(true);
    setError(null);

    try {
      const input = {
        name: name.trim(),
        template_group_id: null,
        planned_blocks: blocks.map(({ id, ...block }) => block),
      };

      if (templateId) {
        const updated = await updateTemplate(token, templateId, input);
        updateTemplateStore(templateId, updated);
      } else {
        const created = await createTemplate(token, input);
        addTemplate(created);
      }

      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save template');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-semibold text-snow">
          {templateId ? 'Edit Template' : 'New Template'}
        </h2>
        <button
          onClick={onClose}
          className="px-3 py-1 text-sm text-cloud hover:text-snow transition-colors duration-micro"
        >
          Close
        </button>
      </div>

      {error && (
        <div className="mb-4 px-4 py-3 bg-error/10 border border-error rounded-lg text-error text-sm">
          {error}
        </div>
      )}

      <div className="space-y-6">
        <div>
          <label className="block text-sm font-medium text-cloud mb-2">Template Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full px-4 py-2 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
            placeholder="e.g., Weekday Work Schedule"
            autoFocus
          />
        </div>

        <TimelineEditor blocks={blocks} categories={categories} onChange={setBlocks} />

        <div className="flex gap-3 pt-4">
          <button
            onClick={handleSave}
            disabled={isSaving || !name.trim()}
            className="px-6 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud disabled:opacity-50"
          >
            {isSaving ? 'Saving...' : templateId ? 'Update' : 'Create'}
          </button>
          <button
            onClick={onClose}
            className="px-6 py-2 text-sm font-semibold text-snow bg-navy/60 border border-slate-grey rounded-lg transition-all duration-micro hover:bg-slate-blue/20"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
