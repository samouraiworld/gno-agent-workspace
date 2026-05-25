# PR #5083: Update LICENSE.md with attribution requirements

URL: https://github.com/gnolang/gno/pull/5083
Author: jmsamv | Base: master | Files: 1 (semantic) / 291 (with merge drift) | +6 -4 (semantic) / +7433 -3150 (with merge drift)
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]

Verdict: NEEDS DISCUSSION — substantive change is a one-paragraph legal-text edit owned by the Licensor (NewTendermint); technical reviewers cannot judge whether the conditional "if the interface enables package uploads" CLA link belongs in `LICENSE.md` (a copyleft license body) vs. `TERMS.md` / `CLA.md` (where the surrounding obligations already live). Branch is also 318 commits behind master, has been Stale for 4+ months, and currently re-introduces removed files via merge. Rebase required regardless of the legal decision.

## Summary

Adds one sentence to the "Strong Attribution" additional terms in [`LICENSE.md:42-44`](../../../../../.worktrees/gno-review-5083/LICENSE.md#L42-L44): any UI showing the Attribution Notice must also surface a conspicuous link to the Gno.land Network Interaction Terms, and — conditionally, "if the interface enables package uploads" — to the Gno.land Contributor License Agreement. Both target documents already exist in master ([`TERMS.md`](../../../../../.worktrees/gno-review-5083/TERMS.md), [`CLA.md`](../../../../../.worktrees/gno-review-5083/CLA.md)), so the cross-references resolve. No code; pure license-text amendment.

The PR has one approval from [@jefft0](https://github.com/jefft0) ("Ready for @jaekwon to review") and is review-requested from [@jaekwon](https://github.com/jaekwon). Sitting in `Stale` for 4+ months; current `mergeable_state: dirty`. The +7433/-3150/291-files headline diff is merge drift, not real scope — the branch was opened off `master@65457904c` (Dec 2025) and never rebased.

## Fix

Before: lines 38-47 of the additional-terms paragraph said the Attribution Notice must be conspicuous and that the Licensor (NewTendermint) determines the URL and attribution terms. After: same paragraph, but a new sentence is inserted between "footer of the interface." and "The Attribution URL …" requiring a link to TERMS.md and (conditionally) CLA.md. Constraint: the link to the CLA is gated on "if the interface enables package uploads"; the link to the Interaction Terms is unconditional whenever the Attribution Notice is shown. See [`LICENSE.md:42-47`](../../../../../.worktrees/gno-review-5083/LICENSE.md#L42-L47).

## Critical (must fix)

None.

## Warnings (should fix)

- **[branch contains files removed from master]** `misc/jaekwon/tictac.md`, `misc/jaekwon/gnoland-whitepaper.md` — branch tree re-introduces files that no longer exist on master, via 318-commit drift.
  <details><summary>details</summary>

  The PR was branched off `65457904c` (Dec 2025). Master since rebased / removed jaekwon's `tictac.md` ([`dd4de568d`](https://github.com/gnolang/gno/commit/dd4de568d), [`6e70cc279`](https://github.com/gnolang/gno/commit/6e70cc279), [`9463ea16c`](https://github.com/gnolang/gno/commit/9463ea16c)) and `gnoland-whitepaper.md` ([`a43270e9b`](https://github.com/gnolang/gno/commit/a43270e9b)) commits, and the current `origin/master` no longer carries those paths under `misc/jaekwon/`. The PR branch still does. `gh api repos/gnolang/gno/pulls/5083 → mergeable_state: dirty`. A naive merge today would either re-introduce those files or surface conflicts on every one of the ~290 files in the drift. Fix: rebase `patch-2` onto current `origin/master` so the diff collapses back to the intended single-file `LICENSE.md` change; resolve no conflicts unilaterally on the legal text.
  </details>

- **[license-text scope]** [`LICENSE.md:42-44`](../../../../../.worktrees/gno-review-5083/LICENSE.md#L42-L44) — added obligation references a non-license document by name; check that this is what NewTendermint legal intended.
  <details><summary>details</summary>

  The Strong Attribution clause (Section 7 additional term, [`LICENSE.md:26-71`](../../../../../.worktrees/gno-review-5083/LICENSE.md#L26-L71)) is a permanent-term copyleft requirement that "travels with the Covered Work forever" ([`LICENSE.md:67-68`](../../../../../.worktrees/gno-review-5083/LICENSE.md#L67-L68)). The newly added sentence makes the displayed Attribution Notice depend on two external documents — "Gno.land Network Interaction Terms" ([`TERMS.md`](../../../../../.worktrees/gno-review-5083/TERMS.md)) and "Gno.land Contributor License Agreement" ([`CLA.md`](../../../../../.worktrees/gno-review-5083/CLA.md)). Those documents are themselves mutable (TERMS § 7: "These Terms may be updated from time to time through a community-governed process or by NewTendermint, LLC"; CLA references a `CLA Hash` updated as an on-chain parameter). Binding a perpetual license obligation to documents that the Licensor can rewrite at will is a substantive choice — only meaningful if the reader's compliance burden is interpreted at "the then-current text". This is exactly the kind of cross-reference question that needs explicit sign-off from whoever owns the license text (NewTendermint / @jaekwon), not approval from technical reviewers. Fix: confirm with the Licensor that (a) `LICENSE.md` is the intended home for this requirement vs. surfacing it only inside `TERMS.md`, and (b) the "Network Interaction Terms" / "Contributor License Agreement" labels match the canonical document names the Licensor will keep stable.
  </details>

- **[CLA conditional vagueness]** [`LICENSE.md:43-44`](../../../../../.worktrees/gno-review-5083/LICENSE.md#L43-L44) — "if the interface enables package uploads" has no defined predicate; arguable for any UI that hosts an `MsgAddPackage` button, ambiguous for read-only explorers that link to a builder.
  <details><summary>details</summary>

  Strong Attribution applies to every UI of an Applicable Work ([`LICENSE.md:28-32`](../../../../../.worktrees/gno-review-5083/LICENSE.md#L28-L32)). The new conditional creates two compliance tiers: every UI must link to TERMS; only "package upload" UIs must link to CLA. The phrase "enables package uploads" is undefined here and undefined in CLA.md / TERMS.md. Real cases that fall in the gap: a block explorer that surfaces an external "deploy via gnokey" link; a tutorial site that embeds a `gnodev`-style sandbox; a wallet UI that signs `MsgAddPackage` payloads constructed elsewhere. Each could be inside or outside the requirement under reasonable readings. Fix: either point at a specific functional test ("UIs that originate an `MsgAddPackage` transaction") or move the CLA-link requirement up into TERMS.md / CLA.md where the package-upload predicate can be defined alongside the CLA Hash submission requirement (`TERMS.md` § 5 already says "package publishing transactions are required to include the applicable CLA Hash" — that's the natural anchor for a UI link requirement, not the GPL-derivative attribution clause).
  </details>

## Nits

- [`LICENSE.md:42-46`](../../../../../.worktrees/gno-review-5083/LICENSE.md#L42-L46) — trailing whitespace on every added line. Original surrounding paragraph has none. Strip before merge.
- [`LICENSE.md:42`](../../../../../.worktrees/gno-review-5083/LICENSE.md#L42) — "Where the Attribution Notice is displayed in any user interface, you must also …" reads slightly awkwardly mid-paragraph; the prior sentence already established that the Attribution Notice "must be in a manner readily visible to users". Could tighten to "The interface must also provide …".
- PR title is bare "Update LICENSE.md with attribution requirements" — fails the `pr-title` CI check ([`actions/runs/21272301381`](https://github.com/gnolang/gno/actions/runs/21272301381/job/61224742612)). Conventional-commit prefix expected: `docs(license): …` or `chore(license): …`.

## Missing Tests

None — pure legal-text edit, nothing to assert in code.

## Suggestions

- Consider whether the CLA-link requirement should live in [`TERMS.md:§ 5`](../../../../../.worktrees/gno-review-5083/TERMS.md) (which already enumerates CLA-related obligations for package publishers) rather than inside the License's Strong Attribution clause. Putting per-UI link requirements in the license body increases the surface area that downstream forks must preserve verbatim.

## Questions for Author

- Was the substantive wording reviewed by NewTendermint legal, or is this still a draft for [@jaekwon](https://github.com/jaekwon) to redline? The PR body is one sentence; the linked LICENSE diff is a real legal change.
- Is "enables package uploads" intended to mean "UIs that originate `MsgAddPackage` transactions" specifically, or any UI that surfaces an upload affordance (link, button, embedded form) regardless of where the transaction is constructed?
- After rebase, can the PR description be updated to summarize the actual one-paragraph change? The current title + empty-body combination triggers Stale and made the PR effectively un-reviewable by anyone scanning the queue.
