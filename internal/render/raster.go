package render

import (
	stdmath "math"

	m "render-inator/internal/math"
	"render-inator/internal/model"
)

const ambient = 0.15

var wireColor = RGBA{255, 64, 0, 255}

func edge(ax, ay, bx, by, cx, cy float32) float32 {
	return (cx-ax)*(by-ay) - (cy-ay)*(bx-ax)
}

const (
	studioGround      = 0.30
	studioSky         = 0.58
	studioKeyStrength = 0.5
	studioSpec        = 0.28
)

func (p *Pipeline) rasterizeTriangle(msh *model.Mesh, ti int, t *model.Tri, a, b, c *vert, toLight m.Vec3) {
	area := edge(a.sx, a.sy, b.sx, b.sy, c.sx, c.sy)
	if area == 0 {
		return
	}
	if p.cullSign != 0 && area*float32(p.cullSign) > 0 {
		return
	}

	fw, fh := float32(p.w), float32(p.h)
	if max3(a.sx, b.sx, c.sx) < 0 || min3(a.sx, b.sx, c.sx) >= fw ||
		max3(a.sy, b.sy, c.sy) < 0 || min3(a.sy, b.sy, c.sy) >= fh {
		return
	}
	invArea := 1 / area

	a0 := (c.sy - b.sy) * invArea
	a1 := (a.sy - c.sy) * invArea
	a2 := (b.sy - a.sy) * invArea

	minX := clampInt(int(floor(min3(a.sx, b.sx, c.sx))), 0, p.w-1)
	maxX := clampInt(int(floor(max3(a.sx, b.sx, c.sx))), 0, p.w-1)
	minY := clampInt(int(floor(min3(a.sy, b.sy, c.sy))), 0, p.h-1)
	maxY := clampInt(int(floor(max3(a.sy, b.sy, c.sy))), 0, p.h-1)

	wide := maxX - minX

	spanSolve := wide >= 16

	bigTri := wide >= 16 || maxY-minY >= 16

	matCol, matImg, matChecker := p.triMaterial(t)
	useTex := p.cfg.Texture && t.UV[0] >= 0 && t.UV[1] >= 0 && t.UV[2] >= 0 && len(msh.UVs) > 0 && (matImg != nil || matChecker != nil)
	useLight := p.cfg.Lighting
	perspTex := useTex && bigTri

	var uvA, uvB, uvC m.Vec2
	var texPix []byte
	var texW, texH int
	var texFW, texFH float32
	hasImg := false
	if useTex {
		uvA, uvB, uvC = msh.UVs[t.UV[0]], msh.UVs[t.UV[1]], msh.UVs[t.UV[2]]
		if it := matImg; it != nil && it.W > 0 && it.H > 0 {
			texelArea := (uvB.X-uvA.X)*(uvC.Y-uvA.Y) - (uvB.Y-uvA.Y)*(uvC.X-uvA.X)
			if texelArea < 0 {
				texelArea = -texelArea
			}
			texelArea *= float32(it.W) * float32(it.H)
			scrArea := area
			if scrArea < 0 {
				scrArea = -scrArea
			}
			texPix, texW, texH, texFW, texFH = it.mipFor(texelArea, scrArea)
			hasImg = true
		}
	}

	spec := useLight && !p.cfg.Flat
	lit := useLight
	baseCol := matCol
	var iA, iB, iC float32
	var dA, dB, dC float32
	if useLight {
		if p.cfg.Flat {
			fi := vertLight(msh.FaceNormals[ti], toLight)
			iA, iB, iC = fi, fi, fi
			if !useTex {
				baseCol = matCol.scale(fi)
				lit = false
			}
		} else {
			nA, nB, nC := a.n, b.n, c.n
			if len(msh.Normals) > 0 && t.N[0] >= 0 {
				nA = msh.Normals[t.N[0]]
				nB = msh.Normals[t.N[1]]
				nC = msh.Normals[t.N[2]]
			}
			iA, dA = p.studioVert(nA)
			iB, dB = p.studioVert(nB)
			iC, dC = p.studioVert(nC)
		}
	}

	anchorX := float32(minX) + 0.5

	for y := minY; y <= maxY; y++ {
		py := float32(y) + 0.5

		r0 := edge(b.sx, b.sy, c.sx, c.sy, anchorX, py) * invArea
		r1 := edge(c.sx, c.sy, a.sx, a.sy, anchorX, py) * invArea
		r2 := edge(a.sx, a.sy, b.sx, b.sy, anchorX, py) * invArea

		x0, x1 := minX, maxX
		if spanSolve {

			lo, hi := float32(0), float32(wide)
			if !spanBound(a0, r0, &lo, &hi) || !spanBound(a1, r1, &lo, &hi) || !spanBound(a2, r2, &lo, &hi) {
				continue
			}
			x0, x1 = minX+int(ceil(lo)), minX+int(floor(hi))
			if x0 > x1 {
				continue
			}
		}

		rowBase := y * p.w
		off := float32(x0 - minX)
		w0 := r0 + a0*off
		w1 := r1 + a1*off
		w2 := r2 + a2*off
		for x := x0; x <= x1; x, w0, w1, w2 = x+1, w0+a0, w1+a1, w2+a2 {
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}

			depth := w0*a.sz + w1*b.sz + w2*c.sz
			idx := rowBase + x
			if depth >= p.zbuf.data[idx] {
				continue
			}

			col := baseCol
			if useTex {
				var uu, vv float32
				if perspTex {
					iw := 1 / (w0*a.invW + w1*b.invW + w2*c.invW)
					uu = (w0*a.invW*uvA.X + w1*b.invW*uvB.X + w2*c.invW*uvC.X) * iw
					vv = (w0*a.invW*uvA.Y + w1*b.invW*uvB.Y + w2*c.invW*uvC.Y) * iw
				} else {
					uu = w0*uvA.X + w1*uvB.X + w2*uvC.X
					vv = w0*uvA.Y + w1*uvB.Y + w2*uvC.Y
				}
				if hasImg {
					uu -= float32(ffloor(uu))
					vv -= float32(ffloor(vv))
					tx := int(uu * texFW)
					ty := int((1 - vv) * texFH)
					if tx >= texW {
						tx = texW - 1
					}
					if ty >= texH {
						ty = texH - 1
					}
					to := (ty*texW + tx) * 4
					_ = texPix[to+3]
					col = RGBA{texPix[to], texPix[to+1], texPix[to+2], texPix[to+3]}
				} else {
					col = matChecker.ColorAt(uu, vv)
				}
			}
			p.zbuf.data[idx] = depth
			o := idx * 4
			if lit {
				li := w0*iA + w1*iB + w2*iC
				rf := float32(col.R) * li
				gf := float32(col.G) * li
				bf := float32(col.B) * li
				if spec {
					if d := w0*dA + w1*dB + w2*dC; d > 0 {
						d2 := d * d
						d4 := d2 * d2
						d8 := d4 * d4
						d16 := d8 * d8
						s := studioSpec * (d16 * d16 * d16) * 255
						rf += s
						gf += s
						bf += s
					}
				}
				p.pixels[o] = satU8(rf)
				p.pixels[o+1] = satU8(gf)
				p.pixels[o+2] = satU8(bf)
				p.pixels[o+3] = col.A
				continue
			}
			p.pixels[o] = col.R
			p.pixels[o+1] = col.G
			p.pixels[o+2] = col.B
			p.pixels[o+3] = col.A
		}
	}
}

func absDot(n, toLight m.Vec3) float32 {
	d := n.Dot(toLight)
	if d < 0 {
		d = -d
	}
	return d
}

func vertLight(n, toLight m.Vec3) float32 {
	return ambient + (1-ambient)*absDot(n, toLight)
}

func (p *Pipeline) studioVert(n m.Vec3) (diff, sdot float32) {
	if n.Dot(p.stCam) < 0 {
		n = n.Neg()
	}
	hemi := studioGround + (studioSky-studioGround)*(0.5+0.5*n.Dot(p.stUp))
	k := n.Dot(p.stKey)
	if k < 0 {
		k = 0
	}
	diff = hemi + studioKeyStrength*k
	sdot = n.Dot(p.stHalf)
	if sdot < 0 {
		sdot = 0
	}
	return diff, sdot
}

func (p *Pipeline) wireTriangle(a, b, c *vert) {
	p.drawLine(int(a.sx), int(a.sy), int(b.sx), int(b.sy), wireColor)
	p.drawLine(int(b.sx), int(b.sy), int(c.sx), int(c.sy), wireColor)
	p.drawLine(int(c.sx), int(c.sy), int(a.sx), int(a.sy), wireColor)
}

func (p *Pipeline) drawLine(x0, y0, x1, y1 int, col RGBA) {
	x0, y0, x1, y1, ok := clipLine(x0, y0, x1, y1, p.w-1, p.h-1)
	if !ok {
		return
	}
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		p.setPixel(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

const (
	outLeft   = 1
	outRight  = 2
	outTop    = 4
	outBottom = 8
)

func clipLine(ix0, iy0, ix1, iy1, maxX, maxY int) (cx0, cy0, cx1, cy1 int, ok bool) {
	x0, y0 := float32(ix0), float32(iy0)
	x1, y1 := float32(ix1), float32(iy1)
	xm, ym := float32(maxX), float32(maxY)
	code := func(x, y float32) int {
		c := 0
		if x < 0 {
			c |= outLeft
		} else if x > xm {
			c |= outRight
		}
		if y < 0 {
			c |= outTop
		} else if y > ym {
			c |= outBottom
		}
		return c
	}
	c0, c1 := code(x0, y0), code(x1, y1)
	for {
		if c0|c1 == 0 {
			return int(x0), int(y0), int(x1), int(y1), true
		}
		if c0&c1 != 0 {
			return 0, 0, 0, 0, false
		}
		c := c0
		if c == 0 {
			c = c1
		}
		var x, y float32
		switch {
		case c&outLeft != 0:
			x, y = 0, y0+(y1-y0)*(0-x0)/(x1-x0)
		case c&outRight != 0:
			x, y = xm, y0+(y1-y0)*(xm-x0)/(x1-x0)
		case c&outTop != 0:
			x, y = x0+(x1-x0)*(0-y0)/(y1-y0), 0
		default:
			x, y = x0+(x1-x0)*(ym-y0)/(y1-y0), ym
		}
		if c == c0 {
			x0, y0 = x, y
			c0 = code(x0, y0)
		} else {
			x1, y1 = x, y
			c1 = code(x1, y1)
		}
	}
}

func (p *Pipeline) setPixel(x, y int, col RGBA) {
	if x < 0 || y < 0 || x >= p.w || y >= p.h {
		return
	}
	o := (y*p.w + x) * 4
	p.pixels[o] = col.R
	p.pixels[o+1] = col.G
	p.pixels[o+2] = col.B
	p.pixels[o+3] = col.A
}

func spanBound(ai, ri float32, lo, hi *float32) bool {
	switch {
	case ai > 0:
		if x := -ri / ai; x > *lo {
			*lo = x
		}
	case ai < 0:
		if x := -ri / ai; x < *hi {
			*hi = x
		}
	default:
		if ri < 0 {
			return false
		}
	}
	return *lo <= *hi
}

func floor(v float32) float32 { return float32(stdmath.Floor(float64(v))) }
func ceil(v float32) float32  { return float32(stdmath.Ceil(float64(v))) }

func min3(a, b, c float32) float32 {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

func max3(a, b, c float32) float32 {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
