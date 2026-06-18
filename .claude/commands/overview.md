---
description: Build the PR overview.html and open it in the browser
argument-hint: [pr-number]
---

Build the visual `overview.html` for PR `$ARGUMENTS` (when empty, the PR under review in the current context), following the "PR overview (`overview.html`)" section of `skills/review.md` exactly: written at the PR directory root `reviews/pr/<thousand>xxx/<number>-<slug>/overview.html`, self-contained (inline CSS/JS, zero external requests), light theme, generating-model name in the `<title>` and visible subtitle, explainer-only with zero review state.

Then run `./scripts/build-indexes.sh`.

Then open it in the default browser: `xdg-open reviews/pr/<thousand>xxx/<number>-<slug>/overview.html`.

Do not commit or push unless asked.
