// Refactor of the five user-view tests PR 6206 adds to
// gno.land/pkg/gnoweb/handler_http_test.go (lines 2004-2164 at head
// 876762bdf2ea6635b27e2b0a42f26f9fdc54af24): 161 lines become 119, and every
// case additionally pins the exact set of chain queries the page made, so
// "no home realm fetch", "no registry lookup for an address" and "invalid
// names must not reach the chain" stop being three separate ad-hoc booleans.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno
//	cd gno
//	git fetch origin pull/6206/head && git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	# baseline: the five tests as the PR ships them
//	go test ./gno.land/pkg/gnoweb/ -run 'TestHTTPHandler_GetUserView' -count=1 -v
//	# apply: replace lines 2004-2164 of gno.land/pkg/gnoweb/handler_http_test.go
//	# with everything below the "---- replacement block ----" line of this file
//	python3 - <<'PY'
//	p = "gno.land/pkg/gnoweb/handler_http_test.go"
//	new = open("<this file>").read().split("---- replacement block ----\n", 1)[1]
//	lines = open(p).readlines()
//	open(p, "w").writelines(lines[:2003] + [new])
//	PY
//	gofmt -l gno.land/pkg/gnoweb/
//	go test ./gno.land/pkg/gnoweb/ -run 'TestHTTPHandler_GetUserView' -count=1 -v
//
// Measured at that sha with go1.25.9: gofmt clean, all ten subtests of
// TestHTTPHandler_GetUserView_Gate pass alongside the two pre-existing
// TestHTTPHandler_GetUserView* tests, "ok github.com/gnolang/gno/gno.land/pkg/gnoweb 0.030s".
//
// Equivalence check: with the `if raw == ""` guard of buildContributions
// (handler_http.go:515) deleted, the PR's five tests and this table are both
// still green, so the table detects what the originals detected.
//
// ---- replacement block ----

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
// name r/sys/users resolves as current — for nothing else, and never on a
// lookup the node could not answer. calls records every query the page made,
// so each case also pins what it must not ask.
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
