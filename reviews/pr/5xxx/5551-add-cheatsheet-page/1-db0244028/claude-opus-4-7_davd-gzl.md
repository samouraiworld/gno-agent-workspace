# PR #5551: docs: add cheat sheet page

URL: https://github.com/gnolang/gno/pull/5551
Author: davd-gzl | Base: master | Files: 6 | +864 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — `docs` CI is red on JSX/URL linter triggers introduced by this PR, the branch is stale against master so `getting-started.md` + `quickstart.md` are duplicated (already landed via #5592), and the curl-vs-inline counter in `quickstart.md` ships an `Increment` whose signature won't accept the call example.

## Summary

Adds a single-page cheat sheet (`docs/cheatsheet.md`, 460 lines) covering install, key management, query, call, send, deploy, multisig, airgap, valoper, and contributor commands, with four audience-segmented ToC sections. The PR also re-adds the `getting-started.md` (317 lines) and `quickstart.md` (75 lines) files that #5592 already merged onto master — the branch's merge base is `a28468f28` (one commit before #5592), so the diff against current master shows these as "new" when they're stale duplicates. CI fails on three lint hits in the duplicated `quickstart.md` / `getting-started.md`; the cheatsheet itself is clean against the linter.

## Glossary

- `xurls.Strict()`: the URL extractor the docs linter uses. Stops at the first non-URL char (e.g. `<`), so `https://rpc.<chainid>...` yields `https://rpc` — a definite 404.
- `extractJSX`: the docs linter's JSX detector. Greedy on `<...>`; `<https://staging.gno.land/r/<your-g1-addr>...>` is read as one tag.
- `gno mod init`: the supported way to create `gnomod.toml`; replaced the unmerged `gno init` wizard previously referenced in the PR.

## Fix

The PR's headline artifact is `docs/cheatsheet.md` — a flat reference of `gnokey` / `gno` / `gnoland` / `gnodev` commands grouped by audience (User, Developer, Valoper, Contributor). The User chapter is the densest: key creation, query, call, send, deploy, multisig, airgap, verify. The Developer chapter covers realm scaffolding through deploy-to-staging. Valoper documents `gnoland secrets`, `r/gnops/valopers.Register`, and the Update* functions. Contributor covers `make install` / `make test` / golden file regeneration.

The "new" `getting-started.md` and `quickstart.md` in the diff are an artifact of the stale branch: master already has both (PR #5592, merged 64c945f1b). When this PR rebases, those two files will conflict, and the resolution should drop both copies in favour of master's versions.

## Critical (must fix)

- **[CI is red: JSX linter rejects `<https://...<your-g1-addr>...>`]** [`docs/builders/quickstart.md:71`](../../../../../.worktrees/gno-review-5551/docs/builders/quickstart.md#L71) — the autolink `<https://staging.gno.land/r/<your-g1-addr>/counter>` reads as a JSX tag because the inner `<` terminates the greedy regex at the placeholder.
  <details><summary>details</summary>

  The docs linter at [`misc/docs/tools/linter/jsx.go:26`](../../../../../.worktrees/gno-review-5551/misc/docs/tools/linter/jsx.go#L26) runs `(?s)<[^>]+>` after stripping code blocks and inline code. Markdown autolinks (`<URL>`) are normally fine, but with a nested `<your-g1-addr>` placeholder the matched string becomes `<https://staging.gno.land/r/<your-g1-addr>`. CI emits:

  ```
  >>> <https://staging.gno.land/r/<your-g1-addr> (found in file: docs/builders/quickstart.md)
  ```

  This is the file that's already on master via #5592, so the cleanest fix is to drop this PR's copy at rebase. If keeping any version: rewrite the line as plain prose so the URL isn't autolinked, e.g. `` Live at `https://staging.gno.land/r/<your-g1-addr>/counter`. ``
  </details>

- **[CI is red: linter resolves `https://gno.land/r/docs` to 404]** [`docs/builders/quickstart.md:75`](../../../../../.worktrees/gno-review-5551/docs/builders/quickstart.md#L75) and [`docs/builders/getting-started.md:305`](../../../../../.worktrees/gno-review-5551/docs/builders/getting-started.md#L305) — `r/docs` returns 404 on mainnet right now.
  <details><summary>details</summary>

  Verified live:

  ```bash
  curl -sS -o /dev/null -w "%{http_code}\n" --max-time 10 https://gno.land/r/docs
  # 404
  ```

  The linter at [`misc/docs/tools/linter/urls.go:127`](../../../../../.worktrees/gno-review-5551/misc/docs/tools/linter/urls.go#L127) treats a definitive 404 as a hard error. Both occurrences are in files that should be discarded at rebase (already on master via #5592). On master the equivalent reference is to `gno.land/r/gnoland/home` or other reachable paths; align with master's version rather than re-introducing the broken link.
  </details>

- **[CI is red: URL extractor sees `https://rpc` (the `.<` terminates)]** [`docs/builders/getting-started.md:150`](../../../../../.worktrees/gno-review-5551/docs/builders/getting-started.md#L150) — the Testnet row contains `` `https://rpc.<chainid>.testnets.gno.land:443` ``; `xurls.Strict()` stops at `<`, yielding `https://rpc.` (trailing dot trimmed → `https://rpc`).
  <details><summary>details</summary>

  [`misc/docs/tools/linter/urls.go:22-58`](../../../../../.worktrees/gno-review-5551/misc/docs/tools/linter/urls.go#L22-L58) does not strip code blocks or inline code before extraction — so anything in a backtick on its own line is still scanned. CI emits:

  ```
  >>> https://rpc (found in file: docs/builders/getting-started.md)
  ```

  Master's `getting-started.md` (the version landed via #5592) has a different shape for this table that doesn't trip the extractor. Rebasing onto master removes this hit; if the version were to survive, the placeholder pattern needs to keep the `<` out of the URL (e.g. `https://rpc.<chainid_testnet>.testnets.gno.land:443` → write the `<chainid_testnet>` segment as a bracketed substitution outside the URL, or drop the row).
  </details>

- **[stale branch — `getting-started.md` and `quickstart.md` already on master]** [`docs/builders/getting-started.md`](../../../../../.worktrees/gno-review-5551/docs/builders/getting-started.md), [`docs/builders/quickstart.md`](../../../../../.worktrees/gno-review-5551/docs/builders/quickstart.md) — merge base is `a28468f28`; PR #5592 (`64c945f1b`) landed both files after that base.
  <details><summary>details</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5551 -R gnolang/gno
  git fetch origin master
  git merge-base HEAD origin/master         # a28468f28...
  git log $(git merge-base HEAD origin/master)..origin/master --oneline -- docs/builders/getting-started.md
  # 64c945f1b docs: add getting started (alternative to #5519) (#5592)
  ```

  The PR body says "docs(getting-started): link cheatsheet sections from getting-started" — implying small edits — but the diff against master shows full file creations. On rebase they'll be three-way conflicts: this PR's `getting-started.md` is **meaningfully behind** master's (master has the newer Phase 3 interrealm signature `Increment(_ realm)` with `Increment(cross(cur))` in tests, this PR still has the old `Increment(_ realm, n int)` with `Increment(cross, 5)`). The cheatsheet (`docs/cheatsheet.md`) is the only actually-new artifact. Resolve by keeping master's versions of `getting-started.md` and `quickstart.md` verbatim, dropping this PR's copies, and shipping only the cheatsheet + README link + sidebar entry + `what-is-gnolang.md` cross-link.
  </details>

- **[curl/inline counter mismatch breaks the call example]** [`docs/builders/quickstart.md:16-17`](../../../../../.worktrees/gno-review-5551/docs/builders/quickstart.md#L16-L17) vs [`:33-36`](../../../../../.worktrees/gno-review-5551/docs/builders/quickstart.md#L33-L36) vs [`:64-68`](../../../../../.worktrees/gno-review-5551/docs/builders/quickstart.md#L64-L68) — the curl fetches `examples/gno.land/r/demo/counter/counter.gno`, whose actual signature is `Increment(_ realm) int` (no `n`). The inline snippet shows `Increment(_ realm, n int) int`. The deploy step then calls `Increment -args "5"`, which fails against the curled file.
  <details><summary>details</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5551 -R gnolang/gno
  cat examples/gno.land/r/demo/counter/counter.gno
  # package counter
  # ...
  # func Increment(_ realm) int {
  #     counter++
  #     return counter
  # }
  ```

  A reader following the curl path (steps 3–8) will:
  1. Fetch a counter that takes no args.
  2. Deploy it.
  3. Hit `gnokey maketx call -func Increment -args "5"` — arg count mismatch, runtime error.

  Two paths shipping different counters is the underlying smell; the inline version belongs in `getting-started.md` (where it's used). For the curl path, drop `-args "5"` so the call works against the real example. This file is already on master via #5592, so the proper fix is again to drop this PR's copy at rebase.
  </details>

## Warnings (should fix)

- **[stale `misc/docs/sidebar.json`]** [`misc/docs/sidebar.json`](../../../../../.worktrees/gno-review-5551/misc/docs/sidebar.json) — adds `cheatsheet` at position 9 in `Build on Gno.land` but omits `builders/getting-started` and `builders/quickstart`, both of which master's sidebar already contains.
  <details><summary>details</summary>

  Master's sidebar (`git show origin/master:misc/docs/sidebar.json`):

  ```
  "builders/getting-started",
  "builders/quickstart",
  "builders/install",
  "builders/what-is-gnolang",
  "builders/anatomy-of-a-gno-package",
  ...
  ```

  This PR's sidebar drops the first two and lists `cheatsheet` last. The file is generated (`make generate` runs `indexparser` against `docs/README.md`) so the right way to fix is to:
  1. Rebase so `docs/README.md` reflects all three (`getting-started`, `quickstart`, `cheatsheet`).
  2. `cd docs && make generate` to regenerate `misc/docs/sidebar.json`.

  Otherwise CI's `make generate -B && git diff` check will reject the PR on top of the lint failures.
  </details>

- **[doc claims a filter that isn't a filename filter]** [`docs/cheatsheet.md:296-297`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L296-L297) — `gno test -run "_filetest.gno" .` is presented as "run only filetests", but [`gnovm/cmd/gno/test.go:141-146`](../../../../../.worktrees/gno-review-5551/gnovm/cmd/gno/test.go#L141-L146) defines `-run` as a **test-name** pattern, not a filename filter.
  <details><summary>details</summary>

  It happens to work because filetest test names are built as `<fsDir>/<testFileName>` at [`gnovm/pkg/test/test.go:408-412`](../../../../../.worktrees/gno-review-5551/gnovm/pkg/test/test.go#L408-L412) and the filename includes `_filetest.gno`. But the regular `_test.gno` files still run, because their test names don't match `_filetest.gno` so `shouldRun` filters them out at the `_test` side too — net effect: filetests run, normal tests don't. So the command does what the doc claims, accidentally.

  Fix: explain *why* the command works (filetest test names contain the filename, so `-run` matches them), or use `gno test -v -run "^.+_filetest\.gno$" .` to make the intent clear. A reader copying this command will be confused if a future refactor changes the filetest test-name format.
  </details>

- **[`--update-golden-tests` documented with leading double-dash]** [`docs/cheatsheet.md:440`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L440), [`:443`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L443) — Go's `flag` package accepts both `-` and `--`, but the gno repo convention is single-dash; [`gnovm/cmd/gno/test.go:122`](../../../../../.worktrees/gno-review-5551/gnovm/cmd/gno/test.go#L122) registers it as `"update-golden-tests"` and every txtar example in `gnovm/cmd/gno/testdata/test/` uses `-update-golden-tests`. Use single-dash for consistency.

## Nits

- [`docs/cheatsheet.md:139`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L139) and [`:154`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L154) — `-broadcast` is included but `broadcast=true` is the default ([`tm2/pkg/crypto/keys/client/maketx.go:94-99`](../../../../../.worktrees/gno-review-5551/tm2/pkg/crypto/keys/client/maketx.go#L94-L99)). Either drop it everywhere (less noise) or keep it everywhere; the cheatsheet is inconsistent — Send (line 139) and addpkg (154) include it, Call (122-128) doesn't.
- [`docs/cheatsheet.md:51-54`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L51-L54) — install + uninstall side by side is fine, but the uninstall comment says "binaries in `$GOPATH/bin`" while [`misc/install.sh`](../../../../../.worktrees/gno-review-5551/misc/install.sh) (and master's `getting-started.md`) lands binaries in `$HOME/.gno/bin`. Confirm which path is current; the comment is likely stale.
- [`docs/cheatsheet.md:69-72`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L69-L72) — listing the `devtest` mnemonic inline is great UX but a reader will paste it into their primary keybase under `devtest`, then find `test1` is also there (same address, different name) when they run `gnodev`. Add one line: "this is the same key gnodev preloads as `test1` — importing it is optional."
- [`docs/cheatsheet.md:354`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L354) — `Register Valoper Profile (on-chain)` has parenthetical that breaks the ToC anchor on line 30 (`#register-valoper-profile`). GitHub renders the anchor as `#register-valoper-profile-on-chain`. Drop `(on-chain)` from the heading or update the ToC link.
- [`docs/cheatsheet.md:202`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L202) — Multisig step 4 (`gnokey multisign`) takes `-signature alice.sig -signature bob.sig` but doesn't mention they have to be in the same order as `--multisig alice --multisig bob --multisig carol` on line 174. Worth a one-line note; off-order produces an opaque "invalid signature" error.
- [`docs/cheatsheet.md:284`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md#L284) — `gnodev -resolver remote=https://rpc.staging.gno.land:443` example is good; consider also showing the test4 / current testnet form so readers don't pin to staging.

## Missing Tests

None — docs-only change.

## Suggestions

- [`docs/README.md:40`](../../../../../.worktrees/gno-review-5551/docs/README.md#L40) — the new "Cheatsheet" link under `Build on Gno.land` is logical, but the line description ("Essential Gno commands: install, create, test, query, call, deploy.") undersells the User chapter, which is the most useful section for non-developers. Consider: "Reference card for users, developers, valopers, and contributors — every common `gnokey`/`gno`/`gnoland` command."
- [`docs/cheatsheet.md`](../../../../../.worktrees/gno-review-5551/docs/cheatsheet.md) — every section that links to a longer reference uses `> [link](path)` blockquotes inconsistently: some sections have them (Install, Generate a Key, Query, Call, Multisig, Airgap, Create a Realm, Run Locally, Test, Create a Run Script, Deploy to Staging, Register Valoper Profile, Start a Local Chain, Update Golden Files), others don't (Manage Keys, Send Coins, Deploy a Package, Verify a Signature, Format & Lint, Update Valoper Profile, Query Valopers, Build & Test Go, Lint & Format Go). Either every section gets a "see also" line or none does; the current pattern looks accidental.

## Questions for Author

- The PR description still says "Depends on: #5492, #5592" — #5592 has already merged. Worth refreshing the body so reviewers don't think this PR is still waiting on something.
- Once rebased on master, the diff should be three small adds (cheatsheet, README link, what-is-gnolang link) plus one sidebar regeneration. Is the plan to rebase and force-push, or close-and-reopen against current master?
