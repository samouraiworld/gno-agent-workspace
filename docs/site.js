// Renders one of the markdown files in this folder into the page, builds the
// table of contents from its headings, and keeps the active heading marked.
// The markdown stays the source of truth: nothing here is copied from it.
(function () {
  'use strict';

  var DOCS = {
    'gnovm-architecture.md': 'How the GnoVM works',
    'overview.md': 'gno.land overview',
    'glossary.md': 'Gno glossary'
  };
  var SOURCE = 'https://github.com/samouraiworld/gno-agent-workspace/blob/main/docs/';

  var article = document.getElementById('article');
  var tocNav = document.getElementById('toc-nav');
  var menu = document.getElementById('menu');
  var backdrop = document.getElementById('backdrop');

  function docName() {
    var q = new URLSearchParams(location.search).get('doc');
    return DOCS[q] ? q : 'gnovm-architecture.md';
  }

  function slugify(text) {
    return text.toLowerCase()
      .replace(/[`*_]/g, '')
      .replace(/[^\w\s-]/g, '')
      .trim()
      .replace(/\s+/g, '-');
  }

  function uniqueId(base, used) {
    var id = base || 'section', n = 1;
    while (used[id]) { n += 1; id = base + '-' + n; }
    used[id] = true;
    return id;
  }

  function decorate(name) {
    var used = {};
    var headings = article.querySelectorAll('h1, h2, h3, h4');
    var h1 = article.querySelector('h1');
    if (h1) document.title = h1.textContent + ' · GnoVM';

    Array.prototype.forEach.call(headings, function (h) {
      if (h.tagName === 'H1') return;
      var id = uniqueId(slugify(h.textContent), used);
      h.id = id;
      var a = document.createElement('a');
      a.className = 'anchor';
      a.href = '#' + id;
      a.setAttribute('aria-label', 'Link to this section');
      a.textContent = '#';
      h.insertBefore(a, h.firstChild);
    });

    // Tables scroll inside their own box on narrow screens.
    Array.prototype.forEach.call(article.querySelectorAll('table'), function (t) {
      var wrap = document.createElement('div');
      wrap.className = 'table-wrap';
      t.parentNode.insertBefore(wrap, t);
      wrap.appendChild(t);
    });

    // The TLDR section of the main document reads as a callout.
    var tldr = article.querySelector('h2#tldr');
    if (tldr) {
      var box = document.createElement('div');
      box.className = 'tldr-box';
      var node = tldr.nextElementSibling;
      tldr.parentNode.insertBefore(box, tldr);
      box.appendChild(tldr);
      while (node && node.tagName === 'P') {
        var next = node.nextElementSibling;
        box.appendChild(node);
        node = next;
      }
    }
    var measured = h1 && h1.nextElementSibling;
    if (measured && measured.tagName === 'P' && /^Measured against/.test(measured.textContent)) {
      measured.className = 'measured';
    }

    // Documents nav and source link.
    Array.prototype.forEach.call(document.querySelectorAll('.docs a'), function (a) {
      a.classList.toggle('active', a.getAttribute('data-doc') === name);
    });
    var src = document.getElementById('source-link');
    if (src) src.href = SOURCE + name;
  }

  function buildToc() {
    var headings = article.querySelectorAll('h2, h3');
    if (!headings.length) { tocNav.innerHTML = ''; return; }
    var root = document.createElement('ul');
    var currentList = root, currentItem = null;
    Array.prototype.forEach.call(headings, function (h) {
      var li = document.createElement('li');
      var a = document.createElement('a');
      a.href = '#' + h.id;
      a.textContent = h.textContent.replace(/^#\s*/, '');
      a.setAttribute('data-target', h.id);
      li.appendChild(a);
      if (h.tagName === 'H2') {
        root.appendChild(li);
        currentItem = li;
        currentList = null;
      } else {
        if (!currentList) {
          currentList = document.createElement('ul');
          (currentItem || root).appendChild(currentList);
        }
        currentList.appendChild(li);
      }
    });
    tocNav.innerHTML = '';
    tocNav.appendChild(root);
    tocNav.addEventListener('click', function (e) {
      if (e.target.tagName === 'A') closeToc();
    });
  }

  var observer = null;
  function trackActive() {
    if (observer) observer.disconnect();
    var links = {};
    Array.prototype.forEach.call(tocNav.querySelectorAll('a'), function (a) {
      links[a.getAttribute('data-target')] = a;
    });
    var headings = Array.prototype.slice.call(article.querySelectorAll('h2, h3'));
    if (!headings.length || !('IntersectionObserver' in window)) return;
    var visible = {};
    function setActive(id) {
      Object.keys(links).forEach(function (k) { links[k].classList.toggle('active', k === id); });
      var a = links[id];
      if (a && a.scrollIntoView) {
        var toc = document.getElementById('toc');
        var r = a.getBoundingClientRect(), t = toc.getBoundingClientRect();
        if (r.top < t.top || r.bottom > t.bottom) a.scrollIntoView({ block: 'nearest' });
      }
    }
    observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) { visible[en.target.id] = en.isIntersecting; });
      var first = null;
      for (var i = 0; i < headings.length; i++) {
        if (visible[headings[i].id]) { first = headings[i]; break; }
      }
      if (!first) {
        // Nothing in view: the last heading above the top edge is current.
        for (var j = headings.length - 1; j >= 0; j--) {
          if (headings[j].getBoundingClientRect().top < 80) { first = headings[j]; break; }
        }
      }
      if (first) setActive(first.id);
    }, { rootMargin: '-56px 0px -70% 0px', threshold: 0 });
    headings.forEach(function (h) { observer.observe(h); });
  }

  function openToc() { document.body.classList.add('toc-open'); menu.setAttribute('aria-expanded', 'true'); }
  function closeToc() { document.body.classList.remove('toc-open'); menu.setAttribute('aria-expanded', 'false'); }
  menu.addEventListener('click', function () {
    if (document.body.classList.contains('toc-open')) closeToc(); else openToc();
  });
  backdrop.addEventListener('click', closeToc);
  document.addEventListener('keydown', function (e) { if (e.key === 'Escape') closeToc(); });

  function jumpToHash() {
    if (!location.hash) return;
    var el = document.getElementById(decodeURIComponent(location.hash.slice(1)));
    if (!el) return;
    // The page just loaded: land on the section, do not animate 7000px.
    var root = document.documentElement;
    root.style.scrollBehavior = 'auto';
    el.scrollIntoView();
    root.style.scrollBehavior = '';
  }

  function render(name) {
    fetch(name, { cache: 'no-cache' })
      .then(function (r) {
        if (!r.ok) throw new Error(r.status + ' ' + r.statusText);
        return r.text();
      })
      .then(function (md) {
        article.innerHTML = window.marked.parse(md, { gfm: true, breaks: false });
        decorate(name);
        buildToc();
        trackActive();
        jumpToHash();
      })
      .catch(function (err) {
        article.innerHTML = '<h1>Could not load ' + name + '</h1><p>' + err.message +
          '. Read it <a href="' + SOURCE + name + '">on GitHub</a> instead.</p>';
      });
  }

  render(docName());
})();
