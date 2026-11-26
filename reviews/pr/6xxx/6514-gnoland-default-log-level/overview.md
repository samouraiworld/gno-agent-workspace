# `gnoland start` default `-log-level` moves from `debug` to `info`
Written by claude-opus-5-5.
PR: [gnolang/gno#6514](https://github.com/gnolang/gno/pull/6514)

## TLDR

A gno.land node started without a log level used to print every debug line, which on a node left running can fill a disk. This change makes the quieter `info` level the default, adds a test that keeps it there, and updates the operator README to match. Anyone who wants the old output passes `-log-level debug`.

## The `-log-level` flag

`gnoland start` registers `-log-level` in `startCfg.RegisterFlags` and hands its value to `log.InitializeZapLogger`, which parses it with `zapcore.ParseLevel` and builds the node's one logger. Every module of the node, consensus, p2p, the app and the VM keeper, logs through that logger, so the flag is the single switch for how much the whole node writes. The change edits only the flag's default value, at [`start.go`](https://github.com/gnolang/gno/pull/6514/files#diff-a723d42a46bf700d9a3ff5e6e2aba1fc7216477d25721f318d0975ff530c71c0R155), from `zapcore.DebugLevel.String()` to `zapcore.InfoLevel.String()`. An explicit `-log-level` on the command line wins as before.

## What each level keeps

zap drops every line below the configured level. At `debug` the node writes per-message consensus and peer chatter; at `info` it keeps startup notices such as `Halt height configured`, and every `Warn` and `Error` line. The events an operator acts on are not debug lines: a genesis tx that fails to deliver logs at `Error` in `gno.land/pkg/gnoland/app.go`, a peer dial failure at `Error` in `tm2/pkg/p2p/switch.go`, the halt-height notice at `Info`. So the quieter default hides volume, not failures.

## The test that pins the default

`TestStart_DefaultLogLevel`, at [`start_test.go`](https://github.com/gnolang/gno/pull/6514/files#diff-d9d703271a578c4c5dd5a141d8ac9a6b1b9c26dca52bc2034480b76a831cbc04R68-R79), registers the flags on a fresh `flag.FlagSet`, parses no arguments and asserts `cfg.logLevel` is `info`. It reads the same default the flag registration sets, so putting the default back to `debug` turns it red with `expected: "info", actual: "debug"`.

## Before and after

**Before:** `gnoland start` with no `-log-level` logged at `debug`, and the README warned operators to pass `-log-level info` by hand.

**After:** the same command logs at `info`; the README's [warning callout](https://github.com/gnolang/gno/pull/6514/files#diff-86616663ad2be202ad117ce93451b625786fcf7b47cf981f3716ad2c04609552R62-R66), its [flag table row](https://github.com/gnolang/gno/pull/6514/files#diff-86616663ad2be202ad117ce93451b625786fcf7b47cf981f3716ad2c04609552R76) and its [hardware note](https://github.com/gnolang/gno/pull/6514/files#diff-86616663ad2be202ad117ce93451b625786fcf7b47cf981f3716ad2c04609552R208-R209) now describe `info` as the default and warn against leaving `debug` on.

## What a user notices

A node started without `-log-level` writes far fewer lines, per process. Scripts that grep a node's output for a debug-only line stop matching; the deployment files and test harnesses in the tree that set a level already pass `info` or build their own logger.

## Words used here

| Name | What it is |
| --- | --- |
| `-log-level` | Flag of `gnoland start`: `debug`, `info`, `warn` or `error`; default `info` after this change. |
| `startCfg.RegisterFlags` | Registers every `gnoland start` flag and its default on a `flag.FlagSet`. |
| `log.InitializeZapLogger` | Builds the node's zap logger from the level and format strings. |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a doc.

### 1. `warm` · The default, [`gno.land/cmd/gnoland/start.go`](https://github.com/gnolang/gno/pull/6514/files#diff-a723d42a46bf700d9a3ff5e6e2aba1fc7216477d25721f318d0975ff530c71c0R155)
Sets the level a node logs at when no flag is given.
**If wrong:** a long-running node writes debug output until its disk fills.

### 2. `cold` · The guard, [`gno.land/cmd/gnoland/start_test.go`](https://github.com/gnolang/gno/pull/6514/files#diff-d9d703271a578c4c5dd5a141d8ac9a6b1b9c26dca52bc2034480b76a831cbc04R68-R79)
Fails if the default changes away from `info`.
**If wrong:** the default can drift back unnoticed.

### 3. `cold` · The operator docs, [`gno.land/cmd/gnoland/README.md`](https://github.com/gnolang/gno/pull/6514/files#diff-86616663ad2be202ad117ce93451b625786fcf7b47cf981f3716ad2c04609552R62-R66)
Tells operators the default and when to pass a level by hand.
**If wrong:** operators on older binaries keep the disk-filling default.

</details>
