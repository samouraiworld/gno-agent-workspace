# PR #5494: feat(gnokey): Pure Go ledger support

URL: https://github.com/gnolang/gno/pull/5494
Author: clockworkgr | Base: master | Files: 15 | +232 -11
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5494 b6bb8efff` (then `gh -R gnolang/gno pr checkout 5494` inside it)

**Verdict: APPROVE** — replaces the CGO `zondax/hid` dep with a pure-Go shim (`misc/puregohid`) over `rafaelmartins/usbhid`, keeping the `zondax/hid` import path stable via `replace`; the binaries that ship ledger features (`gnokey`, `gnoland`, `gnoweb`, `gno`, `gnodev`, `gnokeykc`) now build with `CGO_ENABLED=0`; unsupported-platform stubs preserve the previous "no ledger" build-success behavior. Main caveat is the `replace` directive being applied to only 3 modules out of the ~10 sub-modules that transitively reference `github.com/zondax/hid`, leaving the rest on the CGO-disabled `zondax/hid` stub when built standalone — that is a downgrade vs upgrade gap, not a regression. The unrelated `TestNodeBootWithInitialHeight` CI failure passes locally and was introduced in master by an unrelated PR.

## Summary

`tm2/pkg/crypto/internal/ledger` uses `github.com/zondax/hid`, which requires CGO and the bundled `hidapi`/`libusb` C sources, blocking static cross-compilation of `gnokey`. This PR introduces a pure-Go shim package `misc/puregohid` with the same API surface as `zondax/hid` (`Supported`, `Enumerate`, `DeviceInfo.Open`, `Device.{Read,Write,Close}`, error sentinels), backed by `github.com/rafaelmartins/usbhid` on linux/darwin/windows and stubbed elsewhere. Root `go.mod` gets a `replace github.com/zondax/hid v0.9.2 => ./misc/puregohid`, so all root-module binaries pick up the shim transparently. `discover.go`'s "build with CGO_ENABLED=1" hint is corrected to "not available on this platform".

```
before:                                 after:
  ledger -> zondax/hid -> cgo/libusb     ledger -> zondax/hid (replace) ->
    (cross-compile broken)                 misc/puregohid -> rafaelmartins/usbhid
                                           (pure Go, static builds OK)
```

## Glossary

- `zondax/hid` — upstream CGO-based USB HID library; the public API the shim mimics.
- `rafaelmartins/usbhid` — pure-Go USB HID library (uses `ebitengine/purego` for darwin); the shim's backend.
- `misc/puregohid` — this PR's new local module; a thin adapter wearing the `zondax/hid` package path via `go.mod replace`.
- `DiscoverDefault` — `tm2/pkg/crypto/internal/ledger/discover.go` entry point; gates ledger discovery on `hid.Supported()`.
- `Interface` field — HID interface number on `DeviceInfo`; `ledger-go` uses it in a fallback product-ID match path.

## Fix

A new module at [`misc/puregohid/`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/) re-exports the `zondax/hid` API: [`hid.go`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/hid.go) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/hid.go) holds shared types (`DeviceInfo` is now plain exported data — no captured unexported `*usbhid.Device`, addressing earlier review feedback), [`hid_supported.go`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/hid_supported.go) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/hid_supported.go) (build tag `linux || darwin || windows`) wires `Enumerate`/`Open`/`Read`/`Write`/`Close` to `usbhid`, and [`hid_unsupported.go`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/hid_unsupported.go) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/hid_unsupported.go) (negated build tag) provides stubs that match upstream's CGO-disabled fallback: `Supported() == false`, `Enumerate() == nil`, `Open()` returns `ErrUnsupportedPlatform`. `Open()` re-looks the device up by `Path` via `usbhid.Get`, so a `DeviceInfo` can be copied/serialized safely. Root [`go.mod:111`](https://github.com/gnolang/gno/blob/b6bb8efff/go.mod#L111) · [↗](../../../../../.worktrees/gno-review-5494/go.mod#L111) plus the two contribs that build CLI binaries with ledger ([`contribs/gnodev/go.mod:7-10`](https://github.com/gnolang/gno/blob/b6bb8efff/contribs/gnodev/go.mod#L7-L10) · [↗](../../../../../.worktrees/gno-review-5494/contribs/gnodev/go.mod#L7-L10), [`contribs/gnokeykc/go.mod:7-10`](https://github.com/gnolang/gno/blob/b6bb8efff/contribs/gnokeykc/go.mod#L7-L10) · [↗](../../../../../.worktrees/gno-review-5494/contribs/gnokeykc/go.mod#L7-L10)) carry the `replace`. [`Dockerfile:10`](https://github.com/gnolang/gno/blob/b6bb8efff/Dockerfile#L10) · [↗](../../../../../.worktrees/gno-review-5494/Dockerfile#L10) and [`.dockerignore:8`](https://github.com/gnolang/gno/blob/b6bb8efff/.dockerignore#L8) · [↗](../../../../../.worktrees/gno-review-5494/.dockerignore#L8) make the shim visible to the build context. [`Makefile:16-19`](https://github.com/gnolang/gno/blob/b6bb8efff/Makefile#L16-L19) · [↗](../../../../../.worktrees/gno-review-5494/Makefile#L16-L19) updates the `CGO_ENABLED ?= 0` comment to reference the new pure-Go path.

## Benchmarks / Numbers

Cross-compile success (`CGO_ENABLED=0 GOOS=... go build ./gno.land/cmd/gnokey`):

| GOOS    | Backend selected                 | Result |
|---------|----------------------------------|--------|
| linux   | puregohid (real ledger via usbhid) | OK   |
| darwin  | puregohid (real ledger via usbhid) | OK   |
| windows | puregohid (real ledger via usbhid) | OK   |
| freebsd | puregohid stub (no ledger)       | OK     |

Standalone build of contribs without `replace` (`cd contribs/gnobro && CGO_ENABLED=0 go build .`):

| Binary | hid implementation linked              |
|--------|----------------------------------------|
| gnokey (root) | `rafaelmartins/usbhid` (puregohid)|
| gnobro / gnogenesis / gnokms / etc. | `zondax/hid v0.9.2` (CGO-disabled stub — no ledger) |

## Critical (must fix)

None.

## Warnings (should fix)

- **[partial replace — sibling modules silently keep zondax/hid stub]** [`go.mod:111`](https://github.com/gnolang/gno/blob/b6bb8efff/go.mod#L111) · [↗](../../../../../.worktrees/gno-review-5494/go.mod#L111) — only the root module, `contribs/gnodev`, and `contribs/gnokeykc` carry `replace github.com/zondax/hid => ./misc/puregohid`; `contribs/gnobro`, `contribs/gnogenesis`, `contribs/gnokms`, `contribs/gnomigrate`, `contribs/tx-archive`, `misc/loop`, `misc/autocounterd`, and `misc/stress-test/stress-test-many-posts` all transitively pull `tm2/pkg/crypto/internal/ledger` → `zondax/hid` but were not updated.
  <details><summary>details</summary>

  When built standalone (their own module graph), these binaries resolve `github.com/zondax/hid v0.9.2` from the proxy and select `hid_disabled.go` (`!cgo` branch), so `hid.Supported()` returns false and ledger is unavailable. That matches pre-PR behavior, so it is not a regression. But the PR's stated goal — "Enables ledger support without relying on cgo" — is only realized for the root binaries and the two contribs with the replace. `gnobro` is the main user-visible casualty (it links a `KeyBaseFromDir` flow that today errors with "not available on this platform" if a user tries `ledger:` keys, even with the shim available). For `gnokms` the gap matters more: it is a remote signer that could plausibly back a ledger-stored key. Fix: add `replace github.com/zondax/hid v0.9.2 => ../../misc/puregohid` to every sub-module under `contribs/` and `misc/` whose `go mod why github.com/zondax/hid` resolves through `tm2/pkg/crypto/internal/ledger`, or document explicitly in the PR description that these binaries intentionally remain ledger-less.
  </details>

- **[hardcoded Interface = 0 leaks into ledger-go fallback]** [`misc/puregohid/hid_supported.go:52-55`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/hid_supported.go#L52-L55) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/hid_supported.go#L52-L55) — `rafaelmartins/usbhid` does not expose the HID interface number, so the shim hardcodes `DeviceInfo.Interface = 0`. `ledger-go`'s fallback matcher (when `UsagePage != 0xffa0`) uses `supportedLedgerProductID[productIDMM] == d.Interface`.
  <details><summary>details</summary>

  Today every entry in `supportedLedgerProductID` (Nano X, Nano S, Nano S Plus, Stax, Flex) maps to interface 0, so `0 == 0` always — the fallback works for all currently supported Ledger devices and the primary `UsagePage == 0xffa0` path is unaffected. The risk is forward-only: if Ledger ever ships a product whose USB HID interface for the Cosmos app is not interface 0, the shim will silently misidentify it. The author flagged this in the comment and in [PR review reply r3104605563](https://github.com/gnolang/gno/pull/5494#discussion_r3104605563); upstreaming an `Interface()` accessor to `rafaelmartins/usbhid` is the proper fix. Acceptable as-is for merge; worth tracking as a follow-up issue so the gap does not decay into a silent failure later.
  </details>

## Nits

- [`misc/puregohid/hid_supported.go:36-38`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/hid_supported.go#L36-L38) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/hid_supported.go#L36-L38) — `Enumerate` swallows the `usbhid.Enumerate` error and returns `nil`; matches upstream `zondax/hid` (line 56 of `hid_enabled.go`) which also returns `nil` on enumeration error, so consistent. Optional: log via the `tm2` logger so silent enumeration failures aren't invisible.
- [`misc/puregohid/hid_supported.go:78-90`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/hid_supported.go#L78-L90) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/hid_supported.go#L78-L90) — `Write` always returns `len(b)` on success regardless of how many bytes the HID layer actually accepted. Upstream `zondax/hid` returns the wire count (decremented to strip the Windows 0x00 prefix). For ledger-go's 64-byte packets the difference is invisible (single-call writes), but the contract drift is real. Optional: return the actual write count for forward-compat with consumers that loop on partial writes.
- [`misc/puregohid/hid_supported.go:94-107`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/hid_supported.go#L94-L107) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/hid_supported.go#L94-L107) — `Read` calls `GetInputReport()` which blocks until a report arrives, with no timeout. Upstream `zondax/hid` has the same blocking semantic (also no timeout), so behavior parity holds. Worth flagging only because the shim's read path is now the canonical implementation for the project — long-term, a `ReadWithTimeout` analog would be valuable.
- [`misc/puregohid/hid_supported.go:64`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/hid_supported.go#L64) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/hid_supported.go#L64) — `usbhid.Get(filter, true, false)` opens without exclusive lock (`lock=false`). Upstream `zondax/hid` Open also doesn't take an exclusive lock (it relies on `hid_open_path` semantics, which on macOS is non-exclusive). Parity preserved; this is fine.
- [`go.mod:111`](https://github.com/gnolang/gno/blob/b6bb8efff/go.mod#L111) · [↗](../../../../../.worktrees/gno-review-5494/go.mod#L111) — `replace github.com/zondax/hid v0.9.2 => ./misc/puregohid` pins the exact version. If `zondax/hid` ever bumps to `v0.9.3` (or any sibling pulls a newer transitive), the replace stops applying and the CGO dep returns silently. Optional: drop the version pin (`replace github.com/zondax/hid => ./misc/puregohid`) so the shim covers all versions.

## Missing Tests

- **[no unit tests for the shim]** [`misc/puregohid/`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/) — no `*_test.go` under `misc/puregohid/`. The unsupported-platform branch is trivially testable (call each stub, expect `ErrUnsupportedPlatform`/`nil`) without USB hardware. The supported branch needs a Ledger to exercise end-to-end; at minimum a fake `usbhid` interface would let `Enumerate`/`Open` be unit-tested. Practical: most of this is hard to test without hardware; a single TestUnsupportedPlatform under `//go:build !linux && !darwin && !windows` would at least pin the stub behavior.

## Suggestions

- `contribs/{gnobro,gnogenesis,gnokms,gnomigrate,tx-archive}/go.mod` and `misc/{loop,autocounterd,stress-test/stress-test-many-posts}/go.mod` — mirror the `replace` directive so the entire repo speaks one HID backend regardless of which module is built. The author flagged the puregohid-shim-placement question in the PR body ("not sure about the approach to fix the E2E workflow"); moving the shim to a top-level `tm2/internal/hid` (or similar) and updating each `go.mod` would be the cleanest end-state. Acceptable as follow-up.
- [`misc/puregohid/go.mod:3`](https://github.com/gnolang/gno/blob/b6bb8efff/misc/puregohid/go.mod#L3) · [↗](../../../../../.worktrees/gno-review-5494/misc/puregohid/go.mod#L3) — `go 1.24.0` here vs `1.24.0` + `toolchain go1.24.4` in `contribs/gnodev/go.mod`. Consistency: align the toolchain directive across the modules touched by this PR.
- The PR description still references "Can someone help me test on Windows/Linux?" — based on D4ryl00's comment ("Tested successfully on Linux btw."), Linux is covered; Windows testing would be valuable before merge given the Windows-specific 0x00-prefix handling in `usbhid.setOutputReport` (which is correct, but only exercised by an actual write).

## Questions for Author

- Was the decision to leave `contribs/gnobro`, `contribs/gnokms`, etc. on the disabled-CGO `zondax/hid` stub intentional (because none of them currently expose a ledger-key code path to users), or just an oversight? If intentional, worth a sentence in the PR description; if oversight, the fix is a one-line `replace` per `go.mod`.
- Has anyone tested signing a transaction end-to-end on a Windows host with the shim? The macOS demo in the PR body covers `gnokey add ledger` + `maketx send`; Windows is the next interesting platform because `usbhid.setOutputReport` does a per-report 0x00 prefix prepend that's silent on success but easy to get wrong on the wire.
- Any plan to upstream an `Interface()` accessor to `rafaelmartins/usbhid` so the hardcoded `Interface = 0` can go away?
