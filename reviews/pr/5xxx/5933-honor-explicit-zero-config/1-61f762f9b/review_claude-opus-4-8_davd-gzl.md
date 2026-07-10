# PR [#5933](https://github.com/gnolang/gno/pull/5933): fix(tm2): honor explicit zero values in node config

URL: https://github.com/gnolang/gno/pull/5933
Author: aeddi | Base: master | Files: 25 | +300 -69
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 61f762f9b (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5933 61f762f9b`

**TL;DR:** A node operator who set a `config.toml` key to a zero value (`false`, `0`, `""`, `[]`) whose default is non-zero saw it silently revert to the default at load time. This decodes the file on top of the defaults instead of merging defaults into a zero-valued struct, so explicit zeros stick.

**Verdict: APPROVE** — correct fix, tested against the real library and load path; the mergo merge that caused the bug is removed entirely; no open concerns.

## Summary
`LoadConfig` (used by `gnoland start`) and `LoadOrMakeConfigWithOptions` decoded `config.toml` into a zero-valued `Config`, then called `mergo.Merge(loaded, DefaultConfig())` to fill missing keys. mergo's default mode fills any zero-valued destination field from the source, so it cannot tell an absent key from one explicitly set to the Go zero value: both look zero after decoding, and both get overwritten by the default. Any field with a non-zero default therefore could not be set to its zero value from the file. The fix decodes the TOML document on top of a `DefaultConfig()`-initialized struct and drops mergo. A TOML decoder leaves absent keys untouched and applies present keys including explicit zeros, which is the "defaults for missing keys only" behavior the mergo step was trying to reach. `dario.cat/mergo` is removed from `go.mod` across every module.

## Examples
Same complete, valid `config.toml`, load result before vs after:

| Key in file | Default | Before | After |
| --- | --- | --- | --- |
| `consensus.create_empty_blocks = false` | `true` | `true` | `false` |
| `mempool.recheck = false` | `true` | `true` | `false` |
| `rpc.cors_allowed_methods = []` | 4 items | 4 items | 0 items |
| `p2p.send_rate = 1024000` (non-zero) | `5120000` | `1024000` | `1024000` |
| key omitted | any | default | default |

Only explicit-zero-with-non-zero-default keys change. Absent keys and non-zero values behave identically before and after.

## Fix
`LoadConfigFile` now seeds `DefaultConfig()` and decodes the file into it via a new unexported `loadConfigFile(path, cfg)` that decodes into a caller-supplied `*Config` ([`toml.go:18-49`](https://github.com/gnolang/gno/blob/61f762f9b/tm2/pkg/bft/config/toml.go#L18-L49) · [↗](../../../../../.worktrees/gno-review-5933/tm2/pkg/bft/config/toml.go#L18)). `LoadConfig` drops its mergo step ([`config.go:74-93`](https://github.com/gnolang/gno/blob/61f762f9b/tm2/pkg/bft/config/config.go#L74-L93) · [↗](../../../../../.worktrees/gno-review-5933/tm2/pkg/bft/config/config.go#L74)). `LoadOrMakeConfigWithOptions` applies options to the defaults first, then decodes the file into the same struct, giving precedence file > options > defaults ([`config.go:100-149`](https://github.com/gnolang/gno/blob/61f762f9b/tm2/pkg/bft/config/config.go#L100-L149) · [↗](../../../../../.worktrees/gno-review-5933/tm2/pkg/bft/config/config.go#L100)).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. The four `LoadConfig` subtests plus two `LoadOrMakeConfigWithOptions` subtests cover explicit `false`, explicit empty slice, absent-key default preservation, non-zero passthrough with no slice doubling, and file-over-option precedence ([`config_test.go:33-247`](https://github.com/gnolang/gno/blob/61f762f9b/tm2/pkg/bft/config/config_test.go#L33-L247) · [↗](../../../../../.worktrees/gno-review-5933/tm2/pkg/bft/config/config_test.go#L33)).

## Suggestions
None.

## Verified
- Reverting the loader to the old decode-into-zero-struct plus `mergo.Merge` path reproduces the bug against the PR's own tests: `consensus.create_empty_blocks` and `mempool.recheck` load back as `true`, and `rpc.cors_allowed_methods = []` loads back as the 4-element default `[HEAD GET POST OPTIONS]`. CI only runs these tests against the fixed code, so this negative direction is not covered by CI.

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5933 -R gnolang/gno
  fix=$(git log --grep='honor explicit zero values in node config loader' --format=%H -1)
  git checkout "$fix"^ -- tm2/pkg/bft/config/config.go tm2/pkg/bft/config/toml.go go.mod go.sum
  go test ./tm2/pkg/bft/config/ -run 'TestConfig_LoadConfig/explicit' -v
  git checkout HEAD -- tm2/pkg/bft/config/config.go tm2/pkg/bft/config/toml.go go.mod go.sum
  ```

  ```
  Error: Should be false
  Error: Should be empty, but was [HEAD GET POST OPTIONS]
  --- FAIL: TestConfig_LoadConfig/explicit_zero_values_are_honored
  --- FAIL: TestConfig_LoadConfig/explicit_empty_slice_is_honored
  ```

- `github.com/pelletier/go-toml v1.9.5` allocates a fresh slice when decoding over a pre-populated struct rather than mutating in place: an explicit `[]` in the file loads as 0 entries against the 4-element default, and a complete file whose `cors_allowed_methods` equals the default loads back with 4 entries, not 8. No append and no stale tail.
- `dario.cat/mergo` is used by no `.go` file in the tree; its removal from the root and every contribs/misc `go.mod`/`go.sum` leaves all modules building, and CI is green across every module job.
- Load path is exercised end to end: `TestConfig_LoadConfig` calls the same `LoadConfig` that [`start.go:215`](https://github.com/gnolang/gno/blob/61f762f9b/gno.land/cmd/gnoland/start.go#L215) · [↗](../../../../../.worktrees/gno-review-5933/gno.land/cmd/gnoland/start.go#L215) invokes. Config package and `gno.land/cmd/gnoland` config tests pass at 61f762f9b.

## Open questions
- `config get`/`config set` read through `LoadConfigFile` ([`config_get.go:68`](https://github.com/gnolang/gno/blob/61f762f9b/gno.land/cmd/gnoland/config_get.go#L68) · [↗](../../../../../.worktrees/gno-review-5933/gno.land/cmd/gnoland/config_get.go#L68), [`config_set.go:56`](https://github.com/gnolang/gno/blob/61f762f9b/gno.land/cmd/gnoland/config_set.go#L56) · [↗](../../../../../.worktrees/gno-review-5933/gno.land/cmd/gnoland/config_set.go#L56)), so a hand-edited partial file now reports defaults for absent keys instead of Go zeros, and `config set` materializes those defaults on rewrite. Intended and documented in the ADR consequences; the new value is the one the node would actually use, so more correct. Not posted.
