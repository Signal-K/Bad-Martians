# Playtest roadmap: learn how people use it

We have a first version, a long wishlist, and no idea yet which of it matters.
This plan is for finding out how people actually play, so the next build comes
from what we see, including things we did not think of. It supersedes the
go/no-go framing in the Linear document "Beta pipeline and next-version goals"
(that document stays as the record of the first guess).

## What we want to learn

| Question | Where we look |
| -- | -- |
| What do people try to do that we did not design for? | Watched sessions, the "what did you try?" prompt, rover designs in `hm_runs` |
| Where do they stall, and what do they do next? | Tutorial and funnel events, watched sessions |
| Why do they play: win, tinker, show off, explore? | Playstyle groups (below), watched sessions |
| What do they ask for? | Survey "what would you most like next?", free text, what they say aloud |
| Which parts, places and tricks do they ignore? | `hm_build_launched` parts mix, `hm_run_stunts`, level win rates |
| Does the early game teach itself? | New-player funnel, `hm_tier_reached`, watched sessions with a child |

## Playstyles, the main output

Group players from behaviour, then describe each group in plain words: what
they did, what stopped them, what they asked for.

- **Racers**: low time, few samples, rarely scrape.
- **Collectors**: every sample, over par.
- **Tinkerers**: many builds per mission, varied parts.
- **Stunters**: stunt points above the median, or repeated flip attempts.
- **Bouncers**: quit in the first two minutes. The most important group.

## Ways of seeing, in order of value

1. **Watched sessions** (3 to 5, one with a child). Say only "play this".
   Note the first tap, first read, first "what do I do", first drive, first win,
   when they stop, and what they thought the goal was.
2. **Surprise log.** One line per surprise, in the Beta log, as it happens: a
   design nobody planned, a trick, a misunderstanding, an exploit.
3. **One free-text prompt** after a drive: "What did you try that didn't work?"
4. **Events** (PostHog dashboard "Haphazard Martians release"), for scale and to
   check what the sessions suggest: new-player funnel, tiers reached, drives
   and wins by mission, daily missions, return, stunts.
5. **Rover designs**, the richest data we have: look at what players build, not
   just whether they won.

## Wishlist as hypotheses

Each wishlist item says what we would have to see to justify building it.
Fill in the evidence column as it comes; add rows for things players show us.

| Wishlist item | Build it if we see | Evidence so far |
| -- | -- | -- |
| More parts, chassis, arms (MAR-21) | Builds converge on one design; players hit the credit limit and rebuild a lot | none yet |
| New locations | Players finish content quickly, daily win rate high | none yet |
| Friends and race ghosts (MAR-31) | Players return for dailies, check the leaderboard, ask "who beat me" | none yet |
| Story objectives (MAR-32, MAR-33) | Players ask what the point is, or drop out after the first win | none yet |
| Real-image review (MAR-26) | Players return and ask for purpose beyond the score | none yet |
| Stunt levels (MAR-85) | Players try flips and fail, or stunt points are a large share of play | none yet |
| Better onboarding | Bouncers are a large group; sessions show stalls before the first drive | none yet |

## Safety checks, not goals

If one of these happens, fix it before learning anything else: progress lost
on sync, a crash on load on a common phone, a mission that cannot be won, a
fake leaderboard entry visible to everyone, a production backend missing a
collection the game writes to.

## How many people

There is no player-count gate. Under about 30 players, treat patterns as
hints and say so. A pattern seen in three watched sessions and the events
together is worth acting on; a number alone is not.

## Read-out

At the end of Cycle 12, write one page: the playstyle groups and what each
needs, the surprises, the wishlist table with evidence filled in, and the one
or two things to build next, including anything not on the wishlist.
