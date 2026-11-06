package engine

// Input is a per-frame snapshot supplied by the host. A single pointer
// (mouse or first touch) is enough for this game; keys mirror it for desktop.
type Input struct {
	Pointer Vec2 // virtual-screen pixels
	Down    bool
	Left    bool
	Right   bool

	// Derived by Game.Tick from successive snapshots.
	Pressed  bool
	Released bool
}
