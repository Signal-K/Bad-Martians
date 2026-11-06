# Architecture

## Goals

- A Unity/Godot-shaped runtime from day one: **GameObjects**, **Components**,
  **Scenes**, and a **physics world**, so later games and tools share it.
- One graphics codebase for web (PWA), then iOS/Android.
- Game rules independent of any renderer, platform or backend.

## Runtime (`engine/`)

- `GameObject` has a transform, tag, draw layer and components. A component is
  any value; it opts into behaviour by implementing `Starter`, `Updater` or
  `Drawer`. `engine.Get[T](obj)` looks components up by type.
- `Scene` owns objects and a camera, runs `Start`/`Update` on a **fixed 60 Hz
  step** and draws in layer order. `OnUpdate` holds scene-level logic.
- `Game` accumulates frame time into fixed steps, latches press/release edges so
  a tap is never lost on a frame with no step, and switches scenes at frame
  boundaries (`Load(name, arg)` is safe inside an update).
- `Renderer` (lines, circles, rects, polygons, text) and `Host` (load, save,
  emit) are the only two doors out. Everything else is pure Go.

Scenes here are also the UI: buttons are components, so menus use the same
machinery as gameplay.

## Physics (`physics/`)

Position-based dynamics, chosen because it is tiny, stable and deterministic:

- `Body` is a round particle (some are wheels with angle, spin and a motor).
- `Constraint` keeps two bodies a fixed distance apart. A rover is a lattice of
  bodies joined by constraints, so any grid layout the player draws is a valid
  vehicle with no per-design physics code.
- `Terrain` is a uniformly sampled heightfield with a material per segment.
  Friction is Coulomb in displacement space; wheels exchange slip between
  body and spin, which gives traction limits (sand slips) with no force solver.
- Mars gravity (3.71 m/s²). 10 substeps × 6 constraint iterations per step.

Rigid-body boxes and hinges can be added later behind the same `World`.

## Game (`game/`)

- **Terrain classes** follow AI4Mars (soil, bedrock, sand, big rock); each maps
  to a physics material. Levels are generated deterministically from a seed.
- **Label scene** is the citizen-science task. Each level has 8 segments;
  segments flagged `Gold` have known truth and score the labeller (+2 credits
  for a correct calibration label); the rest are collected for consensus. The
  procedural "Navcam" panels are a stand-in until real AI4Mars imagery is wired in.
- **Build scene** is a 12×7 grid; parts cost credits; validation (one mast, a
  motor, connected, within budget) gates the GO button. The course preview is
  coloured by the *player's* labels, so a bad label misleads their build.
- **Run** (`game/run.go`) holds win/lose rules headless so tests exercise them.
  Tests assert every level is beatable by a reference rover and that the
  starter rover is not enough for level 4.

## Hosts

`client/` adapts Ebitengine to `Renderer` and `Input`.

- **Web/PWA** (`cmd/wasm` + `web/`): `web/host.js` implements `Host` with
  `localStorage` and PocketBase. Run and label events queue offline and flush
  once signed in. A service worker caches the shell.
- **Native shells**: Ebitengine supports `ebitenmobile`, which wraps the same
  Go code in an iOS framework / Android AAR, so a Swift or Kotlin app only
  needs to host the view and implement `Host`. Nothing in `engine/`, `physics/`
  or `game/` changes.

## Backend (`cmd/server`, `migrations/`)

PocketBase with Go migrations. Auth is the built-in `users` collection; Clerk
replaces it in staging/prod by swapping the token check in front of the
collection rules.

| Collection | Access |
|---|---|
| `bm_runs` | Public read (leaderboard); create only as yourself |
| `bm_labels` | Read own; create as yourself; unique per (user, level, segment) |

`correct`/`gold` on labels are client-computed for now; consensus and gold
scoring should move server-side before the data is trusted.

## Not done yet

- Real AI4Mars images and server-side consensus.
- Leaderboard UI, sound, particle effects.
- Clerk, Star Sailors integration.
- Multi-touch (gas and steer at once); a single pointer is supported today.
- Native iOS/Android shells.
