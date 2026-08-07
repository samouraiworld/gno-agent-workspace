# Review: PR [#6028](https://github.com/gnolang/gno/pull/6028)
Event: COMMENT

## Body
The copy-replay path is closed, verified on 37cd55419 on a live node: two tokens minted from a foreign realm in two separate transactions came back as `gno.land/r/idprobe.AAA.gno.land/r/demo/defi/grc20reg:0000001` and `gno.land/r/idprobe.BBB.gno.land/r/demo/defi/grc20reg:0000002`, so the counter advances from another realm's frame and survives the transaction boundary.

The description still specifies the previous design, `gno.land/p/onbloc/identifier` with a sha256 plus cford32 code and `slug` as an alias key, none of which is in the diff, and a squash merge makes it the commit message.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6028-registry-owned-id-generator/2-37cd55419/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## examples/gno.land/p/demo/tokens/grc20/token.gno:15-16 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L15-L16) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L15)
The example id here is not the id [line 59](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) builds. For that exact realm the branch's own [`grc20_registry_emit.txtar`](https://github.com/gnolang/gno/blob/37cd55419/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L19) pins `gno.land/r/demo/defi/foo20.FOO.gno.land/r/demo/defi/grc20reg:0000001`.

## examples/gno.land/p/demo/tokens/grc20/token.gno:32 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L32) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L32)
Master gained [`newtoken_event_filetest.gno`](https://github.com/gnolang/gno/blob/18018c6a3/examples/gno.land/p/demo/tokens/grc20/filetests/newtoken_event_filetest.gno#L20-L21) after this branch's base, and both its calls pass the `seqid.ID` this signature deletes, so the next merge has to decide what a filetest whose premise is a caller reusing an id becomes. [`96c3cee24`](https://github.com/gnolang/gno/commit/96c3cee24) landed on `grc20reg.gno`, `grc20reg_test.gno` and the same two coin values in `storage_deposit_price_change.txtar` this branch retunes, so those numbers need re-deriving from both rather than picking one side.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:19 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L19) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L19)
One counter serves every realm, so `foo20.FOO.…:0000001` means the first token the registry ever issued rather than `foo20`'s first, and the same realm deployed on two chains gets two different ids. [`grc20factory_test.gno`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20factory/grc20factory_test.gno#L41-L44) already gave up asserting exact ids for that reason. Nothing off chain can key on a token id without also pinning the chain, and the change does not say so.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:32-34 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32-L34) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32)
Every realm gets an unconditional call into grc20reg's own counter, so a realm that builds no token and registers nothing advances `idSeq` and bills the storage diff to grc20reg. The counter is also the single thing keeping two registered tokens from sharing an identity, and grc20reg can neither meter nor revoke use of it. Issuing codes through a crossing function that returns the code would keep the call out of a caller's hands.

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
	g := grc20reg.IdentifierGenerator(cross(cur))
	for i := 0; i < n; i++ {
		g.NextID()
	}
}
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/leak_probe' -v -timeout 900s
rm gno.land/pkg/integration/testdata/leak_probe.txtar
```

```
OK!
GAS USED:   10503925
STORAGE DELTA:  10 bytes
EVENTS:     [{"bytes_delta":10,"fee_delta":{"denom":"ugnot","amount":1000},"pkg_path":"gno.land/r/demo/defi/grc20reg"}]
--- PASS: TestTestdata/leak_probe (2.36s)
```
</details>

## gno.land/pkg/gnoland/node_initial_height_test.go:26 [gh](https://github.com/gnolang/gno/blob/37cd55419/gno.land/pkg/gnoland/node_initial_height_test.go#L26) · [↗](../../../../../.worktrees/gno-review-6028/gno.land/pkg/gnoland/node_initial_height_test.go#L26)
This flake is master's own, failing at [`1bf8b2826`](https://github.com/gnolang/gno/commit/1bf8b2826) and [`fe2a4b8e9`](https://github.com/gnolang/gno/commit/fe2a4b8e9), and the test boots from `DefaultGenState` with no examples loaded, so no `examples/` diff reaches it. Raising the height does not fix it either: [the read](https://github.com/gnolang/gno/blob/37cd55419/gno.land/pkg/gnoland/node_initial_height_test.go#L77-L79) races block production, because `Ready()` returns [`firstBlockSignal`](https://github.com/gnolang/gno/blob/37cd55419/tm2/pkg/bft/node/node.go#L758-L760) and the node keeps committing after it. Dropping it from the branch keeps a grc20 squash commit off an unrelated node test in `git blame`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6028 -R gnolang/gno
# 1. master's 100 passes here, ten runs.
sed -i 's/int64(101)/int64(100)/' gno.land/pkg/gnoland/node_initial_height_test.go
go test ./gno.land/pkg/gnoland/ -run TestNodeBootWithInitialHeight -count=10 -timeout 900s
# 2. the branch's 101 fails once the read stops winning the race.
git checkout HEAD -- gno.land/pkg/gnoland/node_initial_height_test.go
sed -i 's/\theight := n.BlockStore/\ttime.Sleep(3 * time.Second)\n\theight := n.BlockStore/' gno.land/pkg/gnoland/node_initial_height_test.go
go test ./gno.land/pkg/gnoland/ -run TestNodeBootWithInitialHeight -count=1 -timeout 600s
git checkout HEAD -- gno.land/pkg/gnoland/node_initial_height_test.go
```

```
# 1.
ok  	github.com/gnolang/gno/gno.land/pkg/gnoland	2.584s
# 2.
    	Error:      	Not equal:
    	            	expected: 101
    	            	actual  : 3002
    	Messages:   	first committed block should be at InitialHeight (101), got 3002
--- FAIL: TestNodeBootWithInitialHeight (4.10s)
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:73-89 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L73-L89) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L73)
Missing test: nothing shows the registry counter surviving a transaction boundary, which is the one thing making two registry-issued ids differ. This test and every other in the file run inside one transaction, and the only multi-transaction coverage is [`grc20_registry_emit.txtar`](https://github.com/gnolang/gno/blob/37cd55419/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L19), whose single token is minted at genesis. The write is cross-realm and lands on grc20reg through the declaring-realm borrow, which a VM change can alter without any unit test noticing.

<details><summary>test cases</summary>

```
loadpkg gno.land/r/demo/defi/grc20reg
loadpkg gno.land/r/idprobe $WORK

gnoland start

gnokey maketx call -pkgpath gno.land/r/idprobe -func Mint -args 'AAA' -gas-fee 1000000ugnot -gas-wanted 40000000 -chainid=tendermint_test test1
stdout OK!
stdout '"key":"id","value":"gno.land/r/idprobe.AAA.gno.land/r/demo/defi/grc20reg:0000001"'

gnokey maketx call -pkgpath gno.land/r/idprobe -func Mint -args 'BBB' -gas-fee 1000000ugnot -gas-wanted 40000000 -chainid=tendermint_test test1
stdout OK!
stdout '"key":"id","value":"gno.land/r/idprobe.BBB.gno.land/r/demo/defi/grc20reg:0000002"'

-- gnomod.toml --
module = "gno.land/r/idprobe"
gno = "0.9"

-- idprobe.gno --
package idprobe

import (
	"chain"

	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

func Mint(cur realm, symbol string) {
	tok, _ := grc20.NewToken("Probe", symbol, 4, grc20reg.IdentifierGenerator(cross(cur)), cur)
	chain.Emit("probe_id", "id", tok.ID())
}
```
</details>

## examples/gno.land/p/demo/tokens/grc20/idgenerator.gno:54-56 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L54-L56) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L54)
Nit: banning `.` and `/` does not keep the id unambiguous for downstream parsers, because [line 59](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) then concatenates a package path carrying both. [`fqname.Parse`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/nt/fqname/v0/fqname.gno#L17-L43) splits `gno.land/r/demo/defi/foo20.FOO.0000000` into a path and `FOO.0000000`, and returns the new form whole with an empty name. The ban is still load-bearing on `:` and on the code's own shape, which is what the comment should say.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:36-37 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L36-L37) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L36)
Nit: `nextIDSeq` is a package-level function reading a package-level var, not a closure, and [line 19](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L19) calls it one too. What puts the write on grc20reg is where it was declared, which the rest of the same sentence already says.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:43 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43)
Nit: this and [the comment above the prefix check](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L60-L61) both write the id as `rlmPath.symbol.<id>`, leaving out the issuer segment that is the reason [the check below](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L67-L69) exists.

## SKIP examples/gno.land/p/demo/tokens/grc20/token.gno:27 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L27) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L27)
Already raised: https://github.com/gnolang/gno/pull/6028#pullrequestreview-3583044847
Suggestion: the only way to get a token grc20reg will accept is to import it and fetch its generator, so a `/p/` standard's useful contract runs through one named realm. The other half of the same point is answered by moving `IDGenerator` into `grc20` itself.

## examples/gno.land/p/demo/tokens/grc20/token.gno:59 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L59)
Suggestion: a registered token's id grows from 38 characters to 68, and the added `gno.land/r/demo/defi/grc20reg:` is the same for every registered token because [`Register`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L67-L69) rejects anything else. It carries one bit for that population, registry-issued against self-issued, paid on every `Transfer`, `Approval`, `Mint` and `Burn` forever. Master's newly merged [`NewToken` event](https://github.com/gnolang/gno/blob/18018c6a3/examples/gno.land/p/demo/tokens/grc20/token.gno#L67-L73) makes announcing the issuer once at construction the cheaper alternative.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:58 [gh](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L58) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L58)
Suggestion: `cur.Previous().PkgPath()` is read with no `cur.IsCurrent()` before it, while [the new check](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L67-L69) reads `cur.PkgPath()` and rests on the same frame being genuine. This matches master and no path reaches it, since the preprocessor admits only `cur` or `cross(rlm)` there and `cross()` validates `IsCurrent` itself.
