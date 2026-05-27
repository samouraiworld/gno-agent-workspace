# PR #5677: docs: list per-function stdlib gaps in compatibility doc

URL: https://github.com/gnolang/gno/pull/5677
Author: davd-gzl | Base: master | Files: 1 | +69 -27
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5677 8ba854d` (then `gh -R gnolang/gno pr checkout 5677` inside it)

**Verdict: REQUEST CHANGES** — one fabricated symbol (`NewChaCha8` in `math/rand`), two miscategorized Gno-only package rows (`chain/params` mislabeled as accessors, `sys/params` advertised as setters-only while it ships getters too), and an incomplete Gno-only table (missing `math/overflow`, `crypto/chacha20/chacha`, `crypto/chacha20/rand`). Renumber and reorder is clean and the legend pass is otherwise accurate.

## Summary

Self-review of a docs-only PR I authored. Replaces a sparse, per-package status table with footnote-level detail for seven packages whose package-level status undersells the API gap (`crypto/cipher`, `crypto/subtle`, `encoding/binary`, `errors`, `io`, `math/rand`, `sort`, `time`), reclassifies `crypto/subtle` `tbd -> part` (it ships `XORBytes`), and adds a `Gno-only standard libraries` section. Renumbering the footnotes to be contiguous in package-table order is a strict readability win.

The PR's load-bearing claims about what is/isn't in each stdlib were verified against [`gnovm/stdlibs/`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs). Most checks pass. The bugs below are claims that don't survive a grep.

## Fix

The diff is purely additive/documentation: nine footnotes get new text, one package's status flips (`crypto/subtle` `tbd -> part`), and a new H2 section appears at the bottom listing seven Gno-only packages. No behavior changes.

## Critical (must fix)

None.

## Warnings (should fix)

- **[fabricated symbol]** [`docs/resources/go-gno-compatibility.md:315`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L315) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L315) — `NewChaCha8` is listed as a v2 constructor in Gno's `math/rand` but does not exist in the port.
  <details><summary>details</summary>

  Footnote `[^12]` instructs readers to use the v2 constructors "`New`, `NewPCG`, `NewChaCha8`". Grepping [`gnovm/stdlibs/math/rand/`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/math/rand) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/math/rand) for `ChaCha8` returns zero matches. Only [`New`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/math/rand/rand.gno#L42) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/math/rand/rand.gno#L42) and [`NewPCG`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/math/rand/pcg.gno) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/math/rand/pcg.gno) are exported. A reader who follows this guidance writes code that fails to compile. Fix: drop `NewChaCha8` from the list, or add the constructor first and update this doc in the same PR.
  </details>

- **[wrong label, wrong scope]** [`docs/resources/go-gno-compatibility.md:334`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L334) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L334) — `chain/params` row reads "Chain-parameter accessors", but the package exposes only setters and they are realm-local, not chain-wide.
  <details><summary>details</summary>

  The package doc at [`gnovm/stdlibs/chain/params/params.gno:1-4`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/chain/params/params.gno#L1-L4) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/chain/params/params.gno#L1-L4) says "functions for setting arbitrary realm-local parameters that can be called from any realm" and the exported API is `SetString`, `SetBool`, `SetInt64`, `SetUint64`, `SetBytes`, `SetStrings`, `UpdateParamStrings` — no `Get*`. Two issues: (a) "accessors" implies reads, but it's write-only; (b) "Chain-parameter" suggests global chain state, but these are realm-scoped. Fix: change description to "Realm-local parameter setters (`SetString`, `SetBool`, `SetInt64`, ..., `UpdateParamStrings`)."
  </details>

- **[undersells the API]** [`docs/resources/go-gno-compatibility.md:336`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L336) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L336) — `sys/params` row says "System-parameter setters" but the package ships getters too.
  <details><summary>details</summary>

  [`gnovm/stdlibs/sys/params/params.gno`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/sys/params/params.gno) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/sys/params/params.gno) exports the full matched pair: `SetSysParam{String,Bool,Int64,Uint64,Bytes,Strings}` AND `GetSysParam{String,Bool,Int64,Uint64,Bytes,Strings}`, plus `UpdateSysParamStrings`. The current description hides half the surface. Fix: change to "System-parameter setters and getters (`SetSysParam*`, `GetSysParam*`, `UpdateSysParamStrings`)."
  </details>

- **[incomplete table]** [`docs/resources/go-gno-compatibility.md:330-338`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L330-L338) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L330-L338) — the Gno-only section omits three packages that match the section's own criterion ("part of the Gno stdlib but no Go counterpart"): `math/overflow`, `crypto/chacha20/chacha`, `crypto/chacha20/rand`.
  <details><summary>details</summary>

  Running `find gnovm/stdlibs -maxdepth 4 -name gnomod.toml` surfaces all three:
  - [`gnovm/stdlibs/math/overflow/`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/math/overflow) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/math/overflow) — generated overflow-checked arithmetic helpers, no Go equivalent.
  - [`gnovm/stdlibs/crypto/chacha20/chacha/`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/crypto/chacha20/chacha) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/crypto/chacha20/chacha) — generic/ref ChaCha20 primitives, exposed as a subpackage.
  - [`gnovm/stdlibs/crypto/chacha20/rand/`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/crypto/chacha20/rand) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/crypto/chacha20/rand) — ChaCha20-backed RNG.

  Decide a rule: either the table lists every top-level Gno-only package (then add `math/overflow`), or it lists every Gno-only import path (then also add the two `chacha20` subpackages). The current half-coverage will rot fast as new Gno-only packages land.
  </details>

## Nits

- [`docs/resources/go-gno-compatibility.md:289-292`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L289-L292) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L289-L292) — `crypto/subtle` footnote says "ships `XORBytes` only" but also exports [`XORBytesUnsafe`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/crypto/subtle/xor.gno#L25) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/crypto/subtle/xor.gno#L25) (Gno-only, has the Go-style `(dst, x, y)` signature).
  <details><summary>details</summary>

  Worth surfacing because `XORBytesUnsafe` is the only function with a Go-equivalent signature (`(dst, x, y []byte) int`); Gno's `XORBytes` deviates from Go by allocating internally and returning the buffer instead of writing into `dst`. A reader porting Go code will hit a signature mismatch on `XORBytes` and probably wants `XORBytesUnsafe`. Mentioning both keeps the footnote useful.
  </details>

- [`docs/resources/go-gno-compatibility.md:311-315`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L311-L315) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L311-L315) — `math/rand` footnote says the "global `Source` interface" is not available, but [`Source`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/math/rand/rand.gno#L31-L33) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/math/rand/rand.gno#L31-L33) is defined — just with the v2 single-method signature (`Uint64() uint64`).
  <details><summary>details</summary>

  The literal statement "global Source interface is not available" is false. What's missing is the v1 `Source` shape (`Int63() int64; Seed(int64)`). Suggest rephrasing as "the v1 `Source` interface shape (`Int63`/`Seed`) is replaced by the v2 `Source` (single `Uint64() uint64` method)."
  </details>

- [`docs/resources/go-gno-compatibility.md:276-280`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L276-L280) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L276-L280) — `crypto/cipher` footnote groups `StreamReader`/`StreamWriter` under "interfaces" but they are structs at [`cipher.gno:82-91`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/crypto/cipher/cipher.gno#L82-L91) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/crypto/cipher/cipher.gno#L82-L91), and they lack the `Read`/`Write` methods that make them satisfy `io.Reader`/`io.Writer` in Go. Currently non-functional wrappers; worth a half-sentence noting the methods are stubbed out.

- [`docs/resources/go-gno-compatibility.md:321-324`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L321-L324) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L321-L324) — `time` footnote lists `After` as missing without disambiguating from the existing `Time.After(u Time) bool` method at [`time.gno:91-92`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/time/time.gno#L91-L92) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/time/time.gno#L91-L92). A reader searching for "time.After" sees results and concludes the footnote is wrong. Spell it out as "top-level `After(d Duration) <-chan Time`".

- [`docs/resources/go-gno-compatibility.md:332`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L332) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L332) — `chain` row description lists `Emit/Event` but the package only exposes the [`Emit`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/chain/emit_event.gno) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/chain/emit_event.gno) function; there is no public `Event` type to call out. Drop `/Event` or rename to "`Emit`".

## Missing Tests

None. Docs-only PR; no executable behavior to test.

## Suggestions

- [`docs/resources/go-gno-compatibility.md:99`](https://github.com/gnolang/gno/blob/8ba854d/docs/resources/go-gno-compatibility.md#L99) · [↗](../../../../../.worktrees/gno-review-5677/docs/resources/go-gno-compatibility.md#L99) — the HTML-comment "generated with" recipe at the top of the table is stale and refers to a `go/src` enumeration. Worth either deleting or replacing with a `gnovm/stdlibs/`-based command so the next person editing this table has a working starting point. Out of scope for this PR; flag for a follow-up.

- Consider committing a small `scripts/check-stdlib-compat-doc.sh` that diffs the table rows against `find gnovm/stdlibs -name gnomod.toml`. Catches future drift (e.g. the `encoding/csv` `todo` row is already wrong — see Questions). Out of scope for this PR.

## Questions for Author

- Pre-existing: `crypto/md5` and `crypto/sha1` are listed as `test`, but there is no [`gnovm/stdlibs/crypto/md5`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/crypto) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/crypto) or `crypto/sha1` directory and `git log --all -- '**/md5*' '**/sha1*'` returns nothing. Either the implementations live under another name or the rows are wrong. Worth a follow-up PR to either add the impls or downgrade the rows to `tbd`/`todo`.

- Pre-existing: `encoding/csv` is listed as `todo` in the package table but [`gnovm/stdlibs/encoding/csv/`](https://github.com/gnolang/gno/blob/8ba854d/gnovm/stdlibs/encoding/csv) · [↗](../../../../../.worktrees/gno-review-5677/gnovm/stdlibs/encoding/csv) ships a full `Reader`/`Writer`. Probably `full`. Same follow-up.
