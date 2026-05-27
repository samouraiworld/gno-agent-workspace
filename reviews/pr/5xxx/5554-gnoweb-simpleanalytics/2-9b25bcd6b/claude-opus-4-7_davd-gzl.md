# PR #5554: feat(gnoweb): add SimpleAnalytics metadata and custom events

URL: https://github.com/gnolang/gno/pull/5554
Author: gfanton | Base: master | Files: 33 | +791 -96
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5554 9b25bcd6b` (then `gh -R gnolang/gno pr checkout 5554` inside it)

**Verdict: APPROVE** — privacy posture tightened since round 1; remaining concerns (scroll_depth surface remap, JS-side outbound cardinality guard) are non-blocking polish.

## Summary

Round 2 review of the SimpleAnalytics taxonomy. Since round 1 ([1-65143109d](../1-65143109d/glm-5.1_davd-gzl.md), commit `65143109d`) only two PR-relevant changes landed; everything else in `git log 65143109d..HEAD` is master merging in. The PR is otherwise mature: CSP-safe bootstrap, 11-value `page_type` enum, 18 named events through 14 listeners, hard cardinality caps on `func` / `pkgpath`, and a `SIMPLEANALYTICS.md` that doubles as the public taxonomy contract.

## What changed since round 1

- [`SIMPLEANALYTICS.md:13`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/SIMPLEANALYTICS.md#L13) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/SIMPLEANALYTICS.md#L13) — privacy bullet narrowed: "never collect form input values" now reads "as event payload" and explicitly calls out the help-page URL pattern (`…$help&func=…&arg1=…`) which SA captures as part of the standard pageview URL. Closes alexis's NIT on overclaim.
- [`SIMPLEANALYTICS.md:18`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/SIMPLEANALYTICS.md#L18) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/SIMPLEANALYTICS.md#L18) — DNT posture documented inline: SA respects `Navigator.doNotTrack` by default.
- [`SIMPLEANALYTICS.md:125-130`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/SIMPLEANALYTICS.md#L125-L130) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/SIMPLEANALYTICS.md#L125-L130) — new "Third-party posture" subsection consolidating SA defaults (DNT, SHA-256 hashed/daily-rotated IPs, no cookies, no fingerprinting, `auto-events.js` outbound URL capture, `no-referrer` noscript pixel).
- [`layouts/analytics.html:8`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/layouts/analytics.html#L8) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/layouts/analytics.html#L8) — `<noscript>` pixel `referrerpolicy` tightened from `no-referrer-when-downgrade` to `no-referrer`. JS-disabled visitors no longer leak the page URL via the cross-origin Referer header.

All three tighten the documented privacy posture; none change the wire format. Net delta is +13 -3 across the two files. The remaining round-1 warnings (`scroll_depth` surface remap, `outbound_<target>` JS-side cardinality guard) and nits stand against the same code.

## Re-verification of round 1 findings

| Finding | Round 1 line | Status now | Note |
|---|---|---|---|
| `scroll_depth` "help"→"action" remap is client-side-only, no test | [`analytics.ts:226-231`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L226-L231) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L226-L231) | unchanged | Comment at L223-225 now spells out the rationale, but coupling and missing test remain. |
| `outbound_<target>` JS-side has no allowlist guard | [`analytics.ts:255-258`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L255-L258) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L255-L258) | unchanged | Today bounded by Go constants in [`components/analytics.go:8-14`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/analytics.go#L8-L14) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/analytics.go#L8-L14) — future drift risk only. |
| `copy_action.kind` priority not documented | [`analytics.ts:38-43`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L38-L43) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L38-L43) | unchanged | Cosmetic. |
| Missing tests: `AnalyticsHostname` rendering, `RedirectAnalytics`, zero-`ViewMode` fallback | various | unchanged | All still gaps; none blocking. |

I'm downgrading the cardinality and surface-remap items from Warnings to Nits in this round: the present cardinality surface is provably bounded by Go constants, the remap rationale is now in-line in the source, and neither failure mode is a correctness or privacy issue today.

## Fix

No code changed since round 1 other than the noscript pixel referrerpolicy ([analytics.html:8](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/layouts/analytics.html#L8) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/layouts/analytics.html#L8)). The remaining delta is documentation.

## Tests run

`GNOROOT=$PWD go test -count=1 ./gno.land/pkg/gnoweb/...` with `TestAnalytics|TestRedirectView|TestEnrichFooterData|TestStaticHeaderGeneralLinks|TestClassifyPageType` filter — all pass. `TestAnalytics` exercises the bootstrap script load, `data-page-type` rendering for 5 routes, and the disabled path. CI (`gh pr checks 5554 -R gnolang/gno`) green: 100+ checks pass, including `main / test`, `gnoweb_front_lint`, `gnoweb_generate`. Merge blocked only on codeowner approval (gfanton + alexiscolin codeowner reciprocity).

## Critical (must fix)

None.

## Warnings (should fix)

None blocking. See round 1 nits / suggestions; nothing has regressed.

## Nits

- [`analytics.ts:226-231`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L226-L231) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L226-L231) — `scroll_depth` derives `surface` from `page_type` ("source"→"source", "help"→"action") in JS only. The comment at L223-225 names the reason. The coupling is fragile only if `ClassifyPageType` ever renames `help`. A one-line table-driven Go test against [`components/analytics.go:39`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/analytics.go#L39) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/analytics.go#L39) asserting `"help"` is the label would lock the contract — cheaper than centralizing the remap. Not blocking.

- [`analytics.ts:255-258`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L255-L258) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/frontend/js/analytics.ts#L255-L258) — `outbound_${target}` interpolation has no JS-side allowlist. All four template emission sites ([`layouts/footer.html:9`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/layouts/footer.html#L9) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/layouts/footer.html#L9), [`:16`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/layouts/footer.html#L16) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/layouts/footer.html#L16), [`:44`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/layouts/footer.html#L44) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/layouts/footer.html#L44), [`layouts/header.html:100`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/layouts/header.html#L100) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/layouts/header.html#L100)) read from the Go `Outbound` field on `FooterLink` / `HeaderLink`, which only takes one of seven `Outbound*` constants. The cardinality is provably bounded today. The robustness gap is only "future template change emits dynamic string" — a doc note in `SIMPLEANALYTICS.md` ("Outbound target: only constants from `components/analytics.go`") would be cheaper than a JS allowlist.

- Same risk class applies to `mode_change.mode`, `theme_toggle.theme`, `list_sort_change.order`, `list_display_change.mode`: the listener forwards whatever the controller / radio value emits, trusting the server-rendered enum. Today all bounded; future drift would leak unbounded strings into SA. Mention it in `SIMPLEANALYTICS.md` § "Adding a new event" rather than guarding in JS.

- [`analytics_test.go:22`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/analytics_test.go#L22) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/analytics_test.go#L22) — the "unknown view + zero mode" case actually uses `ViewModeExplorer`, not the zero mode. Either rename the case or add a separate `ViewMode(0)` test to cover the genuine zero-mode fallthrough.

- [`app_test.go:236-266`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/app_test.go#L236-L266) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/app_test.go#L236-L266) — the `page_type` test covers 5 routes (`home`, `realm`, `help`, `source`, plus `/r/sys/users`→`realm`). No assertion for `pure`, `user`, `directory`, `status`, `redirect`, `explorer`, or `other`. The enum is documented as 11 values; the test covers 4 distinct ones.

- No assertion that `data-hostname` reaches the SA `<script>` tag when `cfg.AnalyticsHostname` is set ([`app.go:121`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/app.go#L121) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/app.go#L121) → [`layouts/analytics.html:4-5`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/layouts/analytics.html#L4-L5) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/layouts/analytics.html#L4-L5)). One additional `t.Run` in `TestAnalytics` that sets `cfg.AnalyticsHostname = "gno.land"` and asserts `data-hostname="gno.land"` in the body would close this gap. Same idea for [`handler_http.go:42-51`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/handler_http.go#L42-L51) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/handler_http.go#L42-L51) `RedirectAnalytics()` — `TestRedirectView` only asserts `Enabled`; it does not verify `Hostname`, `PageType=="redirect"`, or `ChainId` propagate.

## Suggestions

- [`SIMPLEANALYTICS.md`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/SIMPLEANALYTICS.md) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/SIMPLEANALYTICS.md) "Adding a new event" section is the right home for one extra bullet: "JS-side handlers trust server-rendered enum values; if you add a `change`-based listener that forwards a `value` to SA, make sure the only emitters are server-rendered." That keeps the cardinality contract maintainable without introducing JS allowlists that would need to stay in sync.

- The `<noscript>` pixel and the `latest.js`/`auto-events.js` scripts all use the same `data-hostname` value. Consider extracting a small `analyticsBodyAttrs` template helper if a second hostname-conditional attribute lands later — for now, three sites with the same `{{ with .Hostname }}` block is fine.

## Questions for Author

- The Round 1 question on whether the redirect page actually fires custom events is now answered by code reading: it loads `analytics.js` via [`views/redirect.html:13`](https://github.com/gnolang/gno/blob/9b25bcd6b/gno.land/pkg/gnoweb/components/views/redirect.html#L13) · [↗](../../../../../.worktrees/gno-review-5554/gno.land/pkg/gnoweb/components/views/redirect.html#L13), but the redirect body contains no copy buttons, no forms, no scroll content, no breadcrumb. Pageview fires (`page_type=redirect`); custom events have nothing to bind to. Confirm this is the intended behavior — if so, worth a one-line doc note in `SIMPLEANALYTICS.md` next to the `redirect` page_type row.

- The `params_filled` listener iterates all `param-input` elements once (line 187 `forEach((i) => i.removeEventListener("input", onInput))`). On a 50-param action page the `forEach` re-runs once per page-load; fine. But the read-from-`paramInputs`-snapshot loop also de-registers listeners that may have been re-bound by `controller-action-function.ts`. Have you observed the controller swapping out param inputs during a session (e.g. for dynamic forms), and if so does that defeat the one-shot semantics?

