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

func (msh *Mesh) ComputeFaceNormals() {
	msh.FaceNormals = make([]m.Vec3, len(msh.Tris))
	for i, t := range msh.Tris {
		a := msh.Verts[t.V[0]]
		b := msh.Verts[t.V[1]]
		c := msh.Verts[t.V[2]]
		msh.FaceNormals[i] = b.Sub(a).Cross(c.Sub(a)).Normalize()
	}
}

const creaseCos = 0.6427876

func (msh *Mesh) ComputeShadingNormals() {
	nv := len(msh.Verts)
	nt := len(msh.Tris)
	if nt == 0 || len(msh.FaceNormals) != nt {
		return
	}

	const posQuant = 1e5
	gid := make([]int32, nv)
	groups := make(map[[3]int64]int32, nv)
	var ng int32
	for i := range msh.Verts {
		p := msh.Verts[i]
		key := [3]int64{roundI(p.X * posQuant), roundI(p.Y * posQuant), roundI(p.Z * posQuant)}
		g, ok := groups[key]
		if !ok {
			g = ng
			groups[key] = g
			ng++
		}
		gid[i] = g
	}

	start := make([]int32, ng+1)
	for ti := range msh.Tris {
		t := &msh.Tris[ti]
		start[gid[t.V[0]]+1]++
		start[gid[t.V[1]]+1]++
		start[gid[t.V[2]]+1]++
	}
	for i := int32(1); i <= ng; i++ {
		start[i] += start[i-1]
	}
	inc := make([]int32, start[ng])
	cur := make([]int32, ng)
	copy(cur, start[:ng])
	for ti := range msh.Tris {
		t := &msh.Tris[ti]
		for k := 0; k < 3; k++ {
			g := gid[t.V[k]]
			inc[cur[g]] = int32(ti)
			cur[g]++
		}
	}

	weighted := make([]m.Vec3, nt)
	for ti := range msh.Tris {
		t := &msh.Tris[ti]
		a := msh.Verts[t.V[0]]
		b := msh.Verts[t.V[1]]
		c := msh.Verts[t.V[2]]
		weighted[ti] = b.Sub(a).Cross(c.Sub(a))
	}

	const quant = 1e4
	seen := make(map[[3]int64]int32, nv)
	msh.Normals = msh.Normals[:0]
	for ti := range msh.Tris {
		t := &msh.Tris[ti]
		fn := msh.FaceNormals[ti]
		for k := 0; k < 3; k++ {
			g := gid[t.V[k]]
			var sum m.Vec3
			for e := start[g]; e < start[g+1]; e++ {
				tj := inc[e]
				if msh.FaceNormals[tj].Dot(fn) >= creaseCos {
					sum = sum.Add(weighted[tj])
				}
			}
			n := fn
			if sum.Dot(sum) > 0 {
				n = sum.Normalize()
			}
			key := [3]int64{roundI(n.X * quant), roundI(n.Y * quant), roundI(n.Z * quant)}
			idx, ok := seen[key]
			if !ok {
				idx = int32(len(msh.Normals))
				msh.Normals = append(msh.Normals, n)
				seen[key] = idx
			}
			t.N[k] = int(idx)
		}
	}
}
