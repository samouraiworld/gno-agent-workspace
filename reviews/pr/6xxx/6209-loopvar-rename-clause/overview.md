# `initStaticBlocks1`: a statement's own names enter the loop-variable frame at `TRANS_BLOCK2`
Written by claude-opus-5-5.
PR: [gnolang/gno#6209](https://github.com/gnolang/gno/pull/6209)

## TLDR

`initStaticBlocks1` renames every name a `for` init declares to `<name>.loopvar` and renames each reference to it in the loop. It changed its frame of renameable names at `TRANS_BLOCK`, before the statement's first clause was walked, so `for x := x + 1`, `for x := range xs(x)` and `switch x := x.(type)` resolved their clause against the name they were declaring. The walker now fires `TRANS_BLOCK2` on `ForStmt` after `.Init` and on `RangeStmt` after `.X`, as it already did on `SwitchStmt`, and the pass applies each statement's declared names there. `for k = range` inside a loop declaring `k` also renames `k`, since a range key is a declaration only under `:=`.

## Before and after

All four rows are inside an enclosing `for x := ...` (or `for k := ...`).

| Program | Before | After |
| --- | --- | --- |
| `for x := x + 1; ...` | `name x.loopvar not declared`, or the enclosing loop's `x` when a plain `x` sits between | the `x` in scope before the statement, through the `ForStmt` case at `TRANS_BLOCK2` |
| `for x := range make([]int, x)` | `name x not declared`, or an outer plain `x` | the enclosing loop's `x`, through the `RangeStmt` case at `TRANS_BLOCK2` |
| `switch x := x.(type)` | `name x not declared`, or an outer plain `x` | the enclosing loop's `x`, through the `SwitchStmt` case at `TRANS_BLOCK2` |
| `for k = range ...` | `name k not declared`, or writes an outer plain `k` | writes the loop's `k`, through `isLoopvarDeclSite` |

## Read the code in this order

### 1. `hot` · The frame update after the clause, [`preprocess.go`](https://github.com/gnolang/gno/pull/6209/files#diff-ebdb0494324ba82339ca88219105656407e3f3a49240cf32c24c9a266d31e0f4R377-R418)
Adds the `for` init names, or deletes the range key and value and the type-switch variable, once the clause Go resolves outside them has been walked.
**If wrong:** a clause reads the variable its own statement declares, or a loop body reads an outer variable of the same name.
```go
		case TRANS_BLOCK2:
			// After the clauses resolved before the statement's own
			// declared names exist, before the children that see them.
			switch n := n.(type) {
			case *ForStmt:
				// After .Init: rename the declared names and bring them
				// into scope for cond, post and body.
				fsinit, ok := n.Init.(*AssignStmt)
				if !ok || fsinit.Op != DEFINE {
					break
				}
				for _, lx := range fsinit.Lhs {
					nx := lx.(*NameExpr)
					ln := nx.Name
					if ln == blankIdentifier {
						continue
					}
					if strings.HasSuffix(string(ln), ".loopvar") {
						continue // already renamed (initStaticBlocks runs twice)
					}
					nx.Name += ".loopvar"
					ownFrame()[ln] = true
				}
			case *RangeStmt:
				// After .X: the key/value shadow matching names in the body.
				if n.Op != DEFINE {
					break
				}
				f := top()
				if kx, ok := n.Key.(*NameExpr); ok {
					delete(f, kx.Name)
				}
				if vx, ok := n.Value.(*NameExpr); ok {
					delete(f, vx.Name)
				}
			case *SwitchStmt:
				// After .Init and .X: the type switch's VarName shadows a
				// matching name in the clauses.
				if n.IsTypeSwitch && n.VarName != "" {
					delete(top(), n.VarName)
				}
			}
```

### 2. `warm` · The new `TRANS_BLOCK2` calls, [`transcribe.go`](https://github.com/gnolang/gno/pull/6209/files#diff-bc1a23c4a14b14449bf9acb8f6a6ecb52a53525806d01db65a4ac21f8f26887fR424-R431) and [`transcribe.go`](https://github.com/gnolang/gno/pull/6209/files#diff-bc1a23c4a14b14449bf9acb8f6a6ecb52a53525806d01db65a4ac21f8f26887fR517-R524)
Calls back once between `ForStmt.Init` and `.Cond`, and once between `RangeStmt.X` and `.Key`.
**If wrong:** step 1 never runs for `for` and `range`, and every loop variable stays unrenamed.
```go
		// NOTE: special block case for after .Init.
		cnn2, c2 = t(ns, ftype, index, cnn, TRANS_BLOCK2)
		if stopOrSkip(nc, c2) {
			nn = cnn2
			return
		} else {
			cnn = cnn2.(*ForStmt)
		}
```

### 3. `hot` · Declaration sites, [`preprocess.go`](https://github.com/gnolang/gno/pull/6209/files#diff-ebdb0494324ba82339ca88219105656407e3f3a49240cf32c24c9a266d31e0f4R458-R468)
Leaves a name unrenamed only where it is declared: a `var` name, or a range key, value or assignment target under `:=`.
**If wrong:** `for k = range` writes a variable the loop does not own.
```go
func isLoopvarDeclSite(ns []Node, ftype TransField) bool {
	switch ftype {
	case TRANS_VAR_NAME:
		return true
	case TRANS_RANGE_KEY, TRANS_RANGE_VALUE:
		return ns[len(ns)-1].(*RangeStmt).Op == DEFINE
	case TRANS_ASSIGN_LHS:
		return ns[len(ns)-1].(*AssignStmt).Op == DEFINE
	}
	return false
}
```

## Words used here

| Name | What it is |
| --- | --- |
| `initStaticBlocks1` | The pass in `preprocess.go` that renames loop variables to `<name>.loopvar`; it runs twice per file, from `PredefineFileSet` and from `Preprocess`. |
| `TRANS_BLOCK2` | The `Transcribe` stage fired on `SwitchStmt`, `ForStmt` and `RangeStmt` after the clause resolved before their own names exist. |
| `pushClone` | Pushes a copy of the enclosing frame of renameable names, or nothing when that frame is empty and the caller adds nothing. |
| `ownFrame` | Returns the innermost scope's frame for adding a name, pushing one first when `pushClone` skipped it. |
| `isLoopvarDeclSite` | Reports whether a `NameExpr` declares a name rather than reading one. |
