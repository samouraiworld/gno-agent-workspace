# Asserts: TestSecureHeadersMiddlewareStrict goes red when connect-src stops naming <remote>/abci_query.
# Measured: green at the reviewed head with the path mangled, red at the merge base and with the base's assertion restored.
# Fails at the reviewed head: the mutation below passes the suite.
# from a local clone of gnolang/gno:
gh pr checkout 5421 -R gnolang/gno
sed -i "s|%s/abci_query; form-action|%s/abci_querx; form-action|" gno.land/cmd/gnoweb/main.go
go test ./gno.land/cmd/gnoweb/ -run SecureHeaders -count=1   # expect FAIL; ok means the RPC source is unpinned
git checkout -- gno.land/cmd/gnoweb/main.go
