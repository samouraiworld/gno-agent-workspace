Posted: https://github.com/gnolang/gno/pull/5421#discussion_r4281249964, a reply in the main_test.go:116 thread

> AI review, claude-opus-5-5, standard review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5421-builtin-playground-2/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5421-builtin-playground-2/5-dfdeaf676/claims.md) · Status: REQUEST CHANGES

The `with fork param` case at [`handler_http_test.go:1626`](https://github.com/gnolang/gno/blob/dfdeaf676/gno.land/pkg/gnoweb/handler_http_test.go#L1626) passes because the path echoes into the layout, since [`GetPlaygroundView`](https://github.com/gnolang/gno/blob/dfdeaf676/gno.land/pkg/gnoweb/feature/playground/handler.go#L65-L94) reads only `code` and `z` and no handler reads `?from=`. The fork route is `?fork` on a package page, which [`TestHTTPHandler_ForkView`](https://github.com/gnolang/gno/blob/dfdeaf676/gno.land/pkg/gnoweb/handler_http_test.go#L1642-L1667) covers, so this case pins nothing.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5421 -R gnolang/gno
cat > gno.land/pkg/gnoweb/zz_from_repro_test.go <<'GO'
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
GO
go test ./gno.land/pkg/gnoweb/ -run TestFromParamDead -count=1 -v 2>&1 | grep -E "status=|PASS"
rm gno.land/pkg/gnoweb/zz_from_repro_test.go
```

The marker comes back in the page while the fork attribute the playground template sets from `ForkFrom` does not.

```
    zz_from_repro_test.go:22: status=200 marker-echoed=true forked=false
--- PASS: TestFromParamDead (0.00s)
```
</details>
