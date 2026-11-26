# Roadmap

## Next release (pending player feedback)

- **Anchor and specialised parts.** Parts built for a particular body's physics.
- **New level types.** Rovers will not be the same sort of machine on every body.

## Ultra-low-mass bodies: a way forward

Phobos and Deimos are deliberately not realistic. Their real surface gravity is
about 0.006 m/s² and 0.003 m/s², so a rover would hop away at walking pace. Today
Phobos is 1.6 (like the Moon) with 0.8 grip, and Deimos is 1.15.

The physics is accurate for the parameters each body is given; the parameters
are stylised so the existing rover designs can still finish the levels.

The constants cannot simply be lowered. Lower gravity makes Deimos unclearable and
pushes `TestNoSingleDesignClearsEveryLevel` over its limit, because the current
parts and levels are tuned for a narrow range of gravity.

Goal: decide how to treat these bodies properly. Options to evaluate:

- Gameplay beyond rovers for ultra-low-mass bodies (hoppers, tethered or anchored
  craft, crawlers).
- Anchoring and grip parts, with levels designed around them.
- Retuning par times and reference designs together with any gravity change.
