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
- It is also used on the desktop wordmark "Homebase", the notification icon letter and the app icon (`web/public/icon.svg`, the source of the PNGs: a roof over three lines on `--bg`, the first crossed by a tilted highlighter stroke). Nowhere else.

## Tokens

### Colour

Every colour is a CSS variable on `:root`, redefined for dark under `@media (prefers-color-scheme: dark)` guarded as `:root:not([data-theme="light"])`, and again under `:root[data-theme="dark"]`. Components use the tokens only.

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `--bg` | `#F2F3EF` | `#121514` | Page background |
| `--field` | `#FAFAF8` | `#1A1E1D` | Inputs, raised rows, buttons on the page |
| `--ink` | `#15191A` | `#ECEFEA` | Text, primary buttons, checked states |
| `--muted` | `#5E655F` | `#9AA29C` | Secondary text, inactive nav, section labels |
| `--line` | `#D6D9D3` | `#2C3230` | Hairlines between rows. Decoration only (1.3:1 light, 1.4:1 dark) |
| `--border` | `#868D87` | `#6A736D` | Borders of controls: inputs, selects, chips, small buttons, dashed "add" and drop areas, the toggle's off track, key caps (3:1 or better) |
| `--soft` | `#E5E7E1` | `#232927` | Selected rail item, segmented track, icon boxes, banners |
| `--soft-day` | `#E0E3DC` | `#2A312E` | Selected day in the week strip |
| `--highlight` | `#F2E94E` | `#6E6616` | The highlighter mark only |
| `--highlight-text` | `#15191A` | `#ECEFEA` | Text on the mark |
| `--danger` | `#9A3B12` | `#F0A07A` | Only a field's error on Log in, Set up and the account dialogs (owner's request): the 2 px border and focus ring, the message and its warning icon. 6.3:1 on `--bg` |
| `--panel` | `#15191A` | `#1F2523` | The auth screens' desktop panel, dark in both themes; text on it is `--panel-text` `#F2F3EF` and `--panel-muted` `#C4C9C3` |
| `--veil` | ink at 30 % | black at 60 % | Behind sheets and dialogs |
| `--notif-bg` / `--notif-card` | `#1B1C1F` / `#2E3034` | the same | Notification preview (always dark) |

Text on `--bg` and `--field` must stay at 4.5:1 or better (3:1 for text 24 px and up). Measured: light `--muted` 5.4:1 on `--bg` and 4.8:1 on `--soft`; dark ink 15.8:1 on `--bg`, muted 7.0:1 on `--bg` and 5.7:1 on `--soft`, ink on the dark mark 5.1:1 (the band 3.1:1 against `--bg`). Verify after any change. Apart from a field's error in `--danger`, nothing is red: other failures use an icon, a bold title and plain words in ink.

### Dark theme

- It follows `prefers-color-scheme`, with System, Light and Dark under "Time and week" in Settings, for this device only. `public/theme.js` applies the choice before first paint (a blocking same-origin script, since the Content-Security-Policy forbids inline ones) and sets `<meta name="theme-color">` to `#F2F3EF` or `#121514`.
- The highlighter turns dim gold and the text stays light. The mark covers only the lower half of the title, so dark text on bright yellow would leave the top half of every letter on the dark page. Only `--ink` text sits on the mark.
- No pure black or white. Depth comes from the step between `--bg`, `--field` and `--soft`.
- Primary buttons invert: `--ink` fill, `--bg` text. Images and thumbnails are never tinted.
- The lifted drag row uses `0 8px 24px rgba(0,0,0,.5)` plus a 1 px `--line` ring.

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

- Three regions: left rail (240 px), main column, optional right column (300 to 380 px) separated by hairlines. The regions fill the window: the main column takes the space between rail and right column, and its content is capped (see Type) and centred in it; the right column sits at the window's right edge with a full-height hairline.
- Rail: wordmark, then Today, Inbox, Plan, Goals, Weekly review; at the bottom Settings, Reminders and a "Theme" button showing the current choice's icon, which opens a small menu above it with System, Light and Dark (a check marks the current one; the same choice as Settings, and both stay in step). The active item (or the open modal's item) has the `--soft` fill and a 600 label; hovering any item gives it the `--soft` fill. An item with a shortcut shows its key cap at the right (T, I, P, G, R, S).
- Below about 1100 px the right column drops under the main column. Below 900 px the rail becomes the phone tab bar.
- Forms ("New task", "Add a project") live in the right column as plain labelled fields, not modals.

## Components

- **Task panel (desktop):** closes, and `?task` leaves the URL, as soon as its task is marked done, whether from a list's checkbox or its own Status. A task that is already done still opens.
- **Task row:** checkbox, title (marked if planned for today), a meta line (project or goal, due text such as "Today", "Tomorrow", "Friday", "2d overdue", repeat note, attachment count). Tapping the row opens the task; tapping the checkbox toggles done.
- **Your three:** up to three rows, each with a rank number. An open task planned for today has a "Remove from today" button in its sheet or panel; done tasks keep their slot. Desktop adds a grip handle on the left; the row being dragged is lifted (field fill, shadow). Phone reorders by long-press drag, with up and down buttons in the task sheet. An empty slot is a dashed row, "Choose a third".
- **Goal line:** serif text with a one-line meta under it ("5 of 12 tasks done"). Goal detail adds a 4 px progress line (`--line` track, `--ink` fill).
- **Segmented control:** two or three buttons in a `--soft` track (Tasks and Projects; Keep, Pause and Drop in the weekly review, where the chosen option fills with ink).
- **Week strip:** seven equal cells showing a weekday letter, the date, and up to three small ink dots for how much is planned or due. The selected day uses `--soft-day`; other days take the `--soft` fill on hover (pointer devices only). Plan shows three-letter days ("Mon"); Today's narrower side column keeps the letter. Weeks start on Monday by default. Approved: previous and next arrows either side of the range ("5 – 11 October"); away from this week the range is a text button back to it. Tapping a day on Plan filters the list (`/plan?day=…`) under the day's name with "Show all"; on Today's side column it opens Plan on that day.
- **Filter chips:** round, 44 px tall; the selected chip is filled with ink and light text.
- **Attachment row:** a 44 px icon box (note, link, file, image), a title, a one-line meta ("PDF, 240 KB"), and a remove button. Beneath the list: "Add link", "Add note", "Upload file". Desktop adds a dashed drop area.
- **Toggle:** 52 by 32 px; ink track when on.
- **Buttons:** primary is ink fill with light text, 52 px tall, radius 8 (small buttons too); secondary is a 1 px ink outline; text buttons are 600 weight with an underline offset of 3 px. No arrows appended to labels.
- **Inputs:** `--field` fill, 1 px `--border` border, radius 8, 48 px tall, always with a visible or screen-reader label. Focus turns the border ink and adds a 3 px ring of ink at 20% (no outline). A field in error gets a 2 px `--danger` border (and a `--danger` ring on focus) and its message under it in `--danger` after a warning icon, shown only after the person leaves the field or presses the button; what they typed is never cleared.
- **Passphrase fields:** every one has the eye toggle, a 44 px icon button inside the right end (Lucide `eye` while hidden, `eye-off` while shown; `aria-label` "Show passphrase" or "Hide passphrase" with `aria-pressed`). Fields start hidden each time a screen opens, the choice is never stored, and each toggles on its own. Toggling keeps the caret and the value. Autocapitalise, autocorrect and spellcheck are off; autofill is `username`, `current-password` on Log in and `new-password` for new passphrases.
- **Dialogs:** a bottom sheet on the phone, a small centred dialog (radius 20) on desktop, with the title, an optional muted sentence, then the form. Desktop puts the primary button right and a text "Cancel" before it; the phone stacks them.
- **Toast:** approved. An ink bar with light text, an 18 px icon, one line (wrapping when a message needs it), radius 12, at most five stacked, 4 s (6 s with an action). Phone: above the tab bar, or above the capture bar where there is one. Desktop: bottom left of the main column. It confirms what leaves the screen (captured, planned, made a project, uploaded, deleted) and says why something was refused (day full, upload failed). Removing from today and every delete offer Undo; making a project offers Open. Never red.
- **Notification preview:** dark card, a rounded square with a bold "H" carrying the highlighter, app name and time, a 600 title and a body.

## Screens

| Screen | Route | Contents |
| --- | --- | --- |
| Today | `/` | Date and "Today"; Goals (all open goals as serif lines); Your three; capture bar. Desktop right column: goals with progress, this week strip, review link. Review link appears from Friday until done. |
| Choose up to three | A modal over the current screen (`?pick=1`; `/pick` redirects): a bottom sheet on the phone, a centred dialog on desktop; desktop also keeps the inline suggestions. Reached from the dashed slot and from a "Change" text link beside "Your three", which shows even when the day is full. Unticking removes a task from today. | Groups: Due soon, In progress, Moved a few times. Tick tasks; button "Set today (2 chosen)". |
| Inbox | `/inbox` | "Captured, not yet placed". Approved: each item has the done checkbox before its title (ticking it says "Done" in a toast with Undo, since a done task leaves the Inbox) and chips with an icon and a verb, "Do today", "Pick a day", "Add to a goal" and "Make it a project", wrapping to two rows on the phone. Desktop shows today's three alongside. |
| Plan, Tasks | `/plan` | Tasks and Projects switch; week strip; chips Everything, Due this week, Standalone, Repeating; tasks grouped by day, then Repeating. Desktop adds a "New task" form (task, goal, plan for, repeat); the phone has a plus button beside the title ("Add task") that opens the same form in a sheet. |
| Plan, Projects | `/plan/projects` | Projects grouped under their goal with a progress line and the next task. Approved: a flag before each goal heading, a folder before each project, and a meta line of icons with counts (tasks done, attachments, "Notes" when it has notes). Desktop adds "Add a project". On the phone it is the desktop list in one column. |
| Project | `/projects/:id` | Title, progress, Tasks with an add field, Notes (editable), Attachments. |
| Goal | `/goals/:id` | Serif title, progress, Projects, Tasks on their own, Notes, Links. Desktop shows the goal list on the left. |
| Goals | `/goals` | Goal lines, then the approved "New goal" field under the list: a labelled field in the serif (placeholder "A statement of intent") and an ink "Add" button. |
| Weekly review | `/review` | One-sentence summary in serif; "Gone quiet" with Keep, Pause, Drop; "Next week" shows a goal with nothing done and "Plan a task"; button "Finish review". Built as: Keep is chosen by default; "Plan a task" opens the goal. The link on Today (from Friday until done) is a bordered row with the review icon, in the right column on desktop and under Your three on the phone; the rail lists "Weekly review" after Goals. |
| Settings | `?settings=1` over any screen (`/settings` redirects) | Approved: a modal like Choose your three (a sheet on the phone, a 760 px dialog on desktop), opened from the rail, the S key or the link on Today. A tab for each part, kept in the URL (`?settings=time`, `calendar`, `backups`, `account`, `sessions`; `1` opens the first; switching tabs replaces the history entry): a 176 px column of tabs on the left on desktop (the rail's item style, `--soft` fill on the active one) with a "Keyboard shortcuts" text button at its foot, a scrolling row above the content on the phone. "Time and week" (zone select, Monday or Sunday, and Theme: System, Light or Dark on this device), "Calendar" (what the link is for, the link with Copy, "Make a new link"), "Backups" (a row "Back up every day" (or "every week", "every 3 days", "every 6 hours", from the server's interval) with a muted line and the toggle; under a hairline "Folder on the server" with the path in monospace and "Last backup" with the date and time, or "None yet"; then a muted line that a copy on the same disk does not survive losing it), "Account" (Username with the current name and "Change"; Passphrase with "Changed 12 September" and "Change"), "Sessions" (each browser with when it was last used and "Log out" for others, then "Log out of this device"). Rail: above Reminders at the bottom. Phone: a link beside Reminders at the foot of Today. "Change" swaps the modal's content for that form in place (`?settings=username` or `?settings=passphrase`, 560 px): a back button before the title ("Back to Settings"), the title and sentence, the fields; Cancel, Save and Back return to the Account tab, so no dialog ever opens over another. "Keyboard shortcuts" closes Settings first. |
| Reminders | `?reminders=1` over any screen (`/reminders` redirects) | A modal like Settings. Three slots (time, on or off, kind, description); devices; "Send a test now". Built as: one row per slot with the time as a 34 px tabular time input (a complete time saves at once), the kind name and a one-line description, and the 52 by 32 toggle; a line under the title says which zone the times use. "This device" shows its state (not supported, blocked, off with "Get reminders on this device", on with "Send a test now" and "Turn off on this device"). "Other devices" lists the rest with when each was last reached and a Remove button. Desktop reaches it from the bottom of the rail; the phone from a "Reminders" link at the foot of Today, since the tab bar has four places. |
| Set up | `/setup` | Only until the account exists; every other route leads here on a fresh install. The same layout as Log in with the statement "One account, just / for you.": 28 px "Set up Homebase", Username (placeholder "Letters, numbers, dots and dashes"), Passphrase (hint "At least 12 characters. Several random words work well."), Confirm passphrase, and a full-width primary "Create account". It logs in and opens Today. |
| Log in | `/login` | Approved designs. Phone, "statement and thumb-zone form": the highlighted wordmark at the top, a 36 px serif statement "Three things, / chosen on purpose." centred in the space left, and the form at the bottom in reach of the thumb. Desktop (900 px and up), "ink panel": the left 46% is `--panel` with the wordmark as a solid highlight block (28 px, the light theme's yellow `--panel-mark` with `--panel` text in both themes) at the top and the 46 px serif statement with "A calm place for goals, projects and tasks." at the bottom, and between them, centred in the space, a 320 px line drawing in `--panel-muted` (a house on a horizon, the sun a `--panel-mark` disc; decorative, hidden below 700 px of height); the form is 380 px wide, centred in the rest. The form: 28 px "Log in", Username and Passphrase, a full-width primary "Log in" (it reads "Logging in" and is disabled while waiting). A wrong pair shows "That username or passphrase did not match." under the passphrase, never saying which; after five failures, "Too many attempts. Try again in 10 minutes." When the session ended rather than a logout, a `--soft` notice sits above the fields ("Your session ended", "Log in again. Changes made offline are kept and will sync.") and the username is filled in. |
| Change username | sheet or dialog from Settings | "You will use the new username to log in on every device." New username, Your passphrase, primary "Save username", text "Cancel". |
| Change passphrase | sheet or dialog from Settings | "Other devices will be logged out. This one stays logged in." Current passphrase, New passphrase (with the hint), Confirm new passphrase, primary "Save passphrase", text "Cancel". |

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

## States

- **Empty:** says what the thing is and offers one action. No illustrations (the drawing on the Log in and Set up panel is the only one).
- **Loading:** the local copy shows at once. Skeleton bars in `--soft` appear only until the first sync after logging in, with `aria-busy="true"`; a slow pulse, off with reduced motion. No spinners.
- **Error:** an icon, a bold title and one or two plain sentences, in ink. With data on screen, the top banner (see below); with nothing to show, the inline block with "Try again".

| Screen | Empty | Loading | Error |
| --- | --- | --- | --- |
| Today | Goals: "No goals yet." in the serif, "A goal is the reason behind your tasks. Start with one or two." and "Add a goal". Your three: "Pick up to three things that would make today a good day." over the "Choose your first" slot | goal lines and task rows | "Could not load your tasks", "Check your connection, then try again. Nothing has been lost.", "Try again" |
| Inbox | "Inbox is clear." "Anything you capture lands here until you place it." | task rows | as Today |
| Plan, tasks | phone "Nothing planned." "Use the plus to add a task."; desktop "No tasks here." "Type one in the New task form, or capture it from Today." | task rows | as Today |
| Plan, projects | "No projects yet." "A project groups tasks toward a goal, like a trip or a course." with "Add a project", which focuses the add form | lines | "Could not load your projects" |
| Goals | "No goals yet." in the serif, then the New goal field | lines | "Could not load your goals" |
| Project | "No tasks yet. Add the first one.", "No notes yet." with "Add notes", "Links, notes and files for this item." | title, progress and rows | "This project no longer exists", "It may have been deleted on another device.", "Back to Plan" |
| Goal | "No projects or tasks yet." plus the add fields | as Project | "This goal no longer exists" with "Back to Goals" |
| Weekly review | "Nothing has gone quiet this week." | lines | "Could not load the review" with "Try again" |
| Reminders | the device states | lines | "Could not save that change." under the slot, which goes back to its saved value |
| Settings | not applicable | lines for the calendar link and sessions | "Could not save. Try again." under the control |

Uploads that fail stay under the attachment buttons, with the file name in the bold title ("Course map.png did not upload") and "That file is over 25 MB. Choose a smaller one." or "The file did not upload. Try again."

## Offline and sync indicator

- A banner across the top of the main column, above the page header: `--soft` fill, an icon, a bold lead and a muted sentence (on one line on desktop, the lead on its own line on the phone). It pushes content down and never covers it, and is announced politely.
- "Offline. Changes are saved on this phone (computer) and will sync when you reconnect. 2 waiting." and "Could not sync. Your changes are saved here. Trying again in 30 seconds." with a text "Retry".
- Nothing while synced. "Syncing" only once a sync has taken a second, and in the banner only while recovering from trouble; then "Back online. Synced." for 3 seconds.
- The desktop rail repeats it in one quiet line under Reminders: "Offline, 2 waiting", "Could not sync", "Syncing".
- Before the first sync has finished there is nothing on screen to keep, so the screen's error block speaks instead of the banner.

## Keyboard shortcuts (desktop)

| Key | Action |
| --- | --- |
| C | Focus the capture field (the New task field on Plan; elsewhere it opens Today) |
| T, I, P, G | Go to Today, Inbox, Plan, Goals |
| R | Weekly review |
| S | Settings |
| ? | Show the list |
| Esc | Close a dialog, sheet or the task panel |

- Single keys act only when focus is not in an input, textarea, select or editable element, no Ctrl, Alt or Meta key is held, and no dialog is open.
- Keys also show where they act: a small key cap (20 px, `--border` outline, muted letter) at the right of each rail item, and a C inside the empty capture field. Desktop only; hidden from screen readers, which get `aria-keyshortcuts` on the control instead.
- The list is a centred dialog (radius 20, 420 px wide), each key in a small bordered key cap. It traps focus and returns it on close. Settings has a "Keyboard shortcuts" text button so it can be found without knowing `?`.

The repeat picker is approved: it grows in place under "Repeat" in the task form. Without a rule, "Does not repeat" and a "Make it repeat" button (every week, from when it is done). With one: a number and a unit, "Counts from" (When it's done, The due date), weekday circles when weekly from the due date, "Ends" (Never, On a date), a summary sentence in the serif, and "Stop repeating". It saves as it changes.

Approved since: the login screen and goal creation (see Screens); the phone Projects list is the desktop list in one column. Deleting a goal or project asks first: a bottom sheet on the phone, a small centred dialog (radius 20) on desktop, with the title "Delete “Name”?", one sentence of counts ("It has 4 tasks and 2 attachments."), the primary "Keep tasks, delete project", the secondary "Delete project and tasks", and a text "Cancel". With nothing inside, one primary "Delete project".
