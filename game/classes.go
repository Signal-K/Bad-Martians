// Package game is Bad Martians itself, built on engine + physics. It holds no
// rendering or platform code: scenes are GameObjects that draw through
// engine.Renderer and talk to the outside world through engine.Host.
package game

import (
	"github.com/signal-k/bad-martians/engine"
	"github.com/signal-k/bad-martians/physics"
)

// Class is a terrain class from the AI4Mars labelling scheme.
type Class uint8

const (
	Soil Class = iota
	Bedrock
	Sand
	BigRock
	NumClasses
)

type ClassInfo struct {
	Key   string
	Name  string
	Hint  string
	Color engine.Color
	Mat   physics.Material
}

var Classes = [NumClasses]ClassInfo{
	Soil:    {"soil", "Soil", "Loose regolith. Decent grip.", engine.RGB(176, 108, 66), physics.Material{Friction: 0.85}},
	Bedrock: {"bedrock", "Bedrock", "Flat hard rock. Best grip.", engine.RGB(120, 104, 100), physics.Material{Friction: 1.1}},
	Sand:    {"sand", "Sand", "Soft drifts. Wheels sink and slip.", engine.RGB(222, 168, 100), physics.Material{Friction: 0.5, Drag: 1.4}},
	BigRock: {"big_rock", "Big rock", "Boulder field. Bumpy.", engine.RGB(86, 70, 70), physics.Material{Friction: 0.9, Drag: 0.1}},
}

func (c Class) Info() ClassInfo { return Classes[c] }

// ClassFromKey parses an AI4Mars class key; ok is false for unknown keys.
func ClassFromKey(k string) (Class, bool) {
	for i, ci := range Classes {
		if ci.Key == k {
			return Class(i), true
		}
	}
	return 0, false
}
