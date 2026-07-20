# Contribution draft: PR [5835](https://github.com/gnolang/gno/pull/5835) — `realm_only_gate` rule

Not a review round. Unposted. Patch lives in `.worktrees/gno-review-5835`.

## Body

Ran the eight rules against three open realm PRs, [5976](https://github.com/gnolang/gno/pull/5976), [5951](https://github.com/gnolang/gno/pull/5951) and [5946](https://github.com/gnolang/gno/pull/5946). Only `current_guard` fired. The one exploitable bug in the set went undetected.

5976 gates on `if caller.IsUserCall() { panic }` to mean "realms only". `IsUserCall()` is `pkgPath == ""`, so it is false inside the `<domain>/e/<addr>/run` realm that `gnokey maketx run` uses. A user script passes the gate. `payment_user_call` cannot reach it: that rule keys on `OriginSend()`, which 5976 never calls.

Signature for the missing rule is `if x.IsUserCall()` without a `!`, which is `IsUserCall()` used to reject rather than to require. I have it working with a vulnerable/fixed pair; it flags `reputation.gno:17` in 5976. Want it as a PR against your branch?
