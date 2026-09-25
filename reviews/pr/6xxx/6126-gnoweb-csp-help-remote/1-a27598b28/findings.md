# Findings in posting order, from round assemble: 1 to post, 0 SKIP, 3 refuted kept out

## gno.land/cmd/gnoweb/main_test.go:134 [gh](https://github.com/gnolang/gno/blob/a27598b2867c226470b01c23f48ccdf5089c4e87/gno.land/cmd/gnoweb/main_test.go#L134) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: the added test stays green with the fix reverted in setupWeb
Check: tests/solo-test-misses-wiring.sh: revert main.go:283 to NodeRemote, go test ./cmd/gnoweb -run 'TestSecureHeadersMiddleware|TestSetupWeb': expect FAIL, observe ok
Details: The test calls newSecureHeadersMiddleware with an AppConfig it builds itself, so it pins that the wrapper reads cfg.RemoteHelp and nothing about setupWeb. TestSetupWeb calls setupWeb but drops the server and its handler. With main.go:283 reverted to appcfg.NodeRemote the four tests pass. The merge base has neither the wrapper nor the test, so there is no base run to compare; the gap is the branch's own test.
Evidence: bash tests/solo-test-misses-wiring.sh in a scratch worktree at a27598b28: main.go 1 insertion 1 deletion; --- PASS: TestSecureHeadersMiddlewareUsesRemoteHelp (0.00s); ok github.com/gnolang/gno/gno.land/cmd/gnoweb 0.022s
Artifact: tests/solo-test-misses-wiring.sh
