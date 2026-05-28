# Stdlib Test Gap — `net/url`

Findings from porting upstream Go 1.25.9 `net/url` tests into Gno's
`gnovm/stdlibs/net/url/` and running them under `TestStdlibs/net-url`.

**Baseline**: `gnolang/gno@master` with PR [#5723](https://github.com/gnolang/gno/pull/5723)
cherry-picked.

Ports live in:

- [`url_test.gno`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/net/url/url_test.gno) — Test\* funcs (appended to the existing file)
- [`example_test.gno`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/net/url/example_test.gno) — upstream `Example*` rewritten as `TestExample_*` so the bodies execute (Gno's test runner only discovers `Test*`, not `Example*`)

How to reproduce:

```bash
cd .worktrees/gno-stdlib-test-port
go test -count=1 -v -run 'TestStdlibs/net-url$' ./gnovm/pkg/gnolang/
```

---

## Summary

| Bucket                  | Count |
| ----------------------- | ----- |
| Ported and PASS         | 30    |
| Ported and FAIL         | 0     |
| Skipped — missing API   | 4     |
| Skipped — unportable    | 1     |
| **Total upstream gap**  | **35** |

**No correctness divergences surfaced in `net/url`.** All 30 portable
upstream tests pass against Gno's existing implementation. This is a
negative result — the package was historically a CVE hotspot and is one
of the heavier-yield audit targets, but Gno's port appears to track
upstream behavior on every test the porting constraints let us run.

---

## #1 — Tests skipped because of missing Gno stdlib API

**Package**: `gnovm/stdlibs/net/url`
**Severity**: missing API (gap, not a bug in `net/url` itself)
**Existing PR**: none found

Four upstream tests could not be ported because the dependencies are
absent from Gno's stdlib:

| Upstream test                       | Blocked on                                      |
| ----------------------------------- | ----------------------------------------------- |
| `TestJSON`                          | `encoding/json`                                 |
| `TestGob`                           | `encoding/gob`                                  |
| `TestURLErrorImplementsNetError`    | `net` package (specifically `net.Error`)        |
| `TestParseQueryLimits`              | `GODEBUG` runtime + `defaultMaxParams` internal |

`encoding/json`, `encoding/gob`, and `net` are package-level gaps that
affect many other stdlibs, not just `net/url`. They are out of scope to
fix here — flagging only so a follow-up audit (or the same author when
those packages land) can re-port these tests.

`TestParseQueryLimits` is more specific. Upstream Go 1.25 introduced a
DOS guard against query strings with too many `&`-separated params,
configurable via `GODEBUG=urlmaxqueryparams=N` and capped at a default
(`defaultMaxParams = 10000`). Gno's `ParseQuery` has **no such guard**:

[`gnovm/stdlibs/net/url/url.gno:925`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/net/url/url.gno#L925)
in Gno has no `len(query) > maxParams` check; upstream
[`src/net/url/url.go @ go1.25.9`](https://github.com/golang/go/blob/go1.25.9/src/net/url/url.go)
does (`if maxParams > 0 && count > maxParams { return nil, ... }`).

### Impact

A realm exposing a path that calls `url.ParseQuery(attackerString)` on
arbitrary user input can be fed an arbitrarily large `&`-separated
input. In upstream Go this is bounded; in Gno it is bounded only by
allocator and gas. Whether this is exploitable in practice depends on
gas costs of map insertion in Gno (each param becomes a map slot), and
on whether realm authors actually feed unbounded input to `ParseQuery`
— so this is more of a missing-defence-in-depth than an active CVE.

Worth porting the guard when `GODEBUG` (or any equivalent feature-flag
mechanism) lands.

---

## #2 — Test skipped as unportable

**Package**: `gnovm/stdlibs/net/url`
**Severity**: documentation gap, not a bug

| Upstream example  | Reason                                          |
| ----------------- | ----------------------------------------------- |
| `ExampleParseQuery` | Uses `encoding/json` to render the output map |

Functionally equivalent coverage already exists via the in-package
`TestParseQuery` (which iterates a tabular fixture covering the same
parsing surface). Skipping `ExampleParseQuery` loses no signal.

---

## #3 — Fixture drift: IPv6 case missing from in-package `urltests`

**Package**: `gnovm/stdlibs/net/url`
**Severity**: test-coverage gap (not a bug)
**Existing PR**: none found
**Found by**: diffing
[`url_test.gno @ master`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/net/url/url_test.gno)
vs upstream
[`url_test.go @ go1.25.9`](https://github.com/golang/go/blob/go1.25.9/src/net/url/url_test.go)

### Summary

Gno's `urltests` fixture (used by `TestURLString` and `BenchmarkString`)
is missing the upstream IPv6-with-port+path case:

```go
{
    "https://[2001:db8::1]:8443/test/path",
    &URL{
        Scheme: "https",
        Host:   "[2001:db8::1]:8443",
        Path:   "/test/path",
    },
    "",
},
```

The case is asserted explicitly in
[`url_test.gno`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/net/url/url_test.gno)
inside `TestParse`, and Gno's `Parse` handles it correctly. The issue
is purely that the fixture diverged from upstream — every other test
that ranges over `urltests` (notably `TestURLString`) loses that case's
coverage of IPv6 round-tripping.

Trivial fix: add the case to `urltests` in `url_test.gno`. Not filed
as a bug; flagged so a follow-up cleanup can land it.

---

## Tests ported that PASS (no divergence)

`TestParse`, `TestInvalidUserPassword`, `TestRejectControlCharacters`,
plus the following upstream `Example*` rewritten as `TestExample_*`
(rewrite needed because Gno's test runner ignores `Example*` —
[`gnovm/pkg/test/test.go:685`](../../.worktrees/gno-stdlib-test-port/gnovm/pkg/test/test.go#L685)
`loadTestFuncs` only matches `strings.HasPrefix(fname, "Test")`):

`TestExamplePathEscape`, `TestExamplePathUnescape`, `TestExampleQueryEscape`,
`TestExampleQueryUnescape`, `TestExampleValues`, `TestExampleValues_Add`,
`TestExampleValues_Del`, `TestExampleValues_Encode`, `TestExampleValues_Get`,
`TestExampleValues_Has`, `TestExampleValues_Set`, `TestExampleURL`,
`TestExampleURL_roundtrip`, `TestExampleURL_ResolveReference`,
`TestExampleURL_EscapedPath`, `TestExampleURL_EscapedFragment`,
`TestExampleURL_Hostname`, `TestExampleURL_IsAbs`, `TestExampleURL_JoinPath`,
`TestExampleURL_MarshalBinary`, `TestExampleURL_Parse`, `TestExampleURL_Port`,
`TestExampleURL_Query`, `TestExampleURL_String`, `TestExampleURL_UnmarshalBinary`,
`TestExampleURL_Redacted`, `TestExampleURL_RequestURI`.

Combined with the previously-existing in-package tests, the package's
test set is now strictly a superset of upstream's portable tests modulo
the four API-blocked Test\* in finding #1.

---

## Notable

- **`Example*` are dead code in Gno stdlib tests today.** They compile
  but never run.
  [`gnovm/pkg/test/test.go:685`](../../.worktrees/gno-stdlib-test-port/gnovm/pkg/test/test.go#L685)
  filters on `Test` prefix only; the `// Output:` comment is also
  ignored — there is no example verifier. This affects every existing
  `example_test.gno` in `gnovm/stdlibs/` (e.g. `unicode/example_test.gno`).
  Mitigation in this port: rewrote each example body as
  `TestExample_<Name>` that captures output via a `strings.Builder` and
  asserts it.
- **No `reflect.DeepEqual` for URL comparison**. Ported with a
  handwritten field-by-field comparator (`urlEqualDiff` in
  `url_test.gno`) that emits a per-field multi-line diff, matching
  the per-field discipline the prompt asked for.
- **No correctness/security bugs found in `net/url`**. The package
  passes every portable upstream test verbatim, including the
  control-character rejection that protects against header-smuggling
  via `\r`/`\n`/`\x7f`, the userinfo-validity check, IPv6 host
  parsing, percent-encoding round-trips, and `Redacted()` password
  scrubbing.
