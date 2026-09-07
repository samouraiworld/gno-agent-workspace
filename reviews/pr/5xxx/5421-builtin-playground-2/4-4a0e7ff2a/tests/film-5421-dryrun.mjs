// gno #5421, head 4a0e7ff2a: the Run view's Dry Run button, filmed.
//
// Asserts that no value of the "Key name or address" field makes Dry Run
// succeed at this head: a key name is refused by the endpoint as non-bech32,
// and a funded address that has already signed is refused by the node's auth
// ante as unauthorized. The ledger samples the real POST body and the real
// HTTP status, not the rendered text alone.
//
// Run: PLAYWRIGHT_DRIVER=<path to playwright-core/index.mjs> \
//      CHROMIUM=<path to a chromium binary> node film-5421-dryrun.mjs <out-dir>
// Needs gnodev built from this head and serving gnoweb on 127.0.0.1:8888, and
// the address below funded with one broadcast transaction so it carries a
// public key on chain.

const DRIVER = process.env.PLAYWRIGHT_DRIVER || "playwright-core";
const { chromium } = await import(DRIVER);
const fs = (await import("node:fs")).default;

const CHROMIUM = process.env.CHROMIUM;
const URL = "http://127.0.0.1:8888/r/gnoland/home$run";
const ADDRESS = "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5";
const OUT = process.argv[2];
fs.mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch({ executablePath: CHROMIUM, headless: true });
const ctx = await browser.newContext({
  viewport: { width: 1280, height: 800 },
  recordVideo: { dir: OUT, size: { width: 1280, height: 800 } },
});
const page = await ctx.newPage();

try {
  await page.goto(URL, { waitUntil: "networkidle" });
  await page.waitForSelector("#run-key");

  // Overlay: a pointer ring, a caption naming the input, and a ledger whose
  // rows are appended by the patched fetch, so every cell is a value the page
  // itself sent or received.
  await page.evaluate(() => {
    const mk = (css, parent) => {
      const d = document.createElement("div");
      d.style.cssText = css;
      (parent || document.body).append(d);
      return d;
    };
    const dot = mk(
      "position:fixed;z-index:99999;width:26px;height:26px;margin:-13px 0 0 -13px;" +
        "border:3px solid #ffd400;border-radius:50%;box-shadow:0 0 0 2px #000,0 0 12px #000;" +
        "pointer-events:none;left:-100px;top:-100px",
    );
    addEventListener("mousemove", (e) => {
      dot.style.left = `${e.clientX}px`;
      dot.style.top = `${e.clientY}px`;
    });

    const panel = mk(
      "position:fixed;z-index:99998;left:12px;top:12px;right:12px;background:#000e;" +
        "color:#fff;padding:9px 13px;border-radius:10px;pointer-events:none;" +
        "font:600 14px/1.4 ui-monospace,monospace",
    );
    const cap = mk("color:#ffd400;margin-bottom:6px", panel);
    cap.textContent =
      "POST /_/api/dryrun — Key field, address sent, HTTP status, result on screen";
    const table = document.createElement("table");
    table.style.cssText = "border-collapse:collapse;width:100%";
    table.innerHTML =
      "<tr style='color:#8b949e'>" +
      "<th style='text-align:left;padding:2px 10px 2px 0'>Key field</th>" +
      "<th style='text-align:left;padding:2px 10px 2px 0'>address in POST body</th>" +
      "<th style='text-align:left;padding:2px 10px 2px 0'>HTTP</th>" +
      "<th style='text-align:left;padding:2px 0'>Result pane</th></tr>";
    panel.append(table);

    window.__cap = (t) => (cap.textContent = t);
    window.__rows = [];
    window.__addRow = (r) => {
      window.__rows.push(r);
      const tr = document.createElement("tr");
      const ok = r.status === 200 && !r.result.startsWith("Error:");
      tr.innerHTML =
        `<td style='padding:2px 10px 2px 0'>${r.field}</td>` +
        `<td style='padding:2px 10px 2px 0'>${r.sent}</td>` +
        `<td style='padding:2px 10px 2px 0;color:${r.status === 200 ? "#7ee787" : "#ff7b72"}'>${r.status}</td>` +
        `<td style='padding:2px 0;color:${ok ? "#7ee787" : "#ff7b72"}'>${r.result}</td>`;
      table.append(tr);
    };

    // Patch fetch so the ledger reads the request the page actually sent and
    // the status the server actually returned.
    const real = window.fetch;
    window.fetch = async (input, init) => {
      const url = typeof input === "string" ? input : input.url;
      const res = await real(input, init);
      if (url.includes("/_/api/dryrun")) {
        window.__pending = {
          sent: JSON.parse(init.body).address,
          status: res.status,
        };
      }
      return res;
    };
  });

  const cap = (t) => page.evaluate((s) => window.__cap(s), t);
  const key = page.locator("#run-key");
  const dryRun = page.getByRole("button", { name: "Dry Run" });

  // One dry run: type the field value, click, wait for the result pane to
  // settle, then push a ledger row built from the patched fetch.
  const shot = async (field, caption) => {
    await cap(caption);
    await key.click();
    await key.fill("");
    await key.pressSequentially(field, { delay: 55 });
    await page.waitForTimeout(700);
    await dryRun.hover();
    await page.waitForTimeout(350);
    await dryRun.click();
    await page.waitForFunction(
      () =>
        window.__pending !== undefined &&
        !document
          .querySelector("[data-run-target='result']")
          .textContent.startsWith("Running"),
      { timeout: 30000 },
    );
    await page.evaluate((f) => {
      const p = window.__pending;
      window.__pending = undefined;
      window.__addRow({
        field: f,
        sent: p.sent,
        status: p.status,
        result: document.querySelector("[data-run-target='result']").textContent,
      });
    }, field);
    // Park the Key field and the Result pane in the same frame, so the value
    // typed and the value returned are readable together.
    await key.scrollIntoViewIfNeeded();
    await page.mouse.wheel(0, -150);
    await page.waitForTimeout(2400);
  };

  await page.mouse.move(640, 400, { steps: 10 });
  await page.waitForTimeout(1200);

  await shot("mykey", "the field's own placeholder, typed as the label's first option");
  await shot(ADDRESS, "a funded address that has already signed a transaction");

  await cap("no value of this field makes Dry Run succeed at this head");
  await page.waitForTimeout(3000);

  const rows = await page.evaluate(() => window.__rows);
  await ctx.close();
  await browser.close();
  const file = fs.readdirSync(OUT).find((f) => f.endsWith(".webm"));
  console.log(JSON.stringify({ video: `${OUT}/${file}`, rows }, null, 2));
} catch (err) {
  await ctx.close().catch(() => {});
  await browser.close().catch(() => {});
  throw err;
}
