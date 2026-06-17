package model

import (
	"math"
	"testing"
)

func TestParseOBJCube(t *testing.T) {

	const cube = `
# unit cube
v -1 -1 -1
v  1 -1 -1
v  1  1 -1
v -1  1 -1
v -1 -1  1
v  1 -1  1
v  1  1  1
v -1  1  1
f 1 2 3 4
f 5 6 7 8
f 1 5 8 4
f 2 6 7 3
f 4 3 7 8
f 1 2 6 5
`
	msh, err := ParseOBJ([]byte(cube))
	if err != nil {
		t.Fatalf("ParseOBJ: %v", err)
	}
	if len(msh.Verts) != 8 {
		t.Errorf("verts = %d, want 8", len(msh.Verts))
	}
	if len(msh.Tris) != 12 {
		t.Errorf("tris = %d, want 12", len(msh.Tris))
	}
	if len(msh.FaceNormals) != 12 {
		t.Errorf("face normals = %d, want 12", len(msh.FaceNormals))
	}
	if len(msh.VertNormals) != 8 {
		t.Errorf("vert normals = %d, want 8", len(msh.VertNormals))
	}
	for i, n := range msh.VertNormals {
		l := math.Sqrt(float64(n.Dot(n)))
		if math.Abs(l-1) > 1e-4 {
			t.Errorf("vert normal %d not unit: len=%v", i, l)
		}
	}
}

func TestParseOBJRefForms(t *testing.T) {

	const data = `
v 0 0 0
v 1 0 0
v 0 1 0
vt 0 0
vt 1 0
vt 0 1
vn 0 0 1
f 1/1/1 2/2/1 3/3/1
f 1//1 2//1 3//1
f -3 -2 -1
`
	msh, err := ParseOBJ([]byte(data))
	if err != nil {
		t.Fatalf("ParseOBJ: %v", err)
	}
	if len(msh.Tris) != 3 {
		t.Fatalf("tris = %d, want 3", len(msh.Tris))
	}

	if msh.Tris[0].UV[0] != 0 || msh.Tris[0].UV[2] != 2 {
		t.Errorf("face 0 UV = %v, want [0 1 2]", msh.Tris[0].UV)
	}

	if msh.Tris[1].UV[0] != -1 {
		t.Errorf("face 1 UV[0] = %d, want -1", msh.Tris[1].UV[0])
	}

	if msh.Tris[2].V != [3]int{0, 1, 2} {
		t.Errorf("face 2 V = %v, want [0 1 2]", msh.Tris[2].V)
	}
}

func TestCullSafe(t *testing.T) {

	const tetra = `
v 0 0 0
v 1 0 0
v 0 1 0
v 0 0 1
f 1 3 2
f 1 2 4
f 1 4 3
f 2 3 4
`
	msh, err := ParseOBJ([]byte(tetra))
	if err != nil {
		t.Fatalf("ParseOBJ tetra: %v", err)
	}
	if !msh.CullSafe {
		t.Errorf("tetra CullSafe = false, want true")
	}
	if !msh.WindingOutward {
		t.Errorf("tetra WindingOutward = false, want true")
	}

	const tetraWithSliver = tetra + "f 1 1 2\n"
	if msh, err := ParseOBJ([]byte(tetraWithSliver)); err != nil {
		t.Fatalf("ParseOBJ tetra+sliver: %v", err)
	} else if !msh.CullSafe {
		t.Errorf("tetra + degenerate face CullSafe = false, want true")
	}

	const flipped = `
v 0 0 0
v 1 0 0
v 0 1 0
v 0 0 1
f 1 2 3
f 1 2 4
f 1 4 3
f 2 3 4
`
	if msh, err := ParseOBJ([]byte(flipped)); err != nil {
		t.Fatalf("ParseOBJ flipped: %v", err)
	} else if msh.CullSafe {
		t.Errorf("inconsistently wound tetra CullSafe = true, want false")
	}

	const open = `
v 0 0 0
v 1 0 0
v 0 1 0
f 1 2 3
`
	if msh, err := ParseOBJ([]byte(open)); err != nil {
		t.Fatalf("ParseOBJ open: %v", err)
	} else if msh.CullSafe {
		t.Errorf("open triangle CullSafe = true, want false")
	}
}

func TestParseOBJErrors(t *testing.T) {
	cases := map[string]string{
		"empty":        ``,
		"no faces":     "v 0 0 0\nv 1 0 0\nv 0 1 0\n",
		"bad float":    "v 0 0 x\n",
		"oob index":    "v 0 0 0\nf 1 2 3\n",
		"short vertex": "v 0 0\n",
	}
	for name, src := range cases {
		if _, err := ParseOBJ([]byte(src)); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestParseOBJCubeFlatShading(t *testing.T) {

	const cube = `
v -1 -1 -1
v  1 -1 -1
v  1  1 -1
v -1  1 -1
v -1 -1  1
v  1 -1  1
v  1  1  1
v -1  1  1
f 1 2 3 4
f 5 6 7 8
f 1 5 8 4
f 2 6 7 3
f 4 3 7 8
f 1 2 6 5
`
	msh, err := ParseOBJ([]byte(cube))
	if err != nil {
		t.Fatalf("ParseOBJ: %v", err)
	}

	if len(msh.Normals) < 3 {
		t.Errorf("shading normals = %d, want >= 3 (faces must not collapse to one smooth normal)", len(msh.Normals))
	}

	for ti, tri := range msh.Tris {
		if tri.N[0] != tri.N[1] || tri.N[1] != tri.N[2] {
			t.Errorf("tri %d corners not coplanar: N=%v", ti, tri.N)
		}
		for k := 0; k < 3; k++ {
			n := msh.Normals[tri.N[k]]
			if d := float64(n.Dot(msh.FaceNormals[ti])); d < 0.999 {
				t.Errorf("tri %d corner %d not flat: dot(face)=%v", ti, k, d)
			}
			if l := math.Sqrt(float64(n.Dot(n))); math.Abs(l-1) > 1e-4 {
				t.Errorf("shading normal %v not unit: len=%v", n, l)
			}
		}
	}
}

func TestParseOBJFileNormals(t *testing.T) {

	const data = `
v 0 0 0
v 1 0 0
v 0 1 0
vn 0 0 2
f 1//1 2//1 3//1
`
	msh, err := ParseOBJ([]byte(data))
	if err != nil {
		t.Fatalf("ParseOBJ: %v", err)
	}
	if len(msh.Normals) != 1 {
		t.Fatalf("normals = %d, want 1 (from file vn)", len(msh.Normals))
	}
	n := msh.Normals[0]
	if math.Abs(float64(n.X)) > 1e-6 || math.Abs(float64(n.Y)) > 1e-6 || math.Abs(float64(n.Z-1)) > 1e-6 {
		t.Errorf("file normal not normalized to +Z: %v", n)
	}
	if msh.Tris[0].N != [3]int{0, 0, 0} {
		t.Errorf("tri N = %v, want [0 0 0]", msh.Tris[0].N)
	}
}

func TestParseOBJWeldsSeamNormals(t *testing.T) {

	const data = `
v 0 0 0
v 2 0 0
v 1 1 0.2
v 0 0 0
v 2 0 0
v 1 -1 0.2
f 1 2 3
f 4 6 5
`
	msh, err := ParseOBJ([]byte(data))
	if err != nil {
		t.Fatalf("ParseOBJ: %v", err)
	}
	if msh.FaceNormals[0] == msh.FaceNormals[1] {
		t.Fatalf("faces coincide; fixture cannot detect welding")
	}
	n0 := msh.Normals[msh.Tris[0].N[0]]
	n1 := msh.Normals[msh.Tris[1].N[0]]
	if d := float64(n0.Dot(n1)); d < 0.9999 {
		t.Errorf("split-seam corners not welded: dot=%v (n0=%v n1=%v)", d, n0, n1)
	}
}
