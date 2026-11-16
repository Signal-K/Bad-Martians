# Expedition playtest

Seed 1, 160 generated rovers, 2 pilots (full, steady), 300 expeditions per player type. Shipped economy: start 24, par +2, salvage false.

## Stages

| Stage | Level | Par | Legal rovers | Win with a bot | Under par | Cheapest win | Cheapest under par | Fastest |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Gale Warm-up | 15s | 160 | 115 (72%) | 45 | 8 cr | 8 cr | 11.2s |
| 2 | Rolling Plains | 18s | 160 | 130 (81%) | 84 | 8 cr | 8 cr | 13.8s |
| 3 | Small Lander | 20s | 33 | 33 (100%) | 29 | 8 cr | 8 cr | 13.7s |

Minimum valid rover: 6 credits.

## Player types

- **planner** builds the cheapest rover that works, keeping a survivor that still works.
- **racer** builds the fastest rover it can afford, to beat par.
- **tinkerer** builds at random; after a crash it has an even chance of copying a rover that works.

## Economies

| Start | Par + | Player | Complete | Crashes / run | Par awards / run | Salvage credits / run | Credits left (winners) | Failed at 1 / 2 / 3 | Avg score |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 16 | 0 | planner | 100% | 0.00 | 0.00 | 0.00 | 16.0 | 0 / 0 / 0 | 3606 |
| 16 | 0 | racer | 100% | 0.00 | 0.00 | 0.00 | 16.0 | 0 / 0 / 0 | 3736 |
| 16 | 0 | tinkerer | 48% | 0.53 | 0.00 | 0.00 | 15.8 | 106 / 49 / 0 | 1794 |
| 16 | 2 | planner | 100% | 0.00 | 2.00 | 0.00 | 20.0 | 0 / 0 / 0 | 3606 |
| 16 | 2 | racer | 100% | 0.00 | 2.00 | 0.00 | 20.0 | 0 / 0 / 0 | 3736 |
| 16 | 2 | tinkerer | 49% | 0.52 | 0.18 | 0.00 | 16.5 | 106 / 48 / 0 | 1805 |
| 16 | 4 | planner | 100% | 0.00 | 2.00 | 0.00 | 24.0 | 0 / 0 / 0 | 3606 |
| 16 | 4 | racer | 100% | 0.00 | 2.00 | 0.00 | 24.0 | 0 / 0 / 0 | 3736 |
| 16 | 4 | tinkerer | 49% | 0.53 | 0.19 | 0.00 | 17.1 | 106 / 46 / 0 | 1819 |
| 20 | 0 | planner | 100% | 0.00 | 0.00 | 0.00 | 20.0 | 0 / 0 / 0 | 3606 |
| 20 | 0 | racer | 100% | 0.00 | 0.00 | 0.00 | 20.0 | 0 / 0 / 0 | 4040 |
| 20 | 0 | tinkerer | 66% | 0.47 | 0.00 | 0.00 | 18.1 | 69 / 32 / 0 | 2394 |
| 20 | 2 | planner | 100% | 0.00 | 2.00 | 0.00 | 24.0 | 0 / 0 / 0 | 3606 |
| 20 | 2 | racer | 100% | 0.00 | 2.00 | 0.00 | 24.0 | 0 / 0 / 0 | 4040 |
| 20 | 2 | tinkerer | 67% | 0.47 | 0.38 | 0.00 | 19.2 | 69 / 31 / 0 | 2395 |
| 20 | 4 | planner | 100% | 0.00 | 2.00 | 0.00 | 28.0 | 0 / 0 / 0 | 3606 |
| 20 | 4 | racer | 100% | 0.00 | 2.00 | 0.00 | 28.0 | 0 / 0 / 0 | 4040 |
| 20 | 4 | tinkerer | 66% | 0.49 | 0.41 | 0.00 | 20.1 | 69 / 32 / 0 | 2397 |
| 24 | 0 | planner | 100% | 0.00 | 0.00 | 0.00 | 24.0 | 0 / 0 / 0 | 3606 |
| 24 | 0 | racer | 100% | 0.00 | 0.00 | 0.00 | 24.0 | 0 / 0 / 0 | 4055 |
| 24 | 0 | tinkerer | 78% | 0.57 | 0.00 | 0.00 | 19.1 | 44 / 22 / 0 | 2783 |
| 24 | 2 | planner | 100% | 0.00 | 2.00 | 0.00 | 28.0 | 0 / 0 / 0 | 3606 |
| 24 | 2 | racer | 100% | 0.00 | 2.00 | 0.00 | 28.0 | 0 / 0 / 0 | 4055 |
| 24 | 2 | tinkerer | 79% | 0.54 | 0.59 | 0.00 | 20.9 | 44 / 19 / 0 | 2804 |
| 24 | 4 | planner | 100% | 0.00 | 2.00 | 0.00 | 32.0 | 0 / 0 / 0 | 3606 |
| 24 | 4 | racer | 100% | 0.00 | 2.00 | 0.00 | 32.0 | 0 / 0 / 0 | 4055 |
| 24 | 4 | tinkerer | 79% | 0.55 | 0.62 | 0.00 | 22.3 | 44 / 18 / 0 | 2816 |
| 30 | 0 | planner | 100% | 0.00 | 0.00 | 0.00 | 30.0 | 0 / 0 / 0 | 3606 |
| 30 | 0 | racer | 100% | 0.00 | 0.00 | 0.00 | 30.0 | 0 / 0 / 0 | 4055 |
| 30 | 0 | tinkerer | 87% | 0.55 | 0.00 | 0.00 | 25.0 | 17 / 21 / 0 | 3118 |
| 30 | 2 | planner | 100% | 0.00 | 2.00 | 0.00 | 34.0 | 0 / 0 / 0 | 3606 |
| 30 | 2 | racer | 100% | 0.00 | 2.00 | 0.00 | 34.0 | 0 / 0 / 0 | 4055 |
| 30 | 2 | tinkerer | 88% | 0.55 | 0.71 | 0.00 | 26.5 | 17 / 20 / 0 | 3123 |
| 30 | 4 | planner | 100% | 0.00 | 2.00 | 0.00 | 38.0 | 0 / 0 / 0 | 3606 |
| 30 | 4 | racer | 100% | 0.00 | 2.00 | 0.00 | 38.0 | 0 / 0 / 0 | 4055 |
| 30 | 4 | tinkerer | 88% | 0.54 | 0.72 | 0.00 | 28.3 | 17 / 19 / 0 | 3132 |
| 36 | 0 | planner | 100% | 0.00 | 0.00 | 0.00 | 36.0 | 0 / 0 / 0 | 3606 |
| 36 | 0 | racer | 100% | 0.00 | 0.00 | 0.00 | 36.0 | 0 / 0 / 0 | 4055 |
| 36 | 0 | tinkerer | 93% | 0.55 | 0.00 | 0.00 | 29.8 | 11 / 9 / 0 | 3312 |
| 36 | 2 | planner | 100% | 0.00 | 2.00 | 0.00 | 40.0 | 0 / 0 / 0 | 3606 |
| 36 | 2 | racer | 100% | 0.00 | 2.00 | 0.00 | 40.0 | 0 / 0 / 0 | 4055 |
| 36 | 2 | tinkerer | 94% | 0.54 | 0.85 | 0.00 | 31.6 | 11 / 8 / 0 | 3318 |
| 36 | 4 | planner | 100% | 0.00 | 2.00 | 0.00 | 44.0 | 0 / 0 / 0 | 3606 |
| 36 | 4 | racer | 100% | 0.00 | 2.00 | 0.00 | 44.0 | 0 / 0 / 0 | 4055 |
| 36 | 4 | tinkerer | 94% | 0.55 | 0.85 | 0.00 | 33.2 | 11 / 7 / 0 | 3326 |

## Example expeditions (shipped economy)

**planner #1** (completed, score 3606)

- stage 1 Gale Warm-up: 8-credit rover, full pilot, won in 14.8s (par 15) -> 26 credits, +2 par
- stage 2 Rolling Plains: 8-credit rover, full pilot, won in 15.9s (par 18) -> 28 credits, +2 par
- stage 3 Small Lander: 8-credit rover, full pilot, won in 15.0s (par 20) -> 28 credits

**racer #1** (completed, score 4055)

- stage 1 Gale Warm-up: 24-credit rover, full pilot, won in 11.2s (par 15) -> 26 credits, +2 par
- stage 2 Rolling Plains: 24-credit rover, full pilot, won in 13.8s (par 18) -> 28 credits, +2 par
- stage 3 Small Lander: 14-credit rover, full pilot, won in 13.7s (par 20) -> 28 credits

**tinkerer #1** (failed, score 0)

- stage 1 Gale Warm-up: 17-credit rover, steady pilot, stuck, scrapped in 4.0s (par 15) -> 7 credits
- stage 1: nothing affordable with 7 credits

_Generated by `go run ./cmd/playtest` in 32s._
