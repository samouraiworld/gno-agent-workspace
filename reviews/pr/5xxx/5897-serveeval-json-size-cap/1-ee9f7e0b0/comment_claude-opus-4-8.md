# Review: PR [#5897](https://github.com/gnolang/gno/pull/5897)
Event: APPROVE

## Body
Verified on ee9f7e0b0: booted the eval handler and drove each cap. A 2 MiB body, a 100 KiB expression, and a 2000-byte pkg_path each return 400, and a 64 KiB expression still passes. The suite runs none of these paths.

On the sizes you asked about: 1024 and 64 KiB are safely generous. Real pkg paths run to a few dozen bytes and expressions are short function calls, so both leave large headroom while still bounding what reaches the RPC node.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5897 -R gnolang/gno
cat > gno.land/pkg/gnoweb/feature/playground/caps_verify_test.go <<'EOF'
package playground

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCapsVerify(t *testing.T) {
	h := New(Deps{
		Client:  &stubClient{evalResult: []byte("ok")},
		Logger:  discardLogger(),
		Domain:  "gno.land",
		Remote:  "http://localhost:26657",
		ChainId: "test",
	})
	handler := h.EvalHandler()
	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/_/api/eval", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr
	}
	for _, c := range []struct{ name, body string }{
		{"body 2MiB", fmt.Sprintf(`{"pkg_path":"r/x","expression":"%s"}`, strings.Repeat("a", 2<<20))},
		{"expr 100KiB", fmt.Sprintf(`{"pkg_path":"r/x","expression":"%s"}`, strings.Repeat("a", 100*1024))},
		{"pkgpath 2000", fmt.Sprintf(`{"pkg_path":"%s","expression":"R()"}`, strings.Repeat("a", 2000))},
	} {
		if rr := post(c.body); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", c.name, rr.Code)
		} else {
			t.Logf("%s -> 400 %s", c.name, strings.TrimSpace(rr.Body.String()))
		}
	}
	if rr := post(fmt.Sprintf(`{"pkg_path":"r/x","expression":"%s"}`, strings.Repeat("a", 64*1024))); rr.Code != http.StatusOK {
		t.Errorf("expr==64KiB: got %d, want 200", rr.Code)
	}
}
EOF
go test ./gno.land/pkg/gnoweb/feature/playground/ -run TestCapsVerify -v
rm gno.land/pkg/gnoweb/feature/playground/caps_verify_test.go
```

```
    caps_verify_test.go:35: body 2MiB -> 400 {"error":"invalid request body"}
    caps_verify_test.go:35: expr 100KiB -> 400 {"error":"pkg_path or expression is too long"}
    caps_verify_test.go:35: pkgpath 2000 -> 400 {"error":"pkg_path or expression is too long"}
--- PASS: TestCapsVerify
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5897-serveeval-json-size-cap/1-ee9f7e0b0/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoweb/feature/playground/handler.go:225 [↗](../../../../../.worktrees/gno-review-5897/gno.land/pkg/gnoweb/feature/playground/handler.go#L225)
The body cap, the pkg_path cap, and the expression cap add no test, though the [`TestHandlerPlaygroundEval`](https://github.com/gnolang/gno/blob/ee9f7e0b0/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L100-L141) table just above tests every other eval case and the file already covers the fork cap and the decompression bomb. Without a test a later edit can relax or invert a cap and CI stays green. The three cases in the repro above are exactly the assertions to add.
</content>
