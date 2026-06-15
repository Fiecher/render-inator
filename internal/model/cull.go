package model

func (msh *Mesh) analyzeCulling() {
	msh.CullSafe = false
	msh.WindingOutward = true

	const quant = 1e5
	weld := make(map[[3]int64]int, len(msh.Verts))
	id := make([]int, len(msh.Verts))
	for i, v := range msh.Verts {
		key := [3]int64{
			roundI(v.X * quant),
			roundI(v.Y * quant),
			roundI(v.Z * quant),
		}
		wid, ok := weld[key]
		if !ok {
			wid = len(weld)
			weld[key] = wid
		}
		id[i] = wid
	}

	type edge [2]int
	type tally struct{ count, net int }
	edges := make(map[edge]tally, len(msh.Tris)*3)
	for _, t := range msh.Tris {
		a, b, c := id[t.V[0]], id[t.V[1]], id[t.V[2]]
		if a == b || b == c || c == a {

			continue
		}
		for _, he := range [3][2]int{{a, b}, {b, c}, {c, a}} {
			x, y, s := he[0], he[1], 1
			if x > y {
				x, y, s = y, x, -1
			}
			tl := edges[edge{x, y}]
			tl.count++
			tl.net += s
			edges[edge{x, y}] = tl
		}
	}
	for _, tl := range edges {
		if tl.count != 2 || tl.net != 0 {
			return
		}
	}

	var vol6 float64
	for _, t := range msh.Tris {
		a := msh.Verts[t.V[0]]
		b := msh.Verts[t.V[1]]
		c := msh.Verts[t.V[2]]
		vol6 += float64(a.Dot(b.Cross(c)))
	}
	msh.CullSafe = true
	msh.WindingOutward = vol6 >= 0
}

func roundI(x float32) int64 {
	if x < 0 {
		return int64(x - 0.5)
	}
	return int64(x + 0.5)
}
