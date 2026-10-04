# DESIGN.md

The visual and interaction rules for Homebase. The look was drawn as clickable mockups (phone 390 px wide, desktop 1280 px wide); everything needed to rebuild it is written here, so the mockups are a visual check, not the source of truth.

Reference screens are in `mockups/` as standalone HTML (open `mockups/index.html`). Layout, colour and type there are exact; controls are not interactive. The original canvas, if the owner shares it: https://claude.ai/artifact/DfpZ27WAAN2P7LdNFG12fC

## Character

Quiet, typographic, and a little stern about focus. Hairlines and whitespace do the structuring, not boxes. Colour is almost absent, so the one yellow mark means something. It must not look like a generated app: no gradients, no card grids, no emoji, no left-border cards, no tinted shadows, no all-caps labels.

## The one signature: the highlighter

A task that is planned for today is marked like a line in a book: a yellow band behind the lower half of its title.

```css
.mark { box-shadow: inset 0 -0.5em 0 var(--highlight); }
```

Rules:

- Only today's picks get it (Your three, and the same tasks wherever they appear). Not overdue, not suggested, not later days.
- A done task loses the mark and is struck through in the muted colour.
- It is also used on the desktop wordmark "Homebase" and the notification icon letter. Nowhere else.

## Tokens

### Colour (light theme, drawn)

| Token | Value | Use |
| --- | --- | --- |
| `--bg` | `#F2F3EF` | Page background |
| `--field` | `#FAFAF8` | Inputs, raised rows, buttons on the page |
| `--ink` | `#15191A` | Text, primary buttons, checked states |
| `--muted` | `#646B66` | Secondary text, inactive nav, section labels |
| `--line` | `#D6D9D3` | Hairlines between rows, input borders |
| `--soft` | `#E5E7E1` | Selected rail item, segmented track, icon boxes |
| `--soft-day` | `#E0E3DC` | Selected day in the week strip |
| `--dashed` | `#AEB3AC` | Dashed "add" and drop areas |
| `--highlight` | `#F2E94E` | The highlighter mark only |
| `--notif-bg` / `--notif-card` | `#1B1C1F` / `#2E3034` | Notification preview (always dark) |

Text on `--bg` and `--field` must stay at 4.5:1 or better (3:1 for text 24 px and up). Verify `--muted` after any change.

### Colour (dark theme, proposed, not drawn)

Not designed yet; treat as a starting point and show the user before shipping.

| Token | Proposed |
| --- | --- |
| `--bg` | `#121514` |
| `--field` | `#1A1E1D` |
| `--ink` | `#ECEFEA` |
| `--muted` | `#9AA29C` |
| `--line` | `#2C3230` |
| `--soft` | `#232927` |
| `--highlight` | `#F2E94E`, with the marked text switched to `#15191A` so it stays readable |

### Type

Two families, self-hosted through `@fontsource`.

- **Schibsted Grotesk** 400, 500, 600, 700: all interface text.
- **Source Serif 4** 400, 500: goals only. Goals read as statements of intent, in the serif, everywhere they appear.

| Role | Size and weight | Notes |
| --- | --- | --- |
| Screen title | 36/1.1, 700 (desktop 46) | letter-spacing -0.02em |
| Sheet or project title | 30 to 32, 700 | |
| Goal title (detail) | Serif 32 (desktop 44), 500 | |
| Section heading ("Your three") | 22 to 24, 700 | |
| Section label ("Goals", "Tasks") | 14, 600, `--muted` | sentence case, no all-caps |
| Row title | 17 (desktop 18), 500 | line-height 1.3 |
| Goal line in lists | Serif 20 (desktop 21), 400 | line-height 1.25 |
| Body, notes | 16/1.5 to 1.55, 400 | |
| Meta under a row | 13, `--muted` | |
| Chips and small buttons | 14, 600 | |
| Reminder time | 34 to 36, 600 | tabular numerals |
| Tab bar label | 12, 500 (active 600) | |

Line length stays under 80 characters; the main column is capped at 720 to 780 px (desktop).

### Space and shape

- Page padding: 24 px on the phone (20 px top); desktop main column 48 px top and 56 px sides.
- Rows: at least 64 px tall, 10 to 12 px vertical padding, 1 px `--line` rule above each row, no boxes around rows.
- Radii: inputs and large buttons 12; chips 22 (fully round); small buttons and icon boxes 10; segmented control 12 (track) and 9 (active segment); the lifted drag row 12.
- Touch targets: 44 px minimum, including the checkbox (a 26 px circle inside a 44 px button), grip handle, close buttons and nav items.
- Shadow: only on the row being dragged (`0 8px 24px rgba(20,25,24,.16)`) and the active segment (`0 1px 2px rgba(20,25,24,.14)`).
- Icons: Lucide (`lucide-react`), 24 px (20 in rails and buttons, 18 in chips and toasts, 16 in meta lines), 1.8 px stroke, round caps, inline SVG. No icon font, no emoji.

## Layout

### Phone (below 900 px wide)

- Single column. Bottom tab bar, 72 px tall, four items: **Today, Inbox, Plan, Goals**. The active item has a 2 px ink line along its top edge and a 600 label. Inbox shows a small ink count badge.
- Pages with a capture bar (Today, Inbox) place it directly above the tab bar: a text field and a square ink "Add" button.
- Detail screens use a text "back" link at the top left (for example "Run a half marathon in spring") instead of a tab.
- Pickers and edit forms open as a bottom sheet: grab bar, 20 px top radius, slide up 180 ms (none with reduced motion).

### Desktop (900 px and wider)

- Three regions: left rail (200 to 240 px), main column, optional right column (300 to 380 px) separated by hairlines. The regions fill the window: the main column takes the space between rail and right column, and its content is capped (see Type) and centred in it; the right column sits at the window's right edge with a full-height hairline.
- Rail: wordmark, then Today, Inbox, Plan, Goals, Weekly review; Reminders at the bottom. The active item has the `--soft` fill and a 600 label.
- Below about 1100 px the right column drops under the main column. Below 900 px the rail becomes the phone tab bar.
- Forms ("New task", "Add a project") live in the right column as plain labelled fields, not modals.

## Components

- **Task panel (desktop):** closes, and `?task` leaves the URL, as soon as its task is marked done, whether from a list's checkbox or its own Status. A task that is already done still opens.
- **Task row:** checkbox, title (marked if planned for today), a meta line (project or goal, due text such as "Today", "Tomorrow", "Friday", "2d overdue", repeat note, attachment count). Tapping the row opens the task; tapping the checkbox toggles done.
- **Your three:** up to three rows, each with a rank number. An open task planned for today has a "Remove from today" button in its sheet or panel; done tasks keep their slot. Desktop adds a grip handle on the left; the row being dragged is lifted (field fill, shadow). Phone reorders by long-press drag, with up and down buttons in the task sheet. An empty slot is a dashed row, "Choose a third".
- **Goal line:** serif text with a one-line meta under it ("5 of 12 tasks done"). Goal detail adds a 4 px progress line (`--line` track, `--ink` fill).
- **Segmented control:** two or three buttons in a `--soft` track (Tasks and Projects; Keep, Pause and Drop in the weekly review, where the chosen option fills with ink).
- **Week strip:** seven equal cells showing a weekday letter, the date, and up to three small ink dots for how much is planned or due. The selected day uses `--soft-day`. Weeks start on Monday by default. Approved: previous and next arrows either side of the range ("5 – 11 October"); away from this week the range is a text button back to it. Tapping a day on Plan filters the list (`/plan?day=…`) under the day's name with "Show all"; on Today's side column it opens Plan on that day.
- **Filter chips:** round, 44 px tall; the selected chip is filled with ink and light text.
- **Attachment row:** a 44 px icon box (note, link, file, image), a title, a one-line meta ("PDF, 240 KB"), and a remove button. Beneath the list: "Add link", "Add note", "Upload file". Desktop adds a dashed drop area.
- **Toggle:** 52 by 32 px; ink track when on.
- **Buttons:** primary is ink fill with light text, 52 px tall, radius 12; secondary is a 1 px ink outline; text buttons are 600 weight with an underline offset of 3 px. No arrows appended to labels.
- **Inputs:** `--field` fill, 1 px `--line` border, radius 12, 48 px tall, always with a visible or screen-reader label. Focus shows a 2 px ink outline with 2 px offset.
- **Toast:** approved. An ink bar with light text, an 18 px icon, one line, radius 12, at most five stacked, 4 s (6 s with an action). Phone: above the tab bar, or above the capture bar where there is one. Desktop: bottom left of the main column. It confirms what leaves the screen (captured, planned, made a project, uploaded, deleted) and says why something was refused (day full, upload failed). Removing from today and every delete offer Undo; making a project offers Open. Never red.
- **Notification preview:** dark card, a rounded square with a bold "H" carrying the highlighter, app name and time, a 600 title and a body.

## Screens

| Screen | Route | Contents |
| --- | --- | --- |
| Today | `/` | Date and "Today"; Goals (all open goals as serif lines); Your three; capture bar. Desktop right column: goals with progress, this week strip, review link. Review link appears from Friday until done. |
| Choose up to three | A modal over the current screen (`?pick=1`; `/pick` redirects): a bottom sheet on the phone, a centred dialog on desktop; desktop also keeps the inline suggestions. Reached from the dashed slot and from a "Change" text link beside "Your three", which shows even when the day is full. Unticking removes a task from today. | Groups: Due soon, In progress, Moved a few times. Tick tasks; button "Set today (2 chosen)". |
| Inbox | `/inbox` | "Captured, not yet placed". Approved: each item has the done checkbox before its title (ticking it says "Done" in a toast with Undo, since a done task leaves the Inbox) and chips with an icon and a verb, "Do today", "Pick a day", "Add to a goal" and "Make it a project", wrapping to two rows on the phone. Desktop shows today's three alongside. |
| Plan, Tasks | `/plan` | Tasks and Projects switch; week strip; chips Everything, Due this week, Standalone, Repeating; tasks grouped by day, then Repeating. Desktop adds a "New task" form (task, goal, plan for, repeat). |
| Plan, Projects | `/plan/projects` | Projects grouped under their goal with a progress line and the next task. Approved: a flag before each goal heading, a folder before each project, and a meta line of icons with counts (tasks done, attachments, "Notes" when it has notes). Desktop adds "Add a project". On the phone it is the desktop list in one column. |
| Project | `/projects/:id` | Title, progress, Tasks with an add field, Notes (editable), Attachments. |
| Goal | `/goals/:id` | Serif title, progress, Projects, Tasks on their own, Notes, Links. Desktop shows the goal list on the left. |
| Goals | `/goals` | Goal lines, then the approved "New goal" field under the list: a labelled field in the serif (placeholder "A statement of intent") and an ink "Add" button. |
| Weekly review | `/review` | One-sentence summary in serif; "Gone quiet" with Keep, Pause, Drop; "Next week" shows a goal with nothing done and "Plan a task"; button "Finish review". |
| Reminders | `/reminders` | Three slots (time, on or off, kind, description); devices; "Send a test now". |
| Login | `/login` | Approved: the same top-aligned column as other screens. Highlighted wordmark, 36 px "Log in", one muted sentence, a labelled passphrase field, a full-width primary "Log in". A wrong passphrase shows a plain ink sentence under the field. |

## Copy

- Sentence case. Plain verbs. No exclamation marks, no "Oops", no filler.
- Buttons name the result: "Add task", "Set today (2 chosen)", "Finish review", "Send a test now".
- Empty states say what to do: "No tasks here. Type one above."
- Real examples are used throughout the mockups (half marathon, budgeting app, learn Go, household chores); replace with neutral examples in tests, never with lorem ipsum.
- Reminder text is written from current data (see `docs/SPEC.md` section 6).

## Motion

None, except: the bottom sheet slide (180 ms), the lift of a dragged row, and focus outlines. All motion is disabled when `prefers-reduced-motion` is set.

## Accessibility

- Real `<button>`, `<a>`, `<input>` with `<label>`. No clickable `div`s.
- Icon-only buttons carry `aria-label` ("Mark done", "Drag to reorder", "Remove attachment").
- Colours that must be told apart also differ in lightness. Focus is always visible.
- Drag and drop always has a keyboard alternative (up and down buttons).

## Not designed yet

Dark theme, empty, loading and error states, the offline indicator and the repeat picker. Design these in this style and show the owner before building.

Approved since: the login screen and goal creation (see Screens); the phone Projects list is the desktop list in one column. Deleting a goal or project asks first: a bottom sheet on the phone, a small centred dialog (radius 20) on desktop, with the title "Delete “Name”?", one sentence of counts ("It has 4 tasks and 2 attachments."), the primary "Keep tasks, delete project", the secondary "Delete project and tasks", and a text "Cancel". With nothing inside, one primary "Delete project".
