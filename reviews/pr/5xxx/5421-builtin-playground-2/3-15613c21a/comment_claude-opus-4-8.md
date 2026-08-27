# Review: PR [#5421](https://github.com/gnolang/gno/pull/5421)
Event: REQUEST_CHANGES
Status: not posted. Re-anchored to 15613c21a. The body-cap finding is fixed at this head and was dropped; the four remaining SKIPs were re-tested and are still open. On `post as an AI` the Body leads with `[AI review, opus 4.8]`, then `Status: REQUEST_CHANGES`.

## Body
The eval and funcs handlers keep a second rate limiter, while the sibling [`feature/state`](https://github.com/gnolang/gno/blob/15613c21a/gno.land/pkg/gnoweb/feature/state/ratelimit.go#L200-L213) already has a trusted-proxy-gated one. The body caps raised last round landed: [`maxEvalBodyBytes`](https://github.com/gnolang/gno/blob/15613c21a/gnovm/../gno.land/pkg/gnoweb/feature/playground/handler.go#L257) now wraps both eval and dry-run in `http.MaxBytesReader`, and `maxEvalExpressionLen` bounds the expression field.

The prior round's deflate cap and `?fork` bounds check out on 15613c21a. Re-verified the open items on the same sha: a 5 MiB eval body returns 200, 200 back-to-back funcs calls are never throttled, and rotating `X-Forwarded-For` from one peer bypasses the eval limiter.

## SKIP gno.land/pkg/gnoweb/feature/playground/ratelimit.go:88 [gh](https://github.com/gnolang/gno/blob/15613c21a/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88)
Already raised: https://github.com/gnolang/gno/pull/5421#discussion_r3512098587
`clientIP` trusts `X-Forwarded-For` with no trusted-proxy gate, so the eval limiter is bypassable per request; still open on 15613c21a.

## SKIP gno.land/pkg/gnoweb/feature/playground/handler.go:299 [gh](https://github.com/gnolang/gno/blob/15613c21a/gno.land/pkg/gnoweb/feature/playground/handler.go#L299) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L299)
Already raised: https://github.com/gnolang/gno/pull/5421#discussion_r3256256566
`serveFuncs` never calls the limiter, so `/_/api/funcs` forwards an unbounded `vm/qdoc` RPC; still open on 15613c21a.

## SKIP gno.land/pkg/gnoweb/feature/playground/ratelimit.go:40 [gh](https://github.com/gnolang/gno/blob/15613c21a/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L40) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L40)
Already raised: https://github.com/gnolang/gno/pull/5421#discussion_r3256267671
`pruneLoop` runs with no context or shutdown path; still open on 15613c21a.

## SKIP gno.land/pkg/gnoweb/feature/playground/handler.go:292 [gh](https://github.com/gnolang/gno/blob/15613c21a/gno.land/pkg/gnoweb/feature/playground/handler.go#L292) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L292)
Already raised: https://github.com/gnolang/gno/pull/5421#discussion_r3256269388
Backend RPC failure returns 200 with a JSON error on both eval (L243) and funcs (L264); still open on 15613c21a.

## gno.land/pkg/gnoweb/handler_http_test.go:1596 [gh](https://github.com/gnolang/gno/blob/15613c21a/gno.land/pkg/gnoweb/handler_http_test.go#L1596) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http_test.go#L1596)
The `with fork param` case asserts `/_/play?from=gno.land/r/demo/foo` echoes the path into the body, but [`GetPlaygroundView`](https://github.com/gnolang/gno/blob/15613c21a/gno.land/pkg/gnoweb/feature/playground/handler.go#L65-L94) [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L65-L94) reads only `code` and `z`, so nothing forks and the assertion passes purely on the URL echoing into the layout. The `?from=` query is dead; the real fork route is `?fork` on a package page, covered by [`TestHTTPHandler_ForkView`](https://github.com/gnolang/gno/blob/15613c21a/gno.land/pkg/gnoweb/handler_http_test.go#L1611-L1637) [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http_test.go#L1611-L1637). Repoint the case at `?fork` with a stub package, or drop it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5421 -R gnolang/gno
cat > gno.land/pkg/gnoweb/zz_from_repro_test.go <<'EOF'
package gnoweb_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/require"
)

func TestFromParamDead(t *testing.T) {
	cfg := newTestHandlerConfig(t, &stubClient{})
	h, err := gnoweb.NewHTTPHandler(slog.New(slog.NewTextHandler(&testingLogger{t}, nil)), cfg)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/_/play?from=ZZUNIQUEMARKER42", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body := rr.Body.String()
	t.Logf("status=%d marker-echoed=%v forked=%v", rr.Code,
		strings.Contains(body, "ZZUNIQUEMARKER42"),
		strings.Contains(body, "data-playground-fork-from-value"))
}
EOF
go test ./gno.land/pkg/gnoweb/ -run TestFromParamDead -v 2>&1 | grep -E "status=|PASS"
rm gno.land/pkg/gnoweb/zz_from_repro_test.go
```

```
    zz_from_repro_test.go:22: status=200 marker-echoed=true forked=false
--- PASS: TestFromParamDead (0.00s)
```
</details>
