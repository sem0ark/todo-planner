## Web

**Front-Web Style**

- Dark “Navy-Slate” visual system centered on `#001a24` app-void backgrounds, with `#003448` navy surfaces.
- Primary text uses Snow `#f0f0f0`; secondary text uses Cloud `#dee2ef`.
- Borders and dividers use Slate Grey `#afb6cf` at roughly 20% opacity; cards and controls primarily use solid Navy `#003448`, with Slate Blue `#91a6be` used for hover, muted text, and translucent states.
- Error, success, and warning colors are `#ff3b30`, `#34c759`, and `#ffcc00`.
- UI font: `Inter`, falling back to `system-ui, sans-serif`.
- Data, times, and durations use `JetBrains Mono`, falling back to `Courier New, monospace`, with tabular numerals.
- Outer containers use 24px radius; inner controls generally use 8px radius, despite the shared 16px inner-radius token.
- Standard transition duration is 120ms.
- Layout is centered with a maximum width of 1152px; individual content views generally use 768px or less.
- Interaction style is restrained: light hover color changes, subtle vertical button movement, border highlighting on focus, and drag shadows.
- The interface is mostly flat and utilitarian, with dark navy cards, thin low-opacity borders, compact typography, and high contrast. Most spacing and component styling is supplied through Tailwind utility classes.

**Authentication View**

- **What view we describe:** Sign-in and account-registration form shown when no authentication token exists.
- **How the view looks:** Full-screen navy background with a centered, medium-width card. The card contains a title, username and password fields, an optional error message, a primary submit button, and a link-style mode switch between sign-in and registration.
- **Colors and typography:**
  - Page background: App Void `#001a24`.
  - Card background: Navy `#003448`.
  - Card border: Slate Grey `#afb6cf` at 20% opacity.
  - Card radius: 24px.
  - Heading: 30px, semibold, Snow.
  - Labels and secondary text: 14px, medium/regular, Cloud.
  - Inputs: 16px text, navy translucent background, 2px Slate Grey border.
  - Primary button: 16px semibold, Snow background with Navy text.
  - Error message: 14px, Error red with translucent red background.
- Inputs receive a Cloud border and darker navy surface on focus.

**Application Shell and Navigation**

- **What view we describe:** Authenticated application frame surrounding all main routes.
- **How the view looks:** A dark full-height shell with a horizontal top navigation bar. “Todo Planner” appears at the left, followed by links for Home, Review, Categories, Templates, and Schedule. Logout is aligned to the right. The active route is brighter and semibold.
- **Colors and typography:**
  - Shell and navigation background: App Void `#001a24`.
  - Navigation uses the app-void surface with backdrop blur.
  - Bottom border: Slate Grey at 20% opacity.
  - Product title: 20px, semibold, Snow.
  - Navigation links: 14px; active links use Snow and semibold weight, inactive links use Cloud.
  - Main content: centered within a 1152px container with 24px horizontal and 32px vertical padding.

**Home View**

- **What view we describe:** Landing page at `/` after authentication.
- **How the view looks:** Centered welcome section followed by a responsive two-column card grid. Four cards provide entry points to Categories, Templates, Schedule, and the Desktop Widget. A small authentication-token link appears below the grid.
- **Colors and typography:**
  - Main heading: 36px, bold, Snow.
  - Greeting/subtitle: 18px, Cloud.
  - Cards: Navy `#003448` with Slate Grey borders at 20% opacity and 8px radius.
  - Card icons: approximately 36px emoji glyphs.
  - Card titles: 20px, semibold, Snow.
  - Card descriptions: 14px, Cloud.
  - Hover state changes the card surface to Slate Blue at 10% opacity.
  - Token link: 14px, Cloud with underline; hover changes to Snow.

**Categories View**

- **What view we describe:** Category management page at `/categories`.
- **How the view looks:** A compact list-management view. The header contains the “Categories” title and a “New Category” primary button. Categories appear as horizontal bordered rows with a color swatch, category name, optional Pomodoro configuration, and Edit/Delete actions. Creating or editing opens an inline form above the list.
- **Colors and typography:**
  - View width: up to 768px.
  - Page title: 24px, semibold, Snow.
  - Rows and editor panel: Navy surface, Slate Grey border at 20% opacity, 8px radius.
  - Category swatch: 32px square with 2px border and 8px radius.
  - Category name: regular/medium text, Snow.
  - Metadata and labels: 14px, Cloud.
  - Pomodoro durations: JetBrains Mono, 14px, tabular numerals.
  - Color choices: 40px squares using a generated 12-color pastel palette.
  - Delete action: Error red.
  - Primary actions: Snow background with Navy text.
- The Pomodoro section is separated with a top border and exposes numeric duration fields only when enabled.

**Template Library View**

- **What view we describe:** Template list at `/templates`.
- **How the view looks:** A simple flat library with a title and “New Template” button. Each template is represented by a bordered row showing its name, planned-block count, Edit action, and Delete action.
- **Colors and typography:**
  - View width: up to 768px.
  - Title: 24px, semibold, Snow.
  - Template rows: Navy surface, Slate Grey border at 20% opacity, 8px radius.
  - Template name: regular/medium, Snow.
  - Block count: 14px, Cloud.
  - Edit action: 14px, Cloud, becoming Snow on hover.
  - Delete action: 14px, Error red.
  - Empty state: centered 16px Cloud text with vertical padding.

**Template Editor View**

- **What view we describe:** New or edit template screen at `/templates/new` and `/templates/edit/:id`.
- **How the view looks:** A full-width editor with a title row, template-name input, vertical timeline, and Save/Cancel actions. The timeline shows 24 hours with a fixed time-label column and colored, draggable category blocks. Clicking a block opens an edit popover; blocks can be moved or resized vertically.
- **Colors and typography:**
  - Editor title: 24px, semibold, Snow.
  - Field labels: 14px, medium, Cloud.
  - Name input: 16px, Snow, navy translucent background, 2px Slate Grey border.
  - Timeline container: Navy translucent background, Slate Grey border, 8px radius.
  - Time labels: 14px JetBrains Mono, Cloud, tabular numerals.
  - Category blocks: user-selected category colors with automatically selected black or white contrast text.
  - Block labels: 14px, medium/semibold.
  - Block time and duration: 14px JetBrains Mono with tabular numerals.
  - Selected block: Snow focus ring.
  - Dragging block: reduced opacity and elevated shadow.
  - Edit popover: Navy background, Slate Grey border, 8px radius; on mobile it becomes a bottom sheet.
- The editor grid uses 1px per minute and snaps movement/resizing to 15-minute intervals. The timeline scrolls vertically within a maximum height of 70% of the viewport.

**Review View**

- **What view we describe:** Weekly plan-versus-actual review page at `/review`.
- **How the view looks:** A horizontally scrollable seven-day grid headed by week navigation controls and a Today button. Each day has a 24-hour vertical timeline with the planned portion on the left and recorded actual work on the right. A Schedule link provides a route back to schedule management.
- **Colors and typography:**
  - The grid uses App Void, low-opacity Slate Grey borders, and muted Slate Blue monospace hour labels.
  - Day names use Cloud; dates and hour labels use JetBrains Mono with tabular numerals.
  - Category blocks use their configured category colors with small Snow labels.
  - Loading days show a Slate Blue translucent pulse placeholder; API errors display in Error red.
- The grid is 1px per minute, 1440px tall, and has a 64px time-label column plus seven day columns. Review blocks are read-only; drag-and-drop is only enabled in the template editor.

**Schedule View**

- **What view we describe:** Combined `/schedule` route containing Weekly Schedule and Schedule Overrides.
- **How the view looks:** Two vertically stacked management sections using the same compact form-and-list pattern.

**Weekly Schedule section**

- Displays Monday through Sunday as seven bordered rows.
- Each row contains the day name, a template dropdown, and the currently saved assignment.
- Unsaved changes reveal Reset and Save Changes actions in the section header.
- Day labels use medium Snow text; controls use 16px Snow text with navy translucent backgrounds.
- The section is limited to 768px width.
- Empty-template guidance is shown as centered 14px Cloud text.

**Schedule Overrides section**

- Contains an “Add Override” form with date and template controls, followed by a Future Overrides list.
- Override rows show a date in monospace, the assigned template, and a red Remove action.
- Section headings are 24px for the main title and 18px semibold for subsections.
- Dates use JetBrains Mono; form labels and metadata use Cloud.
- Form panels and list rows use Slate Blue at 10% opacity, Slate Grey borders, and 8px radius.

**Desktop Widget Token View**

- **What view we describe:** Authentication-success page at `/token`, used to transfer the web token to the macOS widget.
- **How the view looks:** A centered success card with a green circular checkmark, success heading, welcome message, automatic-transfer notice, copy-to-clipboard button, and manual setup instructions.
- **Colors and typography:**
  - Card: Navy surface with Slate Grey border at 20% opacity, 24px radius, and backdrop blur.
  - Success icon: 64px green circle using `#34c759`, with 30px Navy checkmark.
  - Heading: 24px, semibold, Snow.
  - Welcome message: regular Cloud.
  - Success notice: Snow text, translucent green background, green border.
  - Main button: 16px semibold, Snow background, Navy text.
  - Instructions: 14px Cloud inside a secondary bordered panel.

**Fallback View**

- **What view we describe:** 404 route for unknown paths.
- **How the view looks:** Minimal text-only message within the authenticated navy shell.
- **Typography:** 16px Snow text.
- No card, spacing system, or recovery navigation is provided.

**Implementation Notes**

- The current frontend implements authentication, Home, Review, Categories, Templates, Template Editing, Weekly Schedule, Schedule Overrides, and token transfer.
- Review is implemented as a weekly plan-versus-actual grid; separate Review Day, Analytics, Template Health, and Settings views are not routed or implemented in `front-web`.
- The template editor and Review timeline canvases use App Void backgrounds, but the grid lines in `DraggableColumn` are currently light gray `#e5e7eb`, creating a deliberate high-contrast exception to the otherwise Navy-Slate system.
- `Inter` and `JetBrains Mono` are declared as font families but are not imported by the frontend, so the browser uses fallback fonts unless they are installed locally.

---
