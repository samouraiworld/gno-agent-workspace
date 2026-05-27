# PR #5360: feat(gnokms): add insecure flag

URL: https://github.com/gnolang/gno/pull/5360
Author: mvallenet | Base: master | Files: 5 | +86 -6
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5360 7527911` (then `gh -R gnolang/gno pr checkout 5360` inside it)

Verdict: REQUEST CHANGES — the PR closes the "no auth file" hole but leaves the "auth file with empty whitelist" hole wide open, so a TCP server started after `gnokms auth generate` (the README-recommended flow) still accepts any client. Also unresolved: `--insecure` flag name collides with `--insecure-password-stdin`, and the README step ordering is now broken (step 2 redirects elsewhere; the old step 2 is referenced from the Genesis section).

## Summary

Mutual auth for gnokms TCP listeners is the only thing standing between a network-reachable signing service and arbitrary callers. Before this PR, a missing auth keys file produced a warning and the server happily started; after, it errors unless `--insecure` is passed. The fix is one early `return` in [`NewSignerServer`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L45-L52) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L45-L52) plus a `bool` flag. That correctly handles the "no file" case — but the underlying TCP handshake check at [`tm2/pkg/bft/privval/signer/remote/tcp_conn.go:80-94`](https://github.com/gnolang/gno/blob/7527911/tm2/pkg/bft/privval/signer/remote/tcp_conn.go#L80-L94) · [↗](../../../../../.worktrees/gno-review-5360/tm2/pkg/bft/privval/signer/remote/tcp_conn.go#L80-L94) treats an empty `authorized_keys` list as "allow all", and [`gnokms auth generate`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/auth/auth_generate.go#L51-L68) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/auth/auth_generate.go#L51-L68) — the command the new README points users to in step 2 — produces exactly such a file. Net effect: the PR's promised default (mutual auth required) is satisfied only after the user also runs `gnokms auth authorized add <pubkey>`, which nothing forces them to do.

## Glossary

- `AuthKeysFile` — JSON file holding gnokms' server keypair plus a list of bech32 client pubkeys (`authorized_keys`). See [`auth_keys_file.go:25-31`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/auth_keys_file.go#L25-L31) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/auth_keys_file.go#L25-L31).
- `ClientAuthorizedKeys` — the whitelist field inside `AuthKeysFile`; empty by default after `gnokms auth generate`.
- `checkAuthorizedKeys` — handshake check at [`tcp_conn.go:80`](https://github.com/gnolang/gno/blob/7527911/tm2/pkg/bft/privval/signer/remote/tcp_conn.go#L80) · [↗](../../../../../.worktrees/gno-review-5360/tm2/pkg/bft/privval/signer/remote/tcp_conn.go#L80); short-circuits to `nil` (accept) when the whitelist is empty.
- `--insecure` (new) — explicit opt-out of auth requirement, server-side flag added by this PR.
- `--insecure-password-stdin` — pre-existing gnokey-subcommand flag that takes the gnokey wallet password from stdin; unrelated to mutual auth but shares the `--insecure` prefix.

## Fix

Before: [`server.go:44-48`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L44-L48) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L44-L48) (pre-PR) logged a warning when the auth keys file was absent on a TCP listener and continued to start. After: same branch returns an error unless `commonFlags.Insecure` is true ([`server.go:45-52`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L45-L52) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L45-L52)); the warning path is preserved only for the explicit `--insecure` case. The flag is registered globally on `ServerFlags` ([`flags.go:101-106`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/flags.go#L101-L106) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/flags.go#L101-L106)) so every server-shaped subcommand picks it up. Unix sockets are intentionally untouched — filesystem perms are the access control there.

## Critical (must fix)

- [@aeddi](https://github.com/gnolang/gno/pull/5360#discussion_r2477856612) [auth-file-with-empty-whitelist still accepts any client] [`contribs/gnokms/internal/common/server.go:33-43`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L33-L43) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L33-L43) — TCP server with auth keys file but empty `authorized_keys` admits any client, defeating the PR's stated default.
  <details><summary>details</summary>

  The new error in [`server.go:45-52`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L45-L52) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L45-L52) fires only when `osm.FileExists(commonFlags.AuthKeysFile)` is false. The README's [step 2](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/README.md#L101-L106) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/README.md#L101-L106) tells operators to run `gnokms auth generate`, which [creates the file with `ClientAuthorizedKeys: []string{}`](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/auth/auth_generate.go#L51-L68) via [`GeneratePersistedAuthKeysFile`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/auth_keys_file.go#L181-L200) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/auth_keys_file.go#L181-L200). The handshake check at [`tcp_conn.go:80-84`](https://github.com/gnolang/gno/blob/7527911/tm2/pkg/bft/privval/signer/remote/tcp_conn.go#L80-L84) · [↗](../../../../../.worktrees/gno-review-5360/tm2/pkg/bft/privval/signer/remote/tcp_conn.go#L80-L84) returns `nil` (accept) when `len(authorizedKeys) == 0`. A user who follows the README between step 2 (auth generate) and step 5 (auth authorized add) — i.e. starts the server during that window, or never gets to step 5 — has a TCP listener that signs for anyone who connects, while the server logs no warning at all. The PR description and README both claim "mutual authentication required by default"; that claim is false in this state.

  Fix: in [`server.go`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L44) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L44) extend the TCP-protocol branch so that when an auth keys file IS present, reject if `len(authKeysFile.AuthorizedKeys()) == 0` and `!commonFlags.Insecure`. Suggested message: "auth keys file has empty authorized_keys; add at least one client public key with 'gnokms auth authorized add <pubkey>', or use --insecure". Add a test case covering this branch alongside the new ones in [`server_test.go:42-57`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server_test.go#L42-L57) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server_test.go#L42-L57).
  </details>

## Warnings (should fix)

- [@aeddi](https://github.com/gnolang/gno/pull/5360#discussion_r2477856608) [flag name collides with existing `--insecure-password-stdin`] [`contribs/gnokms/internal/common/flags.go:101-106`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/flags.go#L101-L106) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/flags.go#L101-L106) — `gnokms gnokey` now exposes both `--insecure` (skip auth) and `--insecure-password-stdin` (read password from stdin) on the same command.
  <details><summary>details</summary>

  `gnokeyFlags` embeds `common.ServerFlags` ([`gnokey.go:13-18`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/gnokey/gnokey.go#L13-L18) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/gnokey/gnokey.go#L13-L18)), so `RegisterFlags` at [`gnokey.go:43-58`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/gnokey/gnokey.go#L43-L58) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/gnokey/gnokey.go#L43-L58) registers both. The names imply different threat models — one bypasses mutual auth on the wire, the other relaxes interactive prompting — but the shared `--insecure` stem invites confusion: an operator skimming `-h` could plausibly read `--insecure-password-stdin` as the auth-bypass flag they remember reading about. Test files already alternate between the two: [`gnokey_test.go:73-76`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/gnokey/gnokey_test.go#L73-L76) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/gnokey/gnokey_test.go#L73-L76) passes both in the same invocation.

  Fix: rename to `--insecure-no-auth` (aeddi's suggestion) or `--no-mutual-auth`. Keeps the safe-by-default contract while making the bypass explicit and non-overlapping with the password flag.
  </details>

- [@aeddi](https://github.com/gnolang/gno/pull/5360#discussion_r2477856602) [README step ordering broken, "Genesis" cross-reference stale] [`contribs/gnokms/README.md:41-61`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/README.md#L41-L61) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/README.md#L41-L61) — new step 2 is a forward reference to a section further down; step 4 jumps back to validator config without acknowledging the auth-keys round-trip; the Genesis section at line 65 still says "step 2 from the previous section" but step 2 no longer starts the server.
  <details><summary>details</summary>

  Reading the new flow top-down: step 1 generates a signing key; step 2 says "set up mutual authentication keys (see section below)"; step 3 starts the server; a NOTE block points at `--insecure`; step 4 sets the gnoland server address. The natural reader question between 3 and 4 — "wait, how does the validator know my pubkey?" — has no on-page answer. The actual mutual-auth dance lives at [`README.md:97-134`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/README.md#L97-L134) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/README.md#L97-L134), and steps 3-5 there interleave server-side and client-side commands, so a user following top-to-bottom has to bounce between sections.

  Also: [`README.md:65`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/README.md#L65) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/README.md#L65) still reads "When launching the `gnokms` server (e.g. step 2 from the previous section)" — pre-PR step 2 was the start command, now it's the auth-setup pointer. Reference is stale.

  Fix: do the README rewrite aeddi suggested in [the second comment](https://github.com/gnolang/gno/pull/5360#discussion_r2477856607) — integrate mutual auth into a single linear walkthrough (gnoland-side and gnokms-side steps interleaved in operator order), call out the UDS-or-`--insecure` alternatives once at the top, and update the Genesis section's "step 2" reference.
  </details>

- [misleading warning when `--insecure` is set] [`contribs/gnokms/internal/common/server.go:54-55`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L54-L55) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L54-L55) — the kept-over warning suggests `gnokms auth generate` is the fix, but as the Critical above shows, running `auth generate` alone does not produce mutual auth.
  <details><summary>details</summary>

  Operator runs `gnokms gnokey alice --insecure -listener tcp://...`, sees `"For more security, generate mutual auth keys using 'gnokms auth generate'"`, runs that command, drops the `--insecure` flag, restarts. Server now starts successfully (file present) and still has an empty whitelist — the operator believes they've upgraded from "insecure" to "secure" but the wire-level behaviour is identical. The warning text needs to point at the full flow ("generate keys AND add at least one authorized client pubkey") or at the Mutual TCP Authentication section in the README, not just `auth generate`.

  Fix: change the second warning to reference both `gnokms auth generate` and `gnokms auth authorized add`, or link to the README section.
  </details>

- [no e2e coverage that a TCP server with empty whitelist actually rejects an unauthorized client] [`contribs/gnokms/internal/common/server_test.go:42-92`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server_test.go#L42-L92) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server_test.go#L42-L92) — new tests only assert the constructor returns `(nil, err)`; they don't verify the wire-level handshake actually rejects a stranger.
  <details><summary>details</summary>

  The Critical above is invisible to the current test surface because the tests stop at `NewSignerServer` return values. A targeted assertion — start the server with an auth keys file whose `authorized_keys` is empty, dial it with a random client keypair, expect the connection to fail — would have surfaced both the missing check and any future regression. The `server_test.go` in `tm2/pkg/bft/privval/signer/remote/server/` already has the scaffolding for client/server roundtrips (see [`server_test.go:217-281`](https://github.com/gnolang/gno/blob/7527911/tm2/pkg/bft/privval/signer/remote/server/server_test.go#L217-L281) · [↗](../../../../../.worktrees/gno-review-5360/tm2/pkg/bft/privval/signer/remote/server/server_test.go#L217-L281)).

  Fix: add an integration-style test under `contribs/gnokms/internal/common/` (or a new file) that exercises the full handshake against an empty whitelist. This is the natural pairing for the Critical fix above.
  </details>

## Nits

- [`contribs/gnokms/internal/common/flags.go:106`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/flags.go#L106) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/flags.go#L106) — help text says "on TCP" but the flag has no effect on Unix sockets in either direction; consider "on TCP listeners" or "when using a TCP listener" to mirror the error message in [`server.go:47`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L47) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L47).

- [`contribs/gnokms/internal/common/server.go:53`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L53) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L53) — comment "// --insecure explicitly set: warn but continue." is true at this point in the code but slightly misleads: the warning is logged regardless of the listener (it sits inside the `protocol == "tcp"` branch, so technically only TCP, but a casual reader skimming might assume otherwise). Either drop the comment (the surrounding context makes it obvious) or anchor it to TCP explicitly.

- [`contribs/gnokms/internal/common/server_test.go:43,60,77`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server_test.go#L43) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server_test.go#L43) — the new subtests call `t.Parallel()` inside the subtest but the existing `nil signer` subtest at [`server_test.go:27-40`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server_test.go#L27-L40) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server_test.go#L27-L40) does not. Inconsistent; trivial.

## Missing Tests

- [empty `authorized_keys` rejection] [`contribs/gnokms/internal/common/server_test.go`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server_test.go) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server_test.go) — covered by the Warning above; lifted here so it's not lost.

- [`--insecure` flag is parsed correctly when set explicitly] — flags.go has zero direct unit coverage of the new flag (codecov flagged 0% patch coverage on flags.go for this reason). The server-level tests indirectly use `Insecure: true` on the struct but never via `RegisterFlags` + `fs.Parse`. Low-priority; the codepath is trivial.

## Suggestions

- [`contribs/gnokms/internal/common/server.go:46-51`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L46-L51) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L46-L51) — the error string is long and uses string concatenation across four lines; consider a single multiline raw string with `fmt.Errorf("...")` and explicit `\n` newlines so the rendered CLI output has line breaks. Currently it prints as one wall-of-text line.
  <details><summary>details</summary>

  Current rendering when surfaced via `commands.NewDefaultIO`: a single 200+ char line. With explicit `\n` the operator gets three readable lines: the headline, the two alternatives (auth-keys-file or auth generate), and the `--insecure` escape hatch. Optional.
  </details>

- [`contribs/gnokms/internal/common/server.go:44`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/server.go#L44) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/server.go#L44) — the `else if` chain treats "file missing on TCP" and "file present" as orthogonal branches but the Critical fix above will want to fold a third condition (file present + empty whitelist + !insecure) into the same place. Worth restructuring as a single decision tree to keep the policy in one spot.

## Questions for Author

- Was the empty-`authorized_keys` corner intentionally left out of scope (e.g. defer to a follow-up), or missed? If intentional, what's the rationale — does the team consider "auth-keys file present with empty whitelist" a deliberate dev mode that should NOT require `--insecure`?
- `--insecure` lives on `ServerFlags` which today is only embedded by `gnokms gnokey`; the `gnokms auth ...` subcommands only embed `AuthFlags` and don't get the flag. That's fine for now, but if a future backend (HSM, cloud-KMS) embeds `ServerFlags` it inherits the bypass for free. Worth a one-line code comment near [`flags.go:101`](https://github.com/gnolang/gno/blob/7527911/contribs/gnokms/internal/common/flags.go#L101) · [↗](../../../../../.worktrees/gno-review-5360/contribs/gnokms/internal/common/flags.go#L101) so the next contributor doesn't accidentally widen the surface.
