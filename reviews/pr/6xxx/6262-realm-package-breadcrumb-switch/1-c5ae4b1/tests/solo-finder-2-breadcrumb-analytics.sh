#!/usr/bin/env bash
# solo-finder-2: which links of the breadcrumb still fire the breadcrumb_click
# analytics event, probed in Chromium with gnoweb's own compiled analytics.js.
#
# Repro from a plain clone of gnolang/gno at c5ae4b13a8883303a382d6b1ae1c61b929fd048c:
#   cp solo-finder-2-breadcrumb-analytics_test.go gno.land/pkg/gnoweb/zz_finder2_analytics_test.go
#   FINDER2_HTML=/tmp/page.html go test ./gno.land/pkg/gnoweb -run TestFinder2_BreadcrumbAnalytics -count=1
#   ./solo-finder-2-breadcrumb-analytics.sh /tmp/page.html gno.land/pkg/gnoweb/public/js/analytics.js
#
# element.click() is a probe of which listener fires, not a user-input run.
# Navigation is cancelled in a capture listener; the analytics delegate still
# runs in the bubble phase. Control: the namespace chip, an <a> inside the <ol>.
set -euo pipefail
page=$1
analytics=$2
out=$(mktemp -d)
{
	echo '<script>window.__fired=[];window.sa_event=(n)=>window.__fired.push(n);</script>'
	cat "$page"
	echo '<script type="module">'
	cat "$analytics"
	echo '</script>'
	cat <<'JS'
<script type="module">
addEventListener("click", (e) => e.preventDefault(), true);
const res = [];
const probe = (label, el) => {
	if (!el) { res.push(label + "=missing"); return; }
	const before = window.__fired.length;
	el.click();
	res.push(label + "=" + (window.__fired.length > before ? window.__fired.slice(before).join(",") : "none"));
};
probe("control <ol> anchor " + document.querySelector('ol[data-searchbar-target="breadcrumb"] a')?.getAttribute("href"),
	document.querySelector('ol[data-searchbar-target="breadcrumb"] a'));
probe("kind-switch button", document.querySelector(".kind-switch"));
for (const a of document.querySelectorAll("#kind-switch-menu a")) probe("menu " + a.getAttribute("href"), a);
const pre = document.createElement("pre");
pre.id = "finder2-out";
pre.textContent = res.join("\n");
document.body.append(pre);
</script>
JS
} >"$out/probe.html"
chromium --headless=new --disable-gpu --no-sandbox --user-data-dir="$out/profile" \
	--virtual-time-budget=3000 --dump-dom "file://$out/probe.html" 2>/dev/null |
	sed -n '/id="finder2-out"/,/<\/pre>/p'
rm -rf "$out"
