const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const { webcrypto } = require('node:crypto');
const source = fs.readFileSync('web/host.js', 'utf8');
const rows = [];
let offline = false;
function response(body, status = 200) { return { ok: status < 400, status, statusText: '', json: async () => body }; }
async function fetchMock(url, opts = {}) {
  if (offline) throw Error('offline');
  const auth = opts.headers.Authorization;
  const user = auth === 'token-a' ? 'a' : 'b';
  if (url.includes('auth-refresh')) return response({token: auth, record: {id: user, email: user}});
  if (url.includes('auth-with-password')) return response({token: 'token-b', record: {id: 'b', email: 'b'}});
  if (url.includes('/hm_saves/records')) {
    if (!opts.method) return response({items: rows.filter(r => r.user === user).map(r => structuredClone(r)), totalPages: 1});
    const data = JSON.parse(opts.body);
    assert.equal(data.user, user);
    if (opts.method === 'POST') rows.push({...structuredClone(data), id: String(rows.length + 1)});
    else Object.assign(rows.find(r => r.id === url.split('/').pop() && r.user === user), structuredClone(data));
    return response({});
  }
  return response({});
}
function browser(initial = {}) {
  const data = new Map(Object.entries(initial));
  const handlers = {};
  const elements = new Map();
  const element = id => {
    if (!elements.has(id)) elements.set(id, { value: '', hidden: false, textContent: '', classList: {remove(){}, add(){}}, addEventListener: (event, fn) => { handlers[id + ':' + event] = fn; }, showModal() {}, close() {}, focus() {}, select() {} });
    return elements.get(id);
  };
  const context = { window: { HM_CONFIG: {api: 'https://example.test'}, HMProgressSync: require('../../web/progress-sync.js') }, location: {origin: 'https://game.test'},
    localStorage: {getItem: k => data.has(k) ? data.get(k) : null, setItem: (k,v) => data.set(k, String(v)), removeItem: k => data.delete(k)},
    document: {getElementById: element}, crypto: webcrypto, fetch: fetchMock, console: { ...console, warn() {} }, addEventListener(event, fn) { (handlers['window:' + event] ||= []).push(fn); }, setInterval() {}, setTimeout() {}, navigator: {}, URL, URLSearchParams, Blob, FormData, File, Uint8Array };
  vm.runInNewContext(source, context);
  return { bridge: context.window.haphazard, data, handlers, element };
}
const auth = JSON.stringify({token:'token-a', record:{id:'a',email:'a'}});
const wait = async () => { for (let i=0; i<30; i++) await new Promise(r => setImmediate(r)); };
(async () => {
  const ipad = browser({'hm.auth': auth, 'hm.hm.progress': JSON.stringify({levels:{1:{stars:3,best:12,score:90}}, free_retries:2})});
  ipad.bridge.emit('scene', JSON.stringify({name:'map'}));
  assert.equal(ipad.element('acct').hidden, false, 'account control is visible on the chapter map');
  await wait();
  assert.match(ipad.element('acct').textContent, /synced/, 'successful synchronization is visible');
  assert.equal(rows.length, 1, 'legacy progress uploaded');
  const mac = browser({'hm.auth': auth});
  await wait();
  assert.equal(JSON.parse(mac.bridge.load('hm.progress')).levels['1'].stars, 3, 'other device restores progress');
  offline = true;
  mac.bridge.save('hm.progress', JSON.stringify({levels:{1:{stars:2,best:15,score:70},2:{stars:1,best:30,score:20}},free_retries:1}));
  await wait();
  assert.match(mac.element('acct').textContent, /sync pending/, 'offline synchronization is visible');
  assert.equal(JSON.parse(mac.bridge.load('hm.progress')).levels['2'].stars, 1, 'offline save kept');
  offline = false;
  for (const handler of mac.handlers['window:online']) handler();
  await wait();
  assert.match(mac.element('acct').textContent, /synced/, 'status recovers after reconnecting');
  assert.equal(JSON.parse(mac.bridge.load('hm.progress')).levels['1'].stars, 3, 'merge retains better result');
  assert.equal(JSON.parse(mac.bridge.load('hm.progress')).levels['2'].stars, 1, 'merge retains offline result');
  assert.equal(JSON.parse(mac.bridge.load('hm.progress')).free_retries, 1, 'spent retry stays spent');
  mac.handlers['acct:click']();
  assert.equal(mac.element('acct').textContent, 'Sign in to sync');
  assert.equal(mac.bridge.load('hm.progress'), null, 'sign out hides account save');
  mac.element('email').value = 'b'; mac.element('pw').value = 'password';
  await mac.handlers['form:submit']({preventDefault(){}, submitter:{value:'signin'}});
  await wait();
  assert.equal(mac.bridge.load('hm.progress'), null, 'another account cannot see the previous save');
  mac.handlers['acct:click']();
  mac.bridge.save('hm.progress', JSON.stringify({levels:{3:{stars:1,best:40,score:10}}}));
  await mac.handlers['form:submit']({preventDefault(){}, submitter:{value:'signin'}});
  await wait();
  assert.equal(JSON.parse(mac.bridge.load('hm.progress')).levels['3'].stars, 1, 'guest play is adopted at sign in');
  console.log('account progress sync scenarios passed');
})().catch(e => { console.error(e); process.exitCode = 1; });
