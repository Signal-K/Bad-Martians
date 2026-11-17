// Offline shell: cache-first for the game's static files (refreshed in the
// background), network-only for the PocketBase API.
const CACHE = 'haphazard-martians-v1'; // scripts/build-dist.sh stamps the build id here
// No 'index.html': Cloudflare's static assets 307 it to '/', and './' is the same page.
const SHELL = ['./', 'config.js', 'host.js', 'wasm_exec.js', 'game.wasm', 'manifest.webmanifest', 'icons/icon-192.png', 'icons/icon-512.png'];

// A navigation must never be answered with a response that followed a redirect:
// WebKit (every iOS browser and installed PWA) refuses it with "Response served
// by service worker has redirections" and the page stays blank. Cached copies
// are rebuilt without the flag; a live redirect goes back to the browser as one.
const unredirect = (r) => (r && r.redirected ? new Response(r.body, { status: r.status, statusText: r.statusText, headers: r.headers }) : r);

self.addEventListener('install', (e) => {
  e.waitUntil(caches.open(CACHE).then((c) => Promise.all(SHELL.map(async (f) => {
    const r = await fetch(f, { cache: 'reload' });
    if (!r.ok) throw new Error(f + ' ' + r.status);
    await c.put(f, unredirect(r));
  }))).then(() => self.skipWaiting()));
});
self.addEventListener('activate', (e) => {
  e.waitUntil(caches.keys().then((ks) => Promise.all(ks.filter((k) => k !== CACHE).map((k) => caches.delete(k)))).then(() => self.clients.claim()));
});
self.addEventListener('fetch', (e) => {
  const url = new URL(e.request.url);
  if (e.request.method !== 'GET' || url.origin !== location.origin || url.pathname.startsWith('/api/') || url.pathname.startsWith('/_/')) return;
  const nav = e.request.mode === 'navigate';
  e.respondWith(caches.open(CACHE).then(async (c) => {
    // Every navigation in scope is the one game page ('./'), whatever its path or query.
    const hit = unredirect((await c.match(e.request, { ignoreSearch: true })) || (nav ? await c.match('./') : undefined));
    const net = fetch(e.request).then((r) => {
      if (r.ok && !r.redirected) c.put(e.request, r.clone());
      return nav && r.redirected ? Response.redirect(r.url, 302) : r;
    }).catch(() => hit || new Response('', { status: 503, statusText: 'Offline' }));
    if (hit) { e.waitUntil(net.catch(() => {})); return hit; }
    return net;
  }));
});
