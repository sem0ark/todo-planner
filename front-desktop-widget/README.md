# Design specification
See [Design documenation](../docs/style%20system%20v1.md)

## The Shortcut Map
*Designed for zero-mouse latency.*

| Trigger | Action | UX Intent |
| :--- | :--- | :--- |
| `Space` | **Confirm Actual** | Validates current activity (State 1 -> 2). |
| `Enter` | **Sync to Plan** | In State 3, instantly transitions back to the planned block. |
| `1 - 9` | **Category Jump** | Transitions to the category at index $N$ in the right rail. |
| `[` / `]` | **Offset Nudge** | In State 3, shifts the last transition timestamp back by 5m increments. |
| `Cmd + Z` | **Undo** | Reverts the last transition or offset adjustment. |

## Spatial Layout
The widget uses a fixed **200px x 320px** footprint. It is divided into a **Dynamic Left Panel** (Current Reality) and a **Static Right Rail** (Category List).

```text
[ 200px Total Width ]
_______________________________________
| [Left Panel: 65%]  | [Right Rail: 35%] |
|                    |                   |
| [A] Offset Bar     | [C] Category List |
|     (+5) (+15)     |     - Work        |
|                    |     - Rest        |
| [B] Current        |     - Admin       |
|     Activity       |     - Health      |
|                    |                   |
| [D] Planned Block  |                   |
|     (State 3 Only) |                   |
|____________________|___________________|
```

- **[A] Offset Bar**
- **[B] Current Activity:** Large Domain Hue block.
- **[C] Category List:** Fixed order (Web-configured). Monospaced index numbers for quick-key reference.
- **[D] Planned Block:** Appears only when Actual diverges from Template.

## State Logic & Visual Encoding

### State 1: Confirmation Prompt
*Trigger: Planned block boundary reached.*
- **Left Panel:** Fills with 100% Saturation Domain Hue.
- **Animation:** 1.5s "Breathing" pulse (Opacity 100% -> 70%).
- **Logic:** Clicking the Left Panel or pressing `Space` logs a **Confirmation** at the boundary timestamp.

### State 2: Active/Idle
*Trigger: User is on-schedule.*
- **Left Panel:** Static Domain Hue.
- **Progress Bar:** A 4px `tabular-nums` countdown at the bottom of the panel. No pulsing.
- **Logic:** Designed to be ignored. Peripheral vision only sees a static color block.

### State 3: Off-Schedule / Distraction
*Trigger: User transitions to a non-planned category.*
- **Offset Bar (Top):** Displays `+5m` and `+30m` buttons. 
    - *Action:* Clicking `+5m` updates the `last_event_timestamp` in the background. The UI reflects the change instantly (Optimistic UI).
- **Split Left Panel:**
    - **Top (Actual):** Shows current distraction (e.g., "Rest").
    - **Bottom (Planned):** Shows what *should* be happening. 
    - **Animation:** The **Bottom (Planned)** section pulses (100ms `ease-in-out`) to signal it is a "Return to Plan" trigger.
- **Logic:** Clicking the pulsing bottom section logs a transition back to the plan at the current timestamp.

## The "Speed Choice" Justification
1.  **Fixed Right Rail:** By forbidding the widget from re-sorting categories, we build **spatial muscle memory**. A user learns that "Rest" is always 20px from the bottom, allowing for "no-look" logging.
2.  **Retroactive Offset:** Distractions are rarely logged the moment they start. By placing `+5/30` buttons at the top of the stack, we allow the user to correct reality in <1s without opening a full timeline editor.
3.  **Non-Shifting Schedule:** We treat the Template as a "North Star." By never shifting the plan, we maintain a constant visual baseline, making the "Gap" between reality and intent immediately legible.

### State Transition Timing
- **Pulse Duration:** 1500ms (State 1), 800ms (State 3).
- **Panel Swap:** 180ms `ease-out`.
- **Offset Feedback:** The timestamp text flashes `Cyan` (#00FFFF) for 100ms upon button press to confirm the background sync.

## Unit Testing

Tests are built directly with `swiftc` and run as a standalone binary due to problems with configuring the xcode + it is heavy, just a script should be enough for now.

Running Tests:
```bash
make test
```

This builds a test runner and executes all WidgetStateStore tests. All 19 tests currently pass.

Test file example: `Tests/WidgetTests/WidgetStateStoreTests.swift`

Adding New Tests:
1. Add a test method to `WidgetStateStoreTests` class
2. Register it in the `testMethods` array at the bottom of `TestRunner.main()`
3. Run `make test` — passes and failures are printed immediately
