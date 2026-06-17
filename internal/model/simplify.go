package model

import (
	"container/heap"
	stdmath "math"

	m "render-inator/internal/math"
)

const (
	lodMinTris    = 2000
	lodFloorTris  = 200
	lodMaxExtra   = 4
	lodIdlePixels = 1.5
	lodMovePixels = 6.0
	lodBoundaryW  = 1000.0
)

type LODChain struct {
	Levels []*Mesh
	Errors []float32
	Radius float32
}

func (c *LODChain) Select(camDist, viewportH, fovY float32, interacting bool) *Mesh {
	if c == nil || len(c.Levels) == 0 {
		return nil
	}
	if len(c.Levels) == 1 || camDist <= 0 || viewportH <= 0 || c.Radius <= 0 {
		return c.Levels[0]
	}
	projPx := c.Radius / camDist * (viewportH / (2 * tanf(fovY*0.5)))
	if projPx <= 0 {
		return c.Levels[0]
	}
	budget := float32(lodIdlePixels)
	if interacting {
		budget = lodMovePixels
	}
	allow := budget / projPx
	pick := 0
	for i := 1; i < len(c.Levels); i++ {
		if c.Errors[i] <= allow {
			pick = i
			continue
		}
		break
	}
	return c.Levels[pick]
}

func (msh *Mesh) BuildLODChain() *LODChain {
	radius := boundingRadius(msh.Verts)
	chain := &LODChain{Levels: []*Mesh{msh}, Errors: []float32{0}, Radius: radius}
	t0 := len(msh.Tris)
	if t0 < lodMinTris {
		return chain
	}

	d := newDecimator(msh)
	d.buildHeap()

	target := t0
	for level := 0; level < lodMaxExtra; level++ {
		target /= 2
		if target < lodFloorTris {
			break
		}
		prev := len(chain.Levels[len(chain.Levels)-1].Tris)
		d.collapseTo(target)
		if d.alive >= prev {
			break
		}
		snap := d.snapshot(msh)
		relErr := float32(0)
		if radius > 0 {
			relErr = float32(stdmath.Sqrt(d.maxCost)) / radius
		}
		chain.Levels = append(chain.Levels, snap)
		chain.Errors = append(chain.Errors, relErr)
		if d.alive <= lodFloorTris {
			break
		}
	}
	return chain
}

type quadric [10]float64

func (q *quadric) addPlane(a, b, c, d float64) {
	q[0] += a * a
	q[1] += a * b
	q[2] += a * c
	q[3] += a * d
	q[4] += b * b
	q[5] += b * c
	q[6] += b * d
	q[7] += c * c
	q[8] += c * d
	q[9] += d * d
}

func (q *quadric) add(o *quadric) {
	for i := range q {
		q[i] += o[i]
	}
}

func (q *quadric) errorAt(x, y, z float64) float64 {
	return q[0]*x*x + 2*q[1]*x*y + 2*q[2]*x*z + 2*q[3]*x +
		q[4]*y*y + 2*q[5]*y*z + 2*q[6]*y +
		q[7]*z*z + 2*q[8]*z + q[9]
}

type collapseItem struct {
	cost       float64
	i, j       int
	vi, vj     int
	tx, ty, tz float32
}

type collapseHeap []collapseItem

func (h collapseHeap) Len() int           { return len(h) }
func (h collapseHeap) Less(a, b int) bool { return h[a].cost < h[b].cost }
func (h collapseHeap) Swap(a, b int)      { h[a], h[b] = h[b], h[a] }

func (h *collapseHeap) Push(x any) { *h = append(*h, x.(collapseItem)) }

func (h *collapseHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}

type decimator struct {
	pos   []m.Vec3
	quad  []quadric
	ver   []int
	vdead []bool

	triV  [][3]int
	triUV [][3]int
	tdead []bool
	alive int

	adj [][]int

	heap    collapseHeap
	maxCost float64
}

func newDecimator(msh *Mesh) *decimator {
	nv := len(msh.Verts)
	nt := len(msh.Tris)
	d := &decimator{
		pos:   make([]m.Vec3, nv),
		quad:  make([]quadric, nv),
		ver:   make([]int, nv),
		vdead: make([]bool, nv),
		adj:   make([][]int, nv),
		triV:  make([][3]int, nt),
		triUV: make([][3]int, nt),
		tdead: make([]bool, nt),
		alive: nt,
	}
	copy(d.pos, msh.Verts)
	for i, t := range msh.Tris {
		d.triV[i] = t.V
		d.triUV[i] = t.UV
		for k := 0; k < 3; k++ {
			d.adj[t.V[k]] = append(d.adj[t.V[k]], i)
		}
	}
	d.buildQuadrics()
	return d
}

func (d *decimator) buildQuadrics() {
	bcount := make(map[[2]int]int, len(d.triV)*3)
	for ti := range d.triV {
		a, b, c := d.triV[ti][0], d.triV[ti][1], d.triV[ti][2]
		n, ok := faceNormalD(d.pos[a], d.pos[b], d.pos[c])
		if !ok {
			d.tdead[ti] = true
			d.alive--
			continue
		}
		pd := -(n[0]*float64(d.pos[a].X) + n[1]*float64(d.pos[a].Y) + n[2]*float64(d.pos[a].Z))
		var q quadric
		q.addPlane(n[0], n[1], n[2], pd)
		d.quad[a].add(&q)
		d.quad[b].add(&q)
		d.quad[c].add(&q)
		for _, e := range edgesOf(a, b, c) {
			bcount[e]++
		}
	}
	for ti := range d.triV {
		if d.tdead[ti] {
			continue
		}
		a, b, c := d.triV[ti][0], d.triV[ti][1], d.triV[ti][2]
		n, _ := faceNormalD(d.pos[a], d.pos[b], d.pos[c])
		for _, pair := range [3][2]int{{a, b}, {b, c}, {c, a}} {
			if bcount[orderEdge(pair[0], pair[1])] == 1 {
				d.addBoundary(pair[0], pair[1], n)
			}
		}
	}
}

func (d *decimator) addBoundary(ia, ib int, n [3]float64) {
	a, b := d.pos[ia], d.pos[ib]
	ex := float64(b.X - a.X)
	ey := float64(b.Y - a.Y)
	ez := float64(b.Z - a.Z)
	px := ey*n[2] - ez*n[1]
	py := ez*n[0] - ex*n[2]
	pz := ex*n[1] - ey*n[0]
	l := stdmath.Sqrt(px*px + py*py + pz*pz)
	if l < 1e-12 {
		return
	}
	inv := 1 / l
	px, py, pz = px*inv, py*inv, pz*inv
	pd := -(px*float64(a.X) + py*float64(a.Y) + pz*float64(a.Z))
	var q quadric
	q.addPlane(px, py, pz, pd)
	for i := range q {
		q[i] *= lodBoundaryW
	}
	d.quad[ia].add(&q)
	d.quad[ib].add(&q)
}

func (d *decimator) buildHeap() {
	seen := make(map[[2]int]bool, len(d.triV)*3)
	for ti := range d.triV {
		if d.tdead[ti] {
			continue
		}
		v := d.triV[ti]
		for _, pair := range [3][2]int{{v[0], v[1]}, {v[1], v[2]}, {v[2], v[0]}} {
			e := orderEdge(pair[0], pair[1])
			if e[0] == e[1] || seen[e] {
				continue
			}
			seen[e] = true
			d.pushEdge(e[0], e[1])
		}
	}
}

func (d *decimator) pushEdge(i, j int) {
	if i == j || d.vdead[i] || d.vdead[j] {
		return
	}
	q := d.quad[i]
	q.add(&d.quad[j])
	tx, ty, tz, cost := d.optimal(&q, i, j)
	heap.Push(&d.heap, collapseItem{cost, i, j, d.ver[i], d.ver[j], tx, ty, tz})
}

func (d *decimator) optimal(q *quadric, i, j int) (float32, float32, float32, float64) {
	det := q[0]*(q[4]*q[7]-q[5]*q[5]) -
		q[1]*(q[1]*q[7]-q[5]*q[2]) +
		q[2]*(q[1]*q[5]-q[4]*q[2])
	pa, pb := d.pos[i], d.pos[j]
	if stdmath.Abs(det) > 1e-10 {
		c00 := q[4]*q[7] - q[5]*q[5]
		c01 := q[5]*q[2] - q[1]*q[7]
		c02 := q[1]*q[5] - q[4]*q[2]
		c11 := q[0]*q[7] - q[2]*q[2]
		c12 := q[1]*q[2] - q[0]*q[5]
		c22 := q[0]*q[4] - q[1]*q[1]
		b0, b1, b2 := -q[3], -q[6], -q[8]
		inv := 1 / det
		x := (c00*b0 + c01*b1 + c02*b2) * inv
		y := (c01*b0 + c11*b1 + c12*b2) * inv
		z := (c02*b0 + c12*b1 + c22*b2) * inv
		mx := 0.5 * float64(pa.X+pb.X)
		my := 0.5 * float64(pa.Y+pb.Y)
		mz := 0.5 * float64(pa.Z+pb.Z)
		el := float64(pb.Sub(pa).Len())
		dx, dy, dz := x-mx, y-my, z-mz
		if el == 0 || dx*dx+dy*dy+dz*dz <= 16*el*el {
			return float32(x), float32(y), float32(z), q.errorAt(x, y, z)
		}
	}
	return bestFallback(q, pa, pb)
}

func bestFallback(q *quadric, pa, pb m.Vec3) (float32, float32, float32, float64) {
	mx := 0.5 * float64(pa.X+pb.X)
	my := 0.5 * float64(pa.Y+pb.Y)
	mz := 0.5 * float64(pa.Z+pb.Z)
	ca := q.errorAt(float64(pa.X), float64(pa.Y), float64(pa.Z))
	cb := q.errorAt(float64(pb.X), float64(pb.Y), float64(pb.Z))
	cm := q.errorAt(mx, my, mz)
	if ca <= cb && ca <= cm {
		return pa.X, pa.Y, pa.Z, ca
	}
	if cb <= cm {
		return pb.X, pb.Y, pb.Z, cb
	}
	return float32(mx), float32(my), float32(mz), cm
}

func (d *decimator) collapseTo(target int) {
	for d.alive > target && d.heap.Len() > 0 {
		it := heap.Pop(&d.heap).(collapseItem)
		if d.vdead[it.i] || d.vdead[it.j] {
			continue
		}
		if d.ver[it.i] != it.vi || d.ver[it.j] != it.vj {
			continue
		}
		if !d.tryCollapse(it) {
			continue
		}
		if it.cost > d.maxCost {
			d.maxCost = it.cost
		}
	}
}

func (d *decimator) tryCollapse(it collapseItem) bool {
	i, j := it.i, it.j
	target := m.Vec3{X: it.tx, Y: it.ty, Z: it.tz}
	if d.flips(i, j, target) || d.flips(j, i, target) {
		return false
	}
	d.pos[i] = target
	d.quad[i].add(&d.quad[j])
	d.vdead[j] = true
	d.ver[i]++
	for _, ti := range d.adj[j] {
		if d.tdead[ti] {
			continue
		}
		tv := &d.triV[ti]
		for k := 0; k < 3; k++ {
			if tv[k] == j {
				tv[k] = i
			}
		}
		if tv[0] == tv[1] || tv[1] == tv[2] || tv[2] == tv[0] {
			d.tdead[ti] = true
			d.alive--
			continue
		}
		d.adj[i] = append(d.adj[i], ti)
	}
	d.adj[j] = nil
	d.refresh(i)
	return true
}

func (d *decimator) flips(moved, other int, target m.Vec3) bool {
	for _, ti := range d.adj[moved] {
		if d.tdead[ti] {
			continue
		}
		tv := d.triV[ti]
		if tv[0] == other || tv[1] == other || tv[2] == other {
			continue
		}
		on, ok := faceNormalD(d.pos[tv[0]], d.pos[tv[1]], d.pos[tv[2]])
		if !ok {
			continue
		}
		pa, pb, pc := d.pos[tv[0]], d.pos[tv[1]], d.pos[tv[2]]
		if tv[0] == moved {
			pa = target
		}
		if tv[1] == moved {
			pb = target
		}
		if tv[2] == moved {
			pc = target
		}
		nn, ok := faceNormalD(pa, pb, pc)
		if !ok {
			return true
		}
		if on[0]*nn[0]+on[1]*nn[1]+on[2]*nn[2] < 0.1 {
			return true
		}
	}
	return false
}

func (d *decimator) refresh(i int) {
	for _, ti := range d.adj[i] {
		if d.tdead[ti] {
			continue
		}
		tv := d.triV[ti]
		for k := 0; k < 3; k++ {
			if nb := tv[k]; nb != i && !d.vdead[nb] {
				d.pushEdge(i, nb)
			}
		}
	}
}

func (d *decimator) snapshot(src *Mesh) *Mesh {
	out := &Mesh{}
	remap := make([]int, len(d.pos))
	for i := range remap {
		remap[i] = -1
	}
	hasUV := len(src.UVs) > 0
	uvremap := make([]int, len(src.UVs))
	for i := range uvremap {
		uvremap[i] = -1
	}
	for ti := range d.triV {
		if d.tdead[ti] {
			continue
		}
		tv, tu := d.triV[ti], d.triUV[ti]
		var nt Tri
		for k := 0; k < 3; k++ {
			vi := tv[k]
			if remap[vi] < 0 {
				remap[vi] = len(out.Verts)
				out.Verts = append(out.Verts, d.pos[vi])
			}
			nt.V[k] = remap[vi]
			nt.UV[k] = -1
			nt.N[k] = -1
			if hasUV && tu[k] >= 0 {
				ui := tu[k]
				if uvremap[ui] < 0 {
					uvremap[ui] = len(out.UVs)
					out.UVs = append(out.UVs, src.UVs[ui])
				}
				nt.UV[k] = uvremap[ui]
			}
		}
		out.Tris = append(out.Tris, nt)
	}
	out.ComputeNormals()
	out.analyzeCulling()
	return out
}

func faceNormalD(a, b, c m.Vec3) ([3]float64, bool) {
	ux, uy, uz := float64(b.X-a.X), float64(b.Y-a.Y), float64(b.Z-a.Z)
	vx, vy, vz := float64(c.X-a.X), float64(c.Y-a.Y), float64(c.Z-a.Z)
	nx := uy*vz - uz*vy
	ny := uz*vx - ux*vz
	nz := ux*vy - uy*vx
	l := stdmath.Sqrt(nx*nx + ny*ny + nz*nz)
	if l < 1e-12 {
		return [3]float64{}, false
	}
	inv := 1 / l
	return [3]float64{nx * inv, ny * inv, nz * inv}, true
}

func edgesOf(a, b, c int) [3][2]int {
	return [3][2]int{orderEdge(a, b), orderEdge(b, c), orderEdge(c, a)}
}

func orderEdge(a, b int) [2]int {
	if a > b {
		return [2]int{b, a}
	}
	return [2]int{a, b}
}

func boundingRadius(vs []m.Vec3) float32 {
	if len(vs) == 0 {
		return 0
	}
	lo, hi := vs[0], vs[0]
	for _, v := range vs {
		lo = m.Vec3{X: minF(lo.X, v.X), Y: minF(lo.Y, v.Y), Z: minF(lo.Z, v.Z)}
		hi = m.Vec3{X: maxF(hi.X, v.X), Y: maxF(hi.Y, v.Y), Z: maxF(hi.Z, v.Z)}
	}
	return hi.Sub(lo).Len() * 0.5
}

func tanf(v float32) float32 { return float32(stdmath.Tan(float64(v))) }

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
