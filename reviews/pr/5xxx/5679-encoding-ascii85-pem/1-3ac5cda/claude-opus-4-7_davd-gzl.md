# PR #5679: feat(stdlibs): port encoding/ascii85 and encoding/pem

URL: https://github.com/gnolang/gno/pull/5679
Author: davd-gzl | Base: master | Files: 14 | +1895 -32
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]
Local worktree: `git -C gno worktree add .worktrees/gno-review-5679 3ac5cda` (then `gh -R gnolang/gno pr checkout 5679` inside it)

**Verdict: APPROVE** — Two upstream-faithful stdlib ports (ascii85, pem) plus the bytes prerequisites they need; both port divergences (inlined fallthrough in [`ascii85.gno:43-53`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/ascii85/ascii85.gno#L43-L53) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/ascii85/ascii85.gno#L43-L53), `sort.Strings` for `slices.Sort` in [`pem.gno:292`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/pem/pem.gno#L292) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/pem/pem.gno#L292)) are documented inline and behaviourally equivalent. CI green, all new and adjacent tests pass locally including the apphash regression. Self-review by author; concerns below are nits and one missing ADR.

## Summary

Ports Go 1.26.3's `encoding/ascii85` and `encoding/pem` byte-identical to upstream where gno permits. To unblock `pem.gno`, also lands the `bytes` additions from #5676 (`Cut`, `CutPrefix`, `CutSuffix`, `Clone`, `ContainsFunc`, `Buffer.Available`, `Buffer.AvailableBuffer`, `Buffer.Peek`) and rewrites `docs/resources/go-gno-compatibility.md` with footnote-level gaps for 8 packages plus a new "Gno-only standard libraries" section. Consensus-breaking: adds 2 packages to the genesis save set, shifting the iavlStore Merkle root; apphash pin in [`apphash_crossrealm38_test.go:53`](https://github.com/gnolang/gno/blob/3ac5cda/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L53) · [↗](../../../../../.worktrees/gno-review-5679/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L53) and qpaths golden in [`gnokey_qpaths.txtar:114,118`](https://github.com/gnolang/gno/blob/3ac5cda/gno.land/pkg/integration/testdata/gnokey_qpaths.txtar#L114-L118) · [↗](../../../../../.worktrees/gno-review-5679/gno.land/pkg/integration/testdata/gnokey_qpaths.txtar#L114-L118) updated.

## Glossary

- **fallthrough-from-default**: gno preprocessor rejects `fallthrough` from a `default:` clause even when textually first; only `case→case` works.
- **save set**: the per-tx set of escaped objects that get written to iavlStore; expansion shifts the apphash.
- **apphash pin**: hardcoded multistore commit hash in `apphash_crossrealm38_test.go` that fails any silent save-set drift.

## Fix

`ascii85.gno` and `pem.gno` are dropped under `gnovm/stdlibs/encoding/{ascii85,pem}/` with the two structural divergences from Go 1.26.3 inlined and `// XXX:`-annotated: (i) the `default→case 3→case 2→case 1` fallthrough chain in `Encode` is unrolled into the `default` body because gno's preprocessor rejects `fallthrough` from `default`; (ii) `slices.Sort(h)` on the header-key slice is replaced with `sort.Strings(h)` because gno has no `slices` package. The bytes additions and `Buffer.Peek` are pulled in verbatim from upstream Go 1.26 / 1.21 vintage. Tests are byte-identical where possible; `TestFuzz` and `FuzzDecode` are dropped (no `testing/quick`, no fuzz), `reflect.DeepEqual` is replaced with a local `blockEqual` helper, `TestCVE202224675` is scaled 100x down (regression covers recursion depth, not input size), and the upstream alloc-test assertions on `AvailableBuffer`/`Clone` are commented (no `AllocsPerRun`, no `unsafe`). `generated.go` initOrder grows two entries; apphash + qpaths goldens are bumped.

## Benchmarks / Numbers

Local Go-side regression: `TestAppHashCrossrealm38` passes in 5s, integration suite (txtar) in 128s, both `encoding/ascii85` and `encoding/pem` gno-side suites green (ascii85 1.6s, pem 6s of which 4.4s is `TestCVE202224675` at the reduced N=100k).

## Critical (must fix)

None.

## Warnings (should fix)

- **[non-trivial AI-assisted, consensus-breaking, no ADR]** [`gnovm/adr/`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/adr/) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/adr/) — PR ships new stdlibs that shift the apphash but no ADR documenting the decision.
  <details><summary>details</summary>

  `gno/AGENTS.md` requires an ADR for every non-trivial AI-assisted PR; this one adds two stdlib packages, edits `initOrder`, and bumps the multistore commit hash. The PR body covers the divergences but a real ADR would also lock down: (a) why ascii85 and pem are the next ports (any user demand / dependency, or just "fill the compat table"); (b) the chain-upgrade gating story for adding stdlibs — every new package shifts apphash for any genesis-replay scenario, so the policy for "when can we add a stdlib mid-chain" matters; (c) the precedent set by the two documented divergences (inlined fallthrough, sort.Strings) so the next port stays in the same shape. The apphash test comment at [`apphash_crossrealm38_test.go:28`](https://github.com/gnolang/gno/blob/3ac5cda/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L28) · [↗](../../../../../.worktrees/gno-review-5679/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L28) literally says "See the ADR note in the PR description", but the PR body has no ADR section and no ADR file exists. Fix: add `gnovm/adr/pr5679_encoding_ascii85_pem.md` covering context (compat-table push), decision (port byte-identical, document divergences inline), alternatives (rewrite for gno, wait for generics), and consequences (apphash shift, every new stdlib costs a pin bump).
  </details>

- **[branch behind master]** [`gnovm/stdlibs/generated.go`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/generated.go) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/generated.go) — PR is ~25 commits behind `origin/master` including #5669 (interrealm Phase 3) which restructures `chain/runtime{,/unsafe}` and `chain/banker` natives; merging will require a `go generate` re-run and a fresh apphash pin recapture.
  <details><summary>details</summary>

  Comparing the PR HEAD to `origin/master` surfaces a generated.go delta that is purely a master-side rewrite (runtime/unsafe → runtime, new `assertCallerIsRealm`, new `originSend`). When this PR merges master, the initOrder additions for ascii85/pem need to be re-applied on top of the new generated.go shape, and `TestAppHashCrossrealm38` will likely fail because Phase 3 changes the save set independently of this PR. CI is green only against the stale base. Fix: merge master, re-run `go generate ./gnovm/stdlibs/...`, re-capture `expectedCrossrealm38Hash`, re-run qpaths golden. Per project convention this is the contributor's responsibility before flagging for re-review.
  </details>

## Nits

- [`pem.gno:255-261`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/pem/pem.gno#L255-L261) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/pem/pem.gno#L255-L261) — `Encode` validates header keys for `:` but not `b.Type`; a Type containing `:`, `\n`, or `-----` would silently produce un-roundtrippable output.
  <details><summary>details</summary>

  Upstream has the same gap, so this is parity-faithful, not a regression introduced here. Worth flagging because the gno port is the right place to harden if we ever want to diverge — the upstream contract is "garbage in, garbage out" but gno realms tend to want stronger input validation. No action required for this PR; leaving the note for whoever revisits the package next.
  </details>

- [`pem_test.gno:46-62`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/pem/pem_test.gno#L46-L62) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/pem/pem_test.gno#L46-L62) — `blockEqual` is more permissive than `reflect.DeepEqual`: it treats `nil` and empty `Bytes`/`Headers` as equal.
  <details><summary>details</summary>

  `bytes.Equal(nil, []byte{})` is true and `len(nil) == len(map[string]string{})` is also true, so a regression where `Decode` returns `nil` instead of `[]byte{}` would not be caught. In practice [`pem.gno:182`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/pem/pem.gno#L182) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/pem/pem.gno#L182) always initializes `p.Bytes = []byte{}` before any branch that returns the block, so the test wouldn't catch a contract drift if someone introduced one. Stricter form would explicitly distinguish nil/empty for both fields. Optional.
  </details>

- [`pem_test.gno:189`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/pem/pem_test.gno#L189) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/pem/pem_test.gno#L189) — CVE-2022-24675 regression runs at 100k iterations vs upstream's 10M; consider adding a `testing.Short()` guard and keeping the 10M path for non-short runs, mirroring how upstream skips heavy fuzz.
  <details><summary>details</summary>

  The current divergence is well-justified — at gno-VM speed 10M takes minutes — but the regression check is a one-shot recursion-depth test that benefits from being heavy enough to actually stack-overflow on a regression. 100k passes in 4.4s on the gno VM; whether that's enough to catch a *future* recursion regression depends on how deep the bug would be reachable. Leaving as-is is fine; flag only.
  </details>

- [`ascii85.gno:45-53`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/ascii85/ascii85.gno#L45-L53) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/ascii85/ascii85.gno#L45-L53) — the inlined `default` body could be expressed as a one-line slice copy `binary.BigEndian.Uint32(src[:4])` if `encoding/binary` is stable enough; sticking with byte-by-byte parity is the safer call for now.

## Missing Tests

- **[round-trip with Proc-Type only]** [`pem_test.gno:196-201`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/pem/pem_test.gno#L196-L201) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/pem/pem_test.gno#L196-L201) — `TestEncode` only covers `privateKey2` which has both Proc-Type and one other header; the Proc-Type-only path in `Encode` ([`pem.gno:286-290`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/pem/pem.gno#L286-L290) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/pem/pem.gno#L286-L290)) is exercised only via the `pemData` decode path.
  <details><summary>details</summary>

  Upstream also has this gap (it falls out of the dropped `TestFuzz`). The Proc-Type-first-write branch is small but worth a direct test if we're not running fuzz coverage. A one-block `Block{Headers: {"Proc-Type": "..."}, ...}` round-trip would close it. Optional.
  </details>

## Suggestions

- [`gnomod.toml`](https://github.com/gnolang/gno/blob/3ac5cda/gnovm/stdlibs/encoding/pem/gnomod.toml) · [↗](../../../../../.worktrees/gno-review-5679/gnovm/stdlibs/encoding/pem/gnomod.toml) — consider declaring the explicit deps (`bytes`, `encoding/base64`, `errors`, `io`, `sort`, `strings`) in gnomod.toml so initOrder is enforceable by tooling rather than implicit from the auto-generated file.
  <details><summary>details</summary>

  Looking at sibling packages (`encoding/base64/gnomod.toml`, `encoding/csv/gnomod.toml`) they also omit explicit deps, so this would be a global convention change, not specific to this PR. Punt to a follow-up if it ever matters.
  </details>

## Questions for Author

- Is there a target Go upstream version we're pinning these ports to (currently 1.26.3 per PR body), or do we re-sync per port? A `// upstream: go1.26.3 src/encoding/pem/pem.go` header at the top of each ported file would make the next sync trivially diffable.
- The `// XXX:` annotations on test-side divergences are useful but inconsistent in form (some say "upstream uses X", some say "not in upstream"). Worth a one-line style note in `gno/AGENTS.md` or a `docs/contributing/porting-stdlibs.md` so the next stdlib port doesn't re-invent the convention.
- Should the apphash pin recapture be automated? A `go test -count=1 -run TestAppHashCrossrealm38 -capture` flag would beat the "run, see failure, paste hex" loop documented at [`apphash_crossrealm38_test.go:43-47`](https://github.com/gnolang/gno/blob/3ac5cda/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L43-L47) · [↗](../../../../../.worktrees/gno-review-5679/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L43-L47). Out of scope here.
