## Widget

**Style Overview**

The macOS widget is a compact, dark, glance-first menu-bar popover. It uses a fixed `320 × 200 px` layout with a `208 px` left activity panel and a `112 px` right category rail. The implementation differs from the README wireframe, which describes `200 × 320 px`.

The visual language is based on:

- Deep navy surfaces: `#001a24` outer background and `#003448` widget surface.
- High-saturation category colors for active work or activity blocks.
- Light structural colors: `#dee2ef` muted text and `#afb6cf` borders.
- White, high-contrast primary text.
- Compact spacing, subtle dividers, rounded outer corners, and limited shadows.
- System fonts in the implementation. The design specification calls for Inter, while timers and numeric data should use JetBrains Mono. The code uses the platform system font and SwiftUI’s monospaced system font instead.
- Outer corner radius: `12 px`; inner controls and buttons: approximately `4 px`.
- No decorative imagery; status is communicated through color, text, animation, progress indicators, and menu-bar symbols.

**1. Authentication Loading View**

- **View:** Authentication-checking state in `ContentView`.
- **How it looks:** A full `320 × 200 px` navy panel with a centered white circular progress indicator and a small “Checking authentication...” message underneath.
- **Colors and typography:**
  - Background: `#003448`.
  - Progress indicator: white.
  - Message: `#dee2ef`.
  - Font: system font, approximately `10 px`.
  - Vertical spacing: approximately `8 px`.
- **Behavior:** This is a temporary passive state shown while stored credentials are checked.

**2. Login View**

- **View:** `LoginView`, displayed when the user is unauthenticated.
- **How it looks:** A vertically stacked authentication form inside the dark widget surface. It contains a title, browser-login button, JWT token field, confirmation button, and optional error text.
- **Colors and typography:**
  - Background: `#003448`.
  - Title: white, bold system font, `24 px`.
  - Browser-login button: blue background, white semibold text, `16 px`, `40 px` high.
  - Token field: medium slate overlay based on `#91a6be` at roughly `30%` opacity; monospaced system font, `14 px`.
  - Confirm button: green when populated, gray when disabled; semibold system font, `14 px`, `40 px` high.
  - Error message: red system font, `12 px`.
  - Horizontal padding: `16 px`.
- **Behavior:** Authentication can be initiated through the browser or by entering a JWT token directly. Deep-link tokens are inserted automatically.

**3. Authenticated Widget Shell**

- **View:** The main authenticated `ContentView`.
- **How it looks:** A fixed dark popover split into two permanent panes:
  - Left panel: `208 px`, approximately 65% of the width.
  - Right rail: `112 px`, approximately 35% of the width.
- **Colors and geometry:**
  - Surface: `#003448`.
  - Outer border: `#afb6cf` at approximately `20%` opacity.
  - Outer radius: `12 px`.
  - Shadow: black at approximately `50%` opacity with a `12 px` radius and downward offset.
  - Pane divider: subtle structural-border line.
- **Behavior:** The left panel changes according to activity state, while the right rail remains spatially fixed to support quick keyboard and mouse selection.

**4. Initializing Activity View**

- **View:** `.initializing` state inside `LeftPanelView`.
- **How it looks:** A plain navy activity panel with a centered white circular progress indicator.
- **Colors and typography:**
  - Background: `#003448`.
  - Progress indicator: white.
  - No additional text or controls.
- **Behavior:** Used while schedule and category data are being loaded.

**5. Confirmation Prompt View**

- **View:** `ConfirmationPromptView`, representing State 1.
- **How it looks:** The entire left panel is filled with the planned category’s domain color. The category name appears in the upper-left corner in large uppercase lettering. The panel subtly expands and fades in a breathing animation.
- **Colors and typography:**
  - Background: the planned category color, typically a high-saturation domain hue.
  - Text: white.
  - Category name: uppercase, black-weight system font, `20 px`, maximum two lines.
  - Content padding: `12 px` horizontal and vertical.
  - Animation opacity: `100%` down to approximately `65%`.
  - Animation duration: `2 seconds`, repeating with ease-in-out timing.
- **Behavior:** Clicking anywhere in the panel or pressing `Space` confirms the current activity.

**6. Active Activity View**

- **View:** `ActiveView`, representing the normal active/idle state.
- **How it looks:** A full-height category-colored block with a large uppercase activity name at the top and a compact progress bar along the bottom.
- **Colors and typography:**
  - Background: current category color.
  - Main category name: white, uppercase, black-weight system font, `20 px`.
  - Progress track: black at approximately `20%` opacity.
  - Progress fill: white at approximately `60%` opacity.
  - Elapsed-time label: white at approximately `60%` opacity, monospaced system font, `10 px`, tabular digits.
  - Content padding: `12 px`.
  - Progress bar height: `5 px`; nominal width: `140 px`.
- **Optional Pomodoro indicator:**
  - Circular timer diameter: `60 px`.
  - Stroke width: `5 px`.
  - Work phase ring: white.
  - Rest phase ring: complementary color to the category.
  - Progress animation: linear over approximately `300 ms`.
- **Behavior:** The panel is intentionally visually stable when the user is on schedule, allowing it to be read peripherally.

**7. Off-Schedule Activity View**

- **View:** The deviation variant of `ActiveView`, representing State 3.
- **How it looks:** A compact offset-control bar is added above the colored activity block. The main block shows the actual activity and a smaller expected activity label.
- **Colors and typography:**
  - Offset bar background: `#003448` at approximately `10%` overlay.
  - Offset text: green `#10b981`, monospaced bold font, `10 px`.
  - Offset buttons: navy background, white monospaced text, `10 px`, with a muted green border.
  - “RETURN” button: white monospaced text, `10 px`, translucent navy background.
  - Expected activity label: white at approximately `70%` opacity, bold system font, `10 px`.
  - Offset bar height: `28 px`.
- **Controls:** `+5m`, `+15m`, and `RETURN`.
- **Behavior:** Offset changes are applied optimistically. Returning to the plan transitions back to the expected category.

**8. No Category Selected View**

- **View:** Fallback state in `ActiveView` when no current category exists.
- **How it looks:** A centered two-line message on a plain navy background.
- **Colors and typography:**
  - Background: `#003448`.
  - “Select a category”: muted text, medium system font, `14 px`.
  - “Press 1-9 to start”: muted text at approximately `70%` opacity, small system font, `10 px`.
  - Vertical spacing: approximately `8 px`.
- **Behavior:** Instructs the user to select a category using the numeric keyboard shortcuts.

**9. Right Category Rail**

- **View:** `RightRailView`.
- **How it looks:** A fixed vertical list of categories with compact rows, followed by “Open Web” and “Logout” actions.
- **Colors and typography:**
  - Background: `#003448`.
  - Category indicator: `6 × 6 px` circular dot using the category’s domain color.
  - Active indicator: same color with a small glow.
  - Active category name: white.
  - Inactive category name: `#91a6be`.
  - Category names: uppercase, bold system font, `10 px`, with `1.2 px` character tracking.
  - Shortcut numbers: structural border color at approximately `40%` opacity, monospaced system font, `10 px`.
  - Row height: `28 px`.
  - Row horizontal padding: `12 px`.
  - Active-row background: `#91a6be` at approximately `20%` opacity.
  - Row separators: very subtle structural-border lines.
- **Behavior:** Categories remain in a fixed order. This supports spatial memory and direct `1–9` keyboard selection.

**10. Rail Utility Actions**

- **View:** “Open Web” and “Logout” buttons at the bottom of `RightRailView`.
- **How it looks:** Two narrow, full-width utility rows separated by subtle horizontal lines.
- **Colors and typography:**
  - Text and SF Symbols: `#91a6be`.
  - Font size: approximately `10 px`.
  - Vertical padding: `8 px`.
  - Background: `#003448`.
- **Behavior:** “Open Web” launches the browser application; “Logout” stops refresh activity and clears authentication.

**11. Menu-Bar Status Icon**

- **View:** The AppKit menu-bar integration managed by `MenuBarManager`.
- **How it looks:** A template SF Symbol in the macOS menu bar.
- **Icons:**
  - Idle: `checkmark.circle`.
  - Active: `checkmark.circle.fill`.
  - Confirmation required: `bell.fill`.
- **Colors and typography:** Uses the system template icon tint rather than custom widget colors.
- **Behavior:** Clicking the icon opens or closes the transient `320 × 200 px` popover. The popover also opens automatically for confirmation and completed Pomodoro events.
