# PR [#6206](https://github.com/gnolang/gno/pull/6206): fix(gnoweb): serve /u/<name> only for an address, a namespace with packages or a registered user
Verdict: REQUEST CHANGES. On gnoland-1 the gate 404s `/docs`, the shipped alias onto `/u/docs`; everything else is a Nit or a Suggestion.
Event: REQUEST_CHANGES
Model: claude-opus-5, effort high, standard review
Commit: 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/876762bdf2ea6635b27e2b0a42f26f9fdc54af24) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/876762bdf2ea6635b27e2b0a42f26f9fdc54af24)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6206 876762bdf`
Round: 1. 7 finders, one reflector, 37 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 7 refuted, none of them above Nit.

## Body

Two surfaces outside these files decide which `/u/` URLs a reader ever reaches, and neither matches the rule the handler enforces.

- A trailing slash loses the profile: [`IsDir`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/weburl/url.go#L256) holds for any path ending in `/` and is [tested before `IsUser`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L443-L450), so `/u/moul001/` reaches `GetDirectoryView` and 404s where `/u/moul001` serves the profile.
- gnoweb's own rendered mentions land on both sides of the rule: [`mentionNamePattern`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L28) admits consecutive underscores and names past [`maxUsernameLen`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L548), which the handler refuses, and excludes `-`, which the handler accepts, and the package's golden pins both, [`/u/user_with_many_________underscores_and_numbers_12345`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/golden/ext_mentions/user_mentions.md.txtar#L13) as a link and [`@not-user`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/golden/ext_mentions/user_mentions.md.txtar#L14) unlinked.

## SKIP gno.land/pkg/gnoweb/client.go:221 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/client.go#L221) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/client.go#L221) · Nit

Nit: `ListPaths` binds `limit` in its signature alone and hands `query` the prefix, so [`pathsLimit`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/sdk/vm/handler.go#L199-L209) applies its own `defaultLimit` and [`MaxUserContributions`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L499) is not the bound a user page renders under.

Skipped: line 221 is outside the diff and [`realm_directory.go:26-29`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/realm_directory.go#L26-L29) already records the behaviour, so it goes to an issue rather than this review.

## gno.land/pkg/gnoweb/handler_http.go:625 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L625) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L625) · Warning

`/docs` is a [shipped alias](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/app.go#L30) onto `/u/docs`, and `docs` on gnoland-1 satisfies none of [the three rules](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L600-L602), so this return sends `https://gno.land/docs` to the error page. Repoint the alias, drop it, or register `docs`.

<details>

<summary>repro</summary>

The status `gno.land` serves for `/docs`, then the three answers the chain gives for `docs`:

```bash
curl -s -o /dev/null -w '%{http_code}\n' https://gno.land/docs
gnokey query vm/qpaths --data '@docs' --remote https://rpc.gno.land:443
gnokey query vm/qeval --data 'gno.land/r/sys/users.ResolveName("docs")' --remote https://rpc.gno.land:443
```

`@docs` comes back empty and `ResolveName` answers false, so none of the three rules holds for the name behind a URL answering 200:

```text
200
height: 0
data: 
height: 0
data: (nil *gno.land/r/sys/users.UserData)
(false bool)
```

The in-repo suite cannot see it: the test node loads `examples/`, which ships [`gno.land/r/docs/security_patterns`](https://github.com/gnolang/gno/blob/876762bdf/examples/gno.land/r/docs/security_patterns/gnomod.toml#L1), so `@docs` is non-empty there and the gate admits `/u/docs`.

</details>

## SKIP gno.land/pkg/gnoweb/handler_http.go:575 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L575) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L575) · Missing test

Missing test: no `evalFunc` stub in the package returns a payload outside the two [the switch](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L567-L571) recognises, so replacing this arm with `return true, nil` leaves the package green.

Skipped: `Missing test` is a deep-round band and this round ran the standard preset; the [500](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L1056-L1058) this arm raises stays unpinned.

<details>

<summary>test cases</summary>

```go
// An unrecognised ResolveName payload is an error, not a "no": pin the status
// it surfaces as, so the arm cannot be widened without a red test.
func TestHTTPHandler_GetUserView_UnexpectedResolveNamePayload(t *testing.T) {
	t.Parallel()

	rr := getUserPage(t, &stubClient{
		listPathsFunc: func(context.Context, string, int) ([]string, error) {
			return []string{""}, nil
		},
		evalFunc: func(context.Context, string, string) ([]byte, error) {
			return []byte(`("alice" string)`), nil
		},
	}, "/u/alice")

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "Gnome alice")
}
```

</details>

## SKIP gno.land/pkg/gnoweb/app_test.go:97 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/app_test.go#L97) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/app_test.go#L97) · Nit

Nit: `/u/moul001` and `/u/zoo_ma123` answer 200 only because [`genesis_txs.jsonl`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/genesis/genesis_txs.jsonl#L2-L3) registers those two names, neither of them holds a package, and neither file names the other.

Skipped: the fix is wording inside a code comment, which changes no behaviour.

## SKIP gno.land/pkg/gnoweb/components/template.go:56 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/components/template.go#L56) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/components/template.go#L56) · Nit

Nit: one account reads two ways: [`truncMiddle`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/components/views/overview.html#L45) keeps six runes each side of an ellipsis, and [`CreateUsernameFromBech32`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L580-L585) decodes the address first and keeps four each side of three dots.

Skipped: `components/template.go` carries no line of this diff, and both shorteners predate the branch.

## SKIP gno.land/pkg/gnoweb/handler_http.go:548 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L548) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L548) · Nit

Nit: `maxUsernameLen` keeps its own `64` beside [`maxNameLen`](https://github.com/gnolang/gno/blob/876762bdf/examples/gno.land/r/sys/users/store.gno#L30), so the two can drift and a name the registry accepts loses its page.

Skipped: the registry's own cap is the same `64`, so the drift is latent and the fix is a comment naming the upstream.

## gno.land/pkg/gnoweb/handler_http.go:603 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L603) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L603) · Nit

Nit: [`Username()`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/weburl/url.go#L290-L296) returns `Path[3:]` and `GetUserView` reads no other part of the URL, so `/u/alice:anything` serves the profile of `/u/alice`, one page under unboundedly many crawlable URLs. Reject a non-empty `gnourl.Args`, or redirect to the bare `/u/<name>`.

## gno.land/pkg/gnoweb/handler_http.go:649 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L649) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L649) · Nit

Nit: [`isAddress`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L607) already holds the answer [`CreateUsernameFromBech32`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L580-L586) recomputes, so an address page decodes and checksums twice and the helper's guard can drift from this caller's.

```suggestion
	if isAddress {
		// An address is displayed shortened: g1ab...wxyz.
		username = username[:4] + "..." + username[len(username)-4:]
	}
```

<details>

<summary>what comes out with it</summary>

The exported helper has one non-test caller, this line, so applying the suggestion retires the helper and its table test: +4/-45 measured over `handler_http.go` (+4/-10) and `handler_http_test.go` (-35), `gofmt -l` silent and the package green. `TestHTTPHandler_GetUserView_Address` already asserts `g1ma...dlf5` through the handler, so the shortening stays covered.

</details>

## gno.land/pkg/gnoweb/handler_http_test.go:2037 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http_test.go#L2037) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http_test.go#L2037) · Nit

Test: `resolveNamePayload(false)` ends on the same `(false bool)` line as the `unknown name` literal, and [`userExists`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L566-L567) keeps only that last line, so the two cases test the same thing. Give `renamed alias` a last line `unknown name` cannot produce.

## gno.land/pkg/gnoweb/handler_http_test.go:2076 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http_test.go#L2076) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http_test.go#L2076) · Nit

Test: nothing here records that `evalFunc` ran, so its two assertions never fire and deleting [the registry lookup](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L618-L627) leaves this test green. An `evalCalled` flag asserted after the request pins it.

<details>

<summary>repro: the gate deleted</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6206 -R gnolang/gno
# handler_http.go:618-627 is the whole gate
sed -i '618,627d' gno.land/pkg/gnoweb/handler_http.go
go test ./gno.land/pkg/gnoweb/ -run GetUserView -count=1 -v 2>&1 | grep -E '^\s*--- (PASS|FAIL): '
git checkout -- gno.land/pkg/gnoweb/handler_http.go
```

The admit case passes with the gate gone, and the refuse cases are what catch the deletion.

```text
--- PASS: TestHTTPHandler_GetUserView_Address (0.00s)
--- PASS: TestHTTPHandler_GetUserView_RegisteredWithoutPackages (0.00s)
--- FAIL: TestHTTPHandler_GetUserView_LookupFailureIsNotA404 (0.00s)
# …
--- FAIL: TestHTTPHandler_GetUserView_NotAUser (0.00s)
    --- FAIL: TestHTTPHandler_GetUserView_NotAUser/unknown_name (0.00s)
    --- FAIL: TestHTTPHandler_GetUserView_NotAUser/no_registry (0.00s)
    --- FAIL: TestHTTPHandler_GetUserView_NotAUser/renamed_alias (0.00s)
```

</details>

## SKIP gno.land/pkg/gnoweb/client.go:117 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/client.go#L117) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/client.go#L117) · Suggestion

Suggestion: [`Eval`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/client.go#L159-L165) formats `expr` into the `vm/qeval` data unchecked, so narrowing it to the `ResolveName` lookup the handler needs retires the contract rather than restating it.

Skipped: the one caller validates before it calls, so no read turns the contract into a defect.

## SKIP gno.land/pkg/gnoweb/client.go:231 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/client.go#L231) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/client.go#L231) · Suggestion

Suggestion: the split runs unconditionally, so an empty response becomes `[]string{""}` rather than nil and three call sites each spell their own filter for the phantom entry.

Skipped: line 231 is outside the diff, and the same edit as the section at `gno.land/pkg/gnoweb/handler_http.go:515` closes it.

## SKIP gno.land/pkg/gnoweb/handler_http.go:449 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L449) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L449) · Suggestion

Suggestion: `/u/<name>/` answers 404 from `GetDirectoryView` unless [`IsUser`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/weburl/url.go#L245) is tested before [`IsDir`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/weburl/url.go#L255), which holds for any path ending in `/`.

Skipped: line 449 is outside the diff, so GitHub rejects the anchor and the finding rides in the Body.

## gno.land/pkg/gnoweb/handler_http.go:515 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L515) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L515) · Suggestion

Suggestion: [`ListPaths`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/client.go#L231) renders an empty listing as `[]string{""}`, so this guard is the third filter for that blank, beside [`realm_directory.go:105`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/realm_directory.go#L105) and [`handler_http.go:864`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L864). Return nil from `ListPaths` instead: +9/-7 over two files, and two of the three filters go.

<details>

<summary>patch</summary>

```diff
--- a/gno.land/pkg/gnoweb/client.go
+++ b/gno.land/pkg/gnoweb/client.go
@@ -227,8 +227,15 @@ func (c *rpcClient) ListPaths(ctx context.Context, prefix string, limit int) ([]
 		return nil, err
 	}
 
+	// An empty listing is an empty body: Split would yield one blank path and
+	// every caller would have to filter it out.
+	trimmed := strings.TrimSpace(string(res))
+	if trimmed == "" {
+		return nil, nil
+	}
+
 	// update the paths to be relative to the root instead of the domain
-	paths := strings.Split(strings.TrimSpace(string(res)), "\n")
+	paths := strings.Split(trimmed, "\n")
 	for i, path := range paths {
 		paths[i] = strings.TrimPrefix(path, c.domain)
 	}
--- a/gno.land/pkg/gnoweb/handler_http.go
+++ b/gno.land/pkg/gnoweb/handler_http.go
@@ -511,11 +511,6 @@ func (h *HTTPHandler) buildContributions(ctx context.Context, username string) (
 	contribs := make([]components.UserContribution, 0, len(paths))
 	realmCount := 0
 	for _, raw := range paths {
-		// An empty listing is a single blank line, not a malformed path.
-		if raw == "" {
-			continue
-		}
-
 		trimmed := strings.TrimPrefix(raw, h.Static.Domain)
 		u, err := weburl.Parse(trimmed)
 		if err != nil {
@@ -861,7 +856,7 @@ func (h *HTTPHandler) GetPathsListView(ctx context.Context, gnourl *weburl.GnoUR
 		h.Logger.Debug("query paths", "prefix", prefix, "paths", len(paths))
 	}
 
-	if len(paths) == 0 || paths[0] == "" {
+	if len(paths) == 0 {
 		// Both the realm view and the source view funnel here when nothing is
 		// live at the path, so this is the one place that has to distinguish
 		// "never submitted" from "submitted, not approved yet".
```

The package stays green with it applied, `TestHTTPHandler_GetUserView_NotAUser` included: its stub returns `[]string{""}` and [`weburl.Parse`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/weburl/url.go#L346-L347) already rejects the empty string two lines below the deleted guard. `realm_directory.go`'s own filter stays: `TestRPCRealmDirectory_Paths` feeds a blank entry between two real paths, so removing that copy is a separate call.

</details>

## SKIP gno.land/pkg/gnoweb/handler_http.go:544 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L544) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L544) · Suggestion

Suggestion: the comment claims gnoweb reads the registry's rule, but a realm cannot import `gnovm/pkg/gnolang`, so [`store.gno:27`](https://github.com/gnolang/gno/blob/876762bdf/examples/gno.land/r/sys/users/store.gno#L27) is a copy that agrees, not the same rule.

Skipped: a finding on a code comment's own wording changes no behaviour.

## gno.land/pkg/gnoweb/handler_http.go:555 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L555) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L555) · Suggestion

Suggestion: this call reaches [`withQueryEvalMachine`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/sdk/vm/keeper.go#L1871), which finds [`r/sys/users`](https://github.com/gnolang/gno/blob/876762bdf/examples/gno.land/r/sys/users/users.gno#L6-L17) and builds a GnoVM machine rather than returning at [`ErrInvalidPkgPath`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/sdk/vm/keeper.go#L1882), so every shape-valid unknown name costs one machine build. [`maxGasQuery`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/sdk/vm/keeper.go#L1872) bounds one query and [`rpcSlots`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/client.go#L317) bounds concurrency, neither the rate, so a crawl over random names wants a short negative cache.

## gno.land/pkg/gnoweb/handler_http.go:607 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L607) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L607) · Suggestion

Suggestion: `isAddress` skips both the shape check and the registry arm, so any checksum-valid `g1` address with nothing on chain renders a profile [`head.html`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/components/layouts/head.html#L38) marks `index, follow`. Say so in [the godoc](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L600-L602), or require the address to hold an account, a package or a name.

## gno.land/pkg/gnoweb/handler_http.go:608-610 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L608-L610) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L608) · Suggestion

Suggestion: this check returns before `buildContributions`, so a namespace over `maxUsernameLen` loses its page, a length [`Re_gnoUserPkgPath`](https://github.com/gnolang/gno/blob/876762bdf/gnovm/pkg/gnolang/mempackage.go#L49-L54) does not bound and [`r/sys/names`](https://github.com/gnolang/gno/blob/876762bdf/examples/gno.land/r/sys/names/verifier.gno#L227) does not enforce where disabled. Run `buildContributions` first and gate only the `r/sys/users` lookup on the name shape.

## gno.land/pkg/gnoweb/handler_http.go:632 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L632) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L632) · Suggestion

Suggestion: this fetch reads neither `contribs` nor the registry answer, so it runs after both where [the errgroup `GetOverviewView` uses](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L1199-L1214) would overlap it with the paths query.

## SKIP gno.land/pkg/gnoweb/handler_http.go:1211 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L1211) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http.go#L1211) · Suggestion

Suggestion: this append takes the blank entry [`ListPaths`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/client.go#L231) returns for an empty listing, and only the `rel == ""` clause of [`buildSubpackages`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/components/overview_build.go#L149) keeps it off the page.

Skipped: line 1211 is outside the diff, and the same edit as the section at `gno.land/pkg/gnoweb/handler_http.go:515` closes it.

## gno.land/pkg/gnoweb/handler_http_test.go:2005-2164 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http_test.go#L2005-L2164) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http_test.go#L2005) · Suggestion

Test: the five gate tests differ only in the stub's answers and in three booleans, so one table over them is 118 lines against 160 and pins the queries per case.

```suggestion
// resolveNamePayload mirrors the raw vm/qeval output of ResolveName. The
// UserData line carries a "(false bool)" of its own, so a parser that searches
// the whole payload is fooled.
func resolveNamePayload(current bool) []byte {
	return fmt.Appendf(nil, `(&(struct{("g1manfred47kzduec920z88wfr64ylksmdcedlf5" .uverse.address),("alice" string),(false bool)} gno.land/r/sys/users.UserData) *gno.land/r/sys/users.UserData)
(%t bool)`, current)
}

func getUserPage(t *testing.T, client *stubClient, path string) *httptest.ResponseRecorder {
	t.Helper()

	handler, err := gnoweb.NewHTTPHandler(
		slog.New(slog.NewTextHandler(&testingLogger{t}, nil)),
		newTestHandlerConfig(t, client),
	)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
	return rr
}

// A page is served for an address, for a namespace holding packages and for a
// name r/sys/users resolves as current, never on a lookup the node could not
// answer. calls records every query, so each case pins what it must not ask.
func TestHTTPHandler_GetUserView_Gate(t *testing.T) {
	t.Parallel()

	answer := func(res []byte, err error) func(context.Context, string, string) ([]byte, error) {
		return func(context.Context, string, string) ([]byte, error) { return res, err }
	}
	// An empty prefix comes back as a single blank line; counting it as a
	// contribution would accept every name.
	blank := []string{""}

	for name, tc := range map[string]struct {
		path    string
		paths   []string
		eval    func(context.Context, string, string) ([]byte, error)
		code    int
		body    string
		notBody string
		calls   []string
	}{
		"unknown name": {
			path: "/u/alice", paths: blank, calls: []string{"list", "eval"},
			eval: answer([]byte("(nil *gno.land/r/sys/users.UserData)\n(false bool)"), nil),
			code: http.StatusNotFound, body: "user not found", notBody: "Gnome alice",
		},
		// A renamed-away name still resolves, but not as the current one.
		"renamed alias": {
			path: "/u/alice", paths: blank, calls: []string{"list", "eval"},
			eval: answer(resolveNamePayload(false), nil),
			code: http.StatusNotFound, body: "user not found", notBody: "Gnome alice",
		},
		// A chain that does not deploy the registry.
		"no registry": {
			path: "/u/alice", paths: blank, calls: []string{"list", "eval"},
			eval: answer(nil, gnoweb.ErrClientPackageNotFound),
			code: http.StatusNotFound, body: "user not found", notBody: "Gnome alice",
		},
		// A node that cannot answer is not an answer: a 404 here would delete a
		// real user's page.
		"lookup failure": {
			path: "/u/alice", paths: blank, calls: []string{"list", "eval"},
			eval: answer(nil, gnoweb.ErrClientTimeout), code: http.StatusRequestTimeout,
		},
		// A registered user who has not deployed anything yet still has a page.
		"registered without packages": {
			path: "/u/alice", calls: []string{"list", "eval", "realm"},
			eval: answer(resolveNamePayload(true), nil),
			code: http.StatusOK, body: "Gnome alice",
		},
		// An address is a namespace by construction: no registry lookup.
		"address": {
			path:  "/u/g1manfred47kzduec920z88wfr64ylksmdcedlf5",
			calls: []string{"list", "realm"}, code: http.StatusOK, body: "g1ma...dlf5",
		},
		// A segment that could never be a registered name never reaches the chain.
		"extra path segment": {path: "/u/foo/bar", code: http.StatusNotFound},
		"double separator":   {path: "/u/a--b", code: http.StatusNotFound},
		"trailing separator": {path: "/u/a-", code: http.StatusNotFound},
		"over the length cap": {
			path: "/u/" + strings.Repeat("a", 65), code: http.StatusNotFound,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var calls []string
			rr := getUserPage(t, &stubClient{
				listPathsFunc: func(context.Context, string, int) ([]string, error) {
					calls = append(calls, "list")
					return tc.paths, nil
				},
				evalFunc: func(ctx context.Context, pkgPath, expr string) ([]byte, error) {
					calls = append(calls, "eval")
					require.NotNil(t, tc.eval, "unexpected registry lookup")
					assert.Equal(t, "/r/sys/users", pkgPath)
					assert.Equal(t, `ResolveName("alice")`, expr)
					return tc.eval(ctx, pkgPath, expr)
				},
				realmFunc: func(context.Context, string, string) ([]byte, error) {
					calls = append(calls, "realm")
					return nil, gnoweb.ErrClientPackageNotFound
				},
			}, tc.path)

			assert.Equal(t, tc.code, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.body)
			if tc.notBody != "" {
				assert.NotContains(t, rr.Body.String(), tc.notBody)
			}
			assert.Equal(t, tc.calls, calls, "the page must ask exactly these queries")
		})
	}
}
```

<details>

<summary>equivalence of the table and the five tests</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6206 -R gnolang/gno
# apply the suggestion above, then:
gofmt -l gno.land/pkg/gnoweb/
go test ./gno.land/pkg/gnoweb/ -run TestHTTPHandler_GetUserView -count=1 -v
```

All ten subtests pass beside the two older `TestHTTPHandler_GetUserView*` tests, gofmt clean. Four mutations of the gate redden the same cases on the table as on the five originals, and one the originals miss: with `buildContributions` skipped for an address, the five stay green while the table fails on `[]string{"list", "realm"}` against `[]string{"realm"}`.

</details>

## gno.land/pkg/gnoweb/handler_http_test.go:2013 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http_test.go#L2013) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/handler_http_test.go#L2013) · Suggestion

Test: `getUserPage` replaces eleven lines of handler setup per test, and [`TestHTTPHandler_GetUserView`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http_test.go#L817-L827) and [`TestHTTPHandler_GetUserView_QueryPathsError`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http_test.go#L857-L867) keep that block inline; folding both in is 22 deletions for 2 insertions.

<details>

<summary>patch: the two older tests on the helper</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6206 -R gnolang/gno
git apply <<'EOF'
diff --git a/gno.land/pkg/gnoweb/handler_http_test.go b/gno.land/pkg/gnoweb/handler_http_test.go
--- a/gno.land/pkg/gnoweb/handler_http_test.go
+++ b/gno.land/pkg/gnoweb/handler_http_test.go
@@ -814,17 +814,7 @@ func TestHTTPHandler_GetUserView(t *testing.T) {
 		},
 	}
 
-	cfg := newTestHandlerConfig(t, client)
-
-	handler, err := gnoweb.NewHTTPHandler(
-		slog.New(slog.NewTextHandler(&testingLogger{t}, nil)),
-		cfg,
-	)
-	require.NoError(t, err)
-
-	req := httptest.NewRequest(http.MethodGet, "/u/testuser", nil)
-	rr := httptest.NewRecorder()
-	handler.ServeHTTP(rr, req)
+	rr := getUserPage(t, client, "/u/testuser")
 
 	assert.Equal(t, http.StatusOK, rr.Code)
 	body := rr.Body.String()
@@ -854,17 +844,7 @@ func TestHTTPHandler_GetUserView_QueryPathsError(t *testing.T) {
 		},
 	}
 
-	cfg := newTestHandlerConfig(t, client)
-
-	handler, err := gnoweb.NewHTTPHandler(
-		slog.New(slog.NewTextHandler(&testingLogger{t}, nil)),
-		cfg,
-	)
-	require.NoError(t, err)
-
-	req := httptest.NewRequest(http.MethodGet, "/u/testuser", nil)
-	rr := httptest.NewRecorder()
-	handler.ServeHTTP(rr, req)
+	rr := getUserPage(t, client, "/u/testuser")
 
 	// Should be 500 + internal error
 	assert.Equal(t, http.StatusInternalServerError, rr.Code)
EOF
gofmt -l gno.land/pkg/gnoweb/handler_http_test.go
go test ./gno.land/pkg/gnoweb/ -run TestHTTPHandler_GetUserView -count=1
git checkout -- gno.land/pkg/gnoweb/handler_http_test.go
```

2 insertions, 22 deletions, gofmt clean, every `GetUserView` test passing.

</details>

## SKIP gno.land/pkg/gnoweb/markdown/ext_mentions.go:27 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L27) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L27) · Suggestion

Suggestion: [`mentionNamePattern`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L28) excludes `-` from its trailing character class, so a hyphenated name [the page gate](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L608) accepts never becomes a mention link, and the golden pins [`@not-user`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/golden/ext_mentions/user_mentions.md.txtar#L14) unlinked.

Skipped: `markdown/ext_mentions.go` carries no line of this diff, and the edit that closes this direction is the one at `ext_mentions.go:28`.

## SKIP gno.land/pkg/gnoweb/markdown/ext_mentions.go:28 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L28) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L28) · Suggestion

Suggestion: `mentionNamePattern` admits consecutive underscores and names past [`maxUsernameLen`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L548), which [the page gate](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L608) refuses, so the golden pins [an over-long name](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/golden/ext_mentions/user_mentions.md.txtar#L13) as a rendered link that 404s.

Skipped: `markdown/ext_mentions.go` carries no line of this diff, so an inline anchor there is rejected at submit.

## SKIP gno.land/pkg/gnoweb/markdown/ext_mentions.go:49 [gh](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L49) · [↗](../../../../../.worktrees/gno-review-6206/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L49) · Suggestion

Suggestion: [`bech32.Decode`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L49) accepts any checksum-valid token, so a `g1` string whose payload is not 20 bytes becomes a [`/u/` link](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L114) that `crypto.AddressFromBech32`, the call [`GetUserView`](https://github.com/gnolang/gno/blob/876762bdf/gno.land/pkg/gnoweb/handler_http.go#L606) uses, refuses.

Skipped: `markdown/ext_mentions.go` carries no line of this diff, so an inline anchor there is rejected at submit.
