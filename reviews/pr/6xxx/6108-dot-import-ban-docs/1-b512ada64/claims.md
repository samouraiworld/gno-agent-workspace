# Claims: gnolang/gno#6108 round 1, b512ada64, claude-opus-5-5, solo review

Round shape: solo round, one agent, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Nit | gnovm/pkg/gnolang/preprocess.go:5605 | tests/solo-trypredefine-unreachable.sh: sentinel at preprocess.go:5606 keeps TestFiles/import2.gno green, sentinel at :491 turns it red, base preprocess.go passes | sentinel at :5606 -> ok; sentinel at :491 -> +main/import2.gno:3:8-19: SENTINEL-isb2, FAIL; merge-base preprocess.go with the new filetest -> ok | tests/solo-trypredefine-unreachable.sh | hot |

Hit rate per tier, from the rows above: hot 1/1 confirmed over 1 files, warm 0/0 confirmed over 0 files, cold 0/0 confirmed over 3 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| reach | gnovm/tests/backup/import2.gno:1 | deleting gnovm/tests/backup/import2.gno could drop a test some runner reads | files_test.go:69 dir := "../../tests/files" is the only filetest root; gno fmt reads gnovm/tests/files (fmt.go:243); grep for tests/backup over *.go, Makefiles and .github: 0 hits |
| removed | gnovm/pkg/gnolang/preprocess.go:5606 | a test or golden still asserts the old capitalised 'dot imports not allowed in Gno' | grep -rni 'dot imports\? not allowed' at the head: preprocess.go:491, :5605, :5606 and tests/files/import2.gno:10, all lowercase 'gno'; no hit for the capitalised form |
| claims | docs/resources/go-gno-compatibility.md:51 | the doc's rejection claim or the golden's message might not match what the VM prints | go test ./gnovm/pkg/gnolang -run 'TestFiles/^import2.gno$' at b512ada64: ok, golden main/import2.gno:3:8-19: dot imports not allowed in gno; grep 'import \. ' over every .gno: only tests/files/import2.gno |

## Completeness

- Settled questions. The doc's claim holds: both sites reject a dot import, and the one that fires is `initStaticBlocks2` at `preprocess.go:491`, whose text matches the golden `main/import2.gno:3:8-19: dot imports not allowed in gno` (`go test ./gnovm/pkg/gnolang -run 'TestFiles/^import2.gno$'`: ok at b512ada64). No runner reads `gnovm/tests/backup`: `files_test.go:69` reads `../../tests/files` alone, `gno fmt` reads `gnovm/tests/files` (`fmt.go:243`), and a grep for `tests/backup` over Go files, Makefiles and `.github` returns nothing. No file at the head carries the capitalised `dot imports not allowed in Gno`.
- Catalog. Only *Type-check & preprocess* is touched: preprocessing still rejects the construct, measured above. The panic runs inside `TranscribeB` under `doRecover`, so it surfaces as a positioned preprocess error, and an `// Error:` directive is the right harness for a preprocess-time rejection; *VM-fault recoverability* covers runtime faults and does not apply. No other class is touched.
- Tests. The new filetest goes red with a sentinel in the `initStaticBlocks2` message and stays green with one in `tryPredefine` (`tests/solo-trypredefine-unreachable.sh`), which is candidate 1.
- Siblings. `import \. ` matches no other `.gno` file in the tree. The other 154 files under `gnovm/tests/backup` stay unread by any runner; none carries a dot import.
- Extremes. `import . "does/not/exist"` reaches the `nn == "."` check before `store.GetPackage` in `initStaticBlocks2` (diff lines 134 and 143), so it gets the same message; read, not run.
- Not returned. The rationale in the doc and in the new code comment is written as settled, while issue gnolang/gno#6076 offered it as unconfirmed and asked for a maintainer's word. That is doc wording and nothing the author changes in code, so it goes to the maintainers, not the draft.
