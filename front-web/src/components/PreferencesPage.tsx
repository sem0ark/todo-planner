import { useEffect, useState, type FormEvent } from "react";
import { useAuthStore } from "../store/authStore";
import { updateSettings } from "../services/settings";
import { useSettingsStore } from "../store/settingsStore";

function toTimeInputValue(scheduleTime: string) {
  return scheduleTime.slice(0, 5);
}

function toScheduleTime(timeValue: string) {
  return `${timeValue}:00`;
}

function timeToMinutes(timeValue: string) {
  const [hours, minutes] = timeValue.split(":").map(Number);
  return hours * 60 + minutes;
}

export default function PreferencesPage() {
  const { token } = useAuthStore();
  const { settings, setSettings } = useSettingsStore();
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (!settings) return;
    setStartTime(toTimeInputValue(settings.day_range_start_time));
    setEndTime(toTimeInputValue(settings.day_range_end_time));
  }, [settings]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!token || !startTime || !endTime) return;

    if (timeToMinutes(endTime) <= timeToMinutes(startTime)) {
      setError("End time must be after start time.");
      setSaved(false);
      return;
    }

    setIsSaving(true);
    setError(null);
    setSaved(false);

    try {
      const updatedSettings = await updateSettings(token, {
        day_range_start_time: toScheduleTime(startTime),
        day_range_end_time: toScheduleTime(endTime),
      });
      setSettings(updatedSettings);
      setSaved(true);
    } catch (saveError) {
      setError(
        saveError instanceof Error
          ? saveError.message
          : "Failed to save settings",
      );
    } finally {
      setIsSaving(false);
    }
  };

  if (!settings) {
    return <p className="text-sm text-cloud">Loading settings...</p>;
  }

  return (
    <div className="w-full max-w-3xl">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-snow">Preferences</h2>
      </div>

      {error && (
        <div className="mb-4 px-4 py-3 bg-error/10 border border-error rounded-lg text-error text-sm">
          {error}
        </div>
      )}

      {saved && (
        <div className="mb-4 px-4 py-3 bg-success/10 border border-success rounded-lg text-success text-sm">
          Settings saved.
        </div>
      )}

      <form
        onSubmit={handleSubmit}
        className="mb-4 p-4 bg-navy border border-slate-grey/20 rounded-lg"
      >
        <div className="space-y-4">
          <div>
            <h3 className="text-sm font-medium text-cloud">Tracking day</h3>
            <p className="mt-1 text-sm text-slate-blue">
              Set the hours shown in review and timeline views.
            </p>
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label className="text-sm text-cloud">
              Starts at
              <input
                type="time"
                step="900"
                value={startTime}
                onChange={(event) => {
                  setStartTime(event.target.value);
                  setError(null);
                  setSaved(false);
                }}
                className="mt-1 w-full px-3 py-2 font-mono tabular-nums text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none focus:border-cloud"
                required
              />
            </label>
            <label className="text-sm text-cloud">
              Ends at
              <input
                type="time"
                step="900"
                value={endTime}
                onChange={(event) => {
                  setEndTime(event.target.value);
                  setError(null);
                  setSaved(false);
                }}
                className="mt-1 w-full px-3 py-2 font-mono tabular-nums text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none focus:border-cloud"
                required
              />
            </label>
          </div>

          <button
            type="submit"
            disabled={isSaving}
            className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud disabled:opacity-50"
          >
            {isSaving ? "Saving..." : "Save Settings"}
          </button>
        </div>
      </form>
    </div>
  );
}
