const assert = require('node:assert/strict');
const { hydrateProgress } = require('../web/progress-sync.js');

async function main() {
  const remote = Array.from({ length: 11 }, (_, index) => ({
    level: index + 1, won: true, stars: index % 3 + 1,
    time: 30 - index, score: 500 + index * 10,
  }));
  remote.push({ level: 12, won: false, stars: 0, time: 60, score: 0 });
  const pages = [];
  const restored = await hydrateProgress(async (page, perPage) => {
    pages.push({ page, perPage });
    return { page: 1, totalPages: 1, items: remote };
  }, null);

  assert.equal(restored.changed, true);
  assert.equal(Object.keys(restored.progress.levels).length, 11, 'all eleven completed levels should restore');
  assert.equal(restored.progress.levels['1'].stars, 1);
  assert.equal(restored.progress.levels['1'].best, 30);
  assert.equal(restored.progress.levels['1'].score, 500);
  assert.equal(restored.progress.levels['12'], undefined, 'failed runs must not mark a level complete');
  assert.deepEqual(pages, [{ page: 1, perPage: 500 }]);

  const local = {
    levels: { '1': { stars: 3, best: 20, score: 900 } },
    labels: { '4': { '2': 1 } }, free_retries: 2,
  };
  const merged = await hydrateProgress(async () => ({ items: remote, totalPages: 1 }), local);
  assert.equal(merged.progress.levels['1'].stars, 3, 'merge must keep the best local stars');
  assert.equal(merged.progress.levels['1'].best, 20, 'merge must keep the fastest local time');
  assert.equal(merged.progress.levels['1'].score, 900, 'merge must keep the highest local score');
  assert.deepEqual(merged.progress.labels, local.labels, 'local-only review labels must survive');
  assert.equal(merged.progress.free_retries, 2, 'other local progress fields must survive');

  const paged = await hydrateProgress(async (page, perPage) => {
    assert.equal(perPage, 500);
    return page === 1
      ? { page: 1, totalPages: 2, items: remote.slice(0, 5) }
      : { page: 2, totalPages: 2, items: remote.slice(5) };
  }, null);
  assert.equal(Object.keys(paged.progress.levels).length, 11, 'all pages must be merged');

  const repeated = await hydrateProgress(async () => ({ items: remote, totalPages: 1 }), merged.progress);
  assert.equal(repeated.changed, false, 'restoring the same records must be idempotent');
  console.log('PASS cross-device campaign progress restore, monotonic merge and pagination');
}

main().catch((error) => { console.error(error); process.exitCode = 1; });
