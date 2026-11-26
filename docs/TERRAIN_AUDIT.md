# Terrain engine audit (26 Nov 2026)

Scope: how Haphazard Martians builds, collides with and draws its ground
(`game/level.go`, `physics/world.go`, `game/ground.go`, `game/art.go`), and what
the recent TakeOn changes offer. Numbers were measured on this branch; nothing
here was profiled on a phone or iPad.

## How it works today

- **Generation** (`BuildLevel`): a 1D heightfield sampled every 0.5 m, from
  three summed sines scaled by the level's `Hill`, plus micro-relief per
  surface class (sand ripples, boulders) and flat-shelved crevasses. Eight
  stretches per level each get one of four classes (soil, sand, bedrock,
  boulders). Fully deterministic from the seed.
- **Collision** (`Terrain.contact`, `World.collide`): circles against the
  piecewise-linear surface, deepest contact per body per substep, Coulomb
  friction in displacement space, per-class friction and drag. Position-based
  solver, 10 substeps, 6 constraint iterations.
- **Drawing** (`drawTerrain`): redrawn every frame, one 0.5 m column at a time:
  five layer bands (crust, pebbles, soil, rock, depth shading), then pebbles,
  grit, cracks, rocks and a surface line, all seeded per column so nothing
  flickers. `Ground` (strata and finds) is presentation only; physics never
  reads it.

## Measured

| Check | Result |
| -- | -- |
| Terrain identical across platforms | **Yes.** All 23 levels hash identically on amd64 and `GOOS=js GOARCH=wasm`. Only 6 of 23 had golden hashes; all 23 do now. |
| Steepest non-crevasse slope | 11° on mission 1, up to 50° on mission 17 (hill 3.0 to 4.5 m) |
| Crevasse walls | 30 m drop in one 0.5 m cell (89°), by design |
| Natural airtime, flat-out starter rover | under 0.3 s on most levels; 1 to 1.6 s on levels 10, 12, 17 to 19; 5.3 s on Phobos, 2.6 s on the Moon (low gravity) |
| Natural flips | none on any level |
| Drive scene draw calls per frame | **1,433** (461 polygons, 381 circles, 514 lines, 72 rects), nearly all terrain |

## Findings

1. **Stunts cannot happen by accident, and levels give no jumps.** Without a
   crevasse or low gravity the rover never leaves the ground for more than a
   third of a second. The terrain has no ramp, kicker or crest primitive, so
   MAR-85 needs one in `BuildLevel`. Add it after all random draws, as crevasses
   are, so existing levels do not shift (the golden hashes now enforce that).
2. **Terrain is redrawn from scratch every frame.** About 1,400 draw calls
   re-derive the same pixels each frame, including a fresh random generator per
   column. The ground never changes at runtime, so it can be baked once per
   level into chunk images and blitted. This is the first thing to profile on a
   phone; I have not measured frame time, so I do not know if it hurts yet.
3. **Rock bands extend 60 m down every column** and are drawn even when far off
   the bottom of the view; only a quad that is entirely below the view is
   skipped. Cheap to cut.
4. **Physics is sound for current speeds.** One contact per body per substep
   converges over substeps; the `sd < -3r` skip would only matter above roughly
   600 m/s. No tunnelling risk.
5. **Heightfield limits.** One height per x: no overhangs, caves, loops or
   vertical faces (crevasse walls are the one faked exception). Material changes
   only at the 8 stretch boundaries. Fine for the current game; hard limits for
   any "terrain that does something" idea.
6. **Float determinism is only partly guarded.** `bump` avoids `math.Exp` and
   blocks fused multiply-add with explicit `float64()` conversions; the hill sum
   in `BuildLevel` does not. amd64 and wasm are identical (measured); an arm64
   native build (`make desktop` on Apple Silicon) or an amd64 build with
   `GOAMD64=v3` may fuse and differ in the last bit. Matters for race ghosts
   (MAR-31) and server-side run replay (MAR-87), not for today's game. Not
   tested on arm64 here.

## TakeOn: what changed and what is reusable

TakeOn is now a shared **Go core** (`go/`, module `github.com/Signal-K/takeon-go`)
that drives the web, Pixi, Swift and Android hosts. It is a top-down isometric
**voxel** world, not a side-scrolling heightfield, so its terrain code does not
drop in. The ideas and some tools do.

| TakeOn piece | What it is | Use for Martians |
| -- | -- | -- |
| `IsoTerrainLayer` | Cached terrain in 16-tile chunks, redrawn only when dirty | The pattern for finding 2: bake the ground into chunk images once, blit per frame |
| `crossSectionImage` | Pure (DOM-free) pixel image of the real generated column, in material colours | Level preview and a "what is under you" view from the same data `Ground` already holds |
| `rasterizeFlatTerrain` | Hillshaded, contour-shaded terrain raster, no DOM | An elevation-profile strip on the build screen or map, and the "jump coming up" hint from the stunt sketch |
| `analyzeTerrain` (`slopeField`, `reachableMask`) | Slope distribution and drivable fraction for a climb limit | Automatic level QA: flag a level that is unwinnable or has no jumpable crest |
| Real elevation data (MOLA for Mars, LOLA for the Moon) | Normalised height patches (`mars-jezero`, `moon-imbrium`) | Real transects as level profiles for Gale Crater, Mount Sharp and the Moon. **The patch file is empty** (`generated.ts` has no data) until `scripts/fetch-dem.mjs` runs; `trek.nasa.gov` is reachable from this environment |
| Noise, blue noise, ore bands | Authored noise and depth-stratified minerals | Replace the three-sine hills; ore bands map onto the existing finds and samples |
| Biomes and materials data (`go/data/*.json`) | Chunked biome skins, one data source for all hosts | Per-location look without hand-tuning `Classes` |
| Layers seam (`registerWorldLayer`) | Hook for extra world layers (caves, sky) after generation | Sky or underground layers if the heightfield limit in finding 5 is ever the blocker |
| Synthesised audio (`audio.ts`) | Web Audio sounds, no asset files, per-body ambience | Impact and stunt sounds ("impact sound emphasis" in the sketch) |
| Parity suite (`golden/`, `fdlibm.go`) | Bit-exact cross-platform checks, FMA-safe maths | The model for finding 6 |

TakeOn's Go module is the same language as Martians, so importing its noise and
DEM sampling is possible, but it would couple two repos for little code. Copying
the DEM sampler and the golden-hash approach is simpler.

## Suggested order

1. Ramp, crest and kicker primitive in `BuildLevel` (unblocks MAR-85).
2. Profile the drive scene on a phone; if it is slow, bake the terrain into chunks.
3. Run `scripts/fetch-dem.mjs` in TakeOn and try one real transect (Mount Sharp).
4. Elevation-profile strip with a jump marker (stunt sketch hint).
5. Fuse-proof the hill sum in `BuildLevel` before ghosts or server replay.
