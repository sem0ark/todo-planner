import { useEffect, useState } from "react";
import { Link } from "wouter";
import { useAuthStore } from "../store/authStore";
import { useCategoryStore } from "../store/categoryStore";
import { useSettingsStore } from "../store/settingsStore";
import {
  createDayRecord,
  getDayRecord,
  updateDayBlocks,
  type ActualBlockInput,
  type DayRecord,
} from "../services/dayRecords";
import { getCategories } from "../services/categories";
import TimelineEditor from "./TimelineEditor";
import {
  getLocalTimezoneOffsetMinutes,
  localTimeToUtcTimestamp,
  resolveTimezoneOffsetMinutes,
  utcTimestampToLocalTime,
} from "../utils/timeline";

const MIN_REVIEW_BLOCK_DURATION_MINUTES = 10;
const REVIEW_GRID_MINUTES = 5;

type LocalActualBlock = Omit<ActualBlockInput, "start_at"> & {
  start_time: string;
};

function editableBlocks(record: DayRecord): LocalActualBlock[] {
  const actualBlocks = record.actual
    .filter(
      (block) => block.duration_minutes >= MIN_REVIEW_BLOCK_DURATION_MINUTES,
    )
    .map(
      ({
        category_id,
        block_type,
        start_at,
        duration_minutes,
      }): LocalActualBlock => ({
        category_id,
        block_type,
        start_time: utcTimestampToLocalTime(
          start_at,
          resolveTimezoneOffsetMinutes(record.timezone_offset_minutes),
        ),
        duration_minutes,
      }),
    )
    .sort((left, right) => left.start_time.localeCompare(right.start_time));

  return actualBlocks;
}

export default function DayEditor({
  date,
  onClose,
}: {
  date: string;
  onClose: () => void;
}) {
  const { token } = useAuthStore();
  const { categories, setCategories } = useCategoryStore();
  const { settings } = useSettingsStore();
  const [record, setRecord] = useState<DayRecord | null>(null);
  const [blocks, setBlocks] = useState<LocalActualBlock[]>([]);
  const [loading, setLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    setLoading(true);
    Promise.all([
      getCategories(token),
      getDayRecord(token, date).catch(() =>
        createDayRecord(token, date, getLocalTimezoneOffsetMinutes()),
      ),
    ])
      .then(([nextCategories, nextRecord]) => {
        if (cancelled) return;
        setCategories(nextCategories);
        setRecord(nextRecord);
        setBlocks(editableBlocks(nextRecord));
      })
      .catch((reason) => {
        if (!cancelled) {
          setError(
            reason instanceof Error ? reason.message : "Failed to load day",
          );
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [token, date, setCategories]);

  const handleSave = async () => {
    if (!token) return;
    setIsSaving(true);
    setError(null);
    try {
      const savedRecord = await updateDayBlocks(token, date, {
        actual: [...blocks]
          .sort((left, right) =>
            left.start_time.localeCompare(right.start_time),
          )
          .filter(
            (block) =>
              block.duration_minutes >= MIN_REVIEW_BLOCK_DURATION_MINUTES,
          )
          .map(({ start_time, ...block }) => ({
            ...block,
            start_at: localTimeToUtcTimestamp(
              date,
              start_time,
              resolveTimezoneOffsetMinutes(record?.timezone_offset_minutes),
            ),
          })),
      });
      setRecord(savedRecord);
      setBlocks(editableBlocks(savedRecord));
      onClose();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Failed to save day");
    } finally {
      setIsSaving(false);
    }
  };

  if (loading) return <p className="text-sm text-cloud">Loading day...</p>;

  return (
    <section className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-snow">Day View</h2>
          <p className="font-mono text-sm text-cloud">{date}</p>
        </div>
        <Link href="/review" className="text-sm text-cloud hover:text-snow">
          Back to review
        </Link>
      </div>

      {error && (
        <div className="px-4 py-3 bg-error/10 border border-error rounded-lg text-error text-sm">
          {error}
        </div>
      )}

      {record && (
        <div className="space-y-6">
          <TimelineEditor
            blocks={blocks}
            categories={categories}
            onChange={setBlocks}
            title="Actual Blocks"
            dayRangeStartTime={settings?.day_range_start_time}
            dayRangeEndTime={settings?.day_range_end_time}
            snapInterval={REVIEW_GRID_MINUTES}
            minimumBlockDuration={MIN_REVIEW_BLOCK_DURATION_MINUTES}
            durationStep={REVIEW_GRID_MINUTES}
            snapToEdges
          />

          <div className="flex gap-3">
            <button
              onClick={handleSave}
              disabled={isSaving}
              className="px-6 py-2 text-sm font-semibold text-navy bg-snow rounded-lg hover:bg-cloud disabled:opacity-50"
            >
              {isSaving ? "Saving..." : "Save Changes"}
            </button>
            <button
              onClick={onClose}
              className="px-6 py-2 text-sm font-semibold text-cloud border border-slate-grey rounded-lg hover:bg-slate-blue/10"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </section>
  );
}
