# getting-started#8: make targets, a Render test, AGENTS.md

`claude-opus-5`, high effort

## TLDR

[`gnolang/getting-started`](https://github.com/gnolang/getting-started) is the
five-minute first contact with Gno: one small realm, one test, a Makefile. This
change makes the Makefile the single place a newcomer and CI both run, adds a
test for `Render`, and writes the Go habits that break on the chain into a new
[`AGENTS.md`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/AGENTS.md?plain=1).

## What it is for

A realm is a program that lives on the gno.land chain, written in Gno, a
language close to Go. This repository holds exactly one, so a newcomer can read
all of it: [`hello.gno`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello.gno#L4)
keeps a `message` string,
[`Render`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello.gno#L7)
turns it into the page the chain's web front end serves,
[`Set`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello.gno#L20)
changes it, and
[`gnomod.toml`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/gnomod.toml#L1)
names the on-chain path it deploys to.

## Today, before the change

The [Makefile](https://github.com/gnolang/getting-started/blob/78866a30a32c0fc41775aea458569b3131a2fefa/Makefile#L1-L14)
carries three targets, `install`, `dev` and
[`test`](https://github.com/gnolang/getting-started/blob/78866a30a32c0fc41775aea458569b3131a2fefa/Makefile#L13-L14),
each spelling `gno` directly, and a bare `make` runs `install`. CI calls the
gno binary itself, [`gno test .`](https://github.com/gnolang/getting-started/blob/78866a30a32c0fc41775aea458569b3131a2fefa/.github/workflows/ci.yml#L26)
and [`gno lint .`](https://github.com/gnolang/getting-started/blob/78866a30a32c0fc41775aea458569b3131a2fefa/.github/workflows/ci.yml#L29),
so linting is something CI does and a contributor has no target for. The single
test exercises `Set` and `Get`; nothing calls `Render`. The README's
[Deploying it](https://github.com/gnolang/getting-started/blob/78866a30a32c0fc41775aea458569b3131a2fefa/README.md?plain=1#L44-L56)
section lists a testnet and mainnet with their endpoints and dates. There is no
file telling a coding agent how this repository works.

## What the change does, the after

The [Makefile](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/Makefile#L1-L28)
gains `lint` and `fmt`, routes every gno call through a
[`GNO ?= gno`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/Makefile#L3)
variable an environment can override, and makes a bare `make` print the target
list instead of installing a toolchain, via
[`.DEFAULT_GOAL := help`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/Makefile#L5)
and a `help` target that scrapes the `##` comment off each rule. CI switches to
[`make test`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/.github/workflows/ci.yml#L29)
and [`make lint`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/.github/workflows/ci.yml#L32),
so the local commands and the CI commands are one thing.
[`TestRender`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello_test.gno#L24-L29)
calls `Render("")` and asserts the current message appears in the output.
`AGENTS.md`, with a one-line
[`CLAUDE.md`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/CLAUDE.md?plain=1#L1)
pointing at it, lists the commands, two external tools, and
[five ways Gno differs from Go](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/AGENTS.md?plain=1#L38-L53).
The README picks up a testing section and replaces the per-network deploy list
with [one link](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/README.md?plain=1#L58-L65)
to the docs page that covers all of them.

## The concepts

- **Crossing call.** A realm function that writes state takes a first parameter
  `cur realm`, and a caller reaching into the realm writes `cross(cur)`. The
  chain uses that marker to tell a read from a write. `Set` and its call in
  [`hello_test.gno`](https://github.com/gnolang/getting-started/blob/dda245ddc4556d0b83e471b67830d17c0df7b666/hello_test.gno#L15)
  are the example the new file points at.
- **`Render`.** The one function the outside world calls on a realm, given a
  path and returning markdown. An untested `Render` fails first on-chain, which
  is the argument for the new test.
- **`gnodev`.** A throwaway local chain plus web UI, started by `make dev`,
  reloading the realm when a file is saved.
- **The standard library is a subset.** Gno ships part of Go's library and its
  own `ufmt` in place of `fmt`, so code that compiles in a Go editor can still
  fail or print something else on the chain.
