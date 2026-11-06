// Package engine is a small Unity/Godot-style runtime: GameObjects hold
// Components, Scenes own GameObjects and a physics World, and a Game drives
// scenes on a fixed timestep. It is renderer-agnostic: drawing goes through the
// Renderer interface, so the same game code runs under any host (Ebitengine
// today; a native Swift/Kotlin shell later).
package engine

import "math"

// Vec2 is a 2D vector. World space is metres, +Y up.
type Vec2 struct{ X, Y float64 }

func V(x, y float64) Vec2         { return Vec2{x, y} }
func (a Vec2) Add(b Vec2) Vec2    { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2    { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Mul(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) Dot(b Vec2) float64 { return a.X*b.X + a.Y*b.Y }
func (a Vec2) Len() float64       { return math.Hypot(a.X, a.Y) }
func (a Vec2) Perp() Vec2         { return Vec2{-a.Y, a.X} }
func (a Vec2) Norm() Vec2 {
	l := a.Len()
	if l < 1e-12 {
		return Vec2{}
	}
	return Vec2{a.X / l, a.Y / l}
}
