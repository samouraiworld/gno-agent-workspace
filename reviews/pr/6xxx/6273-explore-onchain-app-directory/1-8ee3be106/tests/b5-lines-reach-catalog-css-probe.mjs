// gnolang/gno#6273 at 8ee3be106: probe store.css on the real store home page.
// Input: the HTML written by b5-lines-reach-catalog-render_test.go (templates as
// shipped, styled by the committed build gno.land/pkg/gnoweb/public/main.css).
//
// Repro from a plain clone (after the render step in that file's header):
//
//	npm i playwright-core            # any recent version; uses a system Chromium
//	xvfb-run -a node b5-lines-reach-catalog-css-probe.mjs /tmp/b5-home.html /tmp/b5-shots
//
// Env: CHROMIUM (default /usr/sbin/chromium), PWCORE (path to playwright-core).
// Real input only: mouse.move and keyboard Tab go through CDP Input, so hover
// and :focus-visible are the browser's own, not dispatched events.
//
// P1 1280px: hover the .trust badge on a Spotlight card, the "community" label
//   and the truncated path on a shelf card, and the hero badge; report the hit and
//   whether the span carrying the title= is hovered.
// P2  800px: Tab to the first category pill, the first Spotlight card and a
//   shelf card; report each focus ring (the pill's own outline, the card's
//   on .b-store-card:has(h3 a:focus-visible)) against the scroll container
//   that clips it. The shelf grid is overflow visible: its row is the baseline.
// P3  375px: the pulse line on a quiet chain (every count 0).
import { createRequire } from 'node:module';
import fs from 'node:fs';
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PWCORE || 'playwright-core');

const [, , html, shots = '/tmp/b5-shots'] = process.argv;
fs.mkdirSync(shots, { recursive: true });
const browser = await chromium.launch({ headless: false, executablePath: process.env.CHROMIUM || '/usr/sbin/chromium' });
const page = await browser.newPage();
await page.route(/^(?!file:)/, (r) => r.abort());
const out = {};

async function open(w, h) {
  await page.setViewportSize({ width: w, height: h });
  await page.goto('file://' + html, { waitUntil: 'load' });
}

// P1
await open(1280, 900);
const hover = async (sel) => {
  await page.locator(sel).first().scrollIntoViewIfNeeded();
  const b = await page.locator(sel).first().boundingBox();
  const x = b.x + b.width / 2, y = b.y + b.height / 2;
  await page.mouse.move(x, y);
  return page.evaluate(([sel, x, y]) => {
    const el = document.querySelector(sel), hit = document.elementFromPoint(x, y);
    const t = hit.closest('[title]');
    return { spanHovered: el.matches(':hover'), hit: hit.tagName + (hit.className ? '.' + hit.className : ''), tooltipFrom: t ? t.getAttribute('title') : null };
  }, [sel, x, y]);
};
const shelfCard = '.b-store-shelf:not(.b-store-spotlight) .b-store-grid';
out.P1 = {
  spotlightCardTrust: await hover(`.b-store-grid--spotlight .b-store-card .trust`),
  shelfCommunity: await hover(`${shelfCard} .b-store-card .community`),
  shelfPath: await hover(`${shelfCard} .b-store-card .path .u-font-mono`),
  heroTrust: await hover('.b-store-hero .trust'),
};

// P2
await open(800, 900);
const ring = async (scope, name) => {
  for (let i = 0; i < 80; i++) {
    await page.keyboard.press('Tab');
    if (await page.evaluate((s) => !!document.activeElement.closest(s) && document.activeElement.matches('h3 a'), scope)) break;
  }
  const r = await page.evaluate((s) => {
    const a = document.activeElement, card = a.closest('.b-store-card'), ul = a.closest('ul');
    const cs = getComputedStyle(card), us = getComputedStyle(ul);
    const c = card.getBoundingClientRect(), u = ul.getBoundingClientRect();
    const reach = parseFloat(cs.outlineOffset) + parseFloat(cs.outlineWidth);
    return {
      focused: a.textContent, focusVisible: a.matches(':focus-visible'),
      outline: `${cs.outlineStyle} ${cs.outlineWidth} offset ${cs.outlineOffset}`,
      container: `overflow-x ${us.overflowX} overflow-y ${us.overflowY}`,
      ringOutside: { top: +(c.top - reach).toFixed(1), bottom: +(c.bottom + reach).toFixed(1), left: +(c.left - reach).toFixed(1) },
      clip: { top: +u.top.toFixed(1), bottom: +u.bottom.toFixed(1), left: +u.left.toFixed(1) },
      clipped: { top: c.top - reach < u.top, bottom: c.bottom + reach > u.bottom, left: c.left - reach < u.left },
    };
  }, scope);
  const box = await page.evaluate(() => { const u = document.activeElement.closest('ul').getBoundingClientRect(); return { x: Math.max(0, u.left - 16), y: Math.max(0, u.top - 16), width: Math.min(u.width + 32, 800), height: Math.min(u.height + 32, 600) }; });
  await page.screenshot({ path: `${shots}/p2-${name}.png`, clip: box });
  return r;
};
out.P2 = {};
for (let i = 0; i < 40; i++) {
  await page.keyboard.press('Tab');
  if (await page.evaluate(() => document.activeElement.matches('.b-store-nav a'))) break;
}
out.P2.categoryPill = await page.evaluate(() => {
  const a = document.activeElement, ul = a.closest('ul'), cs = getComputedStyle(a), us = getComputedStyle(ul);
  const c = a.getBoundingClientRect(), u = ul.getBoundingClientRect();
  const reach = parseFloat(cs.outlineOffset) + parseFloat(cs.outlineWidth);
  return {
    focused: a.textContent.trim(), focusVisible: a.matches(':focus-visible'),
    outline: `${cs.outlineStyle} ${cs.outlineWidth} offset ${cs.outlineOffset}`,
    container: `overflow-x ${us.overflowX} overflow-y ${us.overflowY} padding-top ${us.paddingTop}`,
    clipped: { top: c.top - reach < u.top, bottom: c.bottom + reach > u.bottom, left: c.left - reach < u.left },
  };
});
await page.screenshot({ path: `${shots}/p2-category-pill.png`, clip: await page.evaluate(() => { const u = document.activeElement.closest('ul').getBoundingClientRect(); return { x: Math.max(0, u.left - 16), y: Math.max(0, u.top - 16), width: 400, height: u.height + 32 }; }) });
out.P2.spotlight = await ring('.b-store-grid--spotlight', 'spotlight');
out.P2.shelf = await ring(shelfCard, 'shelf');

// P3
await open(375, 812);
out.P3 = await page.evaluate(() => {
  const p = document.querySelector('.b-store-pulse'), b = p.getBoundingClientRect();
  const kids = [...p.children].map((k) => `${k.className}:${getComputedStyle(k).display}`);
  const dot = getComputedStyle(p, '::before');
  return { visibleText: JSON.stringify(p.innerText.trim()), height: +b.height.toFixed(1), children: kids, dot: `${dot.content} ${dot.width}x${dot.height}` };
});
await page.locator('.b-store-intro').screenshot({ path: `${shots}/p3-pulse-phone.png` });

console.log(JSON.stringify(out, null, 1));
await browser.close();
