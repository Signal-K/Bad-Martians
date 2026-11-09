package game

// Block is a ready-made group of parts the player can drop onto the grid in one
// go: the shapes a good rover is made of. Cells are relative, (0,0) is the
// block's bottom-left.
type Block struct {
	Name  string
	Idea  string // the concept the shape teaches, shown when it is picked
	Cells []Cell
}

var Blocks = []Block{
	{"Starter rover", "Starter rover: wheels at both ends, a motor wheel to drive, the mast on top.", StarterDesign().Cells},
	{"Triangle", "Triangle: its diagonal beam locks the shape, so it will not fold on bumps.",
		[]Cell{{0, 0, KFrame}, {1, 0, KFrame}, {0, 1, KFrame}}},
	{"Box", "Box: braced both ways, the stiffest frame. Good under a heavy mast.",
		[]Cell{{0, 0, KFrame}, {1, 0, KFrame}, {0, 1, KFrame}, {1, 1, KFrame}}},
	{"Beam", "Beam: a long base spreads the wheels out, so the rover tips less.",
		[]Cell{{0, 0, KFrame}, {1, 0, KFrame}, {2, 0, KFrame}}},
}

// Size is the block's width and height in cells.
func (b Block) Size() (w, h int) {
	for _, c := range b.Cells {
		w, h = max(w, c.X+1), max(h, c.Y+1)
	}
	return
}

func (b Block) Cost() int { d := Design{Cells: b.Cells}; return d.Cost() }

// Anchor is where a block dropped with the pointer on cell (cx, cy) lands: the
// pointer holds the block's bottom-middle, and the block is kept on the grid.
func (b Block) Anchor(cx, cy int) (x, y int) {
	w, h := b.Size()
	x = min(max(cx-(w-1)/2, 0), GridW-w)
	y = min(max(cy, 0), GridH-h)
	return
}

// Stamp places the block with its bottom-left at (x, y), replacing what is
// there, and returns the cells it filled.
func (b Block) Stamp(d *Design, x, y int) [][2]int {
	var out [][2]int
	for _, c := range b.Cells {
		d.Set(x+c.X, y+c.Y, c.K)
		out = append(out, [2]int{x + c.X, y + c.Y})
	}
	return out
}
