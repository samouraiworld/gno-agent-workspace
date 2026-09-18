// Asserts the merge base still serves the "/docs" default alias with 200 on a
// chain that has no "@docs" namespace, which is what gno.land answers today
// (curl https://gno.land/docs -> 200, "Gnome docs", "All Packages 0").
// It passes at 7916d1dd6 and is the baseline the reviewed head turns into 404.
//
// Repro, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout 7916d1dd65f326efe46b5ad105412ce028768c4d
//	cp <this file> gno.land/pkg/gnoweb/zz_judge_docs_base_test.go
//	go test ./gno.land/pkg/gnoweb -run TestJudgeDocsAliasBaseIs200 -v
//	rm gno.land/pkg/gnoweb/zz_judge_docs_base_test.go
package gnoweb_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJudgeDocsAliasBaseIs200(t *testing.T) {
	t.Parallel()

	client := &stubClient{
		// gnoland-1: `gnokey query vm/qpaths --data "@docs"` answers with one
		// empty line, so the namespace holds nothing.
		listPathsFunc: func(context.Context, string, int) ([]string, error) {
			return []string{""}, nil
		},
		// gnoland-1 has no gno.land/r/docs/home either.
		realmFunc: func(context.Context, string, string) ([]byte, error) {
			return nil, errors.New("package not found")
		},
	}

	cfg := newTestHandlerConfig(t, client)
	cfg.Aliases = gnoweb.DefaultAliases
	h, err := gnoweb.NewHTTPHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), cfg)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs", nil))

	assert.Equal(t, http.StatusOK, rr.Code,
		"the base serves /docs -> /u/docs as an empty profile, not a 404")
	assert.Contains(t, rr.Body.String(), "Gnome docs")
}
