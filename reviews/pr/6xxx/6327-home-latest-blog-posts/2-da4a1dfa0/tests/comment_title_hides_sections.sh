cat > examples/gno.land/r/gnoland/blog/newline_title_test.gno <<'EOF'
package blog

import "testing"

func TestNewlineTitle(cur realm, t *testing.T) {
	clearState(t)
	testing.SetOriginCaller(adminAddr)
	testing.SetRealm(testing.NewUserRealm(adminAddr))
	ModAddPost(cross(cur), "b", "B", "body", "2026-10-29T00:00:00Z", "", "")
	ModAddPost(cross(cur), "c", "C", "body", "2026-10-28T00:00:00Z", "", "")
	ModAddPost(cross(cur), "upgrade", "Upgrade notice\n<!--", "body", "2026-10-30T00:00:00Z", "", "")
	println("RENDER-BEGIN")
	println(Render(""))
	println("RENDER-END")
}
EOF
(cd examples && go run ../gnovm/cmd/gno test -v -run TestNewlineTitle ./gno.land/r/gnoland/blog 2>&1) | sed -n '/RENDER-BEGIN/,/RENDER-END/p' | sed '1d;$d' > render.md
sed -n '/^BLOG_MARKER=/p;/^latest_blogposts() {/,/^}/p;/^render_override() {/,/^}/p' misc/deployments/home-alias/gnoalias.sh > funcs.sh
mkdir -p home && cp misc/deployments/home-alias/home.mainnet.md home/home-override.md
(cd home && qrender() { cat ../render.md; } && . ../funcs.sh && BLOG_LINK_BASE= render_override > home.md)
go build -o gnoweb-bin ./gno.land/cmd/gnoweb
./gnoweb-bin --aliases=/=static:$PWD/home/home.md --bind 127.0.0.1:8899 > gnoweb.log 2>&1 &
until curl -sf -o /dev/null http://127.0.0.1:8899/; do sleep 1; done
curl -s http://127.0.0.1:8899/ | grep -o "<h2[^>]*>.*</h2>" | sed "s/<[^>]*>//g"
kill %1
