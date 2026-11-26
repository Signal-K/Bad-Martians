# Stunt engine

Source: Liam's design sketch, 26 Nov 2026 (Game Ideas PDF). `game/stunts.go`.

## Rules

| Stunt | Rule | Points |
| -- | -- | -- |
| Backflip / frontflip | 85% of a full turn in one jump (nose up = back, nose down = front); each further full turn pays again | 2 per flip |
| Wheelie | back wheel down, front wheel up, nose 20° to 69° up; pays at every whole second; a dip under 0.25 s does not end it | 2 per second |
| Airtime | every body off the ground for at least 1.5 s; pays on landing | 2 bonus |

A sketch point is 50 score points (`stuntPointScale`), so a flip reads as
`+100`, next to 150 for a sample.

Open choices, all constants at the top of `stunts.go`: the airtime threshold
(the sketch left it blank, 1.5 s is a guess), the 50x scale, and whether a
flip counts if the rover crashes on landing (today it does).

## Not in the score yet

Stunt points are shown (callouts over the rover, a running total in the drive
HUD, a line in the debrief) but are not part of `Score()`. The server rebuilds
every leaderboard score from level, samples and time (`CheckRun`) and cannot
replay a stunt, so adding them would either be rejected or open a forgery
hole. They join the score once runs can be verified.

## UI

Brief, near the rover, never over the controls: `Backflip! +100` floats up
and fades in 1.6 s; the running total sits under the progress bar and only
appears after the first trick. The sketch also asks for an upcoming-jump hint
and impact emphasis on the terrain layer; both are follow-up tickets.

## Analytics

`hm_run_stunts` once per finished drive that earned points: level, body,
points, count, backflips, frontflips, wheelie_s, best_air.

## Still to build

- Levels with jumps and ramps, so tricks are reachable without luck.
- Upcoming-jump hint and impact sound/shake.
- Stunt points in the score, after run verification.
- A stunt star or daily challenge, if the data shows players try tricks.
