export interface TimelinePosition {
  startMinutes: number;
  durationMinutes: number;
}

/** Finds a 30-minute addition point immediately after the latest in-range block. */
export function findNewTimelineBlockPosition(
  blocks: TimelinePosition[],
  dayStartMinutes: number,
  dayEndMinutes: number,
): TimelinePosition | null {
  const latestBlockEndMinutes = blocks.reduce<number | null>(
    (latestEndMinutes, block) => {
      const blockEndMinutes = block.startMinutes + block.durationMinutes;
      if (
        block.startMinutes >= dayEndMinutes ||
        blockEndMinutes <= dayStartMinutes
      ) {
        return latestEndMinutes;
      }

      const visibleBlockEndMinutes = Math.min(dayEndMinutes, blockEndMinutes);
      return latestEndMinutes === null
        ? visibleBlockEndMinutes
        : Math.max(latestEndMinutes, visibleBlockEndMinutes);
    },
    null,
  );
  const startMinutes =
    latestBlockEndMinutes === null
      ? dayStartMinutes
      : latestBlockEndMinutes + 1;
  const durationMinutes = 30;

  if (startMinutes + durationMinutes > dayEndMinutes) return null;

  return { startMinutes, durationMinutes };
}

export type TimelineDragMode = "move" | "resize-top" | "resize-bottom";

export interface TimelineItemPosition {
  id: string;
  offset: number;
  size: number;
}

export interface TimelineConstraintOptions {
  dayStartMinutes: number;
  dayEndMinutes: number;
  minimumDuration: number;
  snapToEdges: boolean;
  edgeSnapThreshold: number;
}

export function constrainTimelinePosition(
  position: TimelinePosition,
  previousPosition: TimelinePosition | undefined,
  nextPosition: TimelinePosition | undefined,
  options: TimelineConstraintOptions,
): TimelinePosition {
  if (position.durationMinutes === 0) return position;

  const previousEnd = previousPosition
    ? previousPosition.startMinutes + previousPosition.durationMinutes
    : options.dayStartMinutes;
  const nextStart = nextPosition?.startMinutes ?? options.dayEndMinutes;
  let duration = Math.max(options.minimumDuration, position.durationMinutes);
  let start = Math.max(options.dayStartMinutes, position.startMinutes);

  if (
    options.snapToEdges &&
    previousPosition &&
    Math.abs(start - previousEnd) <= options.edgeSnapThreshold
  ) {
    start = previousEnd;
  }
  if (
    options.snapToEdges &&
    nextPosition &&
    Math.abs(start + duration - nextStart) <= options.edgeSnapThreshold
  ) {
    duration = nextStart - start;
  }

  start = Math.min(start, options.dayEndMinutes - duration);
  start = Math.max(start, previousEnd);
  if (start + duration > nextStart) {
    const latestStart = nextStart - duration;
    if (latestStart >= previousEnd && latestStart >= options.dayStartMinutes) {
      start = latestStart;
    } else {
      duration = Math.floor(nextStart - start);
    }
    if (duration < options.minimumDuration) {
      duration = options.minimumDuration;
      start = Math.max(previousEnd, nextStart - duration);
    }
  }
  duration = Math.max(options.minimumDuration, Math.floor(duration));

  return {
    startMinutes: start,
    durationMinutes: Math.min(duration, options.dayEndMinutes - start),
  };
}

export function constrainTimelineDragDelta(
  items: TimelineItemPosition[],
  activeId: string,
  dragMode: TimelineDragMode,
  deltaUnits: number,
  minimumSize: number,
  edgeSnapThreshold: number,
): number {
  const activeItem = items.find((item) => item.id === activeId);
  if (!activeItem) return deltaUnits;

  const sortedItems = [...items].sort(
    (left, right) => left.offset - right.offset,
  );
  const activeIndex = sortedItems.findIndex((item) => item.id === activeId);
  const previousItem = sortedItems[activeIndex - 1];
  const nextItem = sortedItems[activeIndex + 1];

  if (dragMode === "move") {
    const proposedOffset = activeItem.offset + deltaUnits;
    const minimumOffset = previousItem
      ? previousItem.offset + previousItem.size
      : -Infinity;
    const maximumOffset = nextItem
      ? nextItem.offset - activeItem.size
      : Infinity;
    const boundedOffset = Math.min(
      maximumOffset,
      Math.max(minimumOffset, proposedOffset),
    );
    if (
      previousItem &&
      Math.abs(boundedOffset - minimumOffset) <= edgeSnapThreshold
    ) {
      return minimumOffset - activeItem.offset;
    }
    if (
      nextItem &&
      Math.abs(boundedOffset + activeItem.size - nextItem.offset) <=
        edgeSnapThreshold
    ) {
      return nextItem.offset - activeItem.size - activeItem.offset;
    }
    return boundedOffset - activeItem.offset;
  }

  if (dragMode === "resize-top" && previousItem) {
    const proposedOffset = activeItem.offset + deltaUnits;
    const minimumOffset = previousItem.offset + previousItem.size;
    const maximumOffset = activeItem.offset + activeItem.size - minimumSize;
    const boundedOffset = Math.min(
      maximumOffset,
      Math.max(minimumOffset, proposedOffset),
    );
    return Math.abs(boundedOffset - minimumOffset) <= edgeSnapThreshold
      ? minimumOffset - activeItem.offset
      : boundedOffset - activeItem.offset;
  }

  if (dragMode === "resize-bottom" && nextItem) {
    const proposedEnd = activeItem.offset + activeItem.size + deltaUnits;
    const minimumEnd = activeItem.offset + minimumSize;
    const maximumEnd = nextItem.offset;
    const boundedEnd = Math.min(maximumEnd, Math.max(minimumEnd, proposedEnd));
    return Math.abs(boundedEnd - maximumEnd) <= edgeSnapThreshold
      ? maximumEnd - activeItem.offset - activeItem.size
      : boundedEnd - activeItem.offset - activeItem.size;
  }

  return deltaUnits;
}
