# PR #5739: fix(gnovm): embedded type identity — struct field name + interface method-set flatten

URL: https://github.com/gnolang/gno/pull/5739
Author: ltzmaxwell | Base: master | Files: 29 | +1007 -92
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: `a3b14e207` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5739 a3b14e207`

Round 3 (head advanced `216e8aee3` → `a3b14e207`, PR content changed). Delta since round 2: a `/simplify` refactor of the flatten/stamp path, new cross-package and conflict filetests, the codec parity case for the new wire field, ADR expansion, and the `gno fmt` blank-line fix that was the only round-2 nit. Re-review focused on the refactor and the new tests.

**TL;DR:** When you embed a type by its name inside `struct{...}`, that name becomes the field's name; for `interface{...}`, embedding contributes only the embedded type's methods. This PR makes struct embeds keep the name exactly as written (so a type alias is its own field, distinct from the aliased type) and makes interface embeds collapse to their flattened method set (so `interface{ Stringer }` equals `interface{ Str() string }`), both matching the Go compiler. A per-method origin-package field keeps an unexported method from one package distinct from a same-named one elsewhere, so a type can't satisfy another package's sealed interface.

**Verdict: APPROVE** — the delta is a behavior-preserving refactor plus added coverage; the round-2 nit is fixed and CI is fully green. The refactor makes the flatten (slow) path and the direct-declaration (fast) path emit identical `FieldType`s for same-package unexported methods, removing the round-2 stamp divergence while keeping every TypeID. Multi-level cross-package identity, same-name-different-package coexistence, and the conflict rejection all match a side-by-side Go run.

## Summary
Round 2 stamped `FieldType.PkgPath` on every unexported method during flatten, including same-package ones, while the fast path (an interface with no embeds, passed through untouched) left them empty. Same TypeID either way, but the two paths produced different serialized `FieldType`s for the same logical interface. The round-3 refactor stamps only cross-package unexported methods ([`types.go:2739`](https://github.com/gnolang/gno/blob/a3b14e207/gnovm/pkg/gnolang/types.go#L2739) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L2739)); same-package methods stay unstamped and fall back to the enclosing package in [`idName`](https://github.com/gnolang/gno/blob/a3b14e207/gnovm/pkg/gnolang/types.go#L424) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L424), so both construction routes now agree byte-for-byte. The stamp lookup is hoisted into a shared [`originPkg`](https://github.com/gnolang/gno/blob/a3b14e207/gnovm/pkg/gnolang/types.go#L412) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L412) helper used by `idName` and [`VerifyImplementedBy`](https://github.com/gnolang/gno/blob/a3b14e207/gnovm/pkg/gnolang/types.go#L1112) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L1112), and the dead `FieldTypeList.Len` is dropped. Cross-package origin is still preserved across multiple hoists: a method's own stamp wins over the embed's package, so `ifaceext.sec` reached through `ifacemid.Mid` keeps origin `ifaceext`.

## Glossary
- TypeID: a type's canonical string identity; type equality and persisted on-chain state both key off it. For interfaces it folds in each method entry.
- flatten (interface): replacing an embedded-interface entry with its individual methods, so identity is the method set, not the embed name.
- sealed interface: an interface containing an unexported method, satisfiable only from the method's defining package.
- stamp: setting `FieldType.PkgPath` on a method to record its origin package, used when the method has been hoisted out of another package's interface.
- filetest: a VM-run `.gno` file asserted against `// Output:` golden directives.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None. The round-2 nit (missing `gno fmt` blank line in `alias2.gno`) is fixed at [`d3ab1e62`](https://github.com/gnolang/gno/commit/d3ab1e621196b4573248352adbc80fbf33645642); `main / build` and `gno-checks / fmt` are green.

## Missing Tests
None. The delta adds the missing coverage: `iface_embed_xpkg2` (multi-level `ifaceext → ifacemid → main`, origin preserved across two hoists), `iface_embed_conflict` (same-name conflicting-signature embed rejected with a positioned VM error and a matching go/types error), `iface_embed_provenance` (stamped vs unstamped construction yields one TypeID), a `TestFlattenInterfaceMethods` case pinning that same-package unexported methods stay unstamped on the slow path, and `parity_test.go` codec cases for the `FieldType.PkgPath` wire field.

## Suggestions
None.

## Verified
- Multi-level cross-package identity matches the Go compiler. Built the `ifaceext`/`ifacemid`/`ifaceext2` package set in Go and compared anonymous-interface `reflect.Type`s: `interface{ ifacemid.Mid } == interface{ ifaceext.Sec }` is true and `interface{ ifaceext.Sec } == interface{ ifaceext2.Sec }` is false, exactly what `iface_embed_xpkg2` and `iface_embed_xpkg` assert in the VM. CI runs the gno golden but does not run go/types side-by-side.
- The refactor is TypeID-preserving: `iface_embed_provenance`, `iface_embed_id`, `iface_embed_more`, `iface_embed_xpkg`, `alias2-4`, `alias6`, and the four `types_test.go` unit tests all pass at `a3b14e207`.

<details><summary>repro: Go interface-identity oracle</summary>

```bash
# multi-package Go oracle mirroring iface_embed_xpkg2 / iface_embed_xpkg
d=$(mktemp -d); cd "$d"
mkdir -p ifaceext ifacemid ifaceext2
printf 'module parity\n\ngo 1.21\n' > go.mod
printf 'package ifaceext\n\ntype Sec interface{ sec() int }\n' > ifaceext/ifaceext.go
printf 'package ifacemid\n\nimport "parity/ifaceext"\n\ntype Mid interface{ ifaceext.Sec }\n' > ifacemid/ifacemid.go
printf 'package ifaceext2\n\ntype Sec interface{ sec() int }\n' > ifaceext2/ifaceext2.go
cat > main.go <<'EOF'
package main

import (
	"reflect"

	"parity/ifaceext"
	"parity/ifaceext2"
	"parity/ifacemid"
)

func main() {
	midSec := reflect.TypeOf((*interface{ ifacemid.Mid })(nil)).Elem()
	sec := reflect.TypeOf((*interface{ ifaceext.Sec })(nil)).Elem()
	sec2 := reflect.TypeOf((*interface{ ifaceext2.Sec })(nil)).Elem()
	println("midSec==sec:", midSec == sec)
	println("sec==sec2:", sec == sec2)
}
EOF
go run .
cd /; rm -rf "$d"
```

```
midSec==sec: true
sec==sec2: false
```
</details>

## Open questions
- Legacy persisted interfaces that embed another interface keep their unflattened representation, so the embedded-interface branches in `FindEmbeddedFieldType`/`VerifyImplementedBy` exist only to serve decoded state until a chain-upgrade migration re-flattens it. Documented consensus break, gated behind the required upgrade, ADR records the migrate-then-drop sequencing. Not posted: author documented it.
- The flatten conflict at [`types.go:2745`](https://github.com/gnolang/gno/blob/a3b14e207/gnovm/pkg/gnolang/types.go#L2745) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L2745) is now confirmed reachable and clean: when go/types is not run, `flattenInterfaceMethods`'s panic surfaces as a positioned `PreprocessError` (`iface_embed_conflict.gno` asserts it via `// Error:`), so it is not an uncatchable crash. Resolves the round-2 open question. Not posted: behaves like Go's duplicate-method rejection.
