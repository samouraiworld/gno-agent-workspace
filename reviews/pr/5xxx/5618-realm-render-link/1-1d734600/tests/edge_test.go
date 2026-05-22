package components

// Drop into gno.land/pkg/gnoweb/components/ and run.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TrimSuffix trims trailing "/" — covered for "/r/foo/" path.
func TestDirectoryView_RealmTrailingSlash(t *testing.T) {
	view := DirectoryView("/r/demo/blog/", []string{"a.gno"}, 1, DirLinkTypeSource, ViewModeRealm)
	tc, _ := view.Component.(*TemplateComponent)
	d, _ := tc.data.(DirData)
	assert.True(t, d.IsRealm)
	assert.Equal(t, "/r/demo/blog", d.RenderURL, "trailing slash must be trimmed once")
}

// User paths (/u/...) must not get a render link.
func TestDirectoryView_UserPathNoRender(t *testing.T) {
	view := DirectoryView("/u/alice", []string{"x.gno"}, 1, DirLinkTypeSource, ViewModeRealm)
	tc, _ := view.Component.(*TemplateComponent)
	d, _ := tc.data.(DirData)
	assert.False(t, d.IsRealm, "/u/ paths must not be flagged as realm")
	assert.Empty(t, d.RenderURL)
}

// Render link must be hidden in explorer mode even for realms.
func TestDirectoryView_RealmExplorerHidesRender(t *testing.T) {
	view := DirectoryView("/r/demo/blog", []string{"sub"}, 1, DirLinkTypeFile, ViewModeExplorer)
	var buf strings.Builder
	assert.NoError(t, view.Render(&buf))
	out := buf.String()
	assert.NotContains(t, out, `class="b-inline-btn">Render</a>`,
		"explorer mode must not show Render button")
}

// Empty path edge — should not panic, should not flag as realm.
func TestDirectoryView_EmptyPath(t *testing.T) {
	view := DirectoryView("", []string{"a.gno"}, 1, DirLinkTypeSource, ViewModeRealm)
	tc, _ := view.Component.(*TemplateComponent)
	d, _ := tc.data.(DirData)
	assert.False(t, d.IsRealm)
	assert.Empty(t, d.RenderURL)
}

// Realm root (/r/) — currently IsRealm=true, RenderURL="/r" after trim.
// Pin behavior to surface if it matters. /r/ alone shouldn't reach
// DirectoryView in practice; this just documents.
func TestDirectoryView_RealmRootBehavior(t *testing.T) {
	view := DirectoryView("/r/", []string{}, 0, DirLinkTypeSource, ViewModeRealm)
	tc, _ := view.Component.(*TemplateComponent)
	d, _ := tc.data.(DirData)
	assert.True(t, d.IsRealm)
	assert.Equal(t, "/r", d.RenderURL)
}
