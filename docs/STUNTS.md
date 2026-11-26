# Stunt engine and launch ramps

Source: Liam's design sketch, 26 Nov 2026 (Game Ideas PDF). `game/stunts.go`,
`game/level.go` (ramps).

## Rules

| Stunt | Rule | Points |
| -- | -- | -- |
| Backflip / frontflip | 85% of a full turn in one jump (nose up = back, nose down = front); each further full turn pays again, up to 3 per jump | 2 per flip |
| Wheelie | back wheel down, front wheel up, nose 20° to 69° up; pays at every whole second; a dip under 0.25 s does not end it | 2 per second |
| Airtime | every body off the ground for at least 1.5 s | 2 bonus |

A sketch point is 50 score points (`stuntPointScale`), so a flip reads as
`+100`, next to 150 for a sample.

**Jump stunts pay on landing, and only for a jump that gets somewhere.** A jump
must cover at least 2.5 m and land at least 6 m past the last paid one. A flip
the rover does not survive to land pays nothing. Wheelies pay as they happen.

Open choices, all constants at the top of `stunts.go`: the airtime threshold
(the sketch left it blank, 1.5 s is a guess), the 50x scale, the 3-flip cap.

## Launch ramps

A level file can carry up to four `ramps`:

```json
"ramps": [ { "at": 0.3, "rise": 1.5, "run": 5, "fall": 3 } ]
```

`at` is the foot, as a fraction of the level length (0.1 to 0.8); `rise` is
the metres gained by the lip (0.3 to 6); `run` is the metres from foot to lip
(2 to 24); `fall` is the landing slope's length (0 to 10, default 3). The ramp
is at most 45°, must end 10 m before the finish, and two ramps need 6 m
between them. A rounded foot turns into a straight slope, so the launch angle is
constant. Ramps are added after every other terrain feature, on top of the
ground that is there, so a level without ramps is bit-identical to before (the
golden terrain hashes cover all 23 campaign levels).

How big, from the headless drives (starter rover, full throttle, firm ground):

| Rise / run | Air without leaning | Flips without leaning |
| -- | -- | -- |
| 1.0 / 5 | 0.8 s | 0 |
| 1.5 / 5 | 1.9 s | 0 |
| 2.0 / 6 | 2.4 s | 0 |
| 3.0 / 6 | 2.9 s | 0 |

So 1.5 m of rise is the least that pays airtime, and no ramp flips a rover by
itself: a flip needs the player to lean in the air.

**Kicker Flats** (`game/data/missions/2026-12-21.json`, level 132) has two ramps
(1.5/5 at 30%, 2.5/6 at 60%). It is dated the first day after the authored
dailies run out, so it does not change the beta; move the date if you want it
sooner.

## Not in the score yet

Stunt points are shown (callouts on landing, a running total in the drive HUD, a
line in the debrief) but are not part of `Score()`. The server rebuilds every
leaderboard score from level, samples and time (`CheckRun`) and cannot replay a
stunt.

## Known issue: lean-pumping

Leaning in the air adds spin with no energy cost, and landing with spin makes
the rover rebound for 2 to 3 seconds, even on flat ground at the start line. A
bot that leans in the air hops along a whole level and earns about 1,900 points
against 200 for plain driving. The landing-only and new-ground rules slow this
but do not stop it. Stunt points must stay out of the score until the rebound is
fixed (see the ticket on MAR-87).

## Tests

`game/stunts_test.go`: rules on hand-posed rovers (flips, airtime, wheelies,
hopping on the spot, repeats, unlanded flips), plus real-physics throws.
`game/ramps_test.go`: ramp shape, file validation, crevasse clearance, and a
real-physics drive of Kicker Flats (reachable, not free). Run:
`go test ./game -run 'Stunt|Ramp|RealPhysics'`.

## Analytics

`hm_run_stunts` once per finished drive that earned points: level, body,
points, count, backflips, frontflips, wheelie_s, best_air.

## Still to build

- A marker on the ramp and an upcoming-jump hint (MAR-86).
- Fix the landing rebound, then count stunts in the score (MAR-87).
- More ramp levels, and ramps on the moons, where air is long.
