# PR [#6126](https://github.com/gnolang/gno/pull/6126): fix(gnoweb): use help remote for CSP connect-src

Verdict: REQUEST CHANGES, because the added test stays green when `setupWeb` passes `NodeRemote` again; the one-line fix itself is right and matches the fetch the Actions panel makes.
Event: COMMENT
Model: claude-opus-5-5 at high effort, quick review, solo shape
Commit: a27598b28
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/a27598b2867c226470b01c23f48ccdf5089c4e87) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/a27598b2867c226470b01c23f48ccdf5089c4e87)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6126 a27598b28`
Round: 1. One agent as finder, judge and writer, no reflector (one bundle), 4 candidates: the Warning run in a scratch worktree by the agent that found it, 3 refuted by read.


## Body

> AI review, claude-opus-5-5, quick review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6126-gnoweb-csp-help-remote/overview.md) · Status: REQUEST CHANGES · not manually verified, posted to help reviewer

## gno.land/cmd/gnoweb/main_test.go:134 [gh](https://github.com/gnolang/gno/blob/a27598b2867c226470b01c23f48ccdf5089c4e87/gno.land/cmd/gnoweb/main_test.go#L134) · [↗](../../../../../.worktrees/gno-review-6126/gno.land/cmd/gnoweb/main_test.go#L134) · Warning

Test: `newSecureHeadersMiddleware` gets a config the test builds itself, so the test never reaches [`setupWeb`](https://github.com/gnolang/gno/blob/a27598b2867c226470b01c23f48ccdf5089c4e87/gno.land/cmd/gnoweb/main.go#L283) and stays green with `appcfg.NodeRemote` put back there.

<details><summary>Repro</summary>

From a checkout of this branch, the test run after reverting the fixed call:

```bash
cd gno.land
sed -i 's|newSecureHeadersMiddleware(app, !cfg.noStrict, appcfg)|SecureHeadersMiddleware(app, !cfg.noStrict, appcfg.NodeRemote)|' cmd/gnoweb/main.go
go test ./cmd/gnoweb -run 'TestSecureHeadersMiddleware|TestSetupWeb' -count=1 -v 2>&1 | grep -E '^(--- |ok|FAIL)'
git checkout -- cmd/gnoweb/main.go
```

The new test passes with the old wiring back in place:

```
--- PASS: TestSetupWeb (0.00s)
--- PASS: TestSecureHeadersMiddlewareStrict (0.00s)
--- PASS: TestSecureHeadersMiddlewareNonStrict (0.00s)
--- PASS: TestSecureHeadersMiddlewareUsesRemoteHelp (0.00s)
ok  	github.com/gnolang/gno/gno.land/cmd/gnoweb	0.022s
```

`TestSetupWeb` is the one test that calls `setupWeb`, and it drops the returned server without reading its handler.

</details>
