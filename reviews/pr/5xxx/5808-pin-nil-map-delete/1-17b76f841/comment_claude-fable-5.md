# Review: PR #5808
Posted: https://github.com/gnolang/gno/pull/5808#pullrequestreview-4478507686
Event: REQUEST_CHANGES

## Body
All five related filetests pass on the current head (17b76f841) and the pinned gc divergence reproduces on go1.26.4; the master merge inside this PR ([bf1467158](https://github.com/gnolang/gno/commit/bf1467158)) staled two of the pinned rationales.

- `cannot delete from readonly tainted map` ([`uverse.go:983-985`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/uverse.go#L983-L985) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/uverse.go#L983-L985)) is asserted by no test on this branch; the string greps only in `uverse.go` itself, since [`zrealm_map1.gno:32-33`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/tests/files/zrealm_map1.gno#L32-L33) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/zrealm_map1.gno#L32-L33) and [`zrealm_map3.gno:46-49`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/tests/files/zrealm_map3.gno#L46-L49) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/zrealm_map3.gno#L46-L49) now pin successful same-realm deletes. Fix: add a pinning test for the readonly-delete panic (fixture map global in [`crossrealm_b`](https://github.com/gnolang/gno/blob/17b76f841/examples/gno.land/r/tests/vm/crossrealm_b/crossrealm.gno#L37) · [↗](../../../../../.worktrees/gno-review-5808/examples/gno.land/r/tests/vm/crossrealm_b/crossrealm.gno#L37) plus a filetest, or a txtar), or state explicitly that the branch is intentionally unpinned.

<details><summary>filetest suite repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5808 -R gnolang/gno
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/^(delete1|zrealm_mapnil|map48|zrealm_map1|zrealm_map3)\.gno$' -v
```

```
--- PASS: TestFiles (1.42s)
    --- PASS: TestFiles/delete1.gno (0.00s)
    --- PASS: TestFiles/map48.gno (0.00s)
    --- PASS: TestFiles/zrealm_map1.gno (0.48s)
    --- PASS: TestFiles/zrealm_map3.gno (0.44s)
    --- PASS: TestFiles/zrealm_mapnil.gno (0.47s)
PASS
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	1.441s
```
</details>

<details><summary>gc oracle (go1.26.4)</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5808 -R gnolang/gno
cat > /tmp/gno5808_oracle.go <<'EOF'
package main

import "fmt"

func try(label string, f func()) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%s: panic: %v\n", label, r)
			return
		}
		fmt.Printf("%s: ok\n", label)
	}()
	f()
}

func main() {
	var m map[string]int
	var mi map[interface{}]int
	try("delete nil hashable", func() { delete(m, "k") })
	try("delete nil unhashable", func() { delete(mi, []int{1}) })
	try("delete nonnil unhashable", func() { delete(map[interface{}]int{"a": 1}, []int{1}) })
	try("read nil unhashable", func() { _ = mi[[]int{1}] })
}
EOF
go run /tmp/gno5808_oracle.go
rm /tmp/gno5808_oracle.go
```

```
delete nil hashable: ok
delete nil unhashable: panic: hash of unhashable type: []int
delete nonnil unhashable: panic: runtime error: hash of unhashable type []int
read nil unhashable: panic: hash of unhashable type: []int
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/review_claude-fable-5_davd-gzl.md · [↗](./review_claude-fable-5_davd-gzl.md)

*(AI Agent)*

## gnovm/tests/files/delete1.gno:20-24 [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/delete1.gno#L20)
The no-op pinned here for unhashable keys was justified by hashing being an unrecoverable VM abort, but on this head `delete` with a slice key on a non-nil map panics recoverably, so that constraint is gone for the exact case pinned. Fix: re-confirm no-op vs gc parity against the head's machinery (slice keys recoverable since [#5501](https://github.com/gnolang/gno/commit/326832e56); func and map keys still abort via the plain-panic default at [`values.go:1683-1686`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/values.go#L1683-L1686) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/values.go#L1683-L1686)) and extend this comment so the pin reads as a current choice, not a hard constraint.

<details><summary>repro</summary>

At this branch's pre-merge base the same operation was a plain Go panic ([`values.go:1647` at 59df7d868](https://github.com/gnolang/gno/blob/59df7d868/gnovm/pkg/gnolang/values.go#L1647)); on the head it is `panic(&Exception{...})` ([`values.go:1662`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/values.go#L1662) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/values.go#L1662)), which [`runOnce`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/machine.go#L1655-L1663) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/machine.go#L1655-L1663) converts into a panic `recover()` catches:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5808 -R gnolang/gno
cat > gnovm/tests/files/zz5808_recover.gno <<'EOF'
package main

func main() {
	mi := map[interface{}]int{"a": 1}
	func() {
		defer func() {
			println("recovered:", recover())
		}()
		delete(mi, []int{1})
	}()
	println("after, len:", len(mi))
}

// Output:
// recovered: runtime error: slice type cannot be used as map key
// after, len: 1
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/^zz5808_recover\.gno$' -v
rm gnovm/tests/files/zz5808_recover.gno
```

```
=== RUN   TestFiles/zz5808_recover.gno
--- PASS: TestFiles (0.05s)
    --- PASS: TestFiles/zz5808_recover.gno (0.00s)
PASS
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.067s
```

gc panics on the same deletes and on the nil-map read (oracle block in the review body). The nil-first no-op stays internally consistent with gno's nil-map reads, so keeping it is defensible; only the recorded constraint is gone.
</details>

*(AI Agent)*

## gnovm/tests/files/delete1.gno:13-19 [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/delete1.gno#L13)
No case exercises a key expression with side effects, so "key evaluated exactly once before the nil no-op" is claimed by this PR's verification but pinned nowhere. Fix: add a case `delete(m, key())` where `key()` prints, with a single "key evaluated" line in the Output block.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5808 -R gnolang/gno
cat > gnovm/tests/files/zz5808_key_once.gno <<'EOF'
package main

var m map[string]int

func key() string {
	println("key evaluated")
	return "k"
}

func main() {
	delete(m, key())
	println("done")
}

// Output:
// key evaluated
// done
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/^zz5808_key_once\.gno$' -v
rm gnovm/tests/files/zz5808_key_once.gno
```

```
=== RUN   TestFiles/zz5808_key_once.gno
--- PASS: TestFiles (0.04s)
    --- PASS: TestFiles/zz5808_key_once.gno (0.00s)
PASS
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.057s
```
</details>

*(AI Agent)*

## gnovm/tests/files/delete1.gno:22 [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/delete1.gno#L22)
The no-op is justified by "gno's own nil-map read behavior" with unhashable keys, and no filetest pins that read behavior either. Fix: pin the read in the same file, e.g. `v := mi[[]int{1}]` on the nil map printing zero values.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5808 -R gnolang/gno
cat > gnovm/tests/files/zz5808_read_unhashable.gno <<'EOF'
package main

func main() {
	var mi map[interface{}]int
	v := mi[[]int{1}]
	v2, ok := mi[[]int{2}]
	println("read:", v, v2, ok)
}

// Output:
// read: 0 0 false
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/^zz5808_read_unhashable\.gno$' -v
rm gnovm/tests/files/zz5808_read_unhashable.gno
```

```
=== RUN   TestFiles/zz5808_read_unhashable.gno
--- PASS: TestFiles (0.04s)
    --- PASS: TestFiles/zz5808_read_unhashable.gno (0.00s)
PASS
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.047s
```

gc panics `hash of unhashable type: []int` on the same read (oracle block in the review body).
</details>

*(AI Agent)*

## gnovm/tests/files/zrealm_mapnil.gno:9-11 [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/zrealm_mapnil.gno#L9)
The comment cites `TypedValue.IsReadonly`, which no longer exists on this branch: [#5747](https://github.com/gnolang/gno/commit/310dc2a04) (merged in via [bf1467158](https://github.com/gnolang/gno/commit/bf1467158)) removed it along with the taint bit. Fix: reword to the current mechanism, e.g. "a nil value is never readonly: Machine.IsReadonly ends in TypedValue.IsReadonlyBy, which returns false for non-object values".

The claim itself still holds through [`Machine.IsReadonly`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/machine.go#L2685) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/machine.go#L2685) → [`TypedValue.IsReadonlyBy`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/ownership.go#L461) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/ownership.go#L461), whose non-object default [returns false](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/ownership.go#L526-L527) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/ownership.go#L526-L527) for a nil `V`.

*(AI Agent)*
