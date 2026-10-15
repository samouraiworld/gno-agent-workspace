# Findings in posting order, from round assemble: 1 to post, 0 SKIP, 0 refuted kept out

## gno.land/cmd/gnoweb/main_test.go:116 [gh](https://github.com/gnolang/gno/blob/dfdeaf6764dae0a81d61023bad81f894ef31f555/gno.land/cmd/gnoweb/main_test.go#L116) · Missing test
State: CONFIRMED, band: Missing test, angle: removed
TL;DR: the strict-CSP test checks only connect-src 'self', so a broken RPC source in connect-src passes
Check: in a scratch worktree at dfdeaf676, delete ` %s/abci_query` from the cspTemplate in gno.land/cmd/gnoweb/main.go and run `go test ./gno.land/cmd/gnoweb/ -run SecureHeaders -count=1`: expect green at head (gap), red with master's assertion restored as `connect-src 'self' http://example.com/abci_query`
Details: Deleting ` %s/abci_query` as the finder proposed fails go vet (Sprintf arg count), so the judge mangled the path instead: `%s/abci_query` -> `%s/abci_querx` at main.go:359. TestSecureHeadersMiddlewareStrict stays green at head; the same mutation at the merge base 21136dcae fails main_test.go:117; at head with the base's assertion `connect-src 'self' http://example.com/abci_query` restored it fails again, and unmutated with that assertion it passes. The PR parent 643dcb30d already carried the weaker assertion; the merge kept it over master's.
Evidence: head mutated: `ok github.com/gnolang/gno/gno.land/cmd/gnoweb 0.307s`; base mutated: `--- FAIL: TestSecureHeadersMiddlewareStrict ... main_test.go:117: Expected Content-Security-Policy connect-src to contain 'self' and the RPC node`; head mutated + base assertion: `--- FAIL: TestSecureHeadersMiddlewareStrict`; head unmutated + base assertion: `ok ... 0.233s`
Artifact: tests/judge-1-connect-src-pin.sh
