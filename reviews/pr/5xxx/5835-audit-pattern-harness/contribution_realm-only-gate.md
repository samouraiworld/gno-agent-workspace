# Contribution draft: PR [5835](https://github.com/gnolang/gno/pull/5835) — `realm_only_gate` rule

Not a review round. A proposed ninth rule plus fixture pair, written against head f380c15f7.
Unposted. Local patch lives in `.worktrees/gno-review-5835`.

## Body

Ran the eight rules against three realm PRs open right now: [5976](https://github.com/gnolang/gno/pull/5976), [5951](https://github.com/gnolang/gno/pull/5951), [5946](https://github.com/gnolang/gno/pull/5946). Only `current_guard` fired, 1 to 5 hits each. Nothing caught the one exploitable bug in the set.

That bug is in 5976. The realm gates on `if caller.IsUserCall() { panic }` to mean "realms only". `IsUserCall()` is `pkgPath == ""`, so it is false inside the ephemeral `<domain>/e/<addr>/run` realm that `gnokey maketx run` executes in. A user script passes the gate and gets realm-level authority. Reproduced end to end with a txtar that credits an arbitrary address in one transaction.

`payment_user_call` is the closest rule but cannot reach it: it keys on `OriginSend()` with no preceding `IsUserCall()`, and 5976 never calls `OriginSend`. The two directions are separate shapes. `payment_user_call` catches a gate that is too loose for taking payment. This one catches a gate that looks strict and is not.

Proposed rule below. It flags `IsUserCall()` used to reject rather than to require, which is the direction that misses `IsUserRun`. Passes a vulnerable/fixed pair, and flags `reputation.gno:17` in 5976:

```
- pattern hits: got `1`, want `0`
  - `reputation.gno:17` `if caller.IsUserCall() {`
```

Happy to open it as a PR against your branch, or leave it here if you would rather fold it in yourself.

## Rule

`internal/auditpattern/run.go`, dispatch arm plus one function:

```go
	case "realm_only_gate":
		return realmOnlyGateHits(dir)
```

```go
// realmOnlyGateHits flags a realm-only gate built on IsUserCall(). Rejecting
// callers for which IsUserCall() is true only rejects a direct maketx call:
// the ephemeral maketx run realm has a non-empty pkgPath, so IsUserCall() is
// false there and the script passes the gate. IsUser() covers both.
func realmOnlyGateHits(dir string) ([]Hit, error) {
	files, err := gnoFiles(dir)
	if err != nil {
		return nil, err
	}

	var hits []Hit
	for _, file := range files {
		data, err := readGnoSource(file)
		if err != nil {
			return nil, err
		}
		orig := strings.Split(string(data), "\n")
		for i, line := range codeLines(data) {
			if !strings.Contains(line, ".IsUserCall()") {
				continue
			}
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "if ") {
				continue
			}
			// "if !x.IsUserCall()" requires a direct user call and is correct.
			// "if x.IsUserCall()" rejects users and misses the run realm.
			if strings.Contains(trimmed, "!") {
				continue
			}
			hits = append(hits, newHit(dir, file, i+1, orig[i]))
		}
	}
	return hits, nil
}
```

## Fixtures

`fixtures/realm-only-gate/vulnerable/reputation.gno`:

```go
package reputation

var scores = map[string]int64{}

func AddPoints(cur realm, target string, points int64) {
	caller := cur.Previous()
	// Vulnerable: IsUserCall() is false inside the ephemeral maketx run realm,
	// so a user script passes this "realms only" gate.
	if caller.IsUserCall() {
		panic("realms only")
	}
	scores[caller.PkgPath()] += points
}
```

`fixtures/realm-only-gate/fixed/reputation.gno`:

```go
package reputation

var scores = map[string]int64{}

func AddPoints(cur realm, target string, points int64) {
	if !cur.IsCurrent() {
		panic("invalid realm")
	}
	caller := cur.Previous()
	// Fixed: IsUser() covers a direct call and the maketx run realm.
	if caller.IsUser() {
		panic("realms only")
	}
	scores[caller.PkgPath()] += points
}
```

Both dirs carry `gnomod.toml` with `module = "gno.land/r/demo/realmonlygate"` and `gno = "0.9"`.

`expected/realm-only-gate.yaml`:

```yaml
id: realm-only-gate
title: realm-only gate built on IsUserCall
rule: realm_only_gate
fixtures:
  - name: vulnerable
    path: ../fixtures/realm-only-gate/vulnerable
    want_gno_test: pass
    want_pattern_hits: 1
  - name: fixed
    path: ../fixtures/realm-only-gate/fixed
    want_gno_test: pass
    want_pattern_hits: 0
```

## Verification

- `go build ./...` clean, `gofmt` clean.
- `go run ./cmd/auditpattern -gno-bin <gno> ./expected/realm-only-gate.yaml` reports `status: PASS`, vulnerable 1 hit, fixed 0.
- Same rule against the 5976 worktree realm reports 1 hit at `reputation.gno:17`, the line carrying the live defect.
- Predicate semantics read from `gnovm/stdlibs/chain/runtime/frame.gno:78-107`: `IsUser()` is `IsUserCall() || IsUserRun()`, and `IsUserRun()` matches `<domain>/e/<addr>/run`. Run path built at `gno.land/pkg/sdk/vm/keeper.go:1018`.

## Notes

- Same false-positive profile as the other rules: a line scan, so `if x.IsUserCall()` guarding something other than a realms-only gate will hit. The README already frames hits as lines to inspect.
- Round 4 of the review at `3700f767f` is still unposted, and the head has since moved to `f380c15f7`. This draft is independent of it.
