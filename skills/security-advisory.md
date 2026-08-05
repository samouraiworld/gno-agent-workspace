---
name: gno-security-advisory
description: Verify a gnolang/gno security finding by execution and write a disclosure-ready advisory. For findings against already-merged or deployed code (disclosures), not open-PR-diff findings. Produces a private advisory with an executed repro, a CVSS vector, a plain-language TL;DR, and a paste-ready body for GitHub's Security Advisory form.
argument-hint: <finding description | audit-finding-id>
---

# Gno Security Advisory

Verify a suspected vulnerability in `gnolang/gno` and write it up for private disclosure.

**Input:** `$ARGUMENTS` — a finding as free text, or an audit-finding id whose analysis lives in the private disclosure repo.

**Before starting:** read `gno/AGENTS.md` (Rules + Gno Security Semantics) and `gno/SECURITY.md`. For any caller-auth / cross-realm finding, read `gno/docs/resources/gno-interrealm.md` first.

## Disclosure gate — decide before writing anything

Deployment decides, not severity.

- Finding against an **open PR's own diff** → normal review. Use `skills/review.md`, stop here.
- Finding against **already-merged or deployed** code → **disclosure**. Keep every artifact out of this (public) repo. Raise it with the user first. All output goes to the private disclosure repo, whose path comes from the user or memory; never write its name into any public file, commit message, or path. A public issue and a public fix PR both telegraph the vector, and `SECURITY.md` forbids the issue.

## Verify by execution, never by reasoning

1. Throwaway worktree at the tip: `git -C gno worktree add <ABSOLUTE-path>/.worktrees/gno-<slug> origin/master`. Absolute path only — a relative path lands the worktree under `gno/.worktrees/` while your files go elsewhere.
2. Write the exploit as gno filetests: attacker realm(s) under `examples/gno.land/r/<slug>/`, each with a `gnomod.toml` (`module = "gno.land/r/<slug>"`, `gno = "0.9"`), plus a `// PKGPATH: gno.land/r/<victim>` filetest `main(cur realm)` that funds/sets up the victim, runs the attack, and prints the balance or state before and after.
3. Run from the examples module: `cd examples && go run ../gnovm/cmd/gno test -v -run . ./gno.land/r/<victim>`.
4. A filetest with no `// Output:` block reports FAIL — that is the missing golden, not a failed exploit. The printed before/after lines are the proof. `undefined` printed for an `error` value means nil (success).
5. Read the numbers. If nothing moved, the finding is wrong — drop it, do not retrofit a rationale.
6. Instrument identity when the mechanism is subtle: print `cur.Previous().Address()`, `cur.Address()`, and every actor address, so caller resolution is shown, not assumed.
7. Remove the worktree (`git worktree remove --force`) and confirm the `gno/` submodule has no leftover repro realms.

## Verify every anchor against the tip

Re-derive each `file:line` with `git show origin/master:<path> | grep -n`, not a remembered grep. A stale line number lands a maintainer on unrelated code.

## Sharpen the finding

Show the guards that hold before naming the crack, so the finding can't be dismissed with "but X already checks Y". State the precondition exactly (e.g. "the victim must directly call the attacker's realm", not "any interaction"). If asked "is this really a bug", re-run with full instrumentation rather than re-arguing.

## Advisory format

One file per finding, in the private repo. Sections, in order:

- `# <impact title>` — what an attacker achieves, plain, no jargon.
- `**Severity:**` · `**Affected:**` (name the realm/package first, then the class it generalizes to) · `**Status:** Deployed. Verified against current master.`
- `## TL;DR (plain language)` — 2-4 sentences a non-technical reader gets, no code.
- `## Summary` — the mechanism in prose.
- `## Details` — minimal code with verified anchors.
- `## Proof of concept` — trimmed repro code and the real captured output.
- `## Reach` — every affected surface/consumer as a table with anchors.
- `## Impact` — who is exposed and how large.
- A CVSS 3.1 vector; state the debatable axis (scope, availability) explicitly.

Omit remediation and provenance unless the user asks. Do not pin a commit hash unless asked — say "current master".

## Second-model review

Before finalizing, dispatch a `fable` agent synchronously (`run_in_background: false`) to verify each claim and anchor against `origin/master` and propose concision cuts. Re-verify anything it flags yourself; apply only what survives.

## Deliver

- Save the advisory and repro filetests in the private repo. Commit only on approval, push only on the literal "push".
- Paste-ready body: strip the H1 title (`tail -n +2 <file>`) — the advisory form carries the title separately.
- Submission channel: `https://github.com/gnolang/gno/security/advisories/new`, one report per finding, never a public issue. Triage access cannot create advisories via API; the user pastes the body and vector into the form.

## Never

- Name the private disclosure repo in any public file, commit message, or path.
- Claim a finding you have not executed.
- Leave a repro realm inside the `gno/` submodule.
