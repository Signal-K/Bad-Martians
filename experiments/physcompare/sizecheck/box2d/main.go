package main

import box2d "github.com/ByteArena/box2d"

func main() {
	w := box2d.MakeB2World(box2d.MakeB2Vec2(0, -1))
	w.Step(1.0/60, 8, 3)
	println(w.GetBodyCount())
}
