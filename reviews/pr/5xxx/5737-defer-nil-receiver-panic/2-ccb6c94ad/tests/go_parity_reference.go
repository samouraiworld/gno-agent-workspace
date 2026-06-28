/* Run: from a gno checkout:
gh pr checkout 5737 -R gnolang/gno && git checkout ccb6c94ad
curl -fsSL -o /tmp/go_parity_reference.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/2-ccb6c94ad/tests/go_parity_reference.go
go run /tmp/go_parity_reference.go
*/

// Go reference for the interface-bound method-value timing the PR's filetests
// assert. Real Go materializes the concrete method + receiver inside the call;
// the gno filetests must print the same lines. Cyclic/nil-fatal cases are
// excluded (Go fatally stack-overflows / aborts, asserted as // Error: in gno).
package main

type T struct{ x int }

func (t T) Get() int    { return t.x }
func (t T) IncGet() int { t.x++; return t.x }

type I interface {
	Get() int
	IncGet() int
}

// --- timing ---
func timing() {
	c := &T{x: 1}
	cg := c.Get // concrete: receiver snapshot at bind
	c.x = 2
	println("concrete", cg())

	p := &T{x: 1}
	var i I = p
	h := i.Get // interface: deref at call
	p.x = 2
	println("snapshot", h())

	q := &T{x: 10}
	var j I = q
	g := j.IncGet
	println("percall", g(), g())
	println("caller", q.x)
}

// --- nil timing ---
type Tn struct{ x int }

func (t Tn) Get() int { return t.x }

type In interface{ Get() int }

type Mid struct{ Tn }

var ptn *Tn
var pmid *Mid

func concrete() (r int) {
	defer func() { recover() }()
	defer ptn.Get() // concrete nil: eager deref at bind
	r = 1
	return
}

func ifaceImmediate() (r int) {
	defer func() { recover() }()
	var i In = ptn
	g := i.Get
	r = 1
	_ = g()
	r = 2
	return
}

func embedded() (r int) {
	defer func() { recover() }()
	var i In = pmid
	defer i.Get()
	r = 1
	return
}

func nilTiming() {
	println("concrete", concrete())
	println("ifaceImmediate", ifaceImmediate())
	println("embedded", embedded())
}

// --- dynamic dispatch ---
type Deep struct{ x int }

func (d Deep) Get() int  { return d.x }
func (d *Deep) Ptr() int { return d.x }

type IG interface{ Get() int }
type IP interface{ Ptr() int }

type Impl struct{ x int }

func (i Impl) Get() int { return i.x }

type Other struct{}

func (Other) Get() int { return 2 }

type Box struct{ IG }
type MidP struct{ Deep }

var pmp *MidP

func redispatch() int {
	s := &Box{IG: Impl{x: 1}}
	var o IG = s
	g := o.Get
	s.IG = Other{}
	return g()
}

func c5() (r int) {
	defer func() { recover() }()
	var i IP = pmp
	defer i.Ptr()
	r = 1
	return
}

func reboxChain() int {
	var i1 IG = &Deep{x: 1}
	var i2 IG = i1
	g := i2.Get
	(i1.(*Deep)).x = 9
	return g()
}

func nestedUnwrap() int {
	p := &Deep{x: 1}
	var i IG = &Box{IG: p}
	g := i.Get
	p.x = 9
	return g()
}

func storedInMap() int {
	m := map[string]IG{"a": &Deep{x: 3}}
	g := m["a"].Get
	(m["a"].(*Deep)).x = 8
	return g()
}

func viaTypeAssert() int {
	var a interface{} = &Deep{x: 5}
	g := a.(IG).Get
	(a.(*Deep)).x = 9
	return g()
}

func dynamic() {
	println("redispatch", redispatch())
	println("c5", c5())
	println("rebox", reboxChain())
	println("nested", nestedUnwrap())
	println("map", storedInMap())
	println("assert", viaTypeAssert())
}

// --- deep converge ---
type Concrete struct{ v int }

func (c Concrete) Get() int { return c.v }

type MidC struct{ IG }
type OuterC struct{ IG }

func deepConverge() {
	var o IG = OuterC{IG: MidC{IG: Concrete{v: 9}}}
	g := o.Get
	println(g())
}

func main() {
	timing()
	nilTiming()
	dynamic()
	deepConverge()
}
