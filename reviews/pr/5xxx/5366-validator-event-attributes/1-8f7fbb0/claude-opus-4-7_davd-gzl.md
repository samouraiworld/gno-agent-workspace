# PR #5366: feat(validators): add attributes to validator event emissions

URL: https://github.com/gnolang/gno/pull/5366
Author: mvallenet | Base: master | Files: 3 | +86 -6
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5366 8f7fbb0` (then `gh -R gnolang/gno pr checkout 5366` inside it)

**Verdict: APPROVE** — additive event-attribute change with passing txtar; only a small testing gap on the remove path and minor stringification nits.

## Summary

`r/sys/validators/v2` emitted `ValidatorAdded` / `ValidatorRemoved` as bare signals — indexers had to make an extra VM call (`GetChanges`) to learn which validator changed. This PR attaches `address`, `pubkey`, and `voting_power` (add only) attributes to those events so external consumers can index off the event payload alone. The consensus path is unchanged: the EndBlocker in [`gno.land/pkg/gnoland/app.go:478-535`](https://github.com/gnolang/gno/blob/8f7fbb0/gno.land/pkg/gnoland/app.go#L478-L535) · [↗](../../../../../.worktrees/gno-review-5366/gno.land/pkg/gnoland/app.go#L478-L535) still scrapes the realm via `valRegexp` against `GetChanges`, and the comment on `validatorUpdate` is updated to reflect that the empty struct is still all the collector needs.

## Glossary

- `validatorUpdate` — empty notification struct used by the EndBlocker collector to detect that a scrape is needed; carries no data.
- `valRegexp` — string-matching extractor in `gno.land/pkg/gnoland/app.go` that parses the VM response of `GetChanges(height)` into typed validator updates.

## Fix

Before, `addValidator` / `removeValidator` in [`examples/gno.land/r/sys/validators/v2/validators.gno:41,70`](https://github.com/gnolang/gno/blob/8f7fbb0/examples/gno.land/r/sys/validators/v2/validators.gno#L41-L75) · [↗](../../../../../.worktrees/gno-review-5366/examples/gno.land/r/sys/validators/v2/validators.gno#L41-L75) called `chain.Emit(validators.ValidatorAddedEvent)` / `Emit(validators.ValidatorRemovedEvent)` with no payload. After, the same calls pass `address` + `pubkey` for both events, plus `voting_power` for adds. The Go-side filter in [`gno.land/pkg/gnoland/validators.go:22-25,52-54`](https://github.com/gnolang/gno/blob/8f7fbb0/gno.land/pkg/gnoland/validators.go#L22-L54) · [↗](../../../../../.worktrees/gno-review-5366/gno.land/pkg/gnoland/validators.go#L22-L54) gets only a comment refresh — the type and behaviour are unchanged because the consensus-side mapping still flows through `GetChanges` regex parsing, not event attrs. New txtar [`gno.land/pkg/integration/testdata/validator_events.txtar`](https://github.com/gnolang/gno/blob/8f7fbb0/gno.land/pkg/integration/testdata/validator_events.txtar) · [↗](../../../../../.worktrees/gno-review-5366/gno.land/pkg/integration/testdata/validator_events.txtar) exercises the Add path end-to-end via gov/dao v3.

Verified locally: `TestTestdata/validator_events` passes (6.6s), `TestEndBlocker` subtests pass, `gno test ./examples/gno.land/r/sys/validators/v2/` passes. Emitted payload in the test run:
```
{"type":"ValidatorAdded","attrs":[
  {"key":"address","value":"g1ut590acnamvhkrh4qz6dz9zt9e3hyu499u0gvl"},
  {"key":"pubkey","value":"gpub1pgfj...mzu0r9h6gny6eg8c9dc303xrrudee6z4he4y7cs5rnjwmyf40yaj"},
  {"key":"voting_power","value":"1"}],
  "pkg_path":"gno.land/r/sys/validators/v2"}
```

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`examples/gno.land/r/sys/validators/v2/validators.gno:45`](https://github.com/gnolang/gno/blob/8f7fbb0/examples/gno.land/r/sys/validators/v2/validators.gno#L45) · [↗](../../../../../.worktrees/gno-review-5366/examples/gno.land/r/sys/validators/v2/validators.gno#L45) — `ufmt.Sprintf("%d", val.VotingPower)` could be `strconv.FormatUint(val.VotingPower, 10)` — same result, no format-string parser involved. Not worth blocking on for one call site.
- [`gno.land/pkg/gnoland/validators.go:24`](https://github.com/gnolang/gno/blob/8f7fbb0/gno.land/pkg/gnoland/validators.go#L24) · [↗](../../../../../.worktrees/gno-review-5366/gno.land/pkg/gnoland/validators.go#L24) — trailing comment ends without a period (`enough to trigger a VM scrape`); other comments in the same file end with periods. Cosmetic.

## Missing Tests

- **[remove path uncovered]** [`gno.land/pkg/integration/testdata/validator_events.txtar:24`](https://github.com/gnolang/gno/blob/8f7fbb0/gno.land/pkg/integration/testdata/validator_events.txtar#L24) · [↗](../../../../../.worktrees/gno-review-5366/gno.land/pkg/integration/testdata/validator_events.txtar#L24) — only `ValidatorAdded` is asserted; `ValidatorRemoved` event attributes (`address`, `pubkey`) are not exercised end-to-end.
  <details><summary>details</summary>

  The new txtar drives one proposal that adds a validator and asserts the three add-event attrs. The remove path goes through the same `chain.Emit(...)` shape but is not covered by integration. The realm's Gno test (`validators_test.gno`) exercises `removeValidator` but does not observe emitted events. A second proposal in the same txtar that removes the just-added validator would close the loop in ~15 lines and is symmetric with the add assertion already present.
  </details>

## Suggestions

- [`examples/gno.land/r/sys/validators/v2/validators.gno:41-46,70-74`](https://github.com/gnolang/gno/blob/8f7fbb0/examples/gno.land/r/sys/validators/v2/validators.gno#L41-L74) · [↗](../../../../../.worktrees/gno-review-5366/examples/gno.land/r/sys/validators/v2/validators.gno#L41-L74) — consider documenting the event attribute schema (keys + value types) in `examples/gno.land/p/sys/validators/types.gno` alongside `ValidatorAddedEvent` / `ValidatorRemovedEvent` constants so indexer authors have one source of truth.
  <details><summary>details</summary>

  Right now the constants in [`examples/gno.land/p/sys/validators/types.gno:39-42`](https://github.com/gnolang/gno/blob/8f7fbb0/examples/gno.land/p/sys/validators/types.gno#L39-L42) · [↗](../../../../../.worktrees/gno-review-5366/examples/gno.land/p/sys/validators/types.gno#L39-L42) only describe when the event fires, not the attribute payload. Issue #5344 explicitly mentions external indexer pain on this realm — a doc comment listing the attrs (`address`, `pubkey`, `voting_power` for adds; `address`, `pubkey` for removes) prevents drift if more attrs are added later.
  </details>

## Questions for Author

- Was the `voting_power` attribute deliberately omitted from `ValidatorRemoved`? Including it (always `0` post-removal, by convention in the realm at [`validators.gno:60-65`](https://github.com/gnolang/gno/blob/8f7fbb0/examples/gno.land/r/sys/validators/v2/validators.gno#L60-L65) · [↗](../../../../../.worktrees/gno-review-5366/examples/gno.land/r/sys/validators/v2/validators.gno#L60-L65)) would make the two events schema-symmetric and avoid surprising indexers that key off attribute presence.
