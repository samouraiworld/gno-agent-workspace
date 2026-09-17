# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 3 refuted kept out

## examples/gno.land/p/nt/grc20/v0/token.gno:33 [gh](https://github.com/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e/examples/gno.land/p/nt/grc20/v0/token.gno#L33) · Warning
State: CONFIRMED, band: Warning, angle: claims
TL;DR: one realm registers the same symbol once per realm.Sub identity, and a sub-created token cannot be registered from its host frame
Check: two filetests in examples/gno.land/p/nt/grc20/v0/filetests, run with gno test -v ./gno.land/p/nt/grc20/v0: one registering the same symbol from two subs and the host, one registering a sub-created token from the host frame
Details: grc20reg.Register reads rlmPath from cur.Previous().PkgPath() (grc20reg.gno:39), which carries the "#sub" realm.Sub synthesizes, and keys with fqname.Construct(rlmPath, symbol). Nothing strips the subpath, so "host#a.SREG", "host#b.SREG" and "host.SREG" coexist for one realm, against the overwrite/alias guard Register's godoc states at grc20reg.gno:28-31. The HasPrefix guard at grc20reg.gno:43 compares against Token.ID(), which keeps the raw creation path, so the host frame is refused for a token its own realm created under a sub identity even though origRealm strips that subpath precisely to keep the token with the host.
Evidence: gno test -v ./gno.land/p/nt/grc20/v0: key a: gno.land/r/demo/grc20subdup#a.SREG, key b: gno.land/r/demo/grc20subdup#b.SREG, key c: gno.land/r/demo/grc20subdup.SREG, all three PASS; the host-frame filetest aborts with grc20reg: token must be registered from its own realm at grc20reg.gno:44
Artifact: tests/solo-subrealm-registry-key.gno

## examples/gno.land/p/nt/grc721/v0/token.gno:13 [gh](https://github.com/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e/examples/gno.land/p/nt/grc721/v0/token.gno#L13) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: the grc721 NewToken godoc and the construction comment 27 lines below it disagree about origRealm
Check: read examples/gno.land/p/nt/grc721/v0/token.gno lines 13 and 40-45 at the head
Details: Line 13 reads "its PkgPath becomes the unforgeable origRealm"; line 40, which this diff rewrote, says origRealm is the host path with the subpath stripped. The diff made the same correction in grc20's NewToken godoc and left grc721's.
Evidence: examples/gno.land/p/nt/grc721/v0/token.gno:13 against :40-45 at the head

## gno.land/adr/acd01fa29_grc20_caller_teller_home_binding.md:47 [gh](https://github.com/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e/gno.land/adr/acd01fa29_grc20_caller_teller_home_binding.md#L47) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: the ADR behind the home binding is the last place spelling the separator ":subpath"
Check: grep -rn '":subpath"' over the head worktree
Details: grep over the head worktree returns exactly one hit for ":subpath" after this diff, in the ADR the corrected comments describe.
Evidence: grep -rn '":subpath"' <head>: gno.land/adr/acd01fa29_grc20_caller_teller_home_binding.md:47
