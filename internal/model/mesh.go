package model

import m "render-inator/internal/math"

type Tri struct {
	V  [3]int
	UV [3]int
}

type Mesh struct {
	Verts []m.Vec3
	UVs   []m.Vec2
	Tris  []Tri

	FaceNormals []m.Vec3

	VertNormals []m.Vec3

	CullSafe bool

	WindingOutward bool
}
