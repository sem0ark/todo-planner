import { useEffect, useState } from 'react';
import { useAuthStore } from '../store/authStore';
import { useScheduleStore } from '../store/scheduleStore';
import { useTemplateStore } from '../store/templateStore';
import { getSchedule, updateWeeklySchedule } from '../services/schedule';
import { getTemplates } from '../services/templates';

const DAYS = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];

export default function WeeklySchedule() {
  const { token } = useAuthStore();
  const { weeklySchedule, setWeeklySchedule, setOverrides } = useScheduleStore();
  const { templates, setTemplates } = useTemplateStore();
  const [localSchedule, setLocalSchedule] = useState<Record<number, number | null>>({});
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    if (token) {
      loadData();
    }
  }, [token]);

  useEffect(() => {
    const schedule: Record<number, number | null> = {};
    weeklySchedule.forEach((slot) => {
      schedule[slot.day_of_week] = slot.day_template_id;
    });
    setLocalSchedule(schedule);
  }, [weeklySchedule]);

  const loadData = async () => {
    if (!token) return;
    try {
      const [scheduleData, templatesData] = await Promise.all([
        getSchedule(token),
        getTemplates(token),
      ]);
      setWeeklySchedule(scheduleData.weekly_schedule);
      setOverrides(scheduleData.overrides);
      setTemplates(templatesData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    }
  };

  const handleChange = (dayOfWeek: number, templateId: string) => {
    const id = templateId === '' ? null : parseInt(templateId);
    setLocalSchedule((prev) => ({ ...prev, [dayOfWeek]: id }));
    setHasChanges(true);
  };

  const handleSave = async () => {
    if (!token) return;
    setIsSaving(true);
    setError(null);

    try {
      const input = {
        weekly_schedule: Array.from({ length: 7 }, (_, i) => ({
          day_of_week: i,
          day_template_id: localSchedule[i] ?? null,
        })),
      };

      const result = await updateWeeklySchedule(token, input);
      setWeeklySchedule(result.weekly_schedule);
      setHasChanges(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save schedule');
    } finally {
      setIsSaving(false);
    }
  };

  const handleReset = () => {
    const schedule: Record<number, number | null> = {};
    weeklySchedule.forEach((slot) => {
      schedule[slot.day_of_week] = slot.day_template_id;
    });
    setLocalSchedule(schedule);
    setHasChanges(false);
  };

  const getTemplateName = (id: number | null) => {
    if (id === null) return 'Unassigned';
    return templates.find((t) => t.id === id)?.name || 'Unknown';
  };

  return (
    <div className="w-full max-w-3xl">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-semibold text-snow">Weekly Schedule</h2>
        {hasChanges && (
          <div className="flex gap-2">
            <button
              onClick={handleReset}
              className="px-4 py-2 text-sm font-semibold text-snow bg-navy/60 border border-slate-grey rounded-lg transition-all duration-micro hover:bg-slate-blue/20"
            >
              Reset
            </button>
            <button
              onClick={handleSave}
              disabled={isSaving}
              className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud disabled:opacity-50"
            >
              {isSaving ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        )}
      </div>

      {error && (
        <div className="mb-4 px-4 py-3 bg-error/10 border border-error rounded-lg text-error text-sm">
          {error}
        </div>
      )}

      <div className="space-y-3">
        {DAYS.map((day, index) => (
          <div
            key={index}
            className="flex items-center gap-4 p-4 bg-slate-blue/10 border border-slate-grey rounded-lg"
          >
            <span className="w-28 text-snow font-medium">{day}</span>
            <select
              value={localSchedule[index] ?? ''}
              onChange={(e) => handleChange(index, e.target.value)}
              className="flex-1 px-4 py-2 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
            >
              <option value="">Unassigned</option>
              {templates.map((template) => (
                <option key={template.id} value={template.id}>
                  {template.name}
                </option>
              ))}
            </select>
            {weeklySchedule[index] && (
              <span className="text-sm text-cloud">
                Current: {getTemplateName(weeklySchedule[index]?.day_template_id)}
              </span>
            )}
          </div>
        ))}
      </div>

      {templates.length === 0 && (
        <p className="mt-4 text-sm text-cloud text-center">
          Create templates first to assign them to days.
        </p>
      )}
    </div>
  );
}
