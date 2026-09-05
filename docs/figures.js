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
      intro: 'f(2, 3) is running. Blocks: f’s block holds a=2, b=3, .res.0 and x.',
      stacks: ['Ops', 'Exprs', 'Stmts', 'Values'],
      steps: [
        { title: 'Before the statement', gas: 0,
          note: 'OpExec is waiting for the statement. Nothing has been read yet.',
          Ops: ['OpExec'], Exprs: [], Stmts: ['x := a + b'], Values: [] },
        { title: 'doOpExec, gas 130', gas: 130,
          note: 'The statement is peeked, not popped. For a define, OpDefine goes on first, then OpEval for the right-hand side, so the eval runs first.',
          Ops: ['OpEval', 'OpDefine'], Exprs: ['a + b'], Stmts: ['x := a + b'], Values: [] },
        { title: 'doOpEval on a + b, gas 82', gas: 212,
          note: 'A binary expression pushes the operator, then an eval of the right side, then an eval of the left side. The expression itself stays under its operands.',
          Ops: ['OpEval', 'OpEval', 'OpAdd', 'OpDefine'], Exprs: ['a', 'b', 'a + b'], Stmts: ['x := a + b'], Values: [] },
        { title: 'doOpEval on a, gas 82 + 4', gas: 298,
          note: 'The name fast path: pop the expression, climb Depth − 1 blocks, read slot Index, push the value. The 4 is the per-hop slope for a name at depth 1.',
          Ops: ['OpEval', 'OpAdd', 'OpDefine'], Exprs: ['b', 'a + b'], Stmts: ['x := a + b'], Values: ['2'] },
        { title: 'doOpEval on b, gas 82 + 4', gas: 384,
          note: 'Same path for b. Both operands are now on the values stack, a below b.',
          Ops: ['OpAdd', 'OpDefine'], Exprs: ['a + b'], Stmts: ['x := a + b'], Values: ['3', '2'] },
        { title: 'doOpAdd, gas 81 for ints', gas: 465,
          note: 'Pops the expression and the right operand, adds into the left operand in place with a per-kind switch. Floats would go through software floating point.',
          Ops: ['OpDefine'], Exprs: [], Stmts: ['x := a + b'], Values: ['5'] },
        { title: 'doOpDefine, gas 114 + 79', gas: 658,
          note: 'Pops the statement and the result, writes 5 into slot 3 of f’s block, or into the heap item in that slot if x were captured. Total for the statement: 658 gas.',
          Ops: [], Exprs: [], Stmts: [], Values: [] }
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
          note: 'Peeks the callee under the arguments. A function value: push a frame that records the stack heights, the callee, the package and realm to restore, then OpCall. A crossing function adds OpEnterCrossing; cross(rlm) installs a fresh cur on the frame.',
          Ops: ['OpCall', 'OpDefine'], Values: ['b', 'a', 'f'], Frames: ['f: heights, package, realm', 'caller'] },
        { title: '3 · OpCall', hl: ['call-3'],
          note: 'A new block from f’s source node and parent block; captured heap items copied in; a and b popped into slots 0 and 1; .res.0 zeroed. OpBody carries the statements, plus a synthetic return if f declared no results.',
          Ops: ['OpBody', 'OpDefine'], Values: [], Frames: ['f: heights, package, realm', 'caller'] },
        { title: '4 · body runs', hl: ['call-4'],
          note: 'OpBody executes the statements one by one. A defer evaluates its function and arguments now and pushes a Defer onto the frame. A native function would have pushed OpReturn and OpCallNativeBody instead.',
          Ops: ['OpBody', 'OpDefine'], Values: ['(result)'], Frames: ['f: defers []', 'caller'] },
        { title: '5 · OpReturn', hl: ['call-5'],
          note: 'One of four flavors, chosen by the preprocessor and the frame: OpReturn, OpReturnAfterCopy, OpReturnFromBlock, or OpReturnToBlock followed by the sticky OpReturnCallDefers when the frame holds defers.',
          Ops: ['OpReturn', 'OpDefine'], Values: ['(result)'], Frames: ['f', 'caller'] },
        { title: '6 · frame popped', hl: ['call-6'],
          note: 'Every stack is truncated to the heights the frame recorded, the result moves down to where the callee was, and the caller’s Package and Realm are restored. If the frame crossed a realm boundary, maybeFinalize finalizes the realm now. OpDefine then stores the result in y.',
          Ops: ['OpDefine'], Values: ['(result)'], Frames: ['caller'] }
      ]
    },
    'finalization': {
      intro: 'zrealm1.gno assigns InnerNode{Key: "somekey"} to the package variable root. Objects: :1 the package, :2 its block, :3 the heap item behind root.',
      stacks: [],
      steps: [
        { title: 'During the call: Assign2', hl: ['obj-3', 'obj-6'],
          note: 'The write into the heap item goes through PointerValue.Assign2, which calls Realm.DidUpdate with the parent :3, no old child, and the new struct. :3 is marked dirty; the struct is marked new-real and stamped with the realm’s PkgID, NewTime still 0.' },
        { title: '1 · processNewCreatedMarks', hl: ['obj-6'],
          note: 'Every object marked new-real gets an id depth-first: the struct becomes :6. Its children are counted; the string inside is not an object, so nothing further becomes real.' },
        { title: '2 · processNewDeletedMarks', hl: [],
          note: 'Objects whose reference count hit zero are walked and their children decremented, recursively. Nothing was dropped here: root held {} before.' },
        { title: '3 · processNewEscapedMarks', hl: ['obj-2'],
          note: 'Objects that ended the transaction with one reference are demoted, the rest confirmed as escaped. :2, the package block, stays escaped with two references.' },
        { title: '4 · realm record', hl: [],
          note: 'The realm’s counter moved, from 5 to 6, so its record under #realm is saved.' },
        { title: '5 · markDirtyAncestors', hl: ['obj-3', 'obj-2'],
          note: 'From every changed object up to the package block, using the persisted owner id: :6 has owner :3, :3 has owner :2. Parent hashes will be refreshed.' },
        { title: '6 · saveUnsavedObjects', hl: ['writes', 'obj-6', 'obj-3', 'obj-2'],
          note: 'New and dirty objects are written, children first, after checking that nothing references a private package’s types. The Realm: directive prints exactly this: c[:6], then u[:3], then u[:2].' },
        { title: '7 · removeDeletedObjects', hl: [],
          note: 'The dead ones are deleted from the store. None this time.' },
        { title: '8 · byte deltas', hl: ['writes'],
          note: 'The size of each write minus the last saved size, summed per realm into the store’s storage-diff map. After the message, processStorageDeposit locks delta × 100 ugnot per byte from the caller into the realm’s deposit address.' }
      ]
    }
  };

  // ---------------------------------------------------------------- typed value tabs

  var TV = [
    ['x := 42', 'int', 'nil', '42', 'SetInt writes an int64 into the eight bytes. No allocation, and the allocator never sees it.'],
    ['ok := true', 'bool', 'nil', '01 00 00 00 00 00 00 00', 'SetBool writes the bool into the first byte of N.'],
    ['f := 2.5', 'float64', 'nil', '0x4004000000000000', 'SetFloat64 stores the IEEE-754 bits. Arithmetic runs in software so every node computes the same bits.'],
    ['s := "hi"', 'string', 'StringValue("hi")', '0', 'A Go string in V. Not an Object: no identity, saved inside its holder.'],
    ['p := &node', '*InnerNode', 'PointerValue{TV, Base, Index}', '0', 'A slot address: the base object, an array, struct, block or heap item, plus an index into it. Index −1 is a block’s blank slot, −2 a map entry reached through Key.'],
    ['xs := []int{1, 2}', '[]int', 'SliceValue{Base, Offset: 0, Length: 2, Maxcap: 2}', '0', 'A window onto an ArrayValue, which is the Object with identity. Copying the slice copies the window.'],
    ['v := InnerNode{...}', 'gno.land/r/test.InnerNode', '*StructValue{ObjectInfo, Fields: [3]TypedValue}', '0', 'An Object: ObjectInfo carries id, owner, reference count and the dirty marks. Copying a struct copies its fields into a new object.'],
    ['var e any', 'nil', 'nil', '0', 'The undefined typed value: no type, no value. In the zrealm1 trace the nil Left and Right fields serialize as {}.'],
    ['lib.Data, read from another realm', 'its type', 'the object', '"ReaDoNLY"', 'The magic bytes mark a composite read from another realm’s storage. The mark survives copies and arguments; a write through it panics with "cannot directly modify readonly tainted object".']
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
