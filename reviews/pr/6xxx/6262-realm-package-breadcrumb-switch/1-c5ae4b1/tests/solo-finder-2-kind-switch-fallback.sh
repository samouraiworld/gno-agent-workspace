#!/usr/bin/env bash
# solo-finder-2: where the kind-switch menu opens in a browser without CSS
# anchor positioning, using the committed bundle public/main.css.
#
# The @supports-not fallback (top/left) comes before the .b-kind-switch rule,
# whose `inset: auto` (bundled as left:auto;left:anchor(left);top:auto;top:anchor(bottom))
# resets top/left once the anchor() declarations are dropped as invalid.
#
# Repro from a plain clone of gnolang/gno at c5ae4b13a8883303a382d6b1ae1c61b929fd048c:
#   cp solo-finder-2-breadcrumb-analytics_test.go gno.land/pkg/gnoweb/zz_finder2_analytics_test.go
#   FINDER2_HTML=/tmp/page.html go test ./gno.land/pkg/gnoweb -run TestFinder2_BreadcrumbAnalytics -count=1
#   ./solo-finder-2-kind-switch-fallback.sh /tmp/page.html gno.land/pkg/gnoweb/public/main.css
#
# Runs Chromium twice: on the bundle as shipped, and on a copy where every
# anchor( is renamed xanchor(, so the anchor() declarations fail to parse and
# @supports not (...) holds, as in a browser without anchor positioning
# (Safari before 26, for one). The blink flag CSSAnchorPositioning does not
# switch the feature off in Chromium 153. showPopover() is a layout probe,
# not a user-input run.
set -euo pipefail
page=$1
css=$(realpath "$2")
out=$(mktemp -d)
sed 's/anchor(/xanchor(/g' "$css" >"$out/noanchor.css"
for mode in shipped noanchor; do
[ $mode = shipped ] && href=$css || href=$out/noanchor.css
{
	echo "<link rel=\"stylesheet\" href=\"file://$href\">"
	cat "$page"
	cat <<'JS'
<script type="module">
const m = document.getElementById("kind-switch-menu");
const b = document.querySelector(".kind-switch");
m.showPopover();
const r = m.getBoundingClientRect(), br = b.getBoundingClientRect();
const cs = getComputedStyle(m);
const pre = document.createElement("pre");
pre.id = "finder2-out";
pre.textContent = [
	"fallback @supports active: " + !CSS.supports("top: " + (document.querySelector("link").href.includes("noanchor") ? "x" : "") + "anchor(bottom)"),
	"menu top/left px: " + Math.round(r.top) + "/" + Math.round(r.left),
	"button bottom/left px: " + Math.round(br.bottom) + "/" + Math.round(br.left),
	"computed top/left: " + cs.top + "/" + cs.left,
].join("\n");
document.body.append(pre);
</script>
JS
} >"$out/probe.html"
	echo "== $mode"
	chromium --headless=new --disable-gpu --no-sandbox --allow-file-access-from-files \
		--window-size=1280,800 --user-data-dir="$out/profile" \
		--virtual-time-budget=3000 --dump-dom "file://$out/probe.html" 2>/dev/null |
		sed -n '/id="finder2-out"/,/<\/pre>/p'
done
rm -rf "$out"
