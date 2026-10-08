import re
page = open('.shots/page.html').read()
names = re.findall(r'<gno-icon name="([^"]+)"', open('.shots/icons.md').read())
pr = re.findall(r'<svg class="gno-icon".*?</svg>', page, re.S)
assert len(pr) == len(names), (len(pr), names)
src = open('gno.land/pkg/gnoweb/markdown/icons/vendored.svg').read()
rows = []
for n, p in zip(names, pr):
    m = re.search(r'<symbol id="ico-%s" ([^>]*)>(.*?)</symbol>' % re.escape(n), src, re.S)
    s = '<svg class="gno-icon" xmlns="http://www.w3.org/2000/svg" %s>%s</svg>' % (m.group(1), m.group(2))
    rows.append(f'<tr><td class="n">{n}</td><td>{p}</td><td>{s}</td></tr>')
html = f'''<!doctype html><html><head><meta charset="utf-8"><link rel="stylesheet" href="/public/main.css">
<style>body{{background:#fff;padding:24px;font-family:sans-serif}} table{{border-collapse:collapse}}
td,th{{padding:6px 28px;text-align:center;font-size:18px}} td.n{{text-align:left;font-family:monospace;white-space:nowrap}}
td .gno-icon{{width:96px;height:96px;vertical-align:middle}} th{{font-weight:600}}</style></head>
<body><table><tr><th></th><th>this PR, as gnoweb renders it</th><th>source, vendored.svg as-is</th></tr>
{"".join(rows)}</table></body></html>'''
open('.shots/static/compare.html', 'w').write(html)
print(len(rows), 'rows')
labels = ['battery', 'half', 'lock', 'wallet', 'calculator', 'menu', '']
def line(svgs):
    return ' '.join((l + ' ' if l else '') + s for l, s in zip(labels, svgs))
srcs = []
for n in names:
    m = re.search(r'<symbol id="ico-%s" ([^>]*)>(.*?)</symbol>' % re.escape(n), src, re.S)
    srcs.append('<svg class="gno-icon" xmlns="http://www.w3.org/2000/svg" %s>%s</svg>' % (m.group(1), m.group(2)))
text = f'''<!doctype html><html><head><meta charset="utf-8"><link rel="stylesheet" href="/public/main.css">
<style>body{{background:#fff;padding:8px 24px}} .k{{display:inline-block;width:19em;white-space:nowrap;font-family:monospace;font-size:.8em;color:#666}}</style></head>
<body><div class="c-realm-view"><p><span class="k">this PR, as gnoweb renders it</span>{line(pr)}</p>
<p><span class="k">source, vendored.svg as-is</span>{line(srcs)}</p></div></body></html>'''
open('.shots/static/compare-text.html', 'w').write(text)
