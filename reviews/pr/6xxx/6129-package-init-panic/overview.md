# A runtime panic in a package-level variable initializer
Written by claude-opus-5-5.

## TLDR

A package-level `var A = a[0]` that panics at runtime used to crash the Go process running the GnoVM with a nil pointer dereference. The change makes it end as an ordinary Gno unhandled panic that carries the real error.

## What it is for

Gno code can panic at runtime: an index out of range, a division by zero, a nil pointer. Inside a function, the VM unwinds the calls, runs deferred functions, and either recovers or stops with an unhandled-panic error whose text is the original message. A chain node turns that error into a failed transaction.

## How it works today

Before the change, `pushPanic` in `gnovm/pkg/gnolang/machine.go` built the exception and then read the last call frame's `LastException`, assuming a call frame exists. A package-level initializer runs with no call frame, so the lookup returned nil and the read crashed in Go:

```go
// before
fr := m.PopUntilLastCallFrame()
if m.Exception == nil {
	m.Exception = ex.WithPrevious(fr.LastException) // fr is nil during package init
}
```

## What the change does

With no frame, `pushPanic` stores the exception and stops at once through `makeUnhandledPanicError`, the terminal path `doOpPanic2` already takes when it finds no frame:

```go
// after
if fr == nil {
	panic(m.makeUnhandledPanicError())
}
```

Where the message ends up, per entry point, before and after:

| Entry point | Before | After |
| --- | --- | --- |
| Transaction adding a package (`AddPackage`), `MsgRun`, genesis | failed tx, error `<error: runtime.errorString>` | failed tx, error `runtime error: index out of range [0] with length 0` |
| `gno test` | `FAIL`, host nil dereference message | `FAIL`, the Gno message |
| `gno run` | Go panic, nil dereference, exit 2 | Go panic, the Gno message, exit 2 |

Three filetests under `gnovm/tests/files/var_init_panic_*.gno` cover an index, a division and a nil dereference, and an ADR records the decision.

## Concepts

- **Call frame**: the VM's record of an active function call. Unwinding a panic walks back to the last one to run its deferred functions.
- **`m.Exception`**: the panic in flight on a `Machine`, chained to earlier ones through `Previous`.
- **Unhandled panic error**: the Go value the VM panics with when no Gno code recovered. Node code recovers it and records it as the transaction's error.
