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

Rigid-body boxes and hinges can be added later behind the same `World`. See
`docs/PHYSICS.md` for the comparison against Box2D and Chipmunk and why we keep
this solver.

## Scene contract (`engine/scenemeta.go`)

Every scene registers with `Game.RegisterScene(SceneMeta, factory)`. It panics
unless the scene has:

- a **type**: menu, build, play, activity, results, sandbox or debug;
- a **layout**: named screen regions (`game/layouts.go`: `center`, `bay`,
  `hud`). Scenes with the same layout share the same bones;
- a **fixture**: the argument it needs to boot on its own.

Any scene runs alone: `?scene=build` in the browser, `go run ./cmd/desktop
-scene build`, or `Game.OpenStandalone(name)` in tests. `engine/harness.go`
has a headless `Recorder` renderer that hashes every draw call.
`game/contract_test.go` walks the registry and, for every scene, checks that it
boots alone, runs 600 frames, draws deterministically, and that every button
is on screen, inside a layout region, and at least 40 px to tap. A new scene is
covered the moment it registers.

## Design system (`game/theme.go`, `widgets.go`, `icons.go`)

The look follows Out There: black icon tiles with white glyphs, thick white
chamfered panels with neon tab labels, resource chips, segmented meters, a dark
nebula backdrop. Tokens (colours, line weight, type sizes) live in `theme.go`;
widgets (`drawPanel`, `drawTile`, `drawChip`, `drawMeter`, `Button`) in
`widgets.go` and `ui.go`; icons are drawn from primitives in `icons.go`, so
there are no image assets to keep consistent. The `style` scene shows all of it.
The assets are our own, in that style; nothing is copied from the game.

## Game (`game/`)

- **Terrain classes** follow AI4Mars (soil, bedrock, sand, big rock); each maps
  to a physics material. Levels are generated deterministically from a seed
  (with multiplication-only maths, so they are bit-identical on every platform).
- **Flow**: menu, then build, run and result. Players never have to label
  anything to play. After a mission the result screen offers an optional
  **review the accuracy** step: in the story, Mission Control wants a second
  opinion on the navcam's terrain read. That is framing; the activity is
  AI4Mars labelling. Reviewing earns build credits, which carry into later
  builds of that mission.
- **Review scene** is the citizen-science task. Each level has 8 segments;
  segments flagged `Gold` have known truth and score the labeller (+2 credits
  for a correct calibration label); the rest are collected for consensus. The
  procedural "Navcam" panels are a stand-in until real AI4Mars imagery is wired in.
- **Build scene** (the rover bay) is a 12×5 grid that stands in the world: the
  Mars sky and the mission's real start terrain sit behind and under it, and
  parts and beams use the same art as the drive (one cell = 1 m). A parts tray
  along the bottom shows each part, its cost and how many more fit the budget;
  tap, swipe sideways or drag a part onto the grid. Beams snap in as parts are
  placed, and hovering a cell previews its links. **Test drop** spawns the
  design on the start terrain with live physics and reports its tilt and
  whether the frame bends (`PoseOf`, a best-fit rotation from blueprint to
  bodies). Validation (one mast, a motor, connected, within budget) gates GO.
  The terrain map under the grid is coloured by the *player's* review, so a
  bad label misleads their build.
- **Run** (`game/run.go`) holds win/lose rules headless so tests exercise them.
  Tests assert every level is beatable by a reference rover and that the
  starter rover is not enough for level 4.
- **Score** (`game/score.go`): finish 500, +150 per sample, plus up to 500 for
  speed (full at 0.6 x par, none at 2 x par, linear between). Stars are
  unchanged (finish, all samples, under par). Best score is saved per level and
  sent with the run.

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
| `hm_runs` | Public read (leaderboard); create only as yourself; carries score |
| `hm_labels` | Read own; create as yourself; unique per (user, level, segment) |

`correct`/`gold` on labels are client-computed for now; consensus and gold
scoring should move server-side before the data is trusted.

## Not done yet

- Real AI4Mars images and server-side consensus.
- Leaderboard UI, sound, particle effects.
- Clerk, Star Sailors integration.
- Multi-touch (gas and steer at once); a single pointer is supported today.
- Native iOS/Android shells.

## Levels and missions as data

Levels are JSON files in `game/data/levels/` and daily missions are JSON files in
`game/data/missions/`. Both are embedded in the binary and validated at startup;
a bad file fails `go test` with the file name and the reason, and unknown fields
are errors so typos are caught.

A level declares `biome`, `length`, `seed`, `credits` (parts budget), `par`,
`hill`, `traction` (multiplies every terrain class's friction) and a `mix` of
AI4Mars classes. `Obstacle()` and `Difficulty()` give the visible obstacle load
and a weighted difficulty from those fields. Campaign ids run 1..n without gaps.

A mission wraps a level (id 100 or more) with a `date`, `title`, `story` and a
`reward` (`credits`, `free_retry`). `MissionFor("YYYY-MM-DD")` returns the day's
mission. A test requires the reference rover to beat every shipped mission, so a
hand-authored mission that is impossible fails before it ships.

## The in-game editor

Every scene runs inside an editor that is part of the engine (`engine/editor.go`).
Open it with F1 or the backtick key, `?edit=1` in the URL, or `haphazard.edit()`
in the browser console. It opens paused and the scene stops seeing the pointer,
so taps select instead of pressing buttons.

- Toolbar: Play/Pause (P), Step one fixed step (.), Hide, Delete (Del), Close.
- Hierarchy lists every GameObject back to front; tap a row to select it.
- Tap an object in the viewport to select it, drag to move it.
- Inspector shows name, tag, position, rotation, layer, active and components.
- A component opts in to viewport picking with `engine.Bounder` and to dragging
  with `engine.Mover`. Buttons and the rover do; objects with only a world
  position are picked at that point.

Edits are live only: they are not saved to the level or scene data.
