package gnolang_test

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

// vmPkgPath is the package declaring StringValue, TypedValue and Value.
const vmPkgPath = "github.com/gnolang/gno/gnovm/pkg/gnolang"

// auditedEqualitySites are the direct == / != sites on a string-carrying
// type that exist today and are safe. Each entry states why; a new site
// only belongs here with the same kind of proof.
var auditedEqualitySites = map[string]string{
	"nodes.go:2404":    "UNPROVEN — StaticBlock.Define2 re-definition guard; see finding B2",
	"op_binary.go:589": "isEql MapKind/SliceKind/FuncKind branch: kind dispatch cannot reach StringKind",
	"op_binary.go:601": "isEql PointerKind, both TVs are DataByteType, which holds DataByteValue",
	"op_binary.go:604": "isEql PointerKind: both .V are PointerValue",
	"values_string.go:34": "seenValues.IndexOf: Put is reached only from writeRefOrPut and " +
		"SliceValue.WriteProtected, with *ArrayValue/*StructValue/*MapValue/PointerValue/*SliceValue",
}

// TestNoDirectStringValueEquality enforces the invariant that the
// StringValue doc comment and gnovm/adr/pr6110_string_backing_id.md state
// in prose only:
//
//	"compare strings by Str, never by struct equality"
//	"New direct == on TypedValue/.V holding strings must not be introduced"
//
// StringValue is now a struct carrying a mint serial, so == on it, or on
// any TypedValue or Value that may hold one, answers false for two equal
// strings minted separately. .github/golangci.yml is `default: none` and
// none of its enabled linters (whitespace, unconvert, tparallel, thelper,
// predeclared, nolintlint, misspell, makezero, importas, govet, gosec,
// dogsled, errname, errorlint, unused, gomodguard, forbidigo, staticcheck)
// inspects a binary expression's operand types: forbidigo matches
// identifiers, and gocritic — whose ruleguard could express this — is
// configured under `settings` but absent from `enable`. A typed walk is
// the smallest thing that catches a reintroduction.
func TestNoDirectStringValueEquality(t *testing.T) {
	t.Parallel()

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes |
			packages.NeedTypesInfo | packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg, "github.com/gnolang/gno/gnovm/...")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	var checked int
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				be, ok := n.(*ast.BinaryExpr)
				if !ok || (be.Op.String() != "==" && be.Op.String() != "!=") {
					return true
				}
				checked++
				// `x == nil` is a nilness test, not a content compare.
				if isNilIdent(be.X) || isNilIdent(be.Y) {
					return true
				}
				// Both sides must be able to hold a StringValue for the
				// comparison to consult a mint ID.
				tx, ty := pkg.TypesInfo.TypeOf(be.X), pkg.TypesInfo.TypeOf(be.Y)
				if tx == nil || ty == nil || !stringCarrying(tx) || !stringCarrying(ty) {
					return true
				}
				pos := pkg.Fset.Position(be.Pos())
				key := fmt.Sprintf("%s:%d", filepath.Base(pos.Filename), pos.Line)
				if _, audited := auditedEqualitySites[key]; audited {
					return true
				}
				t.Errorf("%s: %s on %s compares StringValue.ID, not content. "+
					"Compare .Str (or go through GetString/ComputeMapKey); if the "+
					"operands provably never hold a StringValue, add %q to "+
					"auditedEqualitySites with the reason.", key, be.Op, tx, key)
				return true
			})
		}
	}
	if checked == 0 {
		t.Fatal("no comparisons walked: the loader resolved nothing")
	}
	t.Logf("walked %d comparisons across %d packages", checked, len(pkgs))
}

// stringCarrying reports whether a value of type tt can hold a
// StringValue: the struct itself, TypedValue (whose .V can), or Value.
func stringCarrying(tt types.Type) bool {
	if ptr, ok := tt.(*types.Pointer); ok {
		tt = ptr.Elem()
	}
	named, ok := tt.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj.Pkg() == nil || obj.Pkg().Path() != vmPkgPath {
		return false
	}
	switch obj.Name() {
	case "StringValue", "TypedValue", "Value":
		return true
	}
	return false
}

func isNilIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}
