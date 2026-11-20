# Playtest harness

`make playtest` (or `go run ./cmd/playtest`) plays Haphazard Martians with no
one at the controls and writes `expedition-economy.md`.

What it does:

1. Generates a seeded pool of rovers, from four-part landers to long two-row
   builds (`game.GenerateDesigns`).
2. Drives every legal rover on each expedition stage with the real physics
   (`game.Run`) and two bot pilots: `full` holds the stick forward, `steady`
   drives at 70% and leans against tipping. A stuck rover is scrapped, the way
   a player pressing Restart loses it.
3. Plays seeded expeditions through the real `Expedition.Finish` and
   `Expedition.Salvage` with three player types:
   - **planner**: builds the cheapest rover that works.
   - **racer**: builds the fastest rover it can afford.
   - **tinkerer**: builds at random, and after a crash has an even chance of
     copying a rover that works.

Flags: `-start 20,24,36` and `-par 0,2,4` sweep the starting credits and the
under-par reward, `-no-salvage` turns off salvage credits, `-n` sets the
number of expeditions, `-designs` the pool size, and `-seed` everything else.
The same flags always give the same report.

Limits: bots are not people. The planner knows every result in advance, and
the tinkerer's learning rate is a guess. Use it to compare numbers against
each other, not to predict real completion rates; once real runs reach
PostHog (KES-505), calibrate the tinkerer's crash rate against them.
