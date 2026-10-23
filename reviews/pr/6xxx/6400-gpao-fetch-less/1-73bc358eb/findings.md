# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 1 refuted kept out

## Body

- Nit: contribs/gpao/README.md:190 [gh](https://github.com/gnolang/gno/blob/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4/contribs/gpao/README.md#L190), outside the diff, so no anchor
  State: CONFIRMED, band: Nit, angle: removed
  TL;DR: the README says any file vm/qfile cannot serve leaves the importer pending, while fetch now skips every file but production sources and the module file
  Check: sed -n 188,191p contribs/gpao/README.md, then read TestUnservedLicenceDoesNotHoldUpImporters in parkeddep_test.go and readByVerify in getter.go
  Details: README.md:190 reads 'and one whose files `vm/qfile` cannot serve leaves the package pending as unavailable'; readByVerify (getter.go:219-224) never queries a LICENCE, and TestUnservedLicenceDoesNotHoldUpImporters asserts statusApproved (parkeddep_test.go:284, 302). Line 190 sits outside the diff's hunks.
  Evidence: README.md:190 as quoted; parkeddep_test.go:302 assert.Equal(t, statusApproved, st.Status, ...)

## contribs/gpao/getter.go:220 [gh](https://github.com/gnolang/gno/blob/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4/contribs/gpao/getter.go#L220) · Missing test
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: no test serves a dependency whose module file is gno.mod, and dropping gno.mod from readByVerify turns that importer into a rejection with the suite green
Check: delete "gno.mod" from readByVerify's case and run go test -count=1 -run 'TestRPCGetter|TestVerifier|TestPackageName|TestOracle|TestPrepare' . in contribs/gpao; probe tests/judge-1002-gnomod-dep_test.go both ways
Details: A live dependency carrying gno.mod in place of gnomod.toml verifies with the case as written; with "gno.mod" removed, fetch hands the typecheck the dependency without a module file and TCLatestStrict panics in typeCheckMemPackage, a rejection of the importer. grep finds no gno.mod in contribs/gpao/*_test.go.
Evidence: case kept: prepare <nil>, verifyPackage <nil>; case "gnomod.toml" only: verifyPackage: verification panicked: gnomod.toml not found for package "gno.land/p/test/legacy"; suite under the mutation: ok github.com/gnolang/gno/contribs/gpao 5.184s
Artifact: tests/judge-1002-gnomod-dep_test.go
State: CONFIRMED, band: Missing test, angle: lines
TL;DR: the gno.mod branch of readByVerify can be deleted with every test green; same defect and same edit as candidate 1002
Check: grep -n 'gno.mod' contribs/gpao/*_test.go; mutate the case to `case "gnomod.toml":` and run go test ./contribs/gpao
Details: Same mutation run as 1002: the suite stays ok with "gno.mod" removed, and the probe shows the importer then rejected. One section with 1002.
Evidence: grep printed no hit; mutated suite: ok github.com/gnolang/gno/contribs/gpao 5.184s
Artifact: tests/judge-1002-gnomod-dep_test.go

## contribs/gpao/getter.go:192 [gh](https://github.com/gnolang/gno/blob/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4/contribs/gpao/getter.go#L192) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog
TL;DR: a live dependency holding only test files, a shape stored before AddPackage required a production file, is fetched with an empty name, so its importer goes pending blamed on the oracle's tree where the merge base rejected it
Check: tests/judge-1-onlytests-dep_test.go at head and at merge base 71169cb6e: a live dependency holding gnomod.toml and onlytests_test.gno only, imported by the candidate
Details: readByVerify drops the _test.gno, files is [gnomod.toml], packageName returns "", and buildOne's AddMemPackage refuses the nameless package; prepare returns the no-verdict 'could not prebuild' error. At the merge base the test file named the package, prepare succeeded and the ProdOnly typecheck rejected the importer. Reach is limited to user packages stored before hasProdGnoFile (412ab1962, #5892); no example package has the shape.
Evidence: head: prepare: could not prebuild "gno.land/p/x/onlytests", imported by "gno.land/p/test/user": invalid mempackage: invalid package name "" (... check this oracle's tree); base: prepare: <nil>, verifyPackage: gno.land/p/test/user/user.gno:5:19: undefined: onlytests.X
Artifact: tests/judge-1-onlytests-dep_test.go
