## Vision
**Personal sustainability planner** for designing, executing, and refining versioned "ideal-day" templates across life states and seasons.

The app operates across two distinct surfaces that serve different moments of the day:
- **Live surface** — a persistent, minimal presence always available during the day. Captures activity transitions and confirmations with minimal interruption, without requiring the user to navigate or context-switch.
- **Review surface** — an intentionally opened environment for reconstructing, adjusting, and analyzing past days. Supports both light correction of live-logged data and full retroactive reconstruction when live logging did not occur. Opened intentionally, not during active work.

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
- *Day Template* - a named, versioned ideal-day structure representing the intended time distribution across categorys for a given life state or season.
    - *Template Group* - a named personal context (e.g., full-time employment, vacation) that informs which Day Template is appropriate.
- *Schedule* - the user's assignment of Day Templates to actual days.
    - *Schedule Override* - a one-off assignment of a Day Template to a specific calendar date, superseding the Weekly Schedule for that date.
    - *Weekly Schedule* -> default assignment of Day Templates to days of the week, forming a repeating default structure. Can be changed any time. Individual calendar dates can override the repeating assignment.
- *Activity Category (Block Category for short)* - a top-level category of activity that classifies all time blocks, user defined.
- *Time Block (Block for short)* - a contiguous period of time assigned to a single category, with a defined start time and duration. The atomic unit of templates.
    - *Planned Block* - a category block as defined within a Day Template, representing intended activity.
    - *Blank Block* - a special actual block marking a period the user cannot recall. Distinct from rest and untracked time. Treated as a health signal.
    - *Actual Block* - a category block as recorded for a specific calendar day, representing what the user actually did.
    - *Untracked Block* - a gap in a day's actual record where no block of any type has been placed, treated as blank until the user does not adjust it during the day review.
- *Day Record* - the complete data for a single calendar day: the template in use, all actual blocks, skips, blank blocks, and review status.
    - *Day event* - some event used for logging the real state of day. Can confirm that the activity is still in progress, can state transitions from one category to another. It is important to note that resulting timeline and blocks derived from the events may be different from blocks in the template.
        - *Confirmation* - user confirms staying focused on the activity.
        - *Transition* - the boundary event between two consecutive category blocks, where the user confirms moving from one activity to the next.
    - *Review Status* - the state of a Day Record:
        - `Unreviewed` - included in analysis
        - `Reviewed` - included in analysis
        - `Ignored` - excluded from analysis
- *Day Boundary* - the configured time of day at which one calendar day ends and the next begins for tracking purposes. Defaults to 04:00 AM.

Synchronization:
- *Change Log* - a record of a change operations done on a specific device, storing timestamp and affected entity. Used for conflict resolution and synchronization.
- *Device* - a registered client belonging to a User, identified by id. Device registers itself and stores its id itself, used minly for differentiation in scope of synchronization.

Auth:
- *User* - the account that owns all data within the system.
- *User Settings* - a set of configuration values belonging to a User, including day boundary time and skip reason preferences.

Metrics:
- *Adherence* - the measured ratio of actual time spent on a category to the planned time for that category, calculated across a set of reviewed day records.
- *Template Health* - the aggregate analysis of how closely actual day records match a given Day Template over a user-selected time window.


# Feature Requirements
- All deletions are **soft** - entities become invisible to the user but are retained in storage. Hard delete occurs only on full account removal.
- Actual day blocks are **derived from Day Events** (Confirmations and Transitions) plus any retroactive edits applied during review.
- Categories are **fully user-defined** - no hardcoded set. Defined by name and color only.
- Analysis includes both `Unreviewed` and `Reviewed` Day Records. `Ignored` records are excluded.


## Activity Categories
User-defined top-level classifications for all time blocks across templates and actual days.

- A Category is defined by: **name** (user-defined string) and **color** (user-selected).
- No system-defined / hardcoded categories. The user creates all categories from scratch.
- A Category can be **renamed** or **recolored** at any time. Changes apply everywhere the category appears (templates, day records, analytics) immediately.
- Deletion is **soft**: a deleted category becomes invisible in the UI but is retained in storage. All historical blocks referencing it remain intact and continue to appear in analytics using the category's last known name and color.
- A soft-deleted category cannot be assigned to new blocks but remains visible in historical views where it was previously used.
- No limit on number of categories in v1.


## Day Templates
Named, versioned ideal-day structures representing intended time distribution across categories for a given life state or season.

Template structure:
- A Template contains an ordered sequence of **Planned Blocks**, each with: category, start time, and duration.
- Block constraints: **minimum 30 minutes**, **15-minute step increments**.
- A Template has: name (user-defined), optional association to a Template Group, creation timestamp, last-modified timestamp.
- Templates are **independent** - no inheritance between them.

Template management:
- User can **create** a new empty template.
- User can **Create From**: copy any existing template (including soft-deleted ones visible in history), rename the copy, then edit independently.
- User can **rename** a template at any time.
- User can **edit** a template's blocks via drag timeline (same UI as Day View Actual lane - see F6).
- Deletion is **soft**: deleted templates become invisible in the template library but are retained. Historical Day Records that used a deleted template retain a **snapshot** of the template at the time of use - the live template and the snapshot are decoupled after assignment.
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
- The Weekly Schedule can be changed at any time. Changes apply to future dates only - past Day Records retain the template snapshot that was active at the time.

Schedule Overrides:
- The user can assign a specific Day Template to a specific calendar date, superseding the Weekly Schedule for that date only.
- An override can be removed, reverting that date to the Weekly Schedule assignment.
- Overrides are visible in the Weekly Schedule view as distinct from the repeating assignment.

Unassigned Days:
- If a day has no template assigned (neither via Weekly Schedule nor Override), the app prompts the user to assign one when that day is opened or becomes the current day.
- An unassigned day with no template and no Day Events recorded is treated as fully untracked.


## Day Record & Day Events
The complete data record for a single calendar day. Actual blocks are derived from Day Events and retroactive edits - not stored as independent block entities separate from events.

Each Day Record contains:
- The calendar date.
- A snapshot of the Day Template active for that date at the time of first event or review open.
- All Day Events recorded for that day (in order).
- All retroactive edits applied during review.
- Review Status: `Unreviewed`, `Reviewed`, or `Ignored`.

Day Boundary
- The Day Boundary is a user-configured time of day at which one calendar day ends and the next begins for tracking purposes.
- Default: **04:00 AM**.
- Configurable in User Settings.
- All events and blocks are assigned to a calendar day based on this boundary.

*Day Events* are the atomic inputs that drive the Actual Block timeline.
Two types for now:
- **Confirmation**
  - The user confirms that the current activity is still in progress.
  - Logs: timestamp, current category.
  - Does not create a new block - extends the current one.
- **Transition**
  - The user confirms moving from one category to the next.
  - Logs: timestamp, outgoing category, incoming category.
  - Creates a boundary between two consecutive Actual Blocks.
  - Note that may signal transition both between planned categories and plan overrides, for example, when user takes an unexpected break or records a time region when they got distracted.

Actual Block Derivation:
- The Actual Block timeline for a day is **derived** from the ordered sequence of Day Events. plus any retroactive edits.
- Derived blocks are not stored as separate entities - they are computed from the event log.
- Retroactive edits (applied in Day View during review) are stored as edit events in the same log, preserving the full history of changes.

#### Block Types in Derived Timeline

| Block Type | Description |
|---|---|
| **Actual Block** | A contiguous period assigned to a category, derived from events or retroactive edits. |
| **Blank Block** | A user-marked period the user cannot recall. Minimum 30 min, 15-min steps. Treated as a health signal, not as rest or untracked. |
| **Untracked** | A gap in the day with no block of any type placed. Shown as empty/dimmed. Distinct from Blank. |

#### Review Status

| Status | Behavior |
|---|---|
| `Unreviewed` | Default state. Day is editable. Included in analysis. |
| `Reviewed` | User has marked the day complete. Day is locked from further editing. Included in analysis. |
| `Ignored` | User has permanently dismissed the day. Excluded from analysis. Not editable. Still navigable (read-only). |

- Transition from `Unreviewed` -> `Reviewed` is triggered by a user toggle. **Permanent.**
- Transition from `Unreviewed` -> `Ignored` is triggered by an explicit Ignore action. **Permanent.**
- No transition from `Reviewed` or `Ignored` back to `Unreviewed` in v1.


## Live Widget
A persistent, minimal UI element always visible during the day on desktop and mobile. Handles Day Event input (Confirmations and Transitions). Never requires navigation for its core function.

Display (always on):
- Shows at all times: **current category name** + **time remaining until next planned block**.
- Display is constant regardless of current category (including Rest, Outside, Exercising).
- No additional information displayed in v1.

Transition Prompts:
- At every planned block boundary, the widget surfaces a prompt: *"[Next Block Category] - Start"*.
- **Start** -> one tap. Logs a Transition event (outgoing -> incoming category). Prompt dismisses.
- Prompt does not auto-dismiss - persists until the user acts.
- Missed prompt (no tap) = gap logged as Untracked in the derived timeline.

#### Platform Behavior

**Desktop App:**
- Small floating window, always on top.
- Transition prompt appears inline within the widget.
- Contains: status display, transition prompt (when active), Sync button, *Open Web* link.

**Mobile App:**
- Persistent notification (non-dismissable during an active day).
- Tapping notification opens the app to a Transition Confirmation Screen.
- Transition Confirmation Screen contains: prompt, inline chips, Sync trigger (pull-to-refresh), *Open Web* link.
- After confirmation, app returns to background.


## Review: Day View
Retroactive reconstruction and review of a single calendar day. Primary logging surface. Accessed via the web app only.

#### Opening State
- Opens pre-filled with the active template snapshot as the starting state of the Actual lane.
- Two visual lanes:
  - **Planned lane** - template snapshot, static, always visible for reference. Not editable.
  - **Actual lane** - editable. Pre-filled from template, adjusted by the user to match reality.

#### Editing Interactions

| Interaction | Result |
|---|---|
| Drag block edge | Resize (15-min snap, 30-min minimum) |
| Drag block body | Reposition within the day |
| Tap block | Opens category picker to relabel |
| Tap midpoint + drag | Splits block into two |
| Paint time region as Blank | Creates a Blank Block (min 30 min, 15-min steps) |

- All edits are stored as edit events in the Day Record's event log.
- Editing is available while status is `Unreviewed`. Locked when `Reviewed` or `Ignored`.

#### Review Status Actions
- **Mark Reviewed** toggle: transitions Day Record to `Reviewed`. Permanent. Locks editing.
- **Ignore** action: transitions Day Record to `Ignored`. Permanent. Removes day from unreviewed count and analysis.
- Ignored days remain navigable in read-only mode.


## Review: Week View
A 7-day overview of actual theme distribution. Default landing view of the web app. Surfaces unlogged and unreviewed days.

- Displays a 7-day horizontal strip. Each day = a vertical bar divided by category colors (Actual data only).
- Planned template is **not** shown in this view.
- **Unreviewed days** (status = `Unreviewed` with no Actual blocks): blank and flagged (visual indicator - border or icon). No ghost template shown.
- **Gaps within logged days**: visible as empty/dimmed segments within the day bar.
- **Blank Blocks**: shown as a distinct pattern (e.g., hatched) within the day bar.
- **Ignored days**: shown as explicitly dismissed (distinct visual state, not flagged as unreviewed).
- Status message: *"X days not yet reviewed this week."* Counts only days that are neither `Reviewed` nor `Ignored`.
- Tapping the status message navigates to the earliest `Unreviewed` day's Day View.
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
- Color matches category colors. Opacity reflects frequency - more days at a given block position = more opaque.
- Shape divergence from the template = visible drift.

#### Distribution Comparison
Per-category breakdown for the selected time window:

| Category | Planned (% of day) | Planned (hrs/day) | Actual avg (hrs/day) | Δ |
|---|---|---|---|---|
| Working | 33% | 8h | 9.5h | +1.5h |
| Exercising | 8% | 2h | 0.5h | -1.5h |
| Untracked / Blank | - | - | 1.5h | - |

- Untracked and Blank are reported separately.
- Only `Reviewed` and `Unreviewed` Day Records are included. `Ignored` records are excluded.

#### Time Window
- User-selectable: *Last N days* input.
- Persists per template until changed by the user.


## Overview / Analytics
Cross-template summary of category adherence and unlogged activity. Separate tab in the web app.

- **Adherence per category**: ratio of actual average time to planned time, across all non-Ignored Day Records. e.g., `Exercising: 25% - planned 2h/day, actual avg 0.5h/day`
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
