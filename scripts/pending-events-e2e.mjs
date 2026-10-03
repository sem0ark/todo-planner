#!/usr/bin/env node

/**
This utility replays the widget's manually collected pending-event backup through
the public API, then reads the resulting day records back through the same API
used by the web review page.

For this diagnostic replay, every non-amendment event is sent as a `transition`,
including events originally marked as `confirmation`. This exposes category
changes that the original confirmation semantics hide, and intentionally keeps
same-category transitions so their timestamps can be inspected.

Start the local stack first, then run:

```bash
node scripts/pending-events-e2e.mjs --open
```

The script creates a unique test account, registers a desktop device, creates
the five categories from `MockTodoPlannerRepository`, creates a review template,
replays the events, and prints every resulting block in this format:

```text
2026-09-07
  04:00:00 - 12:53:09   533m  untracked Untracked
  12:53:09 - 13:01:00     8m  actual    Learning
```

The printed credentials and `#/review` URL can be used to inspect the same
records manually in the web frontend. The backup dates are not necessarily the
current week, so use the review page's previous/next week controls to reach the
printed range.

The script fails on persistence/calculation problems: missing day records,
negative durations, overlaps, or unknown category IDs. It also prints human
plausibility warnings without failing:

If a date batch receives HTTP 400, the script reports the backend error and
replays that date one event at a time, skipping only the individual events that
still receive HTTP 400. Other API failures stop the run immediately.

- zero or sub-five-minute blocks often indicate rapid duplicate transitions;
- uninterrupted Working/Learning blocks over two hours deserve a break review;
- Exercise blocks over two hours deserve a continuity review.

These warnings are deliberately separate from correctness. The backend should
calculate exactly what the event stream says, even when the event stream is not
physiologically realistic.

Useful options:

```text
--backup PATH       use another pending-events backup
--api URL           use another backend URL
--frontend URL      print another frontend URL
--username NAME     log in instead of creating a test account
--password VALUE    password for an existing account
--no-plan           skip creating the visual review template
--open              open the review page on macOS
```
**/

import { readFile, writeFile } from "node:fs/promises";
import { spawn } from "node:child_process";

const categories = [
  { sourceId: 1, name: "Working", color: "#2563eb", pomodoro_config: { work_duration: 2100, rest_duration: 300 } },
  { sourceId: 2, name: "Exercise", color: "#dc2626", pomodoro_config: null },
  { sourceId: 3, name: "Rest", color: "#0891b2", pomodoro_config: null },
  { sourceId: 4, name: "Learning", color: "#27b208", pomodoro_config: { work_duration: 1500, rest_duration: 300 } },
  { sourceId: 5, name: "Housework", color: "#e9a663", pomodoro_config: null },
];

const forcedPlan = [
  ["06:00:00", 3],
  ["08:00:00", 1],
  ["12:00:00", 3],
  ["13:00:00", 1],
  ["17:00:00", 2],
  ["19:00:00", 4],
  ["21:00:00", 3],
];

const argumentsMap = new Map();
for (let argumentIndex = 2; argumentIndex < process.argv.length; argumentIndex += 1) {
  const argument = process.argv[argumentIndex];
  if (!argument.startsWith("--")) continue;
  const [name, inlineValue] = argument.split("=", 2);
  argumentsMap.set(name, inlineValue ?? process.argv[++argumentIndex]);
}

if (argumentsMap.has("--help")) {
  console.log(`Usage: node scripts/pending-events-e2e.mjs [options]

Options:
  --backup PATH       Backup file (default: front-desktop-widget/pending-events-backup.json)
  --dump PATH         Write the exported /backup JSON here (default: pending-events-e2e-backup.json)
  --api URL           Backend URL (default: http://localhost:8080)
  --frontend URL      Frontend URL (default: http://localhost:5173)
  --username NAME     Reuse an existing test account
  --password VALUE    Password for --username
  --clone-username NAME  Fresh account used for the backup replication check
  --clone-password VALUE Password for --clone-username
  --utc-offset-minutes NUMBER  Numeric offset for local event timestamps (default: 0)
  --day-range-start TIME  Tracking day start (default: 07:00:00)
  --open              Open the frontend after loading the data
  The exported backup is posted back to /backup automatically to verify a full round trip.
  The review template and weekly schedule are always created from the fixed mock plan.`);
  process.exit(0);
}

const backupPath = argumentsMap.get("--backup") || "front-desktop-widget/pending-events-backup.json";
const dumpPath = argumentsMap.get("--dump") || "pending-events-e2e-backup.json";
const apiUrl = (argumentsMap.get("--api") || "http://localhost:8080").replace(/\/$/, "");
const frontendUrl = (argumentsMap.get("--frontend") || "http://localhost:5173").replace(/\/$/, "");
const reviewUrl = `${frontendUrl}/todo-planner/#/review`;
const username = argumentsMap.get("--username") || `pending-events-e2e-${Date.now()}`;
const password = argumentsMap.get("--password") || `PendingEvents-${Date.now()}!`;
const cloneUsername = argumentsMap.get("--clone-username") || `pending-events-clone-${Date.now()}`;
const clonePassword = argumentsMap.get("--clone-password") || `PendingClone-${Date.now()}!`;
const utcOffsetMinutes = Number(argumentsMap.get("--utc-offset-minutes") || 0);
const dayRangeStartTime = argumentsMap.get("--day-range-start") || "07:00:00";
const legacyDateMigrations = [];
if (!Number.isInteger(utcOffsetMinutes) || utcOffsetMinutes < -14 * 60 || utcOffsetMinutes > 14 * 60) {
  fail("--utc-offset-minutes must be an integer between -840 and 840");
}

function fail(message) {
  throw new Error(message);
}

class ApiError extends Error {
  constructor(path, status, body) {
    super(`${path} failed (${status}): ${body}`);
    this.path = path;
    this.status = status;
    this.body = body;
  }
}

async function request(path, options = {}) {
  const response = await fetch(`${apiUrl}${path}`, {
    ...options,
    headers: {
      ...(options.body ? { "Content-Type": "application/json" } : {}),
      ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}),
    },
  });
  const text = await response.text();
  let body = null;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = text;
  }
  if (!response.ok) throw new ApiError(`${options.method || "GET"} ${path}`, response.status, text);
  return body;
}

function localTimestamp(utcTimestamp) {
  const instant = new Date(utcTimestamp);
  if (Number.isNaN(instant.valueOf())) fail(`Invalid UTC timestamp: ${utcTimestamp}`);
  const shifted = new Date(instant.valueOf() + utcOffsetMinutes * 60 * 1000);
  const pad = (value) => String(value).padStart(2, "0");
  const sign = utcOffsetMinutes >= 0 ? "+" : "-";
  const absoluteOffset = Math.abs(utcOffsetMinutes);
  return `${shifted.getUTCFullYear()}-${pad(shifted.getUTCMonth() + 1)}-${pad(shifted.getUTCDate())}`
    + `T${pad(shifted.getUTCHours())}:${pad(shifted.getUTCMinutes())}:${pad(shifted.getUTCSeconds())}`
    + `${sign}${pad(Math.floor(absoluteOffset / 60))}:${pad(absoluteOffset % 60)}`;
}

function sourceEvents(entries) {
  if (!Array.isArray(entries) || entries.length === 0) fail("Backup must be a non-empty JSON array");
  return entries.map((entry, index) => {
    const event = entry?.event;
    if (!/^\d{4}-\d{2}-\d{2}$/.test(entry?.calendarDate || "")) fail(`Invalid calendarDate at entry ${index}`);
    if (!event?.client_event_id || !event?.event_type || !event?.occurred_at) fail(`Incomplete event at entry ${index}`);
    if (!["transition", "confirmation", "amendment"].includes(event.event_type)) fail(`Invalid event_type at entry ${index}`);
    if (event.event_type === "transition" && (!Number.isInteger(event.category_id) || event.category_id < 1 || event.category_id > 5)) fail(`Invalid mock category_id at entry ${index}`);
    if (Number.isNaN(Date.parse(event.occurred_at))) fail(`Invalid occurred_at at entry ${index}`);
    const occurredAtLocal = event.occurred_at_local || localTimestamp(event.occurred_at);
    if (Number.isNaN(Date.parse(occurredAtLocal)) || !/[+-]\d{2}:\d{2}$/.test(occurredAtLocal)) {
      fail(`Invalid occurred_at_local at entry ${index}`);
    }
    const calendarDate = occurredAtLocal.slice(0, 10);
    if (calendarDate !== entry.calendarDate) {
      legacyDateMigrations.push({ index, from: entry.calendarDate, to: calendarDate });
    }
    const correctedAtLocal = event.corrected_at
      ? (event.corrected_at_local || localTimestamp(event.corrected_at))
      : undefined;
    return {
      calendarDate,
      event: {
        client_event_id: event.client_event_id,
        event_type: event.event_type,
        ...(event.category_id == null ? {} : { category_id: event.category_id }),
        occurred_at: event.occurred_at,
        occurred_at_local: occurredAtLocal,
        ...(event.target_client_event_id ? { target_client_event_id: event.target_client_event_id } : {}),
        ...(event.corrected_at ? { corrected_at: event.corrected_at } : {}),
        ...(correctedAtLocal ? { corrected_at_local: correctedAtLocal } : {}),
      },
    };
  });
}

function grouped(entries) {
  const result = new Map();
  for (const entry of entries) {
    if (!result.has(entry.calendarDate)) result.set(entry.calendarDate, []);
    result.get(entry.calendarDate).push(entry.event);
  }
  for (const [date, events] of result) {
    events.sort((left, right) => new Date(left.occurred_at) - new Date(right.occurred_at));
    const eventIds = new Set();
    for (let index = 0; index < events.length; index += 1) {
      const event = events[index];
      if (eventIds.has(event.client_event_id)) fail(`Duplicate client_event_id ${event.client_event_id}`);
      eventIds.add(event.client_event_id);
    }
  }
  return result;
}

function dateRange(dates) {
  return [dates[0], dates[dates.length - 1]];
}

function minutes(time) {
  const [hours, minutesPart, seconds] = time.split(":").map(Number);
  return hours * 60 + minutesPart + (seconds || 0) / 60;
}

function timelineMinutes(time) {
  const clockMinutes = minutes(time);
  const startMinutes = minutes(dayRangeStartTime);
  return clockMinutes < startMinutes ? clockMinutes + 24 * 60 : clockMinutes;
}

function timeLabel(totalMinutes) {
  const normalized = Math.round(totalMinutes * 60) / 60;
  const dayOffset = Math.floor(normalized / (24 * 60));
  const clockMinutes = normalized % (24 * 60);
  const hours = Math.floor(clockMinutes / 60);
  const remainingMinutes = Math.floor(clockMinutes % 60);
  const seconds = Math.round((normalized % 1) * 60);
  const dayLabel = dayOffset > 0 ? ` (+${dayOffset}d)` : "";
  return `${String(hours).padStart(2, "0")}:${String(remainingMinutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}${dayLabel}`;
}

function humanWarnings(date, blocks, categoryNames) {
  const warnings = [];
  for (const block of blocks.filter((value) => value.block_type === "actual")) {
    const categoryName = categoryNames.get(block.category_id) || "Unknown";
    if (block.duration_minutes === 0) warnings.push(`${date}: zero-minute ${categoryName} block; likely rapid duplicate transitions`);
    else if (block.duration_minutes < 5) warnings.push(`${date}: ${block.duration_minutes}-minute ${categoryName} block; physiologically possible but probably transition noise`);
    if (["Working", "Learning"].includes(categoryName) && block.duration_minutes > 120) warnings.push(`${date}: ${categoryName} lasts ${block.duration_minutes} minutes without a break; review sustained-attention plausibility`);
    if (categoryName === "Exercise" && block.duration_minutes > 120) warnings.push(`${date}: Exercise lasts ${block.duration_minutes} minutes; review whether this is one continuous session`);
  }
  return warnings;
}

function printBlocks(date, blocks, categoryNames) {
  console.log(`\n${date}`);
  for (const block of blocks) {
    const category = block.category_id == null ? "Untracked" : categoryNames.get(block.category_id) || `Category ${block.category_id}`;
    const end = timeLabel(timelineMinutes(block.start_time) + block.duration_minutes);
    console.log(`  ${block.start_time} - ${end}  ${String(block.duration_minutes).padStart(4)}m  ${block.block_type.padEnd(9)} ${category}`);
  }
}

function plannedCategoryForStart(plan, startTime) {
  const start = timelineMinutes(startTime);
  let selectedBlock = plan.at(-1);
  for (const planBlock of plan) {
    if (timelineMinutes(planBlock.start_time) <= start) selectedBlock = planBlock;
  }
  return selectedBlock?.category_id;
}

function planMatchesExpected(plan, categoryIds) {
  if (!Array.isArray(plan) || plan.length !== forcedPlan.length) return false;
  return forcedPlan.every(([startTime, sourceId], index) => {
    const expectedDuration = minutes(forcedPlan[index + 1]?.[0] || "24:00:00") - minutes(startTime);
    const actual = plan[index];
    return actual.start_time === startTime
      && actual.category_id === categoryIds.get(sourceId)
      && actual.duration_minutes === expectedDuration;
  });
}

function circularMinuteDistance(firstMinute, secondMinute) {
  const difference = Math.abs(firstMinute - secondMinute) % (24 * 60);
  return Math.min(difference, 24 * 60 - difference);
}

async function createAccount() {
  const endpoint = argumentsMap.has("--username") ? "/auth/login" : "/auth/register";
  const response = await request(endpoint, { method: "POST", body: JSON.stringify({ username, password }) });
  return response.token;
}

async function createFreshAccount(accountUsername, accountPassword) {
  const response = await request("/auth/register", {
    method: "POST",
    body: JSON.stringify({ username: accountUsername, password: accountPassword }),
  });
  return response.token;
}

function canonicalBackup(backup) {
  const categoriesById = new Map((backup.categories || []).map((category) => [category.id, category.name]));
  const groupsById = new Map((backup.template_groups || []).map((group) => [group.id, group.name]));
  const templatesById = new Map((backup.templates || []).map((template) => [template.id, template.name]));
  const category = (categoryId) => categoryId == null ? null : categoriesById.get(categoryId) || `unknown:${categoryId}`;
  const template = (templateId) => templateId == null ? null : templatesById.get(templateId) || `unknown:${templateId}`;
  const plan = (blocks) => (blocks || []).map((block) => ({
    category: category(block.category_id),
    start_time: block.start_time,
    duration_minutes: block.duration_minutes,
  }));
  return {
    settings: backup.settings && {
      day_range_start_time: backup.settings.day_range_start_time,
      day_range_end_time: backup.settings.day_range_end_time,
    },
    categories: (backup.categories || []).map((value) => ({
      name: value.name,
      color: value.color,
      pomodoro_config: value.pomodoro_config,
    })).sort((left, right) => left.name.localeCompare(right.name)),
    groups: (backup.template_groups || []).map((value) => value.name).sort(),
    templates: (backup.templates || []).map((value) => ({
      name: value.name,
      group: value.template_group_id == null ? null : groupsById.get(value.template_group_id),
      plan: plan(value.plan),
    })).sort((left, right) => left.name.localeCompare(right.name)),
    weekly_schedule: (backup.weekly_schedule || []).map((value) => ({
      day_of_week: value.day_of_week,
      template: template(value.day_template_id),
    })).sort((left, right) => left.day_of_week - right.day_of_week),
    overrides: (backup.overrides || []).map((value) => ({
      calendar_date: value.calendar_date,
      template: template(value.day_template_id),
    })).sort((left, right) => left.calendar_date.localeCompare(right.calendar_date)),
    days: (backup.days || []).map((value) => ({
      calendar_date: value.calendar_date,
      timezone_offset_minutes: value.timezone_offset_minutes ?? null,
      timezone_offset_locked: value.timezone_offset_locked,
      plan: plan(value.plan),
      actual: (value.actual || []).map((block) => ({
        category: category(block.category_id),
        block_type: block.block_type,
        start_time: block.start_time,
        duration_minutes: block.duration_minutes,
      })),
    })).sort((left, right) => left.calendar_date.localeCompare(right.calendar_date)),
    events: (backup.events || []).map((value) => ({
      calendar_date: value.calendar_date,
      client_event_id: value.client_event_id,
      event_type: value.event_type,
      category: category(value.category_id),
      occurred_at: value.occurred_at,
      occurred_at_local: value.occurred_at_local,
      target_client_event_id: value.target_client_event_id || null,
      corrected_at: value.corrected_at || null,
      corrected_at_local: value.corrected_at_local || null,
    })).sort((left, right) => left.client_event_id.localeCompare(right.client_event_id)),
  };
}

async function submitEvents(date, events, deviceId, token, skippedEvents) {
  const requestBody = { device_id: deviceId, events };
  try {
    return await request(`/days/${date}/events`, { method: "POST", token, body: JSON.stringify(requestBody) });
  } catch (error) {
    if (!(error instanceof ApiError) || error.status !== 400) throw error;
    const acceptedEvents = [];
    for (const event of events) {
      try {
        const response = await request(`/days/${date}/events`, {
          method: "POST",
          token,
          body: JSON.stringify({ device_id: deviceId, events: [event] }),
        });
        acceptedEvents.push(...(response.accepted_events || []));
      } catch (individualError) {
        if (!(individualError instanceof ApiError) || individualError.status !== 400) throw individualError;
        skippedEvents.push({ date, event, reason: individualError.body });
      }
    }
    return { accepted_events: acceptedEvents };
  }
}

async function expectBadRequest(path, options) {
  try {
    await request(path, options);
  } catch (error) {
    if (error instanceof ApiError && error.status === 400) return;
    throw error;
  }
  fail(`${options.method || "GET"} ${path} unexpectedly succeeded`);
}

async function verifyDayEndpoints(date, token, categoryIds, originalActual) {
  const day = await request(`/days/${date}`, { token });
  if (day.calendar_date !== date) fail(`GET /days/${date} returned the wrong date`);
  const restorableBlocks = originalActual.map(({ category_id, block_type, start_time, duration_minutes }) => ({
    category_id,
    block_type,
    start_time,
    duration_minutes,
  }));

  const arbitraryBlocks = [
    { category_id: categoryIds.get(1), block_type: "actual", start_time: "08:48:35", duration_minutes: 233 },
    { category_id: categoryIds.get(3), block_type: "actual", start_time: "12:41:52", duration_minutes: 26 },
    { category_id: categoryIds.get(3), block_type: "actual", start_time: "13:08:32", duration_minutes: 74 },
    { category_id: categoryIds.get(1), block_type: "actual", start_time: "14:22:45", duration_minutes: 207 },
    { category_id: categoryIds.get(2), block_type: "actual", start_time: "17:50:13", duration_minutes: 30 },
  ];
  const replacementOptions = {
    method: "PUT",
    token,
    body: JSON.stringify({ actual: arbitraryBlocks }),
  };
  const replacement = await request(`/days/${date}/blocks`, replacementOptions);
  if (replacement.actual.length !== arbitraryBlocks.length) {
    fail(`PUT /days/${date}/blocks did not preserve arbitrary event blocks`);
  }
  const preciseBlock = replacement.actual.find(
    (block) => block.block_type === "actual" && block.duration_minutes === 233,
  );
  if (!preciseBlock || preciseBlock.start_time !== "08:48:35") {
    fail(`PUT /days/${date}/blocks changed the local precise start time`);
  }

  await expectBadRequest(`/days/${date}/blocks`, {
    method: "PUT",
    token,
    body: JSON.stringify({
      actual: [
        arbitraryBlocks[0],
        { ...arbitraryBlocks[1], start_time: "12:41:00" },
      ],
    }),
  });

  await request(`/days/${date}/blocks`, {
    method: "PUT",
    token,
    body: JSON.stringify({
      actual: restorableBlocks,
    }),
  });
}

async function main() {
  const entries = sourceEvents(JSON.parse(await readFile(backupPath, "utf8")));
  const eventsByDate = grouped(entries);
  const dates = [...eventsByDate.keys()].sort();
  const token = await createAccount();
  const device = await request("/devices", { method: "POST", token, body: JSON.stringify({ platform: "desktop" }) });
  await request("/settings", {
    method: "PUT",
    token,
    body: JSON.stringify({ day_range_start_time: dayRangeStartTime, day_range_end_time: "22:00:00" }),
  });
  const settings = await request("/settings", { token });
  if (settings.day_range_start_time !== dayRangeStartTime) fail("GET /settings returned the wrong start time");

  const categoryIds = new Map();
  const categoryNames = new Map();
  const existing = await request("/categories", { token });
  if (!Array.isArray(existing.categories)) fail("GET /categories returned no category list");
  for (const definition of categories) {
    let category = existing.categories.find((value) => value.name === definition.name);
    if (!category) category = await request("/categories", { method: "POST", token, body: JSON.stringify(definition) });
    categoryIds.set(definition.sourceId, category.id);
    categoryNames.set(category.id, category.name);
  }

  {
    const plan = forcedPlan.map(([start_time, sourceId], index) => {
      const nextStart = forcedPlan[index + 1]?.[0] || "24:00:00";
      return {
        start_time,
        category_id: categoryIds.get(sourceId),
        duration_minutes: minutes(nextStart) - minutes(start_time),
      };
    });
    const template = await request("/templates", { method: "POST", token, body: JSON.stringify({ name: "Pending events E2E plan", template_group_id: null, plan }) });
    await request("/schedule/weekly", { method: "PUT", token, body: JSON.stringify({ weekly_schedule: Array.from({ length: 7 }, (_, day_of_week) => ({ day_of_week, day_template_id: template.id })) }) });
  }

  let submitted = 0;
  const acceptedEventIds = new Set();
  const skippedEvents = [];
  for (const [date, sourceDateEvents] of eventsByDate) {
    const apiEvents = sourceDateEvents.map((event) => ({ ...event, category_id: categoryIds.get(event.category_id) }));
    const response = await submitEvents(date, apiEvents, device.device_id, token, skippedEvents);
    for (const event of response.accepted_events || []) acceptedEventIds.add(event.client_event_id);
    submitted += response.accepted_events?.length || 0;
  }

  const [from, to] = dateRange(dates);
  const response = await request(`/days?from=${from}&to=${to}`, { token });
  const records = new Map(response.days.filter((entry) => entry.day_record).map((entry) => [entry.calendar_date, entry.day_record]));
  let structuralErrors = 0;
  const warnings = [];
  const alignmentWarnings = [];
  const workingDayStarts = [];
  const firstDate = dates[0];
  const firstRecord = records.get(firstDate);
  if (!firstRecord) fail(`First day record ${firstDate} is missing`);
  if (firstRecord.timezone_offset_locked !== true) fail("First day offset was not locked");
  if (firstRecord.timezone_offset_minutes !== utcOffsetMinutes) {
    fail(`Expected offset ${utcOffsetMinutes}, got ${firstRecord.timezone_offset_minutes}`);
  }
  for (const date of dates) {
    const record = records.get(date);
    if (!record) { console.error(`ERROR ${date}: day record missing`); structuralErrors += 1; continue; }
    const blocks = record.actual || [];
    const workingCategoryId = categoryIds.get(1);
    const workingStarts = blocks
      .filter((block) => block.block_type === "actual" && block.category_id === workingCategoryId)
      .map((block) => block.start_time);
    if (workingStarts.length) {
      const firstWorkingStart = workingStarts[0];
      const distanceFromEight = circularMinuteDistance(
        timelineMinutes(firstWorkingStart),
        timelineMinutes("08:00:00"),
      );
      workingDayStarts.push({ date, firstWorkingStart, distanceFromEight });
    }
    if (!planMatchesExpected(record.plan, categoryIds)) {
      console.error(`ERROR ${date}: forced review plan does not match the expected schedule`);
      structuralErrors += 1;
    }
    const hasAcceptedLockingEvent = eventsByDate.get(date).some(
      (event) => acceptedEventIds.has(event.client_event_id)
        && ["transition", "confirmation"].includes(event.event_type),
    );
    if (hasAcceptedLockingEvent) {
      if (record.timezone_offset_locked !== true) {
        console.error(`ERROR ${date}: offset was not locked`);
        structuralErrors += 1;
      }
      if (record.timezone_offset_minutes !== utcOffsetMinutes) {
        console.error(`ERROR ${date}: expected offset ${utcOffsetMinutes}, got ${record.timezone_offset_minutes}`);
        structuralErrors += 1;
      }
    }
    printBlocks(date, blocks, categoryNames);
    let previousStart = -1;
    let previousEnd = -1;
    for (const block of blocks) {
       let start = minutes(block.start_time);
       if (previousStart >= 0) {
         while (start < previousStart) start += 24 * 60;
       } else if (start < minutes(dayRangeStartTime)) {
         start += 24 * 60;
       }
       if (block.duration_minutes < 0 || start < previousEnd) { console.error(`ERROR ${date}: overlapping or negative block`); structuralErrors += 1; }
       previousStart = start;
       previousEnd = Math.max(previousEnd, start + block.duration_minutes);
      if (block.category_id != null && !categoryNames.has(block.category_id)) { console.error(`ERROR ${date}: unknown category ${block.category_id}`); structuralErrors += 1; }
      const plannedCategoryId = plannedCategoryForStart(record.plan, block.start_time);
      if (block.category_id !== plannedCategoryId) {
        alignmentWarnings.push(`${date}: ${block.start_time} actual category ${block.category_id ?? "untracked"} differs from plan category ${plannedCategoryId ?? "untracked"}`);
      }
    }
    warnings.push(...humanWarnings(date, blocks, categoryNames));
    const dateEvents = eventsByDate.get(date);
    const amendmentsByTarget = new Map();
    for (const event of dateEvents.filter((value) => value.event_type === "amendment")) {
      amendmentsByTarget.set(event.target_client_event_id, event);
    }
    const transitionEvents = dateEvents.filter((event) => event.event_type === "transition" && acceptedEventIds.has(event.client_event_id));
    for (const event of transitionEvents) {
      const amendment = amendmentsByTarget.get(event.client_event_id);
      const expectedLocal = amendment?.corrected_at_local || event.occurred_at_local;
      const expectedStart = expectedLocal.slice(11, 19);
      if (!blocks.some((block) => block.block_type === "actual" && block.start_time === expectedStart)) {
        console.error(`ERROR ${date}: missing actual block at ${expectedStart} for ${event.client_event_id}`);
        structuralErrors += 1;
      }
    }
  }

  await verifyDayEndpoints(
    firstDate,
    token,
    categoryIds,
    firstRecord.actual || [],
  );

  const exportedBackup = await request("/backup", { token });
  await writeFile(dumpPath, `${JSON.stringify(exportedBackup, null, 2)}\n`, "utf8");
  if (!exportedBackup || exportedBackup.version !== 1) {
    fail("GET /backup returned an invalid backup document");
  }
  const importResult = await request("/backup", {
    method: "POST",
    token,
    body: JSON.stringify(exportedBackup),
  });
  if (importResult?.imported !== true) fail("POST /backup did not confirm the import");
  if (importResult.imported_events !== 0) {
    fail(`POST /backup was not idempotent (${importResult.imported_events} events imported)`);
  }

  const cloneToken = await createFreshAccount(cloneUsername, clonePassword);
  const cloneImportResult = await request("/backup", {
    method: "POST",
    token: cloneToken,
    body: JSON.stringify(exportedBackup),
  });
  if (cloneImportResult?.imported !== true) fail("Fresh-user backup import was not confirmed");
  if (cloneImportResult.imported_events !== (exportedBackup.events || []).length) {
    fail(`Fresh-user import restored ${cloneImportResult.imported_events} of ${(exportedBackup.events || []).length} events`);
  }
  const cloneBackup = await request("/backup", { token: cloneToken });
  if (JSON.stringify(canonicalBackup(exportedBackup)) !== JSON.stringify(canonicalBackup(cloneBackup))) {
    fail("Fresh-user backup does not match the original backup");
  }

  console.log(`\nLoaded ${submitted} events into ${records.size} day records.`);
  console.log(`Backup exported to: ${dumpPath}`);
  console.log("Backup import round trip: passed (no duplicate events imported).");
  console.log(`Fresh-user replication: passed (${cloneUsername}).`);
  console.log(`Skipped ${skippedEvents.length} events rejected with HTTP 400.`);
  console.log(`Frontend: ${reviewUrl}`);
  console.log(`Login: ${username} / ${password}`);
  console.log(`Review range: ${from} through ${to}`);
  console.log(`Locked offset: ${utcOffsetMinutes} minutes`);
  console.log(`Legacy calendar dates remapped to local dates: ${legacyDateMigrations.length}`);
  if (warnings.length) {
    console.log("\nHuman plausibility warnings (not persistence failures):");
    for (const warning of warnings) console.log(`  - ${warning}`);
  }
  if (alignmentWarnings.length) {
    console.log("\nPlan alignment warnings (actual activity differs from the forced plan):");
    for (const warning of alignmentWarnings) console.log(`  - ${warning}`);
  }
  console.log("\nWorking-day start analysis:");
  for (const workingDayStart of workingDayStarts) {
    const assessment = workingDayStart.distanceFromEight <= 90 ? "near 08:00" : "away from 08:00";
    console.log(`  - ${workingDayStart.date}: first Working block ${workingDayStart.firstWorkingStart} (${assessment})`);
  }
  const nearEightCount = workingDayStarts.filter((value) => value.distanceFromEight <= 90).length;
  console.log(`  ${nearEightCount}/${workingDayStarts.length} working days start within 90 minutes of 08:00.`);
  if (skippedEvents.length) {
    console.log("\nSkipped event details:");
    for (const skippedEvent of skippedEvents) {
      console.log(`  - ${skippedEvent.date} ${skippedEvent.event.client_event_id}: ${skippedEvent.reason}`);
    }
  }
  if (structuralErrors) fail(`${structuralErrors} structural block errors found`);
  if (argumentsMap.has("--open")) spawn("open", [reviewUrl], { stdio: "ignore", detached: true }).unref();
}

main().catch((error) => {
  console.error(`\nE2E failed: ${error.message}`);
  process.exitCode = 1;
});
