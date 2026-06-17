package render

import (
	stdmath "math"
	"math/rand"
	"time"

	m "render-inator/internal/math"
	"render-inator/internal/model"
)

const (
	wireRevealDuration = 0.8
	wireWeldQuant      = 1e5
)

type wireEdge struct {
	a, b   int32
	ra, rb int32
	invLen float32
}

type pqItem struct {
	d float32
	n int32
}

type wireReveal struct {
	builtFor *model.Mesh

	edges    []wireEdge
	adjStart []int32
	adjNode  []int32
	adjLen   []float32

	dist    []float32
	maxDist float32

	start   time.Time
	active  bool
	pending bool

	pq []pqItem
}

func weldKey(v m.Vec3) [3]int64 {
	return [3]int64{
		weldRound(v.X * wireWeldQuant),
		weldRound(v.Y * wireWeldQuant),
		weldRound(v.Z * wireWeldQuant),
	}
}

func weldRound(x float32) int64 {
	if x < 0 {
		return int64(x - 0.5)
	}
	return int64(x + 0.5)
}

func (r *wireReveal) build(msh *model.Mesh) {
	r.builtFor = msh
	r.edges = r.edges[:0]
	r.active = false

	nv := len(msh.Verts)
	wmap := make(map[[3]int64]int32, nv)
	weld := make([]int32, nv)
	pos := make([]m.Vec3, 0, nv)
	rep := make([]int32, 0, nv)
	for i := 0; i < nv; i++ {
		k := weldKey(msh.Verts[i])
		id, ok := wmap[k]
		if !ok {
			id = int32(len(pos))
			wmap[k] = id
			pos = append(pos, msh.Verts[i])
			rep = append(rep, int32(i))
		}
		weld[i] = id
	}
	n := len(pos)

	type ek struct{ a, b int32 }
	emap := make(map[ek]struct{}, len(msh.Tris)*3)
	deg := make([]int32, n)
	add := func(o0, o1 int) {
		a, b := weld[o0], weld[o1]
		if a == b {
			return
		}
		if a > b {
			a, b = b, a
		}
		key := ek{a, b}
		if _, ok := emap[key]; ok {
			return
		}
		d := pos[a].Sub(pos[b]).Len()
		if d <= 0 {
			return
		}
		emap[key] = struct{}{}
		r.edges = append(r.edges, wireEdge{a: a, b: b, ra: rep[a], rb: rep[b], invLen: 1 / d})
		deg[a]++
		deg[b]++
	}
	for ti := range msh.Tris {
		t := &msh.Tris[ti]
		add(t.V[0], t.V[1])
		add(t.V[1], t.V[2])
		add(t.V[2], t.V[0])
	}

	if cap(r.adjStart) < n+1 {
		r.adjStart = make([]int32, n+1)
	} else {
		r.adjStart = r.adjStart[:n+1]
	}
	r.adjStart[0] = 0
	for i := 0; i < n; i++ {
		r.adjStart[i+1] = r.adjStart[i] + deg[i]
	}
	total := int(r.adjStart[n])
	if cap(r.adjNode) < total {
		r.adjNode = make([]int32, total)
		r.adjLen = make([]float32, total)
	} else {
		r.adjNode = r.adjNode[:total]
		r.adjLen = r.adjLen[:total]
	}
	cur := make([]int32, n)
	copy(cur, r.adjStart[:n])
	for i := range r.edges {
		e := &r.edges[i]
		l := 1 / e.invLen
		pa := cur[e.a]
		r.adjNode[pa] = e.b
		r.adjLen[pa] = l
		cur[e.a]++
		pb := cur[e.b]
		r.adjNode[pb] = e.a
		r.adjLen[pb] = l
		cur[e.b]++
	}

	if cap(r.dist) < n {
		r.dist = make([]float32, n)
	} else {
		r.dist = r.dist[:n]
	}
}

func (r *wireReveal) begin(msh *model.Mesh) {
	if r.builtFor != msh {
		r.build(msh)
	}
	if len(r.edges) == 0 || len(r.dist) == 0 {
		r.active = false
		return
	}
	seed := 0
	if n := len(r.dist); n > 1 {
		seed = rand.New(rand.NewSource(time.Now().UnixNano())).Intn(n)
	}
	r.dijkstra(seed)
	r.pending = true
	r.active = true
}

func (r *wireReveal) dijkstra(seed int) {
	n := len(r.dist)
	const inf = float32(stdmath.MaxFloat32)
	for i := 0; i < n; i++ {
		r.dist[i] = inf
	}
	r.pq = r.pq[:0]

	r.dist[seed] = 0
	r.pqPush(pqItem{0, int32(seed)})

	scan := 0
	for {
		for len(r.pq) > 0 {
			it := r.pqPop()
			u := it.n
			if it.d > r.dist[u] {
				continue
			}
			for k := r.adjStart[u]; k < r.adjStart[u+1]; k++ {
				v := r.adjNode[k]
				nd := it.d + r.adjLen[k]
				if nd < r.dist[v] {
					r.dist[v] = nd
					r.pqPush(pqItem{nd, v})
				}
			}
		}
		for scan < n && r.dist[scan] != inf {
			scan++
		}
		if scan >= n {
			break
		}
		r.dist[scan] = 0
		r.pqPush(pqItem{0, int32(scan)})
	}

	mx := float32(0)
	for i := 0; i < n; i++ {
		if d := r.dist[i]; d != inf && d > mx {
			mx = d
		}
	}
	if mx <= 0 {
		mx = 1
	}
	r.maxDist = mx
}

func (r *wireReveal) draw(p *Pipeline, now time.Time) {
	if r.pending {
		r.start = now
		r.pending = false
	}
	f := float32(now.Sub(r.start).Seconds()) / wireRevealDuration
	if f >= 1 {
		f = 1
		r.active = false
	}
	inv := 1 - f
	T := r.maxDist * (1 - inv*inv*inv)

	for i := range r.edges {
		e := &r.edges[i]
		va := &p.vs[e.ra]
		vb := &p.vs[e.rb]
		if !va.ok || !vb.ok {
			continue
		}
		fa := (T - r.dist[e.a]) * e.invLen
		fb := (T - r.dist[e.b]) * e.invLen
		if fa <= 0 && fb <= 0 {
			continue
		}
		if fa+fb >= 1 {
			p.drawLine(int(va.sx), int(va.sy), int(vb.sx), int(vb.sy), wireColor)
			continue
		}
		if fa > 0 {
			if fa > 1 {
				fa = 1
			}
			mx := va.sx + (vb.sx-va.sx)*fa
			my := va.sy + (vb.sy-va.sy)*fa
			p.drawLine(int(va.sx), int(va.sy), int(mx), int(my), wireColor)
		}
		if fb > 0 {
			if fb > 1 {
				fb = 1
			}
			mx := vb.sx + (va.sx-vb.sx)*fb
			my := vb.sy + (va.sy-vb.sy)*fb
			p.drawLine(int(vb.sx), int(vb.sy), int(mx), int(my), wireColor)
		}
	}
}

func (r *wireReveal) pqPush(it pqItem) {
	r.pq = append(r.pq, it)
	i := len(r.pq) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if r.pq[parent].d <= r.pq[i].d {
			break
		}
		r.pq[parent], r.pq[i] = r.pq[i], r.pq[parent]
		i = parent
	}
}

func (r *wireReveal) pqPop() pqItem {
	top := r.pq[0]
	last := len(r.pq) - 1
	r.pq[0] = r.pq[last]
	r.pq = r.pq[:last]
	i, nn := 0, len(r.pq)
	for {
		l := 2*i + 1
		rr := l + 1
		small := i
		if l < nn && r.pq[l].d < r.pq[small].d {
			small = l
		}
		if rr < nn && r.pq[rr].d < r.pq[small].d {
			small = rr
		}
		if small == i {
			break
		}
		r.pq[i], r.pq[small] = r.pq[small], r.pq[i]
		i = small
	}
	return top
}
