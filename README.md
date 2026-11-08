# Haphazard Martians

A physics rover-builder set on Mars, and a citizen-science game. You **build a
rover** on a grid and **drive it** to the finish. Collect samples and be fast:
score depends on finishing, samples and time. Afterwards you can **review the
accuracy** of the navcam's terrain read (the [AI4Mars](https://science.nasa.gov/mission/msl-curiosity/)
classes: soil, bedrock, sand, big rock). That is the real labelling task, framed
as helping Mission Control. Reviews earn build credits and feed a consensus dataset.

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
Sign-in is optional: without it, progress stays in the browser and events queue
until you sign in.

## Layout

| Path | What |
|---|---|
| `engine/` | Game objects, components, scenes, fixed-step loop, input, `Renderer`/`Host` interfaces. No dependencies. |
| `physics/` | Deterministic 2D position-based physics: particles, wheels with motors, distance constraints, heightfield terrain with materials. |
| `game/` | Haphazard Martians: terrain classes, procedural levels, parts and rover, scenes (menu, build, run, result, review, style), progress. No rendering or platform code. |
| `client/` | Ebitengine adapter: the only code that touches a graphics library. |
| `cmd/wasm`, `web/` | Browser build and PWA shell (manifest, service worker, account dialog, PocketBase bridge). |
| `cmd/server`, `migrations/` | PocketBase backend (auth, `hm_runs`, `hm_labels`) that also serves the PWA. |
| `cmd/desktop` | Native dev build. |

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for design decisions and [docs/PHYSICS.md](docs/PHYSICS.md) for the physics solver comparison.

## Static demo (no backend)

`scripts/build-dist.sh` builds `dist/` for Cloudflare or any static host. By default it
is a demo: progress stays in the browser and sign-in is hidden. Set `HM_API_URL` to
connect it to a backend. See `docs/DEPLOY_CLOUDFLARE.md`.
