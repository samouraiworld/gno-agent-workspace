# Claims: PR 6192: docs(examples): correct grc20/grc721 comments on the sub-realm separator and Token.ID() round 1, 37e2ce18b, the session model, solo review

Round shape: solo round, one agent, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | examples/gno.land/p/nt/grc20/v0/token.gno:33 | two filetests in examples/gno.land/p/nt/grc20/v0/filetests, run with gno test -v ./gno.land/p/nt/grc20/v0: one registering the same symbol from two subs and the host, one registering a sub-created token from the host frame | gno test -v ./gno.land/p/nt/grc20/v0: key a: gno.land/r/demo/grc20subdup#a.SREG, key b: gno.land/r/demo/grc20subdup#b.SREG, key c: gno.land/r/demo/grc20subdup.SREG, all three PASS; the host-frame filetest aborts with grc20reg: token must be registered from its own realm at grc20reg.gno:44 | tests/solo-subrealm-registry-key.gno |  |
| 2 | CONFIRMED | Nit | examples/gno.land/p/nt/grc721/v0/token.gno:13 | read examples/gno.land/p/nt/grc721/v0/token.gno lines 13 and 40-45 at the head | examples/gno.land/p/nt/grc721/v0/token.gno:13 against :40-45 at the head |  |  |
| 3 | CONFIRMED | Nit | gno.land/adr/acd01fa29_grc20_caller_teller_home_binding.md:47 | grep -rn '":subpath"' over the head worktree | grep -rn '":subpath"' <head>: gno.land/adr/acd01fa29_grc20_caller_teller_home_binding.md:47 |  |  |
| 4 | REFUTED | Nit | examples/gno.land/p/nt/grc20/v0/tellers.gno:144 | grep for the subRealmSep constant in gnovm/pkg/gnolang/uverse.go and gnovm/stdlibs/chain/address.gno | gnovm/stdlibs/chain/address.gno:13 const subRealmSep = "#" |  |  |
| 5 | REFUTED | Nit | examples/gno.land/p/nt/grc20/v0/token.gno:138 | read NewToken's construction at token.gno:60-73 and print a sub-created token's ID() in a filetest | examples/gno.land/p/nt/grc20/v0/token.gno:67 id: pkgPath + "." + symbol + "." + id.String() | tests/solo-subrealm-registry-host.gno |  |
| 6 | REFUTED | Nit | examples/gno.land/p/nt/grc20/v0/token.gno:34 | read fqname.Construct and grc20reg.Register's HasPrefix guard, then register a token and print the key | examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno:43 if !strings.HasPrefix(token.ID(), key+".") | tests/solo-subrealm-registry-key.gno |  |

## Completeness

**Did every file in the diff get read?** Yes, all five, each with the code its
comments describe. `git diff --stat 5d7d8f5f9..37e2ce18b` is
`examples/gno.land/p/nt/grc20/v0/tellers.gno` 2 lines,
`examples/gno.land/p/nt/grc20/v0/token.gno` 11,
`examples/gno.land/p/nt/grc721/v0/tellers.gno` 2,
`examples/gno.land/p/nt/grc721/v0/token.gno` 2 and
`examples/gno.land/p/nt/grc721/v0/types.gno` 2: 11 insertions, 8 deletions, no
code line among them.

**Which angles ran?** One agent carried all of them over 19 changed lines, as
finder, judge and writer. Claims was the load-bearing one: each of the three
corrections was taken back to the code it describes, the separator to the two
`subRealmSep` definitions, the id to `NewToken`'s construction and a printed
`Token.ID()`, the registry key to `fqname.Construct` and `Register`'s prefix
guard. Removed swept the class by shape rather than by name, `grep -rn
'":subpath"'` and `grep -rn 'origRealm'` over the whole tree, which is what
turned up the two siblings the diff left stale. Reach followed `Token.ID()` to
its one consumer outside the package, `r/nt/grc20reg/v0`, and that is where the
Warning is. Lines and catalog had no material: the diff changes no statement.

**What was not run, and why:**

- No mutation pass. The diff adds and changes no test, so there is nothing to
  redden.
- No merge-base run of the two filetests. `r/nt/grc20reg/v0` and `gnovm/` are
  byte-identical between `5d7d8f5f9` and `37e2ce18b`, `git diff --stat` over
  both paths being empty, so the base cannot behave differently.
- Rows 2 and 3 are read-settled, per the rule that a Nit is read and never run.

**What is outstanding:** the Warning is on `grc20reg.Register`, which this diff
does not touch, so it is anchored on the sentence in `token.gno` that documents
the key relation and carries the `Related:` opener. A fix belongs in
`grc20reg.gno`: stripping the subpath there also needs the prefix guard
reworked, since `Token.ID()` keeps the raw creation path on purpose. Whether a
sub identity should register as its host is the author's call, and the round
takes no position on it beyond naming the two directions that break today.
