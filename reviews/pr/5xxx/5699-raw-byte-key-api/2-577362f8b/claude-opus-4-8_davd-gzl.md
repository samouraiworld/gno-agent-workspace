# PR #5699: feat(stdlibs): add raw byte-key API to chain/params

URL: https://github.com/gnolang/gno/pull/5699
Author: notJoon | Base: master | Files: 10 | +207 -5
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `577362f8b` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5699 577362f8b`

Round 2 of 2. Round 1 reviewed `5176f0c` ([1-5176f0c](../1-5176f0c/claude-opus-4-7_davd-gzl.md)). Since then the branch merged master (`849e31b`) and added one commit, `577362f8b`, which repins the `TestAppHashCrossrealm38` forward-guard. No other source changed: the two natives, `pkeyRaw`, the gas entries, the calibrator benches, the `testParams.SetBytes` nil-delete fix, and the `std12.gno` filetest are byte-identical to round 1.

**Verdict: APPROVE** — the round-1 blocker is resolved: CI is green (apphash repinned, test passes locally), and the only remaining "fail" is the bot's human-approval gate. The two earlier Warnings (silent keyspace aliasing with `SetBytes`; gas values copied not fit) survive but are design/calibration calls the author can own with a one-line note, not correctness defects. Recommend the author document the shared keyspace in the gno docstring before merge.

## Summary
Adds `SetBytesKey(key []byte, val []byte)` and `GetBytesKey(key []byte) ([]byte, bool)` to `chain/params` so IBC-style binary keys (which may contain `:`) can be stored without hex-encoding. Motivated by [onbloc/gno-ibc#39](https://github.com/onbloc/gno-ibc/issues/39): hex encoding produces non-canonical storage paths, breaking ICS23 light-client compatibility. The raw-key path bypasses the `:`-rejection check in [`pkey`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/chain/params/params.go#L63) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L63) via a parallel [`pkeyRaw`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/chain/params/params.go#L76) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L76) that builds the same `vm:<realm>:<key>` shape but accepts any bytes after the second `:`.

## What changed since round 1

| Round-1 finding | Status at `577362f8b` |
|---|---|
| CI red — apphash pin stale (Warning) | **Resolved.** Repinned to `bc8eae05…af61d` in `577362f8b`; `TestAppHashCrossrealm38` passes locally. |
| Silent keyspace aliasing with `SetBytes` (Warning) | **Open.** `pkeyRaw` still uses the identical `vm:<realm>:` prefix; cross-API read/delete confirmed empirically (repro below). |
| Gas values copied, not calibrated (Warning) | **Open.** `SetBytesKey`/`GetBytesKey` rows still verbatim copies; no `fit … R²=…` comment. |
| parsePrefix collision (Missing Test) | **Resolved.** `std12.gno` now exercises a colon-in-key (`{0x01, 0x3a, 0x02}`). |
| nil-delete parity untested (Nit/Missing Test) | **Resolved.** `std12.gno` now asserts `SetBytesKey(k, nil)` then `GetBytesKey(k)` returns `(nil, false)`. |
| keyspace-overlap detection test (Missing Test) | **Open.** No test mixes `SetBytes` and `GetBytesKey` on the same key. |
| defensive copy on read (Suggestion) | **Open / unchanged.** `keeper.GetBytes` still aliases store bytes (same as #5698). |
| gno-side docstring lacks shared-keyspace note (Nit/Suggestion) | **Open.** `params.gno` docstring unchanged. |

## Glossary
- `pkey` / `pkeyRaw`: realm-scoped key-prefix builders. `pkey` rejects `:`, `pkeyRaw` accepts any bytes.
- `parsePrefix`: keeper's `<module>:<rest>` splitter; splits on the FIRST `:`.
- `TestAppHashCrossrealm38`: forward-guard pinning the multistore root after a fixed scenario.

## Critical (must fix)

None.

## Warnings (should fix)

- **[silent keyspace aliasing with SetBytes]** [`gnovm/stdlibs/chain/params/params.go:76-82`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/chain/params/params.go#L76) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L76) — `SetBytes("foo", v)` and `SetBytesKey([]byte("foo"), v)` write to the same storage slot; carried from round 1, still unaddressed.
  <details><summary>details</summary>

  `pkey("foo")` and `pkeyRaw([]byte{'f','o','o'})` both produce the byte-identical string `vm:gno.land/r/x:foo` ([`params.go:71`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/chain/params/params.go#L71) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L71) vs [`:81`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/chain/params/params.go#L81) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L81)). There is no separator byte, length-prefix, or namespace tag distinguishing the two key shapes, so a realm mixing the APIs can overwrite or shadow its own state with no indication. A raw-API delete (`SetBytesKey(k, nil)`) also removes a string-API write under the same bytes. Confirmed empirically (repro below). This is fine if the IBC consumer owns that discipline, but the gno docstring at [`params.gno:14-20`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/chain/params/params.gno#L14) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.gno#L14) still doesn't say it. Fix: either namespace `pkeyRaw` (e.g. `vm:<realm>:b:` infix) so the two cannot collide, or state the shared keyspace in the `SetBytesKey`/`GetBytesKey` docstring. The doc note is the minimum.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5699 -R gnolang/gno
  cat > gnovm/tests/files/alias_probe.gno <<'EOF'
  package main

  import "chain/params"

  func main() {
  	params.SetBytes("foo", []byte("via-string"))
  	v, ok := params.GetBytesKey([]byte("foo"))
  	println("string->raw read ok:", ok, "val:", string(v))
  	params.SetBytesKey([]byte("foo"), nil)
  	_, ok2 := params.GetBytesKey([]byte("foo"))
  	println("after raw delete ok:", ok2)
  }

  // Output:
  // string->raw read ok: true val: via-string
  // after raw delete ok: false
  EOF
  go test -run 'TestFiles/alias_probe.gno$' ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/alias_probe.gno
  ```
  ```
  ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.324s
  ```
  The test passing means the `// Output:` block (cross-API read hits, raw delete clears the string-API write) is exactly what the VM produced. A namespaced design would print `ok: false` on the first line and fail the filetest.
  </details>

- **[gas values copied, not calibrated]** [`gnovm/stdlibs/native_gas.go:97-98`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/native_gas.go#L97) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/native_gas.go#L97) — `SetBytesKey`/`GetBytesKey` base & slope are verbatim copies; the new benchmarks were added but never fit. Carried from round 1.
  <details><summary>details</summary>

  `SetBytesKey` reads `Base: 1912, Slope: 13213` (identical to the `SetBytes` row directly above), and `GetBytesKey` reads `Base: 1912, PostSlope: 10584` (base copied from `SetBytes`, post-slope copied from `sys/params.getSysParamBytes` at [`:104`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/native_gas.go#L104) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/native_gas.go#L104)). Neither carries the project's usual `fit base=… slope=… R²=…` comment; they carry `// mirrors …` instead. The 8 benchmarks in [`native_machine_bench_test.go:312-352`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/cmd/calibrate/native_machine_bench_test.go#L312) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/cmd/calibrate/native_machine_bench_test.go#L312) and the 3 calibrator-table entries in [`gen_native_table.py:83-90`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/cmd/calibrate/gen_native_table.py#L83) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/cmd/calibrate/gen_native_table.py#L83) exist precisely to confirm that copy empirically, but the fit was not run. Two wrinkles make the copy slightly imprecise (both conservative, so not a blocker): `pkeyRaw` skips the `strings.Contains(key, ":")` check `pkey` runs, making the raw keying marginally cheaper; and `GetBytesKey` borrows the `SetBytes` base of `1912` (a write base) for a read, overcharging the flat component. Fix: run the new benches through the calibrator and replace the values with the fit, or add a one-line PR note that the copy is an intentional conservative first cut.
  </details>

## Nits

- [`gnovm/stdlibs/chain/params/params.gno:14-20`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/chain/params/params.gno#L14) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.gno#L14) — the `SetBytesKey` docstring says the key may contain `:` but not that it shares a namespace with `SetBytes`. The carve-out is documented in `tm2/pkg/sdk/params/doc.go`, but realm authors read the gno-side docstring, not the keeper doc. Add one sentence: "Keys share the `SetBytes` namespace; `SetBytesKey([]byte("k"), v)` and `SetBytes("k", v)` address the same slot."

- [`gnovm/stdlibs/chain/params/params.go:80-81`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/stdlibs/chain/params/params.go#L80) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L80) — `pkeyRaw` builds the key as `"vm:" + rlmPath + ":" + string(key)`. The `string(key)` cast plus concat allocates twice. Negligible for IBC-sized keys; flag for future tuning only.

## Missing Tests

- **[keyspace overlap detection]** [`gnovm/tests/files/std12.gno`](https://github.com/gnolang/gno/blob/577362f8b/gnovm/tests/files/std12.gno) · [↗](../../../../../.worktrees/gno-review-5699/gnovm/tests/files/std12.gno) — no test mixes `SetBytes("k", v)` with `GetBytesKey([]byte("k"))` (or the reverse) to lock down the aliasing as intended behavior. Carried from round 1.
  <details><summary>details</summary>

  `std12.gno` now covers colon keys, 32-byte keys, missing keys, and raw-API delete, but every case uses the raw API on both ends. A single assertion that crosses the string and raw APIs would either pin the shared-keyspace contract or surface a regression if a future change namespaces `pkeyRaw`. See [`tests/params_byte_key_alias.gno`](tests/params_byte_key_alias.gno).
  </details>

## Suggestions

- [`tm2/pkg/sdk/params/keeper.go:141-150`](https://github.com/gnolang/gno/blob/577362f8b/tm2/pkg/sdk/params/keeper.go#L141) · [↗](../../../../../.worktrees/gno-review-5699/tm2/pkg/sdk/params/keeper.go#L141) — `GetBytes` does `*ptr = bz` with no defensive copy, while `SetBytes` copies on write ([`keeper.go:191-193`](https://github.com/gnolang/gno/blob/577362f8b/tm2/pkg/sdk/params/keeper.go#L191) · [↗](../../../../../.worktrees/gno-review-5699/tm2/pkg/sdk/params/keeper.go#L191)). Whether the returned slice can alias mutable store memory depends on the iavl backend. Same question as #5698's `GetBytes`; worth confirming once for both, but not introduced by this PR (it reuses the existing keeper method).

## Questions for Author

- Was the shared keyspace with `SetBytes` an intentional design choice or a side effect of the most-direct implementation? Round-1 question, still open — a one-line docstring note settles it either way.
- The copied gas values are conservative (write base reused for a read, raw keying slightly cheaper than charged). Intentional first cut, or are the calibrator fits pending? A PR-description line would close this out.
