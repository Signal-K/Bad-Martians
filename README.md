# Haphazard Martians

A physics rover-builder set on Mars, and a citizen-science game. You **build a
rover** on a grid and **drive it** to the finish. Collect samples and be fast:
score depends on finishing, samples and time. The three-mission expedition
challenges you to make one credit budget last across the whole route. Surviving
rovers can be rebuilt at full part value; crashes lose their cost. An under-par
finish earns two credits for the next stage.
The campaign ends off Mars: Phobos (level 21), Deimos (level 22) and Earth's Moon
(level 23) run under their own low gravity, with black skies, their own ground
colours, and Mars (or Earth, for the Moon) in the sky. Open one alone with
`?scene=run&level=21` (or `-level 21`). Phobos's loose ground also gives the wheels
less grip (`Body.Grip`). Each level keeps a ghost of your fastest win, drawn
beside the live rover on later attempts; it is stored on the device only. See [docs/ROADMAP.md](docs/ROADMAP.md) for what comes next.
The lander hops between completed expedition stages; continue when ready to
enter the next rover bay with the surviving design and budget.
The terrain
review experience is paused while its real data flow and design are developed.
Existing review data and previously earned standalone-mission credits are preserved;
the unfinished review scene is not registered.

## Run it

```bash
make run            # builds the WASM game, starts PocketBase + PWA on :8090
# game:  http://127.0.0.1:8090/     admin UI: http://127.0.0.1:8090/_/
make test           # engine, physics and gameplay tests (headless)
make desktop        # native window (needs Ebitengine's platform deps)
# any scene alone: http://127.0.0.1:8090/?scene=build   or   go run ./cmd/desktop -scene build
```

Go 1.27 (the PocketBase dependency pins it; the Go tool fetches it automatically).
Controls: hold the **Gas** / **Reverse** buttons (touch) or →/← (keyboard).
Account page (map, bottom right): shows whether this device is synced, lets a
signed-in player set a username, lists their daily mission history and shows the
leaderboard (best score per mission, summed; players without a username are not
listed, and emails are never public). Shape school and the style sheet are archived:
their scenes and tests remain (`?scene=style`, `LessonSession`), but nothing on the
map opens them.
Sign-in is optional: without it, progress stays in the browser and events queue
until you sign in. Once signed in, campaign progress, tutorial and school
completion, and expedition checkpoints sync across devices. Existing browser
progress is adopted when you sign in; reconnecting retries an offline sync.

Leaderboard runs are checked on the server (`runcheck/`, `game.CheckRun`). A
win must carry the stick inputs the client recorded; the server drives the same
rover on the same level with them (`game.Replay`) and the drive has to finish
with the same samples, stars and time. The score must be one the scoring rules
produce for that, the design must be legal and cost what it says, and a player is
limited to 20 runs a minute. The sim is bit-identical on every platform, so a
browser's drive replays exactly on the server (`make test-wasm` checks it).
The standalone server registers it; the shared Star Sailors backend must call
`runcheck.Register(app)` and run this repo's migrations (4 adds `hm_runs.inputs`),
or production keeps accepting any score. Deploy the game build first: a
cached older client sends wins without inputs, which the server refuses.

## Layout

| Path | What |
|---|---|
| `engine/` | Game objects, components, scenes, fixed-step loop, input, `Renderer`/`Host` interfaces. No dependencies. |
| `physics/` | Deterministic 2D position-based physics: particles, wheels with motors, distance constraints, heightfield terrain with materials. |
| `game/` | Haphazard Martians: terrain classes, procedural levels, parts and rover, scenes (menu, build, run, result, travel, style), progress. No rendering or platform code. |
| `client/` | Ebitengine adapter: the only code that touches a graphics library. |
| `cmd/wasm`, `web/` | Browser build and PWA shell (manifest, service worker, account dialog, PocketBase bridge). |
| `cmd/server`, `migrations/` | PocketBase backend (auth, `hm_runs`, `hm_labels`, `hm_saves`, `hm_profiles`, `hm_leaderboard`) that also serves the PWA. |
| `cmd/desktop` | Native dev build. |

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for design decisions and [docs/PHYSICS.md](docs/PHYSICS.md) for the physics solver comparison.

## Static demo (no backend)

`scripts/build-dist.sh` builds `dist/` for Cloudflare or any static host. By default it
is a demo: progress stays in the browser and sign-in is hidden. Set `HM_API_URL` to
connect it to a backend. See `docs/DEPLOY_CLOUDFLARE.md`.

Analytics: the built site sends anonymous gameplay events (`hm_*`, tagged
`game = haphazard-martians`) to PostHog, plus a one-time "what was confusing?"
survey after the first run. No identify call, no session recording, no player
details. `HM_POSTHOG_KEY=off` builds without it; local `web/config.js` has none.
The events come from `game/analytics.go`; the playtest dashboard is "Haphazard
Martians playtest" in PostHog.

Browser scene smoke tests: install Node.js, Playwright (`npm install --no-save playwright`)
and Chromium, then run `make visual-test`. Set `CHROMIUM_PATH` if Chromium is not
at `/usr/bin/chromium`. The suite opens each scene at desktop and phone sizes,
checks the scene registry and browser errors, and saves screenshots plus a gallery
in `test-results/visual/`. These are rendering smoke checks, not pixel baselines or
a substitute for human visual review. Serve the repository root to use the gallery's
scene links (the generated WASM lives in `web/`).

To run the Go suite, distribution build, disposable PocketBase API checks and
browser scene checks in Docker, install the project's Go toolchain and Playwright
locally (and populate the Go module cache), then run `tests/container/run.sh`.
The test image provides Chromium and Node; the runner mounts the local toolchain,
module cache and Playwright package. Its PocketBase data lives in a temporary
container directory and is deleted when the test finishes.

GitHub Actions runs Go checks for Go changes, browser scene smoke tests for
rendering/game changes, and backend smoke tests for server, migration or web
changes. Shared dependencies and workflow changes select all affected checks;
manual dispatch runs all three. Browser screenshots are retained as a workflow
artifact. These are component-level selections, not function-level coverage.
The deployment workflow also performs its own build and checks.
