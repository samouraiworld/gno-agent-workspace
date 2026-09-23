# Candidate find-b1-4: docs/resources/gno-stdlibs.md:752 — `NewBanker` lists `BankerTypeReadonly` but panics on it

The diff rewrites the `NewBanker` parameter list header to
``- `bt` **BankerType** - type of Banker to get:`` and keeps the sub-bullet
``BankerTypeReadonly - read-only access to coin balances`` under it. The head
implementation rejects that value: `NewBanker` panics with
`use NewReadonlyBanker for BankerTypeReadonly` (gnovm/stdlibs/chain/banker/banker.gno:118-120).
A reader who follows the parameter list and calls
`banker.NewBanker(banker.BankerTypeReadonly, cur)` gets a panic, not a banker.

Read-only, no run needed (documentation claim checked against the source):

```bash
# from a local clone of gnolang/gno, at the PR head 7f309e625:
git fetch origin pull/6230/head && git checkout FETCH_HEAD
sed -n '747,757p' docs/resources/gno-stdlibs.md
sed -n '104,124p' gnovm/stdlibs/chain/banker/banker.gno
```

Observed doc:

```
Returns `Banker` of the specified type. Signature: `func NewBanker(bt BankerType, rlm realm) Banker`.

##### Parameters
- `bt` **BankerType** - type of Banker to get:
    - `BankerTypeReadonly` - read-only access to coin balances
```

Observed source:

```go
func NewReadonlyBanker() Banker {
	return banker{bt: BankerTypeReadonly}
}

func NewBanker(bt BankerType, rlm realm) Banker {
	if bt == BankerTypeReadonly {
		panic("use NewReadonlyBanker for BankerTypeReadonly")
	}
```

Band: Suggestion — the sub-bullet predates the diff, but the diff rewrote the
list header that scopes it to `NewBanker`.
