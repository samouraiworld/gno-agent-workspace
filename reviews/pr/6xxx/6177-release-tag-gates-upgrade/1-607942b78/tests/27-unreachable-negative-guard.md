# Candidate 27: `n < 0` guard at node_params.go:308 is unreachable

Asserts: deleting `|| n < 0` from the `err != nil || n < 0` check at
`gno.land/pkg/gnoland/node_params.go:308` leaves `TestParseReleaseVersion`
fully green, including the `"signed component"` (`v1.+2.0`) and
`"negative component"` (`v1.-2.0`) cases — proving no input ever reaches
`strconv.Atoi` carrying a sign, since line 293 (`strings.Cut(rest, "+")`) and
line 294 (`strings.Cut(rest, "-")`) already strip everything from the first
`+` or `-` before the per-component loop runs.

Measured at PR 6177, head `607942b78fa4fdf6f378fecce32bc1d1d984ab8e`, run in
a scratch worktree off the reviewed head, never the shared review worktree.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
sed -n '300,312p' gno.land/pkg/gnoland/node_params.go
sed -i 's/if err != nil || n < 0 {/if err != nil {/' gno.land/pkg/gnoland/node_params.go
go test ./gno.land/pkg/gnoland/... -run 'TestParseReleaseVersion' -v
git checkout -- gno.land/pkg/gnoland/node_params.go
```

Observed output (trimmed to the signal-bearing lines):

```
--- PASS: TestParseReleaseVersion/negative_component (0.00s)
--- PASS: TestParseReleaseVersion/signed_component (0.00s)
...
PASS
ok  	github.com/gnolang/gno/gno.land/pkg/gnoland	0.069s
```

Both cases the guard's own comment cites as its target ("Reject \"+1\", \"-1\"
...") pass unchanged with the guard deleted. They are rejected earlier, by
the `len(parts) != 3` check after the `+`/`-` truncation splits the string
into extra components.
