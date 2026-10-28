# Claims: https://github.com/gnolang/gno/pull/6412 round 1, 7b79001d7, claude-opus-5-5, solo review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1001 | CONFIRMED | Nit | docs/users/interact-with-gnokey.md:412 | sed -n '412p' docs/users/interact-with-gnokey.md at 7b79001d7; git grep -c '\.\./\.\./examples/' -- 'docs/*.md' for the relative convention the PR body states for links leaving docs/ | sed -n '412p' docs/users/interact-with-gnokey.md: https://github.com/gnolang/gno/tree/master/examples/gno.land/r/demo/counter; git grep -c '../../examples/' docs: community-packages.md 15, gno-packages.md 4, gno-data-structures.md 3 |  | hot |  |
| 1002 | CONFIRMED | Nit | docs/resources/effective-gno.md:964 | sed -n '964p' docs/resources/effective-gno.md at 7b79001d7; git grep -c '\.\./\.\./examples/' -- 'docs/*.md' for the relative convention the PR body states for links leaving docs/ | sed -n '964p' docs/resources/effective-gno.md: https://github.com/gnolang/gno/tree/master/examples/gno.land/r/demo/defi/foo20; git grep -c '../../examples/' docs: community-packages.md 15, gno-packages.md 4, gno-data-structures.md 3 |  | cold |  |
| 1003 | CONFIRMED | Nit | docs/users/explore-with-gnoweb.md:77 | sed -n '77p' docs/users/explore-with-gnoweb.md at 7b79001d7; git grep -c '\.\./\.\./examples/' -- 'docs/*.md' for the relative convention the PR body states for links leaving docs/ | sed -n '77p' docs/users/explore-with-gnoweb.md: https://github.com/gnolang/gno/tree/master/examples/quarantined/gno.land/r/docs/hello; git grep -c '../../examples/' docs: community-packages.md 15, gno-packages.md 4, gno-data-structures.md 3 |  | cold |  |
| 1004 | CONFIRMED | Nit | docs/users/explore-with-gnoweb.md:125 | sed -n '125p' docs/users/explore-with-gnoweb.md at 7b79001d7; git grep -c '\.\./\.\./examples/' -- 'docs/*.md' for the relative convention the PR body states for links leaving docs/ | sed -n '125p' docs/users/explore-with-gnoweb.md: https://github.com/gnolang/gno/tree/master/examples/quarantined/gno.land/r/docs/source; git grep -c '../../examples/' docs: community-packages.md 15, gno-packages.md 4, gno-data-structures.md 3 |  | cold |  |

Hit rate per tier, from the rows above: hot 1/1 confirmed over 1 files, warm 0/0 confirmed over 0 files, cold 3/3 confirmed over 12 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| lines | docs/users/interact-with-gnokey.md:713 | gnokey verify may not take -chainid/-account-number/-account-sequence, or may not fall back to -remote as the new prose says | grep -n '"chainid"\\|"account-number"\\|"account-sequence"' tm2/pkg/crypto/keys/client/verify.go -> 66 "chainid", 72 "account-number", 77 "account-sequence"; LongHelp: 'If --account-number, --account-sequence or --chain-id are not set, the command queries the chain (via --remote)' |
| reach | docs/users/interact-with-gnokey.md:1147 | Run example on local gnodev signs with mykey, which gnodev may not fund | grep -n prefunded docs/resources/gnodev.md -> 89: 'Every key in your local `gnokey` keybase is prefunded on the built-in node.'; ls examples/gno.land/r/demo/counter exists |
| lines | docs/resources/glossary.md:229 | retargeted anchors #prefer-avltree-over-map-for-scalable-storage and `#10-unsanitized-user-input-in-render` may not exist | grep -n -> effective-gno.md:627 '### Prefer avl.Tree over map for scalable storage'; gno-ai-contract-review.md:163 '### 10. Unsanitized user input in `Render`', section imports gno.land/p/nt/markdown/sanitize/v0 |
| lines | docs/users/explore-with-gnoweb.md:782 | new GitHub links to examples/quarantined/gno.land/r/docs/{hello,source}, r/demo/counter, r/demo/defi/foo20 may point at missing paths | ls -d at head: all five directories exist |
| reach | docs/users/interact-with-gnokey.md:1504 | new live links (latest.testnets wugnot page, gno.land grc20 source, latest RPC, gpao status) may be dead | curl GETs: latest.testnets.gno.land/r/gnoland/wugnot 302 -> onyx.testnets.gno.land 200; gno.land/p/nt/grc20/v0$source&file=types.gno 200; rpc.latest.testnets.gno.land/status 200; gpao.latest.testnets.gno.land/status/... 200; rpc.staging.gno.land/status 503 (body's claim holds) |
| lines | docs/users/interact-with-gnokey.md:1030 | -send 1000ugnot to wugnot Deposit may fall below the realm's minimum | examples/gno.land/r/gnoland/wugnot/wugnot.gno:21 'ugnotMinDeposit int64 = 1000', :40 require(amount >= ugnotMinDeposit) |
| removed | docs/users/interact-with-gnokey.md:1365 | dropping -chainid staging from the multisig maketx send could change signing; multisign may also need a chain id | maketx.go:125 chainid is 'only useful with --broadcast', and the send uses -broadcast=false; grep -i chainid multisign.go -> no hits |
| removed | docs/builders/getting-started.md:45 | sibling staging references the sweep missed in docs/ | grep -rn -i staging docs --include='*.md' outside gnoland-networks.md -> only README.md:46 (lists staging as a network, still true), glossary.md:244 (Portal Loop history), glossary.md:272-273 (rewritten), effective-gno.md:992 (inside a TODO HTML comment); grep -e '-chain-id' docs -> none; grep test5 -> only archive rows in gnoland-networks.md |
| lines | docs/Makefile:2 | $(CURDIR) swap could change the linted path for a caller | read: CURDIR is the Makefile's directory after make -C or cd, PWD was the caller's shell directory; the change only fixes make -C docs lint |

## Completeness

- Catalog: skipped, per the project's rule for docs-only PRs; all 13 files are docs or `docs/Makefile`.
- Anchors the PR retargets: `effective-gno.md#prefer-avltree-over-map-for-scalable-storage` matches `### Prefer avl.Tree over map for scalable storage` at line 627 (same at base, so the old `#choose-storage-types-by-access-pattern` was dead), and `gno-ai-contract-review.md#10-unsanitized-user-input-in-render` matches its line 163; `gno-security-guide.md` has no `5.10` heading.
- Stale siblings, swept by name over `docs/*.md` at head: no `staging.gno.land`, `rpc.staging` or `-chainid staging` left outside `gnoland-networks.md` and the glossary entry that marks staging down; `effective-gno.md:992` is a TODO comment, `gnodev staging` is a gnodev mode.
- Claims checked live (read-only GETs): `rpc.latest.testnets.gno.land` answers 200 with `"network": "onyx-1"`; `rpc.staging.gno.land/status` answers 503; `vm/qfuncs` on the testnet lists `Deposit` for `r/gnoland/wugnot` and returns null for `r/demo/counter` and `r/docs/hello`; `auth/gasprice` is 1ugnot per 1000 gas, so `-gas-fee 8000ugnot -gas-wanted 8000000` sits at the floor; `latest.testnets.gno.land/r/gnoland/wugnot` redirects to `onyx.testnets.gno.land`; the gpao status URL answers 200; issue gnolang/gno#6258 is open and titled as the warning says.
- Code claims: `gnokey verify` queries `-remote` for any unset chain ID, account number or sequence (`verify.go:134-164`), and its flags are `chainid`, `account-number`, `account-sequence`, `sig-path`; `gnokey sign` and `maketx` default to chain ID `dev` and the root `-remote` to `127.0.0.1:26657`, as the multisig note says; `multisign` takes no chain ID; `maketx` names its flag `chainid`, so the `-chain-id` correction is right; MsgRun is gated by `RunSubmitters` (`params.go:19`) and onyx restricts it to one member (`misc/deployments/onyx.gno.land/README.md:17`).
- `docs/Makefile`: CI runs `make lint` with `working-directory: docs/`, where `$(PWD)` and `$(CURDIR)` agree; `$(CURDIR)` only changes `make -C docs lint` from another directory.
- Not run: a funded testnet `Deposit` with `-gas-wanted 8000000` (a write; the local receipt used 6975797), and the docs linter, which CI reports green.
- No scratch worktree was created: the round had no run-shaped candidate.
- Threads: the user's earlier CHANGES_REQUESTED asked for relative links; the author reverted the GitHub rewrite.

## Retro

- What failed: nothing died, no cap hit, no stage moved off `claude-opus-5-5`. The finder returned 0 candidates and 9 settled leads; the judge filed the 4 Nits itself from the completeness read.
- What worked: every lead the finder raised on the hot file, `gnokey verify` flags, the local `Run` funding, the `wugnot` minimum, was settled by a grep or a live read-only GET, so no judge run was needed.
- Hit rate per tier: hot 1/1, cold 3/3; the Nits are a link convention, not tier-driven, so no weight to revisit.
- Upgrade: none applied; the round ran at $3.94 and 8 min against a $5, 21 min projection.
