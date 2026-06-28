# Review: PR #5728
Event: COMMENT

## Body
`TokenURI` and `RoyaltyInfo` skip the `OwnerOf` gate that `TokenMetadata` applies, so a host that runs core `Burn` without the extension `Burn` leaves each read serving the prior owner's data. The gate is only a partial guard: it rejects while the tid is burned, but once the tid is re-minted `OwnerOf` succeeds again and even gated `TokenMetadata` returns the old owner's entry, so closing this fully needs core `Burn` to notify the extensions. The repro below confirms the leak on eac94f444.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5728 -R gnolang/gno

cat > examples/gno.land/p/demo/tokens/grc721/metadata/burn_leak_test.gno <<'EOF'
package metadata

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestHostForget_TokenURI_PersistsAcrossRemint(cur realm, t *testing.T) {
	// cross promotes the test's EOA-origin cur into a CodeRealm cur so
	// grc721.NewToken's IsCurrent + non-empty PkgPath checks pass.
	var tok *grc721.Token
	var coreLedger *grc721.PrivateLedger
	func(cur realm) {
		tok, coreLedger = grc721.NewToken(0, cur, "FOO", "FOO")
	}(cross(cur))
	mLedger := NewPrivateLedger(tok)
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")
	tid := grc721.TokenID("1")

	uassert.NoError(t, coreLedger.Mint(alice, tid))
	uassert.NoError(t, mLedger.SetTokenURI(tid, grc721.TokenURI("ipfs://alice-art")))
	uassert.NoError(t, coreLedger.Burn(tid)) // host forgot mLedger.Burn(tid)
	uassert.NoError(t, coreLedger.Mint(bob, tid))

	// bob now owns tid, but TokenURI still returns alice's URI.
	uri, err := mLedger.TokenURI(tid)
	uassert.NoError(t, err)
	uassert.Equal(t, "ipfs://alice-art", uri) // bob's re-minted tid serves alice's URI
}
EOF

(cd examples && go run ../gnovm/cmd/gno test -v -run 'TestHostForget' ./gno.land/p/demo/tokens/grc721/metadata)
rm examples/gno.land/p/demo/tokens/grc721/metadata/burn_leak_test.gno
```

```
=== RUN   TestHostForget_TokenURI_PersistsAcrossRemint
--- PASS: TestHostForget_TokenURI_PersistsAcrossRemint (0.00s)
ok      ./gno.land/p/demo/tokens/grc721/metadata
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5728-grc721-private-ledger-teller/2-eac94f444/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/p/demo/tokens/grc721/metadata/token.gno:37-44 [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L37)
`TokenURI` reads storage with no `OwnerOf` gate, unlike `TokenMetadata` right below it. After a host runs core `Burn` without `mLedger.Burn`, it serves the burned URI, and a re-mint of the same tid inherits it. The gate closes the burn window but not the re-mint case, so the durable fix is for core `Burn` to notify extensions.

<details><summary>repro</summary>

See the burn-and-remint repro in the review Body.
</details>

## examples/gno.land/p/demo/tokens/grc721/royalty/token.gno:49-58 [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L49)
`RoyaltyInfo` reads storage with no `OwnerOf` gate. If the host forgets `rLedger.Burn` after core `Burn`, an EIP-2981 marketplace querying a burned-and-reminted tid routes royalties to the previous owner's payee, with no way to detect it from outside. An `OwnerOf` gate closes the burn window, but the re-mint case needs core `Burn` to notify extensions.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5728 -R gnolang/gno

cat > examples/gno.land/p/demo/tokens/grc721/royalty/burn_leak_test.gno <<'EOF'
package royalty

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestHostForget_RoyaltyInfo_PersistsAcrossRemint(cur realm, t *testing.T) {
	// cross promotes the test's EOA-origin cur into a CodeRealm cur so
	// grc721.NewToken's IsCurrent + non-empty PkgPath checks pass.
	var tok *grc721.Token
	var coreLedger *grc721.PrivateLedger
	func(cur realm) {
		tok, coreLedger = grc721.NewToken(0, cur, "FOO", "FOO")
	}(cross(cur))
	rLedger := NewPrivateLedger(tok, 1000)
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")
	alicePayee := testutils.TestAddress("alicePayee")
	tid := grc721.TokenID("1")

	uassert.NoError(t, coreLedger.Mint(alice, tid))
	uassert.NoError(t, rLedger.SetTokenRoyalty(tid, RoyaltyInfo{PaymentAddress: alicePayee, Bps: 500}))
	uassert.NoError(t, coreLedger.Burn(tid)) // host forgot rLedger.Burn(tid)
	uassert.NoError(t, coreLedger.Mint(bob, tid))

	addr, amount, err := rLedger.RoyaltyInfo(tid, 10000)
	uassert.NoError(t, err)
	uassert.Equal(t, alicePayee.String(), addr.String()) // bob owns tid, alice still paid
	uassert.Equal(t, int64(500), amount)
}
EOF

(cd examples && go run ../gnovm/cmd/gno test -v -run 'TestHostForget' ./gno.land/p/demo/tokens/grc721/royalty)
rm examples/gno.land/p/demo/tokens/grc721/royalty/burn_leak_test.gno
```

```
=== RUN   TestHostForget_RoyaltyInfo_PersistsAcrossRemint
--- PASS: TestHostForget_RoyaltyInfo_PersistsAcrossRemint (0.00s)
ok      ./gno.land/p/demo/tokens/grc721/royalty
```
</details>

## examples/gno.land/p/demo/tokens/grc721/metadata/tellers.gno:9-70 [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/tellers.gno#L9)
`Teller` exposes only read methods, yet four factories all build the same read-only `fnTeller`, and each sets `accountFn`, which nothing reads. The `ReadonlyTeller` godoc promises write methods returning `ErrReadonly`, but `fnTeller` has none. Collapse to one read-only constructor and drop `accountFn`, or mark the field reserved and note the factories are aliases today.

## examples/gno.land/p/demo/tokens/grc721/royalty/tellers.gno:9-70 [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/royalty/tellers.gno#L9)
Same as the metadata extension: four factories produce identical read-only Tellers and `accountFn` is dead. Collapse to one constructor or document the aliasing.

## examples/gno.land/r/demo/foo721/foo721.gno:66-74 [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/demo/foo721/foo721.gno#L66)
This comment describes the pre-#5603 design: `foo *basicNFT`, concrete writer methods, and a Reader/Writer split in `igrc721.gno`. Both `basicNFT` and `igrc721.gno` are gone, and the wrappers now dispatch through `userTeller`. Replace it with the current `CallerTeller` doctrine or delete it.

## examples/gno.land/r/matijamarjanovic/tokenhub/render.gno:68 [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L68)
The rendered example calls `tokenhub.RegisterToken(myToken, "my_token")` with no `cross(cur)`, but `RegisterToken` takes `cur realm` first. Line 79's `RegisterMultiToken` has the same gap while line 74's `RegisterNFT(cross(cur), ...)` is correct, so a reader copying the snippet hits a compile error. Make the snippet compile as-is.

## examples/gno.land/p/demo/tokens/grc721/metadata/types.gno:79 [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L79)
`TokenURIUpdateEvent` emits the wire string `"TokenUriUpdate"`, but its symbol and the `TokenURI`/`SetTokenURI` API spell `URI`. Every other event in the package matches name to value. Align the casing.

## examples/gno.land/p/demo/tokens/grc721/metadata/token.gno:74-78 [↗](../../../../../.worktrees/gno-review-5728/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L74)
Neither the `TokenUriUpdate` nor `MetadataUpdate` emission has a test asserting its type, keys, or values. `grc721_emit.txtar` covers only core events through `foo721`, which never attaches the metadata extension. Add a unit test reading the events emitted by `SetTokenURI` and `SetTokenMetadata`.
