cd misc/deployments/home-alias
printf '<gno-columns>\n### [Upgrade notice\n<!--](/r/gnoland/blog:p/x)\n30 Oct 2026\n</gno-columns>\n' > render.md
sed -n '/^BLOG_MARKER=/p;/^latest_blogposts() {/,/^}/p;/^render_override() {/,/^}/p' gnoalias.sh > funcs.sh
cp home.mainnet.md home-override.md
qrender() { cat render.md; }; . ./funcs.sh; render_override | grep -E '^<!--|^## ' | sed 's/<gno-icon[^>]*> //; s/](.*//'
