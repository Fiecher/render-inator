package model

import (
	"math"
	"testing"

	m "render-inator/internal/math"
)

func uvSphere(stacks, slices int) *Mesh {
	msh := &Mesh{}
	w := slices + 1
	for i := 0; i <= stacks; i++ {
		phi := math.Pi * float64(i) / float64(stacks)
		sp, cp := math.Sincos(phi)
		for j := 0; j <= slices; j++ {
			theta := 2 * math.Pi * float64(j) / float64(slices)
			st, ct := math.Sincos(theta)
			msh.Verts = append(msh.Verts, m.Vec3{X: float32(sp * ct), Y: float32(cp), Z: float32(sp * st)})
			msh.UVs = append(msh.UVs, m.Vec2{X: float32(j) / float32(slices), Y: float32(i) / float32(stacks)})
		}
	}
	for i := 0; i < stacks; i++ {
		for j := 0; j < slices; j++ {
			a := i*w + j
			b := a + 1
			c := a + w
			d := c + 1
			msh.Tris = append(msh.Tris, Tri{V: [3]int{a, c, b}, UV: [3]int{a, c, b}})
			msh.Tris = append(msh.Tris, Tri{V: [3]int{b, c, d}, UV: [3]int{b, c, d}})
		}
	}
	msh.ComputeNormals()
	msh.analyzeCulling()
	return msh
}

func TestBuildLODChain(t *testing.T) {
	src := uvSphere(64, 64)
	chain := src.BuildLODChain()
	if len(chain.Levels) < 2 {
		t.Fatalf("expected multiple LOD levels, got %d", len(chain.Levels))
	}
	if chain.Levels[0] != src {
		t.Fatal("level 0 must be the original mesh")
	}
	if len(chain.Levels) != len(chain.Errors) {
		t.Fatalf("levels/errors length mismatch: %d vs %d", len(chain.Levels), len(chain.Errors))
	}
	for i := 1; i < len(chain.Levels); i++ {
		prev := len(chain.Levels[i-1].Tris)
		cur := len(chain.Levels[i].Tris)
		if cur >= prev {
			t.Fatalf("level %d not coarser: %d >= %d", i, cur, prev)
		}
		if chain.Errors[i] < chain.Errors[i-1] {
			t.Fatalf("errors not monotonic at %d: %g < %g", i, chain.Errors[i], chain.Errors[i-1])
		}
		checkMesh(t, chain.Levels[i], i)
	}
}

func checkMesh(t *testing.T, msh *Mesh, lvl int) {
	if len(msh.VertNormals) != len(msh.Verts) {
		t.Fatalf("level %d normals not recomputed", lvl)
	}
	for _, v := range msh.Verts {
		if v.X != v.X || v.Y != v.Y || v.Z != v.Z {
			t.Fatalf("level %d NaN vertex", lvl)
		}
	}
	for k, tri := range msh.Tris {
		if tri.V[0] == tri.V[1] || tri.V[1] == tri.V[2] || tri.V[0] == tri.V[2] {
			t.Fatalf("level %d degenerate triangle %d", lvl, k)
		}
		for _, vi := range tri.V {
			if vi < 0 || vi >= len(msh.Verts) {
				t.Fatalf("level %d vertex index out of range", lvl)
			}
		}
	}
}

func TestLODSelect(t *testing.T) {
	src := uvSphere(64, 64)
	chain := src.BuildLODChain()
	fov := float32(60 * math.Pi / 180)

	near := chain.Select(2, 900, fov, false)
	far := chain.Select(60, 900, fov, false)
	if len(far.Tris) > len(near.Tris) {
		t.Fatalf("far must be coarser: far=%d near=%d", len(far.Tris), len(near.Tris))
	}

	idle := chain.Select(10, 900, fov, false)
	move := chain.Select(10, 900, fov, true)
	if len(move.Tris) > len(idle.Tris) {
		t.Fatalf("interacting must be coarser or equal: move=%d idle=%d", len(move.Tris), len(idle.Tris))
	}
}
