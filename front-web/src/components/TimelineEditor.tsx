import { useState, useMemo, useEffect, useRef, type MouseEvent } from "react";
import type { PlannedBlock } from "../services/templates";
import type { Category } from "../services/categories";
import { DraggableColumn, type LayoutItem } from "./DraggableColumn";
import { getContrastTextColor } from "../utils/colors";
import { useAuthStore } from "../store/authStore";
import { useSettingsStore } from "../store/settingsStore";
import { getSettings } from "../services/settings";
import { createPortal } from "react-dom";

const GRID_UNIT = 2;
const SNAP_INTERVAL = 15;
const HOUR_HEIGHT = 60 * GRID_UNIT;

function timeToMinutes(time: string): number {
  const [hours, mins] = time.split(":").map(Number);
  return hours * 60 + mins;
}

function minutesToTime(minutes: number): string {
  const hours = Math.floor(minutes / 60) % 24;
  const mins = minutes % 60;
  return `${String(hours).padStart(2, "0")}:${String(mins).padStart(2, "0")}:00`;
}

function formatTime(time: string): string {
  return time.substring(0, 5);
}

function BlockEditPopover({
  block,
  blockIndex,
  categories,
  anchorRect,
  onUpdate,
  onDelete,
  onClose,
}: {
  block: PlannedBlock;
  blockIndex: number;
  categories: Category[];
  anchorRect: DOMRect | null;
  onUpdate: (index: number, updates: Partial<PlannedBlock>) => void;
  onDelete: (index: number) => void;
  onClose: () => void;
}) {
  const popoverRef = useRef<HTMLDivElement>(null);
  const [isNarrow, setIsNarrow] = useState(
    () => window.matchMedia("(max-width: 767px)").matches,
  );

  useEffect(() => {
    const mediaQuery = window.matchMedia("(max-width: 767px)");
    const handleChange = () => setIsNarrow(mediaQuery.matches);
    mediaQuery.addEventListener("change", handleChange);
    return () => mediaQuery.removeEventListener("change", handleChange);
  }, []);

  useEffect(() => {
    const handleOutsideClick = (event: MouseEvent) => {
      if (
        popoverRef.current &&
        !popoverRef.current.contains(event.target as Node)
      )
        onClose();
    };
    document.addEventListener("mousedown", handleOutsideClick as () => void);
    return () =>
      document.removeEventListener(
        "mousedown",
        handleOutsideClick as () => void,
      );
  }, [onClose]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  if (!anchorRect) return null;

  const popoverWidth = 280;
  const gap = 8;
  const left =
    anchorRect.right + gap + popoverWidth <= window.innerWidth - 16
      ? anchorRect.right + gap
      : anchorRect.left - popoverWidth - gap;

  return createPortal(
    <div
      ref={popoverRef}
      className="fixed z-[100] w-full bottom-0 left-0 p-4 bg-navy border border-slate-grey rounded-t-lg shadow-xl space-y-3 animate-in fade-in duration-micro md:bottom-auto md:left-auto md:w-[280px] md:rounded-lg"
      style={
        isNarrow
          ? undefined
          : { top: Math.max(8, anchorRect.top), left: Math.max(8, left) }
      }
    >
      <div>
        <label className="block text-sm font-medium text-cloud mb-1">
          Category
        </label>
        <select
          value={block.category_id}
          onChange={(event) =>
            onUpdate(blockIndex, { category_id: parseInt(event.target.value) })
          }
          className="w-full px-3 py-2 text-sm text-snow bg-navy/80 border border-slate-grey rounded-lg outline-none focus:border-cloud transition-colors duration-micro"
          autoFocus
        >
          {categories.map((category) => (
            <option key={category.id} value={category.id}>
              {category.name}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-cloud mb-1">
          Start
        </label>
        <input
          type="time"
          value={block.start_time.substring(0, 5)}
          onChange={(event) =>
            onUpdate(blockIndex, { start_time: `${event.target.value}:00` })
          }
          className="w-full px-3 py-2 text-sm text-snow font-mono bg-navy/80 border border-slate-grey rounded-lg outline-none focus:border-cloud transition-colors duration-micro"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-cloud mb-1">
          Duration (min)
        </label>
        <input
          type="number"
          value={block.duration_minutes}
          onChange={(event) => {
            const value = Math.max(
              15,
              Math.round((parseInt(event.target.value) || 15) / 15) * 15,
            );
            onUpdate(blockIndex, { duration_minutes: value });
          }}
          min={15}
          step={15}
          className="w-full px-3 py-2 text-sm text-snow font-mono bg-navy/80 border border-slate-grey rounded-lg outline-none focus:border-cloud transition-colors duration-micro"
        />
      </div>

      <div className="flex items-center justify-between pt-2">
        <button
          onClick={() => onDelete(blockIndex)}
          className="px-3 py-1.5 text-sm font-semibold text-error hover:bg-error/10 rounded-lg transition-colors duration-micro"
        >
          Delete
        </button>
        <button
          onClick={onClose}
          className="px-3 py-1.5 text-sm font-semibold text-cloud hover:text-snow transition-colors duration-micro"
        >
          Done
        </button>
      </div>
    </div>,
    document.body,
  );
}

export default function TimelineEditor({
  blocks,
  categories,
  onChange,
}: {
  blocks: PlannedBlock[];
  categories: Category[];
  onChange: (blocks: PlannedBlock[]) => void;
}) {
  const { token } = useAuthStore();
  const { settings, setSettings } = useSettingsStore();
  const [selectedBlockId, setSelectedBlockId] = useState<string | null>(null);
  const [popoverAnchor, setPopoverAnchor] = useState<DOMRect | null>(null);
  const [dayStartMinutes, setDayStartMinutes] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(600);
  const blockIds = useRef<Map<number, string>>(new Map());
  const idCounter = useRef(0);

  useEffect(() => {
    if (!containerRef.current) return;
    const observer = new ResizeObserver(([entry]) => {
      if (entry) setContainerWidth(Math.max(0, entry.contentRect.width - 64));
    });
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    if (token && !settings)
      getSettings(token).then(setSettings).catch(console.error);
  }, [token, settings, setSettings]);

  useEffect(() => {
    if (settings?.day_boundary_time)
      setDayStartMinutes(timeToMinutes(settings.day_boundary_time));
  }, [settings]);

  const layoutItems: LayoutItem[] = useMemo(
    () =>
      blocks.map((block, index) => {
        if (!blockIds.current.has(index))
          blockIds.current.set(index, `block-${idCounter.current++}`);
        const blockMinutes = timeToMinutes(block.start_time);
        let offset = blockMinutes - dayStartMinutes;
        if (offset < 0) offset += 24 * 60;
        return {
          id: blockIds.current.get(index)!,
          offset,
          size: block.duration_minutes,
          categoryId: block.category_id,
          blockIndex: index,
        };
      }),
    [blocks, dayStartMinutes],
  );

  const handleLayoutChange = (newItems: LayoutItem[]) => {
    onChange(
      newItems.map((item) => {
        const originalBlock = blocks[item.blockIndex];
        let absoluteMinutes = dayStartMinutes + item.offset;
        if (absoluteMinutes >= 24 * 60) absoluteMinutes -= 24 * 60;
        return {
          ...originalBlock,
          start_time: minutesToTime(absoluteMinutes),
          duration_minutes: Math.max(15, Math.round(item.size / 15) * 15),
        };
      }),
    );
  };

  const handleBlockClick = (blockId: string, event: MouseEvent) => {
    event.stopPropagation();
    if (selectedBlockId === blockId) {
      setSelectedBlockId(null);
      setPopoverAnchor(null);
      return;
    }
    setSelectedBlockId(blockId);
    setPopoverAnchor(
      (event.currentTarget as HTMLElement).getBoundingClientRect(),
    );
  };

  const updateBlock = (index: number, updates: Partial<PlannedBlock>) => {
    onChange(
      blocks.map((block, blockIndex) =>
        blockIndex === index ? { ...block, ...updates } : block,
      ),
    );
  };

  const removeBlock = (index: number) => {
    onChange(blocks.filter((_, blockIndex) => blockIndex !== index));
    setSelectedBlockId(null);
    setPopoverAnchor(null);
  };

  const addBlock = () => {
    const lastBlock = blocks[blocks.length - 1];
    const newStartMinutes = lastBlock
      ? timeToMinutes(lastBlock.start_time) + lastBlock.duration_minutes
      : dayStartMinutes;
    onChange([
      ...blocks,
      {
        category_id: categories[0]?.id || 0,
        start_time: minutesToTime(newStartMinutes % (24 * 60)),
        duration_minutes: 60,
      },
    ]);
  };

  const selectedBlockIndex = useMemo(() => {
    if (!selectedBlockId) return null;
    return (
      layoutItems.find((item) => item.id === selectedBlockId)?.blockIndex ??
      null
    );
  }, [selectedBlockId, layoutItems]);

  const timeLabels = useMemo(
    () =>
      Array.from({ length: 24 }, (_, index) => ({
        hour: Math.floor((dayStartMinutes / 60 + index) % 24),
      })),
    [dayStartMinutes],
  );

  const getCategoryColor = (categoryId: number) =>
    categories.find((category) => category.id === categoryId)?.color ||
    "#003448";
  const getCategoryName = (categoryId: number) =>
    categories.find((category) => category.id === categoryId)?.name ||
    "Unknown";

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-snow">Planned Blocks</h3>
        <button
          onClick={addBlock}
          disabled={categories.length === 0}
          className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud disabled:opacity-50"
        >
          + Add Block
        </button>
      </div>
      {categories.length === 0 && (
        <p className="text-sm text-cloud">
          Create categories first before adding blocks.
        </p>
      )}

      <div
        ref={containerRef}
        className="bg-navy/20 rounded-lg border border-slate-grey overflow-y-auto max-h-[70vh]"
      >
        <div className="flex">
          <div className="flex-shrink-0 w-16 sticky left-0 bg-navy/60 z-10">
            {timeLabels.map(({ hour }) => (
              <div
                key={hour}
                className="text-sm font-mono text-cloud text-right pr-2 border-b border-slate-grey/20 tabular-nums"
                style={{ height: HOUR_HEIGHT }}
              >
                <span className="relative -top-2">
                  {String(hour).padStart(2, "0")}:00
                </span>
              </div>
            ))}
          </div>
          <div className="flex-1 relative min-w-0">
            <DraggableColumn
              items={layoutItems}
              gridUnit={GRID_UNIT}
              baseWidth={containerWidth}
              snapToInterval={SNAP_INTERVAL}
              containerClassName="border-0"
              onChange={handleLayoutChange}
              renderItem={(item, status) => {
                const color = getCategoryColor(item.categoryId);
                const block = blocks[item.blockIndex];
                const isSelected = selectedBlockId === item.id;
                if (status === "overlay") return null;
                return (
                  <div
                    className={`w-full h-full px-3 py-1.5 text-sm font-medium rounded-lg cursor-pointer transition-shadow duration-micro ${status === "dragging" ? "shadow-lg opacity-60" : "shadow-sm"} ${isSelected ? "ring-2 ring-snow ring-offset-2 ring-offset-navy" : ""}`}
                    style={{
                      backgroundColor: color,
                      color: getContrastTextColor(color),
                    }}
                    onClick={(event) => handleBlockClick(item.id, event)}
                  >
                    {isSelected && (
                      <div className="absolute top-1 right-1 w-2 h-2 bg-snow rounded-full shadow" />
                    )}
                    <div className="font-semibold truncate">
                      {getCategoryName(item.categoryId)}
                    </div>
                    <div className="font-mono text-sm opacity-80 tabular-nums">
                      {formatTime(block.start_time)} · {item.size}m
                    </div>
                  </div>
                );
              }}
            />
          </div>
        </div>
      </div>

      {selectedBlockId && selectedBlockIndex !== null && (
        <BlockEditPopover
          block={blocks[selectedBlockIndex]}
          blockIndex={selectedBlockIndex}
          categories={categories}
          anchorRect={popoverAnchor}
          onUpdate={updateBlock}
          onDelete={removeBlock}
          onClose={() => {
            setSelectedBlockId(null);
            setPopoverAnchor(null);
          }}
        />
      )}
    </div>
  );
}
