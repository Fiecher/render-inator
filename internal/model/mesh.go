package model

import m "render-inator/internal/math"

type Tri struct {
	V   [3]int
	UV  [3]int
	N   [3]int
	Mat int
}

type Image struct {
	Pix  []byte
	W, H int
}

type Material struct {
	BaseColor [4]float32
	Image     int
}

type Mesh struct {
	Verts []m.Vec3
	UVs   []m.Vec2
	Tris  []Tri

	FaceNormals []m.Vec3

	VertNormals []m.Vec3

	Normals []m.Vec3

	Materials []Material
	Images    []Image

	CullSafe bool

	WindingOutward bool
}
