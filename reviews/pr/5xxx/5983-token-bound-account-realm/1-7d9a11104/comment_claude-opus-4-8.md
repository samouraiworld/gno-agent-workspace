# Review: PR [#5983](https://github.com/gnolang/gno/pull/5983)
Posted: https://github.com/gnolang/gno/pull/5983#pullrequestreview-4771987027
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Repros run at 7d9a11104.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5983-token-bound-account-realm/1-7d9a11104/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/zeycan1/tba/tba.gno:15 [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L15) [posted](https://github.com/gnolang/gno/pull/5983#discussion_r3644377306)
A copy of this file deployed at another path keeps this constant, so [`VaultAddress`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L64-L66) and [`Balance`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L68-L71) still answer for the original realm's vault, while [`Withdraw` spends through `cur.Sub`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L83-L85), which derives from the deployed path. The copy reports a balance its owner cannot spend, and the coins stay spendable by whoever holds that token id in `r/zeycan1/tba`. Rejecting a mismatch in `Setup` ties the two derivations together.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5983 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/tba_copy_strands.txtar <<'EOF'
loadpkg gno.land/r/zeycan1/tba

gnoland start

# the same file, deployed at another path, with selfPath untouched
mkdir $WORK/copy
cp $GNOROOT/examples/gno.land/r/zeycan1/tba/tba.gno $WORK/copy/tba.gno
gnokey maketx addpkg -pkgdir $WORK/copy -pkgpath gno.land/r/$test1_user_addr/tba -gas-fee 1000000ugnot -gas-wanted 200000000 -broadcast -chainid=tendermint_test test1
gnokey maketx call -pkgpath gno.land/r/$test1_user_addr/tba -func Setup -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
gnokey maketx call -pkgpath gno.land/r/$test1_user_addr/tba -func Mint -args g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5 -args one -args img -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1

# the copy advertises the original realm's vault
gnokey query vm/qeval --data "gno.land/r/$test1_user_addr/tba.VaultAddress(\"1\")"
stdout 'g13aazlmjgmalp0r9q72jhv6f0t82vj0q76d8cez'

gnokey maketx send -to g13aazlmjgmalp0r9q72jhv6f0t82vj0q76d8cez -send 1000ugnot -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
gnokey query vm/qeval --data "gno.land/r/$test1_user_addr/tba.Balance(\"1\")"
stdout '1000'

! gnokey maketx call -pkgpath gno.land/r/$test1_user_addr/tba -func Withdraw -args 1 -args g10ddhfxv47lhyhpj426u57ckxwl0qtrk6c9rs5v -args 100 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
stderr 'insufficient coins error'   # IS:     the owner cannot spend what Balance reports
# stdout 'OK!'                      # SHOULD: the advertised vault is the one Withdraw spends from

-- copy/gnomod.toml --
module = "gno.land/r/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/tba"
gno = "0.9"
EOF
go test -v -run 'TestTestdata/tba_copy_strands$' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/tba_copy_strands.txtar
```

```
> gnokey query vm/qeval --data "gno.land/r/$test1_user_addr/tba.VaultAddress(\"1\")"
data: ("g13aazlmjgmalp0r9q72jhv6f0t82vj0q76d8cez" .uverse.address)
> gnokey query vm/qeval --data "gno.land/r/$test1_user_addr/tba.Balance(\"1\")"
data: (1000 int64)
> ! gnokey maketx call ... -func Withdraw -args 1 ... -args 100 ...
"gnokey" error: --= Error =--
Data: insufficient coins error
Stacktrace:
bankerSendCoins at chain/banker/banker.gno:native
chain/banker.banker.SendCoins at chain/banker/banker.gno:175
Withdraw at gno.land/r/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/tba/tba.gno:85
# …
--- PASS: TestTestdata/tba_copy_strands (2.09s)
```
</details>

## examples/gno.land/r/zeycan1/tba/tba.gno:30-32 [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L30-L32) [posted](https://github.com/gnolang/gno/pull/5983#discussion_r3644377317)
`admin` ends up as the chain's genesis key, not you: everything under `examples/` is [deployed by that key](https://github.com/gnolang/gno/blob/7d9a11104/gno.land/cmd/gnoland/start.go#L453) unless [`gnomod.toml` names a creator](https://github.com/gnolang/gno/blob/7d9a11104/gno.land/pkg/gnoland/genesis.go#L199-L207), and [`unsafe.PreviousRealm()` at package initialization](https://github.com/gnolang/gno/blob/7d9a11104/gnovm/stdlibs/internal/execctx/realm.go#L58-L69) resolves to the deploy signer. Only that key can call `Setup` or `Mint`. A literal admin address in the source, as in [r/gnoland/home](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/gnoland/home/home.gno#L16), also drops the `chain/runtime/unsafe` import that [the realm checklist](https://github.com/gnolang/gno/blob/7d9a11104/docs/resources/gno-ai-contract-review.md?plain=1#L140-L161) flags alongside `cur realm` parameters.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5983 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/tba_admin_is_deployer.txtar <<'EOF'
loadpkg gno.land/r/zeycan1/tba

adduserfrom alice 'success myself purchase tray reject demise scene little legend someone lunar hope media goat regular test area smart save flee surround attack rapid smoke'

gnoland start

# test1 signs the genesis deployment of everything under examples/
gnokey maketx call -pkgpath gno.land/r/zeycan1/tba -func Setup -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
stdout 'OK!'                          # the genesis key is admin

! gnokey maketx call -pkgpath gno.land/r/zeycan1/tba -func Mint -args g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0 -args one -args img -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test alice
stderr 'only admin can mint'          # nobody else is
EOF
go test -v -run 'TestTestdata/tba_admin_is_deployer$' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/tba_admin_is_deployer.txtar
```

```
> gnokey maketx call -pkgpath gno.land/r/zeycan1/tba -func Setup ... test1
OK!
GAS USED:   5702540
> ! gnokey maketx call -pkgpath gno.land/r/zeycan1/tba -func Mint ... alice
"gnokey" error: --= Error =--
Data: only admin can mint
Stacktrace:
panic: only admin can mint
Mint at gno.land/r/zeycan1/tba/tba.gno:48
# …
--- PASS: TestTestdata/tba_admin_is_deployer (2.42s)
```
</details>

## examples/gno.land/r/zeycan1/tba/tba.gno:96-99 [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L96-L99) [posted](https://github.com/gnolang/gno/pull/5983#discussion_r3644377318)
An unauthorized transfer [returns `ErrCallerIsNotOwnerOrApproved`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L214-L217) instead of aborting, so the transaction commits, reports `OK!`, and moves nothing. [`Setup`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L39-L41), [`Mint`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L47-L49) and [`Withdraw`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L79-L81) all panic on their guards, so a caller reading transaction status has no reason to treat this one differently.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5983 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/tba_transfer_silent.txtar <<'EOF'
loadpkg gno.land/r/zeycan1/tba

adduserfrom alice 'success myself purchase tray reject demise scene little legend someone lunar hope media goat regular test area smart save flee surround attack rapid smoke'
stdout 'g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0'

gnoland start

gnokey maketx call -pkgpath gno.land/r/zeycan1/tba -func Setup -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
gnokey maketx call -pkgpath gno.land/r/zeycan1/tba -func Mint -args g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0 -args one -args img -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1

# test1 owns neither token 1 nor an approval on it
gnokey maketx call -pkgpath gno.land/r/zeycan1/tba -func TransferFrom -args g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0 -args g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5 -args 1 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1

stdout 'OK!'                                          # IS:     the transaction succeeds
stdout 'caller is not token owner or approved'        # IS:     the rejection is only a return value
# ! stdout 'OK!'                                      # SHOULD: an unauthorized transfer aborts

gnokey query vm/qeval --data "gno.land/r/zeycan1/tba.OwnerOf(\"1\")"
stdout 'g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0'
EOF
go test -v -run 'TestTestdata/tba_transfer_silent$' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/tba_transfer_silent.txtar
```

```
> gnokey maketx call ... -func TransferFrom ... test1
(&(struct{("caller is not token owner or approved" string)} errors.errorString) *errors.errorString)

OK!
GAS USED:   5459063
HEIGHT:     5
> gnokey query vm/qeval --data "gno.land/r/zeycan1/tba.OwnerOf(\"1\")"
data: ("g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0" .uverse.address)
# …
--- PASS: TestTestdata/tba_transfer_silent (12.63s)
```
</details>

## examples/gno.land/r/zeycan1/tba/tba.gno:34-43 [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L34-L43) [posted](https://github.com/gnolang/gno/pull/5983#discussion_r3644377320)
Missing test: `gno test ./gno.land/r/zeycan1/tba` reports `[no test files]`, so nothing covers the admin gate, the owner gate, or vault control moving with the token. Writing them surfaces the [`init()` binding](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L30-L32) too: under `gno test` the origin caller is empty, so `admin` is an empty address no caller can match until a test assigns it.

<details><summary>test cases</summary>

Full suite: [`tba_test.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5983-token-bound-account-realm/1-7d9a11104/tests/tba_test.gno). Four of its six cases pass at this commit; these two fail.

```go
func TestWithdrawRejectsNonPositive(cur realm, t *testing.T) {
	tid := setup(cur, t)
	testing.IssueCoins(VaultAddress(tid), chain.NewCoins(chain.NewCoin("ugnot", 1000)))
	testing.SetRealm(testing.NewUserRealm(aliceAddr))

	// A zero-amount withdraw currently commits a no-op transaction.
	abort := revive(func() { Withdraw(cross(cur), tid, bobAddr, 0) })
	uassert.True(t, abort != nil, "zero amount was accepted")
	uassert.Equal(t, int64(1000), Balance(tid))
}

func TestTransferFromAbortsWhenUnauthorized(cur realm, t *testing.T) {
	tid := setup(cur, t)

	// bob owns neither the token nor an approval. The call must not
	// report success back to the caller.
	testing.SetRealm(testing.NewUserRealm(bobAddr))
	abort := revive(func() {
		TransferFrom(cross(cur), aliceAddr, bobAddr, tid)
	})
	uassert.True(t, abort != nil, "unauthorized transfer returned instead of aborting")

	owner, err := OwnerOf(tid)
	uassert.NoError(t, err)
	uassert.Equal(t, aliceAddr.String(), owner.String())
}

// setup makes the realm usable from a unit test. init() reads
// unsafe.PreviousRealm(), which is the empty address under `gno test`,
// so admin has to be assigned before any admin-gated call.
func setup(cur realm, t *testing.T) grc721.TokenID {
	t.Helper()
	admin = adminAddr
	testing.SetRealm(testing.NewUserRealm(adminAddr))
	if nft == nil {
		Setup(cross(cur))
	}
	return Mint(cross(cur), aliceAddr, "art", "ipfs://img")
}
```
</details>

## examples/gno.land/r/zeycan1/tba/tba.gno:45 [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L45) [posted](https://github.com/gnolang/gno/pull/5983#discussion_r3644377327)
Nit: none of the fourteen exported functions carries a doc comment, and the admin-only and owner-only contracts appear only in panic strings. A realm published as a reference pattern gets read from its signatures.

## examples/gno.land/r/zeycan1/tba/tba.gno:52 [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L52) [posted](https://github.com/gnolang/gno/pull/5983#discussion_r3644377331)
Nit: calling `Mint` before `Setup` aborts with `runtime error: nil pointer dereference` raised inside [`BasicNFT.exists`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L425-L427), which names grc721 rather than the step that was skipped. Only the admin can reach it.

## examples/gno.land/r/zeycan1/tba/tba.gno:83-85 [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L83-L85) [posted](https://github.com/gnolang/gno/pull/5983#discussion_r3644377337)
Nit: an amount of zero passes every check and commits a transaction that moves nothing for 5,063,313 gas. Negative amounts, overdrafts and malformed recipients are all caught inside [`banker.SendCoins`](https://github.com/gnolang/gno/blob/7d9a11104/gnovm/stdlibs/chain/banker/banker.gno#L166-L175), so zero is the only value that gets through.

## examples/gno.land/r/zeycan1/tba/tba.gno:131-136 [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L131-L136) [posted](https://github.com/gnolang/gno/pull/5983#discussion_r3644377343)
Nit: `path` is discarded, so a `vm/qrender` of `:token/1` returns the [collection header from `RenderHome`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L442-L448) with no vault address, balance, or owner. The per-token vault is the point of this realm and the page never shows it.

## examples/gno.land/r/zeycan1/tba/tba.gno:116-129 [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L116-L129) [posted](https://github.com/gnolang/gno/pull/5983#discussion_r3644377348)
Suggestion: [`Withdraw`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L73-L86) and [`TransferFrom`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L96-L99) are separate transactions, so a seller can empty the vault after a buyer reads `TokenInfo` and before the transfer lands. ERC-6551 has the same property, and it is the one thing a reader has to know before copying this pattern. A line on `Withdraw` saying the vault is drainable by the current owner up to the moment of transfer covers it.
