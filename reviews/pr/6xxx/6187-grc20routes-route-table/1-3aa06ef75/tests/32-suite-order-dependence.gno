// from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// cd examples/gno.land/r/nt/grc20routes/v0
// # Move TestKeyedProofCoversEveryTokenOnTheRealm (declared at line 183) to
// # above TestUnrecordedTokenFallsBackToCanonical (line 47) in
// # grc20routes_test.gno, keeping both bodies unchanged, then:
// cd ../../../../..  # back to examples/
// gno test -v ./gno.land/r/nt/grc20routes/v0
//
// Candidate #32 (grc20routes_test.gno:60): CONFIRMED behaviorally. Moving
// the proof test above the two convention tests flips both from pass to
// fail, with no guard anywhere catching the reorder:
//
//   === RUN   TestKeyedProofCoversEveryTokenOnTheRealm
//   --- PASS: TestKeyedProofCoversEveryTokenOnTheRealm (0.02s)
//   === RUN   TestUnrecordedTokenFallsBackToCanonical
//   --- FAIL: TestUnrecordedTokenFallsBackToCanonical (0.00s)
//   === RUN   TestJSONFallsBackAndNamesTheSource
//   --- FAIL: TestJSONFallsBackAndNamesTheSource (0.01s)
//
// The file's own comment above TestJSONFallsBackAndNamesTheSource says as
// much ("The convention cases therefore have to run before it — they are
// first in the file, and gno runs tests in declaration order.") but nothing
// enforces it: no t.Skip guard, no assertion that IsKeyedRealm(selfPath) is
// false before the convention cases run. The suite's correctness rests on
// declaration order plus shared realm globals (provenKeyed, declared,
// pending) that persist across tests in one package run.
