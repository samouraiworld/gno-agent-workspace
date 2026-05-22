# PR #5554: feat(gnoweb): add SimpleAnalytics metadata and custom events

**URL:** https://github.com/gnolang/gno/pull/5554
**Author:** gfanton | **Base:** master | **Files:** 33 | **+781 -95**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR adds the full v1 SimpleAnalytics event surface to gnoweb. It covers two layers:

1. **Pageview metadata** (`page_type`, `chain_id`) set via `window.sa_metadata` from a synchronous classic `sa-bootstrap.js` (CSP-safe, no inline scripts). `page_type` is classified server-side by `ClassifyPageType(mode, view)` into 11 categories (home, user, pure, realm, source, help, directory, status, redirect, explorer, other).

2. **Custom events** (18 event names via 14 listeners in `analytics.ts`): copy_action, submit_action, search_action, network_popup_toggle, breadcrumb_click, back_navigation, toc_toggle, mode_change, send_mode_toggle, qeval_preview, address_filled, params_filled, theme_toggle, devmode_toggle, list_filter_search, list_sort_change, list_display_change, scroll_depth, and 7 named outbound_<target> events.

Key architectural decisions:
- `sa-bootstrap.js` is loaded as a synchronous classic `<script src=...>` (IIFE, not ES module) to guarantee `window.sa_metadata` is set before SA's async `latest.js` loads. This avoids CSP `unsafe-inline` issues.
- `analytics.ts` is loaded as an ES module and attaches all event delegation listeners at the document level. Capture phase is used for copy/submit to fire before controllers call `stopPropagation`.
- Qeval placeholder text and error class are shared between the controller and analytics via `data-qeval-*` attributes on the DOM element (single source of truth, no duplicated string literals).
- `data-outbound` is set in Go on `FooterLink`/`HeaderLink` structs and rendered by templates. The outbound enum is defined as Go constants (`OutboundDocs`, etc.) and must stay in sync with analytics.ts and SIMPLEANALYTICS.md.
- `AppConfig.AnalyticsHostname` / `--web-analytics-hostname` lets non-default-port deployments report under the bare hostname.
- `SIMPLEANALYTICS.md` documents the full taxonomy, privacy posture, cardinality caps, and the procedure for adding new events.

Prior review by alexiscolin addressed: splitting `copy_action.kind` into granular categories, adding `pkgpath` to `submit_action`, renaming `ViewModePackage` page_type to `pure`, CSP-safe bootstrap, and removing dead `HeadData.Analytics` bool.

## Test Results

- **Existing tests:** PASS — `TestClassifyPageType` (11 cases), `TestEnrichFooterData_Outbound`, `TestStaticHeaderGeneralLinks_Outbound`, `TestAnalytics` (enabled/disabled/page_type), `TestRedirectView`, `TestEnrichFooterData`.
- **Edge-case tests:** skipped (frontend JS event logic not unit-testable from Go; would need Playwright/Cypress).
- **CI:** All checks pass (build, lint, test, gnoweb_front_lint, gnoweb_generate). Merge-requirements check fails because codeowner (alexiscolin) has not yet approved.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `analytics.ts:226-231` — `scroll_depth` remaps `page_type === "help"` to `surface === "action"`, but this is a client-side-only mapping with no corresponding server-side documentation or test. The SIMPLEANALYTICS.md table says `surface: source | action`, but `ClassifyPageType` returns `"help"` for the help view. A future change to the help-view classification or surface name would silently break the analytics contract. Consider either (a) having `ClassifyPageType` return `"action"` for `HelpViewType` and documenting the mapping server-side, or (b) making the remap in `sa-bootstrap.ts` so it's part of `window.sa_metadata` and thus auditable from a single location.

- [ ] `analytics.ts:255-258` — The `outbound_<target>` event name is constructed by concatenating `data-outbound` with `outbound_`, but `data-outbound` is an arbitrary server-rendered string. Although Go constants limit the values today, a template bug or future Go change could emit an unbounded string as `data-outbound`, creating cardinality explosion in the SA dashboard. Consider adding a guard in `analytics.ts` that validates the target against the known enum set (`docs|faucet|status|github|twitter|discord|youtube`) before firing, or at minimum validates that the string matches `^[a-z_]+$` and is <= 20 chars.

- [ ] `analytics.ts:38-43` — `copy_action` kind detection uses `data-copy-remote-value` prefixes, but if both `data-copy-text-value` and a `remote` starting with `func-` or `action-function-` are present, the `data-copy-text-value` branch wins unconditionally. This seems intentional (link copy is link copy regardless of other attrs), but it's not documented. A comment explaining the priority would help future maintainers.

## Nits

- [ ] `analytics.go:60` — `ClassifyPageType` returns `"other"` for the zero-value `ViewMode(0)` with any view not in the switch. This is fine, but the test at `analytics_test.go:22` uses `ViewType("unknown")` which only tests the view fallback, not the zero-mode fallback. A test case for zero `ViewMode` with a recognized view type (e.g. `ViewMode(0)` + `RealmViewType`) would be more thorough.

- [ ] `footer.html:54` / `redirect.html:13` — Files end without a trailing newline.

- [ ] `layout_index.go:184-188` — Analytics fields are populated after `EnrichFooterData` and `EnrichHeaderData`, which means `EnrichFooterData` cannot reference `Analytics.PageType` during enrichment. This is correct (EnrichFooterData doesn't need it), but the ordering dependency is implicit and fragile. A brief comment noting "Analytics fields must be set after enrichment" would help.

- [ ] `app_test.go:236-266` — The `page_type` test creates a new router per test group but runs multiple routes against it. This is fine, but it means all subtests share the same integration node, so a failure in an earlier subtest doesn't isolate the problem. The `t.Run` closure captures `route` and `pageType` correctly (Go 1.22+ loop variable semantics).

## Missing Tests

- [ ] No test for `AnalyticsHostname` rendering — the `data-hostname` attribute on the SA script tag is untested. An integration test verifying `data-hostname` appears when `cfg.AnalyticsHostname` is set would close this gap. (`app_test.go`)
- [ ] No test for `ClassifyPageType` with the zero-value `ViewMode` (0) — all test cases use named modes. (`analytics_test.go`)
- [ ] No test for the `RedirectAnalytics()` helper — `StaticMetadata.RedirectAnalytics()` constructs `AnalyticsData` for redirect views, but no unit test verifies its fields are correctly populated (especially `Hostname`). (`handler_http.go:42-51`)
- [ ] No frontend test coverage — the 14 event listeners in `analytics.ts` have no automated tests. This is acceptable for an initial PR but should be tracked for a follow-up (Playwright/Cypress).

## Suggestions

- The `scroll_depth` surface remap ("help" → "action") in `analytics.ts:226-231` would be cleaner if `ClassifyPageType` returned `"action"` for `HelpViewType` directly, with a second metadata key like `surface_view: "help"` for disambiguation. This would make the SA metadata self-describing rather than requiring client-side post-processing. However, this changes the documented `page_type` enum from 11 to 12 values, so it's a design decision for the author.

- Consider extracting the outbound target validation into a shared constant (e.g., a `VALID_OUTBOUND_TARGETS` set in `analytics.ts`) that both the event listener and a future unit test can reference, ensuring the TS-side guard stays in sync with the Go-side constants.

## Questions for Author

- The PR body mentions `page_type` cardinality of 11 (including `other`), but `ClassifyPageType` can also return `"other"` for unrecognized combinations. Is the `"other"` category expected to appear in production, or is it purely a fallback? If the latter, should it be monitored/alerted on as a potential bug?
- `SIMPLEANALYTICS.md` lists `page_type` as `redirect` for the redirect view, but the redirect page is a minimal HTML page that only loads SA scripts (no `analytics.js` module). Does the redirect page actually fire custom events, or is only the pageview metadata captured?
- The `scroll_depth` listener uses `{ passive: true }` which is correct for read-only scroll, but it means `preventDefault()` cannot be called. This is fine — just confirming this was intentional.

## Verdict

REQUEST CHANGES — The `scroll_depth` surface remap ("help"→"action") lives only in client-side JS with no server-side audit trail or test, creating a fragile analytics contract. The `outbound_<target>` event name construction has no client-side cardinality guard. Both should be addressed before merge. Everything else is clean and well-structured: the CSP-safe bootstrap approach is sound, the privacy posture is rigorous (no raw user input ever forwarded), cardinality caps are documented, and the Go/TS test coverage for server-side logic is solid.
