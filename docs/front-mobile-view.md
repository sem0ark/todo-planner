## Mobile Style

**Overall Style**

`front-rn-mobile` uses a dark, high-contrast “navy-slate” interface with color encoding the active category.

- **Surfaces:** deep void `#001a24` for the safe area; base navy `#003448` for the application.
- **Text:** white `#ffffff` for primary content; muted `#dee2ef`; secondary slate `#91a6be`.
- **Borders:** structural `#afb6cf`, generally rendered with low opacity.
- **Typography:** Inter for interface text and category names; JetBrains Mono for indexes, actions, offsets, and other data-like labels.
- **Geometry:** compact spacing based on `4 / 8 / 12 / 16 / 24px`; minimum category row height is `56px`.
- **Visual language:** uppercase labels, bold weights, wide letter spacing, flat rectangular blocks, minimal decoration, and saturated category colors used as activity indicators.

## Main Views

### 1. Loading View

**What view we describe:**  
The initial state while fonts and notification setup are loading.

**How the view looks:**  
A full-screen navy background with a centered white activity spinner. No content or navigation is visible.

**Related definitions:**

- Background: base void `#003448`
- Spinner: primary text `#ffffff`
- Layout: centered both vertically and horizontally
- Component: `app/_layout.tsx`

### 2. Main Live Activity View

**What view we describe:**  
The normal active state showing the currently selected category and its progress.

**How the view looks:**  
The upper portion of the screen is dominated by a large full-width activity block filled with the selected category’s color. The category name appears in large uppercase black-weight text near the top-left. A small circular indicator appears below it, followed by a large flexible spacer and a thin progress bar at the bottom.

The lower portion contains the scrollable category list. The screen is divided approximately into a `3:2` ratio: the activity area uses `flex: 3`, and the list uses `flex: 2`.

**Related definitions:**

- Activity block: dynamic category color, for example:
  - Working `#2563eb`
  - Exercise `#dc2626`
  - Rest `#0891b2`
  - Learning `#16a34a`
  - Housework `#e9a663`
- Category title:
  - Font: `Inter_900Black`
  - Size: `28px`
  - Color: `#ffffff`
- Activity block padding: `24px`
- Progress track: `5px` high, black at approximately `20%` opacity
- Progress fill: white at `60%` opacity
- Component: `src/components/Active.tsx`

### 3. Boundary Confirmation Prompt

**What view we describe:**  
A temporary state shown when the schedule reaches a new planned block and the user must confirm it.

**How the view looks:**  
The upper activity area becomes a single large pressable rectangle using the planned category’s color. The category name is displayed in uppercase at the top-left. “TAP TO CONFIRM” is positioned at the bottom-left as a secondary instruction.

The category list remains visible below the divider.

**Related definitions:**

- Prompt background: planned category color
- Category name:
  - Font: `Inter_900Black`
  - Size: `28px`
  - Color: `#ffffff`
- Confirmation hint:
  - Font: `JetBrainsMono_700Bold`
  - Size: `12px`
  - Color: `#ffffff`
  - Opacity: `70%`
  - Letter spacing: `2px`
- Prompt padding: `24px`
- Component: `src/components/Shell.tsx`

### 4. Category List

**What view we describe:**  
The selectable list of activity categories shown beneath the primary activity area.

**How the view looks:**  
A vertically scrollable navy list made of equal-height rows. Each row contains:

1. A small circular color marker.
2. An uppercase category name.
3. A right-aligned numeric index.

The selected category receives a translucent slate highlight. Rows are separated by very subtle horizontal lines.

**Related definitions:**

- Row height: `56px`
- Horizontal padding: `16px`
- Color marker: `10 × 10px`, fully circular
- Category name:
  - Font: `Inter_700Bold`
  - Size: `14px`
  - Color: `#91a6be`
  - Letter spacing: `1.2px`
- Index:
  - Font: `JetBrainsMono_400Regular`
  - Size: `12px`
  - Color: `#91a6be`
- Selected-row background: `#91a6be` at `20%` opacity
- Row divider: `#afb6cf` at approximately `5%` opacity
- Component: `src/components/CategoryList.tsx`

### 5. Off-Schedule Active View

**What view we describe:**  
The active state when the selected category differs from the category planned for the current time.

**How the view looks:**  
A compact control strip appears above the colored activity block. It shows the elapsed offset in green, followed by `+5M` and `RETURN` actions. The activity block below also displays the expected planned category.

**Related definitions:**

- Offset strip height: `48px`
- Offset strip background: base navy at `10%` overlay
- Offset value:
  - Font: `JetBrainsMono_700Bold`
  - Color: offset green `#10b981`
- Actions:
  - Font: `JetBrainsMono_400Regular`
  - Size: `12px`
  - Color: white at `70%` opacity
- Expected category:
  - Font: `Inter_700Bold`
  - Size: `12px`
  - Color: white at `70%` opacity
- Component: `src/components/Active.tsx`

### 6. Empty Selection State

**What view we describe:**  
The fallback state when no current category is available.

**How the view looks:**  
The upper activity area is empty except for centered instructional text: “SELECT A CATEGORY BELOW”. The category list remains available underneath.

**Related definitions:**

- Instruction font: `Inter_700Bold`
- Instruction color: muted text `#dee2ef`
- Centered vertically and horizontally
- Component: `src/components/Active.tsx`

**Implementation note:** The app is a single-screen widget rather than a multi-route mobile application. Its main visual views are state variants of the same shell: loading, boundary prompt, active activity, off-schedule activity, and empty selection.

---

