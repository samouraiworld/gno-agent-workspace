/* Run: from any directory, with a Go 1.24+ toolchain:
mkdir -p /tmp/gopar5969 && cd /tmp/gopar5969 && go mod init gopar >/dev/null 2>&1
curl -fsSL -o go_parity_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5969-defined-pointer-type-semantics/1-7e0728bd5/tests/go_parity_test.go
go test -v -run TestDefinedPointerParity .
*/

// Each case is a standalone Go program compiled with `go build`; the test
// asserts the compiler's verdict for the six shapes PR 5969 touches, so the
// GnoVM filetests can be compared against a real Go run rather than memory.
package gopar

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type parityCase struct {
	name    string
	src     string
	wantErr string // "" means the program must compile
}

var cases = []parityCase{
	{
		name: "ptr_to_defined_pointer_field", // gno: ptr14.gno
		src: `package main
type D2 struct{ A int }
type D1 *D2
func main() { var p *D1; _ = p.A }`,
		wantErr: "p.A undefined",
	},
	{
		name: "defined_pointer_field", // gno: method50.gno
		src: `package main
type D2 struct{ A int }
type D1 *D2
func main() { d2 := D2{A: 5}; var x D1 = &d2; println(x.A) }`,
	},
	{
		name: "defined_pointer_method", // gno: method47.gno
		src: `package main
type D2 struct{}
func (D2) Foo() string { return "m" }
type D1 *D2
func main() { var x D1; _ = x.Foo }`,
		wantErr: "x.Foo undefined",
	},
	{
		name: "alias_of_pointer_embedded", // no gno filetest in the PR
		src: `package main
type D2 struct{ A int }
type P = *D2
type S struct{ P }
func main() { var s S; s.P = &D2{A: 3}; println(s.A) }`,
	},
	{
		name: "defined_pointer_embedded", // gno: struct64.gno
		src: `package main
type D2 struct{ A int }
type D1 *D2
type S struct{ D1 }
func main() { var s S; _ = s }`,
		wantErr: "embedded field type cannot be a pointer",
	},
	{
		name: "pointer_to_interface_embedded", // gno: struct65.gno
		src: `package main
type I interface{ M() string }
type S struct{ *I }
func main() { var s S; _ = s.M }`,
		wantErr: "embedded field type cannot be a pointer to an interface",
	},
	{
		name: "defined_pointer_satisfies_nothing", // gno: method51.gno
		src: `package main
type D2 struct{}
func (D2) Foo() string { return "m" }
type D1 *D2
type Fooer interface{ Foo() string }
func main() { d2 := D2{}; var x D1 = &d2; var f Fooer = x; println(f.Foo()) }`,
		wantErr: "does not implement Fooer",
	},
}

func TestDefinedPointerParity(t *testing.T) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(c.src), 0o600); err != nil {
				t.Fatal(err)
			}
			// A module file keeps `go build` out of the caller's module.
			mod := "module p\n\ngo 1.24\n"
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "out"), ".")
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			got := string(out)
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("expected the program to compile, got:\n%s", got)
			case c.wantErr != "" && err == nil:
				t.Fatalf("expected compile error %q, program compiled", c.wantErr)
			case c.wantErr != "" && !strings.Contains(got, c.wantErr):
				t.Fatalf("expected compile error %q, got:\n%s", c.wantErr, got)
			}
			t.Log(strings.TrimSpace(got))
		})
	}
}
