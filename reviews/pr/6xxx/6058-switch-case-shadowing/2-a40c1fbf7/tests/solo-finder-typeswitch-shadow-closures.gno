// Repro from a plain clone of gnolang/gno at a40c1fbf7 (PR 6058 head):
//   go install ./gnovm/cmd/gno && cp solo-finder-typeswitch-shadow-closures.gno /tmp/p.gno && cd /tmp && gno run p.gno
// Compare: cp solo-finder-typeswitch-shadow-closures.gno /tmp/p.go && go run /tmp/p.go (outputs must match).
// Type switch clause shadowing an init name and tsvar shadowing one, with closures capturing each binding and a loop. Head output matched Go: 5 t 42 / ts 7 / 0 0 / 2 10.

package main

func main() {
	var i any = 42
	switch v := 1; y := i.(type) {
	case int:
		f := func() int { return v }
		v = 5
		v := "s"
		g := func() string { return v }
		v = "t"
		println(f(), g(), y)
	case string, bool:
		const v = 3.5
		println(v, y)
	}
	switch v := 42; v := any(v).(type) {
	case int:
		h := func() int { return v }
		v = 7
		println("ts", h())
	case nil:
		println("nil", v)
	default:
		k := func() any { return v }
		println("def", k())
	}
	for j := 0; j < 2; j++ {
		switch v := j; w := any(v).(type) {
		case int:
			fs := func() int { return v + w }
			v := w * 10
			println(fs(), v)
		}
	}
}
