# Review: [#6072](https://github.com/gnolang/gno/pull/6072)
Posted: https://github.com/gnolang/gno/pull/6072#pullrequestreview-5027346220
Event: COMMENT

## Body
[Automatic AI review]

Second automated pass, covering the test-coverage delta, EIP-721 conformance and the recurring-bug-class checklist. No design judgement on the Token/PrivateLedger/Teller split and no merge verdict.

## examples/gno.land/p/demo/tokens/grc721/token.gno:139 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L139) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L139) [posted](https://github.com/gnolang/gno/pull/6072#discussion_r3860157630)
`RegisterExtension` screens duplicates with `e == ext` one line before the kind check, and comparing two interface values whose dynamic type is uncomparable is a runtime fault, so a second instance of a slice-holding extension type aborts with `comparing uncomparable type` instead of the kind collision the doc above promises.

<details><summary>repro</summary>

The kind check on the next line already rejects every duplicate kind, and the fault is recoverable, so a gno `recover()` catches it. No in-tree extension is a value type, and the series' own `Enumerable` and its `Ledger` are pointers, so nothing in 6073 through 6075 reaches it.

Ready-to-add test, red at head:

```go
type journalExtension struct{ seen []TokenID }

func (e journalExtension) ExtensionKind() string                    { return "journal" }
func (e journalExtension) OnMint(to address, tid TokenID)           {}
func (e journalExtension) OnTransfer(from, to address, tid TokenID) {}
func (e journalExtension) OnBurn(tid TokenID)                       {}

func TestRegisterExtensionRejectsDuplicateKindOfUncomparableType(cur realm, t *testing.T) {
	_, led := newTestToken("Foo", "FOO", 0, cur)
	led.RegisterExtension(journalExtension{seen: []TokenID{"1"}})
	uassert.PanicsContains(t, cur, "grc721: extension kind already registered: journal", func() {
		led.RegisterExtension(journalExtension{seen: []TokenID{"2"}})
	})
}
```

```
Actual panic value: "runtime error: comparing uncomparable type gno.land/p/demo/tokens/grc721.journalExtension"
--- FAIL: TestRegisterExtensionRejectsDuplicateKindOfUncomparableType
```
</details>

## examples/gno.land/p/demo/tokens/grc721/token.gno:179-181 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L179-L181) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L179) [posted](https://github.com/gnolang/gno/pull/6072#discussion_r3860157633)
The hooks run after the ledger write and the emit, so the veto the doc describes is the transaction aborting rather than the movement reverting, and a caller that recovers keeps the mint, the emitted `Transfer` and a nil error from `Mint`.

<details><summary>repro</summary>

`types.gno:40-41` says a hook panic "aborts that operation, an implicit veto over transfers" and the `RegisterExtension` doc says "a panic reverts the movement". Both hold only while nobody recovers. Extensions ahead of the vetoing one have already recorded the movement; those after it never run.

```go
func (e *vetoExt) OnMint(to address, tid TokenID) { panic("veto: mint refused") }

func TestVetoDoesNotRevertWhenCallerRecovers(cur realm, t *testing.T) {
	tok, led := newTestToken("Foo", "FOO", 0, cur)
	led.RegisterExtension(&vetoExt{kind: "veto"})
	func() {
		defer func() { t.Log("caller recovered:", recover()) }()
		_ = led.Mint(alice, "1")
	}()
	owner, err := tok.OwnerOf("1")
	t.Log("totalSupply:", tok.TotalSupply(), "ownerOf(1):", owner, "err:", err)
}
```

```
caller recovered: veto: mint refused
totalSupply: 1 ownerOf(1): g1v9kxjcm9ta047h6lta047h6lta047h6lzd40gh err: <nil>
```
</details>
