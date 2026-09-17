Verdict: REQUEST CHANGES. The new `AGENTS.md` teaches that map iteration order in Gno is unspecified, which is the reverse of what the VM does, in the one file this repository offers as its Gno-versus-Go reference; the rest is nits on the doc's other claims and on the `GNO` override's coverage.
Event: REQUEST_CHANGES
Model: `claude-opus-5`, standard review, high effort
Commit: dda245ddc4556d0b83e471b67830d17c0df7b666
Overview: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/getting-started/8-make-targets-render-test/overview.md
Verify: findings run against gno master bbd9b2ffe, built from source with the toolchain gnolang/gno pins
Open the code: https://github.com/gnolang/getting-started/tree/dda245ddc4556d0b83e471b67830d17c0df7b666
Round: 1. 2 finders, no reflector, 11 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 1 refuted, none of them above Nit.

## Body

[`AGENTS.md`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/AGENTS.md?plain=1) is the only place in this repository where Gno's runtime behaviour is written down, so its bullets are read as fact and not as guidance.

## AGENTS.md:49-50 [gh](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/AGENTS.md?plain=1#L49-L50) · Warning

The premise beside this rule states the reverse: Gno ranges a map in [insertion order](https://github.com/gnolang/gno/blob/4598c267daf8318d6ca05532ed772dbf9a886208/docs/resources/gno-data-structures.md?plain=1#L84-L87) on every run.

```suggestion
- **Render must be deterministic.** Map iteration follows insertion order, which
  is not a guarantee to build output on, so never build output by ranging a map.
```

<details>
<summary>repro: the same key set ranges in insertion order on every run</summary>

```bash
# from a local clone of gnolang/getting-started:
cat > zz_map_order_test.gno <<'EOF'
package hello

import "testing"

func TestMapOrder(t *testing.T) {
	m := map[string]int{}
	for i, k := range []string{"z", "a", "m", "b", "q", "c", "y", "d"} {
		m[k] = i
	}
	out := ""
	for k := range m {
		out += k
	}
	t.Logf("range order = %s", out)
	if out != "zambqcyd" {
		t.Errorf("not insertion order: %s", out)
	}
}
EOF
gno test -v .
rm zz_map_order_test.gno
```

The assertion names the insertion order and holds, which is the finding: an unspecified order would not survive it.

```
# …
=== RUN   TestMapOrder
range order = zambqcyd
--- PASS: TestMapOrder (0.00s)
--- GAS:  603012
# …
ok      . 	0.53s
```

</details>

## hello_test.gno:24-27 [gh](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello_test.gno#L24-L27) · Missing test

Missing test: `TestRender` takes its expected value from `Get()`, the same variable `Render` prints, so it pins nothing about the page a newcomer first meets:

- [`TestSetAndGet`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello_test.gno#L15) sets the message to `Hello, Test!` before it runs, so the default page is never rendered in the suite.
- [`Set`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello.gno#L20) takes an empty message without a guard, and `strings.Contains` is true of every output once the needle is empty.

<details>
<summary>test cases</summary>

```go
func TestRenderDefault(cur realm, t *testing.T) {
	Set(cross(cur), "Hello, Gno!")
	out := Render("")
	if !strings.Contains(out, "Hello, Gno!") {
		t.Errorf("Render() should show the default message, got:\n%s", out)
	}
	if !strings.Contains(out, "gnokey add myaccount") {
		t.Errorf("Render() should show the gnokey instructions, got:\n%s", out)
	}
}
```

</details>

## AGENTS.md:43-44 [gh](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/AGENTS.md?plain=1#L43-L44) · Nit

Nit: [`Set`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello.gno#L20) spells its crossing parameter `_`, not the `cur realm` this sentence promises, so only the [test](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello_test.gno#L8) half of the pair shows one.

## AGENTS.md:45-46 [gh](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/AGENTS.md?plain=1#L45-L46) · Nit

Nit: [`Set`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello.gno#L20) and [`Get`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello.gno#L25) are exported too, and the rendered page itself [links a call to `Set`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello.gno#L16), a claim [the README](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/README.md?plain=1#L56) repeats.

```suggestion
- **`Render(path string) string` is the realm's rendered surface.** gnoweb
  calls it for every page view. Always keep a test that calls it. `Set` and
  `Get` are exported too, and anyone with a key can call them.
```

## AGENTS.md:53 [gh](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/AGENTS.md?plain=1#L53) · Nit

Nit: `ufmt.Sprintf("%-5s", "ab")` returns `(unhandled verb: %-)5s`, four characters longer than documented, and those four land in whatever the realm renders.

```suggestion
  and `%-5s` comes back as `(unhandled verb: %-)5s`.
```

<details>
<summary>repro: the documented value is a prefix of the real one</summary>

```bash
# from a local clone of gnolang/getting-started:
cat > zz_ufmt_test.gno <<'EOF'
package hello

import (
	"testing"

	"gno.land/p/nt/ufmt/v0"
)

func TestUfmtDash(t *testing.T) {
	t.Logf("%%-5s -> %q", ufmt.Sprintf("%-5s", "ab"))
	t.Logf("%%03d -> %q", ufmt.Sprintf("%03d", 7))
}
EOF
gno test -v .
rm zz_ufmt_test.gno
```

The `%-5s` line carries four characters the documented value drops; the `%03d` line is there to show the neighbouring claim is exact.

```
# …
=== RUN   TestUfmtDash
%-5s -> "(unhandled verb: %-)5s"
%03d -> "7"
--- PASS: TestUfmtDash (0.00s)
--- GAS:  1604805
# …
ok      . 	0.53s
```

</details>

## Makefile:19 [gh](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/Makefile#L19) · Nit

Nit: `dev` calls `gnodev` by name while [`GNO ?= gno`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/Makefile#L3) routes the other three, so a toolchain off `PATH` runs test, lint and fmt and fails [the local chain](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/AGENTS.md?plain=1#L17).

```suggestion
GNO ?= gno
GNODEV ?= gnodev
```

## AGENTS.md:23-25 [gh](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/AGENTS.md?plain=1#L23-L25) · Suggestion

Suggestion: [CI builds its gno from master at job time](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/.github/workflows/ci.yml#L22-L25) while [`GNO ?= gno`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/Makefile#L3) takes whatever the contributor installed last, so a green local run promises no green CI.

```suggestion
`make test lint` is the bar for any change. CI runs exactly those two targets
against a gno built from `gnolang/gno` master, so a green run on a freshly
installed toolchain means a green CI. Re-run `make install` when CI disagrees.
```

## README.md:60-62 [gh](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/README.md?plain=1#L60-L62) · Suggestion

Related suggestion: [`Render`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello.gno#L16) hardcodes `/r/example/hello$help&func=Set`, so a reader who re-points `module` ships a page whose one button calls a realm they do not own. `std.CurrentRealm().PkgPath()` makes that link follow the module.

## SKIP Makefile:9 [gh](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/Makefile#L9) · Nit

Nit: the [help recipe](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/Makefile#L9) spells `.*?` in `FS` and in its match pattern, a lazy quantifier POSIX ERE does not define, on the target [`.DEFAULT_GOAL`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/Makefile#L5) runs first.

Skipped: GNU awk folds it to a greedy `.*` and prints the listing correctly, `--posix` included, and no stricter awk was available here to test one that rejects it.
