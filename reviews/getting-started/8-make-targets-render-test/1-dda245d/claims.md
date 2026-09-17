# Claims: PR 8: make targets, Render test, AGENTS.md round 1, dda245ddc, the session model, standard review

Round shape: 2 finders, no reflector, 11 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 1 refuted, none of them above Nit.

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Missing test | hello_test.gno:24 | cp projects/getting-started/reviews/8-make-targets-render-test/1-dda245d/tests/b2-lines-claims-tests-reach-render-state-leak.gno <clone>/zz_state_leak_test.gno && gno test -v . ; then gno test -v . -run TestRenderSeesLeakedState for the isolated half | Read of the head worktree: ls shows hello_test.gno as the only test file (AGENTS.md CLAUDE.md gnomod.toml hello.gno hello_test.gno Makefile README.md). hello_test.gno:26 `if !strings.Contains(out, Get())` names no literal; the one literal in the file, line 10 `if got := Get(); got != "Hello, Gno!"`, is asserted against Get(), never against Render. Finder's run supplies the ordering: full suite logs `message after the suite = "Hello, Test!"`. | projects/getting-started/reviews/8-make-targets-render-test/1-dda245d/tests/b2-lines-claims-tests-reach-render-state-leak.gno | cold |
| 2 | CONFIRMED | Nit | hello_test.gno:26 | in a scratch worktree: sed -i 's/var message string = "Hello, Gno!"/var message string = ""/' hello.gno; perl -0pi -e 's/return `# ` \+ message \+ `/return `# oops/' hello.gno; gno test -v . -run TestRender => expect FAIL, observe PASS. Or drop tests/b2-lines-claims-tests-reach-render-assertion-vacuous.gno into the clone and run gno test -v . | Read: hello_test.gno:25-26 `out := Render("")` / `if !strings.Contains(out, Get())`; hello.gno:20-23 `func Set(_ realm, newMsg string) { message = newMsg }`, no validation; empty substring is contained in every string. Triggering input: Set(cross(cur), "") then any Render. Finder's mutated run agrees: `--- PASS: TestRender` with Render returning `# oops`. | projects/getting-started/reviews/8-make-targets-render-test/1-dda245d/tests/b2-lines-claims-tests-reach-render-assertion-vacuous.gno | cold |
| 3 | CONFIRMED | Suggestion | hello.gno:16 | grep -n 'And click here' hello.gno against README.md:61; then boot `gnodev .` with module set to gno.land/r/g1.../hello and follow the rendered link | hello.gno:16 is byte-identical at head and at the merge base (grep of both trees returns the same line). README.md:61-62 at head: "point `module` in `gnomod.toml` at a path you control — `gno.land/r/<your-address>/hello` always works and needs no registration"; the same grep at base returns nothing, so the instruction is new. gnomod.toml still declares module = "gno.land/r/example/hello". |  |  |
| 5 | CONFIRMED | Warning | AGENTS.md:49 | tests/b1-claims-gno-semantics_test.gno, TestMapIterationOrderIsDeterministic: insert z,a,m,b,q,c,y,d and range twice; expect a varying or non-insertion order per AGENTS.md, observe "zambqcyd" both times. Cross-check docs/resources/gno-data-structures.md:132 in gnolang/gno. | head, my artifact rerun from its file, gno built from gnolang/gno master bbd9b2ffe: INSERT="mqaze" OVERWRITE="mqaze" REINSERT="mqzea" INTKEYS=[9 3 7 1], --- PASS: TestJudge5MapOrder, ok . 0.58s; stable across three consecutive runs, where Go randomizes per range statement. The finder's b1-claims-gno-semantics_test.gno rerun verbatim: MAPORDER="zambqcyd" (the insertion order z,a,m,b,q,c,y,d), --- PASS, same on runs 2 and 3. Read of gnolang/gno at 26c0a7b32: docs/resources/gno-data-structures.md:132 "In Gno, map iteration order follows insertion order, unlike Go which uses randomized iteration order". Base: AGENTS.md does not exist at 78866a30a, so the sentence is introduced by this diff; the VM behaviour is the toolchain's and identical either side. The PR body reports verification on gno master.184+393b6f92a, a different master sha than the bbd9b2ffe built here, but MapList has carried insertion order across both. | projects/getting-started/reviews/8-make-targets-render-test/1-dda245d/tests/judge-5-map-order-insertion_test.gno | cold |
| 6 | CONFIRMED | Nit | hello.gno:20 | grep -n '^func Set' hello.gno against AGENTS.md:43-44 and README.md:54; then tests/b1-claims-cur-realm.sh, which renames `_` to `cur` and runs `make test lint`. | head read: hello.gno:20 `func Set(_ realm, newMsg string) {`; AGENTS.md:43 `first parameter \`cur realm\`, and callers use \`cross(cur)\`. See \`Set\` in`; README.md:54 `state-changing function (note the \`cur realm\` parameter and the \`cross(cur)\``, whose subject two lines up is `hello_test.gno`; hello_test.gno:8 `func TestSetAndGet(cur realm, t *testing.T) {`. The finder's tests/b1-claims-cur-realm.sh rerun with the prebuilt gno: after the rename `gno test .` -> `ok . 0.79s`, `gno lint .` -> no output, then reverted and the tree left clean, so the rename is free. Base at 78866a30a has no AGENTS.md and no `cur realm` line in README.md, and hello.gno is untouched by the diff, so the mismatched cross-reference starts at head. | projects/getting-started/reviews/8-make-targets-render-test/1-dda245d/tests/b1-claims-cur-realm.sh |  |
| 7 | CONFIRMED | Nit | AGENTS.md:53 | tests/b1-claims-gno-semantics_test.gno, TestUfmtDashFlagResidue: assert ufmt.Sprintf("%-5s", "ab") == "(unhandled verb: %-)". | My read of gnolang/gno examples/gno.land/p/nt/ufmt/v0/ufmt.gno in the local checkout at ac586457f: the `digits()` width scan runs before `verb := sTor[i]`, the switch default at :225 writes `"(unhandled verb: %" + string(verb) + ")"`, and the outer loop's non-'%' branch then writes each remaining rune. Finder's run on gno master bbd9b2ffe: `AGENTS.md says "(unhandled verb: %-)", got "(unhandled verb: %-)5s"`. Three shas are in play (ufmt source ac586457f, toolchain bbd9b2ffe, PR body master.184+393b6f92a); the parse order quoted is the same in the source read here. | projects/getting-started/reviews/8-make-targets-render-test/1-dda245d/tests/b1-claims-gno-semantics_test.gno | cold |
| 8 | CONFIRMED | Nit | AGENTS.md:45 | grep -n '^func [A-Z]' hello.gno, and read the $help&func=Set link on hello.gno:16. | My read of head hello.gno: `func Render(path string) string` (:7), `func Set(_ realm, newMsg string)` (:20), `func Get() string` (:25); hello.gno:16 `[And click here](/r/example/hello$help&func=Set&newMsg=...)`. AGENTS.md:45 "**`Render(path string) string` is the realm's whole public surface.**" against AGENTS.md:42-44 on crossing functions; README.md:56 repeats it, and the same grep at base returns nothing, so both sentences are new. |  | cold |
| 9 | CONFIRMED | Suggestion | AGENTS.md:24 | read .github/workflows/ci.yml:29-34 (unpinned master, go-version: stable) against Makefile:3 `GNO ?= gno`; confirm no pin exists on either side. | My read of the head tree: .github/workflows/ci.yml:21-26 installs via `\| sh -s -- --from-source` from master with no ref pin, :19 `go-version: stable`; Makefile:3 `GNO ?= gno`. No version pin on either side. |  | cold |
| 10 | CONFIRMED | Nit | Makefile:19 | grep -n 'GNO\\|gnodev' Makefile; then `make dev GNO=/path/to/gno` with gnodev off PATH. | My read of head Makefile: :3 `GNO ?= gno`, :22 `$(GNO) test .`, :25 `$(GNO) lint .`, :28 `$(GNO) fmt -w .`, :19 `gnodev .`. Base Makefile has no GNO variable at all (:10-11 `dev:` / `gnodev .`), so the inconsistency arrives with this diff. |  | warm |
| 11 | PLAUSIBLE | Nit | Makefile:9 | run `mawk 'BEGIN{FS=":.*?## "} /^[a-zA-Z_-]+:.*?## /{printf " %-8s %s\n",$1,$2}' Makefile` and `busybox awk` with the same script; compare against the gawk output. | My runs in the head worktree on the recipe's exact script: `awk`, `awk --posix` and `awk --traditional` each print all six targets, rc=0, GNU Awk 5.4.1. `command -v mawk busybox original-awk` finds none, so the two implementations the claim rests on were not exercised. What would confirm: the same script under mawk and busybox awk. |  | warm |
| 4 | REFUTED | Suggestion | hello_test.gno:25 | read hello.gno:7-18 for any use of path, then gno test -v . with an added Render("/foo") case | hello.gno:7-8 `func Render(path string) string {` / `return `# ` + message + `` — path is never read |  | cold |

Hit rate per tier, from the rows above: hot 0/0 confirmed over 0 files, warm 1/2 confirmed over 2 files, cold 6/7 confirmed over 4 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| claims | AGENTS.md:51 | sort.Slice does not exist in the gno stdlib | grep -n '^func [A-Z]' gnovm/stdlibs/sort/sort.gno search.gno -> Search, SearchInts, SearchFloat64s, SearchStrings, Sort, Reverse, IsSorted, Ints, Float64s, Strings, *AreSorted, Stable. No Slice. Claim holds. |
| claims | AGENTS.md:52 | ufmt.Sprintf("%03d", 7) returns "7" | gno test -v . -> TestUfmtZeroPadIgnored PASS; the width is parsed into `length` and never passed to writeInt (ufmt.gno doPrintf, case 'd'). Claim holds. |
| claims | AGENTS.md:47 | Example* without an // Output: block is skipped silently | gno test -v . on a package with ExampleNoOutput and ExampleWithOutput -> only `=== RUN ExampleWithOutput` appears and SENTINEL_EXAMPLE_WITHOUT_OUTPUT_RAN never prints. Claim holds. |
| claims | README.md:63 | the Deploy to a shared network link and its anchor resolve | curl -L -> 200 at https://docs.gno.land/builders/getting-started/ and grep of the page finds id="deploy-to-a-shared-network"; https://docs.gno.land/builders/editor-setup -> 200. Both hold. |
| claims | README.md:62 | gno.land/r/<your-address>/hello always works and needs no registration | examples/gno.land/r/sys/names/verifier.gno package doc: "PA (personal-address) namespaces - gno.land/{r,p}/<addr>/* ... Anyone can deploy under their own address." Claim holds; the only exception is the realm-wide emergency SetPaused, which rejects every check including PA. |
| claims | AGENTS.md:17 | make dev gives a local chain plus web UI on http://localhost:8888 | contribs/gnodev/main.go: with no subcommand gnodev runs local; contribs/gnodev/command_local.go:33 webListenerAddr: "127.0.0.1:8888". Claim holds, and `gnodev .` still parses. |
| claims | AGENTS.md:15 | make lists every target | make -C <worktree> -> six lines, help/install/dev/test/lint/fmt, matching the block in the PR body. Holds under gawk; see the Makefile:9 candidate for the non-gawk half. |
| claims | README.md:82 | moul/gno-contracts holds 50+ versioned self-contained contracts | gh api repos/moul/gno-contracts/git/trees/HEAD?recursive=1 -> 191 gnomod.toml paths outside vendor/, e.g. p/moul/addrset/v0, p/moul/authz/v1. Claim holds as a floor. |
| claims | AGENTS.md:20 | make fmt runs gno fmt -w . | gno fmt --help lists -w=false "write result to (source) file instead of stdout". Flag exists; claim holds. |
| claims | README.md:74 | PR body: repo-template's three Make targets all fail on the current toolchain | gh api repos/gnolang/repo-template/contents/Makefile -> six targets: all, dev, test, lint, fmt, install_deps. The count is wrong but it lives only in the PR body, which ships nothing. |
| tests | hello_test.gno:24 | TestRender might never run, the way AGENTS.md says an Example* without // Output: is skipped silently | gno test -v . in the scratch worktree at dda245d: `=== RUN TestRender` / `--- PASS: TestRender (0.00s)` / `--- GAS: 393458`, so it executes |
| lines | hello_test.gno:27 | t.Errorf's %q may degrade under gno's ufmt-based formatting, the way AGENTS.md records %-5s doing | mutated Render, gno test -v . -run TestRender printed `Render() should show the message "Hello, Gno!", got:` verbatim; a direct probe of ufmt.Sprintf("%q", "hi") returned `"hi"` |
| claims | hello_test.gno:24 | PR body: TestRender was checked against a deliberately broken Render (fails) and the restored one (passes) | broken Render: `--- FAIL: TestRender (0.00s)` / `failed: "TestRender"` / `FAIL: 0 build errors, 1 test errors`; restored: `--- PASS: TestRender`. Holds on gno master bbd9b2ffe, a different sha from the body's master.184+393b6f92a |
| claims | hello_test.gno:1 | PR body: gno fmt -diff . reports no diff | gno fmt -diff . at dda245d printed nothing, rc=0 |
| claims | hello_test.gno:21 | the test comment's "Render is the realm's entire public surface" is false: Set and Get are exported and callable, and Render's own output links to $help&func=Set | grep -n '^func ' hello.gno lists Render:7, Set:20, Get:25; a comment's own wording is never a candidate per the Finders rules, and the code it decorates is correct |

## Completeness

**Every changed file read.** Six: `.github/workflows/ci.yml`, `Makefile`,
`README.md`, `AGENTS.md`, `hello_test.gno`, `CLAUDE.md`. `hello.gno` and
`gnomod.toml` are untouched by the diff and were read too, since four candidates
rest on what they declare.

**Blast radius.** The repository is one package. `Render`, `Set` and `Get` have
no callers outside `hello_test.gno`, and `make test lint` is the whole of CI, so
the radius is the two documents a newcomer reads and the four `make` targets.
The reach angle found one path out of the repository: `README.md:60-65` tells a
reader to re-point `module`, and `hello.gno:16` hardcodes the old path.

**What ran.** Every semantic claim in `AGENTS.md` was executed against gno built
this turn from `gnolang/gno` upstream/master bbd9b2ffe with go1.25.9, never an
installed gno: `sort.Slice`, `ufmt.Sprintf("%03d", 7)`, `ufmt.Sprintf("%-5s",
"ab")`, `Example*` without `// Output:`, map range order, `gnodev`'s 8888
default, `gno fmt -w`, and `make`'s target listing. The two `README.md` links
were fetched, both 200, and the `deploy-to-a-shared-network` anchor exists on
the page. The PR body states verification on gno master.184+393b6f92a, a
different master sha: the map order and the `%-5s` residue are VM and stdlib
behaviour that carried across both, and no verdict in this round turns on the
difference.

**What did not run.** The `Makefile:9` candidate needs mawk or busybox awk to
show an implementation rejecting `.*?`; neither exists on this machine, so the
row stays PLAUSIBLE and the section ships SKIP. `make dev` was never booted: no
candidate rests on gnodev's runtime behaviour beyond its listen address, which
was read from `contribs/gnodev/command_local.go`.

**Anchors moved for posting.** Two confirmed rows anchor on `hello.gno`, which
the diff does not touch, so GitHub would reject them. The `hello.gno:20`
candidate is posted on `AGENTS.md:43-44`, the cross-reference that names the
wrong parameter, and the `hello.gno:16` candidate on `README.md:60-62`, the
instruction that makes the hardcoded link reachable, opening `Related
suggestion:`. The table above keeps the finders' anchors.

**Dropped at the author's direction.** Candidates 1 and 2, the `TestRender`
state leak and its vacuous needle, were drafted as one `Missing test` section on
`hello_test.gno:24-27` and cut from the comment on the reviewer's call. Both rows
stay above with their runs; nothing about them is posted.

**Already raised by the reviewer.** The `Render` surface nit at `AGENTS.md:45-46`
duplicates davd-gzl's own inline comment on `README.md:56`, posted at 16:10 on the
same head, which says `entire public surface` is wrong because `Get` and `Set` are
public too. The section ships `SKIP` with an `Already raised:` line.

**Wording set by the reviewer.** Three sections were rewritten after the round:
the Body dropped its `AGENTS.md` framing, the Warning dropped its repro block,
the `Render` surface nit dropped its replacement text, and the hardcoded-link
suggestion names `CurrentRealm().PkgPath()` rather than a package-qualified
form. `Render` takes no `realm` parameter, measured at `hello.gno:7`, so `cur`
is not in scope there; on gno master `bbd9b2ffe` the free function sits in
`chain/runtime/unsafe` and `cur.PkgPath()` is the method form where a `cur`
exists.

**Refuted.** One row: `Render` ignoring `path` is the shape every single-page
realm has, and no claim in the diff says otherwise.

**Not covered.** No round has run on this repository before, so nothing was
compared against a prior round's cleared set. The repository carries no
invariant catalog and no project delta, so the catalog angle had nothing to
walk.

## Retro

**What failed.** The parent's own cost estimate. It reasoned from the repository's
size, 262 lines in eight files, to a per-turn context of 13k and quoted ~$3 to the
user; `./scripts/review-retro.py` measured 88k median against the baseline's 91k,
and ~$14. Per-turn context is the stage's rule file, harness overhead and
accumulation, none of which shrinks with the reviewed tree. The `text` pass was
skipped by `text.min_findings` at 4 against 10 kept findings, which is a threshold
on findings rather than on visible words; the parent ran the style pass by hand
and rewrote nine sentences, so the pass was owed.

**What worked.** The `claims` angle, which carried this diff: 11 candidates from 2
finders, 9 CONFIRMED, 1 REFUTED, 1 PLAUSIBLE. Both toolchain claims in `AGENTS.md`
were settled by running them against a gno built from master this turn, and one
came back four characters short of what the file states. Anchor relocation worked:
two confirmed rows sat on `hello.gno`, outside the diff, and moved to the lines
that make them reachable rather than being dropped.

**Hit rate per tier.** Warm files, `.github/workflows/ci.yml` and `Makefile`, 2 of
6 files, yielded 2 confirmed rows. Cold files, 4 of 6, yielded 8. The tier weights
in `round risk` ranked config above docs on a change whose whole substance is
docs, so the ordering bought nothing here; on a diff of this shape the prose line
count deserves weight the table does not give it.

**Projection against measurement.**

| Line | Projected | Measured |
| --- | --- | --- |
| agents | 13 | 7 |
| minutes | 39 | 10 |
| output | 426k | 171k |
| cache read | 37M | 7.1M |
| cache write | 1.3M | 0.62M |
| cost | ~$42 | ~$14 |

Cost from the `claude-api` model table: Claude Opus 5 at $25/MTok output, cache
read a tenth of input, cache write twice it under a one-hour TTL.

**Upgrade, with its estimate.** Gate the text pass on visible words rather than on
finding count, so a round of ten short findings still gets the pass its prose
needs. Roughly one agent and ~50k output on a round this size, against nine
sentences the parent rewrote by hand. Written to the workspace `TODO.md` this turn.
