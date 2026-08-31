# Review: [#6032](https://github.com/gnolang/gno/pull/6032)
Event: APPROVE

## Body
Looks good. `go test ./gno.land/pkg/integration/ -run 'TestTestdata/wugnot$'` stops at [`testdata_test.go:25`](https://github.com/gnolang/gno/blob/04c5133b7/gno.land/pkg/integration/testdata_test.go#L25) with `no testscript file found below <root>/examples` when `GNOROOT` names a checkout holding no `.txtar` under `examples/`, rather than reporting ok on a suite eleven scripts short.

## gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar:8 [gh](https://github.com/gnolang/gno/blob/04c5133b7/gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar#L8) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar#L8)
Missing test: txtar archives do not nest, so the `-- gnomod.toml --` line inside `local.txtar` closes that file for the outer parser, leaving this `cmp` on two identical 51-byte archives that hold no `.gno`. Quoting both nested archives and unquoting them in the script gives the pin something to fail on when [`txtarFilesInDir`](https://github.com/gnolang/gno/blob/04c5133b7/gnovm/cmd/gno/fix.go#L195) returns nothing.

<details><summary>test cases</summary>

```
# The nested archives are quoted, so the parser keeps each as one file.
unquote local.txtar local.txtar.golden

gno fix -fix stdsplit .
cmp main.gno main.gno.golden
cmp local.txtar local.txtar.golden

-- gnomod.toml --
module = "gno.land/r/demo/local"
gno = "0.9"
-- main.gno --
package main

import "std"

func main() {
	_ = std.Coin{}
}
-- main.gno.golden --
package main

import "chain"

func main() {
	_ = chain.Coin{}
}
-- local.txtar --
># package-local integration script
>
>gnoland start
>
>-- gnomod.toml --
>module = "gno.land/r/demo/local"
>gno = "0.9"
>-- local.gno --
>package local
>
>import "std"
>
>func Take() std.Coin { return std.Coin{} }
-- local.txtar.golden --
># package-local integration script
>
>gnoland start
>
>-- gnomod.toml --
>module = "gno.land/r/demo/local"
>gno = "0.9"
>-- local.gno --
>package local
>
>import "chain"
>
>func Take() chain.Coin { return chain.Coin{} }
```

That fixture reddens once the directory branch stops covering the script, which the committed one does not:

```
--- FAIL: Test_Scripts/fix/fix_dir_txtar (0.10s)
        > cmp local.txtar local.txtar.golden
        FAIL: testdata/fix/fix_dir_txtar.txtar:15: local.txtar and local.txtar.golden differ
```
</details>

## gno.land/pkg/integration/update_gas_wanted.sh:61 [gh](https://github.com/gnolang/gno/blob/04c5133b7/gno.land/pkg/integration/update_gas_wanted.sh#L61) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/update_gas_wanted.sh#L61)
Suggestion: this list comes from the repository's `examples/`, and step 2's run reads `$GNOROOT/examples` through [`gnoenv.GuessRootDir`](https://github.com/gnolang/gno/blob/04c5133b7/gnovm/pkg/gnoenv/gnoroot.go#L41), so a second checkout on `GNOROOT` gets the gas measured there and written back here by base name.

```suggestion
export GNOROOT="$REPO_ROOT"
find gno.land/pkg/integration/testdata examples \
```

## docs/resources/gno-testing.md:182 [gh](https://github.com/gnolang/gno/blob/04c5133b7/docs/resources/gno-testing.md?plain=1#L182) · [↗](../../../../../.worktrees/gno-review-6032/docs/resources/gno-testing.md#L182)
Nit: three bullets follow this line.

```suggestion
Three things to know:
```

## SKIP examples/Makefile:57
Already raised: https://github.com/gnolang/gno/pull/6032#discussion_r3894897100

Measured the same independently: `gno fix -diff -v .` from `examples/` exits 0 printing nothing, against eleven `.txtar` for `./...`. Not posted, gfanton has it on the ADR line.
