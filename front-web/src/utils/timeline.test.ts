import { describe, expect, it } from "vitest";
import {
  constrainTimelineDragDelta,
  constrainTimelinePosition,
  findNewTimelineBlockPosition,
  groupTimelineItemsIntoColumns,
} from "./timeline";

const options = {
  dayStartMinutes: 8 * 60,
  dayEndMinutes: 18 * 60,
  minimumDuration: 1,
  snapToEdges: true,
  edgeSnapThreshold: 8,
};

describe("groupTimelineItemsIntoColumns", () => {
  it("keeps second-precise adjacent actual blocks in the same column", () => {
    const actualBlocks = [
      { id: "first", offset: 8 * 60 + 19 + 36 / 60, size: 246 },
      { id: "second", offset: 12 * 60 + 25 + 52 / 60, size: 104 },
      { id: "third", offset: 14 * 60 + 9 + 52 / 60, size: 30 },
      { id: "fourth", offset: 14 * 60 + 39 + 52 / 60, size: 162 },
      { id: "fifth", offset: 17 * 60 + 21 + 52 / 60, size: 30 },
    ];

    const columns = groupTimelineItemsIntoColumns(actualBlocks);

    expect(columns).toHaveLength(1);
    expect(columns[0].map((block) => block.id)).toEqual([
      "first",
      "second",
      "third",
      "fourth",
      "fifth",
    ]);
  });

  it("keeps actual overlaps in separate columns", () => {
    const overlappingBlocks = [
      { id: "first", offset: 8 * 60, size: 60 },
      { id: "overlapping", offset: 8 * 60 + 59, size: 30 },
    ];

    expect(groupTimelineItemsIntoColumns(overlappingBlocks)).toHaveLength(2);
  });
});

describe("findNewTimelineBlockPosition", () => {
  it("chooses the fitting hole closest to the visible timeline center", () => {
    const result = findNewTimelineBlockPosition(
      [
        { startMinutes: 8 * 60, durationMinutes: 30 },
        { startMinutes: 10 * 60, durationMinutes: 30 },
        { startMinutes: 14 * 60, durationMinutes: 30 },
      ],
      8 * 60,
      18 * 60,
      13 * 60,
    );

    expect(result).toEqual({ startMinutes: 12 * 60 + 45, durationMinutes: 30 });
  });

  it("centers a new block in the visible area when there are no blocks", () => {
    const result = findNewTimelineBlockPosition(
      [],
      8 * 60,
      18 * 60,
      13 * 60,
    );

    expect(result).toEqual({ startMinutes: 12 * 60 + 45, durationMinutes: 30 });
  });

  it("returns no position when no hole can fit 30 minutes", () => {
    const result = findNewTimelineBlockPosition(
      [{ startMinutes: 8 * 60, durationMinutes: 10 * 60 }],
      8 * 60,
      18 * 60,
      13 * 60,
    );

    expect(result).toBeNull();
  });

  it("ignores blocks outside the day range", () => {
    const result = findNewTimelineBlockPosition(
      [{ startMinutes: 7 * 60, durationMinutes: 30 }],
      8 * 60,
      18 * 60,
      13 * 60,
    );

    expect(result).toEqual({ startMinutes: 12 * 60 + 45, durationMinutes: 30 });
  });

  it("uses the visible center to select a hole and snaps to the configured interval", () => {
    const result = findNewTimelineBlockPosition(
      [
        { startMinutes: 8 * 60 + 30, durationMinutes: 30 },
        { startMinutes: 10 * 60, durationMinutes: 2 * 60 + 37 },
        { startMinutes: 14 * 60 + 7, durationMinutes: 3 * 60 },
      ],
      8 * 60,
      18 * 60,
      13 * 60 + 50,
      15,
    );

    expect(result).toEqual({ startMinutes: 13 * 60 + 30, durationMinutes: 30 });
  });

  it("rounds second-precise gap edges up to a minute without overlapping", () => {
    const previousBlockEnd = 13 * 60 + 7 + 52 / 60;
    const result = findNewTimelineBlockPosition(
      [
        { startMinutes: 8 * 60, durationMinutes: 30 },
        { startMinutes: previousBlockEnd - 60, durationMinutes: 60 },
        { startMinutes: 15 * 60, durationMinutes: 180 },
      ],
      8 * 60,
      18 * 60,
      13 * 60 + 23,
      1,
    );

    expect(result).toEqual({ startMinutes: 13 * 60 + 8, durationMinutes: 30 });
    expect(result!.startMinutes).toBeGreaterThanOrEqual(previousBlockEnd);
  });
});

describe("constrainTimelinePosition", () => {
  it("keeps adjacent blocks adjacent without shifting the next block", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 8 * 60, durationMinutes: 60 },
      undefined,
      { startMinutes: 9 * 60, durationMinutes: 45 },
      options,
    );

    expect(result).toEqual({ startMinutes: 8 * 60, durationMinutes: 60 });
  });

  it("clamps a moved block before the next block", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 8 * 60 + 50, durationMinutes: 30 },
      undefined,
      { startMinutes: 9 * 60, durationMinutes: 45 },
      options,
    );

    expect(result).toEqual({ startMinutes: 8 * 60 + 30, durationMinutes: 30 });
  });

  it("snaps a nearby edge to the next block", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 8 * 60, durationMinutes: 57 },
      undefined,
      { startMinutes: 9 * 60, durationMinutes: 45 },
      options,
    );

    expect(result).toEqual({ startMinutes: 8 * 60, durationMinutes: 60 });
  });

  it("preserves zero-duration event blocks", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 8 * 60 + 48, durationMinutes: 0 },
      { startMinutes: 8 * 60, durationMinutes: 30 },
      { startMinutes: 9 * 60, durationMinutes: 30 },
      options,
    );

    expect(result).toEqual({ startMinutes: 8 * 60 + 48, durationMinutes: 0 });
  });

  it("clamps a moved block after the previous block", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 8 * 60 + 5, durationMinutes: 20 },
      { startMinutes: 8 * 60, durationMinutes: 30 },
      { startMinutes: 10 * 60, durationMinutes: 30 },
      options,
    );

    expect(result).toEqual({ startMinutes: 8 * 60 + 30, durationMinutes: 20 });
  });

  it("snaps the start edge to the previous block", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 8 * 60 + 34, durationMinutes: 20 },
      { startMinutes: 8 * 60, durationMinutes: 30 },
      { startMinutes: 10 * 60, durationMinutes: 30 },
      options,
    );

    expect(result).toEqual({ startMinutes: 8 * 60 + 30, durationMinutes: 20 });
  });

  it("reduces duration when the minimum duration cannot fit", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 8 * 60 + 55, durationMinutes: 20 },
      { startMinutes: 8 * 60, durationMinutes: 55 },
      { startMinutes: 9 * 60 + 5, durationMinutes: 30 },
      { ...options, minimumDuration: 15 },
    );

    expect(result).toEqual({ startMinutes: 8 * 60 + 55, durationMinutes: 15 });
  });

  it("keeps a block inside the configured day range", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 17 * 60 + 50, durationMinutes: 30 },
      undefined,
      undefined,
      options,
    );

    expect(result).toEqual({ startMinutes: 17 * 60 + 30, durationMinutes: 30 });
  });

  it("does not edge-snap when edge snapping is disabled", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 8 * 60, durationMinutes: 57 },
      undefined,
      { startMinutes: 9 * 60, durationMinutes: 45 },
      { ...options, snapToEdges: false },
    );

    expect(result).toEqual({ startMinutes: 8 * 60, durationMinutes: 57 });
  });

  it("preserves an exact edge without shifting either block", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 9 * 60, durationMinutes: 30 },
      { startMinutes: 8 * 60, durationMinutes: 60 },
      { startMinutes: 10 * 60, durationMinutes: 30 },
      options,
    );

    expect(result).toEqual({ startMinutes: 9 * 60, durationMinutes: 30 });
  });

  it("keeps second-precise blocks from overlapping", () => {
    const result = constrainTimelinePosition(
      { startMinutes: 13 * 60 + 7, durationMinutes: 75 },
      { startMinutes: 12 * 60 + 41 + 52 / 60, durationMinutes: 26 },
      { startMinutes: 14 * 60 + 22 + 45 / 60, durationMinutes: 207 },
      options,
    );

    expect(result).toEqual({
      startMinutes: 13 * 60 + 7 + 52 / 60,
      durationMinutes: 74,
    });
  });
});

describe("constrainTimelineDragDelta", () => {
  const items = [
    { id: "first", offset: 0, size: 60 },
    { id: "second", offset: 75, size: 30 },
    { id: "third", offset: 120, size: 45 },
  ];

  it("snaps movement to the previous edge", () => {
    expect(constrainTimelineDragDelta(items, "second", "move", -12, 1, 8)).toBe(
      -15,
    );
  });

  it("snaps movement to the next edge", () => {
    expect(constrainTimelineDragDelta(items, "second", "move", 12, 1, 8)).toBe(
      15,
    );
  });

  it("prevents movement through the previous block", () => {
    expect(constrainTimelineDragDelta(items, "second", "move", -40, 1, 8)).toBe(
      -15,
    );
  });

  it("prevents movement through the next block", () => {
    expect(constrainTimelineDragDelta(items, "second", "move", 40, 1, 8)).toBe(
      15,
    );
  });

  it("snaps top resize to the previous edge", () => {
    expect(
      constrainTimelineDragDelta(items, "second", "resize-top", -12, 1, 8),
    ).toBe(-15);
  });

  it("snaps bottom resize to the next edge", () => {
    expect(
      constrainTimelineDragDelta(items, "second", "resize-bottom", 12, 1, 8),
    ).toBe(15);
  });

  it("keeps resize within the minimum duration", () => {
    expect(
      constrainTimelineDragDelta(items, "second", "resize-bottom", -50, 15, 8),
    ).toBe(-15);
  });
});
