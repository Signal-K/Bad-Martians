# Cloudflare deployment plan

Status: plan, nothing deployed. Ticket KES-476.

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

## Phase 1: static PWA on Cloudflare (no backend)

Goal: you can open a stable URL on the iPad, add it to the home screen and play
with progress saved on the device.

1. **Build** (CI and local):
   - `make wasm` produces `web/game.wasm` and `web/wasm_exec.js`.
   - Copy `web/` to `dist/`. Name the wasm by content hash (`game.<hash>.wasm`)
     and write that name into `index.html` and `sw.js`.
2. **Service worker version.** `sw.js` has a fixed cache name
   (`haphazard-martians-v1`). Stamp the build hash into it during the build,
   otherwise returning players keep an old game.
3. **`wrangler.jsonc`:**
   - `assets.directory = "./dist"`, no `not_found_handling` (single page, no
     client routing; `?scene=` is a query string).
   - `name = "haphazard-martians"`, a `compatibility_date`, nothing else.
4. **`dist/_headers`:**
   - `game.*.wasm`: `Content-Type: application/wasm`,
     `Cache-Control: public, max-age=31536000, immutable`.
   - `sw.js` and `index.html`: `Cache-Control: no-cache`.
   - `manifest.webmanifest`: `Content-Type: application/manifest+json`.
5. **Domains.** Preview at `*.workers.dev` first; custom domain later (decision
   below).
6. **CI** (`.github/workflows/deploy.yml`, GitHub Actions):
   - On every push and pull request: `go vet`, `go test ./...`, wasm build.
   - Pull request or branch: `wrangler versions upload`, which gives a preview
     URL to post on the PR.
   - Push to `main`: `wrangler deploy`.
   - Secrets: `CLOUDFLARE_API_TOKEN` (Workers edit only) and
     `CLOUDFLARE_ACCOUNT_ID`.
7. **Post-deploy smoke test** (Playwright, already used locally): load
   `/?scene=menu`, `build`, `run`, `result`, `review`, `style`; fail on any page
   error or console error; check the wasm response is compressed
   (`content-encoding: br` or `gzip`).

Acceptance: preview URL plays a full mission on the iPad, installs to the home
screen, and works offline after the first load.

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

- **Sign in in phase 1.** The menu shows a Sign in button that calls `/api`.
  With no backend it fails, so phase 1 must hide it or show "offline" when
  `/api/health` is unreachable. This is a small change in `web/host.js`.
- **Per-file limit.** 20.5 MB raw against 25 MiB. Watch the size in CI and fail
  the build above 23 MB.
- **Stale service worker.** The cache version must change on every release.
- **First load.** 4.7 MB compressed on mobile data takes a few seconds; the
  loading screen must show progress.
- **arm64 untested.** The iPad runs the wasm in Safari, so arm64 Go rounding
  does not apply to the web build, but it will matter for the native shells.
- **Unverified Cloudflare facts** in the table above come from search results,
  not from reading the pages in full.

## Decisions needed from Liam

1. **Domain.** Which hostname for the game and for the backend? Until then
   everything lives on `*.workers.dev`.
2. **Where PocketBase runs** (phase 2). Recommended: a small VM or volume
   host behind Cloudflare.
3. **A Cloudflare API token and account id** for CI, added as GitHub Actions
   secrets in the repo. Claude cannot create these.

## Work to do, in order

1. Build script and `dist/` with hashed wasm and a versioned service worker.
2. `wrangler.jsonc` and `_headers`.
3. GitHub Actions workflow with preview deploys and the smoke test.
4. First preview deploy and an iPad install check.
5. Phase 2 once the three decisions above are made.
