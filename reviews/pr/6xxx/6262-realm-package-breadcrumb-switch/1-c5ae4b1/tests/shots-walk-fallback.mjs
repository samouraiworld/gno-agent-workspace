// gnolang/gno#6262: click the kind-switch on /r/alice/golf/game with the bundle as shipped (8799) and with anchor( renamed (8798).
export default async (page, { tap }) => {
  const out = process.env.SHOTS;
  for (const [port, name] of [[8799, 'shipped'], [8798, 'noanchor']]) {
    await page.goto(`http://127.0.0.1:${port}/r/alice/golf/game`, { waitUntil: 'networkidle' });
    await tap('.kind-switch');
    const r = await page.evaluate(() => {
      const m = document.getElementById('kind-switch-menu'), b = m.getBoundingClientRect(), c = getComputedStyle(m);
      return `open=${m.matches(':popover-open')} menu top/left px: ${Math.round(b.top)}/${Math.round(b.left)} computed ${c.top}/${c.left}`;
    });
    console.error(name, r);
    await page.screenshot({ path: `${out}/chromium-${name}.png` });
  }
};
