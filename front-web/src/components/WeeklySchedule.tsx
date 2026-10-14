import { useEffect, useState } from "react";
import { useAuthStore } from "../store/authStore";
import { useScheduleStore } from "../store/scheduleStore";
import { useTemplateStore } from "../store/templateStore";
import { getSchedule, updateWeeklySchedule } from "../services/schedule";
import { getTemplates } from "../services/templates";

const DAYS = [
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
  "Sunday",
];

export default function WeeklySchedule() {
  const { token } = useAuthStore();
  const { weeklySchedule, setWeeklySchedule, setOverrides } =
    useScheduleStore();
  const { templates, isLoading, setTemplates, setLoading } = useTemplateStore();
  const [localSchedule, setLocalSchedule] = useState<
    Record<number, number | null>
  >({});
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
    setLoading(true);
    try {
      const [scheduleData, templatesData] = await Promise.all([
        getSchedule(token),
        getTemplates(token),
      ]);
      setWeeklySchedule(scheduleData.weekly_schedule);
      setOverrides(scheduleData.overrides);
      setTemplates(templatesData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (dayOfWeek: number, templateId: string) => {
    const id = templateId === "" ? null : parseInt(templateId);
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
      setError(err instanceof Error ? err.message : "Failed to save schedule");
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
    if (id === null) return "Unassigned";
    return templates.find((t) => t.id === id)?.name || "Unknown";
  };

  const getCurrentTemplateId = (dayOfWeek: number) =>
    weeklySchedule.find((slot) => slot.day_of_week === dayOfWeek)
      ?.day_template_id ?? null;

  return (
    <div className="w-full max-w-3xl">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-snow">Weekly Schedule</h2>
        {hasChanges && (
          <div className="flex gap-2">
            <button
              onClick={handleReset}
              disabled={isLoading}
              className="h-9 px-4 text-sm font-semibold text-cloud bg-transparent border border-slate-grey rounded-lg transition-all duration-micro hover:bg-slate-blue/10"
            >
              Reset
            </button>
            <button
              onClick={handleSave}
              disabled={isSaving || isLoading}
              className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud disabled:opacity-50"
            >
              {isSaving ? "Saving..." : "Save Changes"}
            </button>
          </div>
        )}
      </div>

      {error && (
        <div className="mb-4 px-4 py-3 bg-error/10 border border-error rounded-lg text-error text-sm">
          {error}
        </div>
      )}

      {isLoading && (
        <div
          className="flex min-h-80 items-center justify-center rounded-lg border border-slate-grey/20 bg-navy/60"
          role="status"
          aria-live="polite"
        >
          <div className="flex items-center gap-3 text-sm text-cloud">
            <span className="h-4 w-4 animate-spin rounded-full border-2 border-cloud/30 border-t-cloud" />
            Loading schedule...
          </div>
        </div>
      )}
      {!isLoading && templates.length > 0 && (
        <div className="space-y-3">
          {DAYS.map((day, index) => {
            const currentTemplateId = getCurrentTemplateId(index);
            const isChanged = localSchedule[index] !== currentTemplateId;

            return (
              <div
                key={day}
                className={`flex items-center gap-4 rounded-lg border bg-navy p-4 hover:bg-slate-blue/10 ${
                  isChanged
                    ? "border-warning bg-warning/5"
                    : "border-slate-grey/20"
                }`}
              >
                <span className="w-28 font-medium text-snow">{day}</span>
                <select
                  value={localSchedule[index] ?? ""}
                  onChange={(event) => handleChange(index, event.target.value)}
                  disabled={isLoading}
                  className="flex-1 rounded-lg border-2 border-slate-grey bg-navy/60 px-4 py-2 text-snow outline-none transition-all duration-micro focus:border-cloud disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">Unassigned</option>
                  {templates.map((template) => (
                    <option key={template.id} value={template.id}>
                      {template.name}
                    </option>
                  ))}
                </select>
                {isChanged && (
                  <span className="whitespace-nowrap text-sm text-warning">
                    Changed from {getTemplateName(currentTemplateId)}
                  </span>
                )}
              </div>
            );
          })}
        </div>
      )}
      {!isLoading && templates.length === 0 && (
        <p className="mt-4 text-center text-sm text-cloud">
          Create templates first to assign them to days.
        </p>
      )}
    </div>
  );
}
