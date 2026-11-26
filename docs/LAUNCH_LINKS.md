# Launch links (MAR-64)

One link per channel. Post only these, never the bare URL, so every player is
attributed to the channel that brought them. Campaign is `beta1`.

| Channel | Medium | Link |
| --- | --- | --- |
| Product Hunt | post | `https://martians.playlandnam.space/?utm_source=producthunt&utm_medium=post&utm_campaign=beta1` |
| Reddit | post | `https://martians.playlandnam.space/?utm_source=reddit&utm_medium=post&utm_campaign=beta1` |
| Discord | post | `https://martians.playlandnam.space/?utm_source=discord&utm_medium=post&utm_campaign=beta1` |
| X | post | `https://martians.playlandnam.space/?utm_source=x&utm_medium=post&utm_campaign=beta1` |
| Indie Hackers | post | `https://martians.playlandnam.space/?utm_source=indiehackers&utm_medium=post&utm_campaign=beta1` |
| Hacker News | post | `https://martians.playlandnam.space/?utm_source=hackernews&utm_medium=post&utm_campaign=beta1` |
| Star Sailors players | landnam | `https://martians.playlandnam.space/?utm_source=starsailors&utm_medium=landnam&utm_campaign=beta1` |
| Landnam players | landnam | `https://martians.playlandnam.space/?utm_source=landnam&utm_medium=landnam&utm_campaign=beta1` |
| Email and newsletter | email | `https://martians.playlandnam.space/?utm_source=newsletter&utm_medium=email&utm_campaign=beta1` |
| Friends, sent by hand | friend | `https://martians.playlandnam.space/?utm_source=friend&utm_medium=friend&utm_campaign=beta1` |

Medium is one of `post`, `email`, `friend`, `landnam`. To add a channel, copy a
row and change `utm_source` (lowercase, no spaces).

## Reading it in PostHog

The page view carries `utm_source`, `utm_medium` and `utm_campaign`. The game
also stamps the first visit's values on every later event, so a funnel
(`hm_scene` → `hm_run_finished` with `won`) can be broken down by `utm_source`
to show which channel produces players who win.

Visits from an automation or headless browser are ignored by PostHog. Check a
link with a normal browser, then look for the source under the event's
properties.
