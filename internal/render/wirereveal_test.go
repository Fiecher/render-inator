package render

import (
	stdmath "math"
	"testing"
	"time"

	"render-inator/internal/camera"
	m "render-inator/internal/math"
)

func TestWireRevealGraph(t *testing.T) {
	msh := buildSphere(16, 16)
	var r wireReveal
	r.begin(msh)

	if !r.active {
		t.Fatal("reveal not active after begin")
	}
	if len(r.edges) == 0 {
		t.Fatal("no edges built")
	}
	if len(r.dist) >= len(msh.Verts) {
		t.Fatalf("welding did nothing: nodes=%d verts=%d", len(r.dist), len(msh.Verts))
	}

	inf := float32(stdmath.MaxFloat32)
	zeros := 0
	for i, d := range r.dist {
		if d != d {
			t.Fatalf("node %d NaN dist", i)
		}
		if d == inf {
			t.Fatalf("node %d unreached (disconnected seeding failed)", i)
		}
		if d < 0 {
			t.Fatalf("node %d negative dist %g", i, d)
		}
		if d == 0 {
			zeros++
		}
		if d > r.maxDist {
			t.Fatalf("node %d dist %g exceeds maxDist %g", i, d, r.maxDist)
		}
	}
	if zeros != 1 {
		t.Fatalf("connected sphere must have exactly one seed, got %d", zeros)
	}
	if r.maxDist <= 0 {
		t.Fatalf("maxDist must be positive, got %g", r.maxDist)
	}

	for i := range r.edges {
		e := &r.edges[i]
		if e.a == e.b {
			t.Fatalf("edge %d is a self-loop", i)
		}
		if e.invLen <= 0 || e.invLen != e.invLen {
			t.Fatalf("edge %d bad invLen %g", i, e.invLen)
		}
		if e.ra < 0 || int(e.ra) >= len(msh.Verts) || e.rb < 0 || int(e.rb) >= len(msh.Verts) {
			t.Fatalf("edge %d representative index out of range", i)
		}
	}
}

func TestWireRevealRebuildOnMeshSwap(t *testing.T) {
	a := buildSphere(8, 8)
	b := buildSphere(12, 12)
	var r wireReveal
	r.begin(a)
	na := len(r.edges)
	if r.builtFor != a {
		t.Fatal("builtFor not set to first mesh")
	}
	r.begin(b)
	if r.builtFor != b {
		t.Fatal("builtFor not updated on mesh swap")
	}
	if len(r.edges) == na {
		t.Fatalf("edge graph not rebuilt: still %d edges", na)
	}
}

func TestWireRevealDrawProgresses(t *testing.T) {
	msh := buildSphere(24, 24)
	p := NewPipeline(128, 128)
	cam := camera.New(m.Vec3{}, 2.6, 60*stdmath.Pi/180, 1)
	cam.Snap()

	view := cam.View()
	vp := cam.Projection().Mul(view)
	p.projectVerts(msh, vp, view)

	base := time.Now()
	p.reveal.begin(msh)
	p.reveal.start = base
	p.reveal.pending = false

	count := func(dtSec float64) int {
		p.clear()
		p.reveal.draw(p, base.Add(time.Duration(dtSec*float64(time.Second))))
		n := 0
		px := p.Pixels()
		for i := 0; i+3 < len(px); i += 4 {
			if px[i] == wireColor.R && px[i+1] == wireColor.G && px[i+2] == wireColor.B {
				n++
			}
		}
		return n
	}

	early := count(0.05)
	mid := count(0.40)
	full := count(1.00)

	if full == 0 {
		t.Fatal("no wire pixels at completion")
	}
	if !(early < mid && mid <= full) {
		t.Fatalf("reveal not growing: early=%d mid=%d full=%d", early, mid, full)
	}
	if early >= full {
		t.Fatalf("early reveal must be a strict subset of full: early=%d full=%d", early, full)
	}
}

func TestWireRevealClockAnchorsOnFirstDraw(t *testing.T) {
	msh := buildSphere(24, 24)
	p := NewPipeline(128, 128)
	cam := camera.New(m.Vec3{}, 2.6, 60*stdmath.Pi/180, 1)
	cam.Snap()
	view := cam.View()
	vp := cam.Projection().Mul(view)
	p.projectVerts(msh, vp, view)

	p.reveal.begin(msh)
	if !p.reveal.pending {
		t.Fatal("begin must leave the clock pending, not start it (build time would be charged to the animation otherwise)")
	}

	wireCount := func() int {
		n := 0
		px := p.Pixels()
		for i := 0; i+3 < len(px); i += 4 {
			if px[i] == wireColor.R && px[i+1] == wireColor.G && px[i+2] == wireColor.B {
				n++
			}
		}
		return n
	}

	t0 := time.Now()
	p.clear()
	p.reveal.draw(p, t0)
	if p.reveal.pending {
		t.Fatal("clock still pending after first draw")
	}
	if !p.reveal.start.Equal(t0) {
		t.Fatalf("clock must anchor at first-draw time, got start=%v want %v", p.reveal.start, t0)
	}
	firstFrame := wireCount()

	p.clear()
	p.reveal.draw(p, t0.Add(time.Duration(2*wireRevealDuration*float64(time.Second))))
	full := wireCount()

	if firstFrame >= full {
		t.Fatalf("first draw must start near zero regardless of how long begin/build took: first=%d full=%d", firstFrame, full)
	}
}
