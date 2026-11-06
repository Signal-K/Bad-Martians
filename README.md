# Bad Martians

A Bad Piggies-style physics builder set on Mars, and a citizen-science game:
before each level you **label terrain** (the [AI4Mars](https://science.nasa.gov/mission/msl-curiosity/)
classes: soil, bedrock, sand, big rock). Labels earn parts credits and feed a
consensus dataset. Then you **build a rover** on a grid and **drive it** to the
finish. Collect samples and beat par time for three stars.

## Run it

```bash
make run            # builds the WASM game, starts PocketBase + PWA on :8090
# game:  http://127.0.0.1:8090/     admin UI: http://127.0.0.1:8090/_/
make test           # engine, physics and gameplay tests (headless)
make desktop        # native window (needs Ebitengine's platform deps)
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
| `game/` | Bad Martians: terrain classes, procedural levels, parts and rover, scenes (menu, label, build, run, result), progress. No rendering or platform code. |
| `client/` | Ebitengine adapter: the only code that touches a graphics library. |
| `cmd/wasm`, `web/` | Browser build and PWA shell (manifest, service worker, account dialog, PocketBase bridge). |
| `cmd/server`, `migrations/` | PocketBase backend (auth, `bm_runs`, `bm_labels`) that also serves the PWA. |
| `cmd/desktop` | Native dev build. |

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for design decisions.
