platform: linux/amd64, go1.27.0

### Settling on flat rock (12 s, no throttle)

| metric | Ours (PBD lattice) | Ours, 4x iterations | Box2D (rigid + welds) | Chipmunk (impulse lattice) |
|---|---|---|---|---|
| energy drift | 0.00 J / 3 s | 0.00 J / 3 s | 0.00 J / 3 s | 0.00 J / 3 s |
| rest jitter | 0.0 mm/s | 0.0 mm/s | 0.0 mm/s | 0.0 mm/s |
| settles | 0.3 s | 0.3 s | 0.3 s | 0.2 s |
| sink at rest | 0 mm | 0 mm | 0 mm | 0 mm |

### Soil slope climb (full throttle, 25 s)

| metric | Ours (PBD lattice) | Ours, 4x iterations | Box2D (rigid + welds) | Chipmunk (impulse lattice) |
|---|---|---|---|---|
| 20° climb to 40 m | no | no | 10.3 s | 9.6 s |
| steepest climb | 10° | 10° | 25° | 25° |

### Drop from 30 m, launched at 14 m/s

| metric | Ours (PBD lattice) | Ours, 4x iterations | Box2D (rigid + welds) | Chipmunk (impulse lattice) |
|---|---|---|---|---|
| ends on surface | yes | yes | yes | yes |
| peak speed | 16 m/s | 16 m/s | 23 m/s | 16 m/s |
| tunnelled steps | 0 | 0 | 0 | 0 |

### Drop from 150 m, launched at 14 m/s

| metric | Ours (PBD lattice) | Ours, 4x iterations | Box2D (rigid + welds) | Chipmunk (impulse lattice) |
|---|---|---|---|---|
| ends on surface | yes | yes | yes | yes |
| peak speed | 36 m/s | 29 m/s | 34 m/s | 34 m/s |
| tunnelled steps | 0 | 0 | 0 | 0 |

### Drop from 1000 m (about 86 m/s on impact)

| metric | Ours (PBD lattice) | Ours, 4x iterations | Box2D (rigid + welds) | Chipmunk (impulse lattice) |
|---|---|---|---|---|
| ends on surface | yes | yes | yes | yes |
| peak speed | 46 m/s | 46 m/s | 74 m/s | 72 m/s |
| tunnelled steps | 0 | 0 | 0 | 0 |

### Shipped levels (par is the 3-star time)

| metric | Ours (PBD lattice) | Ours, 4x iterations | Box2D (rigid + welds) | Chipmunk (impulse lattice) |
|---|---|---|---|---|
| L1 reference rover | 12.4 s (par 14) | 12.4 s (par 14) | 10.7 s (par 14) | 10.6 s (par 14) |
| L1 starter rover | 24.5 s (par 14) | 24.5 s (par 14) | 13.1 s (par 14) | 12.4 s (par 14) |
| L2 reference rover | 21.5 s (par 24) | 22.2 s (par 24) | 16.5 s (par 24) | 16.7 s (par 24) |
| L3 reference rover | 21.6 s (par 23) | 21.9 s (par 23) | 17.5 s (par 23) | 18.9 s (par 23) |
| L4 reference rover | 31.3 s (par 28) | 29.2 s (par 28) | stuck at 19 m | 24.0 s (par 28) |
| L5 reference rover | 32.3 s (par 34) | 30.8 s (par 34) | stuck at 61 m | stuck at 25 m |

### Long beam rigidity (10 m, 12 s at rest)

| metric | Ours (PBD lattice) | Ours, 4x iterations | Box2D (rigid + welds) | Chipmunk (impulse lattice) |
|---|---|---|---|---|
| beam parts | 8 | 8 | 8 | 8 |
| beam sag | 200 mm | 200 mm | 33 mm | 203 mm |

### Cost per step (this machine)

| metric | Ours (PBD lattice) | Ours, 4x iterations | Box2D (rigid + welds) | Chipmunk (impulse lattice) |
|---|---|---|---|---|
| 6-body step | 15 µs | 52 µs | 37 µs | 9 µs |
| 84-body sane | yes (moved 46 m) | yes (moved 47 m) | yes (moved 64 m) | yes (moved 67 m) |
| 84-body step | 325 µs | 1343 µs | 302 µs | 130 µs |

### Determinism (12° soil ramp, 20 s, 1200 steps)

| metric | Ours (PBD lattice) | Ours, 4x iterations | Box2D (rigid + welds) | Chipmunk (impulse lattice) |
|---|---|---|---|---|
| digest | 3b57f9e108deb869 | 911f84a811eff810 | 2fb05b8ceb6f6aa9 | d53ad1d9b20d68a9 |
| repeatable | bit-identical | bit-identical | bit-identical | bit-identical |

