import { afterEach, describe, expect, it, vi } from "vitest";
import { toLocalTimestamp } from "./localTimestamp";

describe("toLocalTimestamp", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("preserves the local UTC offset in RFC3339 format", () => {
    vi.spyOn(Date.prototype, "getTimezoneOffset").mockReturnValue(-120);

    expect(toLocalTimestamp(new Date("2026-09-26T12:30:00Z"))).toBe(
      "2026-09-26T14:30:00+02:00",
    );
  });

  it("formats negative offsets", () => {
    vi.spyOn(Date.prototype, "getTimezoneOffset").mockReturnValue(330);

    expect(toLocalTimestamp(new Date("2026-09-26T12:30:00Z"))).toBe(
      "2026-09-26T07:00:00-05:30",
    );
  });
});
