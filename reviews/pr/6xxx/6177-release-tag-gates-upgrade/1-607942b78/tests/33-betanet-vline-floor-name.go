// Run: from a gnolang/gno clone:
//   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
//   patch -p1 gno.land/pkg/gnoland/node_params_version_test.go <<'PATCH'
// (see the two added lines below, inserted after the
//  "betanet does not satisfy a v-line floor" case)
//   go test -run TestMeetsMinVersion ./gno.land/pkg/gnoland/ -v
//
// Asserts: the case named "betanet does not satisfy a v-line floor"
// (node_params_version_test.go:109) pins arithmetic ordering, not shape
// isolation between the two tag namespaces. Adding two cases with the same
// intent — a legacy binary against a v-line floor — but where the legacy
// side numerically outranks the floor, both FAIL at head 607942b78.
//
// Added cases (insert after line 109, "betanet does not satisfy a
// v-line floor"):
//   {"betanet meets an equal v floor", "chain/gnoland1.1", "v1.1.0", false},
//   {"legacy major beats v floor", "chain/gnoland2.0", "v1.9.9", false},
//
// Observed output at head:
//   --- FAIL: TestMeetsMinVersion/legacy_major_beats_v_floor (0.00s)
//   --- FAIL: TestMeetsMinVersion/betanet_meets_an_equal_v_floor (0.00s)
//   --- PASS: TestMeetsMinVersion/betanet_does_not_satisfy_a_v-line_floor (0.00s)
//   FAIL	github.com/gnolang/gno/gno.land/pkg/gnoland	0.091s
package gnoland_test
