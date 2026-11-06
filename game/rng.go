package game

// rng is splitmix64: tiny, deterministic, identical on every platform.
type rng struct{ s uint64 }

func newRng(seed uint64) *rng { return &rng{seed*0x9E3779B97F4A7C15 + 1} }
func (r *rng) next() uint64 {
	r.s += 0x9E3779B97F4A7C15
	z := r.s
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}
func (r *rng) float() float64              { return float64(r.next()>>11) / (1 << 53) }
func (r *rng) rangef(a, b float64) float64 { return a + (b-a)*r.float() }
func (r *rng) intn(n int) int              { return int(r.next() % uint64(n)) }
