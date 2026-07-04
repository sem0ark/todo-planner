# Core Philosophy

## Universal Core Principles
The following principles apply across all platforms to ensure a consistent interaction model.
- *Spatial Permanence*: elements must not shift position during loading or state changes.
    - Use fixed-width/height containers to prevent layout shifts.
    - *Skeleton screens* must match exact pixel dimensions of final content.
    - Contain scrollable regions within bounded areas.
- *Chromatic Activity Encoding*: color functions as data, not decoration.
    - Assign distinct, high-saturation hues to functional domains. EX: Inventory = Amber, Transactions = Cyan.
    - Represent state changes (active, loading, error) via brightness/saturation shifts within the domain hue, never a new hue.
    - Status indicators **must** use both shape and color for accessibility and high-glare readability.
- *Time-Feedback Loop*: every action triggers an immediate response.
    - *Optimistic UI*: interface reflects success instantly while background processes sync.
    - **Latency Targets**:
        - Micro-feedback (button press): 80-120ms.
        - State changes (panel swap): 150-180ms.
        - Navigation: 200-250ms.
        - Maximum limit: 300ms.
- *Transition as Utility*: animations only confirm intent or show directionality.
    - EX: Right-to-left motion for drill-down.
    - Duration must be <200ms using `ease-out` or `linear` timing; never delay start.
- *Resilience & Fault Tolerance*:
    - *Undo* is a global requirement for the last $N$ actions.
    - Confirmation dialogs are **forbidden** except for irreversible destruction.
    - Offline states must be visually distinct but non-blocking for cached data.
    - Auto-preserve form states to prevent data loss.


## Platform-Specific Requirements

### Desktop requirements
Desktop interfaces are optimized for high-throughput operations using precision pointing devices and keyboard input.

*Focus: Maximizing throughput via screen real estate and peripheral input.*
- *Home Row Philosophy*: design for zero-mouse latency. Every primary action must have a documented keyboard shortcut.
- *Command-K Palette* - global, context-aware palette for all navigation and system functions.
- *Multi-Pane Persistence*: use master-detail views and side-panels. Navigation requiring a "Back" button is a design failure.
- *Density*: use the "12px Rule" (tight spacing/borders). Use `tabular-nums` (monospaced) for all numeric data to facilitate vertical scanning.
- *Tab Order* must follow logical task sequence, not Document Object Model (DOM) order.

### Website requirements (Connectivity & Accessibility)
*Focus: Universal access, performance, and browser-native efficiency.*
- *Focus Management*: programmatically set focus to the most likely next input on page load or modal open.
- *Input Masking*: auto-format dates, currency, and IDs; users must never manually type separators (dashes/slashes).
- *Sacred Ground*: use viewport corners for fixed-position status indicators (health, sync) readable via peripheral vision.
- *Zero-Layout Shift (CLS)*: define heights for all containers to prevent content jumping during async fetches.
- *Progressive Disclosure*: display only parameters required for the primary task. Advanced options expand in-place rather than navigating to new pages.

### Mobile requirements (Velocity & Reach)
*Focus: One-handed operation, haptic feedback, and high-glare legibility.*
- *Thumb-Zone Architecture*: place primary actions (Submit, Add, Search) in the bottom 33% of the screen; top area is read-only.
- *Gesture-as-Shortcut*: implement short/long-swipe logic for list items (EX: swipe left to archive) and "Pull-to-Action" for creation.
- *Tactile Feedback*: use distinct haptic patterns. Success = light double-tap; Error = heavy vibration/thud.
- *High-Glare UI*: target WCAG AAA contrast using "True Black" (#000) and "Pure White" (#FFF) with bold weights.
- **Touch Targets**: minimum 48x48px; 64x64px preferred for high-velocity contexts.

### Desktop Widget Requirements
*Glance-First Architecture*: deliver primary value without interaction via high-density typography.
- *Information Hierarchy*: the most volatile data point must occupy the largest visual area.
- *Ephemeral Interaction*: restricted to micro-tasks (<5s). 
- **Prohibited**: deep navigation or multi-step workflows. Use *in-place toggles* or *deep-linking* to parent apps for complexity.

Widget State Logic:
- *Passive State*: Read-only, minimal CPU usage.
- *Active State*: Triggered by focus; reveals 2-3 primary controls.
- *Configuration State*: Overlay/flipped view for settings (API keys, intervals).

Widget Anti-Patterns:
- **Scrollbars**: indicates data is too complex; use pagination or parent app links.
- **Authentication Loops**: must inherit credentials from parent application.
- **Auto-Play Media**: audio/video triggers are strictly forbidden.

System Integration & Performance:
- *Z-Index Management*: support "Always on Top" and "Desktop Integrated" (wallpaper pin) modes.
- *Click-through*: allow interaction with background windows during passive states.
- *Resource Throttling*: scale polling frequency by visibility.
    - `Visible -> High Frequency` (~1s) -> `Obscured -> Low Frequency` (60s+).
- **NB!** Auto-suspend widgets exceeding memory thresholds.
- *Edge-Anchored Snap*: snap to grid/edges to maintain *Spatial Permanence*.

Implementation Metrics (KPIs)
- *Time-to-Insight (TTI)*: duration required to process data (Target: <2s).
- *Idle Wake-ups*: CPU wake-up events (Target: <10/hr).
- *Pixel Footprint*: data-to-chrome ratio (Target: >80% data).
- *Glanceability Score*: readability rating from a 2-meter distance.


## Anti-Patterns to Explicitly Avoid
- **Hover-Only Affordances:** If it requires a hover to see, it doesn't exist for mobile or power users.
- **Skeleton Shifting:** Skeletons that don't match the final UI dimensions, causing a layout "jump."
- **The "Hidden Meatball":** Hiding frequent primary actions inside a `...` menu.
- **Decorative Motion:** Any animation that exists for "delight" rather than "direction."
- **Color-Only Status:** Using color without a shape or icon (fails accessibility and sunlight tests).

## Implementation Metrics (KPIs)

To evaluate the effectiveness of a high-productivity interface, the following metrics are used:
- **Interaction Count:** The number of discrete actions required to complete a primary task must be <3.
- **Visual Search Latency:** The time required for a user to identify system status via peripheral vision.
- **Input Friction:** The presence of automatic masking and context-aware keyboard types (e.g., numeric pads for currency fields).

## Output Requirements
When asked to design a feature or component, your response must include:
1. **The Shortcut Map:** A list of keyboard/gesture triggers for the task.
2. **Spatial Layout:** An ASCII or descriptive wireframe showing fixed anchors.
3. **State Logic:** How the UI handles "Optimistic" success and "In-place" errors.
4. **The "Speed Choice":** A brief justification of why this layout maximizes throughput.
5. **Technical Implementation (only when asked):** Production-ready code (Tailwind/CSS) using semantic tokens for color and spacing.


## Visual Language & Industrial Palette

### The "Navy-Slate" Color System
All surfaces use the following architectural palette. Color is used strictly for structural hierarchy and state signaling.

| Role | Hex Code | RGB | Usage |
| :--- | :--- | :--- | :--- |
| **Base Void** | `#003448` | `(0,52,72)` | Primary background for Dark Mode / Widget Shell. |
| **Secondary Surface** | `#91a6be` | `(145,166,190)` | Inactive block fills / Secondary card backgrounds. |
| **Structural Border** | `#afb6cf` | `(175,182,207)` | Dividers, borders, and inactive icon states. |
| **Muted Text** | `#dee2ef` | `(222,226,239)` | Labels, metadata, and "Future" block outlines. |
| **Primary Text/Highlight** | `#f0f0f0` | `(240,240,240)` | High-contrast data, active timers, and headers. |

### Chromatic Activity Encoding (CAE) - The Focus-Hue Rule
To prevent "Skittles Effect" (visual noise), color is applied via **Temporal Desaturation**:
1.  **The Present (Active):** 100% Saturation of the user-defined Domain Hue (e.g., Amber, Cyan, Emerald). This is the **only** colored element on the screen.
2.  **The Past (Logged):** Grayscale. Fill: `#91a6be` at 20% opacity. Border: `#afb6cf`.
3.  **The Future (Planned):** Grayscale. No fill. Border: `#dee2ef` (Dashed 2px).
4.  **The Gap (Untracked):** Neutral Slate (`#91a6be`) with a 45° hatched pattern. Turns **Amber** only during the "Review" phase.

### Typography
- **Semantic Text (Inter):** Used for Category names, notes, and UI labels.
- **Data & Metrics (JetBrains Mono):** Used for all Timers, Timestamps, and Durations.
    - **Requirement:** `font-variant-numeric: tabular-nums` to ensure zero-layout shift during countdowns.


# Interaction Model & State Logic

### The Time-Feedback Loop
- **Micro-feedback:** Button presses trigger a `+2px` Y-axis "sink" and a 10% brightness shift within **80ms**.
- **Optimistic UI:** All logs are recorded locally and reflected instantly. Syncing happens in the background.
- **Undo:** `Cmd+Z` (Desktop) or "Shake/Swipe-to-Undo" (Mobile) is a global requirement for the last 5 actions.

### The Shortcut Map (Global)
| Trigger | Action | Context |
| :--- | :--- | :--- |
| `Space` | **Confirm** | "Still doing the current activity." |
| `Enter` | **Transition** | Move to the next planned block in the Template. |
| `1-9` | **Instant Switch** | Transition to Category at Index $N$. |
| `[` / `]` | **Nudge** | Shift current block boundary by +/- 5 minutes. |
| `Cmd+K` | **Command Palette** | Global search and navigation. |

# Implementation Metrics

1.  **Interaction Count:** Transitioning from one activity to another must never exceed **1 action** (if planned) or **2 actions** (if unplanned).
2.  **Visual Search Latency:** A user must be able to identify "Off-Schedule" status in **<300ms** via the pulsing state of the Active Block.
3.  **Zero-Layout Shift (CLS):** 0.0. All containers (Timers, Block Lists) must have fixed pixel dimensions to prevent "jumping" during data updates.
4.  **Time-to-Insight:** The "Gap" (Untracked time) must be visually quantifiable at a glance using the hatched pattern density.



# Technical Implementation Tokens (CSS)

```css
:root {
  /* Architectural Palette */
  --app-navy: #003448;
  --app-slate-blue: #91a6be;
  --app-slate-grey: #afb6cf;
  --app-cloud: #dee2ef;
  --app-snow: #f0f0f0;

  /* State Colors */
  --state-error: #ff3b30;
  --state-warning: #ffcc00;
  --state-success: #34c759;

  /* Typography */
  --font-ui: 'Inter', sans-serif;
  --font-data: 'JetBrains Mono', monospace;

  /* Geometry */
  --radius-outer: 24px;
  --radius-inner: 16px;
  --radius-button: 8px;
}

.active-block {
  background-color: var(--domain-hue);
  color: var(--app-snow);
  box-shadow: 0 0 20px var(--domain-hue-glow);
  transition: all 120ms ease-out;
}

.future-block {
  border: 2px dashed var(--app-cloud);
  background: transparent;
  color: var(--app-slate-grey);
}

.untracked-gap {
  background: repeating-linear-gradient(
    45deg,
    var(--app-navy),
    var(--app-navy) 10px,
    var(--app-slate-blue) 10px,
    var(--app-slate-blue) 11px
  );
}
```