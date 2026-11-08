// Bridge between the Go game (WASM) and the browser: storage, PocketBase auth,
// and an offline-tolerant queue for run/label events. The game only ever calls
// window.haphazard.{load,save,emit}.
(() => {
  // config.js decides where the backend is. No config means the page's own origin
  // (the bundled PocketBase server); an empty api means demo mode with no backend.
  const cfg = window.HM_CONFIG;
  const API = cfg ? cfg.api : location.origin;
  const DEMO = !API;
  const LS = {
    get(k) { try { return localStorage.getItem(k); } catch { return null; } },
    set(k, v) { try { localStorage.setItem(k, v); } catch {} },
    del(k) { try { localStorage.removeItem(k); } catch {} },
  };

  let auth = null;
  try { auth = JSON.parse(LS.get('hm.auth') || 'null'); } catch {}
  let queue = [];
  try { queue = JSON.parse(LS.get('hm.queue') || '[]'); } catch {}
  const saveQueue = () => LS.set('hm.queue', JSON.stringify(queue.slice(-500)));

  async function pb(path, opts = {}) {
    const headers = { 'Content-Type': 'application/json' };
    if (auth && auth.token) headers.Authorization = auth.token;
    const res = await fetch(API + path, { ...opts, headers });
    const body = await res.json().catch(() => ({}));
    if (!res.ok) { const e = new Error(body.message || res.statusText); e.status = res.status; e.body = body; throw e; }
    return body;
  }

  const toRecord = (ev, p) => {
    if (ev === 'run') return ['hm_runs', { level: p.level, won: p.won, stars: p.stars, time: p.time, samples: p.samples, score: p.score, cost: p.cost, design: p.design }];
    if (ev === 'label') return ['hm_labels', { level: p.level, segment: p.segment, x0: p.x0, x1: p.x1, class: p.class, gold: p.gold, correct: p.correct }];
    return null;
  };

  let flushing = false;
  async function flush() {
    if (flushing || !auth || !queue.length) return;
    flushing = true;
    try {
      while (queue.length) {
        const [col, data] = queue[0];
        try {
          await pb(`/api/collections/${col}/records`, { method: 'POST', body: JSON.stringify({ ...data, user: auth.record.id }) });
        } catch (e) {
          if (e.status === 400) { /* invalid or duplicate: drop it */ }
          else throw e; // offline / auth expired: keep for later
        }
        queue.shift(); saveQueue();
      }
    } catch (e) {
      if (e.status === 401 || e.status === 403) signOut();
    } finally { flushing = false; }
  }

  // ---- account UI ----
  const $ = (id) => document.getElementById(id);
  const btn = $('acct'), dlg = $('dlg'), msg = $('msg');
  function render() { if (DEMO) { btn.hidden = true; return; } btn.textContent = auth ? auth.record.email + ' · sign out' : 'Sign in'; }
  function signOut() { auth = null; LS.del('hm.auth'); render(); }
  async function setAuth(r) { auth = { token: r.token, record: r.record }; LS.set('hm.auth', JSON.stringify(auth)); render(); flush(); }
  async function login(email, password) {
    await setAuth(await pb('/api/collections/users/auth-with-password', { method: 'POST', body: JSON.stringify({ identity: email, password }) }));
  }
  btn.addEventListener('click', () => { if (auth) signOut(); else { msg.textContent = ''; dlg.showModal(); } });
  $('close').addEventListener('click', () => dlg.close());
  $('form').addEventListener('submit', async (e) => {
    e.preventDefault();
    try { await login($('email').value, $('pw').value); dlg.close(); } catch (err) { msg.textContent = err.message; }
  });
  $('signup').addEventListener('click', async () => {
    const email = $('email').value, password = $('pw').value;
    if (!email || password.length < 8) { msg.textContent = 'Enter an email and a password of 8+ characters.'; return; }
    try {
      await pb('/api/collections/users/records', { method: 'POST', body: JSON.stringify({ email, password, passwordConfirm: password }) });
      await login(email, password); dlg.close();
    } catch (err) { msg.textContent = err.message; }
  });
  render();
  if (auth && !DEMO) pb('/api/collections/users/auth-refresh', { method: 'POST' }).then(setAuth).catch((e) => { if (e.status === 401 || e.status === 403) signOut(); });
  addEventListener('online', flush);

  window.haphazard = {
    load: (k) => LS.get('hm.' + k),
    save: (k, v) => LS.set('hm.' + k, v),
    emit: (ev, json) => {
      let p; try { p = JSON.parse(json); } catch { return; }
      if (ev === 'scene') { btn.hidden = DEMO || p.name !== 'menu'; return; } // keep the HTML chrome off the in-game HUD
      if (DEMO) return; // nothing to send to: progress is already saved locally
      const rec = toRecord(ev, p);
      if (!rec) return;
      queue.push(rec); saveQueue(); flush();
    },
  };
})();
