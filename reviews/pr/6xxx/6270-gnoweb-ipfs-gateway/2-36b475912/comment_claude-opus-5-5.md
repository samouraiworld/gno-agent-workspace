# Review: [#6270](https://github.com/gnolang/gno/pull/6270)
Posted: https://github.com/gnolang/gno/pull/6270#pullrequestreview-5548549030

Event: COMMENT
Verdict: APPROVE. The round-1 Warnings and the secret redaction hold at this head; what remains is a Missing test on the renderer's warning and a Nit on the flag help, neither blocking.
Model: claude-opus-5-5, standard review, solo: 2 finders at xhigh, one judge and writer at high
Commit: 36b475912eb31badc961e9b35fa765def60f250f
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6270 36b475912`
Round: 2. Solo round: 2 finders, a reflector pass by the judge, 6 candidates (3 from the finders, 3 from the reflector), each run or read from scratch by a judge that was not its finder; 3 confirmed, 3 refuted, 14 more settled by the finders.

## Body

> AI review, claude-opus-5-5, standard review, reviewed at [36b475912](https://github.com/gnolang/gno/pull/6270/commits/36b475912eb31badc961e9b35fa765def60f250f), [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6270-gnoweb-ipfs-gateway/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6270-gnoweb-ipfs-gateway/2-36b475912/claims.md) · Status: APPROVE

## gno.land/pkg/gnoweb/render.go:98 [gh](https://github.com/gnolang/gno/blob/36b475912eb31badc961e9b35fa765def60f250f/gno.land/pkg/gnoweb/render.go#L98) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/pkg/gnoweb/render.go#L98) · Missing test [posted](https://github.com/gnolang/gno/pull/6270#discussion_r4295257883)

Missing test: nothing reads this warning's output, so a `"gateway", cfg.IPFSGateway` pair added to it would log a credential with every test green.

<details><summary>test cases</summary>

The secret-shape loop in [`TestNewRouterIPFSGateway`](https://github.com/gnolang/gno/blob/36b475912eb31badc961e9b35fa765def60f250f/gno.land/pkg/gnoweb/ipfs_test.go#L113-L128) reaches only `NewRouter`'s error. This case passes as written, and fails once the warning logs `cfg.IPFSGateway`.

```go
func TestNewHTMLRendererIPFSGatewayLog(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	cfg := NewDefaultRenderConfig()
	cfg.IPFSGateway = "https://user:s3cret@gw.example"
	NewHTMLRenderer(slog.New(slog.NewTextHandler(&logs, nil)), cfg, nil)
	assert.Contains(t, logs.String(), "ignoring invalid IPFS gateway")
	assert.NotContains(t, logs.String(), "s3cret")
}
```

With the gateway value added to the warning, this case fails on the logged credential:

```text
Error: "time=... level=WARN msg=\"ignoring invalid IPFS gateway\" gateway=https://user:s3cret@gw.example error=\"must not contain credentials\"\n" should not contain "s3cret"
FAIL	github.com/gnolang/gno/gno.land/pkg/gnoweb
```

</details>

## gno.land/cmd/gnoweb/main.go:241 [gh](https://github.com/gnolang/gno/blob/36b475912eb31badc961e9b35fa765def60f250f/gno.land/cmd/gnoweb/main.go#L241) · [↗](../../../../../.worktrees/gno-review-6270/gno.land/cmd/gnoweb/main.go#L241) · Nit [posted](https://github.com/gnolang/gno/pull/6270#discussion_r4295257899)

Nit: this help names only `ipfs://` and retired-gateway URLs, though [`ipfsRewrittenGateways`](https://github.com/gnolang/gno/blob/36b475912eb31badc961e9b35fa765def60f250f/gno.land/pkg/gnoweb/markdown/ext_ipfs.go#L28) also sends `ipfs.filebase.io` links to whichever gateway is set. The same sentence is in gnodev's [`app_config.go`](https://github.com/gnolang/gno/blob/36b475912eb31badc961e9b35fa765def60f250f/contribs/gnodev/app_config.go#L152) and [README](https://github.com/gnolang/gno/blob/36b475912eb31badc961e9b35fa765def60f250f/contribs/gnodev/README.md#L113).
