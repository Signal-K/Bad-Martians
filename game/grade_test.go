package game

import "testing"

func TestGradeFollowsTheScore(t *testing.T) {
	l := GetLevel(1)
	g := func(won bool, stars, samples int, tm float64) string {
		o := Fell
		if won {
			o = Won
		}
		return GradeFor(l, &ResultArg{Outcome: o, Stars: stars, Score: ScoreFor(l, won, samples, tm)}).Letter
	}
	if got := g(false, 0, 0, 10); got != "F" {
		t.Fatalf("failed run graded %s", got)
	}
	if got := g(true, 3, 3, l.Par*0.5); got != "S" {
		t.Fatalf("perfect run graded %s", got)
	}
	if got := g(true, 1, 0, l.Par*3); got != "D" {
		t.Fatalf("slow empty-handed finish graded %s", got)
	}
	order := "SABCD"
	prev := -1
	for _, tm := range []float64{0.5, 1, 1.4, 1.8, 3} {
		i := indexOf(order, g(true, 2, 3, l.Par*tm))
		if i < prev {
			t.Fatalf("slower run graded higher at %.1fx par", tm)
		}
		prev = i
	}
}

func indexOf(s, c string) int {
	for i := range s {
		if s[i:i+1] == c {
			return i
		}
	}
	return -1
}
