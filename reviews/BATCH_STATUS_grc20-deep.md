# Batch status — deep review, grc20 (started 2026-09-08)

Model claude-opus-5, reviewer davd-gzl. **Deep mode** on three targets, three lens agents each
(red team, blue team, correctness), nine in parallel, synthesis and critics at the parent.

Synced head `43d2bf7` (`origin/main`, `samouraiworld`), `0 0` against `HEAD`, so `reviews/pr/` is
current. gno `origin/master` at dispatch: `93b592ff4`.

Go 1.25.9 unpacked at `/tmp/go1259`; every agent exports `PATH=/tmp/go1259/go/bin:$PATH` before a
test run. The system Go is a different minor and reddens filetests unrelated to the diff.

## Scope

The user asked for every recent grc20 pull request, then cut
[6138](https://github.com/gnolang/gno/pull/6138) by name. The final set is **3**.

## Dropped

| Reason | PRs |
|---|---|
| Cut by the user | [6138](https://github.com/gnolang/gno/pull/6138) |
| Reviewer's own PR | [5993](https://github.com/gnolang/gno/pull/5993) |
| Draft, last touched 2026-08-19 | [5965](https://github.com/gnolang/gno/pull/5965) |
| Matched the search, not grc20 | [6120](https://github.com/gnolang/gno/pull/6120) bank, [5728](https://github.com/gnolang/gno/pull/5728) and [6075](https://github.com/gnolang/gno/pull/6075) grc721 |

## The set

| PR | Head sha | Merge-base | Round | Worktree | Review dir |
|---|---|---|---|---|---|
| [6101](https://github.com/gnolang/gno/pull/6101) | `20d2a9f2e` | `bdeccddf6` | 2 | `.worktrees/gno-review-6101` | `reviews/pr/6xxx/6101-realm-scoped-token-ids/2-20d2a9f2e/` — **closed unmerged 2026-09-09, record only** |
| [6123](https://github.com/gnolang/gno/pull/6123) | `16f54a89c` | `d43dc0c50` | 2 | `.worktrees/gno-review-6123` | `reviews/pr/6xxx/6123-grc20-eoa-writes/2-16f54a89c/` — **superseded by `310f9c1ab`, record only** |
| [6139](https://github.com/gnolang/gno/pull/6139) | `011afff91` | `d4bb7ab93` | 1 | `.worktrees/gno-review-6139` | `reviews/pr/6xxx/6139-objectid-derived-ids/1-011afff91/` |

## The re-review gate

[6101](https://github.com/gnolang/gno/pull/6101) moved `911e1a57a` to `20d2a9f2e` and the patch-id
is **identical**, `72afada76`: the head is a merge of master and the branch content did not change.
A merge head is never base-only, so the round runs. `git show 20d2a9f2e --cc` prints no hunk, so
there is no conflict resolution to read; what the merge brings is
[#6134](https://github.com/gnolang/gno/pull/6134), [#6116](https://github.com/gnolang/gno/pull/6116)
and [#6124](https://github.com/gnolang/gno/pull/6124), and the suites those touch run on the new head.

[6123](https://github.com/gnolang/gno/pull/6123) moved `e014175d2` to `16f54a89c` and the patch-ids
differ, `f13a8dbbe` against `94eb12d07`. Six commits: `remove trusted host`, then a private
delegation and a public user teller in its place, then three test commits. Round 1's first Warning
was the trusted-relay capability, so the round is aimed at what replaced it.

## What the targets are to each other

[6139](https://github.com/gnolang/gno/pull/6139) is not independent work. Its body says it "takes the
direction the review of #6101 asked for", and both branches close
[issue 6026](https://github.com/gnolang/gno/issues/6026) by different means while editing the same
four files: `gnovm/stdlibs/chain/runtime/native.go`, `native.gno`, `generated.go` and
`native_gas.go`. [moul](https://github.com/moul) requested changes on 6101 on 2026-09-04. Both
cannot land as they stand, and the reconciliation is the parent's, per *Parallel dispatch* step 6.

## Prior rounds, unposted

Round 1 on [6101](https://github.com/gnolang/gno/pull/6101) and on
[6123](https://github.com/gnolang/gno/pull/6123) both ended REQUEST CHANGES and neither reached
GitHub. `gh api repos/gnolang/gno/pulls/<n>/reviews` carries no `davd-gzl` entry for any of the three.

## Resume

Worktrees are at the heads in the table. Re-dispatch any lens that died, per *Talking to the user*,
before anything else. The round directories above are written by the parent after synthesis, the
critic pass and the claim-verification gate.

## Where the round landed

Re-checked at the handover, per *Parallel dispatch*: a batch outlives its snapshot.

- [6101](https://github.com/gnolang/gno/pull/6101) **closed unmerged** on 2026-09-09, in favour of
  [#6139](https://github.com/gnolang/gno/pull/6139), which is what the round recommended. Record only.
- [6123](https://github.com/gnolang/gno/pull/6123) moved `16f54a89c` to `310f9c1ab`, patch-ids
  `94eb12d07` against `61ebca31c`. [`7ec45e078`](https://github.com/gnolang/gno/pull/6123/commits/7ec45e078)
  moves `UserTeller` from `*Token` back to `*PrivateLedger` and adds `IsCanonicalUserTeller`, so the
  round's own Suggestion is implemented and the round is superseded. A round 3 measures what is left.
- [6139](https://github.com/gnolang/gno/pull/6139) is unchanged at `011afff91`, so its round stands and
  its comment draft is the only one offered.
