# Review: PR [#5981](https://github.com/gnolang/gno/pull/5981)
Event: COMMENT

## Body
`gno lint` reports both rejection paths as `gnoPreprocessError`, the `for` init one from the first static pass rather than [`Reserve`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/nodes.go#L2325).

## gnovm/pkg/gnolang/preprocess.go:299-304 [gh](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L299-L304) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L299)
Refactor: [`Reserve`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/nodes.go#L2325) sees this name as `iota.loopvar`, so trimming that suffix there covers the `for` init and this block comes out.

<details><summary>patch</summary>

```diff
--- a/gnovm/pkg/gnolang/nodes.go
+++ b/gnovm/pkg/gnolang/nodes.go
 func (sb *StaticBlock) Reserve(isConst bool, nx *NameExpr, origin Node, nstype NSType, index int) {
-	// iota is a non-shadowable builtin; reject binding it as a receiver,
-	// parameter, named result, type-switch guard, short-var-define, or
-	// range key/value name. (uverse's own "iota" registration goes through
-	// Define2 directly, bypassing Reserve, so it is unaffected.)
-	if nx.Name == iotaIdentifier {
-		panic(fmt.Sprintf("builtin identifiers cannot be shadowed: %s", nx.Name))
+	// iota is a non-shadowable builtin. A three-clause for init reaches here
+	// renamed to "iota.loopvar"; uverse's own registration goes through
+	// Define2, bypassing Reserve, so it is unaffected.
+	if name := Name(strings.TrimSuffix(string(nx.Name), ".loopvar")); name == iotaIdentifier {
+		panic(fmt.Sprintf("builtin identifiers cannot be shadowed: %s", name))
 	}

--- a/gnovm/pkg/gnolang/preprocess.go
+++ b/gnovm/pkg/gnolang/preprocess.go
 						if strings.HasSuffix(string(ln), ".loopvar") {
 							continue
 						}
-						// iota is a non-shadowable builtin. Reject it here,
-						// before it's renamed to "iota.loopvar" and slips
-						// past the Reserve() guard in initStaticBlocks2.
-						if ln == iotaIdentifier {
-							panic(fmt.Sprintf("builtin identifiers cannot be shadowed: %s", ln))
-						}
 						nx.Name += ".loopvar"
```
</details>

## gnovm/pkg/gnolang/preprocess.go:20 [gh](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L20) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L20)
Nit: [`def("iota", undefined)`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/uverse.go#L761) is the literal this constant should have replaced.

## SKIP gnovm/pkg/gnolang/nodes.go:2325 [gh](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/nodes.go#L2325) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2325)
Already raised: https://github.com/gnolang/gno/pull/5981#discussion_r3660033767

Master runs nine forms this head rejects, and node startup re-preprocesses every stored package with no per-package recover, so an on-chain package using one fails at boot. The open thread counts three forms. The head has four. Correction drafted at `thread_r3660033767_edit.md`.

## SKIP gnovm/pkg/gnolang/nodes.go:2326 [gh](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/nodes.go#L2326) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2326)
Already raised: https://github.com/gnolang/gno/pull/5981#discussion_r3660033770

Nit: the message says builtin identifiers cannot be shadowed, and `func f(len int) int { return len }` still compiles. Unchanged at this head.
