cd "$1"
for v in gnoalias gnoalias_fixed; do
  sed -n '/^BLOG_MARKER=/p;/^latest_blogposts() {/,/^}/p;/^render_override() {/,/^}/p' $v.sh > f_$v.sh
  for r in comment heading mainnet; do
    cp home.mainnet.md home-override.md
    ( qrender() { cat render_$r.md; }; . ./f_$v.sh; BLOG_LINK_BASE= render_override ) > out_${AWKNAME}_${v}_$r.md
  done
done
