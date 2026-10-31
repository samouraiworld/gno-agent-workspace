// Brings the figures under figures/ to life on the site. The markdown embeds
// each one as a plain image, which is what GitHub shows. Here the SVG is
// inlined so the theme applies, its data-note attributes become hover text,
// and three figures get a step-through of the process they draw.
(function () {
  'use strict';

  // ---------------------------------------------------------------- steppers

  // Each step: title, note, panels (name -> cells, drawn top to bottom; a
  // stack's top is its first cell), hl (ids in the SVG to highlight).
  var STEPS = {
    'machine-trace': {
      intro: 'f(2, 3) is running: f’s block holds a = 2, b = 3, the result and x. Each stack lists its top first.',
      stacks: ['Ops', 'Exprs', 'Values'],
      steps: [
        { title: 'Before the statement', gas: 0,
          note: 'The sticky OpBody walks f’s statements through a cursor in f’s block. The function value f sits at the bottom of Values for the whole call.',
          Ops: ['OpBody'], Exprs: [], Values: ['f'] },
        { title: 'OpBody takes x := a + b, gas 73', gas: 73,
          note: 'OpBody hands the statement to the statement switch. For a define it pushes OpDefine, then OpEval for the right-hand side, so the eval runs first. The statement stays in the cursor, where OpDefine will find it.',
          Ops: ['OpEval', 'OpDefine', 'OpBody'], Exprs: ['a + b'], Values: ['f'] },
        { title: 'doOpEval on a + b, gas 82', gas: 155,
          note: 'A binary expression pushes the operator, then an eval for each side, the right side first so that the left side runs first. The expression itself stays under its operands.',
          Ops: ['OpEval', 'OpEval', 'OpAdd', 'OpDefine', 'OpBody'], Exprs: ['a', 'b', 'a + b'], Values: ['f'] },
        { title: 'doOpEval on a, gas 82 + 4', gas: 241,
          note: 'The name fast path: pop the expression, climb Depth − 1 blocks, read slot Index, push the value. The 4 is the price per unit of depth for a name at depth 1.',
          Ops: ['OpEval', 'OpAdd', 'OpDefine', 'OpBody'], Exprs: ['b', 'a + b'], Values: ['2', 'f'] },
        { title: 'doOpEval on b, gas 82 + 4', gas: 327,
          note: 'Same path for b. Both operands are now on the values stack, a below b.',
          Ops: ['OpAdd', 'OpDefine', 'OpBody'], Exprs: ['a + b'], Values: ['3', '2', 'f'] },
        { title: 'doOpAdd, gas 81 for ints', gas: 408,
          note: 'Pops a + b and the 3, and adds the 3 into the 2 beneath it with a switch on the operand type. Floats would go through software floating point.',
          Ops: ['OpDefine', 'OpBody'], Exprs: [], Values: ['5', 'f'] },
        { title: 'doOpDefine, gas 114 + 79', gas: 601,
          note: 'Pops 5 and the statement, and writes 5 into x’s slot, or into the heap item in that slot when a closure captured x. OpBody is on top again and takes return x. Total for the statement: 601 gas.',
          Ops: ['OpBody'], Exprs: [], Values: ['f'] }
      ]
    },
    'call-sequence': {
      intro: 'y := f(a, b), with f a Gno function of two parameters and one result.',
      stacks: ['Ops', 'Values', 'Frames'],
      steps: [
        { title: '1 · OpEval on the call', hl: ['call-1'],
          note: 'Pushes OpPrecall, then an OpEval per argument in reverse, then an OpEval for the callee. Pushes go in reverse, so the callee is read first.',
          Ops: ['OpEval f', 'OpEval a', 'OpEval b', 'OpPrecall', 'OpDefine'], Values: [], Frames: ['caller'] },
        { title: 'callee and arguments read', hl: ['call-1'],
          note: 'Three evals later the values stack holds the function value under its arguments.',
          Ops: ['OpPrecall', 'OpDefine'], Values: ['b', 'a', 'f'], Frames: ['caller'] },
        { title: '2 · OpPrecall', hl: ['call-2'],
          note: 'Looks at the callee beneath the arguments. A function value: push a call frame that records the stack heights, the callee, and the package and realm to restore, then OpCall. A crossing callee adds OpEnterCrossing, and a call written with cross(cur) gets a fresh cur on the frame. A nil function panics.',
          Ops: ['OpCall', 'OpDefine'], Values: ['b', 'a', 'f'], Frames: ['f: heights, package, realm', 'caller'] },
        { title: '3 · OpCall', hl: ['call-3'],
          note: 'f’s block comes from the Machine’s pool, or is allocated, with the file block as its parent; a closure’s captured heap items go into the last slots; named results are zeroed; a and b are popped into the first slots. OpBody carries the statements, plus a synthetic empty return if f declares no results.',
          Ops: ['OpBody', 'OpDefine'], Values: [], Frames: ['f: heights, package, realm', 'caller'] },
        { title: '4 · body runs', hl: ['call-4'],
          note: 'OpBody executes the statements one by one. A defer evaluates its function and arguments now and pushes a Defer onto the frame. A native function would have pushed OpReturn and OpCallNativeBody instead.',
          Ops: ['OpBody', 'OpDefine'], Values: ['(result)'], Frames: ['f: defers []', 'caller'] },
        { title: '5 · OpReturn', hl: ['call-5'],
          note: 'One of four forms, chosen by the return statement and the frame: OpReturn; OpReturnAfterCopy when the results are named and return lists values; OpReturnFromBlock for a bare return with named results; OpReturnCallDefers when the frame holds deferred calls.',
          Ops: ['OpReturn', 'OpDefine'], Values: ['(result)'], Frames: ['f', 'caller'] },
        { title: '6 · frame popped', hl: ['call-6'],
          note: 'If the frame is a realm boundary, maybeFinalize finalizes the realm first. Then every stack is cut back to the frame’s heights, the call’s blocks go back to the pool, the result moves down over the callee, and the caller’s Package and Realm are restored. OpDefine then stores the result in y.',
          Ops: ['OpDefine'], Values: ['(result)'], Frames: ['caller'] }
      ]
    },
    'finalization': {
      intro: 'zrealm1.gno assigns InnerNode{Key: "somekey"} to the package variable root. Objects: :1 the package, :2 its block, :3 the heap item behind root.',
      stacks: [],
      steps: [
        { title: 'During the call: Assign2', hl: ['obj-3', 'obj-6'],
          note: 'The write into the heap item goes through PointerValue.Assign2, which calls Realm.DidUpdate with the parent :3, no child before, and the struct after. :3 is marked dirty with ModTime 5; the struct gains a reference, takes :3 as its owner and is marked new-real. Its PkgID was stamped when it was allocated; NewTime is still 0.' },
        { title: '1 · processNewCreatedMarks', hl: ['obj-6'],
          note: 'Every object marked new-real gets an id, depth first: the struct becomes :6. Its fields hold a string and two nils, none of them objects, so nothing further becomes real.' },
        { title: '2 · processNewDeletedMarks', hl: [],
          note: 'Objects still at zero references are deleted, and each of their children loses a reference, recursively. Nothing is deleted here: root was nil before.' },
        { title: '3 · processNewEscapedMarks', hl: ['obj-2'],
          note: 'An object marked new-escaped that ended with one reference is demoted, and the others are confirmed. Nothing escapes here; :2, the package block, was already escaped with two references.' },
        { title: '4 · realm record', hl: [],
          note: 'The realm’s counter moved, from 5 to 6, so its record under oid:<pkgid>:1#realm is saved.' },
        { title: '5 · markDirtyAncestors', hl: ['obj-3', 'obj-2'],
          note: 'From each created or dirty object up through its owners, each owner marked dirty so it stores a fresh hash of its child: :6 has owner :3, and :3 has owner :2. The walk stops at :2, because it is escaped.' },
        { title: '6 · saveUnsavedObjects', hl: ['writes', 'obj-6', 'obj-3', 'obj-2'],
          note: 'Created objects are written children first, then dirty ones, refusing an object or a type of a private package and any realm value. The Realm: directive prints exactly this: c[:6], then u[:3], then u[:2].' },
        { title: '7 · removeDeletedObjects', hl: [],
          note: 'Dead objects are deleted from the store, and an escaped one’s Merkle tree entry with them. None this time.' },
        { title: '8 · byte deltas', hl: ['writes'],
          note: 'Each save returns its change in bytes, summed per realm on the store: 269 + 156 + 5 = 430 for gno.land/r/test. After the message, the keeper locks 430 × storage_price from the caller into the realm’s deposit address.' }
      ]
    }
  };

  // ---------------------------------------------------------------- typed value tabs

  var TV = [
    ['x := 42', 'int', 'nil', '42', 'SetInt writes an int64 into the eight bytes. No allocation, and the allocator never sees it.'],
    ['ok := true', 'bool', 'nil', '01 00 00 00 00 00 00 00', 'SetBool writes the bool into the first byte of N.'],
    ['f := 2.5', 'float64', 'nil', '0x4004000000000000', 'SetFloat64 stores the IEEE-754 bits. Arithmetic runs in software so every node computes the same bits.'],
    ['s := "hi"', 'string', 'StringValue("hi")', '0', 'A Go string in V. Not an Object: no identity, saved inside its holder.'],
    ['p := &node', '*InnerNode', 'PointerValue{TV, Base, Index}', '0', 'A slot address: the base object, an array, struct, block, map or heap item, plus an index into it. Index −1 is a block’s blank slot, −2 a map entry reached through Key.'],
    ['xs := []int{1, 2}', '[]int', 'SliceValue{Base, Offset: 0, Length: 2, Maxcap: 2}', '0', 'A window onto an ArrayValue, which is the Object with identity. Copying the slice copies the window.'],
    ['v := InnerNode{...}', 'gno.land/r/test.InnerNode', '*StructValue{ObjectInfo, Fields: [3]TypedValue}', '0', 'An Object: ObjectInfo carries id, owner, reference count and the dirty marks. Copying a struct copies its fields into a new object.'],
    ['var e any', 'nil', 'nil', '0', 'The undefined typed value: no type, no value. In the zrealm1 trace the nil Left and Right fields serialize as {}.'],
    ['var e any = 7', 'int', 'nil', '7', 'An interface has no box of its own: converting 7 to any leaves the cell as it is, so T holds the concrete type int, exactly as for x := 7.']
  ];

  // ---------------------------------------------------------------- helpers

  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text != null) e.textContent = text;
    return e;
  }

  function notes(fig) {
    var hots = fig.querySelectorAll('[data-note]');
    if (!hots.length) return;
    var panel = el('div', 'fig-note', 'Hover or tap a highlighted part for its note.');
    fig.insertBefore(panel, fig.querySelector('figcaption'));
    var pinned = null;
    function show(h) {
      panel.textContent = h.getAttribute('data-note');
      Array.prototype.forEach.call(hots, function (o) { o.classList.toggle('on', o === h); });
    }
    Array.prototype.forEach.call(hots, function (h) {
      h.setAttribute('tabindex', '0');
      h.addEventListener('mouseenter', function () { if (!pinned) show(h); });
      h.addEventListener('focus', function () { show(h); });
      h.addEventListener('click', function (e) {
        e.preventDefault();
        pinned = pinned === h ? null : h;
        show(h);
        panel.classList.toggle('pinned', !!pinned);
      });
    });
    fig.addEventListener('mouseleave', function () { if (pinned) show(pinned); });
  }

  function stepper(fig, name) {
    var spec = STEPS[name];
    var box = el('div', 'stepper');
    var head = el('div', 'stepper-head');
    var prev = el('button', 'stepper-btn', '← back');
    var next = el('button', 'stepper-btn', 'next →');
    var title = el('span', 'stepper-title');
    var count = el('span', 'stepper-count');
    head.appendChild(prev); head.appendChild(title); head.appendChild(count); head.appendChild(next);
    box.appendChild(el('p', 'stepper-intro', spec.intro));
    box.appendChild(head);
    var panels = el('div', 'stepper-panels');
    box.appendChild(panels);
    var note = el('p', 'stepper-note');
    box.appendChild(note);
    if (typeof spec.steps[0].gas === 'number') {
      var gas = el('p', 'stepper-gas');
      box.appendChild(gas);
    }
    var i = 0;
    function render() {
      var s = spec.steps[i], p = spec.steps[i - 1] || {};
      title.textContent = s.title;
      count.textContent = (i + 1) + ' / ' + spec.steps.length;
      note.textContent = s.note;
      if (gas) gas.textContent = 'gas so far: ' + s.gas;
      panels.innerHTML = '';
      spec.stacks.forEach(function (k) {
        var col = el('div', 'stk');
        col.appendChild(el('div', 'stk-name', k));
        var cells = s[k] || [], before = p[k] || [];
        if (!cells.length) col.appendChild(el('div', 'stk-empty', 'empty'));
        cells.forEach(function (c, idx) {
          var fresh = before.indexOf(c) === -1 || before.length - before.indexOf(c) !== cells.length - idx;
          col.appendChild(el('div', 'stk-cell' + (idx === 0 ? ' top' : '') + (fresh ? ' fresh' : ''), c));
        });
        panels.appendChild(col);
      });
      var svg = fig.querySelector('svg');
      if (svg) {
        Array.prototype.forEach.call(svg.querySelectorAll('.step-on'), function (n) { n.classList.remove('step-on'); });
        (s.hl || []).forEach(function (id) {
          var n = svg.querySelector('#' + id);
          if (n) n.classList.add('step-on');
        });
      }
      prev.disabled = i === 0;
      next.disabled = i === spec.steps.length - 1;
    }
    prev.addEventListener('click', function () { if (i > 0) { i -= 1; render(); } });
    next.addEventListener('click', function () { if (i < spec.steps.length - 1) { i += 1; render(); } });
    box.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowRight') next.click();
      if (e.key === 'ArrowLeft') prev.click();
    });
    box.tabIndex = 0;
    fig.insertBefore(box, fig.querySelector('figcaption'));
    render();
  }

  function typedValueTabs(fig) {
    var box = el('div', 'tv');
    var tabs = el('div', 'tv-tabs');
    var cells = el('div', 'tv-cells');
    var why = el('p', 'tv-why');
    box.appendChild(tabs); box.appendChild(cells); box.appendChild(why);
    var buttons = [];
    function show(k) {
      var c = TV[k];
      buttons.forEach(function (b, j) { b.classList.toggle('on', j === k); });
      cells.innerHTML = '';
      [['T', c[1], 'the concrete type'], ['V', c[2], 'a Value, maybe an Object'], ['N', c[3], 'eight inline bytes']].forEach(function (t) {
        var cell = el('div', 'tv-cell' + (t[1] === 'nil' || t[1] === '0' ? ' idle' : ''));
        cell.appendChild(el('div', 'tv-k', t[0]));
        cell.appendChild(el('div', 'tv-v', t[1]));
        cell.appendChild(el('div', 'tv-d', t[2]));
        cells.appendChild(cell);
      });
      why.textContent = c[4];
    }
    TV.forEach(function (c, k) {
      var b = el('button', 'tv-tab', c[0]);
      b.addEventListener('click', function () { show(k); });
      tabs.appendChild(b);
      buttons.push(b);
    });
    var svg = fig.querySelector('.fig-svg');
    if (svg) svg.hidden = true;
    fig.insertBefore(box, fig.querySelector('figcaption'));
    show(0);
  }

  var WIDGETS = {
    'machine-trace': function (fig) { stepper(fig, 'machine-trace'); },
    'call-sequence': function (fig) { stepper(fig, 'call-sequence'); },
    'finalization': function (fig) { stepper(fig, 'finalization'); },
    'typed-value': typedValueTabs
  };

  function mountOne(img) {
    var src = img.getAttribute('src');
    var name = src.replace(/^figures\//, '').replace(/\.svg$/, '');
    var fig = el('figure', 'fig');
    fig.setAttribute('data-fig', name);
    var cap = el('figcaption', null, img.getAttribute('alt') || '');
    var p = img.parentNode;
    var holder = el('div', 'fig-svg');
    fig.appendChild(holder);
    fig.appendChild(cap);
    p.parentNode.replaceChild(fig, p);
    return fetch(src, { cache: 'no-cache' })
      .then(function (r) { if (!r.ok) throw new Error(r.status); return r.text(); })
      .then(function (svg) {
        holder.innerHTML = svg;
        var s = holder.querySelector('svg');
        if (s) { s.removeAttribute('width'); s.removeAttribute('height'); }
        notes(fig);
        if (WIDGETS[name]) WIDGETS[name](fig);
      })
      .catch(function () { holder.appendChild(img); });
  }

  window.GnoFigures = {
    mount: function (article) {
      var imgs = Array.prototype.slice.call(article.querySelectorAll('img[src^="figures/"]'));
      return Promise.all(imgs.map(mountOne));
    }
  };
})();
