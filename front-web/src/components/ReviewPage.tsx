import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "wouter";
import { useAuthStore } from "../store/authStore";
import { useCategoryStore } from "../store/categoryStore";
import { useTemplateStore } from "../store/templateStore";
import { getCategories } from "../services/categories";
import { getSchedule, type Schedule } from "../services/schedule";
import { getTemplates } from "../services/templates";
import { getDayRecords, type DayRecord } from "../services/dayRecords";
import { useSettingsStore } from "../store/settingsStore";
import { DraggableColumn, type LayoutItem } from "./DraggableColumn";
import { downloadBackup, importBackup } from "../services/backup";

const DAY_NAMES = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
const GRID_UNIT = 1;

function mondayOf(date: Date) {
  const result = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const day = result.getDay();
  result.setDate(result.getDate() - (day === 0 ? 6 : day - 1));
  return result;
}

function dateValue(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

function addDays(date: Date, days: number) {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

function minutes(time: string) {
  const [hours, mins] = time.split(":").map(Number);
  return hours * 60 + mins;
}

function visibleMinutes(startTime: string, endTime: string) {
  const startMinutes = minutes(startTime);
  const endMinutes = minutes(endTime);
  return Math.max(0, endMinutes - startMinutes);
}

function isWithinDayRange(
  blockTime: string,
  rangeStartTime: string,
  rangeEndTime: string,
) {
  const blockMinutes = minutes(blockTime);
  const rangeStartMinutes = minutes(rangeStartTime);
  const rangeEndMinutes = minutes(rangeEndTime);
  return blockMinutes >= rangeStartMinutes && blockMinutes < rangeEndMinutes;
}

interface ReviewDay {
  date: string;
  record: DayRecord | null;
  plan: DayRecord["plan"];
  actual: DayRecord["actual"];
}

export default function ReviewPage() {
  const { token } = useAuthStore();
  const { categories, setCategories } = useCategoryStore();
  const { templates, setTemplates } = useTemplateStore();
  const { settings } = useSettingsStore();
  const [weekStart, setWeekStart] = useState(() => mondayOf(new Date()));
  const [schedule, setSchedule] = useState<Schedule | null>(null);
  const [records, setRecords] = useState<DayRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [backupBusy, setBackupBusy] = useState(false);
  const [dataRevision, setDataRevision] = useState(0);
  const fileInputReference = useRef<HTMLInputElement>(null);

  const from = dateValue(weekStart);
  const to = dateValue(addDays(weekStart, 6));

  const exportBackup = async () => {
    if (!token) return;
    setBackupBusy(true);
    setError(null);
    try {
      const blob = await downloadBackup(token);
      const downloadUrl = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = downloadUrl;
      link.download = "todo-planner-backup.json";
      link.click();
      URL.revokeObjectURL(downloadUrl);
    } catch (reason) {
      setError(
        reason instanceof Error ? reason.message : "Failed to export backup",
      );
    } finally {
      setBackupBusy(false);
    }
  };

  const selectBackup = () => fileInputReference.current?.click();

  const handleBackupSelected = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!token || !file) return;
    setBackupBusy(true);
    setError(null);
    try {
      await importBackup(token, file);
      setDataRevision((revision) => revision + 1);
      setError("Backup imported successfully");
    } catch (reason) {
      setError(
        reason instanceof Error ? reason.message : "Failed to import backup",
      );
    } finally {
      setBackupBusy(false);
    }
  };

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    const templatesRequest = templates.length
      ? Promise.resolve(templates)
      : getTemplates(token);
    const categoriesRequest = categories.length
      ? Promise.resolve(categories)
      : getCategories(token);
    Promise.all([
      getDayRecords(token, from, to),
      getSchedule(token),
      templatesRequest,
      categoriesRequest,
    ])
      .then(([nextRecords, nextSchedule, nextTemplates, nextCategories]) => {
        if (cancelled) return;
        setRecords(nextRecords);
        setSchedule(nextSchedule);
        setTemplates(nextTemplates);
        setCategories(nextCategories);
      })
      .catch(
        (reason) =>
          !cancelled &&
          setError(
            reason instanceof Error
              ? reason.message
              : "Failed to load review data",
          ),
      )
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [token, from, to, dataRevision]);

  const days = useMemo<ReviewDay[]>(
    () =>
      Array.from({ length: 7 }, (_, index) => {
        const date = dateValue(addDays(weekStart, index));
        const record =
          records.find((item) => item.calendar_date === date) || null;
        const weekly = schedule?.weekly_schedule.find(
          (slot) => slot.day_of_week === index,
        );
        const override = schedule?.overrides.find(
          (item) => item.calendar_date === date,
        );
        const templateId = override?.day_template_id ?? weekly?.day_template_id;
        const template = templates.find((item) => item.id === templateId);
        return {
          date,
          record,
          plan: record?.plan ?? template?.plan ?? [],
          actual: record?.actual ?? [],
        };
      }),
    [weekStart, records, schedule, templates],
  );

  if (!settings) {
    return (
      <section className="space-y-4">
        <p className="text-sm text-cloud">Loading user settings...</p>
      </section>
    );
  }

  const dayRangeStartMinutes = minutes(settings.day_range_start_time);
  const dayRangeEndMinutes = minutes(settings.day_range_end_time);
  const dayRangeMinutes = visibleMinutes(
    settings.day_range_start_time,
    settings.day_range_end_time,
  );
  const dayRangeHours = Math.max(1, Math.ceil(dayRangeMinutes / 60));
  const isBlockInDayRange = (startTime: string) =>
    isWithinDayRange(
      startTime,
      settings.day_range_start_time,
      settings.day_range_end_time,
    );

  const toItems = (
    blocks: ReviewDay["plan"] | ReviewDay["actual"],
  ): LayoutItem[] =>
    blocks.map((block, index) => ({
      id: `${block.start_time}-${index}`,
      offset: minutes(block.start_time) - dayRangeStartMinutes,
      size: Math.min(
        block.duration_minutes,
        Math.max(0, dayRangeEndMinutes - minutes(block.start_time)),
      ),
      categoryId: block.category_id,
    }));

  const visiblePlanBlocks = (day: ReviewDay) =>
    day.plan.filter((block) => {
      const category = categories.find((item) => item.id === block.category_id);
      return (
        isBlockInDayRange(block.start_time) &&
        !(category?.name === "Rest" && minutes(block.start_time) < 7 * 60)
      );
    });

  const visibleActualBlocks = (day: ReviewDay) =>
    day.actual.filter((block) => isBlockInDayRange(block.start_time));

  const renderBlock = (
    item: LayoutItem,
    elementType: "idle" | "dragging" | "overlay" = "idle",
  ) => {
    const category = categories.find((value) => value.id === item.categoryId);
    return (
      <div
        className="h-full rounded-sm px-1 text-[10px] text-snow overflow-hidden"
        style={{ backgroundColor: category?.color || "#003448" }}
      >
        {elementType !== "overlay" && category?.name}
      </div>
    );
  };

  const renderPlannedBlock = (item: LayoutItem) => renderBlock(item, "overlay");

  return (
    <section className="space-y-4">
      <div className="flex items-center gap-3">
        <button
          aria-label="Previous week"
          onClick={() => setWeekStart((date) => addDays(date, -7))}
          className="px-3 py-1 text-cloud hover:text-snow"
        >
          ←
        </button>
        <h2 className="text-xl font-semibold text-snow">Week of {from}</h2>
        <button
          aria-label="Next week"
          onClick={() => setWeekStart((date) => addDays(date, 7))}
          className="px-3 py-1 text-cloud hover:text-snow"
        >
          →
        </button>
        <button
          onClick={() => setWeekStart(mondayOf(new Date()))}
          className="ml-2 px-3 py-1 text-sm text-cloud border border-slate-grey/30 rounded-lg"
        >
          Today
        </button>
        <div className="ml-auto flex items-center gap-2">
          <input
            ref={fileInputReference}
            type="file"
            accept="application/json,.json"
            className="hidden"
            onChange={handleBackupSelected}
          />
          <button
            type="button"
            disabled={backupBusy}
            onClick={selectBackup}
            className="rounded-lg border border-slate-grey/30 px-3 py-1 text-sm text-cloud disabled:opacity-50"
          >
            Import backup
          </button>
          <button
            type="button"
            disabled={backupBusy}
            onClick={exportBackup}
            className="rounded-lg border border-slate-grey/30 px-3 py-1 text-sm text-cloud disabled:opacity-50"
          >
            Export backup
          </button>
        </div>
        <Link
          href="/schedule"
          className="text-sm text-slate-blue hover:text-cloud"
        >
          Schedule
        </Link>
      </div>
      {error && <p className="text-sm text-error">{error}</p>}
      <div className="grid grid-cols-[64px_repeat(7,minmax(100px,1fr))] overflow-auto border border-slate-grey/20">
        <div />
        {days.map((day, index) => (
          <div
            key={day.date}
            className="border-l border-slate-grey/20 p-2 text-center text-sm text-cloud"
          >
            {DAY_NAMES[index]}
            <br />
            <span className="font-mono text-xs">{day.date}</span>
            <br />
            <Link
              href={`/review/${day.date}`}
              className="text-xs text-slate-blue hover:text-cloud"
            >
              Edit
            </Link>
          </div>
        ))}
        <div className="relative" style={{ height: `${dayRangeHours * 60}px` }}>
          {Array.from({ length: dayRangeHours + 1 }, (_, hour) => {
            const displayedHour = Math.floor(dayRangeStartMinutes / 60 + hour);
            return (
              <div
                key={hour}
                className={`${hour === dayRangeHours ? "h-0" : "h-[60px]"} border-t border-slate-grey/10 text-right pr-2 text-xs font-mono text-slate-blue`}
              >
                {String(displayedHour).padStart(2, "0")}:00
              </div>
            );
          })}
        </div>
        {days.map((day) => (
          <div
            key={day.date}
            className="relative border-l border-slate-grey/20"
            style={{ height: `${dayRangeHours * 60}px` }}
          >
            {day.record && !day.record.timezone_offset_locked && (
              <span className="absolute right-1 top-1 z-10 rounded bg-amber-500/80 px-1 text-[10px] text-snow">
                Time zone not yet recorded for this day
              </span>
            )}
            {loading ? (
              <div className="absolute inset-x-1 top-20 h-64 rounded bg-slate-blue/20 animate-pulse" />
            ) : (
              <>
                <DraggableColumn
                  items={toItems(visiblePlanBlocks(day))}
                  gridUnit={GRID_UNIT}
                  baseWidth="28%"
                  onChange={() => undefined}
                  editable={false}
                  renderItem={renderPlannedBlock}
                  containerClassName="!absolute inset-y-0 left-0 w-[40%] bg-transparent"
                  itemClassName="px-1"
                />
                <DraggableColumn
                  items={toItems(visibleActualBlocks(day))}
                  gridUnit={GRID_UNIT}
                  baseWidth="72%"
                  onChange={() => undefined}
                  editable={false}
                  renderItem={renderBlock}
                  containerClassName="!absolute inset-y-0 right-0 w-[60%] bg-transparent"
                  itemClassName="px-1"
                />
              </>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}
