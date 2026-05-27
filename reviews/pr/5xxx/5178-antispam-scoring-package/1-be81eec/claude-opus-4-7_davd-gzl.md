# PR #5178: feat: antispam scoring package

URL: https://github.com/gnolang/gno/pull/5178
Author: alexiscolin | Base: master | Files: 42 | +11833 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5178 be81eec` (then `gh -R gnolang/gno pr checkout 5178` inside it)

**Verdict: NEEDS DISCUSSION** — author explicitly marks the PR WIP/PoC and lists "boards2 integration, threshold tuning, weight feedback" as blockers; CI is red and the branch has been idle for ~2 months. Beyond that, several real defects remain: `AdminLoadDefaults` swallows the regex compile error, `Score` silently overwrites three caller-supplied `ReputationData` fields, content is truncated inside the package (unresolved [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r1972437183) thread), and the homoglyph / leet pipelines have non-obvious blind spots.

## Summary

A new `p/gnoland/antispam` library plus a shared `r/gnoland/antispam` realm implement SpamAssassin-style multi-signal scoring (19 rules, 1-99 points each). Library is pure functions over caller-owned state (Corpus, Blocklist, KeywordDict, FingerprintStore); realm owns shared state and forwards `Score()` calls. Score is the sum of triggered rule weights with `EarlyExitAt` short-circuiting expensive rules (regex, Bayes, fingerprints) once a cheap-rule threshold is reached. Surface area is large (~3000 LoC of source, ~5000 LoC of tests) and there is no consumer yet — the only listed integration (boards2) is a separate not-yet-opened PR.

```
ScoreInput → [allow check] → [blocked check, 99pts, return]
              → rate → reputation → earlyExit?
              → content+unicode O(n) → earlyExit?
              → regex pattern → earlyExit?
              → tokenize → bayes → earlyExit?
              → keywords → earlyExit?
              → fingerprint MinHash
```

## Glossary

- `Corpus` — AVL of token → (spam, ham) counts; backs the Bayesian rule.
- `KeywordDict` — AVL of token → weight (1-3); backs `KEYWORD_SPAM` co-occurrence rule.
- `FingerprintStore` — AVL of int-key → MinHash signature; backs `NEAR_DUPLICATE`.
- `Blocklist` — three AVLs (blocked/allowed addresses, regex patterns) + one combined `*regexp.Regexp`.
- `TrainingGuard` — circuit breaker around auto-`Train()` (min score, min triggered rules, max trains).
- `EarlyExitAt` — score threshold above which expensive rules are skipped (0 = disabled).
- `OriginCaller` / `PreviousRealm` — chain-runtime helpers used to gate admin vs trusted-caller methods.

## Fix

Pure additive PR: introduces a new package and a new realm. No existing code is modified. The library is intentionally state-free (caller owns state, `Score()` reads only); the realm wraps it with shared corpus/keywords/blocklist and per-address moderation counters that are auto-merged into the caller's `ReputationData` at `Score()` time. Detection is layered cheapest-first with `earlyExit()` gates after each O(1) → O(n) tier.

## Critical (must fix)

- **[PR is explicitly WIP, not yet integrated, idle ~2 months]** [body](https://github.com/gnolang/gno/pull/5178) — body says "Needs: boards2 integration for testing, threshold tuning, and weight feedback before merge"; last code commit 2026-03-10, last comment 2026-05-11 from [@lbrown2007](https://github.com/gnolang/gno/pull/5178#issuecomment-2871556666) asking the author whether to push it to a later cycle. Author has not replied.
  <details><summary>details</summary>

  Without a downstream consumer (boards2 PR #5185 is referenced as "companion" but is not opened against this branch), there is no way to know whether the 19 rule weights and two thresholds (hide=5, reject=8) are calibrated to real traffic — they are picked from intuition and reviewer-by-reviewer micro-adjustments in the commit history (`b9e81704`: "adjust scoring thresholds for ALL_CAPS"). A scoring system with no production sample is a scoring system that decays on contact. Fix: hold this PR open until (a) the boards2 integration PR exists and exercises the API end-to-end, (b) the author publishes a calibration set (corpus of real spam/ham, score distributions, false-positive rate at hide=5), and (c) someone other than the author signs off on the weight table after seeing those numbers. CI is also currently red on `Run gno test`, `Run gno lint`, `gno2go` — those are due to a stale base; rebase needed before any merge consideration regardless.
  </details>

- **[silently ignored regex error in AdminLoadDefaults]** [`r/gnoland/antispam/antispam.gno:135`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/r/gnoland/antispam/antispam.gno#L135) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/r/gnoland/antispam/antispam.gno#L135) — `bl.AddPattern(defaultPattern)` discards its error; if the default pattern ever fails to compile after an edit, the realm boots with zero patterns and `BLOCKED_PATTERN` silently stops firing.
  <details><summary>details</summary>

  `AddPattern` returns `error` for invalid regex or cap exhaustion. Here the return is dropped, so a developer who breaks the pattern in a future PR — even via a one-character typo inside the 66-line string literal — gets a green deployment with the realm's flagship 21-category regex disabled. Nothing in `Render()` distinguishes "patterns loaded" from "patterns failed to load" beyond a count, and `AdminLoadDefaults` is the only path to the 21-category defaults. Fix: panic on error (consistent with `AdminAddPattern` at `antispam.gno:231`), or at minimum surface the error so `init`-time tests catch it. `AdminBulkAddKeywords` has the same shape — it calls `keywords.BulkAdd(data)` and `BulkAdd` returns nothing about malformed lines; consider returning an error or a count of skipped lines.
  </details>

## Warnings (should fix)

- **[Score() silently overwrites caller's ReputationData fields]** [`r/gnoland/antispam/antispam.gno:181-183`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/r/gnoland/antispam/antispam.gno#L181-L183) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/r/gnoland/antispam/antispam.gno#L181-L183) — if the address has an internal `addrReputation` row, the realm replaces `rep.FlaggedCount`, `rep.BanCount`, `rep.TotalAccepted` from the caller's input.
  <details><summary>details</summary>

  The docstring frames this as "auto-populate" — but the behavior is overwrite-without-merge. A caller realm (boards2, say) that already tracks its own per-author flag count cannot pass that count through; it will be silently replaced whenever the address has any entry in the shared `reputations` tree. Worse, the shape of the overwrite is invisible to the caller — they see the score result but not which fields were ignored. Two concrete consequences: (1) per-realm flag/ban data is unusable through this API once a realm starts recording, callers must keep two parallel ledgers; (2) any trusted realm that calls `RecordFlag(_, addr)` can poison every other realm's reputation view of that address. Fix: either (a) document the precedence loudly and add an "override" flag to the `Score()` signature, or (b) merge — sum the counters rather than replace — so caller and realm contributions both count.
  </details>

- **[allowlist beats blocklist, no admin warning]** [`p/gnoland/antispam/antispam.gno:164-166`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/antispam.gno#L164-L166) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/antispam.gno#L164-L166) — if an address is in both `allowed` and `blocked`, it scores 0; nothing in admin methods prevents the inconsistent state.
  <details><summary>details</summary>

  `AdminAllowAddress` and `AdminBlockAddress` both write without checking the other tree. The scoring path checks allowlist first and short-circuits to score 0 — so an address slipped into the allowlist by mistake permanently bypasses the 99-point blocked-address rule until an admin notices and calls `AdminRemoveAllow`. Fix: in `AdminAllowAddress`, panic or remove from the blocklist; in `AdminBlockAddress`, do the reverse. At minimum, document the precedence in the `Blocklist` doc comment and surface "allowed-and-blocked" in `Render()`.
  </details>

- **[content truncation inside Score, unresolved review thread]** [`p/gnoland/antispam/antispam.gno:157-159`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/antispam.gno#L157-L159) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/antispam.gno#L157-L159) — `Score()` truncates `Content` to 4096 bytes before any rule runs.
  <details><summary>details</summary>

  [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r1980737869) raised this on 2026-03-05 and the thread is still open. Two problems: (1) the truncation is by byte, not by rune — slicing mid-UTF-8 leaves an invalid trailing byte that `Tokenize` will decode as `U+FFFD`, mildly polluting the corpus; (2) spam tail content (the actual CTA, link, address) is dropped, so any pattern after byte 4096 escapes scoring. The fingerprint signature also changes with truncation, defeating cross-realm duplicate detection of long pasted spam. Fix: either truncate by rune boundary using `utf8.RuneCountInString` and a backwards scan to the last full rune, or move the cap up to the caller (as the reviewer suggested) and document the recommended ceiling.
  </details>

- **[regex panic claim assumes individual validation suffices]** [`p/gnoland/antispam/rule_blocklist.gno:149-156`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_blocklist.gno#L149-L156) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_blocklist.gno#L149-L156) — comment says "Patterns were individually validated in AddPattern. Compilation of (?:p1|p2|...) should always succeed."
  <details><summary>details</summary>

  Empirically this holds for the patterns RE2 supports today (verified locally: duplicate named groups, anchors, and flags survive union). But the `rebuildCombined` panic surface is permanent and externally reachable: any admin-controlled `AdminAddPattern` triggers a rebuild after `RemovePattern`, which itself can put the realm into a stuck state — `RemovePattern` panics → user can't remove the offending pattern. Fix: handle the error gracefully (preserve the previous compiled regex, return an error from `RemovePattern`), or, better, build patterns one at a time into a slice of `*regexp.Regexp` as [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r1980746421) suggested (also removes the 30-pattern cap and enables short-circuit matching).
  </details>

- **[leet normalize false-positive surface]** [`p/gnoland/antispam/rule_keywords.gno:146-181`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_keywords.gno#L146-L181) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_keywords.gno#L146-L181) — every digit in every token is rewritten regardless of context, so `web3`→`webe`, `h2o`→`h2o`, `cz`→`cz`, `eth2`→`etha`, `s3`→`se`, and any token that happens to leet-decode to a dict entry triggers (`fr33` → `free` is the intended path, but `fre3` → `free` also fires).
  <details><summary>details</summary>

  Tokenization already drops short tokens (<3 chars), so the false-positive surface is narrower than it looks, but it is not zero — and the scoring rule only needs **2** matches over a long doc to trip `KEYWORD_SPAM` (3 pts). Combined with one other low-weight rule the post hits the hide threshold (5) without containing any actual spam keyword. The deeper issue is that `normalizeLeet` is digit-only and unconditional — it doesn't gate on "token is alphanumeric mix where digits look like letters". A safer shape: only normalize when the resulting string matches a dict entry **and** the original token contains digits adjacent to letters (not pure-numeric, not numeric-suffix). Fix: condition normalization on `containsLetter(s) && containsDigit(s)` and gate on `getWeightDirect(norm)` hitting before counting the match (already done — but the doc-scaled `minMatches` makes this still tippable on long posts).
  </details>

- **[homoglyph script coverage is narrow]** [`p/gnoland/antispam/rule_unicode.gno:51-63`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_unicode.gno#L51-L63) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_unicode.gno#L51-L63) — `isLatin` stops at `0x024F` (Latin Extended-A), missing Latin Extended-B (`0x0250-0x02AF`), Latin Extended Additional (`0x1E00-0x1EFF`), and IPA Extensions; `isCyrillic`/`isGreek` cover only base blocks.
  <details><summary>details</summary>

  HOMOGLYPH_MIX fires when a single word contains both Latin and a non-Latin script. Real-world IDN attacks use Greek/Cyrillic letters inside Latin domains, which is covered — but Vietnamese (Latin Extended Additional) and African Latin variants (Latin Extended-B) sit outside the `isLatin` range and so are flagged as `wordHasOther = false`, breaking the heuristic for legitimately-Vietnamese text mixed with English (no mix detected, no false positive — that's fine) and breaking it for Latin Extended Additional homoglyphs (Vietnamese vowel composites used as homoglyphs — undetected). Lower priority because the realistic attacker uses Cyrillic, which is covered. Fix: extend `isLatin` to include 0x1E00-0x1EFF, document the script set in a comment block, and add a test case for Vietnamese-text mixed with Latin.
  </details>

- **[no ADR despite AGENTS.md mandate]** [`AGENTS.md`](https://github.com/gnolang/gno/blob/be81eec/AGENTS.md) · [↗](../../../../../.worktrees/gno-review-5178/AGENTS.md) — `AGENTS.md` says "Every non-trivial AI-assisted PR must include an ADR" with naming `pr5178_<description>.md`; no such file exists.
  <details><summary>details</summary>

  PR body and commit headlines do not explicitly disclose AI assistance, but the surface area (11833 lines, ~50% comments, README sections like "Why this matters", file naming like `z1_…filetest.gno` with sequential prefix) reads as AI-assisted scaffolding. If it is, the ADR is required; if it isn't, the PR is fine in that respect but the AGENTS.md disclosure rule still says to flag agent activity. Fix: ask the author to confirm and, if AI-assisted, add `gno.land/adr/pr5178_antispam_scoring.md` covering rule weight rationale, threshold calibration plan, training-guard parameters, and the precedence design (allowlist > blocklist > scoring).
  </details>

- **[fingerprint eviction reads as ambiguous]** [`p/gnoland/antispam/fingerprint.gno:140-147`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/fingerprint.gno#L140-L147) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/fingerprint.gno#L140-L147) — flagged by [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r2003186611); the `Iterate("", "", cb)` + `cb returns true` shape correctly stops at the first key, but it reads like it walks the whole tree.
  <details><summary>details</summary>

  The callback assigns `oldest = key; return true`. In gno's `avl.Tree.Iterate`, returning `true` stops iteration — so this does correctly grab the first (smallest) key in one step. But the loop body assigns to `oldest` before returning, which makes the code look like an "iterate all, keep the last" pattern. At `fpMaxStoreSize = 500` the eviction cost is fine, but the misread is real and the reviewer's `tree.GetByIndex(0)` is both clearer and avoids the callback indirection. Fix: switch to `tree.GetByIndex(0)` (or `ReverseIterate` for the largest key if oldest = highest count).
  </details>

- **[OriginCaller used for admin auth across cross-realm boundary]** [`r/gnoland/antispam/antispam.gno:121-125`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/r/gnoland/antispam/antispam.gno#L121-L125) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/r/gnoland/antispam/antispam.gno#L121-L125) — `assertAdmin` checks `runtime.OriginCaller()` against `adminAddr` so any realm in the call chain can invoke admin methods on the admin's behalf as long as the admin signed the tx.
  <details><summary>details</summary>

  This is documented as intentional ("admin operations are authorized by wallet signature regardless of the call chain") but gno convention strongly prefers `PreviousRealm()` for admin gates — see `gno-interrealm.md`. The risk: if the admin EOA ever signs a transaction to a misbehaving intermediate realm, that realm can call `AdminBlockAddress(targetAddr)`, `AdminTrain(content, true)`, or worst case `AdminSetAdmin(attackerAddr)` and lock out the legitimate admin. Concrete attack: a faucet/airdrop realm that the admin uses asks them to sign a "claim" call; under the hood it `cross()`-calls `antispamr.AdminSetAdmin(attackerAddr)`. Fix: use `PreviousRealm().IsUser() && PreviousRealm().Address() == adminAddr` for admin methods (mirrors `assertTrustedCaller` at `reputation.gno:46`), and add an explicit "admin must call via a user realm" check.
  </details>

- **[`repBanPenalty=1` is a no-op multiplier]** [`p/gnoland/antispam/rule_reputation.gno:17,55-61`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_reputation.gno#L17-L61) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_reputation.gno#L17-L61) — [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r1980770237) flagged it; still in code.
  <details><summary>details</summary>

  `penalty := rep.BanCount * repBanPenalty` with `repBanPenalty = 1` is `penalty := rep.BanCount`. The constant exists nominally for tuning but has been at 1 since introduction; the comment in `antispam.gno:41` already says "+1 per past ban, capped at 3." Either pick a non-1 value with a reasoned weighting or drop the constant — the dead-multiplier reads as decay risk: any future contributor who edits this might over-correct because they read the constant as load-bearing.
  </details>

## Nits

- [`p/gnoland/antispam/rule_blocklist.gno:69`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_blocklist.gno#L69) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_blocklist.gno#L69) — `RemoveAllow` is asymmetric with `UnblockAddress`; rename to `DisallowAddress` or `UnallowAddress` (or rename `UnblockAddress` to `RemoveBlock`) so the four address operations share a naming pattern.
- [`p/gnoland/antispam/antispam_test.gno:14`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/antispam_test.gno#L14) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/antispam_test.gno#L14) — [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r2003204143) suggested using realistic-semantics training content; still open.
- [`p/gnoland/antispam/bayes.gno:21`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/bayes.gno#L21) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/bayes.gno#L21) — typo `time.` → `time` was fixed elsewhere; comment still reads "more than this% of the time are considered" (missing space before `%`).
- [`p/gnoland/antispam/rule_keywords.gno:209`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_keywords.gno#L209) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_keywords.gno#L209) — combine the two nil/size/empty guards into one as [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r2003162049) suggested.
- [`p/gnoland/antispam/rule_blocklist.gno:42`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_blocklist.gno#L42) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_blocklist.gno#L42) — `bl.blocked.Set(addr, true)` uses `true` as value, but `AllowAddress` should match and use `struct{}{}` as [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r1980727734) suggested for memory efficiency.
- [`p/gnoland/antispam/fingerprint.gno:182-189`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/fingerprint.gno#L182-L189) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/fingerprint.gno#L182-L189) — `intToKey` is brittle: it assumes `n >= 0` (negative `n` produces bytes below `'0'`, breaking AVL ordering); add a comment or guard. In practice `n` is `fs.count` which only increments, so this is theoretical, but the function reads as general-purpose.
- [`r/gnoland/antispam/antispam.gno:148-157`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/r/gnoland/antispam/antispam.gno#L148-L157) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/r/gnoland/antispam/antispam.gno#L148-L157) — `Score()` has 9 positional parameters; even after `ScoreInput` was adopted in the engine, the realm wrapper keeps the long signature. Wrapping in a `ScoreRequest` struct (author's own counter-proposal at [discussion](https://github.com/gnolang/gno/pull/5178#discussion_r1972481876)) was agreed but not landed.

## Missing Tests

- **[allowlist + blocklist conflict]** `r/gnoland/antispam/antispam_test.gno` — no test verifying behavior when an address is in both lists (allowlist wins). Critical because the precedence is invisible from the API surface.
- **[AdminLoadDefaults error path]** `r/gnoland/antispam/antispam_test.gno` — no test that exercises a broken-default-pattern scenario; the silent-error path has zero coverage.
- **[content > 4096 bytes scoring]** `p/gnoland/antispam/antispam_test.gno` — boundary case: ensure truncation is rune-safe (no `U+FFFD` artifact when slicing across UTF-8) and that fingerprint signatures of the truncated and full versions are intentionally different.
- **[`OriginCaller`-via-intermediate-realm admin call]** `r/gnoland/antispam/antispam_test.gno` — no test simulates a misbehaving realm in the call chain invoking `AdminSetAdmin` on behalf of the admin. The threat model deserves a regression test.
- **[trusted-realm forging reputation]** `r/gnoland/antispam/reputation_test.gno` — no test for the case where a trusted realm calls `RecordFlag(_, victim)` repeatedly; the trust model assumes registered callers are well-behaved, but the test suite doesn't pin that assumption with a comment.

## Suggestions

- [`p/gnoland/antispam/antispam.gno:155`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/antispam.gno#L155) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/antispam.gno#L155) — [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r1972486420) proposed `[]ScoringFunc` slice; author preferred procedural. Reasonable trade-off but the procedural shape forces every new rule to learn the `earlyExit()` cadence; a function slice with an `expensive: bool` flag would centralize the cost-ordering invariant. Worth revisiting before adding a 20th rule.
- [`p/gnoland/antispam/rule_reputation.gno:6`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_reputation.gno#L6) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_reputation.gno#L6) — [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r1980763451) suggested `Balance chain.Coins` instead of `int64`; opens the door to non-GNOT balance signals in the future.
- [`p/gnoland/antispam/rule_reputation.gno:14`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_reputation.gno#L14) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_reputation.gno#L14) — `repMinAgeDays = 1` ([discussion](https://github.com/gnolang/gno/pull/5178#discussion_r1980767180)) is very low; new genuine users routinely have <1 day of history. Reviewer suggested 15 or 30 days.
- [`p/gnoland/antispam/rule_keywords.gno:17`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/rule_keywords.gno#L17) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/rule_keywords.gno#L17) — [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r2002989654) on `keywordSumThreshold = 4`: with weights capped at 3, the threshold trips after only 2 max-weight keywords or 4 weight-1 keywords. Worth tuning with real data.
- [`p/gnoland/antispam/bayes.gno:120`](https://github.com/gnolang/gno/blob/be81eec/examples/gno.land/p/gnoland/antispam/bayes.gno#L120) · [↗](../../../../../.worktrees/gno-review-5178/examples/gno.land/p/gnoland/antispam/bayes.gno#L120) — open question from [@jeronimoalbi](https://github.com/gnolang/gno/pull/5178#discussion_r1985283823) on token-overlap between spam and ham training; not answered by author. Worth documenting the expected ratio drift.

## Questions for Author

- Why does `Score()` overwrite `rep.FlaggedCount`/`BanCount`/`TotalAccepted` instead of merging? Boards2 will need both per-realm counts and shared counts.
- Was this PR AI-assisted? If so, where is the ADR (`gno.land/adr/pr5178_*.md`)? If not, the test naming convention (`z1_..z10_*_filetest.gno`) is unusual — what motivated it?
- The `defaultPattern` is one 66-line concatenated string with 21 alternations. Is there a tooling story (admin CLI, generator) for maintaining this, or does every category change require a realm redeploy?
- `assertAdmin` uses `OriginCaller`. Have you considered the call-chain-spoofing attack on `AdminSetAdmin`? Switching to `PreviousRealm` closes it at the cost of admins being unable to call admin methods through wallet UIs that interpose a realm. Which trade-off is intended?
- The PR is in WIP/PoC state per the body. What's the path to ready-for-merge — is it gated on the boards2 PR, on a calibration dataset, or both?
