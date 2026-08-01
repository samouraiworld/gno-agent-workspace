# Batch status — review all (started 2026-08-01)

Model claude-opus-5, reviewer davd-gzl. Normal (non-deep) mode.

Synced head `30bdd39` (`samouraiworld/main`) before building the set. The working tree first read the
set from the parent repo's recorded gitlink, `db4e141`, one commit behind that head, which hid the
already-reviewed 6025; the set below was rebuilt after checking out `main`.

gno master at dispatch: `d1a33f574`.

## External-contribution safety gate

Not applicable. All four PRs come from `MEMBER` accounts (jinoosss, notJoon, Villaquiranm); no
`FIRST_TIME_CONTRIBUTOR` in the set. 6022 was the one such PR and the user excluded it.

## Dropped

| Reason | PRs |
|---|---|
| already reviewed (present in `reviews/pr/`) | 6025 and the rest of the open non-draft set |
| excluded by the user | 6022 |
| dependabot | 6021, 6008, 5992, 5990, 5989 |
| WIP-titled | 5922, 5263, 5223, 4949 |
| authored by reviewer (davd-gzl) | 6006, 5993, 5950, 5936, 5934 |

None of the four in scope carries a prior review or review comment from `davd-gzl` on GitHub.

## Final set (4)

All four are first rounds. No head-unchanged, already-APPROVED, or patch-id gate applied.

| PR | Head sha | Author | Size | Worktree | Review dir | Mode |
|---|---|---|---|---|---|---|
| [6029](https://github.com/gnolang/gno/pull/6029) | `3b5b4a701` | jinoosss | +4282-1700, 46f | `.worktrees/gno-review-6029` | `reviews/pr/6xxx/6029-grc721-token-ledger-teller/1-3b5b4a701/` | normal |
| [6028](https://github.com/gnolang/gno/pull/6028) | `37182a315` | jinoosss | +342-145, 26f | `.worktrees/gno-review-6028` | `reviews/pr/6xxx/6028-registry-owned-id-generator/1-37182a315/` | normal |
| [6027](https://github.com/gnolang/gno/pull/6027) | `854b03529` | notJoon | +221-104, 12f | `.worktrees/gno-review-6027` | `reviews/pr/6xxx/6027-slug-alias-registrations/1-854b03529/` | normal |
| [6020](https://github.com/gnolang/gno/pull/6020) | `764ac4d84` | Villaquiranm | +1447-48, 21f | `.worktrees/gno-review-6020` | `reviews/pr/6xxx/6020-compute-map-keys-once/1-764ac4d84/` | normal |

6029, 6028 and 6027 all touch the token standards. 6028 and 6027 both change how a registration is
keyed in `grc20reg`, so they are likely to collide; read the pair together when synthesizing.

## Dispatch

One `general-purpose` agent per PR, all in one message. The parent created every worktree and
checked out every PR head; subagents never run `worktree add`, `gh pr checkout`, or any branch
switch. Subagents write `review_claude-opus-5_davd-gzl.md` and `comment_claude-opus-5.md`, and do
not commit, push, regenerate indexes, or post.

Environment: no Go toolchain on `PATH`. go1.25.9 lives at `/tmp/go/bin/go`; agents export
`PATH=/tmp/go/bin:$PATH` before running any suite.

## Progress

All four returned.

| PR | Verdict | Findings |
|---|---|---|
| 6029 | REQUEST CHANGES | 1 Critical, 5 Warnings, 2 Missing tests, 3 Nits, 2 Suggestions |
| 6028 | NEEDS DISCUSSION | 6 Warnings, 3 Missing tests, 5 Nits, 2 Suggestions |
| 6027 | NEEDS DISCUSSION | 1 Warning, 1 Missing test, 3 Nits, 2 Suggestions |
| 6020 | APPROVE | 2 Missing tests, 2 Nits, 2 Suggestions |

6020 notes: the map-key encoding never reaches persisted state. `MapKey` exists only as the in-memory
`vmap` index type and `copyValueWithRefs` rebuilds a `MapValue` from `List` alone, so the only
consensus-visible effect is gas, plus one deliberate output change for a composite key holding a NaN
ahead of an object-bearing field. The build-and-probe flag is derived once per call site from the
map's static key type, and `grep '\.vmap\['` finds five index accesses, all inside the one build and
the three accessors, so the build-and-lookup-must-agree hazard is structurally closed. All three new
realm guards are real: deleting `ensureVmap` from `GetPointerForKey`, `fillMapKeyRefs` before the
copy in `GetPointerAtIndex`, or the same line in `doOpMapLit` each reddens exactly one of
`zrealm_map7/8/9` and no sibling. Only `zrealm_map9` fixes a defect present on master, which prints
`stored key: 1 99`; the other two pass unchanged at the merge base and guard against breakage this
restructuring itself could cause. Headline findings: the PR summary's gas table reports
`compute_map_key_concrete_key` as 125449 to 124849, but 125449 is a mid-branch value and the measured
merge-base number is 135249, making the real delta -7.7% rather than -0.5%, with the 10400
reconciling as 9800 for the dropped per-write call plus 600 for the prefix; `delete` is the one write
path the compute-once sweep missed, measured at 104938 wasted gas on a `[1<<18]byte` key; `map51.gno`
claims to pin the displaced-key handoff but stays green when `mli.Key = key` is deleted, and its
`-0.0` is the constant `+0`; no filetest persists an interface-keyed map, leaving the prefix-keeping
branch of the predicate untested across the load path; and `mapKeyOmitType`'s `baseOf` is a no-op
since `DeclaredType.Kind()` already delegates. PR 5710 drops the `*Machine` parameter this PR relies
on to keep the lazy build unmetered, so that merge order needs a deliberate decision. APPROVE needs
human confirmation before it can be posted with `--approve`.

6029 notes: the core `Token`/`PrivateLedger`/`Teller` split tracks grc20 closely and holds up. Hooks
take no `realm` parameter so an extension cannot re-enter a teller write, state is written before the
hooks fan out, `IsCanonicalTeller` is present with an embedding-bypass test, and balances use
`overflow.Add64p`/`Sub64p`. The Critical is in the new `r/demo/grc721reg`: `Register` accepts any
`grc721.ExtensionView` from the calling realm and `extensionBadges` concatenates the kind string it
reports straight into `Render`, which lists every registered collection. Proven on chain through
`gno.land/pkg/integration` with a hostile realm returning a kind carrying a markdown heading and
link; both land in the shared listing, and the same string corrupts the `register` event's
comma-joined `extensions` attribute. Section 10 of `docs/resources/gno-ai-contract-review.md`; the
slug on the same code path is already charset-checked. Lead Warning: `RegisterExtension` has no
lifecycle guard, so attaching an enumerable after the first mint leaves the core at two tokens and
the extension at one, and moving a pre-attach token makes `TokenOfOwnerByIndex` answer with a token
the global list does not hold while the registry still advertises `enumerable`. One candidate finding
was killed by evidence and dropped to a Suggestion: the published metadata read view returns an
aliased `Attributes` slice, but an on-chain run showed a foreign realm's write is rejected by the
readonly taint, so only the in-realm case survives. CI green; the red `Merge Requirements` is the
approval bot.

6028 notes: the mechanism works, verified live — two post-genesis `grc20factory.New` calls minted
through grc20reg's shared generator on a running node and got distinct ids, so the cross-realm write
path holds via borrow rule 2 outside genesis. Three things stop a clean approve. `p/onbloc/identifier`
documents realm-scoped uniqueness it does not provide: two Generators bound to one realm at one
height emit byte-identical id streams, and the PR's own `newTestToken` helper produces two tokens
sharing a `Token.ID()`, the exact defect the PR sets out to remove, with
`TestNextIDShapeAndDeterminism` asserting the repeat as intended. Registered tokens escape only
because grc20reg happens to build exactly one Generator. `IdentifierGenerator()` returns a raw
`*Generator` into grc20reg's persisted state, section 8 of `docs/resources/gno-ai-contract-review.md`:
a four-line realm holding no token and registering nothing drove `NextID()` fifty times and the
receipt billed the storage diff to `gno.land/r/demo/defi/grc20reg`. Damage is bounded to a monotonic
counter, but grc20reg cannot rate-limit or revoke use of its own sequence. And one shared counter
makes a token id a function of the whole deployment set: inserting one `loadpkg` line ahead of foo20
moved its id and reddened the golden, so ids are chain-specific and no off-chain artifact can key on
one. Two `NewToken` doc claims are false, and the `realm.symbol` to `realm.slug` rekey ships with no
migration while the GovDAO treasury drops unresolvable keys silently.

Merge-order conflict between the two reviews, reconciled before either posts. The 6027 review
recommended landing 6028 first, the 6028 review recommended landing 6027 first; each had priced only
the rebase remainder it could see. Grounded against the issues: 6027 fixes issue 5988 by keying on
the slug, which removes the incidental guard that kept two byte-identical `Token.ID()` values out of
the registry and so widens issue 6026; 6028 closes 6026 properly with a generator-issued id whose
issuance `Register` verifies, and does the slug rekey too, so it covers 5988 as well. 6027 first is
the cheaper rebase but opens the 6026 window for as long as the two are apart, and 6028 carries six
Warnings so that gap is not obviously short. Both review files now carry the same conclusion: the
symbol leaves the registry key only once something else enforces id uniqueness, and a third order
costs neither side, with 6027 shipping a duplicate-id rejection of its own that 6028 then replaces
with the generator. Both drafts pose the order as a question and assert no order to the authors.

6027 notes: the registry key moves from `realm.symbol` to `realm.slug`, so `registry.Has(key)` now
guards slug reuse only. Two tokens built from the same caller-supplied seqid carry a byte-identical
`Token.ID()` and both register under different aliases, which breaks mapping a `Transfer`/`Mint`/
`Burn`/`Approval` event back to one entry. Delta proven with a filetest: the merge-base rejection
passes at `d1a33f574` and fails at `854b03529`. That is issue 6026. No in-tree registry key actually
moves; every updated call site passes its own symbol as the slug. CI green at the head; the red
`Merge Requirements` is the bot awaiting a review-team approval. The author's own "unnecessary"
thread on the new `cur.IsCurrent()` guard resolves in their favour: `cross` runs the same predicate
and aborts first, and no same-realm caller of `Register` exists, shown with a filetest.

Cross-PR, 6027 vs 6028: hard conflict. 6028 rewrites the same `Register` body and nine of 6027's
twelve files, already contains 6027's slug-keyed aliasing, and additionally closes the duplicate-id
hole 6027 opens via a registry-owned id generator whose issuance `Register` verifies. Divergences a
merge must settle: slug required (6027) vs optional (6028); event field `token_path` vs `token_key`;
origin prefix `rlmPath.symbol.` vs `rlmPath.`; `cur.IsCurrent()` added vs absent; `grc20.NewToken`'s
signature unchanged vs taking a `*identifier.Generator`. Landing 6028 first leaves 6027 a small
remainder. Landing 6027 first costs a second event-schema rename and leaves a window where the
registry accepts colliding ids. The draft recommends 6028 first and puts the order in the Body.

## Finalize

1. Parent commits once: `review: batch of 4 open PRs (6029, 6028, 6027, 6020)`.
2. Push to `review/pr-5999-r2`, the branch this turn started on; it already carries PR 7. No second
   PR on this repo.
3. Nothing reaches GitHub without the literal `post`.

## Carried from the 2026-07-29 batch

- [6002](https://github.com/gnolang/gno/pull/6002) draft verdict is APPROVE and still needs human
  confirmation before posting with `--approve`.
- [5991](https://github.com/gnolang/gno/pull/5991) draft verdict is APPROVE and still needs human
  confirmation before posting with `--approve`.
