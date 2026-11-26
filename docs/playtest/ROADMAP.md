# Playtest roadmap and readiness

The decision framework (bars, sample size, what to build for each result)
lives in Linear: "Beta pipeline and next-version goals". This file adds what
the 26 Nov playtest feedback requires, and how PostHog answers it.

## Playtest steps

1. **Onboarding fix** (shipped): first open goes to the bay, systems unlock by
   wins, word budgets per screen.
2. **Beta week**: bug fixes only. Read the dashboards daily.
3. **Watched sessions**: three to five newcomers, at least one child, playing
   unprompted. Record where they stall (see "Session script").
4. **Read-out** (end of Cycle 12): compare to the bars below, pick version 2.

## Onboarding and complexity measures

| Question | Event | Bar (proposal) |
| -- | -- | -- |
| Do new players start building straight away? | `hm_first_open` then `hm_tutorial_step` | 80% do tutorial step 1 within 60 s |
| Where does the tutorial lose people? | `hm_tutorial_step` by step, `hm_tutorial_done` skipped | no step loses more than 20% |
| Do they reach a first drive? | `hm_build_launched` after `hm_first_open` | 60% within 5 min |
| Do they reach a first win? | `hm_run_finished` won | 40% (existing bar) |
| Is pacing right? | `hm_tier_reached` tier 1 and 2 | 60% of first-win players reach tier 2 |
| Did the text cut work? | before/after funnel on 26 Nov | first-drive rate up |

## Stunt measures

| Question | Event | Bar (proposal) |
| -- | -- | -- |
| Do players try tricks? | `hm_run_stunts` | 25% of drives with a win also earn stunt points |
| Which tricks? | `hm_run_stunts` backflips, frontflips, wheelie_s, best_air | one trick is not 90% of points |
| Does it replace the goal? | win rate with stunts vs without | win rate does not fall |

## Playstyles

Group players from the data, not by guessing:
- **Racers**: low time, few samples, rarely scrape.
- **Collectors**: all samples, over par.
- **Tinkerers**: many `hm_build_launched` per mission, parts mix varies.
- **Stunters**: stunt points above the median.
Each group's retention and difficulty tell us which version 2 bet it needs.

## When can version 2 start?

All of these, with at least 30 players:
1. The existing bars in the Linear doc are met or consciously overridden.
2. The onboarding bars above are met.
3. Watched sessions show no new blocking confusion.
4. No stop condition (lost progress, crash on load, unwinnable mission, fake leaderboard entry) is open.
If we reach 30 players and miss a bar, the answer is a fix, not version 2.

## Session script (watched play)

Say only "play this". Do not explain. Note: first thing they tap, first thing
they read, first time they say "what do I do", first drive, first win, when
they stop. Ask afterwards: what did you think the goal was?
