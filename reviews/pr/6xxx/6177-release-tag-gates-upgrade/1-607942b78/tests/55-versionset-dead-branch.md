# Candidate #55: dead branch in `VersionSet.CompatibleWith`

Asserts: inside the `semver.Major(pv1mm) == semver.Major(pv2mm)` branch at
`tm2/pkg/versionset/versionset.go:91-96`, the nested
`semver.Compare(semver.Major(pv1mm), semver.Major(pv2mm)) > 0` at line 92 is
`Compare(x, x) > 0`, always false, so the line-93 arm never runs. Pre-existing
(the diff never touches `versionset.go`), and reachable through the package's
only caller, `node_info.go:135`, which discards the returned `res` with
`if _, err := ...`.

Measured at reviewed head `607942b78fa4fdf6f378fecce32bc1d1d984ab8e`.

```bash
# from a local clone of gnolang/gno, at 607942b78fa4fdf6f378fecce32bc1d1d984ab8e:
go vet ./tm2/pkg/versionset/...
```
```
# tm2/pkg/versionset
tm2/pkg/versionset/versionset.go:94:6: unreachable code
```

Confirmed with a sentinel: inserting `panic("reached")` as the first
statement of the line-93 arm (line 92's `if` body) and running the package's
only exercising suites shows no panic, i.e. the arm is never entered by any
test in the tree:

```bash
go test ./tm2/pkg/versionset/... ./tm2/pkg/p2p/types/... ./tm2/pkg/bft/version/...
```
```
?   	github.com/gnolang/gno/tm2/pkg/versionset	[no test files]
ok  	github.com/gnolang/gno/tm2/pkg/p2p/types	0.119s
ok  	github.com/gnolang/gno/tm2/pkg/bft/version	0.016s
```

Only caller, confirmed to discard the result:

```bash
grep -rn 'CompatibleWith' --include=*.go . | grep -v _test.go
```
```
tm2/pkg/bft/version/version.go:20:     // negotiated with peers: VersionSet.CompatibleWith rejects a peer whose major
tm2/pkg/p2p/types/node_info.go:130:// CompatibleWith checks if two NodeInfo are compatible with each other.
tm2/pkg/p2p/types/node_info.go:133:func (info NodeInfo) CompatibleWith(other NodeInfo) error {
tm2/pkg/p2p/types/node_info.go:135:	if _, err := info.VersionSet.CompatibleWith(other.VersionSet); err != nil {
tm2/pkg/versionset/versionset.go:59:func (pvs VersionSet) CompatibleWith(other VersionSet) (res VersionSet, err error) {
tm2/pkg/p2p/types/node_info.go:135 is the only non-declaration, non-comment call site, and it discards `res`.
```

Confirmed behaviorally: the arm is dead, cost is zero today (result discarded), and this predates the diff — `versionset.go` itself is untouched by PR 6177.
