// Repro for the stale sticky comment. Comment returns an empty string for a
// plan with nothing to preview, so a push that removes the last realm change
// leaves the previous push comment in place, describing an older head.
//
// The publish side never recovers: .github/workflows/pr-preview.yml gates the
// render and the upload on `if: steps.plan.outputs.skip != 'true'`, so no
// artifact is uploaded; .github/workflows/pr-preview-publish.yml downloads it
// with `continue-on-error: true` and gates every later step on
// `steps.download.outcome == 'success'`, so the publish job stays green and
// silent, and its "Upsert the sticky comment" step would in any case exit 0 on
// `[ -s _preview/comment.md ] || ...`.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp b5-catalog-stale-sticky-comment_test.go misc/gnopreview/
//	cd misc/gnopreview && go test -run TestCatalogEmptyPlanCanClearTheComment -v .
package main

import (
	"strings"
	"testing"
)

func TestCatalogEmptyPlanCanClearTheComment(t *testing.T) {
	t.Parallel()
	// An empty plan must still produce a marked body, so the upsert step can
	// replace the stale comment instead of leaving it.
	got := Comment(&Plan{}, "https://example.test/pr-1", "1")
	if !strings.Contains(got, CommentMarker) {
		t.Errorf("Comment(empty plan) = %q; a body carrying %q is the only way to replace the previous push comment",
			got, CommentMarker)
	}
}
