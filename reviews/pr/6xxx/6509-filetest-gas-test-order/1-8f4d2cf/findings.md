# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 0 refuted kept out

## Body

- Suggestion: gnovm/pkg/gnolang/files_test.go:128 [gh](https://github.com/gnolang/gno/blob/8f4d2cfa37a865f98e3b096cc6e27a2c58f5919c/gnovm/pkg/gnolang/files_test.go#L128), outside the diff, so no anchor
  State: CONFIRMED, band: Suggestion, angle: removed
  TL;DR: the alloc_* fresh-store carve-out in TestFiles is no longer needed: the alloc_* goldens that are not _long pass on pooled stores at head and fail at the merge base
  Check: Drop the `strings.HasPrefix(subTestName, "alloc")` clause at files_test.go:128. Run `go test ./gnovm/pkg/gnolang -run 'TestFiles' -count=3` at head. Green means the carve-out is dead. Run the same at base to confirm it was needed there.
  Details: With the clause removed, alloc_* filetests draw pooled stores. Run as a filtered set, the case the files_test.go comment says always diverges, head passes three times and the merge base fails at least ten alloc_* tests. Each alloc_* test still builds a whole store at head for nothing. Run filtered to TestFiles/alloc, not the whole suite.
  Evidence: sed 's/if isLong || strings.HasPrefix(subTestName, "alloc") {/if isLong {/' files_test.go; go test -count=1 -run 'TestFiles/alloc' .: head ok 18.9s; base FAIL alloc_4, alloc_defer_gc, alloc_7, alloc_7a, alloc_6a, alloc_6, alloc_3, alloc_5, alloc_1, alloc_10c (first 10 shown). Head -count=3: ok 57.4s.
- Suggestion: gnovm/pkg/test/filetest.go:471 [gh](https://github.com/gnolang/gno/blob/8f4d2cfa37a865f98e3b096cc6e27a2c58f5919c/gnovm/pkg/test/filetest.go#L471), outside the diff, so no anchor
  State: CONFIRMED, band: Suggestion, angle: lines
  TL;DR: realm filetests preprocess on a gasless forked allocator, so the preprocessing charge the PR adds skips every PKGPATH under /r/
  Check: Write a filetest with `// PKGPATH: gno.land/r/test/x` whose main has one `make([]byte, 10)` site, then a copy with two sites, and RunFiletest both. Do the same pair with no PKGPATH. Expected: the main pair's per-site gas delta includes the 113 preprocessing charge and the realm pair's does not. Add `txs.SetAllocator(m.Alloc)` after filetest.go:471 and the realm delta should gain it.
  Details: runTest replaces m.Store with m.Store.BeginTransaction(nil, nil, nil, nil) at filetest.go:471; that store's allocator is ds.alloc.Fork().Reset() (store.go:273), which copies maxBytes and bytes and no gas meter (alloc.go:381-389). The make site's preprocessing copy is charged in a main filetest and not in a realm one. Realm filetests were uncharged at the merge base too, so this predates the branch; the branch's parity claim covers main-package filetests only. Check run with a literal-vs-make pair rather than one-vs-two make sites, after a stdlib warm-up load.
  Evidence: tests/judge-2-realm-preprocess-gas_test.go, go test -run TestJudge2RealmGas -v in gnovm/pkg/test: head main literal=4172 make=7097 delta=2925, realm literal=9222 make=12022 delta=2800; base main delta=2812, realm delta=2800. The realm delta is the same at head and base, and 125 short of the head main delta.
  Artifact: tests/judge-2-realm-preprocess-gas_test.go

## gnovm/pkg/test/imports.go:248 [gh](https://github.com/gnolang/gno/blob/8f4d2cfa37a865f98e3b096cc6e27a2c58f5919c/gnovm/pkg/test/imports.go#L248) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: gno run's machine and its store stop sharing an allocator, so store-side allocations leave the 500 MB cap and the GC
Check: In both worktrees, add `fmt.Fprintln(os.Stderr, m.Store.GetAllocator() == m.Alloc)` after the machine is built at run.go:240. Run `go run ./gnovm/cmd/gno run x.gno` on a main-package file, and again with `-pkgpath gno.land/r/demo/x`. Expected: base prints true both times, head prints false both times.
Details: machine.go:189-190 installs the machine's allocator on the store only when store.GetAllocator() == nil. StoreWithOptions now gives the root store a MaxInt64 allocator, and BeginTransaction forks it (store.go:273), so the install never fires for gno run (run.go:229-239, MaxAllocBytes 500_000_000) in either mode, nor for gno repl on the root store. Object loads and preprocessing copies then count against a separate MaxInt64 allocator with no gas meter and no GC hook. Practical size is small for gno run (non-realm mode persists nothing), so Suggestion.
Evidence: run.go patched to print `SHARED m.Store.GetAllocator() == m.Alloc`, GNOROOT set to the tree, go1.27.3: head `SHARED false` for plain and `-pkgpath gno.land/r/demo/x`; base `SHARED true` for both.
