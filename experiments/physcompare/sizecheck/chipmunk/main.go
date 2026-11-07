package main

import "github.com/jakecoffman/cp/v2"

func main() { s := cp.NewSpace(); s.Step(1.0 / 60); println(s.Iterations) }
