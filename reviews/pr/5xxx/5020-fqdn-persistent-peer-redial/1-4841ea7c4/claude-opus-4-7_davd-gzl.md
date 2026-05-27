# PR #5020: feat: Re-resolve FQDN persistent peers on redial

URL: https://github.com/gnolang/gno/pull/5020
Author: D4ryl00 | Base: master | Files: 2 | +127 -3
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5020 4841ea7c4` (then `gh -R gnolang/gno pr checkout 5020` inside it)

Verdict: NEEDS DISCUSSION — fix works for the headline scenario, but the "fall back to last-known IP" rationale the author gave on the thread is not what the code does, and three smaller items (no Validate after re-resolve, no IPv4 preference / shuffle when picking `addrs[0]`, hostname re-resolution loses the original `Hostname` only on the dial-time value copy) deserve a confirmation before merge.

## Summary

`persistent_peers = "...@gno-sen-1:26656"` resolves the FQDN once at startup, caches the IP on `NetAddress`, and reuses it forever — so sentries that rotate IPs (typical in containerised / DHCP / cloud setups) leave the validator wedged on a dead address. The PR adds a `Hostname` field that preserves the original host string and a `PrepareForDial(ctx)` step that calls `net.DefaultResolver.LookupIPAddr` and overwrites `na.IP` immediately before each `DialContext`. Pure IPs skip resolution. Net effect: redial now picks up DNS changes; non-FQDN configurations are unchanged.

## Glossary

- `NetAddress` — peer identity + dial coordinates ([ID, IP, Hostname, Port]).
- `PrepareForDial(ctx)` — new method: parses `Hostname` as IP or DNS-resolves it, writes result to `na.IP`.
- Persistent peers — peers in `p2p.persistent_peers` config; the switch redials them indefinitely via `runRedialLoop`.

## Fix

Before: [`NewNetAddressFromString`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L79-L132) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L79-L132) did one blocking `net.LookupIP` at parse time, stored the first result in `na.IP`, and that was the IP for the lifetime of the process. After: the original host string is also kept in `na.Hostname` (omitempty, additive on JSON), and [`DialContext`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L258-L271) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L258-L271) calls [`PrepareForDial`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L232-L255) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L232-L255) first — which re-resolves the hostname with context-aware DNS and overwrites `na.IP`. The load-bearing constraint is that [`MultiplexSwitch.runDialLoop`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/switch.go#L308) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/switch.go#L308) calls `sw.transport.Dial(dialCtx, *peerAddr, ...)` — the dereference makes a value copy, so `PrepareForDial` mutates the copy, not the `*NetAddress` stored in `sw.persistentPeers`. That kills the race I worried about, but it also means the original-pointer `IP` stays frozen at startup forever (irrelevant for dialing because every dial re-resolves, but the stored value is misleading).

## Warnings (should fix)

- **[stated fallback does not exist]** [`tm2/pkg/p2p/types/netaddress.go:243-250`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L243-L250) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L243-L250) — author told reviewers "if the lookup fails we retry with the known IP"; code returns an error and aborts the dial instead.
  <details><summary>details</summary>

  In the [comment thread](https://github.com/gnolang/gno/pull/5020#discussion_r2717374562) the author defended keeping both `IP` and `Hostname` with: "during the redial the hostname lookup failed, so our last solution is to retry with the known IP address. That's why we keep IP and Hostname in the NetAddress struct." But `PrepareForDial` returns `fmt.Errorf("unable to resolve host ...")` on any lookup failure, and `DialContext` propagates that error without ever attempting `d.DialContext(ctx, "tcp", na.DialString())` with the previously-known IP. Worse, because the switch passes `*peerAddr` by value to `transport.Dial`, even if `PrepareForDial` *did* set a fallback, the original pointer in `persistentPeers` never updates — so "last known IP" is always the startup IP, not the most recent successful resolution. Either the code should match the stated intent (on lookup failure, log + dial `na.IP` if non-nil + valid) or the comment defending the two-field design should be retracted. Fix: pick one; right now the design rationale and the implementation disagree.
  </details>

- **[no Validate after re-resolve]** [`tm2/pkg/p2p/types/netaddress.go:252`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L252) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L252) — `PrepareForDial` blindly assigns `addrs[0].IP`; nothing checks the result against `Validate()` before dialing.
  <details><summary>details</summary>

  [`NetAddress.Validate`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L289-L317) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L289-L317) rejects unspecified IPs, broadcast, RFC3849, malformed lengths. A misconfigured / hijacked DNS that returns `0.0.0.0`, `255.255.255.255`, or `2001:db8::1` will now be dialed without complaint, because the dial path bypasses `Validate` entirely (it was only enforced at construction time). Pre-PR, the IP was validated indirectly when the operator-supplied string was parsed; post-PR, the operator-supplied string is parsed but the *resolved* IP isn't. Fix: call `na.Validate()` after the assignment in `PrepareForDial` and return the validation error.
  </details>

- **[non-deterministic IP selection]** [`tm2/pkg/p2p/types/netaddress.go:252`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L252) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L252) — `addrs[0].IP` picks the first answer; no IPv4 preference, no IPv6 fallback, no rotation on failure.
  <details><summary>details</summary>

  `net.DefaultResolver.LookupIPAddr` returns addresses in the order the resolver provided them. For a dual-stack `localhost` that's typically IPv6 first, but for a containerised sentry it depends on the resolver and round-robin scheme. If `addrs[0]` is unreachable (firewall, IPv6-disabled host, stale record), the dial fails and the redial loop will keep picking `addrs[0]` next time. The legacy `NewNetAddressFromString` had the exact same `ips[0]` flaw ([line 119](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L119) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L119)), so this isn't a regression — but the PR re-introduces it in a new place that runs on every dial rather than once at startup, multiplying its impact. Fix: prefer IPv4 if the address pool contains both (parameter or config knob), or at least iterate the returned slice on dial failure before returning.
  </details>

- **[blocking DNS at config parse, still]** [`tm2/pkg/p2p/types/netaddress.go:114`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L114) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L114) — `NewNetAddressFromString` still calls `net.LookupIP` (no context, no timeout), so a slow/failing DNS server during `persistent_peers` parsing blocks node startup.
  <details><summary>details</summary>

  The PR introduces the *correct* pattern (`net.DefaultResolver.LookupIPAddr(ctx, ...)`) in `PrepareForDial` but leaves the legacy `net.LookupIP(host)` call in `NewNetAddressFromString`. Since the new design re-resolves on every dial anyway, the startup-time resolution is now mostly cosmetic — its only purpose is to populate `na.IP` so `Validate()` passes at parse time. If the operator's DNS is down at boot, [`node.go:650`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/bft/node/node.go#L650) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/bft/node/node.go#L650) `NewNetAddressFromStrings` for `PersistentPeers` will return an error per entry, dropping FQDN persistent peers from the node entirely — the very scenario the PR exists to fix is brittle at boot. Fix: when `host` doesn't parse as IP, accept the address with `na.IP = nil, na.Hostname = host` and skip the initial resolution; let `PrepareForDial` do the first resolution at first-dial time. Adjust `Validate` (or its callers) to tolerate `IP == nil && Hostname != ""`.
  </details>

## Nits

- [`tm2/pkg/p2p/types/netaddress.go:230-231`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L230-L231) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L230-L231) — godoc says "updates the IP field with the latest lookup result" but the early-return on `Hostname == ""` means it does nothing for IP-only peers. Worth one sentence: "no-op if `Hostname` is empty (e.g. address constructed from a literal IP)".

- [`tm2/pkg/p2p/types/netaddress.go:233`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L233) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L233) — `na == nil` guard in `PrepareForDial` is dead in practice (callers always have a non-nil receiver, `DialContext` would have panicked at the method dispatch otherwise). Harmless but inconsistent with `DialContext` and the rest of the file. Drop or apply consistently.

- [`tm2/pkg/p2p/types/netaddress.go:237-241`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L237-L241) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L237-L241) — when `Hostname` parses as a literal IP, `na.IP = ip` is assigned and returned with no further check; if the operator supplied something like `::ffff:0.0.0.0` (unspecified mapped) the dial will quietly target an invalid address. Same root cause as the no-Validate warning above; combining both fixes covers this.

- [`tm2/pkg/p2p/types/netaddress.go:83`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L83) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L83) — `hostname := ""` declared up-front and assigned conditionally; tighter as a single `var hostname string` inside the `if ip == nil` block, since it's only meaningful there. Tiny.

## Missing Tests

- **[fallback path]** [`tm2/pkg/p2p/types/netaddress_test.go:330-347`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress_test.go#L330-L347) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress_test.go#L330-L347) — `TestNetAddressDialContextInvalidHostname` asserts an error on unresolvable hostname, but no test exercises the *intended* "fall back to known IP" behaviour (because, per the warning above, it doesn't exist). Once the design intent is settled, a test should pin it: hostname unresolvable + non-nil `IP` → either error (current) or successful dial to `IP` (stated intent), but tested.
  <details><summary>details</summary>

  The existing test confirms `DialContext` errors out. Either keep that behaviour and update the inline comment / commit message to say "no fallback by design", or add a test that injects a failing resolver and confirms the dialer falls back to `na.IP`. Today the test passes because resolution fails AND the `IP` field is set to `127.0.0.1`, but the dial doesn't actually use the fallback IP — it just errors before reaching the dialer.
  </details>

- **[Hostname propagation in Amino/JSON round-trip]** [`tm2/pkg/p2p/types/netaddress.go:198-217`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L198-L217) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L198-L217) — no test confirms whether `Hostname` survives `MarshalAmino`/`UnmarshalAmino` or whether the field is meant to be local-only.
  <details><summary>details</summary>

  `MarshalAmino` uses `DialString()`, which only emits `na.IP.String():Port` — so `Hostname` is NOT gossiped over p2p, which is correct (peer-exchange must not propagate operator hostnames). But the JSON tag `json:"hostname,omitempty"` *will* serialise it in config dumps / state snapshots. A round-trip test would document this asymmetry: Amino loses the Hostname (and a `UnmarshalAmino` of the Amino string would re-trigger DNS via the legacy path), JSON preserves it. Worth a single test that pins the behaviour.
  </details>

- **[`PrepareForDial` with context cancellation]** [`tm2/pkg/p2p/types/netaddress.go:243`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L243) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L243) — no test exercises `LookupIPAddr` honouring an already-cancelled `ctx`. The whole point of switching from `net.LookupIP` to `net.DefaultResolver.LookupIPAddr(ctx, ...)` was context awareness; a 3-line test that passes a cancelled context and asserts `ctx.Err()` propagates would lock it in.

## Suggestions

- [`tm2/pkg/p2p/types/netaddress.go:265`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L265) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L265) — log when the resolved IP differs from `na.IP` before overwriting it.
  <details><summary>details</summary>

  The PR description promises: "Log when a persistent peer's resolved IP changes to aid debugging." I don't see that log anywhere in the diff — `PrepareForDial` silently overwrites `na.IP` without comparing or emitting a structured log. The Logger isn't available inside the `types` package, but `MultiplexSwitch.runDialLoop` could log before/after the dial (compare `peerAddr.IP` before `Dial` to the post-dial peer's `SocketAddr().IP`). Worth adding so the PR delivers what the body says.
  </details>

- [`tm2/pkg/p2p/types/netaddress.go:232`](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/netaddress.go#L232) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/netaddress.go#L232) — `PrepareForDial` is exported but only called from `DialContext` in this package. Either unexport it (lowercase `prepareForDial`) or document why external callers might need it.

## Questions for Author

- The thread says "we keep `IP` and `Hostname` so that if hostname lookup fails we retry with the known IP" — can you point to the code that implements that fallback? I don't see it; `PrepareForDial` returns the lookup error and `DialContext` propagates it.
- Was IPv4-vs-IPv6 selection considered for `addrs[0]`? For dual-stack containerised sentries it's a frequent footgun.
- Should `node_info.Validate` (which round-trips via `NewNetAddressFromString` at [node_info.go:79](https://github.com/gnolang/gno/blob/4841ea7c4/tm2/pkg/p2p/types/node_info.go#L79) · [↗](../../../../../.worktrees/gno-review-5020/tm2/pkg/p2p/types/node_info.go#L79)) tolerate hostnames? Currently `DialString` only emits IPs so this path can't trigger DNS from remote data, but if a future change ever marshals `Hostname`, that validation suddenly becomes a remote-triggered DNS lookup.
