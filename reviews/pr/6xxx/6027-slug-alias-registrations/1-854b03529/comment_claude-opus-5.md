# Review: PR [#6027](https://github.com/gnolang/gno/pull/6027)
Event: COMMENT

## Body
PR [#6028](https://github.com/gnolang/gno/pull/6028) rewrites this same [`Register`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L30) body and nine of these twelve files, keys on `rlmPath.slug` too, and adds a registry-owned id generator that `Register` verifies. That check is what keeps two byte-identical `Token.ID()` values out of the registry once the symbol leaves the key, so this PR needs it or a duplicate-id guard of its own before the rekey lands. Which order do you want?

Verified on 854b03529: one token registered under two slugs stays one object across a transaction boundary, so a mint through one alias shows in the balance read back through the other.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6027-slug-alias-registrations/1-854b03529/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:51 [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51)
Two tokens [built from one seqid](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/p/demo/tokens/grc20/token.gno#L53) carry a byte-identical [`Token.ID()`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/p/demo/tokens/grc20/token.gno#L114-L116) and now both register, because this guard compares slugs only; on master the key carried the symbol and the second call aborted. `Transfer`, `Mint`, `Burn` and `Approval` [carry that id alone](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/p/demo/tokens/grc20/token.gno#L210-L212), so a consumer mapping an event back to a registry entry finds two.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6027 -R gnolang/gno

cat > examples/gno.land/r/demo/defi/grc20reg/filetests/duplicate_token_id_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/grc20reg_dupid

package grc20reg_dupid

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

func main(cur realm) {
	first, _ := grc20.NewToken("First", "DUP", 6, 0, cur)
	second, _ := grc20.NewToken("Second", "DUP", 6, 0, cur)

	if first.ID() != second.ID() {
		panic("precondition: both tokens must share one Token.ID()")
	}

	a := grc20reg.Register(cross(cur), first, "a")
	b := grc20reg.Register(cross(cur), second, "b")

	println("both registered under one token id:",
		grc20reg.MustGet(a).ID() == grc20reg.MustGet(b).ID(),
		grc20reg.MustGet(a).GetName(), grc20reg.MustGet(b).GetName())
}

// Error:
// grc20reg: token already registered
EOF

cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/demo/defi/grc20reg/; cd ..
rm examples/gno.land/r/demo/defi/grc20reg/filetests/duplicate_token_id_filetest.gno
```

```
--- FAIL: ./gno.land/r/demo/defi/grc20reg/duplicate_token_id_filetest.gno (elapsed: 0.04s, gas: 986049, storage: gno.land/r/demo/defi/grc20reg:+3414b gno.land/r/demo/grc20reg_dupid:+5326b)
unexpected output:
both registered under one token id: true First Second

./gno.land/r/demo/defi/grc20reg/duplicate_token_id_filetest.gno failed
FAIL    ./gno.land/r/demo/defi/grc20reg
```

The same file passes on the merge base, where the second registration aborts with `grc20reg: token already registered`.
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:73-74 [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L73)
Missing test: nothing pins that both aliases resolve to one object. Comparing [`Token.ID()`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/p/demo/tokens/grc20/token.gno#L114-L116) strings passes just as well for two separate tokens, each with its own [`PrivateLedger`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/p/demo/tokens/grc20/types.gno#L103-L112).

<details><summary>test cases</summary>

```gno
func TestAliasesShareOneObject(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/grc20reg_alias_identity"))
	token, ledger := grc20.NewToken("Aliased", "ALI", 4, 0, cur)
	ledger.Mint(cur.Address(), 42)

	first := Register(cross(cur), token, "first")
	second := Register(cross(cur), token, "second")

	urequire.True(t, Get(first) == Get(second), "both aliases must resolve to one token object")
	urequire.Equal(t, int64(42), Get(second).BalanceOf(cur.Address()))
}
```

Passes at 854b03529.
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:61 [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L61)
Nit: adding `token_id` here leaves [`token_path`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L57) holding the alias, not a prefix of [`Token.ID()`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/p/demo/tokens/grc20/token.gno#L114-L116). An indexer relying on that prefix breaks silently, and the field name still reads as a token path.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:93 [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L93)
Nit: every info link here 404s. It is built as `/r/demo/grc20reg:<key>`, but the realm's module path is [`gno.land/r/demo/defi/grc20reg`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/gnomod.toml#L1), so the `defi` segment is missing, here and in the [golden](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L22) this PR updates.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:36 [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L36)
Suggestion: this guard cannot fire. [`cross` runs the same predicate](https://github.com/gnolang/gno/blob/854b03529/gnovm/pkg/gnolang/uverse.go#L1836-L1838) and aborts with `cross: rlm is not the current cur` before [`Register`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L30) starts, and no same-realm caller exists to skip it. [`gno-ai-contract-review.md` §1](https://github.com/gnolang/gno/blob/854b03529/docs/resources/gno-ai-contract-review.md?plain=1#L12) still prescribes the guard before every `cur.Previous()` read, so keep it as documented defense in depth or drop it as dead code, and close the thread either way.
