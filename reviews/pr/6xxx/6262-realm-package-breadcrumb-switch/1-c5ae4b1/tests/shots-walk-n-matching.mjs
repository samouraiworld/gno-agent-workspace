// gnolang/gno#6262: open the kind-switch menu on /p/tests/vm/crossrealm, then follow its "2 matching realms" link.
export default async (page, { tap, wait, url }) => {
  const out = process.env.SHOTS;
  await page.goto(url + '/p/tests/vm/crossrealm', { waitUntil: 'networkidle' });
  await tap('.kind-switch');
  const m = await page.locator('#kind-switch-menu').boundingBox();
  const b = await page.locator('.kind-switch').boundingBox();
  console.error('menu box', JSON.stringify(m), 'button box', JSON.stringify(b));
  console.error('primary', await page.locator('#kind-switch-menu .item--primary').evaluate(e => e.getAttribute('href') + ' | ' + e.innerText.replace(/\n/g, ' / ')));
  await page.screenshot({ path: out + '/dir-menu.png' });
  await tap('#kind-switch-menu .item--primary', 1200);
  console.error('landed', page.url());
  console.error('mentions crossrealm/subtests:', (await page.content()).includes('/r/tests/vm/crossrealm'), (await page.content()).includes('/r/tests/vm/subtests'));
  await page.screenshot({ path: out + '/dir-landed.png' });
};
