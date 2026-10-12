# Findings in posting order, from round assemble: 1 to post, 0 SKIP, 4 refuted kept out

## Body

- Suggestion: tm2/pkg/bft/consensus/replay.go:239 [gh](https://github.com/gnolang/gno/blob/1a1a313105e35c8d31fabac32413d0558a9ad476/tm2/pkg/bft/consensus/replay.go#L239), outside the diff, so no anchor
  State: CONFIRMED, band: Suggestion, angle: removed
  TL;DR: Handshake reads ResponseInfo without checking its Error, so an app refusing Info still completes the handshake; no in-tree app sets that field
  Check: an app whose Info returns Error, run through NewHandshaker(...).Handshake
  Details: replay.go:239-243 checks only the Go error from InfoSync. A handshake against an app whose Info returns ResponseBase.Error completes with a nil error. No non-test Info implementation sets the field (BaseApp, kvstore, persistent_kvstore, counter, BaseApplication), so nothing reaches it today, and it predates the branch. Ships SKIP: no issue, for the same reason as round 3 row 1003.
  Evidence: go test -overlay with tests/parent-5-info-error-ignored_test.go ./tm2/pkg/bft/consensus -run TestZZInfoErrorIgnored -v: 'handshake err=<nil>', --- PASS
  Artifact: tests/parent-5-info-error-ignored_test.go
