# Contribution: PR [#5835](https://github.com/gnolang/gno/pull/5835) — `realm_only_gate` rule
Event: COMMENT

## Body
Ran the ten rules against open realm PR [#5976](https://github.com/gnolang/gno/pull/5976). None fire on its access gate. `reputation.gno:17` writes `if caller.IsUserCall() { panic("realms only") }`. `IsUserCall()` is `pkgPath == ""`, which is false inside the ephemeral `maketx run` realm, so a user script passes the gate. `payment_user_call` is the only rule that reads `IsUserCall()`, and it fires only when the same function also calls `OriginSend()`; 5976 never does.

A `realm_only_gate` rule closes the gap. It flags `if x.IsUserCall()` used to reject, and leaves the require-direction `if !x.IsUserCall()` alone. The fixed fixture switches to `IsUser()`, which covers a direct call and the run realm.

Verified on 96cce07a2: the fixtures pass RunRule (vulnerable 1 hit at `reputation.gno:9`, fixed 0), both compile under `gno` in the WithGNO contract, and dropping 5976's realm file into the rule reports `reputation.gno:17`.

Want it as a PR against `codex/audit-guide-examples`?

<details><summary>rule — <code>internal/auditpattern/run.go</code></summary>

```go
// in RunRule, after the payment_user_call case:
	case "realm_only_gate":
		return realmOnlyGateHits(dir)

// realmOnlyGateHits flags a "realms only" gate written as `if x.IsUserCall() {
// panic }`. IsUserCall() is pkgPath == "", true only for a direct maketx call.
// The ephemeral maketx run realm has a non-empty pkgPath, so IsUserCall() is
// false there and a user script passes the gate. `if !x.IsUserCall()` requires
// a direct user call and is correct, so only the negation-free form is flagged.
// IsUser() covers both entry points.
func realmOnlyGateHits(dir string) ([]Hit, error) {
	files, err := gnoFiles(dir)
	if err != nil {
		return nil, err
	}

	var hits []Hit
	for _, file := range files {
		src, err := loadGnoSource(file)
		if err != nil {
			return nil, err
		}
		for i, line := range src.code {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "if ") {
				continue
			}
			if !strings.Contains(trimmed, ".IsUserCall()") {
				continue
			}
			if strings.Contains(trimmed, "!") {
				continue
			}
			hits = append(hits, src.hit(dir, file, i))
		}
	}
	return hits, nil
}
```
</details>

<details><summary>expected — <code>expected/realm-only-gate.yaml</code></summary>

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
</details>

<details><summary>fixtures — <code>fixtures/realm-only-gate/</code></summary>

`vulnerable/reputation.gno`:

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

`fixed/reputation.gno`:

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

`vulnerable/gnomod.toml` and `fixed/gnomod.toml`:

```toml
module = "gno.land/r/demo/realmonlygate"
gno = "0.9"
```
</details>

<details><summary>test — <code>internal/auditpattern/run_test.go</code></summary>

```go
func TestRealmOnlyGateRule(t *testing.T) {
	assertRuleCounts(t, "realm_only_gate", "realm-only-gate", 1, 0)
}
```
</details>
