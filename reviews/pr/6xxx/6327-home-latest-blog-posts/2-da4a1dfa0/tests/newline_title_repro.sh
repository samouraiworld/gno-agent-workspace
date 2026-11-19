cat > examples/gno.land/r/gnoland/blog/newline_title_test.gno <<'EOF'
package blog

import "testing"

func TestNewlineTitle(cur realm, t *testing.T) {
	clearState(t)
	testing.SetOriginCaller(adminAddr)
	testing.SetRealm(testing.NewUserRealm(adminAddr))
	ModAddPost(cross(cur), "b", "B", "body", "2026-10-29T00:00:00Z", "", "")
	ModAddPost(cross(cur), "c", "C", "body", "2026-10-28T00:00:00Z", "", "")
	ModAddPost(cross(cur), "upgrade", "Upgrade notice\n\n## Action required: move your GNOT\n[Migrate now](/r/attacker/migrate)", "body", "2026-10-30T00:00:00Z", "", "")
	println("RENDER-BEGIN")
	println(Render(""))
	println("RENDER-END")
}
EOF
(cd examples && gno test -v -run TestNewlineTitle ./gno.land/r/gnoland/blog 2>&1) | sed -n '/RENDER-BEGIN/,/RENDER-END/p' | sed '1d;$d' > render.md
sed -n '/^latest_blogposts() {/,/^}/p' misc/deployments/home-alias/gnoalias.sh > latest.sh
qrender() { cat render.md; }; . ./latest.sh; BLOG_LINK_BASE= latest_blogposts
