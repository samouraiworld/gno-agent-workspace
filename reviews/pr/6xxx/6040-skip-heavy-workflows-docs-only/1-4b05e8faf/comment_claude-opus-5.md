# Review: PR [#6040](https://github.com/gnolang/gno/pull/6040)
Event: APPROVE

## Body
Replayed the path filters over the whole tracked tree at 4b05e8faf and at its merge base, and the before and after sets match the table in the description row for row, including `ci / val-scenarios` surviving on [#6039](https://github.com/gnolang/gno/pull/6039). `master` carries no required status checks, `required_status_checks` is empty on both `contexts` and `checks`, so a pull request that starts no workflow merges instead of hanging.

No dir workflow lists [`_ci-go.yml`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/_ci-go.yml#L1) or [`_ci-gno.yml`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/_ci-gno.yml#L1) in its `paths`, and those two define lint, build and test for every module. A change to either used to fall through to [`ci / e2e`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-e2e.yml#L9-L24) and [`deploy / pages`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/deploy-pages.yml#L16-L23); after this it starts [actionlint](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/meta-actions-lint.yml#L10-L11) and nothing else. The root [`Makefile`](https://github.com/gnolang/gno/blob/4b05e8faf/Makefile#L104) is in the same position, and it holds the `tidy` target [`ci / codegen-verify`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-codegen-verify.yml#L39) runs.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6040-skip-heavy-workflows-docs-only/1-4b05e8faf/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## .github/workflows/ci-e2e.yml:12 [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-e2e.yml#L12)
`Dockerfile` is listed here and [`.dockerignore`](https://github.com/gnolang/gno/blob/4b05e8faf/.dockerignore#L1-L12) is not, so no pull request workflow now runs when it changes. `.dockerignore` alone decides which files reach the image, since [the build context is the repository root](https://github.com/gnolang/gno/blob/4b05e8faf/misc/e2e/docker-compose.yml#L3-L6) and [the Dockerfile does `COPY . ./`](https://github.com/gnolang/gno/blob/4b05e8faf/Dockerfile#L13). `ci / val-scenarios` [builds from the same root context](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-val-scenarios.yml#L79-L83) and [its list](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-val-scenarios.yml#L20-L29) has the same gap.

## .github/workflows/ci-dir-gnovm.yml:16-21 [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-dir-gnovm.yml#L16-L21)
[`gnovm/tests/files/extern/*`](https://github.com/gnolang/gno/blob/4b05e8faf/gnovm/tests/files/extern/redeclaration1/README.md?plain=1#L1) is a third tree where `.md` is mempackage content, [read](https://github.com/gnolang/gno/blob/4b05e8faf/gnovm/pkg/test/imports.go#L157-L160) through the same [`MustReadMemPackage`](https://github.com/gnolang/gno/blob/4b05e8faf/gnovm/pkg/gnolang/mempackage.go#L847) call as a stdlib, and the re-include names only two. Nothing breaks today: the file is carried into the mempackage and never parsed.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6040 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_md_test.go <<'EOF'
package gnolang

import "testing"

func TestMdIsMemPackageContent(t *testing.T) {
	for _, dir := range []string{"../../tests/files/extern/redeclaration1", "../../stdlibs/errors"} {
		names := []string{}
		for _, f := range MustReadMemPackage(dir, "filetests/extern/x", MPStdlibProd).Files {
			names = append(names, f.Name)
		}
		t.Logf("%s -> %v", dir, names)
	}
}
EOF
go test ./gnovm/pkg/gnolang/ -run TestMdIsMemPackageContent -v
rm gnovm/pkg/gnolang/zz_md_test.go
```

```
=== RUN   TestMdIsMemPackageContent
    zz_md_test.go:11: ../../tests/files/extern/redeclaration1 -> [README.md redeclaration.gno redeclaration2.gno]
    zz_md_test.go:11: ../../stdlibs/errors -> [README.md errors.gno gnomod.toml join.gno wrap.gno]
```
</details>
