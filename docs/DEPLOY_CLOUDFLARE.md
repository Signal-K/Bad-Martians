# Cloudflare deployment plan

Status: phase 1 is built and tested locally but not deployed (it needs your Cloudflare
secrets). Phases 2 and 3 are plans. Ticket KES-476.

## Summary

- **The game ships as static files on Cloudflare** (Workers static assets). It is
  a WebAssembly PWA and already works with no backend, using browser storage.
- **PocketBase does not run on Workers or Pages.** It is a long-running Go
  process that writes a SQLite file to disk. It needs a host with a persistent
  disk. Phase 2 puts it on a small VM or a volume-backed container host and
  fronts it with Cloudflare.
- **Same origin.** A small Worker serves the static files and forwards `/api/*`
  and `/_/` to PocketBase, so `web/host.js` keeps using relative URLs and there
  is no CORS setup.

## Facts that shape the plan

| Fact | Source | Checked |
|---|---|---|
| Our stripped `game.wasm` is 20.5 MB raw and 4.7 MB gzipped | built here with `make wasm` flags | yes |
| Workers static assets allow 25 MiB per file and 20,000 files per version | [Workers limits](https://developers.cloudflare.com/workers/platform/limits/) | search result only, re-check before relying on it |
| Cloudflare Containers disk is ephemeral: a container that sleeps restarts from a fresh image | third-party summaries of the [pricing docs](https://developers.cloudflare.com/containers/platform/pricing/) | search result only, re-check |
| Durable Objects are the Cloudflare option with persistent SQLite | same searches | search result only |

The wasm has about 5 MB of headroom under the per-file limit. If the game grows
past it, split the wasm or move to a smaller build before anything else.

## Phase 1: static demo on Cloudflare (no backend) — built

Goal: a stable URL where you can open the game on the iPad, add it to the home
screen and play, with progress saved on the device. No backend is needed.

What is in the repo:

- `scripts/build-dist.sh` builds `dist/`: the wasm, the shell, a service worker
  stamped with the build id (so returning players get the new game), `_headers`
  and `config.js`. It fails the build if `game.wasm` passes 23 MiB.
- `web/config.js` says where the backend is. The build writes `api: ""` by
  default, which is **demo mode**: the Sign in button is hidden, nothing is sent
  anywhere, progress stays in the browser. Set the `HM_API_URL` repository
  variable (a public URL, not a secret) and the same build connects to that
  backend instead. The committed default is `location.origin`, so the bundled
  PocketBase server (`make run`) behaves as before.
- `wrangler.jsonc` serves `dist/` as static assets, no Worker code.
- `deploy/_headers` sets the wasm content type, revalidating cache for the game
  files, `no-cache` for `index.html`, `sw.js` and `config.js`.
- `.github/workflows/deploy.yml` runs vet, tests and the dist build on every
  push. If `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` are set, `main`
  runs `wrangler deploy` and other branches run `wrangler versions upload` for a
  preview. Without the secrets it builds, tests and skips the deploy. The first
  deploy has to come from `main`, because a preview version needs the Worker to
  exist.

Checked here: the dist build, `wrangler deploy --dry-run` (reads the 11 asset
files), and a browser run of `dist/` showing the Sign in button hidden, the
service worker registered, no `/api` requests and no console errors.
Not checked: a real deploy, the workflow on GitHub, and the iPad.

To go live, you add the two secrets, merge or push to `main`, and open the
`*.workers.dev` URL Wrangler prints.

### Connecting to the Star Sailors backend

Set `HM_API_URL` to its public URL and rebuild. The game then uses the same
calls as today: `users` auth and the `hm_runs` and `hm_labels` collections. For
that to work:

- the backend needs those two collections (migration
  `migrations/1_haphazard_martians.go`) and a CORS rule that allows the game's
  origin;
- sign-in uses PocketBase email and password for now. If the shared backend
  authenticates differently, `web/host.js` is the only file that changes.

I have not seen that backend, so I have not tested any of this against it.

## Phase 2: PocketBase backend

PocketBase needs a persistent disk. Options, in order of recommendation:

1. **A small VM or volume-backed container host** (Fly.io machine with a
   volume, or any VPS), running `bin/server serve`.
   Add scheduled snapshot backups to R2 (`pb_data` is one directory).
   Cloudflare proxies the hostname (`pb.<domain>`), which adds TLS and DDoS
   protection and hides the origin.
2. **Cloudflare Containers plus Litestream to R2.** Everything stays on one
   platform, but the container disk is ephemeral, so every cold start restores
   the database from R2 and every write must stream out before the container
   sleeps. A crash loses the last few seconds of writes. Not recommended until
   persistent volumes exist; re-check Cloudflare's docs before deciding.
3. **Rewrite the backend on Workers and D1.** `web/host.js` only needs
   `load`, `save` and `emit` (runs and labels), so this is realistic later and
   would fit Clerk sign-in well. It is a rewrite and drops PocketBase, so it is
   a separate decision.

Wiring for option 1:

- Add a Worker `fetch` handler in front of the static assets: `/api/*` and
  `/_/` are proxied to the PocketBase origin; everything else falls through to
  the assets. Keep the admin UI (`/_/`) behind Cloudflare Access.
- Environment on the origin: PocketBase superuser created once from the CLI;
  `pb_data` on the volume; the binary built from this repo.
- Migrations already run on start (`Automigrate`), so a deploy is "replace the
  binary and restart".
- `web/host.js` already calls `location.origin`, so same-origin proxying needs
  no game change. Check PocketBase auth and the service worker (`sw.js` already skips `/api/`
  and `/_/`).

Acceptance: sign in on the iPad, finish a mission, see the run in `hm_runs`
in the admin UI, and restore a backup into a scratch instance.

## Phase 3: daily missions

Missions are JSON compiled into the wasm and date-gated, so a mission can be
authored ahead of time and shows up on its date with no deploy. Two follow-ups:

- A GitHub Action on a schedule that validates `game/data/missions/*.json`
  (it is just `go test`) and opens a pull request for the next day's file.
- Later: serve missions from `/missions/index.json` at runtime so publishing a
  mission does not need a new wasm. This is a small game change tracked in
  KES-475.

## Environments

| Environment | Source | URL | Backend |
|---|---|---|---|
| preview | each branch or pull request | per-version preview URL | none |
| production | `main` | custom domain | PocketBase (phase 2) |

## Cost

Phase 1 should fit the Workers free plan at this traffic, since static asset
requests are expected to be free, and one 4.7 MB download per new player is
small (not verified; check the pricing page). The paid plan is only needed for Containers or Durable Objects. Phase 2
cost is whatever the small VM costs. Re-check current Cloudflare pricing before
committing to a paid product.

## Risks

- **Per-file limit.** 20.5 MB raw against 25 MiB. Watch the size in CI and fail
  the build above 23 MB.
- **Stale service worker.** The build stamps the cache version, but only if every
  release goes through `scripts/build-dist.sh`.
- **First load.** 4.7 MB compressed on mobile data takes a few seconds; the
  loading screen must show progress.
- **arm64 untested.** The iPad runs the wasm in Safari, so arm64 Go rounding
  does not apply to the web build, but it will matter for the native shells.
- **Unverified Cloudflare facts** in the table above come from search results,
  not from reading the pages in full.

## Decisions

1. **Domain.** Decided: `*.workers.dev` for now.
2. **Backend.** Decided: the demo has none; it will connect to the Star Sailors
   backend later. Phase 2 hosting is only needed if a separate PocketBase is
   still wanted.
3. **Secrets.** Liam adds `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` as
   GitHub Actions secrets himself.

## Work to do, in order

1. Add the Cloudflare secrets, push to `main`, check the URL on the iPad.
2. Add the post-deploy smoke test to the workflow.
3. Connect to the Star Sailors backend (needs its URL and CORS rule).
4. Phase 2 hosting only if a separate PocketBase is still wanted.
