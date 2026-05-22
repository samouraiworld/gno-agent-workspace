# PR #5699: feat(stdlibs): add raw byte-key API to chain/params

URL: https://github.com/gnolang/gno/pull/5699
Author: notJoon | Base: master | Files: 9 | +206 -4
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — CI is red on the apphash forward-guard; the new keyspace silently aliases the existing string-keyed `SetBytes` (collision risk); and the new `SetBytesKey`/`GetBytesKey` gas costs are copied wholesale from their string-keyed siblings rather than calibrated from the new benchmarks.

## Summary
Adds `SetBytesKey(key []byte, val []byte)` and `GetBytesKey(key []byte) ([]byte, bool)` to `chain/params` so IBC-style binary keys (which may contain `:`) can be stored without hex-encoding. Motivated by [onbloc/gno-ibc#39](https://github.com/onbloc/gno-ibc/issues/39): hex encoding produces non-canonical storage paths like `prefix ++ hex(key)` instead of `prefix ++ raw_key`, breaking ICS23 light-client compatibility. The raw-key path bypasses the `:`-rejection check in [`pkey`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L51) by introducing a parallel [`pkeyRaw`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L74) that builds the same `vm:<realm>:<key>` shape but accepts any bytes after the second `:`.

```
existing:  pkey(m, "foo")        →  "vm:gno.land/r/x:foo"        (rejects ":" in key)
new:       pkeyRaw(m, []byte{…}) →  "vm:gno.land/r/x:" + raw     (accepts any bytes)
                                            ↑ same storage prefix
```

## Glossary
- `pkey` / `pkeyRaw`: the realm-scoped key-prefix builders. `pkey` rejects `:`, `pkeyRaw` accepts any bytes.
- `ICS23`: IBC commitment-proof spec; keys are raw bytes, not strings.
- `parsePrefix`: the keeper's `<module>:<rest>` splitter (splits on the FIRST `:`).
- `TestAppHashCrossrealm38`: forward-guard test pinning the multistore root after a fixed scenario.

## Fix
The diff adds two thin wrappers in [`gnovm/stdlibs/chain/params/params.go:40-44,55-60`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L40) that delegate to the existing `ParamsInterface.SetBytes` / `GetBytes` on a `pkeyRaw`-built storage key. The keeper's storage layer is already byte-keyed (raw `[]byte` via [`storeKey`](../../../../../.worktrees/gno-review-5699/tm2/pkg/sdk/params/keeper.go#L23)) and `SetBytes` already deletes on `value == nil` ([`keeper.go:184-190`](../../../../../.worktrees/gno-review-5699/tm2/pkg/sdk/params/keeper.go#L184)). The PR also fixes the in-memory test stub in [`gnovm/pkg/test/test.go:122-129`](../../../../../.worktrees/gno-review-5699/gnovm/pkg/test/test.go#L122) so `testParams.SetBytes(k, nil)` deletes too, matching production. Native gas adds two entries in [`native_gas.go:99-100`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/native_gas.go#L99) and the calibrator gets one Set/Get benchmark pair in [`native_machine_bench_test.go:322-362`](../../../../../.worktrees/gno-review-5699/gnovm/cmd/calibrate/native_machine_bench_test.go#L322).

## Critical (must fix)

None.

## Warnings (should fix)

- **[silent keyspace aliasing with SetBytes]** [`gnovm/stdlibs/chain/params/params.go:74-82`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L74) — `SetBytesKey([]byte("foo"), ...)` and `SetBytes("foo", ...)` write to the *same* storage slot.
  <details><summary>details</summary>

  `pkey("foo")` produces `vm:gno.land/r/x:foo`, and `pkeyRaw([]byte{'f','o','o'})` produces the byte-identical string `vm:gno.land/r/x:foo`. The two APIs therefore share a single keyspace whenever a raw key happens to be valid UTF-8 without `:`. A realm that mixes the two APIs (e.g. some keys via `SetBytes`, others via `SetBytesKey`) can overwrite or shadow its own state without any indication. There is no separator byte, length-prefix, or namespace tag distinguishing the two key shapes. The PR description argues raw keys avoid the cost of hex encoding for IBC, but the trade-off — that the binary keyspace silently overlaps the ASCII keyspace — isn't acknowledged in the docstring at [`params.gno:14-18`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.gno#L14). Fix: either (a) namespace `pkeyRaw` differently (e.g. `vm:<realm>:b:` prefix for raw-keyed entries) so the two APIs cannot collide, or (b) document the shared keyspace prominently and let callers manage their own discipline. Option (a) is the safer default; option (b) is acceptable if the IBC consumer is happy to own that discipline. Either way, please make the choice explicit in the doc.
  </details>

- **[CI red — pin hash needs updating]** [`gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:53`](../../../../../.worktrees/gno-review-5699/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L53) — `main / test` fails: expected `71ceb2a2…708fc`, got `2f70765e…b6ae`.
  <details><summary>details</summary>

  Same root cause as the sibling PR #5698: the forward-guard pins the multistore root after running `crossrealm38`, and `LoadStdlib` at [`common_test.go:69-76`](../../../../../.worktrees/gno-review-5699/gno.land/pkg/sdk/vm/common_test.go#L69) persists the (now-modified) `chain/params` package bytes into the iavl store before the scenario runs. The diff adds two function declarations to [`params.gno:14-20`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.gno#L14) plus the generated dispatcher in [`generated.go`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/generated.go), so the stored byte content shifts and the multistore root with it. The test's own comment ([line 50-52](../../../../../.worktrees/gno-review-5699/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L50-L52)) is explicit: confirm the change is consensus-affecting (it is — new natives), then re-run locally and paste the observed hash into the constant. Fix: `go test ./gno.land/pkg/sdk/vm/ -run TestAppHashCrossrealm38 -v`, copy the observed hash into `expectedCrossrealm38Hash`, and flag consensus impact in the PR description.
  </details>

- **[gas values copied, not calibrated]** [`gnovm/stdlibs/native_gas.go:99-100`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/native_gas.go#L99) — `SetBytesKey` / `GetBytesKey` base & slope are byte-identical to `SetBytes` / `GetBytes`; the new benchmarks were never fit.
  <details><summary>details</summary>

  The PR adds 8 new benchmarks in [`native_machine_bench_test.go:322-362`](../../../../../.worktrees/gno-review-5699/gnovm/cmd/calibrate/native_machine_bench_test.go#L322) and 3 calibrator-table entries in [`gen_native_table.py:83-89`](../../../../../.worktrees/gno-review-5699/gnovm/cmd/calibrate/gen_native_table.py#L83). The two gas entries in `native_gas.go` read `Base: 1912, Slope: 13213` (verbatim from the `SetBytes` row two lines up) and `Base: 1912, PostSlope: 10584` (the latter copied from `sys/params.getSysParamBytes`). No comment of the usual `fit base=… slope=… R²=…` form. The reasoning is defensible — `pkeyRaw` does the same `fmt.Sprintf`-equivalent string concat as `pkey`, and the keeper path is the same `SetBytes`/`GetBytes` — but the *purpose* of the calibrator is to confirm that assumption empirically. Notably `pkeyRaw` skips the `strings.Contains(key, ":")` check ([params.go:55-57](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L55)) that `pkey` performs, so the raw path is marginally cheaper at the keying step. Fix: run the new benchmarks through the calibrator and replace the copied values with the fit, or add a one-line PR-description note explaining why the copy is intentional and acceptable.
  </details>

## Nits

- [`gnovm/stdlibs/chain/params/params.go:74-82`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L74) — `pkeyRaw` builds the key as `"vm:" + rlmPath + ":" + string(key)`. The cast `string(key)` is cheap, but if `key` is huge (IBC isn't, but the API doesn't cap it) you allocate twice (cast then concat). A `make([]byte, ...)+copy` builder avoids the double allocation. Probably not worth changing for IBC-sized keys; flag for future tuning.

- [`gnovm/pkg/test/test.go:122-129`](../../../../../.worktrees/gno-review-5699/gnovm/pkg/test/test.go#L122) — the `SetBytes` change to delete on `nil` matches production ([`keeper.go:184-190`](../../../../../.worktrees/gno-review-5699/tm2/pkg/sdk/params/keeper.go#L184)). A one-line comment "match keeper.go SetBytes nil-deletes" would help the next reader who wonders why this method has special-case logic when the siblings don't.

- [`tm2/pkg/sdk/params/doc.go:18`](../../../../../.worktrees/gno-review-5699/tm2/pkg/sdk/params/doc.go#L18) — the new bullet ("VM realm-scoped parameter names may contain arbitrary bytes, including ':'.") is the right place to document this carve-out, but it should also be in the gno-side docstring at [`params.gno:14-20`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.gno#L14) (realm authors live there, not in `tm2/pkg/sdk/params/doc.go`).

## Missing Tests

- **[parsePrefix collision]** [`gnovm/tests/files/std12.gno`](../../../../../.worktrees/gno-review-5699/gnovm/tests/files/std12.gno) — what happens when a binary key contains the realm path itself, or when the binary key begins with bytes that look like a registered module prefix?
  <details><summary>details</summary>

  The keeper's `validate` calls `parsePrefix(key)` which splits on the *first* `:` ([keeper.go:317-329](../../../../../.worktrees/gno-review-5699/tm2/pkg/sdk/params/keeper.go#L317)). For a `pkeyRaw`-built key the first `:` is always after `vm`, so the module always parses to `"vm"` and the rest is `"<realm>:<raw_key>"` — fine in practice. But the assumption "the first `:` is always the module separator" is now load-bearing for the validate hook, and is not exercised by a test. A filetest with a raw key whose first byte is `:` (i.e. `vm:gno.land/r/x::…`) would lock that behavior down.
  </details>

- **[keyspace overlap detection]** [`gnovm/tests/files/std12.gno`](../../../../../.worktrees/gno-review-5699/gnovm/tests/files/std12.gno) — there's no test that calls `SetBytes("k", v1)` then `GetBytesKey([]byte("k"))` (or vice versa) to demonstrate the silent aliasing called out in the Warning above. Adding one would either confirm the design or surface a surprise in CI.

## Suggestions

- [`gnovm/stdlibs/chain/params/params.gno:14-20`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.gno#L14) — the docstring for `SetBytesKey` says "Unlike SetBytes, the key may contain any bytes, including ':'." Add a sentence stating that the two APIs share a storage namespace (i.e. `SetBytesKey([]byte("k"), v)` and `SetBytes("k", v)` collide). Important contract for realm authors planning to mix the two.

- [`gnovm/stdlibs/chain/params/params.go:55-60`](../../../../../.worktrees/gno-review-5699/gnovm/stdlibs/chain/params/params.go#L55) — consider returning the value via a defensive copy in `GetBytesKey`, mirroring `SetBytes`' write-side copy ([keeper.go:191-193](../../../../../.worktrees/gno-review-5699/tm2/pkg/sdk/params/keeper.go#L191)). Whether this matters depends on the iavl backend (whether `stor.Get` aliases internal storage) and is the same question as PR #5698's `GetBytes`. Probably already safe today, worth confirming once for both PRs.

## Questions for Author

- Was the shared keyspace with `SetBytes` an intentional design choice (so realms can opt into either API for any key) or a side effect of the most-direct implementation? If intentional, please document; if not, consider a `:b:` infix in `pkeyRaw` to keep the two disjoint.
- Same gas-calibration question as PR #5698: were the copied `Base`/`Slope` values intended as a conservative first cut, or are the calibrator fits pending?
- Was the `testParams.SetBytes` nil-delete change motivated by an observed divergence between `gno test` and chain behaviour, or proactive parity? Worth calling out either way in the PR description.
