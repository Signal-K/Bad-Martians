package main

import (
	"github.com/signal-k/haphazard-martians/physics"
)

func main() { w := physics.NewWorld(); w.Step(1.0 / 60); println(len(w.Bodies)) }
