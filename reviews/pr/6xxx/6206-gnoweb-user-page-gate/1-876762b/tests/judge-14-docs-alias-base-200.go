// judge-14: the merge base serves /u/docs as a 200 profile page for the same
// chain answers that make the branch 404 it, so the status flip is the diff's.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout 7916d1dd65f326efe46b5ad105412ce028768c4d
//	cp <this file> gno.land/pkg/gnoweb/zz_judge14_base_test.go
//	go test ./gno.land/pkg/gnoweb/ -run TestJudge14_DocsIsAPageAtBase -count=1 -v
//
// The stub answers what rpc.gno.land answers for "docs" (2026-09-19): the
// namespace holds no package (vm/qpaths '@docs' -> empty) and there is no
// r/docs/home realm. Result at 7916d1dd6: PASS — 200, "Gnome docs".

package gnoweb_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJudge14_DocsIsAPageAtBase(t *testing.T) {
	// /docs is a shipped alias onto the user page.
	alias, ok := gnoweb.DefaultAliases["/docs"]
	require.True(t, ok, "/docs alias is gone")
	require.Equal(t, "/u/docs", alias.Value)

	client := &stubClient{
		// vm/qpaths '@docs' is empty on gnoland-1; ListPaths renders that as [""].
		listPathsFunc: func(context.Context, string, int) ([]string, error) {
			return []string{""}, nil
		},
		// there is no r/docs/home realm on gnoland-1
		realmFunc: func(context.Context, string, string) ([]byte, error) {
			return nil, errors.New("package not found")
		},
	}

	handler, err := gnoweb.NewHTTPHandler(
		slog.New(slog.NewTextHandler(&testingLogger{t}, nil)),
		&gnoweb.HTTPHandlerConfig{
			ClientAdapter: client,
			Renderer:      &rawRenderer{},
			Aliases:       gnoweb.DefaultAliases,
		},
	)
	require.NoError(t, err)

	for _, path := range []string{"/u/docs", "/docs"} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		t.Logf("%s -> %d", path, rr.Code)
		assert.Equal(t, http.StatusOK, rr.Code, "%s is a page at the merge base", path)
		assert.Contains(t, rr.Body.String(), "Gnome docs", "%s renders a fabricated profile", path)
	}
}
