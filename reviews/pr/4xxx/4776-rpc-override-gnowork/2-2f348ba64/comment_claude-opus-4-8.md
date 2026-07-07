# Review: PR [#4776](https://github.com/gnolang/gno/pull/4776)
Event: COMMENT

## Body
Reproduced on 2f348ba64.

- moul's merge condition, keeping the fully qualified domain in the download path so removing the override later re-fetches, is not met in effect. The modcache is keyed by pkgPath only, and `DownloadPackageToCache` early-returns when the `.markers/<bech32(pkgPath)>` file exists, so bytes fetched through an override land under the canonical path and are reused after the override is removed, until the cache is wiped. The domain is in the path, but the re-fetch it was meant to force does not happen.
- The schema still ships the `[domains."<domain>"] rpc = ...` form the thread asked to reconsider: top-level `rpc` versus per-domain keying, the `dep_source` naming, a fallback list. That decision is a one-way door once workspaces adopt it, so it wants settling before merge, not after.
- `docs/resources/configuring-gno-projects.md` still says `gnowork.toml` has no configuration options and should be an empty file, which is now the opposite of what the PR ships. Document the `[domains."<domain>"] rpc = ...` schema there, and update the dependency-source TODO note just below it.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/4xxx/4776-rpc-override-gnowork/2-2f348ba64/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/packages/load.go:167-177 [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L167-L177)
gnodev's default fetcher `disabledFetcher` does not implement `OverrideDomainsRPCs`, and neither does `domainFetcher` under `-remote`, so any workspace whose `gnowork.toml` carries an rpc override makes gnodev abort at startup. Forwarding the type-assert alone is not enough: `domainFetcher` refuses any domain not also passed via `-remote`. Decide whether a workspace override should reach gnodev at all, and pin it with a test.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4776 -R gnolang/gno
( cd contribs/gnodev && go build -o /tmp/gnodev-4776 . )
WS=$(mktemp -d); mkdir -p "$WS/r/demo/foo"
printf '[domains."gno.land"]\nrpc = "http://localhost:26657"\n' > "$WS/gnowork.toml"
printf 'module = "gno.land/r/demo/foo"\ngno = "0.9"\n' > "$WS/r/demo/foo/gnomod.toml"
printf 'package foo\n' > "$WS/r/demo/foo/foo.gno"
( cd "$WS" && timeout 25 /tmp/gnodev-4776 -no-watch 2>&1 | grep -m1 "does not support" )
printf '' > "$WS/gnowork.toml"   # drop the override
( cd "$WS" && timeout 25 /tmp/gnodev-4776 -no-watch 2>&1 | grep -m1 "node is ready" )
rm -rf "$WS" /tmp/gnodev-4776
```
```
unable to initialize the node: reload packages: load packages: gnowork.toml requests rpc overrides but the configured package fetcher (packages.disabledFetcher) does not support them
            ┃ I node is ready took=3.4s
```
</details>

## gnovm/pkg/packages/gnowork.go:9-11 [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go#L9-L11)
go-toml does no strict decoding here and the field has no `toml:"rpc"` tag, so a misspelled table like `[domain."gno.land"]` or a misspelled key like `rcp =` parses with no override and no error, and dependencies then download from the public `https://rpc.gno.land:443` instead of the intended endpoint. Reject unknown keys at parse time so a typo fails loudly.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4776 -R gnolang/gno
cat > gnovm/pkg/packages/zz_typo_test.go <<'EOF'
package packages
import "testing"
func TestZZTypo(t *testing.T) {
	gw, err := ParseGnowork([]byte("[domain.\"gno.land\"]\nrpc = \"http://local:26657\"\n"))
	if err != nil { t.Fatal(err) }
	if len(gw.rpcOverrides()) != 0 { t.Fatalf("expected no override, got %v", gw.rpcOverrides()) }
	t.Log("misspelled table: no override, no error -> silent fallback to public endpoint")
}
EOF
go test -run TestZZTypo -v ./gnovm/pkg/packages/
rm gnovm/pkg/packages/zz_typo_test.go
```
```
--- PASS: TestZZTypo (0.00s)
    zz_typo_test.go:7: misspelled table: no override, no error -> silent fallback to public endpoint
```
</details>

## gnovm/pkg/packages/load.go:201-205 [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L201-L205)
`findLoaderContextFor` now parses `gnowork.toml` on every workspace resolution, where before the content was never read here, so a syntax error in the file aborts `gno test`, `lint`, `list`, `deplist`, `transpile`, and gnodev, not just the fetch path. On master those commands ignore the file content. Consider parsing only where the overrides are consumed, so an unrelated toml error does not fail every workspace command.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4776 -R gnolang/gno
go build -o /tmp/gno-4776 ./gnovm/cmd/gno
WS=$(mktemp -d); mkdir -p "$WS/r/demo/foo"
printf '[domains."gno.land"\n' > "$WS/gnowork.toml"   # unclosed table header
printf 'module = "gno.land/r/demo/foo"\ngno = "0.9"\n' > "$WS/r/demo/foo/gnomod.toml"
printf 'package foo\n' > "$WS/r/demo/foo/foo.gno"
( cd "$WS" && /tmp/gno-4776 list ./... )
rm -rf "$WS" /tmp/gno-4776
```
```
parse gnowork file ".../gnowork.toml": (1, 2): unexpected token unclosed table key, was expecting a table key
```
</details>

## gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go:61-71 [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L61-L71)
`gno mod download --remote-overrides=gno.land=X` seeds the fetcher, then `Load` merges the workspace map on top, so for a shared domain the `gnowork.toml` value silently wins over the explicit flag. No test or doc states this precedence. Pick a direction and document it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4776 -R gnolang/gno
cat > gnovm/pkg/packages/zz_prec_test.go <<'EOF'
package packages
import (
	"testing"
	"github.com/gnolang/gno/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher"
)
func TestZZPrec(t *testing.T) {
	cli := map[string]string{"gno.land": "http://CLI"}       // --remote-overrides
	f := rpcpkgfetcher.New(cli)
	_ = applyRPCOverrides(f, map[string]string{"gno.land": "http://WORKSPACE"}) // gnowork.toml
	t.Logf("caller CLI map is now %v", cli)
}
EOF
go test -run TestZZPrec -v ./gnovm/pkg/packages/
rm gnovm/pkg/packages/zz_prec_test.go
```
```
--- PASS: TestZZPrec (0.00s)
    zz_prec_test.go:10: caller CLI map is now map[gno.land:http://WORKSPACE]
```
</details>

## gnovm/pkg/packages/pkgdownload/pkgfetcher.go:13-16 [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/pkgfetcher.go#L13-L16)
`RPCPackageFetcher` has no godoc; `PackageFetcher` directly above it does.

## gnovm/pkg/packages/load.go:59-62 [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L59-L62)
`applyRPCOverrides` runs before the `if !conf.Deps { return }` short-circuit, so a local-only load that never fetches still errors on an override it would not use. Gate it on `conf.Deps`.

## gnovm/pkg/packages/gnowork.go:24 [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go#L24)
`rpcOverrides` skips only an exactly-empty rpc; `rpc = " "` passes through and is used raw as the endpoint, surfacing only as an opaque client error at fetch time.

## gnovm/pkg/packages/load_rpcoverrides_test.go:23 [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load_rpcoverrides_test.go#L23)
Missing test: this file exercises `applyRPCOverrides` as a unit, but nothing drives the real `Load` path where findLoaderContext reads a workspace `gnowork.toml`, which is where the fetcher-abort and the precedence behavior actually live.

<details><summary>test cases</summary>

White-box tests (package `packages`), all green at 2f348ba64: an override plus a non-RPC fetcher through the real `Load` (errors), an empty rpc through `Load` (no abort), gnowork-vs-flag precedence, and the go-toml silent-accept cases. Drop into `gnovm/pkg/packages/`:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4776 -R gnolang/gno
curl -fsSL -o gnovm/pkg/packages/gnowork_rpcoverride_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4776-rpc-override-gnowork/2-2f348ba64/tests/gnowork_rpcoverride_blue_test.go
go test -v -run 'Review_|GoTomlBehavior' ./gnovm/pkg/packages/
rm gnovm/pkg/packages/gnowork_rpcoverride_test.go
```
</details>

## gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go:68-69 [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L68-L69)
This write lands in the map `New` stored by reference, so applying workspace overrides mutates the `--remote-overrides` map the caller passed in. Confirmed behaviorally: the caller's map is rewritten after `applyRPCOverrides`. Copy in `New`, or allocate here instead of mutating.
