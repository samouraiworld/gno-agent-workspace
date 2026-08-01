# Review: PR [#6028](https://github.com/gnolang/gno/pull/6028)
Event: COMMENT

## Body
The guarantees this design rests on live in comments rather than in code, and two of those comments describe behaviour the code does not have.

[PR 6027](https://github.com/gnolang/gno/pull/6027) rewrites the same `Register`, [`grc20reg_test.gno`](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno), [`token.gno`](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/p/demo/tokens/grc20/token.gno) and four token realms, and keys by `realm.slug` too. It also requires a non-empty slug, which the four realms here rely on being allowed to leave empty. Which of the two should land first?

Verified on 37182a315: two [`grc20factory.New`](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L38) calls at different heights on a live node minted through grc20reg's generator and got distinct ids, so the cross-realm write path holds outside genesis.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6028-registry-owned-id-generator/1-37182a315/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## examples/gno.land/p/onbloc/identifier/identifier.gno:1-6 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/onbloc/identifier/identifier.gno#L1-L6)
Realm-scoped uniqueness holds only while a realm keeps exactly one generator, and nothing checks that. [`newTestToken`](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L23-L29) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L23-L29) builds a fresh generator per call, so two tokens it creates in one block share a `Token.ID()`. Registered tokens escape only because grc20reg happens to build one generator.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6028 -R gnolang/gno
cat > examples/gno.land/p/demo/tokens/grc20/dup_probe_test.gno <<'EOF'
package grc20

import "testing"

func TestDupProbe(cur realm, t *testing.T) {
	a, _ := newTestToken("Dummy", "DUMMY", 4, cur)
	b, _ := newTestToken("Other", "DUMMY", 4, cur)
	if a.ID() == b.ID() {
		t.Errorf("two tokens share an id: " + a.ID())
	}
}
EOF
(cd examples && go run ../gnovm/cmd/gno test -v -run TestDupProbe ./gno.land/p/demo/tokens/grc20)
rm examples/gno.land/p/demo/tokens/grc20/dup_probe_test.gno
```

```
=== RUN   TestDupProbe
two tokens share an id: gno.land/p/demo/tokens/grc20.DUMMY.y2dqkvttbgqtp
--- FAIL: TestDupProbe (0.00s)
FAIL    ./gno.land/p/demo/tokens/grc20 	5.47s
FAIL: 0 build errors, 1 test errors
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:27-29 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L27-L29)
Returning the `*Generator` hands every realm a live write handle on grc20reg's own state: [`NextID`](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/p/onbloc/identifier/identifier.gno#L55-L72) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/onbloc/identifier/identifier.gno#L55-L72) writes its receiver, and under [borrow rule #2](https://github.com/gnolang/gno/blob/37182a315/gnovm/pkg/gnolang/machine.go#L2398-L2400) · [↗](../../../../../.worktrees/gno-review-6028/gnovm/pkg/gnolang/machine.go#L2398-L2400) that write lands on grc20reg. A realm that creates no token and registers nothing advances the sequence and bills the storage diff to grc20reg. This is the shape [§8 of the contract review guide](https://github.com/gnolang/gno/blob/37182a315/docs/resources/gno-ai-contract-review.md?plain=1#L90-L113) · [↗](../../../../../.worktrees/gno-review-6028/docs/resources/gno-ai-contract-review.md#L90-L113) names.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6028 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/leak_probe.txtar <<'EOF'
loadpkg gno.land/r/demo/defi/grc20reg
loadpkg gno.land/r/probe $WORK

gnoland start

gnokey maketx call -pkgpath gno.land/r/probe -func Grief -args '50' -gas-fee 1000000ugnot -gas-wanted 40000000 -chainid=tendermint_test test1
stdout OK!

-- gnomod.toml --
module = "gno.land/r/probe"
gno = "0.9"

-- probe.gno --
package probe

import "gno.land/r/demo/defi/grc20reg"

// Grief holds no token and registers nothing.
func Grief(cur realm, n int) {
	g := grc20reg.IdentifierGenerator()
	for i := 0; i < n; i++ {
		g.NextID()
	}
}
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/leak_probe' -v -timeout 900s
rm gno.land/pkg/integration/testdata/leak_probe.txtar
```

```
STORAGE DELTA:  10 bytes
EVENTS:     [{"bytes_delta":10,"fee_delta":{"denom":"ugnot","amount":1000},"pkg_path":"gno.land/r/demo/defi/grc20reg"}]
--- PASS: TestTestdata/leak_probe (13.59s)
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:21-23 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L21-L23)
One shared counter makes a token's id depend on how many tokens other realms minted first, so the same realm deployed on two chains gets two different ids. Adding one `loadpkg` line ahead of foo20 in [`grc20_registry_emit.txtar`](https://github.com/gnolang/gno/blob/37182a315/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L19) · [↗](../../../../../.worktrees/gno-review-6028/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L19) moves foo20's id from `5fyzrxhg8tsh2` to `wctv536rq4cv2`. Nothing in the change says the id is chain-specific, so anything off-chain keying on one must pin the chain too.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6028 -R gnolang/gno
sed -i 's|^loadpkg gno.land/r/demo/defi/foo20$|loadpkg gno.land/r/tests/vm/test20\nloadpkg gno.land/r/demo/defi/foo20|' \
  gno.land/pkg/integration/testdata/grc20_registry_emit.txtar
go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20_registry_emit' -v -timeout 900s
git checkout HEAD -- gno.land/pkg/integration/testdata/grc20_registry_emit.txtar
```

```
EVENTS:     [{"type":"Transfer","attrs":[{"key":"token","value":"gno.land/r/demo/defi/foo20.FOO.wctv536rq4cv2"},...
FAIL: testdata/grc20_registry_emit.txtar:19: no match for `EVENTS: ... "value":"gno.land/r/demo/defi/foo20.FOO.5fyzrxhg8tsh2" ...`
--- FAIL: TestTestdata/grc20_registry_emit (20.90s)
```
</details>

## examples/gno.land/p/demo/tokens/grc20/token.gno:22-25 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L22-L25)
Two claims in this doc comment are wrong. [The body](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/p/demo/tokens/grc20/token.gno#L32-L63) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L32-L63) never compares `gen.PackagePath()` to `rlm.PkgPath()`, and [foo20](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/demo/defi/foo20/foo20.gno#L22) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/foo20/foo20.gno#L22) passes grc20reg's generator from its own frame on purpose. [Line 31](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/p/demo/tokens/grc20/token.gno#L31) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L31) calls the registry key `Token.ID()`, which [`grc20reg_test.gno:19`](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L19) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L19) asserts is a different value.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:63 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L63)
Registry keys move from `realm.symbol` to `realm.slug` with no read path for the old form. [`treasury.gno:26-33`](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/gov/dao/v3/treasury/treasury.gno#L26-L33) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/gov/dao/v3/treasury/treasury.gno#L26-L33) skips any key `Get` cannot resolve, so a key stored before this change makes the token disappear from the GovDAO treasury with no panic and no event.

## examples/gno.land/p/onbloc/identifier/identifier_test.gno:45-56 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/onbloc/identifier/identifier_test.gno#L45-L56)
Missing test: nothing advances the chain, so the sequence reset that keeps ids apart across blocks is never exercised. This test holds the height constant by design, so [the reset branch](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/p/onbloc/identifier/identifier.gno#L59-L62) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/onbloc/identifier/identifier.gno#L59-L62) fires once per generator and its purpose goes unasserted.

<details><summary>test cases</summary>

```go
func TestNextIDAcrossBlocks(cur realm, t *testing.T) {
	g := newTestGenerator(0, cur)

	seen := map[string]struct{}{}
	for block := 0; block < 3; block++ {
		for i := 0; i < 3; i++ {
			id := g.NextID()
			if _, dup := seen[id]; dup {
				t.Errorf("NextID repeated across blocks: " + id)
			}
			seen[id] = struct{}{}
		}
		// Advance the chain so the next NextID takes the sequence-reset branch.
		testing.SkipHeights(1)
	}

	if len(seen) != 9 {
		t.Errorf("expected 9 distinct ids across 3 blocks")
	}
}
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:69-79 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L69-L79)
Missing test: no test registers successfully with an empty slug, which is what [foo20](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/demo/defi/foo20/foo20.gno#L25) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/foo20/foo20.gno#L25), [wugnot](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L27) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L27) and [test20](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/tests/vm/test20/test20.gno#L23) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/tests/vm/test20/test20.gno#L23) all do. This test and [the next one](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L81-L90) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L81-L90) both pass `""` and both abort before the key is built, so the bare realm path they land under is pinned nowhere.

<details><summary>test cases</summary>

```go
func TestRegisterEmptySlug(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/grc20reg_empty_slug"))
	token, _ := grc20.NewToken("Empty", "EMT", 4, IdentifierGenerator(), cur)

	key := Register(cross(cur), token, "")
	urequire.Equal(t, "gno.land/r/demo/grc20reg_empty_slug", key)
	urequire.Equal(t, token.ID(), Get(key).ID())

	// The bare realm path is a single slot: a second empty-slug registration
	// from the same realm is rejected.
	other, _ := grc20.NewToken("Other", "OTH", 4, IdentifierGenerator(), cur)
	urequire.AbortsContains(t, cur, "token already registered", func() {
		Register(cross(cur), other, "")
	})
}
```
</details>

## examples/gno.land/p/demo/tokens/grc20/token.gno:49-51 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L49-L51)
Missing test: `ErrNilGenerator` has no test. [`TestNewTokenValidation`](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L59-L108) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L59-L108) covers name, symbol and decimals but not this. A realm that declares `var gen *identifier.Generator` and forgets to construct it hands in exactly this value.

<details><summary>test cases</summary>

```go
func TestNewTokenRejectsNilGenerator(cur realm, t *testing.T) {
	urequire.AbortsContains(t, cur, ErrNilGenerator.Error(), func() {
		func(cur realm) {
			NewToken("Nil", "NIL", 6, nil, cur)
		}(cross(cur))
	})
}
```
</details>

## examples/gno.land/r/gov/dao/v3/treasury/test/treasury_test.gno:89-90 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/gov/dao/v3/treasury/test/treasury_test.gno#L89-L90)
Nit: the returned key is the slug alias, not `Token.ID()`. [`grc20reg_test.gno:19`](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L19) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L19) asserts the two differ.

## examples/gno.land/p/onbloc/identifier/identifier.gno:25 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/onbloc/identifier/identifier.gno#L25)
Nit: the reset fires only when the height increases, not whenever it changes. [The branch](https://github.com/gnolang/gno/blob/37182a315/examples/gno.land/p/onbloc/identifier/identifier.gno#L59-L62) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/onbloc/identifier/identifier.gno#L59-L62) tests `currentHeight > g.latestHeight`, and that asymmetry is what keeps ids from repeating.

## examples/gno.land/p/onbloc/identifier/identifier.gno:18-20 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/onbloc/identifier/identifier.gno#L18-L20)
Nit: grc20reg's generator issues codes for every token realm on the chain, so the collision population is chain-wide, not within-realm.

## examples/gno.land/p/demo/tokens/grc20/token.gno:32 [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L32)
Suggestion: `Register` accepts only grc20reg's generator, so a registrable token now needs a grc20reg import, which is why [tokenhub's how-to-register snippet](https://github.com/gnolang/gno/blob/37182a315/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/render.gno#L56-L70) · [↗](../../../../../.worktrees/gno-review-6028/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/render.gno#L56-L70) now tells users to import a registry to build a token. The `grc20` package doc should say the registry is on the critical path.
