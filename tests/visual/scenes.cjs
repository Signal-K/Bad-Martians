// Browser-rendered scene smoke tests and screenshot gallery (KES-465/466/467/470).
// Run: make visual-test. Screenshots are written to test-results/visual/.
const { chromium } = require('playwright');
const { spawn } = require('node:child_process');
const { mkdir, writeFile } = require('node:fs/promises');
const assert = require('node:assert/strict');
const path = require('node:path');

const scenes = ['menu', 'build', 'launch', 'run', 'result', 'briefing', 'style'];
const sizes = [{ name: 'desktop', width: 960, height: 540 }, { name: 'phone', width: 390, height: 844 }];
const output = path.resolve('test-results/visual');
const base = 'http://127.0.0.1:18764';

(async () => {
  await mkdir(output, { recursive: true });
  const server = spawn('python3', ['-m', 'http.server', '18764', '--bind', '127.0.0.1', '--directory', 'web'], { stdio: 'ignore' });
  let browser;
  try {
    for (let i = 0; i < 100; i++) {
      try { if ((await fetch(base)).ok) break; } catch (_) {}
      if (i === 99) throw new Error('Static server did not start');
      await new Promise(r => setTimeout(r, 100));
    }
    browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', args: ['--no-sandbox', '--use-gl=angle', '--use-angle=swiftshader'] });
    const images = [];
    for (const size of sizes) {
      const context = await browser.newContext({ viewport: { width: size.width, height: size.height }, deviceScaleFactor: 1, serviceWorkers: 'block' });
      const page = await context.newPage();
      const errors = [];
      page.on('pageerror', e => errors.push(e.message));
      await page.addInitScript(() => {
        let bridge;
        window.__sceneEvents = [];
        Object.defineProperty(window, 'haphazard', {
          configurable: true,
          get: () => bridge,
          set: value => {
            bridge = value;
            const emit = value.emit;
            value.emit = (event, payload) => {
              if (event === 'scene') window.__sceneEvents.push(JSON.parse(payload).name);
              return emit(event, payload);
            };
          },
        });
      });
      for (const scene of scenes) {
        await page.goto(`${base}/?scene=${scene}`, { waitUntil: 'load' });
        await page.waitForFunction(() => window.haphazard && typeof window.haphazard.scenes === 'function' && !document.querySelector('#boot'), null, { timeout: 30000 });
        await page.waitForSelector('canvas', { state: 'visible' });
        await page.waitForFunction(name => window.__sceneEvents.includes(name), scene, { timeout: 30000 });
        await page.waitForTimeout(700); // allow the first WebGL frames to be presented
        assert.equal(await page.title(), 'Haphazard Martians');
        const registered = JSON.parse(await page.evaluate(() => window.haphazard.scenes()));
        assert.deepEqual(registered.map(s => s.Name).sort(), [...scenes].sort());
        assert.ok(registered.every(s => s.Title && s.Type && s.Layout && Object.keys(s.Regions).length));
        const bounds = await page.locator('canvas').boundingBox();
        assert.ok(bounds && bounds.width > 0 && bounds.height > 0 && bounds.x >= 0 && bounds.y >= 0 && bounds.x + bounds.width <= size.width + 1 && bounds.y + bounds.height <= size.height + 1, `${size.name}/${scene}: canvas outside viewport`);
        const filename = `${size.name}-${scene}.png`;
        const png = await page.screenshot({ path: path.join(output, filename) });
        assert.ok(png.length > 5000, `${size.name}/${scene}: blank or incomplete screenshot`);
        images.push({ size: size.name, scene, filename, png });
        assert.deepEqual(errors, [], `${size.name}/${scene}: browser errors`);
        console.log(`PASS ${size.name}/${scene} (${png.length} bytes)`);
      }
      await context.close();
    }
    for (const size of sizes) {
      const group = images.filter(i => i.size === size.name);
      for (let i = 1; i < group.length; i++) assert.notDeepEqual(group[i].png, group[0].png, `${group[i].scene} looks identical to menu`);
    }
    const cards = images.map(i => `<figure><a href="../../web/?scene=${i.scene}"><img src="${i.filename}" alt="${i.size} ${i.scene}"></a><figcaption>${i.size}: ${i.scene} — PASS</figcaption></figure>`).join('\n');
    await writeFile(path.join(output, 'index.html'), `<!doctype html><meta charset="utf-8"><title>Scene visual test gallery</title><style>body{background:#201719;color:white;font:16px sans-serif}main{display:flex;flex-wrap:wrap;gap:1rem}figure{margin:0;width:360px}img{width:100%}</style><h1>Scene visual test gallery</h1><p>Click a screenshot to open the scene when served from the repository root.</p><main>${cards}</main>`);
  } finally {
    if (browser) await browser.close();
    server.kill();
  }
})().catch(e => { console.error(e); process.exitCode = 1; });
