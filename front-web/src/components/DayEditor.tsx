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

const MIN_REVIEW_BLOCK_DURATION_MINUTES = 5;

function editableBlocks(record: DayRecord): ActualBlockInput[] {
  const actualBlocks = record.actual
    .filter(
      (block) =>
        block.block_type !== "untracked" &&
        block.duration_minutes >= MIN_REVIEW_BLOCK_DURATION_MINUTES,
    )
    .map(
      ({
        category_id,
        block_type,
        start_time,
        duration_minutes,
      }): ActualBlockInput => ({
        category_id,
        block_type: block_type === "blank" ? "blank" : "actual",
        start_time,
        duration_minutes,
      }),
    )
    .sort((left, right) => left.start_time.localeCompare(right.start_time));

  if (actualBlocks.length > 0) return actualBlocks;

  return record.plan
    .map(({ category_id, start_time, duration_minutes }) => ({
      category_id,
      block_type: "actual" as const,
      start_time,
      duration_minutes,
    }))
    .sort((left, right) => left.start_time.localeCompare(right.start_time));
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
  const [blocks, setBlocks] = useState<ActualBlockInput[]>([]);
  const [loading, setLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    setLoading(true);
    Promise.all([
      getCategories(token),
      getDayRecord(token, date).catch(() => createDayRecord(token, date)),
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
          ),
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
            allowBlank
            dayRangeStartTime={settings?.day_range_start_time}
            dayRangeEndTime={settings?.day_range_end_time}
            snapInterval={1}
            minimumBlockDuration={1}
            durationStep={1}
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
