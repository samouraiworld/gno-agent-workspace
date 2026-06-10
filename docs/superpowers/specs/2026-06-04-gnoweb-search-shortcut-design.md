# gnoweb: `/`-to-focus shortcut for the omnibar search

Date: 2026-06-04
Status: Approved (design)
Follow-up to: gnolang/gno PR #5761 (omnibar search)

## Problem

PR #5761 turns the gnoweb search bar into an omnibar with keyboard navigation
once focused (↑/↓/⏎/Esc). There is no way to reach the bar from the keyboard:
the user must click it. GitHub and similar sites bind `/` to jump focus to
search. This adds the same affordance to gnoweb, plus a small visual hint so the
shortcut is discoverable.

## Scope

In scope:
- Global `/` keyboard shortcut that focuses the search input.
- A subtle, decorative `/` key hint in the bar, hidden while the bar is focused.

Out of scope:
- No Go/backend changes.
- No Esc-to-blur (Esc already closes the dropdown in #5761; not overloaded).
- No additional shortcut keys beyond `/`.

## Base

Branched from the #5761 head (`feat/gnoweb-search`, commit `a54f574a9`) in the
worktree `.worktrees/gno-search-shortcut` (branch `feat/gnoweb-search-shortcut`).
The submodule is never edited in place.

## Behavior

1. Pressing `/` anywhere on the page focuses the search input. The existing
   select-on-focus (`selectInput`) then selects the prefilled path so typing
   replaces it.
2. The shortcut is ignored when the user is already typing in an editable
   element: `<input>`, `<textarea>`, `<select>`, or any `contenteditable` node
   (this includes the search input itself, so a literal `/` typed there is not
   hijacked).
3. The shortcut is ignored when a modifier is held (Ctrl/Meta/Alt), so it never
   collides with browser/OS shortcuts.
4. The `/` hint is visible by default and hidden via CSS `:focus-within` the
   moment the bar gains focus (by click or by `/`); it reappears on blur. It
   shows regardless of the input's prefilled value, so it stays visible on realm
   pages where the bar holds the current path.

## Implementation

### 1. Controller — `frontend/js/controller-searchbar.ts`

- In `connect()`, register a document-level keydown listener using the existing
  `BaseController.on` helper:
  `this.on("keydown", this.focusShortcut.bind(this))`.
- New private method `focusShortcut(e: KeyboardEvent)`:
  - return early unless `e.key === "/"`;
  - return early if `e.ctrlKey || e.metaKey || e.altKey`;
  - return early if `SearchbarController.isEditable(document.activeElement)`;
  - else `e.preventDefault()` and focus the input
    (`(this.getDOMElement("input") as HTMLInputElement | null)?.focus()`).
- New static helper `isEditable(el: Element | null): boolean`:
  - false for null;
  - true if `el.isContentEditable`;
  - true if tagName is `INPUT`, `TEXTAREA`, or `SELECT`;
  - else false.

The handler is registered on `document` and the controller is a long-lived
header controller, so no teardown beyond the existing controller lifecycle is
needed.

### 2. Markup — `components/layouts/header.html`

Add a decorative hint inside the `.searchbar` form, after the input:

```html
<kbd class="searchbar-hint" aria-hidden="true">/</kbd>
```

`aria-hidden` keeps it out of the accessibility tree (the bar already has an
sr-only label and combobox semantics from #5761).

### 3. Styles — `frontend/css/06-blocks.css`

Add `.searchbar-hint`:
- positioned at the right edge of the input (the `.searchbar` form is the
  positioning context; add `position: relative` to it if not already present);
- small key-chip look: border, rounded corners, muted foreground, using #5761's
  existing design tokens; dark-mode aware via the same token approach;
- `pointer-events: none` so it never blocks clicks on the input;
- hidden when the form is focused: `.searchbar:focus-within .searchbar-hint { display: none; }`.

Tokens only, matching the PR's CSS conventions.

### 4. Built assets

Regenerate the committed build artifacts the way #5761 does:
- `public/js/controller-searchbar.js`
- `public/main.css`

Run the frontend build in the worktree (per `gno/AGENTS.md` / the gnoweb
frontend tooling) rather than hand-editing the built files.

## Testing

- If the PR's JS test harness (jsdom-based) supports unit-testing controllers,
  add a focused test for `focusShortcut`: `/` focuses the input; `/` is ignored
  when an editable element is active; `/` is ignored with a modifier held.
  #5761 ships no controller unit test, so if no harness exists, do not scaffold
  one for this follow-up.
- Manual verification: load a realm page, press `/` from page body (focuses +
  selects), confirm the hint hides on focus and returns on blur, confirm typing
  `/` inside the focused bar inserts a literal `/`.

## Risks / notes

- `URL.parse` and other #5761 APIs are untouched.
- The only global side effect is one document keydown listener; guarded so it is
  a no-op unless `/` is pressed outside an editable element with no modifier.
