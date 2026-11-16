# Expedition playtest

Seed 1, 160 generated rovers, 2 pilots (full, steady), 300 expeditions per player type. Shipped economy: start 36, par +2, salvage true.

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
| 16 | 0 | planner | 100% | 0.00 | 0.00 | 8.00 | 24.0 | 0 / 0 / 0 | 3606 |
| 16 | 0 | racer | 100% | 0.00 | 0.00 | 12.00 | 28.0 | 0 / 0 / 0 | 3736 |
| 16 | 0 | tinkerer | 55% | 0.49 | 0.00 | 3.77 | 21.9 | 106 / 30 / 0 | 1940 |
| 16 | 2 | planner | 100% | 0.00 | 2.00 | 8.00 | 28.0 | 0 / 0 / 0 | 3606 |
| 16 | 2 | racer | 100% | 0.00 | 2.00 | 12.00 | 32.0 | 0 / 0 / 0 | 3736 |
| 16 | 2 | tinkerer | 56% | 0.48 | 0.24 | 3.77 | 22.5 | 106 / 27 / 0 | 1971 |
| 16 | 4 | planner | 100% | 0.00 | 2.00 | 8.00 | 32.0 | 0 / 0 / 0 | 3606 |
| 16 | 4 | racer | 100% | 0.00 | 2.00 | 12.00 | 36.0 | 0 / 0 / 0 | 3736 |
| 16 | 4 | tinkerer | 56% | 0.46 | 0.24 | 3.76 | 23.6 | 106 / 25 / 0 | 1989 |
| 20 | 0 | planner | 100% | 0.00 | 0.00 | 8.00 | 28.0 | 0 / 0 / 0 | 3606 |
| 20 | 0 | racer | 100% | 0.00 | 0.00 | 12.00 | 32.0 | 0 / 0 / 0 | 4040 |
| 20 | 0 | tinkerer | 68% | 0.49 | 0.00 | 4.63 | 24.0 | 69 / 27 / 0 | 2447 |
| 20 | 2 | planner | 100% | 0.00 | 2.00 | 8.00 | 32.0 | 0 / 0 / 0 | 3606 |
| 20 | 2 | racer | 100% | 0.00 | 2.00 | 12.00 | 36.0 | 0 / 0 / 0 | 4040 |
| 20 | 2 | tinkerer | 69% | 0.49 | 0.44 | 4.77 | 25.3 | 69 / 24 / 0 | 2467 |
| 20 | 4 | planner | 100% | 0.00 | 2.00 | 8.00 | 36.0 | 0 / 0 / 0 | 3606 |
| 20 | 4 | racer | 100% | 0.00 | 2.00 | 12.00 | 40.0 | 0 / 0 / 0 | 4040 |
| 20 | 4 | tinkerer | 69% | 0.48 | 0.45 | 4.74 | 26.7 | 69 / 23 / 0 | 2483 |
| 24 | 0 | planner | 100% | 0.00 | 0.00 | 8.00 | 32.0 | 0 / 0 / 0 | 3606 |
| 24 | 0 | racer | 100% | 0.00 | 0.00 | 6.00 | 30.0 | 0 / 0 / 0 | 4055 |
| 24 | 0 | tinkerer | 81% | 0.57 | 0.00 | 5.49 | 25.5 | 44 / 13 / 0 | 2863 |
| 24 | 2 | planner | 100% | 0.00 | 2.00 | 8.00 | 36.0 | 0 / 0 / 0 | 3606 |
| 24 | 2 | racer | 100% | 0.00 | 2.00 | 6.00 | 34.0 | 0 / 0 / 0 | 4055 |
| 24 | 2 | tinkerer | 80% | 0.57 | 0.63 | 5.41 | 27.0 | 44 / 15 / 0 | 2844 |
| 24 | 4 | planner | 100% | 0.00 | 2.00 | 8.00 | 40.0 | 0 / 0 / 0 | 3606 |
| 24 | 4 | racer | 100% | 0.00 | 2.00 | 6.00 | 38.0 | 0 / 0 / 0 | 4055 |
| 24 | 4 | tinkerer | 81% | 0.56 | 0.65 | 5.39 | 28.6 | 44 / 12 / 0 | 2870 |
| 30 | 0 | planner | 100% | 0.00 | 0.00 | 8.00 | 38.0 | 0 / 0 / 0 | 3606 |
| 30 | 0 | racer | 100% | 0.00 | 0.00 | 6.00 | 36.0 | 0 / 0 / 0 | 4055 |
| 30 | 0 | tinkerer | 89% | 0.55 | 0.00 | 5.70 | 31.0 | 17 / 16 / 0 | 3168 |
| 30 | 2 | planner | 100% | 0.00 | 2.00 | 8.00 | 42.0 | 0 / 0 / 0 | 3606 |
| 30 | 2 | racer | 100% | 0.00 | 2.00 | 6.00 | 40.0 | 0 / 0 / 0 | 4055 |
| 30 | 2 | tinkerer | 89% | 0.54 | 0.73 | 5.67 | 32.5 | 17 / 15 / 0 | 3174 |
| 30 | 4 | planner | 100% | 0.00 | 2.00 | 8.00 | 46.0 | 0 / 0 / 0 | 3606 |
| 30 | 4 | racer | 100% | 0.00 | 2.00 | 6.00 | 44.0 | 0 / 0 / 0 | 4055 |
| 30 | 4 | tinkerer | 89% | 0.54 | 0.74 | 5.71 | 34.3 | 17 / 15 / 0 | 3176 |
| 36 | 0 | planner | 100% | 0.00 | 0.00 | 8.00 | 44.0 | 0 / 0 / 0 | 3606 |
| 36 | 0 | racer | 100% | 0.00 | 0.00 | 6.00 | 42.0 | 0 / 0 / 0 | 4055 |
| 36 | 0 | tinkerer | 95% | 0.56 | 0.00 | 6.27 | 35.9 | 11 / 5 / 0 | 3343 |
| 36 | 2 | planner | 100% | 0.00 | 2.00 | 8.00 | 48.0 | 0 / 0 / 0 | 3606 |
| 36 | 2 | racer | 100% | 0.00 | 2.00 | 6.00 | 46.0 | 0 / 0 / 0 | 4055 |
| 36 | 2 | tinkerer | 95% | 0.55 | 0.87 | 6.32 | 37.9 | 11 / 5 / 0 | 3346 |
| 36 | 4 | planner | 100% | 0.00 | 2.00 | 8.00 | 52.0 | 0 / 0 / 0 | 3606 |
| 36 | 4 | racer | 100% | 0.00 | 2.00 | 6.00 | 50.0 | 0 / 0 / 0 | 4055 |
| 36 | 4 | tinkerer | 95% | 0.54 | 0.88 | 6.31 | 39.6 | 11 / 4 / 0 | 3355 |

## Example expeditions (shipped economy)

**planner #1** (completed, score 3606)

- stage 1 Gale Warm-up: 8-credit rover, full pilot, won in 14.8s (par 15) -> 42 credits, +2 par, salvaged motor
- stage 2 Rolling Plains: 8-credit rover, full pilot, won in 15.9s (par 18) -> 48 credits, +2 par, salvaged motor
- stage 3 Small Lander: 8-credit rover, full pilot, won in 15.0s (par 20) -> 48 credits

**racer #1** (completed, score 4055)

- stage 1 Gale Warm-up: 24-credit rover, full pilot, won in 11.2s (par 15) -> 38 credits, +2 par
- stage 2 Rolling Plains: 24-credit rover, full pilot, won in 13.8s (par 18) -> 46 credits, +2 par, salvaged grip_motor
- stage 3 Small Lander: 14-credit rover, full pilot, won in 13.7s (par 20) -> 46 credits

**tinkerer #1** (completed, score 3376)

- stage 1 Gale Warm-up: 10-credit rover, steady pilot, won in 16.5s (par 15) -> 40 credits, salvaged motor
- stage 2 Rolling Plains: 19-credit rover, full pilot, won in 22.2s (par 18) -> 40 credits
- stage 3 Small Lander: 14-credit rover, steady pilot, won in 18.7s (par 20) -> 40 credits

_Generated by `go run ./cmd/playtest` in 33s._
