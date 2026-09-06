## Vision
**Personal sustainability planner** for designing, executing, and refining versioned "ideal-day" templates across life states and seasons.

The app operates across two distinct surfaces that serve different moments of the day:
- **Live surface** — a persistent, minimal presence always available during the day. Captures activity transitions and confirmations with minimal interruption, without requiring the user to navigate or context-switch.
- **Day View** — an environment for reconstructing, adjusting, and analyzing actual blocks. Supports both live correction and retroactive reconstruction when live logging did not occur.

The mental model: Microsoft Clock (live, minimal, always present) + Obsidian Day Planner (timeline editing and review) - two separate surfaces, not one combined view.

The core question the app answers over time: "Is the life structure I designed for myself actually sustainable?"

## Goals
- **Design and version sustainable day structures** per life state and season. Templates are independent, named, and maintained in a personal library. "Create From" workflow allows iterative refinement without starting from scratch.
- **Capture reality with near-zero friction** - the live surface is the primary input mechanism, designed to stay out of the way during active work and surface only at natural transition points. The review surface provides a safety net: any day can be reconstructed/adjusted after the fact.
- **Make untracked time and unlogged days a visible signal** - gaps in the record are never silently ignored or treated as neutral. They are surfaced as potentially meaningful data, distinct from rest and distinct from intentional blank periods the user cannot recall.
- **Reveal template health over time** - the app accumulates the gap between planned structure and lived reality, and makes that gap legible: per-template, per-category, and across time windows the user selects. The goal is to surface whether a template is realistic, not to judge whether the user followed it.
- **Account for personal seasonal and contextual variation** - life state and season are first-class dimensions of the planning system. Templates reflect individual patterns (energy, schedule, environment) rather than general norms, and the user maintains distinct versions for distinct contexts.


## Glossary
Domain Entities:
- *Day Template* - a named ideal-day structure with retained snapshots representing the intended time distribution across categories for a given life state or season.
    - *Template Group* - a named personal context (e.g., full-time employment, vacation) that informs which Day Template is appropriate.
- *Schedule* - the user's assignment of Day Templates to actual days.
    - *Schedule Override* - a one-off assignment of a Day Template to a specific calendar date, superseding the Weekly Schedule for that date.
    - *Weekly Schedule* -> default assignment of Day Templates to days of the week, forming a repeating default structure. Can be changed any time. Individual calendar dates can override the repeating assignment.
- *Activity Category (Block Category for short)* - a top-level category of activity that classifies all time blocks, user defined.
- *Time Block (Block for short)* - a contiguous period of time assigned to a single category, with a defined start time and duration. The atomic unit of templates.
    - *Snapshot Block* - a category block stored within a Template Snapshot, representing intended activity for that saved template schedule.
    - *Blank Block* - a special actual block marking a period the user cannot recall. Distinct from rest and untracked time. Treated as a health signal.
    - *Actual Block* - a category block as recorded for a specific calendar day, representing what the user actually did.
    - *Untracked Block* - a gap in a day's actual record where no block of any type has been placed, treated as blank until the user does not adjust it during the day review.
- *Day Record* - the complete data for a single calendar day: the template in use, all actual blocks, skips, and blank blocks.
    - *Day event* - some event used for logging the real state of day. Can confirm that the activity is still in progress, can state transitions from one category to another. It is important to note that resulting timeline and blocks derived from the events may be different from blocks in the template.
        - *Confirmation* - user confirms staying focused on the activity.
        - *Transition* - the boundary event between two consecutive category blocks, where the user confirms moving from one activity to the next.
- *Day Boundary* - the configured time of day at which one calendar day ends and the next begins for tracking purposes. Defaults to 04:00 AM.

Synchronization:
- *Change Log* - a record of a change operations done on a specific device, storing timestamp and affected entity. Used for conflict resolution and synchronization.
- *Device* - a registered client belonging to a User, identified by id. Device registers itself and stores its id itself, used minly for differentiation in scope of synchronization.

Auth:
- *User* - the account that owns all data within the system.
- *User Settings* - a set of configuration values belonging to a User, including day boundary time.

Metrics:
- *Adherence* - the measured ratio of actual time spent on a category to the planned time for that category, calculated across a selected set of day records.
- *Template Health* - the aggregate analysis of how closely actual day records match a given Day Template over a user-selected time window.


# Feature Requirements
- All deletions are **soft** - entities become invisible to the user but are retained in storage. Hard delete occurs only on full account removal.
- Actual day blocks are **derived from Day Events** (Confirmations and Transitions) until the user replaces them with corrected actual blocks.
- Categories are **fully user-defined** - no hardcoded set. Defined by name, color, and an optional Pomodoro configuration.
- Analysis includes all Day Records in the selected time window.


## Activity Categories
User-defined top-level classifications for all time blocks across templates and actual days.

- A Category is defined by: **name** (user-defined string), **color** (user-selected), and an optional Pomodoro configuration with work and rest durations in minutes.
- No system-defined / hardcoded categories. The user creates all categories from scratch.
- A Category can be **renamed** or **recolored** at any time. Changes apply everywhere the category appears (templates, day records, analytics) immediately.
- Deletion is **soft**: a deleted category becomes invisible in the UI but is retained in storage. All historical blocks referencing it remain intact and continue to appear in analytics using the category's last known name and color.
- A soft-deleted category cannot be assigned to new blocks but remains visible in historical views where it was previously used.
- No limit on number of categories in v1.


## Day Templates
Named, versioned ideal-day structures representing intended time distribution across categories for a given life state or season.

Template structure:
- A Template contains general metadata. Its schedule is an ordered sequence of **Snapshot Blocks**, each with: category, start time, and duration.
- Block constraints: **minimum 30 minutes**, **15-minute step increments**.
- A Template has: name (user-defined), optional association to a Template Group, creation timestamp, and last-modified timestamp. Template snapshots contain the schedule blocks for each saved version of that template.
- Templates are **independent** - no inheritance between them.

Template management:
- User can **create** a new empty template.
- User can **Create From**: copy any existing template (including soft-deleted ones visible in history), rename the copy, then edit independently.
- User can **rename** a template at any time.
- User can **edit** a template's schedule via drag timeline (same UI as Day View Actual lane - see F6). Each schedule edit creates a new template snapshot; previous snapshots and their blocks are retained.
- Deletion is **soft**: deleted templates become invisible in the template library but are retained. Past Day Records retain their pinned template snapshot, which is unaffected by later template changes or deletion. Active/future Day Records can follow the template's current snapshot through automatic or explicit re-pinning.
- Template library: flat list in v1. No grouping enforced by the system, though Template Groups provide optional labeling.

Template Groups:
- A Template Group is a named label (e.g., `Full-Time Employment`, `Vacation`) that can be assigned to one or more templates.
- Groups are user-defined, optional, and informational only in v1 - they do not drive any scheduling or analysis logic.
- Deletion is **soft**.


## Schedule
The user's assignment of Day Templates to actual calendar days, forming the planned structure against which reality is measured.

Weekly Schedule:
- The user assigns a Day Template to each day of the week (Monday–Sunday), forming a repeating default structure.
- Any day of the week can be left unassigned.
- The Weekly Schedule can be changed at any time. Changes apply to active and future dates; past Day Records retain their snapshot assignment. Existing active/future Day Records are re-resolved when their assignment changes. A schedule assignment resolves to a template; a day record stores both that template and the snapshot currently applied to it.

Schedule Overrides:
- The user can assign a specific Day Template to a specific calendar date, superseding the Weekly Schedule for that date only.
- An override can be removed, reverting that date to the Weekly Schedule assignment.
- Overrides are visible in the Weekly Schedule view as distinct from the repeating assignment.

Unassigned Days:
- If a day has no template assigned (neither via Weekly Schedule nor Override), the app prompts the user to assign one when that day is opened or becomes the current day.
- An unassigned day with no template and no Day Events recorded is treated as fully untracked.


## Day Record & Day Events
The complete data record for a single calendar day. Actual blocks are derived from Day Events during live tracking and can be replaced during review.

Each Day Record contains:
- The calendar date.
- The logical Day Template assigned to the date.
- The Template Snapshot currently applied to the day. This snapshot can change while its date is active or future, but is fixed for past dates.
- All Day Events recorded for that day (in order).
- The current actual blocks, including any review corrections or reconstruction.

Template changes:
- Updating a Day Template creates a new Template Snapshot.
- Existing active/future Day Records using that template are automatically re-pinned to the new snapshot. Their events and actual blocks are unchanged because the plan changed, not reality.
- A user can explicitly re-pin an active/future day by choosing another template or re-resolving the current schedule.
- Past Day Records retain their snapshot and are frozen from re-pinning, but their actual blocks remain editable.

Day Boundary
- The Day Boundary is a user-configured time of day at which one calendar day ends and the next begins for tracking purposes.
- Default: **04:00 AM**.
- Configurable in User Settings.
- All events and blocks are assigned to a calendar day based on this boundary.
- A day is **active** when it is the current tracking date, **future** when it follows the current tracking date, and **past** once its tracking date has passed. Past days keep their snapshot assignment, while actual blocks remain editable.

*Day Events* are the atomic inputs that drive the Actual Block timeline.
Two types for now:
- **Confirmation**
  - The user confirms that the current activity is still in progress.
  - Logs: timestamp, current category.
  - Used to track ongoing activity without creating a transition boundary.
- **Transition**
  - The user confirms moving from one category to the next.
  - Logs: timestamp and category.
  - Creates a boundary between two consecutive Actual Blocks.
  - Note that may signal transition both between planned categories and plan overrides, for example, when user takes an unexpected break or records a time region when they got distracted.

Actual Block Derivation:
- The Actual Block timeline for a day is **derived** from the ordered sequence of Day Events during live tracking.
- Untracked gaps remain distinct from submitted actual and blank blocks and are derived from the gaps in the current timeline.

#### Block Types in Derived Timeline

| Block Type | Description |
|---|---|
| **Actual Block** | A contiguous period assigned to a category, derived from events or supplied as a review correction. |
| **Blank Block** | A user-marked period the user cannot recall. Minimum 30 min, 15-min steps. Treated as a health signal, not as rest or untracked. |
| **Untracked** | A gap in the day with no block of any type placed. Shown as empty/dimmed. Distinct from Blank. |

## Live Widget
A persistent, minimal UI element always visible during the day on desktop and mobile. Handles Day Event input (Confirmations and Transitions). Never requires navigation for its core function.

### Core Display
- **Always-On Status**: Shows current category name (large, bold, uppercase) and progress bar indicating time remaining in current planned block.
- **Category List (Right Rail)**: Fixed vertical list of all categories. Each category shows:
  - Color indicator dot with a distinct active state
  - Category name (uppercase, abbreviated if needed)
  - Keyboard shortcut index (1-9) for direct jump
- **Elapsed Time Display**: Shows elapsed time in current activity (MM:SS format) during active states.
- **No Additional Information**: Widget surface displays only what's needed for immediate action. Detailed analysis lives in the web review surface.

### State-Specific Visuals

#### State 1: Prompted (Block Boundary)
- **Appearance**: Full-width category block for the planned category with a visually distinct treatment.
- **Animation**: A breathing animation indicates that user action is required.
- **Affordance**: Tap to confirm, or press 1-9 to switch activity.
- **Persistence**: No auto-dismiss. Waits for user action.

#### State 2: Active (Any Activity)
- **Core Display**: Current category name (large, uppercase) + progress bar showing block progress.
- **Progress Indicator**: bar showing % elapsed, with remaining time. Updates real-time.
- **Pomodoro** (when enabled for category): Circular ring tracking work/rest phases, with distinct phase and completion states. Toggle via Space.
- **Schedule Deviation** (when off-plan): Subtle info banner showing planned category + return button (→ [Category]).
- **Offset Adjustments** (retroactive time correction): Keyboard `[`/`]` for ±5m, UI buttons for +5m/+15m increments. Recalculates `occurredAt` and updates timeline.
- **Key Principle**: Full feature set (pomodoro, offsets, category switch) available *always*. On/off schedule is purely informational.

### Transition & Confirmation Mechanics

**Boundary Detection & Prompts**:
- At every planned block boundary, widget enters State 1 (Prompted).
- Prompt displays the next planned category, offering choices:
  - **Space/Return**: Confirm the planned category → transition to Active.
  - **Key 1-9**: Switch to selected category → stay in Active (may be on/off plan).
  - **No action**: Prompt persists until user acts (no auto-dismiss).

**Off-Plan Tracking**:
- Selecting an unplanned category → user enters Active state with that category.
- Widget displays schedule deviation indicator (subtle banner).
- All features available: pomodoro (if enabled), offset adjustments, category switching.
- Return-to-plan button visible when diverged, labeled with planned category name.

#### Live Nudging (Offset Adjustment)
- **Feature Summary**: Retroactively adjust transition timestamps when logging is delayed.
- **Value**: Solves friction without requiring full day review. User remembers activity started 5 min ago → two key presses to correct.
- **When Available**: Always in Active state.
- **Controls**:
  - Keyboard: `[` nudge -5m, `]` nudge +5m (activity started earlier/later).
  - UI: +5m, +15m buttons.
- **Behavior**: Each nudge recalculates `occurredAt`, updates derived timeline, displays cumulative offset (e.g., "T-10m").
- **Limits**: Cannot move before previous event or after current time.

### Keyboard Shortcut Map

| Trigger | Action | State(s) | Intent |
|---|---|---|---|
| `Space` / `Return` | Confirm (Prompted) / Pomodoro Toggle (Active) | State 1, State 2 | Confirm planned activity or toggle pomodoro phase. |
| `1` - `9` | Category Jump | All | Switch to category at index N in right rail. |
| `[` / `]` | Offset Nudge ±5m | State 2 | Adjust transition timestamp (retroactive correction). |
| `Enter` (from State 2) | Return to Plan | State 2 (when off-plan) | Explicit return to planned category (if diverged). |
| `Cmd+Z` | Undo | State 2 | Revert last offs≈et/category change (future: v2). |

### Pomodoro Integration
- **Per-Category Config**: Each category can have an optional pomodoro configuration (work duration + rest duration in minutes).
- **Work Phase**: Timer counts up. When elapsed time reaches work duration, timer enters rest phase.
- **Rest Phase**: Timer counts up. User can:
  - Press Space to return to work phase early.
  - Let timer auto-advance to work phase if rest duration expires.
- **Visual Indicator**: Circular progress ring appears only in State 2 (Active) when pomodoro is enabled for the current category.
- **Notifications**: Sends a notification when pomodoro timer completes (transitions from work to rest or rest to work).

#### Platform Behavior & Implementation Details

**Desktop App (macOS Widget):**
- Small, floating window, always on top, independent of other app windows. Made of two main panels with activity information and category selection.
- Navigation:
  - Keyboard-primary: All state changes via `Space, Return, 1-9, [, ]` keys.
  - Mouse secondary: Click left panel for confirmation, click categories on right rail to jump.
- **Sync Integration**: Manual sync button visible in right rail footer. Shows last sync timestamp on hover.
- **Web App Link**: "Open Web" button in right rail footer for accessing the full review surface.
- **Authentication**: Uses stored token after initial web login. No login UI on desktop widget after first auth.
- **Menu Bar Icon**: Updates based on widget state (active, confirmation needed), while schedule deviation remains an informational widget indicator.

**Mobile App (Future - v2):**
- **Surface**: Persistent notification (non-dismissable during active day).
- **Interaction**: Tapping notification opens app to Transition Confirmation Screen.
- **Confirmation Screen**: Shows prompt, category chips (1-9 selectable), sync button (pull-to-refresh), *Open Web* link.
- **Return to Background**: After confirming transition, app auto-minimizes.


## Review: Day View
Retroactive reconstruction and review of a single calendar day. Primary logging surface. Accessed via the web app only.

#### Opening State
- Opens with the current actual blocks, or with the active template snapshot as the starting state when no actual blocks exist.
- Two visual lanes:
  - **Planned lane** - the pinned template snapshot's blocks, static, always visible for reference. Not editable.
  - **Actual lane** - editable. Pre-filled from template, adjusted by the user to match reality.

#### Editing Interactions

| Interaction | Result |
|---|---|
| Drag block edge | Resize (15-min snap, 30-min minimum) |
| Drag block body | Reposition within the day |
| Tap block | Opens category picker to relabel |
| Tap midpoint + drag | Splits block into two |
| Paint time region as Blank | Creates a Blank Block (min 30 min, 15-min steps) |

- Editing actual blocks is available for active, future, and past days. Snapshot assignment is editable only for active and future days.


## Review: Week View
A 7-day overview of actual theme distribution. Default landing view of the web app. Surfaces unlogged days.

- Displays a 7-day horizontal strip. Each day = a vertical bar divided by category colors (Actual data only).
- Planned template is **not** shown in this view.
- **Unlogged days** (with no Actual blocks): blank and flagged (visual indicator - border or icon). No ghost template shown.
- **Gaps within logged days**: visible as empty/dimmed segments within the day bar.
- **Blank Blocks**: shown as a distinct pattern (e.g., hatched) within the day bar.
- User can navigate to any past week.
- Tapping any day bar navigates to that day's Day View.


## User Settings
User-level configuration values stored server-side and synced across devices.
- **Day Boundary time**: configurable time of day at which the tracking day resets. Default: 04:00 AM.
- **Manual sync trigger**: button to push/pull data with the server. Displays last sync timestamp.
- **Account management**: accessible from Settings. Includes full account deletion (hard delete of all data).


## Template Health View
Aggregate analysis of how closely actual Day Records match a given Day Template over a user-selected time window. Lives inside the Template Editor on the web app.

Overlay view:
- All Day Records that used this template (within the selected time window), overlaid as semi-transparent actual-day shapes on top of the template structure.
- Color matches category colors. Visual intensity reflects frequency - more days at a given block position should be more prominent.
- Shape divergence from the template = visible drift.

#### Distribution Comparison
Per-category breakdown for the selected time window:

| Category | Planned (% of day) | Planned (hrs/day) | Actual avg (hrs/day) | Δ |
|---|---|---|---|---|
| Working | 33% | 8h | 9.5h | +1.5h |
| Exercising | 8% | 2h | 0.5h | -1.5h |
| Untracked / Blank | - | - | 1.5h | - |

- Untracked and Blank are reported separately.
- All Day Records in the selected time window are included.

#### Time Window
- User-selectable: *Last N days* input.
- Persists per template until changed by the user.


## Overview / Analytics
Cross-template summary of category adherence and unlogged activity. Separate tab in the web app.

- **Adherence per category**: ratio of actual average time to planned time, across all Day Records in the selected time window. e.g., `Exercising: 25% - planned 2h/day, actual avg 0.5h/day`
- **Weekly gap strip**: mini 7-day strip showing unlogged day gaps across recent weeks. Visual only - no numbers in this view.
- No drill-down from this tab. Template Health is accessed via the Template Editor.


## Synchronization
Manual, device-aware data sync between the web app and native clients.

- Sync is **manual** in v1. No background or real-time sync.
- Each client is registered as a **Device** with a unique ID stored locally.
- A **Change Log** records all write operations per device: timestamp and affected entity.
- **Conflict resolution**: most recently modified version (by Change Log timestamp) wins. User is notified of any overwrite on next open.
- **Desktop**: Sync button in the widget. Triggers push/pull with server.
- **Mobile**: Pull-to-refresh on the Transition Confirmation Screen. Triggers push/pull with server.
- **Web**: Sync button in Settings. Triggers push/pull with server.
- Last sync timestamp visible on all three platforms.


## Authentication
User account and session management. Handled on the web app only.

- All data is scoped to a **User** account.
- Authentication is handled exclusively on the web app.
- Desktop and mobile apps use a **stored token** after initial web login. No login UI on native clients.
- Token storage follows platform security best practices (keychain on desktop/mobile).
- Full account deletion (triggered from Settings) performs a **hard delete** of all user data. This is the only hard delete operation in the system.
