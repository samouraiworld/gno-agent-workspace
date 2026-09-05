#!/usr/bin/env python3
"""Draws the figures beside gnovm-architecture.md.

Run it from anywhere: python3 docs/figures/gen.py. It rewrites every *.svg beside it. Each
figure states what the document states, at gnolang/gno a7e4c34b0; the numbers
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
            tcls='f-t b', lcls='f-t s', lh=14, pad=10, nowrap=False):
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
        cw = MONO if 'f-m' in lcls else (SMALL if ' s' in lcls else SANS)
        maxc = max(8, int((w - 2 * pad) / cw))
        for l in lines:
            for piece in ([l] if nowrap else wrap(l, maxc)):
                self.text(x + pad, ty, piece, lcls)
                ty += lh
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
    f = Fig('vm-stack', 470)
    x, w, bh, gap = 40, 400, 42, 22
    layers = [
        ('Wallet', 'gnokey maketx call: a signed transaction',
         'The user signs a message: which package, which function, which arguments, how much gas. Nothing here knows about the VM.'),
        ('Tendermint2', 'orders transactions into blocks',
         'Consensus. Every validator receives the same transactions in the same order and hands them to the application one by one.'),
        ('BaseApp, tm2/pkg/sdk', 'ante handler: signature, fee. runMsgs: route by route string',
         'The application layer. The ante handler verifies the signature and takes the fee, then each message goes to the handler registered for its route.'),
        ('vm keeper, gno.land/pkg/sdk/vm', 'MsgAddPackage, MsgCall, MsgRun; queries vm/qeval, vm/qrender',
         'The keeper owns one long-lived gno.Store and creates one gno.Machine per message. It is the only chain code that calls the interpreter.'),
        ('gno.Machine, gnovm/pkg/gnolang', 'parse, preprocess, execute, finalize',
         'The interpreter state: stacks, the active package and realm, the allocator, the gas meter, and a Context the standard library reads.'),
        ('gno.Store', 'per-transaction caches for objects, types, syntax trees',
         'The layer between the interpreter and the database. Forked at the start of each transaction, committed only if the transaction succeeded.'),
        ('tm2 stores, then the database', 'baseStore for objects, iavlStore for the Merkle tree',
         'Two key-value stores. Only the IAVL contents feed the block\'s application hash.'),
    ]
    labels = ['RPC', 'ABCI', 'route "vm"', 'one Machine per message', '', '', '']
    y = 30
    for i, (t, l, n) in enumerate(layers):
        f.box(x, y, w, bh, t, [l], 'acc' if i == 4 else '', note=n)
        if i < len(layers) - 1:
            f.arrow(x + w / 2, y + bh, x + w / 2, y + bh + gap - 1)
            if labels[i]:
                f.text(x + w / 2 + 8, y + bh + 15, labels[i], 'f-t s')
        y += bh + gap
    # off-chain
    ox, oy, ow, oh = 480, 240, 260, 130
    f.box(ox, oy, ow, oh, 'Off-chain: gno test, gno run, filetests',
          ['The same interpreter with no chain. A Machine over an in-memory store seeded from examples/ and stdlibs/, with a mock context: fixed height, timestamp and caller.'],
          'mut', note='gno test and gno run build a Machine the same way the keeper does, over an in-memory store.')
    my = 30 + 4 * (bh + gap) + bh / 2
    f.path('M%g %gL%g %g' % (ox, oy + oh / 2, x + w + 2, my), 'dash')
    return f


# ---------------------------------------------------------------- source map

@fig
def source_map():
    f = Fig('source-map', 450, 'Direct imports between the documented directories, from go list')
    bw, bh = 168, 36
    pos = {
        'gnovm/cmd/gno': (30, 50), 'gno.land/pkg/gnoland': (560, 50),
        'gno.land/pkg/sdk/vm': (560, 120),
        'gnovm/pkg/test': (30, 190), 'gnovm/pkg/transpiler': (230, 190), 'gnovm/pkg/doc': (430, 190),
        'gnovm/stdlibs': (30, 260), 'gnovm/pkg/packages': (330, 260),
        'gnovm/pkg/gnolang': (230, 330),
        'gnovm/pkg/gnomod': (30, 400), 'gnovm/pkg/benchops': (430, 400),
    }
    edges = {
        'gnovm/cmd/gno': ['gno.land/pkg/sdk/vm', 'gnovm/pkg/doc', 'gnovm/pkg/gnolang', 'gnovm/pkg/gnomod', 'gnovm/pkg/packages', 'gnovm/pkg/test', 'gnovm/pkg/transpiler'],
        'gnovm/pkg/gnolang': ['gnovm/pkg/benchops', 'gnovm/pkg/gnomod'],
        'gnovm/pkg/test': ['gnovm/pkg/gnolang', 'gnovm/pkg/packages', 'gnovm/stdlibs'],
        'gnovm/pkg/transpiler': ['gnovm/pkg/gnolang', 'gnovm/stdlibs'],
        'gnovm/pkg/doc': ['gnovm/pkg/gnolang', 'gnovm/pkg/gnomod', 'gnovm/pkg/packages'],
        'gnovm/stdlibs': ['gnovm/pkg/gnolang'],
        'gnovm/pkg/packages': ['gnovm/pkg/gnolang', 'gnovm/pkg/gnomod'],
        'gno.land/pkg/sdk/vm': ['gnovm/pkg/doc', 'gnovm/pkg/gnolang', 'gnovm/pkg/gnomod', 'gnovm/stdlibs'],
        'gno.land/pkg/gnoland': ['gno.land/pkg/sdk/vm', 'gnovm/pkg/gnolang', 'gnovm/pkg/packages'],
    }
    notes = {
        'gnovm/pkg/gnolang': 'The VM itself. It imports only the manifest reader and the optional benchmarking hooks, so nothing above it is needed to run Gno.',
        'gnovm/stdlibs': 'The standard library sources and their Go native bodies. It needs the VM for the value types the natives marshal.',
        'gnovm/pkg/test': 'The test harness: store construction, filetest directives, package loading.',
        'gnovm/cmd/gno': 'The command line tool. It imports everything, including the keeper, which it uses for the query commands.',
        'gno.land/pkg/sdk/vm': 'The keeper. The only chain code that constructs a Machine.',
        'gno.land/pkg/gnoland': 'The node application: mounts the stores and registers the vm route.',
        'gnovm/pkg/transpiler': 'Gno to Go source translation, behind gno lint and gno fix.',
        'gnovm/pkg/doc': 'The gno doc engine, also served by vm/qdoc.',
        'gnovm/pkg/packages': 'Package discovery on disk.',
        'gnovm/pkg/gnomod': 'The gnomod.toml manifest. A leaf.',
        'gnovm/pkg/benchops': 'Per-opcode timing when compiled in. A leaf.',
    }
    for src, targets in edges.items():
        sx, sy = pos[src]
        for t in targets:
            tx, ty = pos[t]
            cls = 'acc' if t == 'gnovm/pkg/gnolang' else 'thin'
            f.path('M%g %gL%g %g' % (sx + bw / 2, sy + bh, tx + bw / 2, ty), cls)
    for name, (x, y) in pos.items():
        cls = 'acc' if name == 'gnovm/pkg/gnolang' else ''
        f.box(x, y, bw, bh, None, [name], cls, note=notes[name], lcls='f-m', pad=8)
    f.text(560, 300, 'accent arrows: an import of the VM', 'f-t s')
    f.text(560, 316, 'grey arrows: every other direct import', 'f-t s')
    return f


# ---------------------------------------------------------------- package pipeline

@fig
def package_pipeline():
    f = Fig('package-pipeline', 330)
    stages = [
        ('1 · MemPackage', ['name, path, files', 'the path letter picks the kind: r realm, p pure, e ephemeral', 'gnomod.toml: module, gno 0.9'],
         'A MemPackage is a name, a path, a list of files and a type tag. No timestamps, owners or subdirectories, so the same bytes hash the same everywhere.'),
        ('2 · Type-check', ['go/types, the checker gopls uses', '.gnobuiltins.gno declares realm, address, cross, revive', 'modes: strict, genesis, relaxed'],
         'Gno\'s own parser is lax, so Go\'s type checker runs first, with a generated shim declaring the names Go does not know.'),
        ('3 · Parse', ['go/parser, then Go2Gno builds the Gno nodes', 'additions: ConstExpr, RefNode, StaticBlock', 'a node\'s span is its identity'],
         'Go2Gno walks the go/ast tree and builds Gno nodes. Nodes that open a scope carry a StaticBlock recording the names declared there.'),
        ('4 · Preprocess', ['PredefineFileSet, then one walk per file', 'names become slots, constants fold, crossing rules are checked', 'block nodes saved by location'],
         'The compile step. Every name becomes a ValuePath, every constant expression a ConstExpr. A file is preprocessed exactly once.'),
        ('5 · Run once', ['package variables in Kahn order', 'save, run init.1, init.2, ..., save again', 'types stored under their id'],
         'runMemPackage: variable declarations in dependency order, then every init function in file order with the deployer as the previous realm.'),
    ]
    bw, bh, gap = 136, 150, 14
    for i, (t, ls, n) in enumerate(stages):
        x = 20 + i * (bw + gap)
        f.box(x, 40, bw, bh, t, ls, 'acc' if i == 3 else '', note=n, lh=13)
        if i < 4:
            f.arrow(x + bw, 115, x + bw + gap - 1, 115)
    f.box(470, 230, 270, 70, 'Saved package',
          ['package value and blocks, objects, types under tid:. Block nodes are not stored: on restart every package is re-preprocessed to rebuild them.'],
          'mut', note='Only values and types are persisted. The syntax tree is rebuilt from source at node start.', lh=13)
    f.arrow(620, 190, 620, 229)
    f.box(20, 230, 400, 70, 'Every later call',
          ['MsgCall on a stored package loads the saved values lazily and skips straight to execution. Steps 2 to 5 ran once, at upload.'],
          '', note='A call after upload does no type-check, no parse and no preprocess: it loads the package value and runs.', lh=13)
    f.arrow(469, 265, 421, 265, 'dash')
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
    f.text(20, 236, 'Dependencies recorded by codaInitOrderDeps, method bodies included', 'f-t s')
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
    f = Fig('type-ids', 424, 'What each Type implementation puts into its TypeID')
    cards = [
        ('int', 'PrimitiveType', 'One of the fixed kinds. The untyped constant kinds, bigint and bigdec, are primitive types too.',
         'Also DataByteType, an internal view into a byte array.'),
        ('[]int   *T   map[K]V', 'SliceType, PointerType, MapType', 'The id is built from the element ids. Channels have a type and no operations.',
         'ArrayType and ChanType complete the set.'),
        ('struct{ Key Key; Left Node }', 'StructType', 'The package path enters the id only for unexported field names: struct{ N int } from two packages is one type.',
         'Carries the declaring package path for its own use.'),
        ('func(cur realm, n int) string', 'FuncType', 'Parameter names are not part of the id. IsCrossing reads whether the first parameter is realm.',
         ''),
        ('interface{ Render(...) }', 'InterfaceType', 'Methods sorted before hashing. VerifyImplementedBy is the runtime check behind a type assertion.',
         ''),
        ('type InnerNode struct{ ... }', 'DeclaredType', 'id: gno.land/r/test.InnerNode. Declared inside a function: pkgpath[location].Name.',
         'Name, package path, base type, method list.'),
        ('address   realm   error', 'Declared in the universe block', 'address is declared on string. realm is an interface; its hidden .grealm holds an address, a path and the previous realm.',
         ''),
        ('RefType{ ID }', 'Placeholder', 'Stands in for a persisted type until the store loads it. Loading resolves it by id.',
         ''),
        ('8 · 8 · 128 · 128', 'Limits, checked when a type is built', 'Embedding depth 8, type nesting depth 8, 128 methods through embedding, 128 struct fields.',
         'They bound adversarial type shapes.'),
    ]
    cw, ch = 236, 108
    for i, (code, impl, rule, extra) in enumerate(cards):
        col, row = i % 3, i // 3
        x, y = 20 + col * (cw + 6), 40 + row * (ch + 10)
        f.open_hot(rule + (' ' + extra if extra else ''))
        f.rect(x, y, cw, ch, 'acc' if i == 5 else '')
        f.text(x + 10, y + 18, code, 'f-m b')
        f.text(x + 10, y + 34, impl, 'f-t s')
        ty = y + 50
        for piece in wrap(rule, 38)[:4]:
            f.text(x + 10, ty, piece, 'f-t s')
            ty += 14
        f.close()
    f.text(20, 406, 'Equality is TypeID equality, never pointer equality: the same type is deserialized in many transactions.', 'f-t s')
    return f


# ---------------------------------------------------------------- typed value

TV_CASES = [
    ('x := 42', 'int', 'nil', '42', 'An int lives in the eight bytes: SetInt writes an int64 straight into N. No allocation.'),
    ('ok := true', 'bool', 'nil', '01 at byte 0', 'SetBool writes the bool into the first byte of N.'),
    ('f := 2.5', 'float64', 'nil', '0x4004000000000000', 'SetFloat64 stores the IEEE-754 bits in N. Arithmetic on them runs in software so every node computes the same bits.'),
    ('s := "hi"', 'string', 'StringValue "hi"', '0', 'A string is a Go string in V. Not an Object: it has no identity and is saved inside its holder.'),
    ('p := &node', '*InnerNode', 'PointerValue{TV, Base, Index}', '0', 'A pointer is a slot address: the base object, an array, struct, block or heap item, plus an index into it.'),
    ('xs := []int{1, 2}', '[]int', 'SliceValue{Base, Offset 0, Length 2, Maxcap 2}', '0', 'A slice is a window onto an ArrayValue, which is the Object with identity.'),
    ('v := InnerNode{...}', 'InnerNode', '*StructValue{ObjectInfo, Fields}', '0', 'A struct is an Object: ObjectInfo carries its id, owner, reference count and dirty marks. Copying it copies the fields into a new object.'),
    ('var e any', 'nil', 'nil', '0', 'A nil interface value is the undefined typed value: no type, no value. It serializes as {} in a realm trace.'),
    ('lib.Data (other realm)', 'its type', 'the object', '"ReaDoNLY"', 'A composite read from another realm\'s storage carries the magic bytes in N. The mark survives copies and arguments; writing through it panics.'),
]


@fig
def typed_value():
    f = Fig('typed-value', 332, 'TypedValue{T, V, N}: nine values, three cells each')
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
    f.text(20, 322, 'T is the concrete type, V a Value, N eight bytes. Fixed-size numbers use N only; everything with identity is a Value that is also an Object.', 'f-t s')
    return f


# ---------------------------------------------------------------- blocks and heap items

@fig
def blocks_heap():
    f = Fig('blocks-heap', 500, 'Blocks alive inside add(3), from the names filetest above')

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
               'Created by OpCall from add\'s source node with one slot per declared name. NewBlock pre-fills the heap-marked slots with heap items; writes go into the item, not over it.', 'acc')
    y_for = 44 + hp + 30 + hf + 30 + ha + 30
    hfor = block(20, y_for, 'for block', ['0  i.loopvar'],
                 'One per loop. At the end of each iteration the VM replaces every heap item in the init slots with a new one, which is how closures see Go 1.22 loop variables. Here i is not captured, so it is a plain slot.')
    # parent arrows
    f.arrow(230, 44 + hp + 30 + 10, 230, 44 + hp + 4, label='Parent', dx=28, dy=3)
    f.arrow(230, 44 + hp + 30 + hf + 30 + 10, 230, 44 + hp + 30 + hf + 4)
    f.arrow(230, y_for + 10, 230, y_for - 26)
    f.text(20, y_for + hfor + 24, 'The if arm gets a block too, with no names of its own here.', 'f-t s')

    # heap items and the closure
    ay = 44 + hp + 30 + hf + 30
    f.box(330, ay, 170, 46, 'HeapItem', ['Value: n = 3'], 'mut', note='A one-slot box. Pointers and captures reference the item, so the value survives the block it was declared in.', id='hi-n')
    f.box(330, ay + 64, 170, 46, 'HeapItem', ['Value: sum'], 'mut', note='The same kind of box for sum. Assignments in the loop write into this item.', id='hi-sum')
    f.arrow(230, ay + 34 - 4, 329, ay + 23)
    f.arrow(230, ay + 34 + 32 - 4, 329, ay + 64 + 23)
    f.box(540, ay, 200, 92, 'FuncValue f', ['Captures: [HeapItem sum, HeapItem n]', 'body: return sum + n', 'source, type, parent block'], '',
          note='Created by OpFuncLit. The capture list from the dump, sum then n, is copied out of add\'s block into Captures. Each call of f copies the items into a fresh block, so a persisted closure never references the block it was born in.', lh=14)
    f.arrow(540, ay + 40, 501, ay + 23, 'acc')
    f.arrow(540, ay + 50, 501, ay + 64 + 23, 'acc')
    f.box(540, ay + 110, 200, 60, '&sum', ['PointerValue{Base: HeapItem, Index 0}'], 'mut',
          note='Taking the address of a captured or escaping local yields a pointer whose base is the heap item, never the block.', lcls='f-m s')
    f.arrow(540, ay + 133, 501, ay + 64 + 30, 'dash')
    f.lines(330, ay + 200, wrap('When add returns, its block and the for and if blocks are popped and gone. '
                                'The two heap items stay alive inside f\'s captures. That is the whole memory model: '
                                'blocks are transient, heap items are what escape.', 66), 'f-t s', lh=15)
    return f


# ---------------------------------------------------------------- machine trace

@fig
def machine_trace():
    f = Fig('machine-trace', 380)
    f.text(20, 24, 'func f(a, b int) int { x := a + b; return x }   called as f(2, 3)', 'f-m b')
    f.text(20, 44, 'The stacks after doOpEval has expanded a + b. Gas so far: 212, OpExec 130 plus OpEval 82.', 'f-t s')
    cols = [
        ('Ops', ['OpDefine', 'OpAdd', 'OpEval', 'OpEval'], 'Opcodes wait here; the loop pops from the top. Pushes went in reverse, so OpEval for a runs first, then OpEval for b, then OpAdd, then OpDefine.'),
        ('Exprs', ['a + b', 'b', 'a'], 'The binary expression stays underneath its operands until doOpAdd pops it. Each OpEval on a name pops that name.'),
        ('Stmts', ['x := a + b'], 'doOpExec only peeked the statement. doOpDefine pops it at the end.'),
        ('Values', [], 'Empty until the names are read. After both: 2, 3. After OpAdd: 5, computed in place into a\'s slot on the stack.'),
        ('Blocks', ['f: a=2 b=3', 'file', 'package'], 'The scope stack. OpDefine writes 5 into slot 3 of f\'s block, or into the heap item there if x were captured.'),
    ]
    cw, gap, cellh = 136, 12, 26
    base = 300
    for i, (title, cells, note) in enumerate(cols):
        x = 20 + i * (cw + gap)
        f.open_hot(note)
        f.rect(x, 60, cw, base - 60 + 4, 'mut', r=4)
        f.text(x + cw / 2, 78, title, 'f-t b', 'middle')
        for k, c in enumerate(cells):
            cy = base - 26 - k * (cellh + 4)
            f.rect(x + 8, cy, cw - 16, cellh, 'acc' if k == len(cells) - 1 else '', r=4)
            f.text(x + cw / 2, cy + 17, c, 'f-m', 'middle')
        if not cells:
            f.text(x + cw / 2, base - 20, '(empty)', 'f-t s', 'middle')
        f.close()
    f.text(20, 332, 'Frames: one for the call to f, recording the stack heights to unwind to. The top cell of each column runs next.', 'f-t s')
    f.text(20, 350, 'Gas per opcode, from machine.go: OpExec 130, OpEval 82 plus 4 per block hop for a name, int add 81, OpDefine 114 plus 79 per name.', 'f-t s')
    f.text(20, 368, 'The site steps through all seven states.', 'f-t s')
    return f


# ---------------------------------------------------------------- control flow

@fig
def control_flow():
    f = Fig('control-flow', 320)
    f.text(20, 24, 'for i := 0; i < n; i++ { body }', 'f-m b')
    f.box(20, 40, 200, 64, 'Frame', ['label, stack heights to unwind to'], '',
          note='Every for, range, switch and function call pushes a frame, so break, continue and return know how far to unwind.')
    f.box(236, 40, 210, 64, 'Block', ['i.loopvar; bodyStmt{NextBodyIndex, Cond, Post}'], '',
          note='The loop block owns a bodyStmt that remembers the next statement index, the condition and the post statement. The sticky OpForLoop walks that state.', lcls='f-m s')
    # state machine
    sx = 20
    nodes = [('Cond', 'Evaluate the condition.'), ('body stmt at NextBodyIndex', 'OpExec on the next statement of the body; the index advances.'), ('Post', 'i++ runs, then back to the condition. continue jumps here.')]
    xs = [sx, sx + 150, sx + 360]
    ws = [100, 190, 90]
    for (t, n), x, w in zip(nodes, xs, ws):
        f.box(x, 140, w, 36, None, [t], 'acc', note=n, lcls='f-t b', pad=10)
    f.arrow(xs[0] + ws[0], 158, xs[1] - 1, 158)
    f.text(xs[0] + ws[0] + 12, 152, 'true', 'f-t s')
    f.arrow(xs[1] + ws[1], 158, xs[2] - 1, 158)
    f.path('M%g 176 C%g 215 %g 215 %g 176' % (xs[2] + 45, xs[2] + 45, xs[0] + 50, xs[0] + 50))
    f.text(230, 228, 'OpForLoop is sticky: popped and pushed back every iteration, so loops do not recurse', 'f-t s', 'middle')
    f.box(20, 240, 420, 60, 'Cond false, or break',
          ['pop the block, pop the frame, drop the sticky opcode. break and continue pop frames until the loop with the right label.'],
          'mut', note='Leaving the loop restores the stacks to the heights the frame recorded.', lh=13)
    # the others
    others = [
        ('if', 'pushes a block only, no frame; the block is expanded with the chosen arm\'s names'),
        ('switch', 'a frame; clauses and cases tried one at a time; a type switch compares type ids or checks interface implementation'),
        ('range', 'one opcode per kind: arrays and slices, strings (one rune per step), maps (one linked-list entry per step)'),
        ('goto', 'resets the stacks to the heights recorded on the body statement'),
    ]
    y = 40
    for t, d in others:
        f.box(470, y, 270, 60, t, [d], '', note=d, lh=13)
        y += 66
    return f


# ---------------------------------------------------------------- function calls

CALL_STAGES = [
    ('1 · OpEval on the call', 'pushes OpPrecall, then an OpEval per argument in reverse, then one for the callee, so the callee evaluates first: Values f, a, b',
     'The call expression y := f(a, b). Because pushes go in reverse, the callee is read before any argument.'),
    ('2 · OpPrecall', 'peeks the callee under the arguments. A type value: OpConvert. A function: push a Frame and OpCall; OpEnterCrossing for a crossing function; cross(rlm) installs a fresh cur',
     'PushFrameCall also decides the storage realm for the call, the borrow rules.'),
    ('3 · OpCall', 'new block from the callee\'s source node and parent block; captured heap items copied in; arguments popped into the first slots, variadic ones packed; named results zeroed',
     'The block\'s slots follow the dump above: parameters, then results, then locals.'),
    ('4 · Body', 'Gno function: OpBody with the statements, plus a synthetic return when it declares no results. Native: OpReturn then OpCallNativeBody. defer: evaluate now, push a Defer on the frame',
     'A native function is a FuncValue with a Go body attached by the store\'s NativeResolver.'),
    ('5 · OpReturn, four flavors', 'OpReturn; OpReturnAfterCopy when named results live on the heap; OpReturnFromBlock when a bare return reads them; OpReturnToBlock plus the sticky OpReturnCallDefers when the frame holds defers',
     'The preprocessor and the frame choose the flavor.'),
    ('6 · Pop the frame', 'truncate every stack to the recorded heights, move the results down, restore the caller\'s Package and Realm. maybeFinalize finalizes the realm if the frame crossed a boundary',
     'This is where a realm call ends and its objects are saved.'),
]


@fig
def call_sequence():
    f = Fig('call-sequence', 350)
    f.text(20, 24, 'y := f(a, b)', 'f-m b')
    bw, bh = 232, 126
    for i, (t, d, n) in enumerate(CALL_STAGES):
        col, row = i % 3, i // 3
        x, y = 20 + col * (bw + 12), 40 + row * (bh + 30)
        f.box(x, y, bw, bh, t, [d], 'acc' if i in (1, 5) else '', note=n, id='call-%d' % (i + 1), lh=13)
        if col < 2:
            f.arrow(x + bw, y + bh / 2, x + bw + 11, y + bh / 2)
    f.path('M%g %g L%g %g L%g %g L%g %g' % (20 + 2 * 244 + bw / 2, 40 + bh, 20 + 2 * 244 + bw / 2, 40 + bh + 15,
                                            20 + bw / 2, 40 + bh + 15, 20 + bw / 2, 40 + bh + 29))
    return f


# ---------------------------------------------------------------- panics

@fig
def panic_unwind():
    f = Fig('panic-unwind', 380)
    f.box(20, 30, 340, 62, 'panic(v) inside an opcode handler',
          ['pushPanic records the Exception, unwinds to the last call frame, pushes OpPanic2 and OpReturnCallDefers'],
          'warn', note='The cooperative path. Nothing on the Go stack is consumed.', lh=13)
    f.box(400, 30, 340, 62, 'Go panic carrying an Exception, from the value layer',
          ['runOnce recovers it and Run converts it into the same path, so half a million panicking defers cost no Go stack'],
          'mut', note='Machine.Panic or a direct panic with an Exception deep in values.go. The iterative-recovery record explains why.', lh=13)
    f.arrow(190, 92, 190, 129)
    f.path('M570 92 L570 110 L360 110 L360 129')
    f.box(20, 130, 340, 40, None, ['the frame\'s deferred calls run, one at a time'], 'acc', note='doOpReturnCallDefers pops each Defer and runs it like a call.', lcls='f-t b')
    f.arrow(190, 170, 190, 209)
    f.box(20, 210, 340, 40, None, ['recover() called directly by a deferred function?'], '', note='The Go rule: only a direct call from the deferred function sees the exception.', lcls='f-t b')
    f.arrow(360, 230, 399, 230, label='yes', dy=-6)
    f.box(400, 210, 340, 40, None, ['exception cleared; the function returns normally'], 'acc', note='Recover returns the exception value and clears it. Exceptions chain to the previous one when a defer panics again.', lcls='f-t b')
    f.arrow(190, 250, 190, 289, label='no', dx=14, dy=4)
    f.box(20, 290, 340, 62, 'OpReturnCallDefers pops the frame',
          ['and re-raises in the caller. The next frame up runs its own defers: back to the top.'], '',
          note='The unwinding continues frame by frame, each one running its deferred calls first.', lh=13)
    f.path('M10 321 L4 321 L4 150 L19 150', 'dash')
    f.box(400, 290, 340, 62, 'Realm boundary reached',
          ['the transaction aborts with UnhandledPanicError, unless a revive frame catches it. The callee\'s state may be half written, so the caller cannot recover it.'],
          'warn', note='A panic that crosses a realm boundary is never recoverable by the caller. Tests use revive to observe the abort.', lh=13)
    f.arrow(360, 321, 399, 321)
    f.text(20, 372, 'Panic values print through a bounded printer: no user methods, 1024 bytes at most. A Stacktrace keeps 128 calls with the middle elided.', 'f-t s')
    return f


# ---------------------------------------------------------------- uverse

@fig
def uverse():
    f = Fig('uverse', 330)
    chain = [('if block', 'Depth 1'), ('for block', 'Depth 2'), ("add's block", 'Depth 3'), ('file block', 'Depth 4'), ('package block', 'Depth 5'), ('.uverse', 'Depth 0')]
    y = 30
    for i, (t, d) in enumerate(chain):
        cls = 'acc' if t == '.uverse' else ''
        f.box(20, y, 150, 30, None, [t], cls, lcls='f-t b', pad=10,
              note='A name written in the if arm reaches this block at this depth.' if t != '.uverse' else
              'The universe block, a package named .uverse at the root of every chain. A name whose path has Depth 0 is read from it directly: the op_eval fast path checks nx.Path.Depth == 0 before climbing anything.')
        f.text(180, y + 19, d, 'f-t s')
        if i < len(chain) - 1:
            f.arrow(95, y + 30, 95, y + 44)
        y += 44
    f.text(20, 322, 'Depth counts hops from the block where the name is used, so the same name has a different depth in each scope.', 'f-t s')
    rows = [
        ('Types', 'bool string int int8 ... uint64 float32 float64 byte rune any error address realm',
         'The basic types are defined here, as type values in the block\'s slots.'),
        ('Values', 'true  false  nil  iota', 'Constants, resolved by the preprocessor into ConstExprs.'),
        ('Functions', 'append cap copy delete len make new print println panic recover',
         'Each is a FuncValue whose body is a Go closure over the Machine, installed by DefineNative.'),
        ('Gno additions', 'cross   attach (reserved, panics)   istypednil   revive',
         'cross checks its argument is the running frame\'s own cur. revive works only when the machine enables it, which tests do.'),
        ('Hidden', '.cur   .origin   cross1', 'For the compiler\'s own use. The leading dot or reserved spelling means user source cannot produce them.'),
    ]
    y = 30
    for t, body, n in rows:
        f.box(260, y, 480, 50, t, [body], 'mut' if t == 'Hidden' else '', note=n, lcls='f-m s', lh=13)
        y += 56
    return f


# ---------------------------------------------------------------- object ids

@fig
def object_ids():
    f = Fig('object-ids', 275)
    f.text(20, 24, 'PkgID', 'f-t h')
    steps = [
        ('"gno.land/r/test"', 'the package path', 118),
        ('sha256', 'a8ada09dee16 ... 084a30, 32 bytes', 140),
        ('keep 20 bytes', 'then the top four bits become flags: stdlib, immutable, internal', 180),
        ('08ada09dee16d791fd40 6d629fe29bb0ed084a30', 'PkgID: a user realm has no flag set, so a8 became 08', 232),
    ]
    x = 20
    notes = ['The path is the only input. The same path on any node yields the same id.',
             'Measured: sha256 of the path starts with a8ad. The id in the zrealm1 trace starts with 08ad.',
             'Immutable covers standard library and pure packages, whose objects are never reference-counted from other realms.',
             'The id in the zrealm1 filetest trace, produced by the same path.']
    for (t, d, w), n in zip(steps, notes):
        f.box(x, 40, w, 76, None, [t, d], 'acc' if w == 232 else '', note=n, lcls='f-m s' if w == 232 else 'f-t s', lh=14, pad=8)
        if w != 232:
            f.arrow(x + w, 78, x + w + 12, 78)
        x += w + 14
    f.text(20, 150, 'ObjectID', 'f-t h')
    f.box(20, 166, 300, 56, None, ['PkgID : NewTime', '08ada09d...084a30:6  the struct created in zrealm1'], '', lcls='f-m s', lh=15,
          note='NewTime is a per-realm counter. The package value itself is always :1.')
    f.box(340, 166, 400, 56, 'Three states', ['empty  →  allocated: PkgID stamped, NewTime 0  →  finalized: counter taken'], '', lcls='f-m s',
          note='The allocator stamps the creating realm as soon as it sees the object. Only finalization takes a counter value, and only a finalized object is real.')
    f.text(20, 254, 'An object is saved under the realm that allocated it, even when another realm\'s code did the saving.', 'f-t s')
    return f


# ---------------------------------------------------------------- refcount and escape

@fig
def refcount_escape():
    f = Fig('refcount-escape', 250)
    panels = [
        ('Owned: RefCount 1', 'acc',
         ['one holder', 'owner id set', 'hash folded into the owner\'s bytes', 'one Merkle chain down the tree'],
         'The common case. A tree of singly referenced objects is one Merkle chain, saved with its root.', 1),
        ('Escaped: RefCount 2', '',
         ['two holders', 'owner cleared, saved on its own', 'hash written to the IAVL store under its id', 'once escaped, always escaped'],
         'The second reference makes the object independent. The separate Merkle index of escaped hashes is a stub at this commit.', 2),
        ('Deleted: RefCount 0', 'warn',
         ['no holder left', 'removed with everything it owned', 'cycles are not supported', 'named for a later phase'],
         'processNewDeletedMarks decrements the children recursively, so a dropped subtree goes in one pass.', 0),
    ]
    for i, (t, cls, ls, n, refs) in enumerate(panels):
        x = 20 + i * 246
        f.open_hot(n)
        f.rect(x, 36, 234, 190, cls)
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
        ty = 164
        for l in ls:
            f.text(x + 10, ty, l, 'f-t s')
            ty += 14
        f.close()
    return f


# ---------------------------------------------------------------- finalization

@fig
def finalization():
    f = Fig('finalization', 420)
    f.text(20, 24, 'zrealm1.gno: root = InnerNode{Key: "somekey"}, inside a realm call', 'f-m b')
    f.box(20, 44, 220, 52, ':1 PackageValue', ['the package itself; always NewTime 1'], '', id='obj-1',
          note='Every realm\'s first object. It holds the package block and one block per file.')
    f.box(20, 130, 220, 66, ':2 Block (package)', ['IsEscaped, RefCount 2', 'slot 0 → :3'], '', id='obj-2',
          note='The package block. It is escaped because it has two holders, so it is saved on its own. Its ModTime moves from 0 to 6 in the trace.')
    f.arrow(130, 96, 130, 129)
    f.box(300, 130, 220, 66, ':3 HeapItem root', ['owner :2, RefCount 1', 'Value: was {}, now → :6'], '', id='obj-3',
          note='The heap item behind the package variable root. Assign2 writes into it and calls DidUpdate with the old child, none, and the new child, the struct.')
    f.arrow(240, 163, 299, 163)
    f.box(300, 230, 220, 66, ':6 InnerNode', ['owner :3, RefCount 1', 'Key: "somekey", Left {}, Right {}'], 'acc', id='obj-6',
          note='Created this transaction. DidUpdate marked it new-real; processNewCreatedMarks assigned :6 depth-first. The string inside it is not an object, so nothing else became real.')
    f.arrow(410, 196, 410, 229)
    f.box(540, 44, 200, 252, 'Realm: writes, in order', [
        'c[...:6]  (269 bytes)',
        '  the new struct, owner :3',
        'u[...:3]',
        '  Value: RefType InnerNode',
        '  + RefValue{:6, hash}',
        'u[...:2]',
        '  hash of :3 changed,',
        '  ModTime 0 → 6',
        '',
        'then: byte delta per realm',
        '  → storage deposit',
    ], 'mut', id='writes', lcls='f-m s', lh=16, nowrap=True,
        note='Exactly what the filetest\'s Realm: directive prints. Children are saved before parents, and only new or dirty objects are written.')
    steps = ['1 new-real marks: ids depth-first, children counted', '2 deleted marks: children decremented', '3 escaped marks: demote or confirm',
             '4 realm record saved if the counter moved', '5 dirty ancestors up to the package block', '6 save new and dirty, children first',
             '7 remove the deleted', '8 byte deltas per realm, for the deposit']
    y = 322
    for i, s in enumerate(steps):
        col, row = i % 2, i // 2
        f.text(20 + col * 370, y + row * 16, s, 'f-t s')
    f.text(20, 404, 'FinalizeRealmTransaction runs when a call returns across a realm boundary, and again after package initialization.', 'f-t s')
    return f


# ---------------------------------------------------------------- persist copy

@fig
def persist_copy():
    f = Fig('persist-copy', 340)
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
        note='From the zrealm1 trace. Every child object becomes a RefValue with its id and hash, every declared type a RefType with its id, every syntax node a RefNode with its location.')
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
    f.text(20, 308, 'The same three stand-ins cover the three kinds of reference: RefValue for objects, RefType for declared types, RefNode for syntax nodes.', 'f-t s')
    f.text(20, 326, 'A RefNode carries only a Location: package path, file, line and column span. The store loads the node on demand.', 'f-t s')
    return f


# ---------------------------------------------------------------- store layout

@fig
def store_layout():
    f = Fig('store-layout', 350)
    f.rect(20, 30, 720, 160, 'mut')
    f.text(30, 50, 'defaultStore, the only Store implementation', 'f-t b')
    f.box(30, 62, 345, 118, 'baseStore: a plain database adapter on baseKey', [
        'oid:<pkgid>:<newtime>   objects, hash || bytes',
        'tid:<typeid>            types',
        '#realm                  realm records',
        'pkgidx:                 the ordered package list',
        'commits with no hash: not in the application hash',
    ], '', lcls='f-m s', lh=15, nowrap=True,
        note='Objects live here. The adapter commits with no hash, so this store does not feed the block\'s application hash.')
    f.box(385, 62, 345, 118, 'iavlStore: the Merkle tree on mainKey', [
        'pkg:<path>              package sources',
        'escaped object hashes   under their ids',
        '',
        'its root feeds the application hash',
        'a Merkle read costs a depth-scaled fee',
    ], 'acc', lcls='f-m s', lh=15, nowrap=True,
        note='Only the IAVL contents are part of consensus state. Package sources and the hashes of escaped objects are what every validator agrees on.')
    f.box(20, 206, 232, 118, 'Per transaction', [
        'BeginTransaction forks the store with fresh object, type and realm caches and a write log around the syntax-node cache. Write commits the log, only when the transaction succeeded.',
    ], '', lh=13, note='The base app creates the forked store at the start of each transaction and commits it only on success.')
    f.box(264, 206, 232, 118, 'Hooks', [
        'PackageGetter: loads packages from disk on first import, off-chain. NativeResolver: attaches Go bodies to native functions after a FuncValue is loaded. Standard library bytes are cached at node start and read without I/O gas.',
    ], '', lh=13, note='The two hooks make the store complete; the stdlib cache keeps library reads free.')
    f.box(508, 206, 232, 118, 'Between messages', [
        'ClearObjectCache runs before every message so nothing leaks between them. GarbageCollectObjectCache drops cached objects the last GC cycle did not visit.',
    ], '', lh=13, note='The object cache is per transaction; the GC pass keeps it from growing across one.')
    return f


# ---------------------------------------------------------------- cur chain

@fig
def cur_chain():
    f = Fig('cur-chain', 280)
    f.text(20, 24, 'MsgCall app.F, where F calls lib.G(cross(cur), ...)', 'f-m b')
    boxes = [
        ('signer', ['g1abc... the transaction\'s signer'], 'mut', 'At the chain root there is no caller frame. MsgCall synthesizes pkg.F(.origin, args...), and .origin lowers to a crossing call whose prev is built from the signer.'),
        ('F\'s cur', ['addr: r/demo/app', 'prev → signer'], 'acc', 'installCrossingCur minted this .grealm value for F. cur.Previous() is the signer, unforgeably.'),
        ('G\'s cur', ['addr: r/demo/lib', 'prev → app'], 'acc', 'Minted for G because F wrote cross(cur). G sees app as its previous realm, whatever helper made the call.'),
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
    f.box(20, 160, 350, 74, 'H(cur) inside the same realm', ['no new link: H receives F\'s cur unchanged. Written against another realm\'s function, the preprocessor rejects it.'], '', lh=13,
          note='Calling a crossing function with a bare cur is only legal within its own realm.')
    f.box(390, 160, 350, 74, 'chain/runtime/unsafe', ['PreviousRealm and CurrentRealm walk the frames counting WithCross frames and can be fooled from a helper. Hence the name.'], 'warn', lh=13,
          note='The older API. Prefer cur.Previous(), which reads the captured chain.')
    f.text(20, 262, 'Realm values are never persisted: store the address instead. main(cur realm) and init(cur realm) are called with .cur, already crossed.', 'f-t s')
    return f


# ---------------------------------------------------------------- interrealm cases

CASES = [
    ('lib.F(cross(cur), x)', 'lib', 'lib', True, 'crossing call: a new cur whose prev is app',
     'The explicit switch. Identity and storage both move to lib for the duration of F, and finalization runs when F returns.'),
    ('tree.Set(k, v)', 'app', 'app', True, 'borrow rule 2: tree is owned by app, Set comes from p/avl',
     'A pure package method on a receiver that belongs to app runs with app\'s storage, so avl.Tree.Set can write the tree app owns. cur.Previous() is unchanged.'),
    ('lib.F(cur, x)', '', '', False, 'rejected: cur belongs to app',
     'A crossing function of another realm called with a bare cur. The preprocessor refuses it; only cross(cur) may switch identity.'),
    ('lib.Data.N = 1', 'app', 'app', False, 'readonly taint: the value was read from lib\'s storage',
     'Reading a field, element or package variable of another realm marks the result readonly. The mark survives copies and arguments, and any write through it panics.'),
    ('obj.N = 1', 'app', 'app', False, 'ownership: obj is a real object of lib, and lib is not the storage realm',
     'A write to a real object is refused unless the object\'s realm is the active storage realm.'),
    ('lib.T{}   new(lib.T)   make(lib.S, n)', 'app', 'app', False, 'construction: lib\'s types only through lib\'s constructors',
     'A composite literal, new or make of a type declared in another realm panics at the allocation site.'),
]


@fig
def interrealm_cases():
    f = Fig('interrealm-cases', 420)
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
    f.text(20, y + 14, 'Identity is who acts, cur; storage is whose objects may be written. The borrow rules set storage for every call that is not an explicit cross.', 'f-t s')
    return f


# ---------------------------------------------------------------- gas

@fig
def gas_sources():
    f = Fig('gas-sources', 306)
    srcs = [
        ('CPU', ['a constant per opcode: OpExec 130, OpEval 82, int add 81, OpDefine 114', 'slopes: per block hop, per defined name, per parameter, per element, per bit', 'natives: a table by package and name'],
         'One gas is one nanosecond on the reference Xeon. cmd/calibrate produced the constants.'),
        ('Allocation', ['bytes per value, 500 MB per transaction', 'gas from a table indexed by log2 of the size', 'over budget: recount every reachable object, charged per visit, and continue if it fits'],
         'An accounting GC: Go\'s collector frees the memory, the VM only recounts.'),
        ('Store', ['a cache miss: amino decode gas per byte, plus tm2 I/O gas', 'a Merkle read: a depth-scaled flat fee plus a per-byte fee', 'a cache hit costs nothing'],
         'The constants and the reasoning are in the gas refactor record.'),
        ('Storage deposit', ['byte delta per realm × 100 ugnot per byte', 'locked from the caller into the realm\'s deposit address', 'refunded on a negative delta: the one cost that comes back'],
         'Settled after the message by processStorageDeposit, from the finalization byte deltas.'),
    ]
    bw = 172
    for i, (t, ls, n) in enumerate(srcs):
        x = 20 + i * (bw + 10)
        f.box(x, 30, bw, 168, t, ls, 'acc' if i == 3 else '', note=n, lh=13)
        f.arrow(x + bw / 2, 198, x + bw / 2, 230)
    f.rect(20, 232, 720, 34, 'mut')
    f.text(380, 254, 'one gas meter per transaction', 'f-t b', 'middle')
    f.text(20, 294, 'Queries run under their own meter, 3 billion gas and a 1.5 GB allocator, and are never committed.', 'f-t s')
    return f


# ---------------------------------------------------------------- natives

@fig
def native_binding():
    f = Fig('native-binding', 250)
    steps = [
        ('time.gno', ['func now() int64', '// injected', 'no body'], 'acc', 'A standard library function declared without a body and marked injected has a Go implementation in a sibling .go file.'),
        ('misc/genstd', ['scans the stdlibs', 'writes generated.go'], '', 'A generator run at build time, not on chain.'),
        ('generated.go', ['45 bindings at this commit', 'path + name → Go func', 'reflection marshals values'], '', 'The table the store consults. Gno values are marshalled to Go with reflection and back.'),
        ('NativeResolver', ['a store hook', 'attaches the Go body when the FuncValue is loaded'], '', 'Native functions are persisted like any FuncValue; the body is re-attached on every load.'),
        ('ExecContext', ['chain id, height, timestamp', 'origin caller, coins sent', 'banker, parameter store'], 'mut', 'A native that needs chain state takes the Machine and reads its Context. That is how time.Now returns the block time.'),
    ]
    bw = 136
    for i, (t, ls, cls, n) in enumerate(steps):
        x = 20 + i * (bw + 14)
        if i == 0:
            f.box(x, 40, bw, 110, t, ls, cls, note=n, lcls='f-m s', lh=15, nowrap=True)
        else:
            f.box(x, 40, bw, 110, t, ls, cls, note=n, lh=13)
        if i < 4:
            f.arrow(x + bw, 95, x + bw + 13, 95)
    f.text(20, 186, 'Under test, gnovm/tests/stdlibs overlays a testing package: SetRealm, SetOriginCaller, SkipHeights, IssueCoins rewrite the mock context.', 'f-t s')
    f.text(20, 204, 'Native gas: a base plus a slope on the argument length, from a table keyed by package and name.', 'f-t s')
    return f


# ---------------------------------------------------------------- keeper flows

@fig
def keeper_flows():
    f = Fig('keeper-flows', 400)
    lanes = [
        ('AddPackage', ['validate the path; refuse an existing public package', 'type-check strict, genesis-strict at height 0', 'namespace ownership; coins to the package address', 'preprocess allocator: folding is metered', 'run with save on: variables, then init.<n>', 'settle the storage deposit'],
         'MsgAddPackage. The package runs once and is saved; every later call skips to execution.'),
        ('Call', ['refuse a non-crossing function', 'bind the package as pkg in a throwaway main', 'parse pkg.F(cross, args...) written as text', 'replace the first argument with .origin', 'evaluate; results joined as strings', 'settle the storage deposit'],
         'MsgCall. The arguments arrive as strings and are converted to the parameter types.'),
        ('Run', ['path forced to <domain>/e/<caller>/run', 'type-check relaxed; a manifest is generated', 'the package is marked private', 'run without saving', 'main called in a second machine', 'nothing is stored'],
         'MsgRun. A throwaway main package; nothing persists beyond what its calls write into other realms.'),
        ('Query', ['fork a throwaway store', 'load the package', 'parse the expression; .cur prepended for a crossing callee', 'evaluate under 3 billion gas and 1.5 GB', 'qrender is the same with Render(path)', 'never committed'],
         'vm/qeval and vm/qrender. Read-only, on a forked store that is dropped afterwards.'),
    ]
    lw, ch = 172, 46
    for i, (t, steps, n) in enumerate(lanes):
        x = 20 + i * (lw + 10)
        f.open_hot(n)
        f.rect(x, 30, lw, 30, 'acc', r=4)
        f.text(x + lw / 2, 50, t, 'f-t b', 'middle')
        for k, s in enumerate(steps):
            y = 66 + k * (ch + 6)
            f.rect(x, y, lw, ch, '', r=4)
            ty = y + 16
            for piece in wrap(s, 27)[:3]:
                f.text(x + 8, ty, piece, 'f-t s')
                ty += 13
            if k < len(steps) - 1:
                f.arrow(x + lw / 2, y + ch, x + lw / 2, y + ch + 5, 'thin')
        f.close()
    f.text(20, 392, 'Restart: Initialize re-preprocesses every stored package, type-checks the standard library into the permanent cache, fills the byte cache.', 'f-t s')
    return f


# ---------------------------------------------------------------- tooling map

@fig
def tooling_map():
    f = Fig('tooling-map', 260)
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

    span(84, 1, 2, 'gno lint: type-check and preprocess, no run', 'Per the lint and transpile record. Errors come out as file:line messages without executing anything.')
    span(118, 0, 4, 'gno test, gno run: everything, over the production test store with a mock context', 'A filetest sets PKGPATH, MAXALLOC and SEND in comments and asserts on Output, Error, Realm, Events, Preprocessed, Stacktrace, Gas, Storage and TypeCheckError blocks.')
    span(152, 0, 0, 'gno fix: transpiler', 'gno fix rewrites source from older Gno versions through the transpiler, Gno to Go source.', 'mut')
    span(152, 3, 3, 'debugger: gno run -debug', 'Steps opcodes with breakpoints.', 'mut')
    span(186, 3, 4, 'benchops: opcode and store timing', 'The numbers behind the gas constants come from cmd/calibrate and cmd/benchstore.', 'mut')
    f.text(20, 240, 'Build tags debug and debugAssert turn on tracing and invariant panics. gno doc and vm/qdoc read the sources without running them.', 'f-t s')
    return f


# ---------------------------------------------------------------- determinism

@fig
def determinism():
    f = Fig('determinism', 250)
    f.text(20, 24, 'Source of drift between validators', 'f-t b')
    f.text(390, 24, 'What the VM does', 'f-t b')
    rows = [
        ('goroutines, channels, select', 'the opcodes panic; the types exist, the operations do not'),
        ('the CPU\'s floating point', 'software floats, so every node computes the same bits'),
        ('Go map iteration order', 'a MapValue iterates its insertion-ordered list'),
        ('clock, randomness, network', 'time.Now is the block timestamp; nothing else exists'),
        ('the capacity of []byte(s)', 'equals the length; Go\'s growth policy is unspecified'),
        ('type-check error text', 'the Go toolchain is pinned in the Makefile'),
    ]
    y = 40
    for a, b in rows:
        f.rect(20, y, 350, 26, 'warn', r=4)
        f.text(30, y + 17, a, 'f-t')
        f.arrow(370, y + 13, 389, y + 13)
        f.rect(390, y, 350, 26, 'acc', r=4)
        f.text(400, y + 17, b, 'f-t')
        y += 32
    f.text(20, y + 14, 'Determinism is enforced in the language, not trusted to the program: the same transaction yields the same state on every node.', 'f-t s')
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
