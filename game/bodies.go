package game

import (
	"math"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/physics"
)

// Body is the world a level is set on (KES-490): Mars, or one of its moons.
// It sets the gravity the drive runs under, the sky, and a tint over the
// ground's colours. Mars is the default and leaves everything as it was.
type Body struct {
	Key, Name string
	// Gravity in m/s². The moons' real gravity (millimetres per second²) would
	// make a rover drift for minutes after one bump, so they are stylised: low
	// enough that hops are long and grip is light, high enough to drive.
	Gravity float64
	// Ground tint: every ground colour is mixed this far toward Tint.
	Tint    engine.Color
	TintMix float64
	// Moon skies are black with stars and Mars hanging over the horizon.
	Airless  bool
	MarsSize float64 // pixel radius of Mars in a moon's sky
	Ridges   [3]engine.Color
}

// Bodies lists the worlds a level may declare in its "body" field.
var Bodies = map[string]*Body{
	"mars": {Key: "mars", Name: "Mars", Gravity: 3.71},
	// Phobos is very dark (albedo about 0.07), grey-brown, grooved and cratered.
	"phobos": {Key: "phobos", Name: "Phobos", Gravity: 1.6, Tint: engine.RGB(92, 82, 76), TintMix: 0.72,
		Airless: true, MarsSize: 120, Ridges: [3]engine.Color{engine.RGB(70, 64, 62), engine.RGB(58, 52, 50), engine.RGB(46, 41, 40)}},
	// Deimos is smaller and smoother, its craters filled with pale dust.
	"deimos": {Key: "deimos", Name: "Deimos", Gravity: 1.15, Tint: engine.RGB(148, 132, 116), TintMix: 0.62,
		Airless: true, MarsSize: 54, Ridges: [3]engine.Color{engine.RGB(112, 102, 94), engine.RGB(96, 87, 80), engine.RGB(80, 72, 66)}},
}

// BodyOf returns the level's body, Mars when it declares none.
func BodyOf(d LevelDef) *Body {
	if b, ok := Bodies[d.Body]; ok {
		return b
	}
	return Bodies["mars"]
}

// NewWorldFor is physics.NewWorld under the level's gravity.
func NewWorldFor(d LevelDef) *physics.World {
	w := physics.NewWorld()
	w.Gravity = physics.V(0, -BodyOf(d).Gravity)
	return w
}

// ground mixes a ground colour toward the body's tint.
func (b *Body) ground(c engine.Color) engine.Color {
	if b == nil || b.TintMix == 0 {
		return c
	}
	return lerpColor(c, b.Tint, b.TintMix)
}

// drawMoonSky paints an airless sky: black, a field of fixed stars, a hard
// white sun, Mars hanging large over the horizon, and grey ridges with no haze.
func drawMoonSky(r engine.Renderer, camX float64, b *Body) {
	r.Rect(0, 0, ScreenW, ScreenH, engine.RGB(8, 8, 14), true)
	rg := newRng(4909)
	for i := 0; i < 90; i++ {
		x := math.Mod(rg.float()*ScreenW*1.5-camX*0.05+ScreenW*3, ScreenW*1.5)
		r.Circle(engine.V(x, rg.float()*ScreenH*0.6), 0.6+rg.float()*1.1, engine.RGB(230, 232, 244).Alpha(uint8(110+rg.intn(140))), true)
	}
	sun := engine.V(880-camX*0.4, 96)
	r.Circle(sun, 18, engine.RGB(255, 254, 246).Alpha(60), true)
	r.Circle(sun, 11, engine.RGB(255, 254, 246), true)
	// Mars: a rust disc, lit from the sun's side, with a pale cap and dark Syrtis.
	mars := engine.V(300-camX*0.15, 300)
	R := b.MarsSize
	r.Circle(mars, R*1.08, engine.RGB(214, 120, 84).Alpha(26), true)
	r.Circle(mars, R, engine.RGB(176, 88, 56), true)
	r.Circle(mars.Add(engine.V(R*0.18, -R*0.14)), R*0.82, engine.RGB(204, 112, 72), true)
	r.Circle(mars.Add(engine.V(-R*0.2, R*0.1)), R*0.22, engine.RGB(132, 66, 48), true)
	r.Circle(mars.Add(engine.V(R*0.05, -R*0.8)), R*0.2, engine.RGB(238, 228, 220), true)

	ridge(r, camX, 0.06, func(wx float64) float64 {
		return 340 + 26*math.Sin(wx*0.008) + 14*math.Sin(wx*0.021+1)
	}, b.Ridges[0])
	ridge(r, camX, 0.15, func(wx float64) float64 {
		return 366 + 18*math.Sin(wx*0.013+2) + 9*math.Sin(wx*0.031)
	}, b.Ridges[1])
	ridge(r, camX, 0.35, func(wx float64) float64 {
		return 396 + 14*math.Sin(wx*0.019) + 8*math.Sin(wx*0.043+1)
	}, b.Ridges[2])
}
