export interface TimelinePosition {
  startMinutes: number;
  durationMinutes: number;
}

export interface TimelineColumnItem {
  id: string;
  offset: number;
  size: number;
}

/** Groups adjacent timeline items without treating floating-point noise as overlap. */
export function groupTimelineItemsIntoColumns<T extends TimelineColumnItem>(
  items: T[],
): T[][] {
  const positionTolerance = 1e-6;
  const sortedItems = [...items].sort(
    (left, right) => left.offset - right.offset,
  );
  const columns: T[][] = [];

  for (const item of sortedItems) {
    const availableColumn = columns.find((column) => {
      const lastItem = column[column.length - 1];
      return item.offset + positionTolerance >= lastItem.offset + lastItem.size;
    });

    if (availableColumn) {
      availableColumn.push(item);
    } else {
      columns.push([item]);
    }
  }

  return columns;
}

/** Finds the 30-minute addition point closest to the requested timeline center. */
export function findNewTimelineBlockPosition(
  blocks: TimelinePosition[],
  dayStartMinutes: number,
  dayEndMinutes: number,
  targetCenterMinutes: number,
  snapInterval = 1,
): TimelinePosition | null {
  const durationMinutes = 30;
  const timelineCenterMinutes = Math.min(
    dayEndMinutes,
    Math.max(dayStartMinutes, targetCenterMinutes),
  );
  const sortedBlocks = blocks
    .map((block) => ({
      startMinutes: Math.max(dayStartMinutes, block.startMinutes),
      endMinutes: Math.min(
        dayEndMinutes,
        block.startMinutes + block.durationMinutes,
      ),
    }))
    .filter((block) => block.endMinutes > block.startMinutes)
    .sort((left, right) => left.startMinutes - right.startMinutes);
  const gaps: TimelinePosition[] = [];
  let gapStartMinutes = dayStartMinutes;

  for (const block of sortedBlocks) {
    if (block.startMinutes > gapStartMinutes) {
      gaps.push({
        startMinutes: gapStartMinutes,
        durationMinutes: block.startMinutes - gapStartMinutes,
      });
    }
    gapStartMinutes = Math.max(gapStartMinutes, block.endMinutes);
  }
  if (gapStartMinutes < dayEndMinutes) {
    gaps.push({
      startMinutes: gapStartMinutes,
      durationMinutes: dayEndMinutes - gapStartMinutes,
    });
  }

  const fittingCandidates = gaps
    .filter((gap) => gap.durationMinutes >= durationMinutes)
    .map((gap) => {
      const earliestStartMinutes =
        Math.ceil(gap.startMinutes / snapInterval) * snapInterval;
      const latestStartMinutes =
        Math.floor(
          (gap.startMinutes + gap.durationMinutes - durationMinutes) /
            snapInterval,
        ) * snapInterval;
      if (earliestStartMinutes > latestStartMinutes) return null;

      const centeredStartMinutes = timelineCenterMinutes - durationMinutes / 2;
      const preferredStartMinutes = Math.min(
        latestStartMinutes,
        Math.max(earliestStartMinutes, centeredStartMinutes),
      );
      const snappedStartMinutes =
        Math.round(preferredStartMinutes / snapInterval) * snapInterval;
      const startMinutes = Math.min(latestStartMinutes, snappedStartMinutes);
      return {
        startMinutes,
        durationMinutes,
        distanceFromCenter: Math.abs(
          startMinutes + durationMinutes / 2 - timelineCenterMinutes,
        ),
      };
    })
    .filter((candidate) => candidate !== null)
    .sort((left, right) => left.distanceFromCenter - right.distanceFromCenter);

  return fittingCandidates[0]
    ? {
        startMinutes: fittingCandidates[0].startMinutes,
        durationMinutes,
      }
    : null;
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
