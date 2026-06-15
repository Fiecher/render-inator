package model

import m "render-inator/internal/math"

func (msh *Mesh) ComputeNormals() {
	msh.FaceNormals = make([]m.Vec3, len(msh.Tris))
	msh.VertNormals = make([]m.Vec3, len(msh.Verts))

	for i, t := range msh.Tris {
		a := msh.Verts[t.V[0]]
		b := msh.Verts[t.V[1]]
		c := msh.Verts[t.V[2]]

		weighted := b.Sub(a).Cross(c.Sub(a))

		msh.FaceNormals[i] = weighted.Normalize()

		msh.VertNormals[t.V[0]] = msh.VertNormals[t.V[0]].Add(weighted)
		msh.VertNormals[t.V[1]] = msh.VertNormals[t.V[1]].Add(weighted)
		msh.VertNormals[t.V[2]] = msh.VertNormals[t.V[2]].Add(weighted)
	}

	for i := range msh.VertNormals {
		msh.VertNormals[i] = msh.VertNormals[i].Normalize()
	}
}
