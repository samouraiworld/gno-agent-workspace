# Review: PR [#5421](https://github.com/gnolang/gno/pull/5421)
Event: REQUEST_CHANGES
Status: drafted 2026-09-07 against 4a0e7ff2a, not posted. Round 3's review is live at https://github.com/gnolang/gno/pull/5421#pullrequestreview-5061295856 as a COMMENT; its four SKIPs are re-measured here and unchanged, and the `?from=` finding it posted is still open, so neither is re-sent. On `post as an AI` the Body leads with `[AI review, opus 5, effort high] (not manually verified)`.

## Body
Merging master brought [#6088](https://github.com/gnolang/gno/pull/6088), which made `.app/simulate` verify signatures on any transaction carrying Gno source, and that is why this is not an approve.

## gno.land/pkg/gnoweb/client.go:285-286 [gh](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/client.go#L285-L286) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/client.go#L285-L286)
Critical: `Simulate` fills `tx.Signatures` with a public key and no signature bytes, which [`txCarriesCode`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoland/app.go#L1298) now refuses for `MsgRun`, so every Dry Run ends in `unauthorized error`. Take a signed transaction from the caller, or drop the button until a wallet can sign one.

![Dry Run failing on both a key name and an address](https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5421-builtin-playground-2/4-4a0e7ff2a/media/dry-run-never-succeeds.gif)

The clip is gnodev built from 4a0e7ff2a, driven headless. The ledger reads the `address` out of the request the page sent and the status the server returned, and the last row is the funded case.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno, with go and jq on PATH:
gh pr checkout 5421 -R gnolang/gno
ADDR=g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5
MNEMONIC='source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast'
TMP=$(mktemp -d)
go build -o "$TMP/gnokey" ./gno.land/cmd/gnokey
(cd contribs/gnodev && go build -o "$TMP/gnodev" .)
"$TMP/gnodev" local --web-listener 127.0.0.1:8888 --node-rpc-listener 127.0.0.1:26657 --no-watch >"$TMP/gnodev.log" 2>&1 &
DEV=$!
until curl -s -o /dev/null http://127.0.0.1:8888/; do sleep 3; done
cat > "$TMP/script.gno" <<'EOF'
package main

import "gno.land/r/gnoland/home"

func main() {
	println(home.Render(""))
}
EOF
printf '%s\n\n\n' "$MNEMONIC" | "$TMP/gnokey" add --home "$TMP/kb" --recover --insecure-password-stdin test1 >/dev/null 2>&1
printf '\n' | "$TMP/gnokey" maketx send --home "$TMP/kb" --send 1ugnot --to "$ADDR" --gas-fee 1000000ugnot --gas-wanted 2000000 --broadcast --chainid dev --remote 127.0.0.1:26657 --insecure-password-stdin test1 >/dev/null
echo "== gnoweb dry run, pubkey-only placeholder signature =="
jq -n --arg s "$(cat "$TMP/script.gno")" --arg a "$ADDR" '{pkg_path:"gno.land/r/gnoland/home", script:$s, address:$a}' | curl -s -w '\nHTTP %{http_code}\n' -X POST http://127.0.0.1:8888/_/api/dryrun -H 'Content-Type: application/json' -d @-
echo "== gnokey simulate of the same script, real signature =="
printf '\n' | "$TMP/gnokey" maketx run --home "$TMP/kb" --gas-fee 1000000ugnot --gas-wanted 200000000 --simulate only --broadcast --chainid dev --remote 127.0.0.1:26657 --insecure-password-stdin test1 "$TMP/script.gno" 2>&1 | tail -4
kill $DEV; rm -rf "$TMP"
```

The first block is the finding: the endpoint refuses the transaction it built, at HTTP 200. The second is the control, showing the node runs the same script once the transaction carries a real signature.

```
== gnoweb dry run, pubkey-only placeholder signature ==
{"error":"error encountered during simulation: unauthorized error"}

HTTP 200
== gnokey simulate of the same script, real signature ==
EVENTS:     []
INFO:       estimated gas usage: 144124346 (suggested, with 5% margin: 151330564), gas fee: 151331ugnot, current gas price: 1ugnot/1000gas

TX HASH:
```

The [`AnteOptions`](https://github.com/gnolang/gno/blob/4a0e7ff2a/tm2/pkg/sdk/auth/ante.go#L39-L56) doc names this case: a transaction the predicate selects "must carry a real signature, not a pubkey-only placeholder". The same button returned the rendered output on a gnodev built from 15613c21a, whose base predates the predicate.
</details>

## gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts:134 [gh](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L134) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L134)
The dry run posts this field as `address` and [`serveDryRun`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L372-L376) answers 400 to anything that is not bech32, while the field is [labelled "Key name or address" and placeholdered `mykey`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/run/templates/page.html#L53-L56) and [`_buildCmd`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L76-L101) drops it into a `gnokey` line where a key name is the ordinary value. An address is the only value both readers take, and the page does not say so.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno, with gnodev from this branch serving 127.0.0.1:8888:
curl -s -w '\nHTTP %{http_code}\n' -X POST http://127.0.0.1:8888/_/api/dryrun \
  -H 'Content-Type: application/json' \
  -d '{"pkg_path":"gno.land/r/gnoland/home","script":"package main\n\nfunc main() {}\n","address":"mykey"}'
```

The placeholder the field offers is refused before the request reaches the node.

```
{"error":"address must be a bech32 address"}

HTTP 400
```
</details>

## gno.land/pkg/gnoweb/feature/playground/handler_test.go:65-67 [gh](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L65-L67) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L65-L67)
Missing test: `stubClient` carries `Simulate` plus `simulateResult` and `simulateErr`, and nothing in the package sets or reads them, so `DryRunHandler` is the one endpoint with no case at all.

<details><summary>test cases</summary>

```go
// TestHandlerPlaygroundDryRun tests the POST /_/api/dryrun handler directly.
func TestHandlerPlaygroundDryRun(t *testing.T) {
	t.Parallel()

	h := New(Deps{
		Client:  &stubClient{simulateResult: &abci.ResponseDeliverTx{ResponseBase: abci.ResponseBase{Data: []byte("mock run")}}},
		Logger:  discardLogger(),
		Domain:  "gno.land",
		Remote:  "http://localhost:26657",
		ChainId: "test",
	})

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantResult string
		wantError  string
	}{
		{
			name:       "valid dry run",
			body:       `{"pkg_path":"r/mock/path","script":"package main\n\nfunc main() {}\n","address":"g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"}`,
			wantStatus: http.StatusOK,
			wantResult: "mock run",
		},
		{
			name:       "key name rejected",
			body:       `{"pkg_path":"r/mock/path","script":"package main\n","address":"mykey"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "address must be a bech32 address",
		},
		{
			name:       "empty script rejected",
			body:       `{"pkg_path":"r/mock/path","script":"","address":"g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "script is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/_/api/dryrun", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			h.DryRunHandler().ServeHTTP(rr, req)
			assert.Equal(t, tc.wantStatus, rr.Code)

			var resp dryRunResponse
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, tc.wantResult, resp.Result)
			assert.Contains(t, resp.Error, tc.wantError)
		})
	}
}
```
</details>

## SKIP gno.land/pkg/gnoweb/feature/playground/handler.go:299 [gh](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L299) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L299)
Already raised: https://github.com/gnolang/gno/pull/5421#discussion_r3256256566
`serveFuncs` never calls the limiter that `serveEval` and `serveDryRun` share, so `/_/api/funcs` forwards a `vm/qdoc` per call; 60 back-to-back calls from one address all return 200 at this head.

## SKIP gno.land/pkg/gnoweb/feature/playground/ratelimit.go:88 [gh](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88)
Already raised: https://github.com/gnolang/gno/pull/5421#discussion_r3512098587
`clientIP` trusts `X-Forwarded-For` with no trusted-proxy gate, so 30 eval calls rotating the header from one peer all pass where 30 without it stop at 10; since #6035 the same bucket guards the simulate endpoint. Settled at https://github.com/gnolang/gno/pull/5421#discussion_r3718851602, so this is a measurement and not a proposal.

## SKIP gno.land/pkg/gnoweb/feature/playground/ratelimit.go:40 [gh](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L40) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L40)
Already raised: https://github.com/gnolang/gno/pull/5421#discussion_r3256267671
`pruneLoop` runs with no context and no shutdown path, so every `playground.New` in a test leaks a goroutine; unchanged at this head.

## SKIP gno.land/pkg/gnoweb/feature/playground/handler.go:292 [gh](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L292) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L292)
Already raised: https://github.com/gnolang/gno/pull/5421#discussion_r3256269388
Eval, funcs and dry run all answer `200 {"error":…}` on a node failure, which is why `unauthorized error` reaches the Result pane styled like a script's own error.

## SKIP gno.land/pkg/gnoweb/handler_http_test.go:1626 [gh](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/handler_http_test.go#L1626) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http_test.go#L1626)
Already raised: https://github.com/gnolang/gno/pull/5421#discussion_r3889878022
The `with fork param` case still asserts a URL echo rather than a fork, and `?from=` is still read by no handler. Skipped: this reviewer posted it in the previous round and the thread is open.

## SKIP gno.land/pkg/gnoweb/feature/playground/handler.go:120 [gh](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L120) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L120)
Nit: `fileanme` is misspelled at L120, L126, L159 and L169. Skipped: cosmetic, and no linter enabled in `.github/golangci.yml` catches it.
