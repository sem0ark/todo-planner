import { useEffect, useMemo, useState } from "react";
import { Link } from "wouter";
import { useAuthStore } from "../store/authStore";
import { useCategoryStore } from "../store/categoryStore";
import { useTemplateStore } from "../store/templateStore";
import { getCategories } from "../services/categories";
import { getSchedule, type Schedule } from "../services/schedule";
import { getTemplates } from "../services/templates";
import { getDayRecords, type DayRecord } from "../services/dayRecords";
import { DraggableColumn, type LayoutItem } from "./DraggableColumn";

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

interface ReviewDay {
  date: string;
  record: DayRecord | null;
  snapshotBlocks: NonNullable<DayRecord["snapshot"]>["blocks"];
  actualBlocks: NonNullable<DayRecord>["actual_blocks"];
}

export default function ReviewPage() {
  const { token } = useAuthStore();
  const { categories, setCategories } = useCategoryStore();
  const { templates, setTemplates } = useTemplateStore();
  const [weekStart, setWeekStart] = useState(() => mondayOf(new Date()));
  const [schedule, setSchedule] = useState<Schedule | null>(null);
  const [records, setRecords] = useState<DayRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const from = dateValue(weekStart);
  const to = dateValue(addDays(weekStart, 6));

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
  }, [token, from, to]);

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
          snapshotBlocks:
            record?.snapshot?.blocks ??
            template?.current_snapshot?.snapshot_blocks ??
            [],
          actualBlocks: record?.actual_blocks ?? [],
        };
      }),
    [weekStart, records, schedule, templates],
  );

  const toItems = (
    blocks: ReviewDay["snapshotBlocks"] | ReviewDay["actualBlocks"],
  ): LayoutItem[] =>
    blocks.map((block, index) => ({
      id: `${block.start_time}-${index}`,
      offset: minutes(block.start_time),
      size: block.duration_minutes,
      categoryId: block.category_id,
    }));

  const renderBlock = (item: LayoutItem) => {
    const category = categories.find((value) => value.id === item.categoryId);
    return (
      <div
        className="h-full rounded-sm px-1 text-[10px] text-snow overflow-hidden"
        style={{ backgroundColor: category?.color || "#003448" }}
      >
        {category?.name}
      </div>
    );
  };

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
        <Link
          href="/schedule"
          className="ml-auto text-sm text-slate-blue hover:text-cloud"
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
          </div>
        ))}
        <div className="relative h-[1440px]">
          {Array.from({ length: 24 }, (_, hour) => (
            <div
              key={hour}
              className="h-[60px] border-t border-slate-grey/10 text-right pr-2 text-xs font-mono text-slate-blue"
            >
              {String(hour).padStart(2, "0")}:00
            </div>
          ))}
        </div>
        {days.map((day) => (
          <div
            key={day.date}
            className="relative h-[1440px] border-l border-slate-grey/20"
          >
            {loading ? (
              <div className="absolute inset-x-1 top-20 h-64 rounded bg-slate-blue/20 animate-pulse" />
            ) : (
              <>
                <DraggableColumn
                  items={toItems(day.snapshotBlocks)}
                  gridUnit={GRID_UNIT}
                  baseWidth="100%"
                  onChange={() => undefined}
                  editable={false}
                  renderItem={renderBlock}
                  containerClassName="absolute inset-y-0 left-0 w-[40%] bg-transparent"
                  itemClassName="px-1"
                />
                <DraggableColumn
                  items={toItems(day.actualBlocks)}
                  gridUnit={GRID_UNIT}
                  baseWidth="100%"
                  onChange={() => undefined}
                  editable={false}
                  renderItem={renderBlock}
                  containerClassName="absolute inset-y-0 right-0 w-[60%] bg-transparent"
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
