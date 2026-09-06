import { useEffect, useState } from "react";
import { useAuthStore } from "../store/authStore";
import { useScheduleStore } from "../store/scheduleStore";
import { useTemplateStore } from "../store/templateStore";
import {
  deleteScheduleOverride,
  updateScheduleOverride,
} from "../services/schedule";

export default function ScheduleOverrides() {
  const { token } = useAuthStore();
  const { overrides, addOrUpdateOverride, removeOverride } = useScheduleStore();
  const { templates } = useTemplateStore();
  const [date, setDate] = useState("");
  const [templateId, setTemplateId] = useState<string>("");
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  const getTodayDate = () => {
    const today = new Date();
    return today.toISOString().split("T")[0];
  };

  useEffect(() => {
    setDate(getTodayDate());
  }, []);

  const getTemplateName = (id: number | null) => {
    if (id === null) return "Unassigned";
    return templates.find((t) => t.id === id)?.name || "Unknown";
  };

  const handleSave = async () => {
    if (!token || !date) return;
    setIsSaving(true);
    setError(null);

    try {
      if (templateId === "") {
        setError("Select a template before adding an override");
        return;
      }
      const result = await updateScheduleOverride(token, date, {
        day_template_id: parseInt(templateId, 10),
      });

      addOrUpdateOverride(result);

      setDate(getTodayDate());
      setTemplateId("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save override");
    } finally {
      setIsSaving(false);
    }
  };

  const handleRemove = async (overrideDate: string) => {
    if (!token) return;
    if (!confirm("Remove this schedule override?")) return;

    try {
      await deleteScheduleOverride(token, overrideDate);
      removeOverride(overrideDate);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to remove override",
      );
    }
  };

  const futureOverrides = overrides
    .filter((o) => o.calendar_date >= getTodayDate())
    .sort((a, b) => a.calendar_date.localeCompare(b.calendar_date));

  return (
    <div className="w-full max-w-3xl">
      <h2 className="text-xl font-semibold text-snow mb-6">
        Schedule Overrides
      </h2>

      {error && (
        <div className="mb-4 px-4 py-3 bg-error/10 border border-error rounded-lg text-error text-sm">
          {error}
        </div>
      )}

      <div className="mb-6 p-4 bg-navy border border-slate-grey/20 rounded-lg">
        <h3 className="text-lg font-semibold text-snow mb-4">Add Override</h3>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-cloud mb-2">
              Date
            </label>
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              min={getTodayDate()}
              className="w-full px-4 py-2 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-cloud mb-2">
              Template
            </label>
            <select
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
              className="w-full px-4 py-2 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
            >
              <option value="">Select a template</option>
              {templates.map((template) => (
                <option key={template.id} value={template.id}>
                  {template.name}
                </option>
              ))}
            </select>
          </div>
          <button
            onClick={handleSave}
            disabled={isSaving || !date}
            className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud disabled:opacity-50"
          >
            {isSaving ? "Saving..." : "Add Override"}
          </button>
        </div>
      </div>

      <div>
        <h3 className="text-lg font-semibold text-snow mb-4">
          Future Overrides
        </h3>
        {futureOverrides.length === 0 ? (
          <p className="text-center text-cloud py-8">
            No future overrides scheduled.
          </p>
        ) : (
          <div className="space-y-2">
            {futureOverrides.map((override) => (
              <div
                key={override.id}
                className="flex items-center gap-4 p-4 bg-navy border border-slate-grey/20 rounded-lg hover:bg-slate-blue/10"
              >
                <span className="w-32 text-snow font-mono">
                  {override.calendar_date}
                </span>
                <span className="flex-1 text-snow">
                  {getTemplateName(override.day_template_id)}
                </span>
                <button
                  onClick={() => handleRemove(override.calendar_date)}
                  className="px-3 py-1 text-sm text-error hover:text-error/80 transition-colors duration-micro"
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
