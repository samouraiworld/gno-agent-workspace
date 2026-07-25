# Contribution: PR [#5835](https://github.com/gnolang/gno/pull/5835) — `realm_only_gate` rule + regex gaps
Event: APPROVE
Posted: https://github.com/gnolang/gno/pull/5835#pullrequestreview-4780204840

## Body
New example proposition, from [#5976](https://github.com/gnolang/gno/pull/5976), that slips every rule:

```go
if caller.IsUserCall() { // "realms only"
	panic("realms only")
}
```

`maketx run` passes it. Case 2 of `gno-ai-contract-review.md` covers the mirror (payment: require `!IsUserCall()`), but a realms-only gate needs `IsUser()`, and neither the checklist nor the harness has it. Want a `realm_only_gate` rule plus that checklist line?

Follow-up: a `-scan` mode to run every rule against a realm dir, since the CLI only takes fixture yamls today:

```sh
auditpattern -scan /path/to/my/realm
```

Suggestion: the security material is spread across `gno-security-guide.md`, `gno-security.md`, and now `gno-ai-contract-review.md`, which overlap a lot. Worth consolidating to one deep reference plus the concise checklist, dropping the duplication. The harness rule comments and the contract-test corpus still point at `gno-security-guide.md`, not the concise guide.

Verified on 96cce07a2: ran all 10 rules against every example above; the four slipped cases get zero hits, and the file-scope and New-name cases fire on the safe fixtures shown.

After those fixes it looks good to merge to me. I can work on the follow-ups if needed.

## misc/audit-pattern-harness/internal/auditpattern/run.go:20 [↗](../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L20) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3651238718)
Exported methods returning a pointer slip: no receiver branch.

```go
type Store struct{ acct *Account }

// slips exported_pointer_leak (no receiver branch) and
// pkg_mutable_pointer (type is not *avl.Tree)
func (s *Store) Account() *Account {
	return s.acct
}
```

<details><summary>repro</summary>

```sh
cat > /tmp/rx.go <<'EOF'
package main

import (
	"fmt"
	"regexp"
)

func main() {
	exportedPointerFuncRE := regexp.MustCompile(`^func\s+([A-Z]\w*)\([^)]*\)\s+\*`)
	fmt.Println(exportedPointerFuncRE.MatchString(`func (s *Store) Account() *Account`))
}
EOF
go run /tmp/rx.go
```

```
false
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:36 [↗](../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L36) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3651238740)
Multi-return slips: the type must follow `)` directly.

```go
// returns the shared tree; not flagged
func Get() (*avl.Tree, error) {
	return gTree, nil
}
```

<details><summary>repro</summary>

```sh
cat > /tmp/rx.go <<'EOF'
package main

import (
	"fmt"
	"regexp"
)

func main() {
	pkgMutablePointerTypeRE := `\*avl\.Tree\b`
	pkgMutableReturnRE := regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?[A-Z]\w*\([^)]*\)\s+` + pkgMutablePointerTypeRE)
	fmt.Println(pkgMutableReturnRE.MatchString(`func Get() (*avl.Tree, error)`))
}
EOF
go run /tmp/rx.go
```

```
false
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:19 [↗](../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L19) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3651238783)
Nit: grouped `var (...)` blocks slip: `^var` required.

```go
var (
	// exported pointer to mutable state; slips the grouped-var check, and the
	// type is not *avl.Tree so pkg_mutable_pointer misses it too
	Shared *Account
)
```

<details><summary>repro</summary>

```sh
cat > /tmp/rx.go <<'EOF'
package main

import (
	"fmt"
	"regexp"
)

func main() {
	exportedPointerVarRE := regexp.MustCompile(`^var\s+[A-Z]\w*\s+\*`)
	fmt.Println(exportedPointerVarRE.MatchString(`Shared *Account`))
}
EOF
go run /tmp/rx.go
```

```
false
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:31 [↗](../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L31) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3651238813)
Suggestion: only `*avl.Tree` is matched; other mutable `/p/` pointer types slip.

```go
// mutable /p/ pointer via method; slips pkg_mutable_pointer (not *avl.Tree)
// and exported_pointer_leak (no receiver branch)
func (s *Store) Node() *avl.Node {
	return s.node
}
```

<details><summary>repro</summary>

```sh
cat > /tmp/rx.go <<'EOF'
package main

import (
	"fmt"
	"regexp"
)

func main() {
	pkgMutablePointerTypeRE := regexp.MustCompile(`\*avl\.Tree\b`)
	fmt.Println(pkgMutablePointerTypeRE.MatchString(`func (s *Store) Node() *avl.Node`))
}
EOF
go run /tmp/rx.go
```

```
false
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:201 [↗](../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L201) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3651238854)
Nit: `crossing` is file-wide, so a non-crossing helper's `PreviousRealm()` is flagged too.

```go
func Enter(cur realm) { /* crossing func in the same file */ }

// non-crossing helper; PreviousRealm() is the normal caller lookup
func who() string {
	return runtime.PreviousRealm().PkgPath() // flagged anyway
}
```

## misc/audit-pattern-harness/internal/auditpattern/run.go:438 [↗](../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L438) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3651238917)
Nit: the fresh-constructor skip needs a `New` name, so other constructors are flagged.

```go
// fresh pointer, safe; flagged only because the name is not New*
func MakeThing() *Thing {
	return &Thing{}
}
```
