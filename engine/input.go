package engine

// Input is a per-frame snapshot supplied by the host. A single pointer
// (mouse or first touch) is enough for this game; keys mirror it for desktop.
type Input struct {
	Pointer Vec2 // virtual-screen pixels
	Down    bool
	Left    bool
	Right   bool
	Up      bool
	DownKey bool // down arrow (Down is the pointer button)

	// Touches is every finger on the screen (or the held mouse, ID -1), so a
	// control can follow its own finger while others tap buttons.
	Touches []Touch

	// Editor commands, sent by the host as one-frame edges.
	ToggleEdit, Pause, Step, Delete bool

	// Text entry, sent by the host. Typed holds the characters typed this
	// frame; Backspace and Enter are one-frame edges. Replace, when Replacing
	// is set, is a whole string from a host prompt (a phone's text box) that
	// replaces the field being edited. Game.Tick latches all of these so each
	// reaches exactly one fixed step.
	Typed     []rune
	Backspace bool
	Enter     bool
	Replace   string
	Replacing bool

	// Derived by Game.Tick from successive snapshots.
	Pressed  bool
	Released bool
}

// Touch is one finger (or the held mouse button) in virtual-screen pixels.
type Touch struct {
	ID  int
	Pos Vec2
}

// MouseTouch is the Touch ID the held mouse button reports.
const MouseTouch = -1

// TouchByID finds a touch, or reports false once that finger has lifted.
func (in Input) TouchByID(id int) (Vec2, bool) {
	for _, t := range in.Touches {
		if t.ID == id {
			return t.Pos, true
		}
	}
	return Vec2{}, false
}
