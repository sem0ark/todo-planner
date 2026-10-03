import { useState, useMemo, useEffect, useRef, type MouseEvent } from "react";
import type { PlannedBlock } from "../services/templates";
import type { ActualBlockInput } from "../services/dayRecords";
import type { Category } from "../services/categories";
import { DraggableColumn, type LayoutItem } from "./DraggableColumn";
import { getContrastTextColor } from "../utils/colors";
import { createPortal } from "react-dom";
import { constrainTimelinePosition } from "../utils/timeline";

const GRID_UNIT = 2;
const SNAP_INTERVAL = 15;
const HOUR_HEIGHT = 60 * GRID_UNIT;

function timeToMinutes(time: string): number {
  const [hours, mins, seconds = 0] = time.split(":").map(Number);
  return hours * 60 + mins + seconds / 60;
}

function minutesToTime(minutes: number): string {
  const totalSeconds = Math.round(minutes * 60);
  const hours = Math.floor(totalSeconds / 3600) % 24;
  const mins = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return `${String(hours).padStart(2, "0")}:${String(mins).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

function snapTimeToInterval(time: string, interval: number): string {
  const snappedMinutes = Math.round(timeToMinutes(time) / interval) * interval;
  return minutesToTime(snappedMinutes);
}

function formatTime(time: string): string {
  return time.substring(0, 5);
}

type EditableBlock = PlannedBlock | ActualBlockInput;

function BlockEditPopover({
  block,
  blockIndex,
  categories,
  anchorRect,
  onUpdate,
  onDelete,
  onClose,
  minimumDuration,
  durationStep,
  timeStep,
}: {
  block: EditableBlock;
  blockIndex: number;
  categories: Category[];
  anchorRect: DOMRect | null;
  onUpdate: (index: number, updates: Partial<EditableBlock>) => void;
  onDelete: (index: number) => void;
  onClose: () => void;
  minimumDuration: number;
  durationStep: number;
  timeStep: number;
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
          value={block.category_id ?? ""}
          onChange={(event) => {
            if ("block_type" in block) {
              onUpdate(blockIndex, {
                category_id: event.target.value
                  ? parseInt(event.target.value)
                  : null,
                block_type: event.target.value ? "actual" : "blank",
              });
              return;
            }
            onUpdate(blockIndex, { category_id: parseInt(event.target.value) });
          }}
          className="w-full px-3 py-2 text-sm text-snow bg-navy/80 border border-slate-grey rounded-lg outline-none focus:border-cloud transition-colors duration-micro"
          autoFocus
        >
          {"block_type" in block && <option value="">Blank</option>}
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
          step={timeStep * 60}
          value={block.start_time.substring(0, 5)}
          onChange={(event) => {
            const [hours, minutes] = event.target.value.split(":").map(Number);
            const roundedMinutes =
              Math.round((hours * 60 + minutes) / timeStep) * timeStep;
            const roundedHours = Math.floor(roundedMinutes / 60) % 24;
            const displayMinutes = roundedMinutes % 60;
            onUpdate(blockIndex, {
              start_time: `${String(roundedHours).padStart(2, "0")}:${String(displayMinutes).padStart(2, "0")}:00`,
            });
          }}
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
              minimumDuration,
              Math.round(
                (parseInt(event.target.value) || minimumDuration) /
                  durationStep,
              ) * durationStep,
            );
            onUpdate(blockIndex, { duration_minutes: value });
          }}
          min={minimumDuration}
          step={durationStep}
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

export type TimelineBlock = EditableBlock;

export default function TimelineEditor<T extends EditableBlock>({
  blocks,
  categories,
  onChange,
  title = "Snapshot Blocks",
  allowBlank = false,
  dayRangeStartTime = "00:00:00",
  dayRangeEndTime = "24:00:00",
  snapInterval = SNAP_INTERVAL,
  minimumBlockDuration = 30,
  durationStep = 15,
  snapToEdges = false,
}: {
  blocks: T[];
  categories: Category[];
  onChange: (blocks: T[]) => void;
  title?: string;
  allowBlank?: boolean;
  dayRangeStartTime?: string;
  dayRangeEndTime?: string;
  snapInterval?: number;
  minimumBlockDuration?: number;
  durationStep?: number;
  snapToEdges?: boolean;
}) {
  const [selectedBlockId, setSelectedBlockId] = useState<string | null>(null);
  const dayStartMinutes = timeToMinutes(dayRangeStartTime);
  const dayEndMinutes =
    dayRangeEndTime === "24:00:00" ? 24 * 60 : timeToMinutes(dayRangeEndTime);
  const dayRangeMinutes = Math.max(60, dayEndMinutes - dayStartMinutes);
  const dayRangeHours = Math.ceil(dayRangeMinutes / 60);
  const [popoverAnchor, setPopoverAnchor] = useState<DOMRect | null>(null);
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

  const layoutItems: LayoutItem[] = useMemo(
    () =>
      blocks.flatMap((block, index) => {
        const blockStartMinutes = timeToMinutes(block.start_time);
        const blockEndMinutes = blockStartMinutes + block.duration_minutes;
        if (
          blockStartMinutes < dayStartMinutes ||
          blockStartMinutes >= dayEndMinutes
        ) {
          return [];
        }
        if (!blockIds.current.has(index))
          blockIds.current.set(index, `block-${idCounter.current++}`);
        const visibleDuration =
          Math.min(blockEndMinutes, dayEndMinutes) - blockStartMinutes;
        return {
          id: blockIds.current.get(index)!,
          offset: blockStartMinutes - dayStartMinutes,
          size: visibleDuration,
          categoryId: block.category_id,
          blockIndex: index,
        };
      }),
    [blocks, dayStartMinutes, dayEndMinutes],
  );

  const constrainBlock = (blockIndex: number, candidate: T): T => {
    const orderedBlocks = blocks
      .map((block, index) => ({
        block: index === blockIndex ? candidate : block,
        index,
      }))
      .sort(
        (left, right) =>
          timeToMinutes(left.block.start_time) -
          timeToMinutes(right.block.start_time),
      );
    const orderedIndex = orderedBlocks.findIndex(
      ({ index }) => index === blockIndex,
    );
    const previousBlock = orderedBlocks[orderedIndex - 1]?.block;
    const nextBlock = orderedBlocks[orderedIndex + 1]?.block;
    const position = constrainTimelinePosition(
      {
        startMinutes: timeToMinutes(candidate.start_time),
        durationMinutes: candidate.duration_minutes,
      },
      previousBlock && {
        startMinutes: timeToMinutes(previousBlock.start_time),
        durationMinutes: previousBlock.duration_minutes,
      },
      nextBlock && {
        startMinutes: timeToMinutes(nextBlock.start_time),
        durationMinutes: nextBlock.duration_minutes,
      },
      {
        dayStartMinutes,
        dayEndMinutes,
        minimumDuration: minimumBlockDuration,
        snapToEdges,
        edgeSnapThreshold: 8,
      },
    );
    return {
      ...candidate,
      start_time: minutesToTime(position.startMinutes),
      duration_minutes: position.durationMinutes,
    };
  };

  const handleLayoutChange = (newItems: LayoutItem[]) => {
    const changedItems = newItems.filter((item) => {
      const originalItem = layoutItems.find(
        (layoutItem) => layoutItem.blockIndex === item.blockIndex,
      );
      return (
        originalItem &&
        (originalItem.offset !== item.offset || originalItem.size !== item.size)
      );
    });
    const changedBlocks = new Map(
      changedItems.map((item) => {
        const originalBlock = blocks[item.blockIndex];
        const absoluteMinutes = Math.max(
          dayStartMinutes,
          dayStartMinutes + item.offset,
        );
        return [
          item.blockIndex,
          constrainBlock(item.blockIndex, {
            ...originalBlock,
            start_time: minutesToTime(absoluteMinutes),
            duration_minutes: Math.max(
              minimumBlockDuration,
              Math.min(
                Math.round(item.size / snapInterval) * snapInterval,
                dayEndMinutes - absoluteMinutes,
              ),
            ),
          }),
        ];
      }),
    );
    onChange(
      blocks.map((block, blockIndex) => changedBlocks.get(blockIndex) || block),
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

  const updateBlock = (index: number, updates: Partial<EditableBlock>) => {
    const candidate = { ...blocks[index], ...updates } as T;
    if (updates.start_time !== undefined) {
      candidate.start_time = snapTimeToInterval(
        updates.start_time,
        snapInterval,
      );
    }
    onChange(
      blocks.map((block, blockIndex) =>
        blockIndex === index ? constrainBlock(index, candidate) : block,
      ),
    );
  };

  const removeBlock = (index: number) => {
    onChange(blocks.filter((_, blockIndex) => blockIndex !== index));
    setSelectedBlockId(null);
    setPopoverAnchor(null);
  };

  const findNewBlockPosition = () => {
    const defaultDuration = Math.min(60, dayRangeMinutes);
    const sortedBlocks = blocks
      .map((block) => ({
        startMinutes: timeToMinutes(block.start_time),
        endMinutes: timeToMinutes(block.start_time) + block.duration_minutes,
      }))
      .sort((left, right) => left.startMinutes - right.startMinutes);
    let cursorMinutes = dayStartMinutes;

    for (const block of sortedBlocks) {
      const blockStartMinutes = Math.max(dayStartMinutes, block.startMinutes);
      const blockEndMinutes = Math.min(dayEndMinutes, block.endMinutes);
      const snappedStartMinutes =
        Math.ceil(cursorMinutes / snapInterval) * snapInterval;
      if (
        blockEndMinutes > cursorMinutes &&
        snappedStartMinutes + defaultDuration <= blockStartMinutes
      ) {
        return {
          startMinutes: snappedStartMinutes,
          durationMinutes: defaultDuration,
        };
      }
      cursorMinutes = Math.max(cursorMinutes, blockEndMinutes);
    }

    const snappedStartMinutes =
      Math.ceil(cursorMinutes / snapInterval) * snapInterval;
    if (snappedStartMinutes + defaultDuration <= dayEndMinutes) {
      return {
        startMinutes: snappedStartMinutes,
        durationMinutes: defaultDuration,
      };
    }
    return null;
  };

  const addBlock = () => {
    const newBlockPosition = findNewBlockPosition();
    if (!newBlockPosition) return;
    onChange([
      ...blocks,
      {
        category_id: categories[0]?.id || 0,
        start_time: minutesToTime(newBlockPosition.startMinutes),
        duration_minutes: newBlockPosition.durationMinutes,
        ...(blocks[0] && "block_type" in blocks[0]
          ? { block_type: "actual" as const }
          : {}),
      } as T,
    ]);
  };

  const addBlankBlock = () => {
    const newBlockPosition = findNewBlockPosition();
    if (!newBlockPosition) return;
    onChange([
      ...blocks,
      {
        category_id: null,
        block_type: "blank",
        start_time: minutesToTime(newBlockPosition.startMinutes),
        duration_minutes: newBlockPosition.durationMinutes,
      } as T,
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
      Array.from({ length: dayRangeHours }, (_, index) => ({
        hour: Math.floor((dayStartMinutes / 60 + index) % 24),
      })),
    [dayStartMinutes, dayRangeHours],
  );

  const getCategoryColor = (categoryId: number | null) =>
    categories.find((category) => category.id === categoryId)?.color ||
    "#003448";
  const getCategoryName = (categoryId: number | null) =>
    categories.find((category) => category.id === categoryId)?.name ||
    "Unknown";

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-snow">{title}</h3>
        <div className="flex gap-2">
          <button
            onClick={addBlock}
            disabled={categories.length === 0}
            className="h-9 px-4 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud disabled:opacity-50"
          >
            + Add Block
          </button>
          {allowBlank && (
            <button
              onClick={addBlankBlock}
              className="h-9 px-4 text-sm font-semibold text-cloud border border-slate-grey rounded-lg transition-all duration-micro hover:bg-slate-blue/10"
            >
              + Blank
            </button>
          )}
        </div>
      </div>
      {categories.length === 0 && (
        <p className="text-sm text-cloud">
          Create categories first before adding blocks.
        </p>
      )}

      <div
        ref={containerRef}
        className="bg-navy rounded-lg border border-slate-grey/20 overflow-y-auto max-h-[70vh]"
      >
        <div className="flex">
          <div className="flex-shrink-0 w-16 sticky left-0 bg-navy/60 z-10">
            {timeLabels.map(({ hour }, index) => (
              <div
                key={hour}
                className="text-sm font-mono text-cloud text-right pr-2 tabular-nums"
                style={{ height: HOUR_HEIGHT }}
              >
                <span
                  className={index === 0 ? "relative" : "relative -top-2.5"}
                >
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
              snapToInterval={snapInterval}
              minimumSize={minimumBlockDuration}
              snapToEdges={snapToEdges}
              edgeSnapThreshold={8}
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
                      {"block_type" in block && block.block_type === "blank"
                        ? "Blank"
                        : getCategoryName(item.categoryId)}
                    </div>
                    <div className="font-mono text-sm opacity-80 tabular-nums">
                      {formatTime(block.start_time)} · {Math.floor(item.size)}m
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
          minimumDuration={minimumBlockDuration}
          durationStep={durationStep}
          timeStep={snapInterval}
          onClose={() => {
            setSelectedBlockId(null);
            setPopoverAnchor(null);
          }}
        />
      )}
    </div>
  );
}
