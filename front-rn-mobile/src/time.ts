import type { ActualBlock, PlannedBlock } from "./models";

export function secondsOfDay(ts: number) {
  const d = new Date(ts);
  return d.getHours() * 3600 + d.getMinutes() * 60 + d.getSeconds();
}

export function parseTime(hms: string) {
  const p = hms.split(":").map(Number);
  return (p[0] ?? 0) * 3600 + (p[1] ?? 0) * 60 + (p[2] ?? 0);
}

export function formatTime(ts: number) {
  const d = new Date(ts);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

export function todayString() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export function findCurrentBlock(blocks: PlannedBlock[], now: number) {
  return (
    blocks.find(
      (b) =>
        now >= parseTime(b.startTime) &&
        now < parseTime(b.startTime) + b.durationMinutes * 60,
    ) ?? null
  );
}

export function findNextBlock(blocks: PlannedBlock[], now: number) {
  return (
    blocks
      .filter((b) => parseTime(b.startTime) > now)
      .sort((a, b) => parseTime(a.startTime) - parseTime(b.startTime))[0] ??
    null
  );
}

export function findCurrentActualBlock(blocks: ActualBlock[], now: number) {
  for (let i = blocks.length - 1; i >= 0; i--) {
    const b = blocks[i];
    const start = parseTime(b.startTime);
    if (b.durationMinutes <= 0 && now >= start) return b;
    if (now >= start && now < start + b.durationMinutes * 60) return b;
  }
  return null;
}

export function blockProgress(block: PlannedBlock, now: number) {
  return Math.min(
    1,
    Math.max(0, now - parseTime(block.startTime)) /
      Math.max(1, block.durationMinutes * 60),
  );
}

export function minutesBetween(a: string, b: string) {
  return Math.max(1, Math.round((parseTime(b) - parseTime(a)) / 60));
}
