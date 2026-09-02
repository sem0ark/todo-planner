import type { Category } from "../services/categories";
import type { PlannedBlock } from "../services/templates";

interface TemplateSummaryProps {
  plannedBlocks: PlannedBlock[];
  categories: Category[];
}

interface CategoryTotal {
  categoryId: number;
  durationMinutes: number;
  name: string;
  color: string;
}

interface BarSegment {
  key: string;
  durationMinutes: number;
  color: string;
  isGap: boolean;
}

const MUTED_COLOR = "#91a6be";
const DAY_START_MINUTES = 7 * 60;
const DAY_END_MINUTES = 22 * 60;

function timeToMinutes(time: string): number {
  const [hours, minutes] = time.split(":").map(Number);
  return hours * 60 + minutes;
}

export function formatDuration(durationMinutes: number): string {
  if (durationMinutes < 60) return `${durationMinutes}m`;

  const hours = Math.floor(durationMinutes / 60);
  const minutes = durationMinutes % 60;
  return minutes === 0 ? `${hours}h` : `${hours}h ${minutes}m`;
}

function buildCategoryTotals(
  plannedBlocks: PlannedBlock[],
  categories: Category[],
): CategoryTotal[] {
  const categoryById = new Map(categories.map((category) => [category.id, category]));
  const totals = new Map<number, number>();

  plannedBlocks.forEach((block) => {
    const blockStart = timeToMinutes(block.start_time);
    const visibleStart = Math.max(DAY_START_MINUTES, blockStart);
    const visibleEnd = Math.min(
      DAY_END_MINUTES,
      blockStart + block.duration_minutes,
    );
    const visibleDuration = visibleEnd - visibleStart;

    if (visibleDuration <= 0) return;

    totals.set(
      block.category_id,
      (totals.get(block.category_id) ?? 0) + visibleDuration,
    );
  });

  return [...totals.entries()]
    .sort(([, firstDuration], [, secondDuration]) => secondDuration - firstDuration)
    .map(([categoryId, durationMinutes]) => {
      const category = categoryById.get(categoryId);
      return {
        categoryId,
        durationMinutes,
        name: category?.name ?? "Unknown category",
        color: category?.color ?? MUTED_COLOR,
      };
    });
}

function buildBarSegments(
  plannedBlocks: PlannedBlock[],
  categories: Category[],
): BarSegment[] {
  const categoryById = new Map(categories.map((category) => [category.id, category]));
  const blocks = [...plannedBlocks].sort((first, second) =>
    timeToMinutes(first.start_time) - timeToMinutes(second.start_time),
  );
  const segments: BarSegment[] = [];
  let cursor = DAY_START_MINUTES;

  blocks.forEach((block, index) => {
    const blockStart = Math.max(DAY_START_MINUTES, timeToMinutes(block.start_time));
    const blockEnd = Math.min(
      DAY_END_MINUTES,
      timeToMinutes(block.start_time) + block.duration_minutes,
    );
    const visibleDuration = blockEnd - blockStart;

    if (visibleDuration <= 0 || blockStart >= DAY_END_MINUTES) return;

    const gap = Math.max(0, blockStart - cursor);
    if (gap > 0) {
      segments.push({
        key: `gap-${index}`,
        durationMinutes: gap,
        color: MUTED_COLOR,
        isGap: true,
      });
    }

    segments.push({
      key: `block-${block.id ?? index}`,
      durationMinutes: visibleDuration,
      color: categoryById.get(block.category_id)?.color ?? MUTED_COLOR,
      isGap: false,
    });
    cursor = Math.max(cursor, blockEnd);
  });

  if (cursor < DAY_END_MINUTES) {
    segments.push({
      key: "gap-end",
      durationMinutes: DAY_END_MINUTES - cursor,
      color: MUTED_COLOR,
      isGap: true,
    });
  }

  return segments;
}

export default function TemplateSummary({
  plannedBlocks,
  categories,
}: TemplateSummaryProps) {
  if (plannedBlocks.length === 0) {
    return (
      <div className="mt-2">
        <div className="h-2 overflow-hidden rounded bg-slate-blue/15" />
        <span className="mt-1 block text-sm text-slate-grey">
          No blocks planned
        </span>
      </div>
    );
  }

  const totals = buildCategoryTotals(plannedBlocks, categories);
  const segments = buildBarSegments(plannedBlocks, categories);

  return (
    <div className="mt-2">
      <div className="flex h-2 overflow-hidden rounded">
        {segments.map((segment) => (
          <span
            key={segment.key}
            className="h-full transition-opacity duration-micro group-hover:!opacity-100"
            style={{
              flex: `${segment.durationMinutes} 1 0%`,
              minWidth: segment.isGap ? undefined : "3px",
              backgroundColor: segment.color,
              opacity: segment.isGap ? 0.2 : 0.9,
            }}
          />
        ))}
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        {totals.slice(0, 5).map((total) => (
          <span key={total.categoryId} className="flex items-center gap-1 text-sm text-cloud">
            <span className="size-2 shrink-0 rounded-full" style={{ backgroundColor: total.color }} />
            <span className="font-mono">{formatDuration(total.durationMinutes)}</span>
            <span>{total.name}</span>
          </span>
        ))}
        {totals.length > 5 && (
          <span className="text-sm text-slate-grey">+{totals.length - 5} more</span>
        )}
      </div>
    </div>
  );
}