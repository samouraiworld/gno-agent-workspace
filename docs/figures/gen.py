#!/usr/bin/env python3
"""Draws the figures beside gnovm-architecture.md.

Run it from anywhere: python3 docs/figures/gen.py. It rewrites every *.svg beside it. Each
figure states what the document states, at gnolang/gno 5a51b55f4; the numbers
that come from a run are named in the figure. The site (../figures.js) inlines
these files and reads their data-note attributes, so a note written here shows
up there as hover text and nowhere else.
"""
import html
import os

W = 760
SANS = 6.7      # px per character, 13px sans
SMALL = 5.9     # 11.5px sans
MONO = 7.25     # 12px mono
# Wider estimates for wrapping new text, so it still fits in system fonts
# wider than Helvetica: SF Pro, Segoe UI, Inter.
SAFE_SMALL = 6.4
SAFE_SANS = 7.3
SAFE_MONO_S = 6.7

STYLE = """
.f-bg{fill:#fbfaf7}
.f-box{fill:#fff;stroke:#c9c4b6;stroke-width:1}
.f-box.acc{fill:#e4f1ec;stroke:#226b5b}
.f-box.warn{fill:#fbeaea;stroke:#b3423f}
.f-box.mut{fill:#f1efe9}
.f-box.dash{stroke-dasharray:4 3}
.f-t{font:13px ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;fill:#1f2422;white-space:pre}
.f-t.b{font-weight:700}
.f-t.h{font-size:15px;font-weight:700}
.f-t.s{font-size:11.5px;fill:#4f5652}
.f-t.acc{fill:#226b5b}
.f-t.warn{fill:#b3423f}
.f-m{font:12px ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;fill:#1f2422;white-space:pre}
.f-m.b{font-weight:700}
.f-m.s{font-size:11px;fill:#4f5652}
.f-m.acc{fill:#226b5b}
.f-m.warn{fill:#b3423f}
.f-ar{stroke:#6b726e;stroke-width:1.4;fill:none;marker-end:url(#f-ah)}
.f-ar.acc{stroke:#226b5b;marker-end:url(#f-ah2)}
.f-ar.warn{stroke:#b3423f;marker-end:url(#f-ah3)}
.f-ar.thin{stroke-width:1;stroke:#b9b4a6}
.f-ar.dash{stroke-dasharray:4 3}
.f-ln{stroke:#c9c4b6;stroke-width:1;fill:none}
.f-hot{cursor:pointer}
"""

DEFS = """<defs>
<marker id="f-ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill="#6b726e"/></marker>
<marker id="f-ah2" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill="#226b5b"/></marker>
<marker id="f-ah3" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill="#b3423f"/></marker>
</defs>"""


def esc(s):
    return html.escape(str(s), quote=True)


def wrap(text, width):
    """Greedy word wrap to at most width characters per line."""
    words = text.split()
    lines, cur = [], ''
    for w in words:
        if cur and len(cur) + 1 + len(w) > width:
            lines.append(cur)
            cur = w
        else:
            cur = (cur + ' ' + w) if cur else w
    if cur:
        lines.append(cur)
    return lines


class Fig:
    def __init__(self, name, h, title=None):
        self.name = name
        self.h = h
        self.parts = []
        if title:
            self.text(20, 24, title, 'f-t h')

    def raw(self, s):
        self.parts.append(s)

    def open_hot(self, note=None, id=None, cls=''):
        attrs = ' class="f-hot %s"' % cls if note else (' class="%s"' % cls if cls else '')
        if id:
            attrs += ' id="%s"' % esc(id)
        if note:
            attrs += ' data-note="%s" tabindex="0"' % esc(note)
        self.raw('<g%s>' % attrs)

    def close(self):
        self.raw('</g>')

    def rect(self, x, y, w, h, cls='', r=6, id=None):
        idattr = ' id="%s"' % esc(id) if id else ''
        self.raw('<rect x="%g" y="%g" width="%g" height="%g" rx="%g" class="f-box %s"%s/>'
                 % (x, y, w, h, r, cls, idattr))

    def text(self, x, y, s, cls='f-t', anchor='start', id=None):
        a = '' if anchor == 'start' else ' text-anchor="%s"' % anchor
        idattr = ' id="%s"' % esc(id) if id else ''
        self.raw('<text x="%g" y="%g" class="%s"%s%s>%s</text>' % (x, y, cls, a, idattr, markup(s)))

    def lines(self, x, y, lines, cls='f-t s', lh=15, anchor='start'):
        for i, l in enumerate(lines):
            self.text(x, y + i * lh, l, cls, anchor)
        return y + len(lines) * lh

    def box(self, x, y, w, h, title=None, lines=(), cls='', note=None, id=None,
            tcls='f-t b', lcls='f-t s', lh=14, pad=10, nowrap=False, cw=None, para=0):
        """A rounded box with a title and wrapped lines. Returns the bottom y of the text."""
        if note or id:
            self.open_hot(note, id)
        self.rect(x, y, w, h, cls)
        ty = y + 18
        if title:
            self.text(x + pad, ty, title, tcls)
            ty += lh + 3
        else:
            ty = y + 15
        cw = cw or (MONO if 'f-m' in lcls else (SMALL if ' s' in lcls else SANS))
        maxc = max(8, int((w - 2 * pad) / cw))
        for l in lines:
            for piece in ([l] if nowrap else wrap(l, maxc)):
                self.text(x + pad, ty, piece, lcls)
                ty += lh
            ty += para
        if note or id:
            self.close()
        return ty

    def arrow(self, x1, y1, x2, y2, cls='', label=None, lcls='f-t s', dx=0, dy=-5):
        self.raw('<path d="M%g %gL%g %g" class="f-ar %s"/>' % (x1, y1, x2, y2, cls))
        if label:
            self.text((x1 + x2) / 2 + dx, (y1 + y2) / 2 + dy, label, lcls, 'middle')

    def path(self, d, cls='', arrow=True):
        self.raw('<path d="%s" class="%s %s"/>' % (d, 'f-ar' if arrow else 'f-ln', cls))

    def code(self, x, y, lines, lh=17, cls='f-m'):
        for i, l in enumerate(lines):
            self.text(x, y + i * lh, l, cls)
        return y + len(lines) * lh

    def foot(self, y, text, x=20, w=720, lh=16):
        """Small text wrapped to the width, for the lines under a figure. Returns the next y."""
        return self.lines(x, y, wrap(text, int(w / SAFE_SMALL)), 'f-t s', lh=lh)

    def render(self):
        return ('<svg xmlns="http://www.w3.org/2000/svg" xml:space="preserve" viewBox="0 0 %d %d" width="%d" height="%d" '
                'font-family="ui-sans-serif, system-ui, sans-serif">\n<style>%s</style>\n%s\n'
                '<rect width="%d" height="%d" class="f-bg" rx="8"/>\n%s\n</svg>\n'
                % (W, self.h, W, self.h, STYLE, DEFS, W, self.h, '\n'.join(self.parts)))


def markup(s):
    """[[text|note]] becomes a hot tspan carrying the note; everything else is escaped."""
    out = []
    i = 0
    s = str(s)
    while i < len(s):
        if s.startswith('[[', i):
            j = s.index(']]', i)
            text, _, note = s[i + 2:j].partition('|')
            out.append('<tspan class="f-hot" data-note="%s">%s</tspan>' % (esc(note), esc(text)))
            i = j + 2
        else:
            k = s.find('[[', i)
            k = len(s) if k == -1 else k
            out.append(esc(s[i:k]))
            i = k
    return ''.join(out)


FIGS = {}


def fig(fn):
    FIGS[fn.__name__.replace('_', '-')] = fn
    return fn


# ---------------------------------------------------------------- where the VM sits

@fig
def vm_stack():
    f = Fig('vm-stack', 690)
    x, w, cx = 40, 400, 240
    f.box(x, 30, w, 46, 'Wallet', ['gnokey maketx call: a signed transaction'],
          note='The user signs a transaction: its messages, the gas wanted and a flat fee in ugnot. Nothing here knows about the VM.')
    f.arrow(cx, 76, cx, 111)
    f.text(cx + 8, 99, 'sent by RPC, after a trial run with .app/simulate', 'f-t s')
    f.box(x, 112, w, 46, 'Tendermint2, the consensus engine', ['mempool, proposer, prevote, precommit'],
          note='CheckTx admits a transaction to the mempool. One validator proposes a chain block; it is final once validators holding more than two thirds of the voting power sign it.')
    f.arrow(cx, 158, cx, 193)
    f.text(cx + 8, 181, 'ABCI', 'f-t s')
    f.path('M%g 177L469 177' % (cx + 40), 'thin dash')

    # the application: runTx, with the ante handler before the messages
    f.box(x, 194, w, 106, 'BaseApp, tm2/pkg/sdk', ['runTx: decode, ValidateBasic, then:'],
          note='The ante handler checks the gas wanted, takes the fee, verifies signatures and increments sequence numbers; gno.land adds the session and code-submission checks. A failed message keeps only the ante handler\'s writes.')
    f.box(52, 240, 182, 50, 'ante handler', ['gas, fee, signatures'], 'mut',
          note='Refuses a gas wanted above the chain block\'s limit, takes the whole fee, checks each signature and increments the account\'s sequence number, so the same signed bytes cannot run twice.')
    f.arrow(234, 265, 245, 265)
    f.box(246, 240, 182, 50, 'runMsgs', ['route each message'], 'mut',
          note='Looks up each message\'s handler by its route and calls it, one message after the other, stopping at the first failure.')
    f.arrow(270, 290, 270, 359)
    f.text(262, 330, 'route "vm"', 'f-t s', 'end')
    for px, pw, name in ((290, 40, 'auth'), (338, 42, 'bank'), (388, 52, 'params')):
        f.arrow(px + pw / 2, 290, px + pw / 2, 317, 'thin')
        f.open_hot('gno.land registers four routes: auth, bank, params and vm. Each goes to its own module\'s handler.')
        f.rect(px, 318, pw, 20, 'mut', r=4)
        f.text(px + pw / 2, 332, name, 'f-m s', 'middle')
        f.close()
    f.text(448, 332, 'the other routes', 'f-t s')

    f.box(x, 360, w, 72, 'vm keeper, gno.land/pkg/sdk/vm',
          ['five messages: MsgAddPackage, MsgCall, MsgRun, MsgEnablePackage, MsgRejectPackage',
           '14 query paths, such as vm/qrender'],
          note='The keeper owns one long-lived gno.Store, forked per transaction, and builds a fresh gno.Machine for each run of Gno code.',
          cw=SAFE_SMALL)
    f.arrow(cx, 432, cx, 469)
    f.text(cx + 8, 456, 'a fresh Machine for each run', 'f-t s')
    f.box(x, 470, w, 46, 'gno.Machine, gnovm/pkg/gnolang', ['parse, preprocess, execute, finalize'], 'acc',
          note='The interpreter state: stacks, the active package and realm, the allocator, the gas meter, and a Context the standard library reads.')
    f.arrow(cx, 516, cx, 539)
    f.box(x, 540, w, 46, 'gno.Store', ['per-transaction caches for objects, types, syntax trees'],
          note='The layer between the interpreter and the database. Forked before a transaction runs its messages, written back only if every message succeeded.')
    f.arrow(cx, 586, cx, 609)
    f.box(x, 610, w, 62, 'tm2 stores, then the database',
          ['plain store: objects', 'B+32 tree: package sources and hashes of shared objects'],
          note='Two key-value stores. Only the B+32 tree\'s contents feed the application hash.', nowrap=True)

    # the seven ABCI calls
    tx, ty = 470, 112
    calls = [('InitChain', 'at genesis: first state'), ('CheckTx', 'admit to the mempool'),
             ('BeginBlock', 'open a chain block'), ('DeliverTx', 'run one transaction'),
             ('EndBlock', 'gas price, validators'), ('Commit', 'save, return the hash'),
             ('Query', 'answer a read')]
    f.open_hot('ABCI is a Go interface with one method per event. For each committed chain block a node calls BeginBlock, DeliverTx once per transaction, EndBlock, then Commit, which returns the application hash.')
    f.rect(tx, ty, 270, 144, '')
    f.text(tx + 10, ty + 18, 'ABCI: one call per event', 'f-t b')
    for i, (name, what) in enumerate(calls):
        acc = ' acc' if name == 'Query' else ''
        f.text(tx + 10, ty + 38 + i * 16, name, 'f-m s' + acc)
        f.text(tx + 96, ty + 38 + i * 16, what, 'f-t s' + acc)
    f.close()
    # a query skips runTx and the ante handler
    f.path('M700 256L700 386Q700 396 690 396L442 396', 'acc dash')
    f.lines(452, 374, ['Query: no transaction, no fee,', 'over a read-only snapshot'], 'f-t s acc', lh=14)

    # the same VM with no chain
    f.box(470, 470, 270, 172, 'Without a chain',
          ['gno test, gno run and the filetest runner build the same Machine over an in-memory store.',
           'Each import loads on first use: gnovm/stdlibs, the given packages, and examples/ for gno run and the filetests.',
           'Mock context: chain id dev, height 123, timestamp 1234567890.'],
          'mut', note='gno test and gno run build a Machine the same way the keeper does, over an in-memory store.',
          cw=SAFE_SMALL, para=5)
    f.path('M470 493L442 493', 'dash')
    return f


# ---------------------------------------------------------------- source map

@fig
def source_map():
    bw, bh, rh, top = 168, 30, 80, 52
    cols = [20, 296, 572]
    # row and column of each box: every import points down the page, or
    # sideways from the VM to the leaves beside it
    slot = {
        'gnovm/cmd/gno': (0, 0), 'gno.land/pkg/gnoland': (0, 2),
        'gnovm/pkg/test': (1, 0), 'gnovm/pkg/transpiler': (1, 1), 'gno.land/pkg/sdk/vm': (1, 2),
        'gnovm/stdlibs': (2, 0), 'gnovm/pkg/doc': (2, 2),
        'gnovm/pkg/packages': (3, 1),
        'gnovm/pkg/gnomod': (4, 0), 'gnovm/pkg/gnolang': (4, 1), 'gnovm/pkg/benchops': (4, 2),
        'gnovm/pkg/parser': (5, 1),
    }
    # GOTOOLCHAIN=local go list -f '{{.ImportPath}} {{join .Imports " "}}' \
    #   ./gnovm/... ./gno.land/pkg/sdk/vm/... ./gno.land/pkg/gnoland/...
    # mapped onto the boxes by exact import path
    edges = {
        'gnovm/cmd/gno': ['gno.land/pkg/sdk/vm', 'gnovm/pkg/doc', 'gnovm/pkg/gnolang', 'gnovm/pkg/gnomod', 'gnovm/pkg/packages', 'gnovm/pkg/test', 'gnovm/pkg/transpiler'],
        'gnovm/pkg/gnolang': ['gnovm/pkg/benchops', 'gnovm/pkg/gnomod', 'gnovm/pkg/parser'],
        'gnovm/pkg/test': ['gnovm/pkg/gnolang', 'gnovm/pkg/packages', 'gnovm/stdlibs'],
        'gnovm/pkg/transpiler': ['gnovm/pkg/gnolang', 'gnovm/stdlibs'],
        'gnovm/pkg/doc': ['gnovm/pkg/gnolang', 'gnovm/pkg/gnomod', 'gnovm/pkg/packages'],
        'gnovm/stdlibs': ['gnovm/pkg/gnolang'],
        'gnovm/pkg/packages': ['gnovm/pkg/gnolang', 'gnovm/pkg/gnomod'],
        'gno.land/pkg/sdk/vm': ['gnovm/pkg/doc', 'gnovm/pkg/gnolang', 'gnovm/pkg/gnomod', 'gnovm/stdlibs'],
        'gno.land/pkg/gnoland': ['gno.land/pkg/sdk/vm', 'gnovm/pkg/gnolang', 'gnovm/pkg/packages'],
    }
    notes = {
        'gnovm/pkg/gnolang': 'The VM itself. It imports only the manifest reader, the parser copy and the optional benchmarking hooks, so nothing above it is needed to run Gno.',
        'gnovm/pkg/parser': 'Go\'s go/parser, patched to charge gas per token. A leaf.',
        'gnovm/stdlibs': 'The standard library sources and their Go native bodies. It needs the VM for the value types the natives marshal.',
        'gnovm/pkg/test': 'The test harness: store construction, filetest directives, package loading.',
        'gnovm/cmd/gno': 'The command line tool. It imports everything, including the keeper, which it uses for the query commands.',
        'gno.land/pkg/sdk/vm': 'The keeper. The only chain code that constructs a Machine.',
        'gno.land/pkg/gnoland': 'The node application: mounts the stores, installs the ante handler and registers the routes.',
        'gnovm/pkg/transpiler': 'Gno to Go source translation, behind gno tool transpile.',
        'gnovm/pkg/doc': 'The gno doc engine, also served by vm/qdoc.',
        'gnovm/pkg/packages': 'Package discovery on disk.',
        'gnovm/pkg/gnomod': 'The gnomod.toml manifest. A leaf.',
        'gnovm/pkg/benchops': 'Per-opcode timing when compiled in. A leaf.',
    }
    nrows = 1 + max(r for r, c in slot.values())
    h = top + (nrows - 1) * rh + bh + 44
    f = Fig('source-map', h, 'Direct imports between the documented directories, from go list')
    pos = {n: (cols[c], top + r * rh) for n, (r, c) in slot.items()}
    down = [(s, t) for s, ts in edges.items() for t in ts if slot[s][0] < slot[t][0]]
    side = [(s, t) for s, ts in edges.items() for t in ts if slot[s][0] == slot[t][0]]

    def spread(n, others, key):
        # one point per arrow along a box's edge, ordered left to right
        others = sorted(others, key=key)
        x = pos[n][0]
        return {o: x + 14 + (bw - 28) * (i + .5) / len(others) for i, o in enumerate(others)}

    outx = {s: spread(s, [t for a, t in down if a == s], lambda t: (pos[t][0], -pos[t][1])) for s in slot}
    inx = {t: spread(t, [s for s, b in down if b == t], lambda s: (pos[s][0], pos[s][1])) for t in slot}

    def gaps(r):
        # the stretches of row r that no box covers, where an arrow may pass
        occ = sorted(cols[c] for rr, c in slot.values() if rr == r)
        out, x = [], 8
        for o in occ:
            if o - 8 > x + 8:
                out.append((x, o - 8))
            x = o + bw + 8
        out.append((x, 752))
        return out

    # an arrow crossing a row passes through the gap nearest its straight line
    routes, users = {}, {}
    for s, t in down:
        x0, y0 = outx[s][t], pos[s][1] + bh
        x1, y1 = inx[t][s], pos[t][1]
        way = []
        for r in range(slot[s][0] + 1, slot[t][0]):
            want = x0 + (x1 - x0) * (top + r * rh + bh / 2 - y0) / (y1 - y0)
            g = min(gaps(r), key=lambda g: 0 if g[0] <= want <= g[1] else min(abs(want - g[0]), abs(want - g[1])))
            way.append((r, g))
            users.setdefault((r, g), []).append(((s, t), want))
        routes[(s, t)] = (x0, y0, x1, y1, way)
    # arrows sharing a gap get evenly spaced lanes in it
    lane = {}
    for (r, (a, b)), lst in users.items():
        lst.sort(key=lambda e: e[1])
        if b > 740:
            b = a + 12 * len(lst) + 10
        for i, (e, want) in enumerate(lst):
            lane[(e, r)] = a + (b - a) * (i + 1) / (len(lst) + 1)

    def curve(xa, ya, xb, yb):
        k = (yb - ya) / 2
        return 'C%g %g %g %g %g %g' % (xa, ya + k, xb, yb - k, xb, yb)

    paths = []
    for (s, t), (x0, y0, x1, y1, way) in routes.items():
        d, xa, ya = 'M%g %g' % (x0, y0), x0, y0
        for r, g in way:
            xl = lane[((s, t), r)]
            d += curve(xa, ya, xl, top + r * rh - 8) + 'L%g %g' % (xl, top + r * rh + bh + 8)
            xa, ya = xl, top + r * rh + bh + 8
        d += curve(xa, ya, x1, y1)
        paths.append(('acc' if t == 'gnovm/pkg/gnolang' else 'thin', d))
    for cls, d in sorted(paths, key=lambda p: p[0] != 'thin'):
        f.path(d, cls)
    for s, t in side:
        (sx, sy), (tx, ty) = pos[s], pos[t]
        if tx < sx:
            f.path('M%g %gL%g %g' % (sx, sy + bh / 2, tx + bw + 1, ty + bh / 2), 'thin')
        else:
            f.path('M%g %gL%g %g' % (sx + bw, sy + bh / 2, tx - 1, ty + bh / 2), 'thin')
    for name, (x, y) in pos.items():
        f.open_hot(notes[name])
        f.rect(x, y, bw, bh, 'acc' if name == 'gnovm/pkg/gnolang' else '')
        f.text(x + bw / 2, y + 19, name, 'f-m', 'middle')
        f.close()
    f.text(20, h - 16, 'Green arrows: imports of the VM, gnovm/pkg/gnolang. Grey arrows: every other direct import.', 'f-t s')
    return f


# ---------------------------------------------------------------- package pipeline

@fig
def package_pipeline():
    f = Fig('package-pipeline', 378)
    stages = [
        ('1 · MemPackage', ['name, path, files', 'the path letter picks the kind: r realm, p pure, e ephemeral', 'gnomod.toml: module, gno 0.9'],
         'A MemPackage is a name, a path, a list of files and a type tag. No timestamps, owners or subdirectories, so the same files encode to the same bytes everywhere; the keeper stamps the creator and the height into gnomod.toml instead.'),
        ('2 · Type-check', ['go/types, from Go\'s standard library', '.gnobuiltins.gno declares realm, address, cross, revive', 'modes: strict, genesis-strict, relaxed'],
         'Gno\'s own parser is lax, so Go\'s type checker runs first, with a generated shim declaring the names Go does not know.'),
        ('3 · Parse', ['gnovm/pkg/parser, a patched go/parser, then Go2Gno', 'additions: ConstExpr, RefNode, StaticBlock', 'a node\'s span is its identity'],
         'The parser charges one gas per token, plus one per level of nesting. Go2Gno walks the go/ast tree and builds Gno nodes. Nodes that open a scope carry a StaticBlock recording the names declared there.'),
        ('4 · Preprocess', ['PredefineFileSet, then one walk per file', 'names become slots, constants fold, crossing rules are checked', 'block nodes saved by location'],
         'The compile step. Every name becomes a ValuePath, every constant expression a ConstExpr. A finished node is skipped if met again; a restart preprocesses every file again.'),
        ('5 · Run once', ['package variables in Kahn order', 'save, run each init.<n>, save again', 'types stored under their id'],
         'runMemPackage: variable declarations in dependency order, then every init function in file order with the deployer as the previous realm.'),
    ]
    bw, bh, gap = 136, 174, 14
    for i, (t, ls, n) in enumerate(stages):
        x = 20 + i * (bw + gap)
        f.box(x, 40, bw, bh, t, ls, 'acc' if i == 3 else '', note=n, lh=13, cw=SAFE_SMALL, para=5)
        if i < 4:
            f.arrow(x + bw, 121, x + bw + gap - 1, 121)
    f.box(420, 250, 320, 112, 'Saved package',
          ['package value and blocks, objects, types under tid:',
           'source under pkg:<path>, test files under pkg:<path>#allbutprod',
           'Block nodes are not stored: on restart every package is re-preprocessed to rebuild them.'],
          'mut', note='Only values, types and source are persisted. The syntax tree is rebuilt from source at node start.',
          lh=13, cw=SAFE_SMALL, para=4)
    f.arrow(688, 214, 688, 249)
    f.box(20, 250, 380, 70, 'Every later call',
          ['MsgCall on a stored package loads the saved values lazily and skips straight to execution. Steps 2 to 5 ran once, at upload.'],
          '', note='A call after upload does no type-check, no parse and no preprocess: it loads the package value and runs.',
          lh=13, cw=SAFE_SMALL)
    f.arrow(419, 285, 401, 285, 'dash')
    return f


# ---------------------------------------------------------------- preprocessed dump

SOURCE = """package names

var total int

func add(n int) int {
	sum := total
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			sum += i
		}
	}
	f := func() int { return sum + n }
	return f()
}

func main(cur realm) {
	total = 1
	println(add(3))
}""".replace('\t', '    ').split('\n')

DUMP = [
    'package names',
    'var [[total<!~VPBlock(2,0)>|Declared in the file block, so the path climbs one block to the package block and takes slot 0. The ! marks the defining occurrence; the ~ says the slot holds a heap item, which is how every package variable is stored.]] (const-type int)',
    'func add([[n~ (const-type int)|A parameter, slot 0 of add\'s block. The ~ says it lives in a heap item, because the closure below captures it.]]) [[.res.0|The unnamed result gets a hidden name and slot 1. A dot makes it unspellable from source.]] (const-type int) {',
    '    [[sum<!~VPBlock(1,2)>|Defined in add\'s own block (depth 1), slot 2: n is 0, .res.0 is 1, sum is 2, f is 3. Heap item, since the closure captures sum.]] := [[total<~VPBlock(3,0)>|Three hops up from add\'s block: file block, then package block, slot 0. The read goes through the heap item.]]',
    '    for [[i.loopvar<!VPBlock(1,0)>|The loop variable is renamed with a .loopvar suffix so it cannot collide with a same-named declaration in the body. Slot 0 of the for block. No ~: nothing captures it.]] := (const (0 int));',
    '        i.loopvar<VPBlock(1,0)> < [[n<~VPBlock(2,0)>|From the for block, n is one hop up, in add\'s block, slot 0.]]; i.loopvar<VPBlock(1,0)>++ {',
    '        if [[i.loopvar<VPBlock(2,0)>|The if arm has its own block, so i is now two hops up.]] % [[(const (2 int))|A literal the preprocessor folded into a ConstExpr carrying the finished typed value.]] == (const (0 int)) {',
    '            [[sum<~VPBlock(3,2)>|From the if block: for block, add\'s block, slot 2. Same variable as above, one more hop.]] += i.loopvar<VPBlock(2,0)>',
    '        }',
    '    }',
    '    [[f<!VPBlock(1,3)>|Slot 3 of add\'s block. A function value, not a heap item: nothing takes its address.]] := func func() .res.0 (const-type int){',
    '        return [[sum<~VPBlock(1,1)>|Inside the closure the captured sum sits in the closure\'s own block at slot 1, after its .res.0 at slot 0. Every call copies the captured heap items into a fresh block.]] + [[n<~VPBlock(1,2)>|The captured n, slot 2 of the closure\'s block.]]',
    '    }[[<sum<()~VPBlock(1,2)>, n<()~VPBlock(1,0)>>|The capture list. When the closure is created, the heap items at slots 2 and 0 of add\'s block are copied into FuncValue.Captures. The ()~ marks a closure capture.]]',
    '    return f<VPBlock(1,3)>()',
    '}',
    'func main([[cur (const-type .uverse.realm)|The crossing parameter. Its type is the realm interface from the universe block.]]) {',
    '    total<~VPBlock(3,0)> = (const (1 int))',
    '    [[(const (println func(...interface {})))|A builtin resolves at preprocess time into a constant holding the function value.]]([[add<VPBlock(3,1)>|From main\'s block: file block, package block, slot 1. Package slots: total 0, add 1, main 2.]]((const (3 int))))',
    '}',
]


@fig
def preprocess_dump():
    f = Fig('preprocess-dump', 700)
    f.text(20, 24, 'Source: names_filetest.gno, a realm at gno.land/r/demo/names', 'f-t h')
    half = 11
    f.code(20, 46, SOURCE[:half], lh=16)
    f.code(390, 46, SOURCE[half:], lh=16)
    f.text(20, 236, 'After preprocessing, as the filetest directive Preprocessed: prints it', 'f-t h')
    f.code(20, 258, DUMP, lh=17)
    y = 258 + len(DUMP) * 17 + 8
    f.rect(20, y, 720, 88, 'mut')
    f.lines(30, y + 18, [
        'name<VPBlock(d, i)>: climb d − 1 parent blocks, take slot i. Depth 0 means the universe block.',
        '!  the defining occurrence      ~  the slot holds a heap item      ()~  captured into a closure when it is created',
        '(const ...)  folded by the preprocessor      .loopvar  .res.0  .uverse  hidden names no source can spell',
        'Measured by running the filetest with gno test at this commit; the site shows a note for each highlighted token.',
    ], 'f-t s', lh=17)
    return f


# ---------------------------------------------------------------- init order

@fig
def init_order():
    f = Fig('init-order', 260)
    decls = [
        ('var a = mark("a", b+1)', 'a reads b'),
        ('var b = mark("b", f())', 'b calls f, which reads c'),
        ('func f() int { return c }', 'a function runs nothing at init; its reads count for its callers'),
        ('var c = mark("c", 2)', 'c reads nothing'),
    ]
    for i, (code, n) in enumerate(decls):
        y = 44 + i * 48
        f.box(20, y, 300, 36, None, [code], '', note=n, lcls='f-m', pad=10)
    f.text(20, 236, 'Dependencies recorded by codaInitOrderDeps, method bodies and method expressions included', 'f-t s')
    # dependency arrows on the right edge: a -> b, b -> f, f -> c
    for i in range(3):
        y1 = 44 + i * 48 + 18
        y2 = 44 + (i + 1) * 48 + 18
        f.path('M320 %g C360 %g 360 %g 320 %g' % (y1, y1, y2, y2))
    f.text(368, 62 + 24, 'reads', 'f-t s')
    f.text(368, 110 + 24, 'calls', 'f-t s')
    f.text(368, 158 + 24, 'reads', 'f-t s')
    f.box(440, 44, 300, 86, 'Kahn\'s sort, ties by declaration order',
          ['1  c = 2', '2  b = f() = 2', '3  a = b + 1 = 3'], 'acc', lcls='f-m', lh=16,
          note='A topological sort over the recorded reads. A min-heap breaks ties by declaration order. A cycle panics with the chain of names.')
    f.box(440, 146, 300, 74, 'Output of the filetest',
          ['init c 2', 'init b 2', 'init a 3'], 'mut', lcls='f-m', lh=15,
          note='Measured with gno test. The file order is a, b, f, c; the run order is c, b, a.')
    return f


# ---------------------------------------------------------------- types

@fig
def type_ids():
    f = Fig('type-ids', 466, 'What each Type implementation puts into its TypeID')
    # the rule shows on the card; the extra joins it in the hover note
    cards = [
        ('int', 'PrimitiveType', 'One of the fixed kinds. The untyped constant kinds, bigint and bigdec, are primitive types too.',
         'Also DataByteType, an internal view into one byte of a byte array, which has no id.'),
        ('[]int   *T   map[K]V', 'SliceType, PointerType, MapType', 'The id is built from the element ids. Channels have a type and no operations.',
         'ArrayType and ChanType complete the set.'),
        ('struct{ Key Key; Left Node }', 'StructType', 'The package path enters the id only for unexported field names: struct{ N int } from two packages is one type.',
         'An embedded field is named as written: struct{ Int } has field Int. Carries the declaring package path for its own use.'),
        ('func(cur realm, n int) string', 'FuncType', 'Parameter and result names are not part of the id. IsCrossing reads whether the first parameter is realm.',
         ''),
        ('interface{ Render(...) }', 'InterfaceType', 'Embedded interfaces are flattened into their methods, which are sorted by name before the id is formed.',
         'checkImplementedBy, billed per method, is the check behind a type assertion and a type switch.'),
        ('type InnerNode struct{ ... }', 'DeclaredType', 'id: gno.land/r/test.InnerNode. Declared inside a function: pkgpath[location].Name.',
         'Name, package path, base type, method list.'),
        ('address   realm   error', 'Declared in the universe block', 'address is declared on string. realm is an interface; its hidden .grealm holds an address, a path and the previous realm.',
         'A sub-realm\'s .grealm also holds its subpath and its parent. A .seal method no source can declare keeps user types out.'),
        ('RefType{ ID }', 'Placeholder', 'Stands in for a persisted type until the store loads it. Loading resolves it by id.',
         ''),
        ('8 · 8 · 128 · 128', 'Limits on type shapes', 'Embedding depth 8, type nesting depth 8, 128 methods on an interface, embedded included, 128 struct fields, promoted included.',
         'Checked when a type is built, they bound adversarial type shapes.'),
    ]
    cw, ch = 236, 122
    for i, (code, impl, rule, extra) in enumerate(cards):
        col, row = i % 3, i // 3
        x, y = 20 + col * (cw + 6), 40 + row * (ch + 10)
        f.open_hot(rule + (' ' + extra if extra else ''))
        f.rect(x, y, cw, ch, 'acc' if i == 5 else '')
        f.text(x + 10, y + 18, code, 'f-m b')
        f.text(x + 10, y + 34, impl, 'f-t s')
        pieces = wrap(rule, int((cw - 20) / SAFE_SMALL))
        assert len(pieces) <= 5, impl
        f.lines(x + 10, y + 50, pieces, 'f-t s', lh=14)
        f.close()
    f.text(20, 448, 'Equality is TypeID equality, never pointer equality: each transaction loads its own copy of a saved type.', 'f-t s')
    return f


# ---------------------------------------------------------------- typed value

TV_CASES = [
    ('x := 42', 'int', 'nil', '42', 'An int lives in the eight bytes: SetInt writes an int64 straight into N. No allocation.'),
    ('ok := true', 'bool', 'nil', '01 at byte 0', 'SetBool writes the bool into the first byte of N.'),
    ('f := 2.5', 'float64', 'nil', '0x4004000000000000', 'SetFloat64 stores the IEEE-754 bits in N. Arithmetic on them runs in software so every node computes the same bits.'),
    ('s := "hi"', 'string', 'StringValue "hi"', '0', 'A string is a Go string in V. Not an Object: it has no identity and is saved inside its holder.'),
    ('p := &node', '*InnerNode', 'PointerValue{TV, Base, Index}', '0', 'A pointer is a slot address: the base object, an array, struct, block, map or heap item, plus an index into it.'),
    ('xs := []int{1, 2}', '[]int', 'SliceValue{Base, Offset 0, Length 2, Maxcap 2}', '0', 'A slice is a window onto an ArrayValue, which is the Object with identity.'),
    ('v := InnerNode{...}', 'InnerNode', '*StructValue{ObjectInfo, Fields}', '0', 'A struct is an Object: ObjectInfo carries its id, owner, reference count and dirty marks. Copying it copies the fields into a new object.'),
    ('var e any', 'nil', 'nil', '0', 'A nil interface value is the undefined typed value: no type, no value. It serializes as {} in a realm trace.'),
    ('var e any = 7', 'int', 'nil', '7', 'An interface has no box: converting to an interface leaves T as the concrete type, so this slot is filled exactly as x := 7 is.'),
]


@fig
def typed_value():
    f = Fig('typed-value', 350, 'TypedValue{T, V, N}: nine values, three cells each')
    cw, ch = 236, 82
    for i, (code, T, V, N, note) in enumerate(TV_CASES):
        col, row = i % 3, i // 3
        x, y = 20 + col * (cw + 6), 40 + row * (ch + 8)
        f.open_hot(note)
        f.rect(x, y, cw, ch, 'acc' if i == 8 else '')
        f.text(x + 10, y + 17, code, 'f-m b')
        f.text(x + 10, y + 36, 'T  ' + T, 'f-m s')
        for k, piece in enumerate(wrap('V  ' + V, 30)[:2]):
            f.text(x + 10, y + 50 + k * 13, piece, 'f-m s')
        f.text(x + 10, y + 76, 'N  ' + N, 'f-m s')
        f.close()
    f.foot(322, 'T is the concrete type, V a Value, N eight bytes. Fixed-size numbers use N only; everything with identity is a Value that is also an Object.')
    return f


# ---------------------------------------------------------------- blocks and heap items

@fig
def blocks_heap():
    f = Fig('blocks-heap', 500, 'Blocks alive inside add(3)')

    def block(x, y, title, slots, note, cls='', w=210):
        h = 34 + 16 * len(slots)
        f.open_hot(note)
        f.rect(x, y, w, h, cls)
        f.text(x + 10, y + 17, title, 'f-t b')
        for i, s in enumerate(slots):
            f.text(x + 10, y + 34 + i * 16, s, 'f-m s')
        f.close()
        return h

    hp = block(20, 44, 'package block', ['0  total  → HeapItem', '1  add    FuncValue', '2  main   FuncValue'],
               'The bottom of every scope chain in this package. Its parent is the universe block. Package variables sit in heap items, which is why the dump marks them with ~.')
    hf = block(20, 44 + hp + 30, 'file block', ['(one per file)'],
               'One block per file, between the function blocks and the package block. It is why a package variable is three hops from a function body.')
    ha = block(20, 44 + hp + 30 + hf + 30, "add's block", ['0  n      → HeapItem', '1  .res.0', '2  sum    → HeapItem', '3  f      FuncValue'],
               'Taken from the machine\'s block pool by OpCall, or allocated when the pool has none that fits, with one slot per declared name. Each heap-marked slot starts with a heap item; writes go into the item, not over it.', 'acc')
    y_for = 44 + hp + 30 + hf + 30 + ha + 30
    hfor = block(20, y_for, 'for block', ['0  i.loopvar'],
                 'One per loop. At the end of each iteration the VM replaces every heap item in the init slots with a new one, which is how closures see Go 1.22 loop variables. Here i is not captured, so it is a plain slot.')
    # parent arrows
    f.arrow(230, 44 + hp + 30 + 10, 230, 44 + hp + 4, label='Parent', dx=28, dy=3)
    f.arrow(230, 44 + hp + 30 + hf + 30 + 10, 230, 44 + hp + 30 + hf + 4)
    f.arrow(230, y_for + 10, 230, y_for - 26)
    f.lines(20, y_for + hfor + 24, wrap('The if arm gets a block too, with no names of its own here.', 44), 'f-t s', lh=15)

    # heap items and the closure
    ay = 44 + hp + 30 + hf + 30
    f.box(330, ay, 170, 46, 'HeapItem', ['Value: n = 3'], 'mut', note='A one-slot box. Pointers and captures reference the item, so the value survives the block it was declared in.', id='hi-n')
    f.box(330, ay + 64, 170, 46, 'HeapItem', ['Value: sum'], 'mut', note='The same kind of box for sum. Assignments in the loop write into this item.', id='hi-sum')
    f.arrow(230, ay + 34 - 4, 329, ay + 23)
    f.arrow(230, ay + 34 + 32 - 4, 329, ay + 64 + 23)
    f.box(540, ay, 200, 92, 'FuncValue f', ['Captures: [HeapItem sum, HeapItem n]', 'body: return sum + n', 'source, type; no parent block'], '',
          note='Created by OpFuncLit. The capture list, sum then n, is copied out of add\'s block into Captures. Each call of f puts the same items into its new block, and a closure has no parent block, so a saved closure never references the block it was born in.', lh=14)
    f.arrow(540, ay + 40, 501, ay + 23, 'acc')
    f.arrow(540, ay + 50, 501, ay + 64 + 23, 'acc')
    f.box(540, ay + 110, 200, 60, '&sum', ['PointerValue{Base: HeapItem, Index 0}'], 'mut',
          note='Taking the address of a captured or escaping local yields a pointer whose base is the heap item, never the block.', lcls='f-m s')
    f.arrow(540, ay + 133, 501, ay + 64 + 30, 'dash')
    f.lines(330, ay + 200, wrap('When add returns, its block and the for and if blocks are popped, zeroed '
                                'and returned to the machine\'s block pool. '
                                'The two heap items stay alive inside f\'s captures. That is the whole memory model: '
                                'blocks are transient, heap items are what escape.', int(410 / SAFE_SMALL)), 'f-t s', lh=15)
    return f


# ---------------------------------------------------------------- machine trace

@fig
def machine_trace():
    f = Fig('machine-trace', 394)
    f.text(20, 24, 'func f(a, b int) int { x := a + b; return x }   called as f(2, 3)', 'f-m b')
    f.text(20, 44, 'The stacks after doOpEval has expanded a + b. Gas so far: 155, OpBody 73 plus OpEval 82.', 'f-t s')
    cols = [
        ('Ops', ['OpBody', 'OpDefine', 'OpAdd', 'OpEval', 'OpEval'], 'Opcodes wait here; the loop pops from the top. Pushes went in reverse, so OpEval for a runs first, then OpEval for b, then OpAdd, then OpDefine. OpBody stays beneath, sticky, and takes return x next.'),
        ('Exprs', ['a + b', 'b', 'a'], 'The binary expression stays underneath its operands until doOpAdd pops it. Each OpEval on a name pops that name.'),
        ('Stmts', [('f\'s body cursor', 'x := a + b')], 'OpBody holds the statement as the cursor\'s active one; doOpDefine pops it at the end.'),
        ('Values', ['f'], 'The function value f sits at the bottom for the whole call, and the return writes the result over it. After both names: f, 2, 3. doOpAdd then adds 3 into the 2 beneath it, leaving f, 5.'),
        ('Parent chain', ['f: a=2 b=3', 'file', 'package'], 'The blocks a name climbs: f\'s block, its parent the file block, then the package block. Name lookup ignores the Blocks stack and follows these parent pointers. OpDefine writes 5 into slot 3 of f\'s block, or into the heap item there if x were captured.'),
    ]
    cw, gap, cellh = 136, 12, 26
    base = 300
    for i, (title, cells, note) in enumerate(cols):
        x = 20 + i * (cw + gap)
        f.open_hot(note)
        f.rect(x, 60, cw, base - 60 + 4, 'mut', r=4)
        f.text(x + cw / 2, 78, title, 'f-t b', 'middle')
        cy = base - 26
        for k, c in enumerate(cells):
            # a stack's top runs next; the parent chain marks f's block, where x goes
            top = k == 0 if title == 'Parent chain' else k == len(cells) - 1
            if isinstance(c, tuple):
                # a cell holding a cursor: what it is, then its active statement
                f.rect(x + 8, cy - 18, cw - 16, cellh + 18, 'acc' if top else '', r=4)
                f.text(x + cw / 2, cy - 2, c[0], 'f-t s', 'middle')
                f.text(x + cw / 2, cy + 17, c[1], 'f-m', 'middle')
            else:
                f.rect(x + 8, cy, cw - 16, cellh, 'acc' if top else '', r=4)
                f.text(x + cw / 2, cy + 17, c, 'f-m', 'middle')
            cy -= cellh + 4
        if not cells:
            f.text(x + cw / 2, base - 20, '(empty)', 'f-t s', 'middle')
        f.close()
    y = f.foot(332, 'Frames: one for the call to f, recording the stack heights to unwind to. The top cell of each stack runs next.')
    y = f.foot(y, 'Gas per opcode, from machine.go: OpBody 73, OpEval 82 plus 4 per unit of depth for a name, int add 81, OpDefine 114 plus 79 per name: 601 for the statement.')
    f.text(20, y, 'The site steps through all seven states.', 'f-t s')
    return f


# ---------------------------------------------------------------- control flow

@fig
def control_flow():
    f = Fig('control-flow', 364)
    f.text(20, 24, 'for i := 0; i < n; i++ { body }', 'f-m b')
    f.box(20, 40, 200, 80, 'Frame', ['label, stack heights to unwind to'], '',
          note='Every for, range, switch and function call pushes a frame, so break, continue and return know how far to unwind.',
          lh=13, cw=SAFE_SMALL)
    f.box(236, 40, 214, 80, 'Block', ['i.loopvar, and the cursor bodyStmt: next statement index, condition, post statement'], '',
          note='The loop block owns a bodyStmt that remembers the next statement index, the condition and the post statement. The sticky OpForLoop walks that state.',
          lh=13, cw=SAFE_SMALL)
    # state machine
    sx = 20
    nodes = [('Cond', 'Evaluate the condition.'),
             ('next body statement', 'One body statement per turn of OpForLoop; the index advances.'),
             ('Post', 'Captured loop variables get fresh heap items, i++ runs, then back to the condition. continue jumps here.')]
    xs = [sx, sx + 150, sx + 360]
    ws = [100, 190, 76]
    for (t, n), x, w in zip(nodes, xs, ws):
        f.box(x, 140, w, 36, None, [t], 'acc', note=n, lcls='f-t b', pad=10)
    f.arrow(xs[0] + ws[0], 158, xs[1] - 1, 158)
    f.text(xs[0] + ws[0] + 12, 152, 'true', 'f-t s')
    f.arrow(xs[1] + ws[1], 158, xs[2] - 1, 158)
    f.path('M%g 176 C%g 215 %g 215 %g 176' % (xs[2] + 38, xs[2] + 38, xs[0] + 50, xs[0] + 50))
    f.text(20, 230, 'OpForLoop is sticky: it stays on the stack, so loops do not recurse', 'f-t s')
    f.box(20, 244, 430, 64, 'Cond false, or break',
          ['pop the block, pop the frame, drop the sticky opcode. break and continue pop frames until the loop with the right label.'],
          'mut', note='Leaving the loop restores the stacks to the heights the frame recorded.', lh=13, cw=SAFE_SMALL)
    # the others
    others = [
        ('if', 'pushes a block only, no frame; the block is expanded with the chosen arm\'s names',
         'An if pushes no frame, because no branch statement targets an if. OpPopBlock releases its block afterwards.'),
        ('switch', 'a frame and one shared block; cases tried one at a time; default runs last; a type switch compares type ids',
         'default runs only once every other clause has failed. fallthrough swaps in the next clause\'s names. A type switch also checks each interface case against the value\'s methods.'),
        ('range', 'one opcode per kind: arrays and slices, pointers to arrays, strings one rune per step, maps one entry per step',
         'Arrays and slices, with pointers to arrays under their own opcode; strings, one rune per step; maps, one list entry per step, in insertion order.'),
        ('goto', 'pops frames and blocks, cuts the stacks back to the heights of the target body, moves its cursor to the label',
         'The preprocessor counted the frames and blocks between the goto and its label, and the label\'s statement index.'),
    ]
    y = 36
    for t, d, n in others:
        f.box(470, y, 270, 74, t, [d], '', note=n, lh=13, cw=SAFE_SMALL)
        y += 80
    return f


# ---------------------------------------------------------------- function calls

CALL_STAGES = [
    ('1 · OpEval on the call', 'pushes OpPrecall, then an OpEval per argument in reverse, then one for the callee, so the callee evaluates first: Values f, a, b',
     'The call expression y := f(a, b). Because pushes go in reverse, the callee is read before any argument.'),
    ('2 · OpPrecall', 'peeks the callee under the arguments. A type: OpConvert, no frame. A function: push a frame and OpCall, plus OpEnterCrossing for a crossing function. A nil callee panics: call of nil function',
     'PushFrameCall also decides the storage realm for the call, the borrow rules. A call written f(cross(cur), ...) gets a fresh cur from installCrossingCur.'),
    ('3 · OpCall', 'a block from the machine\'s pool or freshly allocated, its parent the file block; captured heap items copied in; results zeroed; arguments popped into the first slots, variadic ones packed',
     'The block\'s slots hold the parameters, then the results, then the locals. The parent is the file block for a top-level function; a closure has none.'),
    ('4 · Body', 'Gno function: OpBody with the statements, plus a synthetic return when it declares no results. Native: OpReturn then OpCallNativeBody. defer: evaluate now, push a Defer on the frame',
     'A native function is a FuncValue with a Go body, attached by DefineNative for builtins, looked up in the store for the standard library.'),
    ('5 · OpReturn, four forms', 'OpReturn; OpReturnAfterCopy when results are named and return lists values; OpReturnFromBlock for a bare return; OpReturnToBlock plus the sticky OpReturnCallDefers when the frame holds defers',
     'The return statement and the frame choose the form.'),
    ('6 · Pop the frame', 'maybeFinalize saves the realm if the frame is a realm boundary; then the stacks are cut back to the recorded heights, the call\'s blocks released to the pool, the results moved down, the caller\'s package and realm restored',
     'This is where a realm call ends and its objects are saved.'),
]


@fig
def call_sequence():
    f = Fig('call-sequence', 370)
    f.text(20, 24, 'y := f(a, b)', 'f-m b')
    bw, bh = 232, 140
    for i, (t, d, n) in enumerate(CALL_STAGES):
        col, row = i % 3, i // 3
        x, y = 20 + col * (bw + 12), 40 + row * (bh + 30)
        f.box(x, y, bw, bh, t, [d], 'acc' if i in (1, 5) else '', note=n, id='call-%d' % (i + 1), lh=13, cw=SAFE_SMALL)
        if col < 2:
            f.arrow(x + bw, y + bh / 2, x + bw + 11, y + bh / 2)
    f.path('M%g %g L%g %g L%g %g L%g %g' % (20 + 2 * 244 + bw / 2, 40 + bh, 20 + 2 * 244 + bw / 2, 40 + bh + 15,
                                            20 + bw / 2, 40 + bh + 15, 20 + bw / 2, 40 + bh + 29))
    return f


# ---------------------------------------------------------------- panics

@fig
def panic_unwind():
    f = Fig('panic-unwind', 476)
    f.box(20, 30, 340, 72, 'panic(v) inside an opcode handler',
          ['pushPanic records the Exception, unwinds to the last call frame, pushes OpPanic2 and OpReturnCallDefers'],
          'warn', note='The cooperative path. Nothing on the Go stack is consumed.', lh=13, cw=SAFE_SMALL)
    f.box(400, 30, 340, 72, 'A Go panic carrying an Exception',
          ['from the value layer: runOnce recovers it and Run takes the same path, so half a million panicking defers cannot crash the node'],
          'mut', note='Machine.Panic or a direct panic with an Exception deep in values.go. The iterative-recovery record explains why.', lh=13, cw=SAFE_SMALL)
    f.arrow(190, 102, 190, 139)
    f.path('M570 102 L570 120 L360 120 L360 139')
    f.box(20, 140, 340, 50, None, ['the frame\'s deferred calls run, one at a time'], 'acc', note='doOpReturnCallDefers pops each Defer and runs it like a call.', lcls='f-t b', cw=7.8, lh=16)
    f.arrow(190, 190, 190, 219)
    f.box(20, 220, 340, 50, None, ['recover() called directly by a deferred function?'], '', note='The Go rule: only a direct call from the deferred function sees the exception.', lcls='f-t b', cw=7.8, lh=16)
    f.arrow(360, 245, 399, 245, label='yes', dy=-6)
    f.box(400, 220, 340, 50, None, ['exception cleared; the function returns normally'], 'acc', note='Recover returns the exception value and clears it. Exceptions chain to the previous one when a defer panics again.', lcls='f-t b', cw=7.8, lh=16)
    f.arrow(190, 270, 190, 299, label='no', dx=14, dy=4)
    f.box(20, 300, 340, 80, 'OpReturnCallDefers pops the frame',
          ['and re-raises in the caller. The next frame up runs its own defers: back to the top.'], '',
          note='The unwinding continues frame by frame, each one running its deferred calls first.', lh=13, cw=SAFE_SMALL)
    f.path('M10 340 L4 340 L4 165 L19 165', 'dash')
    f.box(400, 300, 340, 80, 'Realm boundary reached',
          ['the transaction aborts with UnhandledPanicError, unless a revive frame catches it. No caller recovers it, so one realm cannot catch another\'s refusal and retry.'],
          'warn', note='A panic that crosses a realm boundary is never recoverable by the caller. Tests use revive to observe the abort.', lh=13, cw=SAFE_SMALL)
    f.arrow(360, 340, 399, 340)
    y = f.foot(406, 'On gno.land a bounded printer renders the latest exception alone, with no user methods, 1024 bytes at most. Elsewhere every chained exception prints, oldest first, and String and Error run, at 4 gas per output byte.')
    f.foot(y, 'A panic while package variables initialize has no call frame to unwind to and fails the message at once. A Stacktrace keeps 128 calls with the middle elided.')
    return f


# ---------------------------------------------------------------- uverse

@fig
def uverse():
    f = Fig('uverse', 352)
    chain = [('if block', 'Depth 1'), ('for block', 'Depth 2'), ("add's block", 'Depth 3'), ('file block', 'Depth 4'), ('package block', 'Depth 5'), ('.uverse', 'Depth 0')]
    y = 30
    for i, (t, d) in enumerate(chain):
        cls = 'acc' if t == '.uverse' else ''
        f.box(20, y, 150, 30, None, [t], cls, lcls='f-t b', pad=10,
              note='A name written in the if arm reaches this block at this depth.' if t != '.uverse' else
              'The universe block, a package named .uverse at the root of every chain. A name no scope declares resolves here with Depth 0, and the preprocessor replaces each builtin name with a constant holding its value, so running code never looks a builtin up.')
        f.text(180, y + 19, d, 'f-t s')
        if i < len(chain) - 1:
            f.arrow(95, y + 30, 95, y + 44)
        y += 44
    f.text(20, 340, 'Depth counts hops from the block where the name is used, so the same name has a different depth in each scope.', 'f-t s')
    rows = [
        ('Types', 'bool string int int8 ... uint64 float32 float64 byte rune any error address realm',
         'The basic types are defined here, as type values in the block\'s slots.'),
        ('Values', 'true  false  nil  iota', 'Constants, resolved by the preprocessor into ConstExprs.'),
        ('Functions', 'append cap copy delete len make new print println panic recover',
         'Each is a FuncValue whose body is a Go closure over the Machine, installed by DefineNative.'),
        ('Gno additions', 'cross   attach (reserved, panics)   istypednil   revive',
         'cross checks that its argument is the cur of the innermost crossing call still running, or a sub-realm minted from it. revive works only when the machine enables it, which tests do.'),
        ('Hidden', '.cur   .origin   .grealm   .runtimeError', 'For the compiler\'s own use. The leading dot means user source cannot spell them.'),
    ]
    y = 30
    for t, body, n in rows:
        f.box(260, y, 480, 54, t, [body], 'mut' if t == 'Hidden' else '', note=n, lcls='f-m s', lh=13, cw=SAFE_MONO_S)
        y += 58
    return f


# ---------------------------------------------------------------- object ids

@fig
def object_ids():
    f = Fig('object-ids', 275)
    f.text(20, 24, 'PkgID', 'f-t h')
    steps = [
        ('"gno.land/r/test"', 'the package path', 118),
        ('SHA-256', 'a8ada09dee16 ... 59dad84c, 32 bytes', 128),
        ('keep 20 bytes', 'the top four bits become flags: standard library, immutable, internal, reserved', 196),
        ('08ada09dee16d791fd40 6d629fe29bb0ed084a30', 'PkgID: a user realm has no flag set, so a8 became 08', 232),
    ]
    x = 20
    notes = ['The path is the only input. The same path on any node yields the same id.',
             'Measured: the SHA-256 of the path starts with a8ad. The id in the zrealm1 trace starts with 08ad.',
             'Immutable covers the standard library, pure packages, _test overlays and .uverse; their objects are not counted from other packages.',
             'The id in the zrealm1 filetest trace, produced by the same path.']
    for (t, d, w), n in zip(steps, notes):
        f.box(x, 40, w, 80, None, [t, d], 'acc' if w == 232 else '', note=n, lcls='f-m s' if w == 232 else 'f-t s', lh=14, pad=8,
              cw=SAFE_MONO_S if w == 232 else SAFE_SMALL)
        if w != 232:
            f.arrow(x + w, 80, x + w + 12, 80)
        x += w + 14
    f.text(20, 150, 'ObjectID', 'f-t h')
    f.box(20, 166, 300, 56, None, ['PkgID : NewTime', '08ada09d...084a30:6  the struct created in zrealm1'], '', lcls='f-m s', lh=15,
          note='NewTime is a per-realm counter. The package value itself is always :1.')
    f.box(340, 166, 400, 56, 'Three states', ['empty  →  allocated: PkgID stamped, NewTime 0  →  finalized: counter taken'], '', lcls='f-m s',
          note='The allocator stamps a PkgID while building the object: the realm that declared its type, or else the active realm. Only finalization takes a counter value, and only a finalized object is real.')
    f.text(20, 254, 'An object is saved under the realm its PkgID names, even when another realm\'s code did the saving.', 'f-t s')
    return f


# ---------------------------------------------------------------- refcount and escape

@fig
def refcount_escape():
    f = Fig('refcount-escape', 248)
    panels = [
        ('Owned: RefCount 1', 'acc',
         ['one holder', 'owner id set', 'hash carried in the owner\'s bytes', 'one Merkle chain down the tree'],
         'The common case. A tree of singly referenced objects is one Merkle chain, saved with its root.', 1),
        ('Escaped: RefCount 2', '',
         ['two holders', 'owner cleared, saved on its own', 'hash written to the Merkle tree under its id', 'once escaped, always escaped'],
         'Parents carry its id alone. SetObject writes the hash to the Merkle tree; the saveNewEscaped pass is an empty stub.', 2),
        ('Deleted: RefCount 0', 'warn',
         ['no holder left', 'deleted, with each child whose count falls to 0', 'a child with another holder stays', 'cycles are not supported'],
         'processNewDeletedMarks decrements the children recursively, so a dropped subtree goes in one pass.', 0),
    ]
    for i, (t, cls, ls, n, refs) in enumerate(panels):
        x = 20 + i * 246
        f.open_hot(n)
        f.rect(x, 36, 234, 196, cls)
        f.text(x + 10, 56, t, 'f-t b')
        # little drawing
        cy = 96
        for k in range(refs):
            hx = x + 30 + k * 60
            f.rect(hx, cy - 22, 44, 22, 'mut', r=4)
            f.text(hx + 22, cy - 7, 'holder', 'f-t s', 'middle')
            f.path('M%g %g L%g %g' % (hx + 22, cy, x + 117, cy + 22), 'thin')
        f.rect(x + 92, cy + 22, 50, 24, 'mut' if refs else 'warn', r=4)
        f.text(x + 117, cy + 38, 'object', 'f-t s', 'middle')
        if refs == 0:
            f.path('M%g %g L%g %g M%g %g L%g %g' % (x + 92, cy + 22, x + 142, cy + 46, x + 142, cy + 22, x + 92, cy + 46), 'warn thin', arrow=False)
        ty = 160
        for l in ls:
            ty = f.lines(x + 10, ty, wrap(l, int(214 / SAFE_SMALL)), 'f-t s', lh=14)
        f.close()
    return f


# ---------------------------------------------------------------- finalization

@fig
def finalization():
    f = Fig('finalization', 420)
    f.text(20, 24, 'zrealm1.gno: root = InnerNode{Key: "somekey"}, inside a realm call', 'f-m b')
    f.box(20, 44, 220, 56, ':1 PackageValue', ['the package itself; always NewTime 1'], '', id='obj-1',
          note='Every realm\'s first object. It holds the package block and one block per file.')
    f.box(20, 130, 220, 66, ':2 Block (package)', ['IsEscaped, RefCount 2', 'slot 0 → :3'], '', id='obj-2',
          note='The package block. It is escaped because it has two holders, so it is saved on its own. Its ModTime moves from 0 to 6 in the trace.')
    f.arrow(130, 100, 130, 129)
    f.box(300, 130, 220, 66, ':3 HeapItem root', ['owner :2, RefCount 1', 'Value: was {}, now → :6'], '', id='obj-3',
          note='The heap item behind the package variable root. Assign2 writes into it and calls DidUpdate with the old child, none, and the new child, the struct.')
    f.arrow(240, 163, 299, 163)
    f.box(300, 230, 220, 66, ':6 InnerNode', ['owner :3, RefCount 1', 'Key: "somekey", Left {}, Right {}'], 'acc', id='obj-6',
          note='Created this transaction. DidUpdate marked it new-real; processNewCreatedMarks assigned :6 depth-first. The string inside it is not an object, so nothing else became real.')
    f.arrow(410, 196, 410, 229)
    f.box(540, 44, 200, 252, 'Realm: writes, in order', [
        'c[...:6]  (+269 bytes)',
        '  the new struct, owner :3',
        'u[...:3]  (+156 bytes)',
        '  Value: RefType InnerNode',
        '  + RefValue{:6, hash}',
        'u[...:2]  (+5 bytes)',
        '  hash of :3 changed,',
        '  ModTime 0 → 6',
        '',
        'then: 430 bytes in all',
        '  → storage deposit',
    ], 'mut', id='writes', lcls='f-m s', lh=16, nowrap=True,
        note='Exactly what the filetest\'s Realm: directive prints. Children are saved before parents, and only new or dirty objects are written.')
    steps = ['1 new-real marks: ids depth-first, children counted', '2 deleted marks: children decremented', '3 escaped marks: demote or confirm',
             '4 realm record saved if the counter moved', '5 owners marked dirty, up to an escaped object', '6 save new and dirty, children first',
             '7 remove the deleted', '8 byte deltas per realm, for the deposit']
    y = 322
    for i, s in enumerate(steps):
        col, row = i % 2, i // 2
        f.text(20 + col * 370, y + row * 16, s, 'f-t s')
    f.foot(404, 'FinalizeRealmTransaction runs when a call returns across a realm boundary, and twice when a package is deployed.')
    return f


# ---------------------------------------------------------------- persist copy

@fig
def persist_copy():
    f = Fig('persist-copy', 362)
    f.box(20, 30, 250, 120, 'Live object :3, HeapItemValue', [
        'Value: TypedValue{',
        '  T: *DeclaredType InnerNode',
        '  V: *StructValue :6 }',
        'ObjectInfo{ID :3, Owner :2, ...}',
    ], '', lcls='f-m s', lh=15, nowrap=True,
        note='What the interpreter holds in memory: a pointer to the child object and a pointer to the type.')
    f.arrow(270, 90, 309, 90)
    f.box(310, 30, 430, 140, 'Persist copy, made by copyValueWithRefs', [
        '"Value": {',
        '  "T": {"@type": "/gno.RefType",',
        '        "ID": "gno.land/r/test.InnerNode"},',
        '  "V": {"@type": "/gno.RefValue",',
        '        "ObjectID": "...:6",',
        '        "Hash": "9b6e58b7899427bb...328b4537d0"}}',
    ], 'acc', lcls='f-m s', lh=15, nowrap=True,
        note='From the zrealm1 trace. An owned child becomes a RefValue with its id and hash, an escaped child with its id alone, a package with its path; every declared type a RefType with its id, every syntax node a RefNode with its location.')
    f.arrow(525, 170, 525, 199)
    f.box(310, 200, 430, 80, 'amino bytes, then the key', [
        'hash = sha256(bytes)[:20]',
        'key  = oid:08ada09dee16d791fd406d629fe29bb0ed084a30:3',
        'value = hash || bytes',
    ], 'mut', lcls='f-m s', lh=15, nowrap=True,
        note='Amino is tm2\'s deterministic codec; every persisted type is registered under a name such as /gno.StructValue. The hash of the child is what the parent\'s RefValue carries, which is how the Merkle chain forms.')
    f.box(20, 170, 250, 110, 'Loading is lazy', [
        'loadObjectSafe decodes, allocates the size, resolves RefTypes. A RefValue stays in place until the slot is read: fillValueTV swaps in the object then.',
    ], '', lh=13,
        note='Reading one field of a large realm loads one path through the object graph, not the graph.')
    y = f.foot(308, 'The same three stand-ins cover the three kinds of reference: RefValue for objects, RefType for declared types, RefNode for syntax nodes.')
    f.foot(y + 2, 'A RefNode carries only a Location: package path, file, line and column span. The store looks the node up by that location.')
    return f


# ---------------------------------------------------------------- store layout

@fig
def store_layout():
    f = Fig('store-layout', 364)
    f.rect(20, 30, 720, 160, 'mut')
    f.text(30, 50, 'defaultStore, the only Store implementation', 'f-t b')
    f.box(30, 62, 345, 118, 'baseStore: a plain database adapter on baseKey', [
        'oid:<pkgid>:<newtime>   objects, hash || bytes',
        'tid:<typeid>            types',
        'oid:<pkgid>:1#realm     realm records',
        'pkgidx:                 the list of uploads',
        'commits with no hash: not in the application hash',
    ], '', lcls='f-m s', lh=15, nowrap=True,
        note='Objects live here. The adapter commits with no hash, so this store does not feed the application hash.')
    f.box(385, 62, 345, 118, 'iavlStore: the B+32 tree on mainKey', [
        'pkg:<path>              production files',
        'pkg:<path>#allbutprod   test files',
        'inert_pkg:<path>        inert packages',
        '<pkgid>:<newtime>       escaped object hashes',
        'its root feeds the application hash',
    ], 'acc', lcls='f-m s', lh=15, nowrap=True,
        note='Only the Merkle tree\'s contents feed the application hash: package files, escaped hashes, inert packages, and the other keepers\' accounts, balances and parameters.')
    f.box(20, 206, 232, 140, 'Per transaction', [
        'BeginTransaction forks the store with fresh object, type and realm caches and a write log around the syntax-node cache. Write seals and publishes the log, only when the transaction succeeded.',
    ], '', lh=13, cw=SAFE_SMALL,
        note='The keeper forks the store from the base app\'s begin hook; the end hook calls Write only when the transaction succeeded.')
    f.box(264, 206, 232, 140, 'Hooks', [
        'PackageGetter loads a package from disk on first import, off-chain only. NativeResolver attaches the Go body of a native function. Standard library objects and types sit in memory from node start: no I/O gas.',
    ], '', lh=13, cw=SAFE_SMALL,
        note='The two hooks make the store complete; the in-memory standard library keeps library reads free of I/O gas.')
    f.box(508, 206, 232, 140, 'Between messages', [
        'ClearObjectCache runs at the start of every message so nothing leaks between them. GarbageCollectObjectCache drops the cached objects the last garbage-collection pass did not reach.',
    ], '', lh=13, cw=SAFE_SMALL,
        note='The object cache is per message; the garbage-collection pass keeps it from growing within one.')
    return f


# ---------------------------------------------------------------- cur chain

@fig
def cur_chain():
    f = Fig('cur-chain', 292)
    f.text(20, 24, 'MsgCall app.F, where F calls lib.G(cross(cur), ...)', 'f-m b')
    boxes = [
        ('signer', ['g1abc... the transaction\'s signer'], 'mut', 'At the chain root there is no caller frame. MsgCall synthesizes pkg.F(.origin, args...), and .origin lowers to a crossing call whose prev is built from the signer: its address, an empty package path and no link further.'),
        ('F\'s cur', ['path: r/demo/app', 'prev → signer'], 'acc', 'installCrossingCur minted this .grealm value for F. cur.Previous() is the signer, unforgeably.'),
        ('G\'s cur', ['path: r/demo/lib', 'prev → app'], 'acc', 'Minted for G because F wrote cross(cur). G sees app as its previous realm, whatever helper made the call.'),
    ]
    x = 20
    for t, ls, cls, n in boxes:
        f.box(x, 44, 200, 68, t, ls, cls, note=n, lcls='f-m s', lh=15)
        x += 260
    f.arrow(220, 78, 279, 78, label='.origin', dy=-8)
    f.arrow(480, 78, 539, 78, label='cross(cur)', dy=-8)
    f.path('M540 100 C500 130 300 130 221 100', 'dash')
    f.text(380, 138, 'cur.Previous()', 'f-t s', 'middle')
    f.path('M280 100 C250 130 40 130 21 100', 'dash')
    f.text(150, 138, 'cur.Previous()', 'f-t s', 'middle')
    f.box(20, 160, 350, 86, 'H(cur) inside the same realm', ['no new link: H runs as F\'s cur, which must be F\'s own live value. Written against another realm\'s function, the preprocessor rejects it.'], '', lh=13, cw=SAFE_SMALL,
          note='Calling a crossing function with a bare cur is only legal within its own realm.')
    f.box(390, 160, 350, 86, 'chain/runtime/unsafe', ['PreviousRealm and CurrentRealm read the chain of the nearest crossing frame. In a plain function of realm B called by realm A, PreviousRealm names whoever called A. Hence the name.'], 'warn', lh=13, cw=SAFE_SMALL,
          note='The older API. Prefer cur.Previous(): it answers for the function that holds cur.')
    f.foot(270, 'A realm value cannot be saved, the signer\'s link at the chain root aside: store the address instead. main(cur realm) and init(cur realm) are called with .cur, already crossed.')
    return f


# ---------------------------------------------------------------- interrealm cases

CASES = [
    ('lib.F(cross(cur), x)', 'lib', 'lib', True, 'crossing call: a new cur whose prev is app',
     'The explicit switch. Identity and storage both move to lib for the duration of F, and finalization runs when F returns.'),
    ('lib.Reset()', 'app', 'lib', True, 'borrow rule 1: Reset is declared in lib',
     'A plain call to a function another realm declared runs with that realm\'s storage, so Reset can write lib\'s variables. The identity stays app.'),
    ('lib.Tree().Set(k, v)', 'app', 'lib', True, 'borrow rule 2: the tree is stamped lib, Set comes from p/nt/avl/v0',
     'A pure package method on a receiver another realm owns runs with that realm\'s storage. Any realm holding the pointer can write lib\'s tree this way.'),
    ('lib.F(cur, x)', '', '', False, 'rejected: cur belongs to app',
     'A crossing function of another realm called with a bare cur. The preprocessor refuses it; only cross(cur) may switch identity.'),
    ('lib.Data.N = 1', 'app', 'app', False, 'ownership: lib.Data is stamped with lib, and app is the storage realm',
     'Before every write the VM compares the stamp of the object being written with the storage realm, and panics when they differ.'),
    ('lib.T{}   new(lib.T)   make(lib.S, n)', 'app', 'app', False, 'construction: lib\'s types only through lib\'s constructors',
     'A composite literal, new or make of a type declared in another realm panics at the allocation site.'),
]


@fig
def interrealm_cases():
    f = Fig('interrealm-cases', 436)
    f.text(20, 24, 'Code running in r/demo/app, calling into r/demo/lib', 'f-t h')
    cols = [(20, 'call written in app'), (330, 'identity'), (420, 'storage'), (510, 'verdict')]
    for x, t in cols:
        f.text(x + 8, 52, t, 'f-t s')
    f.path('M20 58 L740 58', '', arrow=False)
    y = 66
    for code, ident, stor, ok, verdict, n in CASES:
        f.open_hot(n)
        f.rect(20, y, 720, 48, 'acc' if ok else 'warn', r=4)
        f.text(28, y + 20, code, 'f-m b')
        f.text(338, y + 20, ident or '·', 'f-m')
        f.text(428, y + 20, stor or '·', 'f-m')
        f.text(518, y + 20, ('✓  ' if ok else '✗  ') + verdict.split(':')[0], 'f-t b')
        f.text(28, y + 38, verdict.split(': ', 1)[1] if ': ' in verdict else '', 'f-t s')
        f.close()
        y += 54
    f.foot(y + 14, 'Identity is who acts, cur; storage is whose objects may be written. The borrow rules set storage for every call that is not an explicit cross.')
    return f


# ---------------------------------------------------------------- gas

@fig
def gas_sources():
    f = Fig('gas-sources', 514)
    srcs = [
        ('CPU', ['a constant per opcode: OpExec 130, OpEval 82, int add 81, OpDefine 114', 'slopes: per unit of name depth, per name defined, per parameter, per element, per bit', 'natives: a table by package and name', 'output formatting: 4 gas per byte'],
         'One gas is one nanosecond on the reference Xeon. cmd/calibrate produced the constants.'),
        ('Allocation', ['bytes per value, 500,000,000 per message', 'gas from a table by the base-2 log of the size', 'over budget: recount every reachable object, charged per visit, and continue if it fits'],
         'An accounting GC: Go\'s collector frees the memory, the VM only recounts.'),
        ('Store', ['encoding: 3 gas per byte, each way', 'a cache miss: a flat fee plus a per-byte fee; a hit is free', 'a write: a flat fee plus a per-byte fee; a key written twice is billed once', 'the Merkle tree scales the flat fee by depth'],
         'Flat and per-byte fees are tm2 code defaults; depths are VM parameters.'),
        ('Code size', ['MsgAddPackage and MsgRun', 'preprocess_gas_per_byte for each source byte', 'charged before the type-check'],
         'preprocess_gas_per_byte is a VM parameter; its code default is 1,250.'),
    ]
    bw, bh = 172, 226
    for i, (t, ls, n) in enumerate(srcs):
        x = 20 + i * (bw + 10)
        f.box(x, 30, bw, bh, t, ls, '', note=n, lh=13, cw=SAFE_SMALL, para=6)
        f.arrow(x + bw / 2, 30 + bh, x + bw / 2, 30 + bh + 27)
    f.rect(20, 284, 720, 46, 'mut')
    f.text(380, 304, 'one gas meter per transaction', 'f-t b', 'middle')
    f.text(380, 321, 'opened by the ante handler, which first charges the transaction\'s bytes and signatures', 'f-t s', 'middle')

    # the storage deposit is coins, beside the meter
    f.text(20, 358, 'Paid in coins, beside the meter, and returned when the bytes are freed:', 'f-t s')
    f.box(20, 368, 360, 90, 'Storage deposit',
          ['bytes a realm grew × storage_price, a VM parameter',
           'refunded in proportion: deposit × freed bytes / stored bytes'],
          'acc', note='Settled after each message by processStorageDeposit, from the byte changes finalization reports.',
          lh=13, cw=SAFE_SMALL, para=6)
    f.arrow(380, 413, 419, 413)
    f.box(420, 368, 320, 90, 'The realm\'s storage deposit address',
          ['coins locked from the caller; a refund goes back to the caller, or to storage_fee_collector when ugnot is restricted'],
          '', note='An address computed from the realm\'s path. The total one message locks may not exceed the caller\'s -max-deposit, or default_deposit when none is set.',
          lh=13, cw=SAFE_SMALL)
    f.foot(482, 'Queries run under their own meter and allocator, 3 billion gas and 1.5 GB, both code constants, and are never committed.')
    return f


# ---------------------------------------------------------------- natives

@fig
def native_binding():
    f = Fig('native-binding', 298)
    steps = [
        ('time.gno', ['func now() (', '  sec int64,', '  nsec int32,', '  mono int64)', '// injected', 'no body'], 'acc', 'A standard library function declared without a body and marked injected has a Go implementation in a sibling .go file.'),
        ('misc/genstd', ['scans the stdlibs', 'writes generated.go'], '', 'A generator run by go generate, never on chain.'),
        ('generated.go', ['72 bindings at this commit', 'package path and name → a Go closure', 'reflection converts the values'], '', 'The table the store consults. Gno values are converted to Go with reflection and back.'),
        ('NativeResolver', ['a store hook', 'attaches the Go body at preprocessing, again on the first call after a load'], '', 'The body is never saved; the first call after a load looks it up again by package path and name.'),
        ('ExecContext', ['chain id and domain', 'height, timestamp', 'origin caller, coins sent', 'banker, parameters', 'event logger, session account'], 'mut', 'A native that needs chain state takes the Machine and reads its Context. That is how time.Now returns the block time.'),
    ]
    bw, bh = 136, 152
    for i, (t, ls, cls, n) in enumerate(steps):
        x = 20 + i * (bw + 14)
        if i == 0:
            f.box(x, 40, bw, bh, t, ls, cls, note=n, lcls='f-m s', lh=15, nowrap=True)
        else:
            f.box(x, 40, bw, bh, t, ls, cls, note=n, lh=13, cw=SAFE_SMALL, para=4)
        if i < 4:
            f.arrow(x + bw, 116, x + bw + 13, 116)
    y = f.foot(222, 'Under test, gnovm/tests/stdlibs overlays a testing package: SetRealm, SetOriginCaller, SetOriginSend, SetHeight, SkipHeights and IssueCoins rewrite the mock context.')
    f.foot(y + 2, 'Native gas: each binding\'s row charges a base plus up to two slopes on argument lengths, and may charge on result lengths after the call; a native with no row panics.')
    return f


# ---------------------------------------------------------------- keeper flows

@fig
def keeper_flows():
    f = Fig('keeper-flows', 432)
    lanes = [
        ('AddPackage', ['validate the path; refuse an existing public package', 'gas per source byte; type-check production files, strict', 'manifest rules; names and CLA realms; coins to the package address', 'preprocess allocator: folding is metered', 'run with saving on: variables, then each init', 'settle the storage deposit'],
         'MsgAddPackage. The package runs once and is saved; every later call skips to execution. Under the inert policy the package is parked instead, and MsgEnablePackage runs these steps after approval.'),
        ('Call', ['refuse a non-crossing function', 'bind the package as pkg in a throwaway main', 'parse pkg.F(cross, args...) written as text', 'replace the first argument with .origin', 'evaluate; results joined as strings', 'refuse coins never read; settle the deposit'],
         'MsgCall. The arguments arrive as strings and are converted to the parameter types.'),
        ('Run', ['path forced to <domain>/e/<caller>/run', 'gas per byte; relaxed type-check, manifest optional', 'the package is marked private', 'run without saving', 'main called in a second machine', 'output returned; the script is never stored'],
         'MsgRun. A throwaway main package; nothing persists beyond what its calls write into other realms.'),
        ('Query', ['fork a throwaway store over a committed snapshot', 'load the package', 'parse the expression; .cur prepended for a crossing callee', 'evaluate under 3 billion gas and 1.5 GB', 'qrender is the same with Render(path)', 'never committed'],
         'vm/qeval and vm/qrender. Read-only, on a forked store that is dropped afterwards.'),
    ]
    lw, ch = 172, 48
    for i, (t, steps, n) in enumerate(lanes):
        x = 20 + i * (lw + 10)
        f.open_hot(n)
        f.rect(x, 30, lw, 30, 'acc', r=4)
        f.text(x + lw / 2, 50, t, 'f-t b', 'middle')
        for k, s in enumerate(steps):
            y = 66 + k * (ch + 6)
            f.rect(x, y, lw, ch, '', r=4)
            pieces = wrap(s, int((lw - 16) / SAFE_SMALL))
            assert len(pieces) <= 3, s
            f.lines(x + 8, y + 16, pieces, 'f-t s', lh=13)
            if k < len(steps) - 1:
                f.arrow(x + lw / 2, y + ch, x + lw / 2, y + ch + 5, 'thin')
        f.close()
    f.foot(406, 'Restart: Initialize re-preprocesses every stored package, type-checks the standard library into the permanent cache and loads its bytes into memory.')
    return f


# ---------------------------------------------------------------- tooling map

@fig
def tooling_map():
    f = Fig('tooling-map', 296)
    stages = ['MemPackage', 'type-check', 'preprocess', 'execute', 'finalize']
    bw, gap = 128, 20
    for i, s in enumerate(stages):
        x = 20 + i * (bw + gap)
        f.box(x, 30, bw, 34, None, [s], 'acc', lcls='f-t b', pad=10)
        if i < 4:
            f.arrow(x + bw, 47, x + bw + gap - 1, 47)

    def span(y, a, b, label, note, cls=''):
        x1 = 20 + a * (bw + gap)
        x2 = 20 + b * (bw + gap) + bw
        f.open_hot(note)
        f.rect(x1, y, x2 - x1, 26, cls, r=4)
        f.text(x1 + 8, y + 17, label, 'f-t s')
        f.close()

    span(84, 1, 2, 'gno lint: type-check and preprocess, no run', 'Per the lint and transpile record. Every file is checked, tests included, and errors come out as file:line messages without executing anything.')
    span(118, 0, 4, 'gno test, gno run: every stage, over an in-memory store with a mock context', 'gno test runs Test functions, Example functions with an Output comment, and filetests. A filetest sets PKGPATH, MAXALLOC and SEND in comments and asserts on Output, Error, Realm, Events, Preprocessed, Stacktrace, Gas, Storage, Types and TypeCheckError blocks.')
    span(152, 0, 0, 'gno fix: rewrites', 'gno fix applies named rewrites such as stdsplit; gno tool transpile is what uses the transpiler.', 'mut')
    span(152, 3, 4, 'debugger: -debug on gno run, gno test', 'Stops at breakpoints and steps by source line or by single opcode.', 'mut')
    span(186, 3, 4, 'benchops: opcode, store, native timing', 'The numbers behind the gas constants come from cmd/calibrate and cmd/benchstore.', 'mut')
    span(220, 0, 4, 'gnodev: a whole node in memory, with the real keeper', 'gnodev reloads a changed package and replays past transactions, so state survives the edit. Unlike gno test and gno run, it runs the real keeper.', 'mut')
    f.foot(274, 'Build tags debug and debugAssert turn on tracing and invariant panics. gno doc and vm/qdoc read the sources without running them.')
    return f


# ---------------------------------------------------------------- determinism

@fig
def determinism():
    f = Fig('determinism', 330)
    f.text(20, 24, 'Source of drift between validators', 'f-t b')
    f.text(290, 24, 'What the VM does', 'f-t b')
    rows = [
        ('goroutines, channels, select', 'the parser refuses go, select and channels'),
        ('the CPU\'s floating point', 'software floats, so every node computes the same bits'),
        ('Go map iteration order', 'a map ranges in insertion order'),
        ('clock, randomness, network', 'time.Now is the block time; math/rand starts from a fixed seed'),
        ('the capacity of []byte(s)', 'equals the length, for []rune(s) too; Go leaves it open'),
        ('gas', 'constants by operand kind and size, never timed'),
        ('the type-check verdict', 'Go 1.18 rules everywhere; a failure records one fixed error'),
        ('panic messages', 'the hashed error is the panic value; traces go to the log'),
    ]
    y = 40
    for a, b in rows:
        f.rect(20, y, 250, 26, 'warn', r=4)
        f.text(30, y + 17, a, 'f-t')
        f.arrow(270, y + 13, 289, y + 13)
        f.rect(290, y, 450, 26, 'acc', r=4)
        f.text(300, y + 17, b, 'f-t')
        y += 32
    f.foot(y + 14, 'Determinism is enforced in the language, not trusted to the program: the same transaction yields the same state on every node.')
    return f


# ================================================================ overview figures

@fig
def package_kinds():
    f = Fig('package-kinds', 330)
    kinds = [
        ('r/  realm', ['gno.land/r/<ns>/<name>', 'persistent globals, an address, coins', 'may declare crossing functions', 'renders a page with Render(path)', 'immutable once deployed; private = true may be redeployed'],
         'acc', 'A realm is a package whose package-level variables are saved when a call returns. It has a bech32 address, so it can hold and send coins.'),
        ('p/  pure package', ['gno.land/p/<ns>/<name>', 'no persistent state', 'no crossing functions', 'may not import a realm', 'the code two realms can trust between them'],
         '', 'Library code. Its methods run with the storage of whoever owns the receiver, which is how avl.Tree.Set writes a realm\'s tree.'),
        ('e/  ephemeral', ['gno.land/e/<address>/run', 'uploaded by gnokey maketx run', 'main runs once, nothing is stored', 'several calls in one transaction'],
         'mut', 'A throwaway main package. Type-checked in relaxed mode, marked private, run in a second machine, never saved.'),
    ]
    for i, (t, ls, cls, n) in enumerate(kinds):
        x = 20 + i * 246
        f.box(x, 30, 234, 140, t, ls, cls, note=n, lh=15)
    f.rect(20, 190, 720, 116, 'mut')
    f.text(30, 210, 'Who may deploy under a namespace, per r/sys/names', 'f-t b')
    f.lines(30, 230, [
        'gno.land/r/g1abc.../x      your own address: always, no registration',
        'gno.land/r/alice/x         a registered name: when r/sys/users maps alice to you as your current name',
        'the keeper asks r/sys/names on every upload; a rename gives up the old namespace for new deploys',
        'on pearl and sapphire the check is on from block one; before Enable() every check passes',
    ], 'f-t s', lh=17)
    return f


@fig
def dev_workflow():
    f = Fig('dev-workflow', 250)
    steps = [
        ('write', ['myapp.gno', 'gnomod.toml'], 'A package directory with a manifest naming its path and Gno version.'),
        ('gno test', ['_test.gno functions', '_filetest.gno golden files'], 'No node needed. Filetests assert on Output, Error, Realm and Events blocks.'),
        ('gnodev', ['in-memory node', 'gnoweb on :8888', 'hot reload'], 'Every account in the local keybase starts with 10T ugnot. State survives reloads by replaying transactions.'),
        ('gnokey addpkg', ['type-check, run once, save', 'storage deposit locked'], 'The keeper refuses an existing public path. -simulate only reports gas and bytes first.'),
        ('gnokey call', ['pkg.F(cross, args)', 'cur.Previous() = you'], 'Only a crossing function can be called. The signer becomes the first link of the cur chain.'),
        ('gnoweb', ['Render(path) → markdown', 'vm/qrender'], 'The page is the realm\'s own Render output, served read-only under a query gas limit.'),
    ]
    bw = 112
    for i, (t, ls, n) in enumerate(steps):
        x = 20 + i * (bw + 10)
        f.box(x, 40, bw, 120, t, ls, 'acc' if i in (3, 4) else '', note=n, lh=13)
        if i < 5:
            f.arrow(x + bw, 100, x + bw + 9, 100)
    f.path('M%g 160 L%g 190 L%g 190 L%g 160' % (20 + 5 * 122 + bw / 2, 20 + 5 * 122 + bw / 2, 20 + bw / 2, 20 + bw / 2), 'dash')
    f.text(380, 208, 'a deployed path can never be reused: the next version is a new path', 'f-t s', 'middle')
    f.text(20, 236, 'Steps 1 to 3 need no chain. Steps 4 to 6 target any network by -remote and -chainid.', 'f-t s')
    return f


@fig
def staging_cycle():
    f = Fig('staging-cycle', 250)
    nodes = [
        ('staging runs', 'The chain behaves like any network while master does not change.'),
        ('master changes', 'A merge on the gno monorepo triggers a rebuild.'),
        ('tx-archive saves every transaction', 'Every transaction so far is exported, so the history survives the rebuild.'),
        ('a new genesis from master, examples/ redeployed', 'All packages under examples/ are deployed first; permissionless deploys of the same path are superseded.'),
        ('the archive replays into it', 'A transaction that no longer passes under the new master fails to replay, and its data is lost. Heights and timestamps restart.'),
    ]
    bw = 136
    for i, (t, n) in enumerate(nodes):
        x = 20 + i * (bw + 10)
        f.box(x, 50, bw, 70, None, [t], 'acc' if i == 0 else '', note=n, lcls='f-t b', lh=15)
        if i < 4:
            f.arrow(x + bw, 85, x + bw + 9, 85)
    f.path('M%g 120 L%g 160 L%g 160 L%g 120' % (20 + 4 * 146 + bw / 2, 20 + 4 * 146 + bw / 2, 20 + bw / 2, 20 + bw / 2))
    f.text(380, 180, 'back to running, with the old transactions replayed and the new code live', 'f-t s', 'middle')
    f.text(20, 226, 'From docs/resources/gnoland-networks.md. Pearl, sapphire and topaz are the opposite: fresh genesis, no replay.', 'f-t s')
    return f


@fig
def govdao():
    f = Fig('govdao', 420)
    f.text(20, 24, 'Tiers, from r/gov/dao/v3/memberstore', 'f-t h')
    tiers = [
        ('T1', ['base power 3', '3 invitation points', 'at least 70 members', 'power uncapped'], 'acc',
         'Core members. Added and promoted by proposal only; a T1 proposal is voted by T1.'),
        ('T2', ['base power 2', '2 invitation points', 'a quarter to twice the T1 count', 'total power ≤ 2/3 of T1\'s'], '',
         'When the cap binds, each T2 member\'s power shrinks to fit: the tier as a whole never outweighs two thirds of T1.'),
        ('T3', ['base power 1', '1 invitation point', 'no size rule', 'total power ≤ 1/3 of T1\'s'], 'mut',
         'Added directly by a T1 or T2 member, who spends one invitation point. Not by proposal.'),
    ]
    for i, (t, ls, cls, n) in enumerate(tiers):
        x = 20 + i * 246
        f.box(x, 40, 234, 100, t, ls, cls, note=n, lh=15)
    f.text(20, 176, 'A proposal, from r/gov/dao/v3/impl', 'f-t h')
    steps = [
        ('create', ['title, description', 'executor callback'], 'Seven request types: change the law, upgrade the implementation, add, withdraw or promote a member, pay from the treasury, update the GRC-20 list.'),
        ('vote', ['YES, NO, ABSTAIN', 'against total power'], 'Percentages are computed against all eligible voting power, so a member who does not vote weighs like a NO. A member proposal restricts the vote to the target tier.'),
        ('66.66% YES', ['accepted', 'executor runs'], 'The Law\'s Supermajority value, changeable by a change-law proposal.'),
        ('66.66% NO', ['denied'], 'Symmetric: the same threshold closes the proposal.'),
    ]
    bw = 170
    for i, (t, ls, n) in enumerate(steps):
        x = 20 + i * (bw + 13)
        cls = 'acc' if i == 2 else ('warn' if i == 3 else '')
        f.box(x, 192, bw, 76, t, ls, cls, note=n, lh=15)
        if i < 2:
            f.arrow(x + bw, 230, x + bw + 12, 230)
    f.path('M%g 268 L%g 290 L%g 290 L%g 268' % (20 + 1 * 183 + bw / 2, 20 + 1 * 183 + bw / 2, 20 + 3 * 183 + bw / 2, 20 + 3 * 183 + bw / 2), 'warn')
    f.lines(20, 322, [
        'r/gov/dao is a proxy: UpdateImpl swaps the implementation by proposal, within an allowed list.',
        'What it controls: the validator set (r/sys/validators), chain parameters (r/sys/params), the treasury,',
        'name registration controllers and prices (r/sys/users, r/sys/namereg), and the namespace verifier\'s pause.',
        'A fresh chain starts with one T1 member, who proposes the next ones.',
    ], 'f-t s', lh=17)
    return f


@fig
def gnot_flows():
    f = Fig('gnot-flows', 310)
    f.box(20, 30, 190, 232, 'Caller', ['g1abc..., pays in ugnot', '', 'gas fee: spent', 'storage deposit: a bond,', 'returned when bytes are freed'], 'acc',
          note='The transaction signer. The whole -gas-fee leaves the account whatever the transaction used; the deposit only moves when a realm\'s byte count changes.', lh=15)
    f.box(300, 30, 200, 52, 'Fee collector', ['the auth module\'s address'], 'mut',
          note='DeductFees moves the fee here. Distribution is a placeholder in r/sys/txfees, so fees accumulate.')
    f.box(300, 110, 200, 90, 'Realm deposit address', ['one per realm, derived from its path; locks delta × storage_price, releases the same on freed bytes'], '',
          note='One deposit address per realm, DeriveStorageDepositAddr. Bytes added lock ugnot in; bytes freed release it to whoever made the call.', lh=13)
    f.box(300, 216, 200, 46, 'Storage fee collector', ['for restricted denominations'], 'mut',
          note='Where a refund goes instead of the sender when the chain restricts ugnot transfers, as betanet does.')
    f.arrow(210, 56, 299, 56)
    f.text(255, 48, 'gas fee, spent', 'f-t s', 'middle')
    f.arrow(210, 140, 299, 140)
    f.text(294, 132, 'bytes added: deposit', 'f-t s', 'end')
    f.arrow(299, 170, 210, 170, 'acc')
    f.text(294, 186, 'bytes freed: refund', 'f-t s', 'end')
    f.arrow(400, 200, 400, 215, 'dash')
    f.text(300, 282, 'when ugnot is restricted the refund lands here, not with the caller', 'f-t s')
    f.box(540, 30, 200, 232, 'Numbers', [
        '1 GNOT = 1,000,000 ugnot',
        'storage: 100 ugnot per byte',
        '1 GNOT buys 10 KB',
        '1e9 GNOT = 10 TB',
        'deposit default: 600 GNOT',
        'gas: 1 ugnot per 1000 gas',
        'cap 1.333e9 GNOT, no inflation',
    ], '', lcls='f-m s', lh=17, nowrap=True,
        note='Code defaults at this commit and the Constitution\'s ceiling. The live values were the same on staging and betanet on 2026-09-05.')
    f.text(20, 300, 'The deposit is the one cost that comes back: it is a bond on bytes, not a payment for them.', 'f-t s')
    return f


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    for name, fn in FIGS.items():
        svg = fn().render()
        with open(os.path.join(here, name + '.svg'), 'w') as out:
            out.write(svg)
        print('%-22s %6d bytes' % (name + '.svg', len(svg)))


if __name__ == '__main__':
    main()
