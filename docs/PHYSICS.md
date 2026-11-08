# Physics: solver comparison and recommendation

Ticket KES-469. The experiment lives in `experiments/physcompare` (separate Go
module, so Box2D and Chipmunk never reach the game build). Raw tables:
`experiments/physcompare/results.md`.

Compared on the same rover and terrain scenarios:

- **Ours**: deterministic position-based dynamics. Round particles joined by
  rigid distance links on an 8-neighbour lattice (`physics/`).
- **Ours, 4x iterations**: the same solver with 4x the constraint iterations.
- **Box2D**: `ByteArena/box2d` v1.0.2, a Go port of Box2D 2.3. Rigid bodies
  (boxes and circles) welded together, revolute motor joints on wheels. This is
  the Bad Piggies model.
- **Chipmunk**: `jakecoffman/cp/v2` v2.4.0. The same lattice as ours, solved
  with sequential impulses.

## What came out

| | Ours | Box2D | Chipmunk |
|---|---|---|---|
| Rests still on flat rock | yes | yes | yes |
| Tunnels at 86 m/s impact | no | no | no |
| Bit-identical rerun | yes | yes | yes |
| amd64 and wasm give identical bits | yes | yes | yes |
| Beam sag, 10 m single-row beam | 200 mm (on the ground) | 33 mm | 203 mm (on the ground) |
| Step cost, 6 bodies | 17 µs | 26 µs | 9 µs |
| Step cost, 84 bodies | 343 µs | 313 µs | 165 µs |
| WASM size added over an empty Go program | +0.25 MB | +0.75 MB | +0.96 MB |
| Last release | this repo | Aug 2018 | Dec 2025 |

Everything that makes a sim feel solid held for all three: settling, no
tunnelling at 86 m/s, no jitter, repeatable runs. Speed is not a deciding
factor either; an 84-part rover costs well under a millisecond a step in all of
them, against a 16 ms frame.

Three findings do matter:

1. **A single row of parts has no bending stiffness in a lattice solver.** Our
   solver and Chipmunk-as-lattice both let a 10 m beam fold onto the ground.
   Pin-jointed collinear links have no first-order stiffness. Spending CPU does
   not help (4x iterations, same 200 mm), and extra links between parts two
   apart did not help either. Players have to triangulate. Box2D welds hold the
   beam at 33 mm. This is a game design choice more than a solver quality gap
   (see the decision below).
2. **Level generation was the one thing not portable.** The solvers are
   bit-identical between amd64 and wasm, but `math.Exp` in the level generator
   uses an assembly routine on amd64 and plain Go on wasm, so the same level
   differed in its last bits. Fixed: boulders now use a multiplication-only
   bump, and `TestLevelTerrainGolden` pins every level's terrain hash (verified
   under native and wasm).
3. **Swapping solvers means re-tuning every level.** Pars, hills and motor
   strength were tuned to our solver. On the Box2D and Chipmunk setups the
   same reference rover is faster on levels 1 to 3, gets stuck on levels 4 and 5
   with Box2D, and on level 5 with Chipmunk, as-is. The climb numbers (10° vs 25°) mostly reflect motor tuning, not
   solver quality, and should not be read as a win for either side.

Not tested: arm64 (iOS, Android, Apple silicon). Go may fuse `x*y+z` into one
fused multiply-add there, which changes results in the last bit. In our own
code that is fixed by wrapping products in `float64(...)` (done in the new
level code). In a third-party solver we would have to fork it to do the same.
That needs a real arm64 run before we trust cross-device replays.

## Recommendation

**Keep our own solver. Do not swap now.**

- It is as fast and as well behaved as the alternatives on every measure we
  could test, adds the least to the download, and we can read and change all
  of it. Parts in the tech tree (durability, sand sinking, breakable joints)
  need solver changes that are easy in 300 lines we own and awkward in a fork
  of a 2018 port.
- It is the only one we can make bit-reproducible on arm64 without forking.
  That matters if the server is to check recorded runs.
- Nothing in the comparison showed a fault it would fix by switching.

The one real gap is rigid structure. Decision needed (this is the game design
question, so it is yours):

- **Truss builder (current).** Players triangulate, like a bridge builder.
  Cheap, deterministic, readable. A single-row beam collapses.
- **Rigid blocks like Bad Piggies.** Add an angular (bend) constraint so frames
  hold shape and can break at joints. In our solver this is a three-body
  constraint, roughly a day's work, and it keeps everything else. If we later
  decide a full rigid-body model is needed, Chipmunk v2 is the better base than
  the Box2D port (maintained, smaller to ship, smaller API), but we would
  retune every level.

**Decision (Liam, 2026-11-08): truss builder.** Structure stays as it is: players
triangulate their designs, and no bend constraint is added. Consequences:

- A single row of parts folds, so the build screen should teach triangulation
  (a hint when a design has an unbraced span; tracked on KES-474 with the parts).
- Part count is the economy. A crash loses every part used in that attempt
  (KES-471), so cost per part and the parts budget are the main balance levers.
- Chipmunk and the bend constraint are not pursued. Revisit only if players
  find trusses frustrating.

Follow-up: an arm64 determinism run.

## Regression suite

The scenarios are now tests in `game/physics_regression_test.go` and run on
every `go test`: rests still on rock, no tunnelling from 1000 m, repeatable
runs, and the reference rover climbs a 20° slope. Level terrain is pinned in
`TestLevelTerrainGolden`.
