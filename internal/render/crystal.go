package render

import (
	stdmath "math"

	m "render-inator/internal/math"
	"render-inator/internal/model"
)

const crystalLUTSize = 256

type RGBf struct{ R, G, B float32 }

var accentf = RGBf{1.0, 0.251, 0.0}

func (p *Pipeline) initCrystal() {

	a := RGBf{0.5, 0.5, 0.5}
	b := RGBf{0.5, 0.5, 0.5}
	c := RGBf{1.0, 1.0, 1.0}
	d := RGBf{0.00, 0.33, 0.67}
	for i := 0; i < crystalLUTSize; i++ {
		t := float32(i) / float32(crystalLUTSize)
		p.crystalLUT[i] = RGBf{
			a.R + b.R*cos32(twoPi*(c.R*t+d.R)),
			a.G + b.G*cos32(twoPi*(c.G*t+d.G)),
			a.B + b.B*cos32(twoPi*(c.B*t+d.B)),
		}
	}
}

const twoPi = 2 * stdmath.Pi

func (p *Pipeline) renderCrystal(msh *model.Mesh, view m.Mat4) {
	p.computeSparkleScale(msh)

	p.crystalCenterView = view.MulVec4(m.V4(p.sparkleCenter, 1)).XYZ()
	coreR := 0.6 / p.sparkleInvR
	if coreR <= 0 {
		coreR = 1
	}
	p.crystalCoreInvR = 1 / coreR

	for i := range p.backVZ {
		p.backVZ[i] = 0
	}

	p.crystalPass(msh, view, true)
	p.zbuf.Clear()
	p.crystalPass(msh, view, false)
}

func (p *Pipeline) computeSparkleScale(msh *model.Mesh) {
	if len(msh.Verts) == 0 {
		p.sparkleCenter, p.sparkleInvR = m.Vec3{}, 1
		return
	}
	lo, hi := msh.Verts[0], msh.Verts[0]
	for _, v := range msh.Verts {
		if v.X < lo.X {
			lo.X = v.X
		} else if v.X > hi.X {
			hi.X = v.X
		}
		if v.Y < lo.Y {
			lo.Y = v.Y
		} else if v.Y > hi.Y {
			hi.Y = v.Y
		}
		if v.Z < lo.Z {
			lo.Z = v.Z
		} else if v.Z > hi.Z {
			hi.Z = v.Z
		}
	}
	p.sparkleCenter = lo.Add(hi).Scale(0.5)
	r := hi.Sub(lo).Len() * 0.5
	if r == 0 {
		r = 1
	}
	p.sparkleInvR = 1 / r
}

func (p *Pipeline) crystalPass(msh *model.Mesh, view m.Mat4, back bool) {
	for ti := range msh.Tris {
		t := &msh.Tris[ti]
		a, b, c := &p.vs[t.V[0]], &p.vs[t.V[1]], &p.vs[t.V[2]]
		if !a.ok || !b.ok || !c.ok {
			continue
		}
		area := edge(a.sx, a.sy, b.sx, b.sy, c.sx, c.sy)
		if area == 0 {
			continue
		}
		isFront := area < 0
		if back == isFront {
			continue
		}

		n := view.MulVec4(m.V4(msh.FaceNormals[ti], 0)).XYZ().Normalize()
		cx := (a.vx + b.vx + c.vx) / 3
		cy := (a.vy + b.vy + c.vy) / 3
		cz := (a.vz + b.vz + c.vz) / 3
		toCam := m.Vec3{X: -cx, Y: -cy, Z: -cz}.Normalize()
		if n.Dot(toCam) < 0 {
			n = n.Neg()
		}

		hue := atan2_32(n.Y, n.X)*(1/twoPi) + 0.5
		hue -= floor32(hue)
		li := int(hue*crystalLUTSize) & (crystalLUTSize - 1)
		rainbow := p.crystalLUT[li]

		p.crystalTriangle(a, b, c, n, rainbow, back)
	}
}

func (p *Pipeline) crystalTriangle(a, b, c *vert, n m.Vec3, rainbow RGBf, back bool) {
	area := edge(a.sx, a.sy, b.sx, b.sy, c.sx, c.sy)
	invArea := 1 / area

	fw, fh := float32(p.w), float32(p.h)
	if max3(a.sx, b.sx, c.sx) < 0 || min3(a.sx, b.sx, c.sx) >= fw ||
		max3(a.sy, b.sy, c.sy) < 0 || min3(a.sy, b.sy, c.sy) >= fh {
		return
	}

	a0 := (c.sy - b.sy) * invArea
	a1 := (a.sy - c.sy) * invArea
	a2 := (b.sy - a.sy) * invArea

	minX := clampInt(int(floor(min3(a.sx, b.sx, c.sx))), 0, p.w-1)
	maxX := clampInt(int(floor(max3(a.sx, b.sx, c.sx))), 0, p.w-1)
	minY := clampInt(int(floor(min3(a.sy, b.sy, c.sy))), 0, p.h-1)
	maxY := clampInt(int(floor(max3(a.sy, b.sy, c.sy))), 0, p.h-1)

	wide := maxX - minX
	spanSolve := wide >= 16

	p.spHas = false
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

			vx := w0*a.vx + w1*b.vx + w2*c.vx
			vy := w0*a.vy + w1*b.vy + w2*c.vy
			vz := w0*a.vz + w1*b.vz + w2*c.vz
			vdir := m.Vec3{X: -vx, Y: -vy, Z: -vz}.Normalize()

			p.zbuf.data[idx] = depth
			o := idx * 4
			if back {
				ndv := n.Dot(vdir)
				if ndv < 0 {
					ndv = 0
				}
				const dim = 0.40
				p.pixels[o] = satU8(accentf.R * dim * (0.4 + 0.6*ndv) * 255)
				p.pixels[o+1] = satU8(accentf.G * dim * 0.5 * ndv * 255)
				p.pixels[o+2] = 0
				p.pixels[o+3] = 255
				p.backVZ[idx] = vz
				continue
			}

			obj := m.Vec3{
				X: w0*a.ox + w1*b.ox + w2*c.ox,
				Y: w0*a.oy + w1*b.oy + w2*c.oy,
				Z: w0*a.oz + w1*b.oz + w2*c.oz,
			}
			col, alpha := p.crystalShadeFront(n, vdir, obj, rainbow, vz, idx)
			p.blendPixel(o, col, alpha)
		}
	}
}

func (p *Pipeline) crystalShadeFront(n, v, obj m.Vec3, rainbow RGBf, vz float32, idx int) (RGBf, float32) {

	ndv := n.Dot(v)
	if ndv < 0 {
		ndv = 0
	}
	fres := 1 - ndv
	rim := fres * fres

	half := m.Vec3{X: keyLight.X + v.X, Y: keyLight.Y + v.Y, Z: keyLight.Z + v.Z}.Normalize()
	s := n.Dot(half)
	if s < 0 {
		s = 0
	}
	s2 := s * s
	s4 := s2 * s2
	spec := s4 * s4 * s4 * s4

	body := 0.30 + 0.70*ndv
	col := RGBf{accentf.R * body, accentf.G * body, accentf.B * body}

	rimGain := 1.4 * rim
	col.R += (rainbow.R - col.R) * rim
	col.G += (rainbow.G - col.G) * rim
	col.B += (rainbow.B - col.B) * rim
	col.R += rimGain*rainbow.R + spec
	col.G += rimGain*rainbow.G + spec
	col.B += rimGain*rainbow.B + spec

	thick := (vz - p.backVZ[idx]) * p.sparkleInvR
	if thick < 0 {
		thick = 0
	}

	kt := absorbK * thick
	absorb := kt / (1 + kt)
	dense := RGBf{accentf.R * 0.30, accentf.G * 0.09, 0.015}
	at := absorb * absorbTint
	col.R += (dense.R - col.R) * at
	col.G += (dense.G - col.G) * at
	col.B += (dense.B - col.B) * at

	d2 := perpDist2(p.crystalCenterView, v) * (p.crystalCoreInvR * p.crystalCoreInvR)
	core := 1 - d2
	if core < 0 {
		core = 0
	}
	core *= core
	col.R += coreCol.R * core * coreGain
	col.G += coreCol.G * core * coreGain
	col.B += coreCol.B * core * coreGain

	gx := fastFloorI((obj.X - p.sparkleCenter.X) * p.sparkleInvR * sparkleDensity)
	gy := fastFloorI((obj.Y - p.sparkleCenter.Y) * p.sparkleInvR * sparkleDensity)
	gz := fastFloorI((obj.Z - p.sparkleCenter.Z) * p.sparkleInvR * sparkleDensity)
	var rn m.Vec3
	if p.spHas && gx == p.spCellX && gy == p.spCellY && gz == p.spCellZ {
		rn = p.spRN
	} else {
		rd := sparkleCellDir(gx, gy, gz)
		rn = m.Vec3{X: rd.X + n.X, Y: rd.Y + n.Y, Z: rd.Z + n.Z}.Normalize()
		p.spCellX, p.spCellY, p.spCellZ, p.spRN, p.spHas = gx, gy, gz, rn, true
	}
	sd := v.Dot(rn)
	if sd > 0 {
		sd2 := sd * sd
		sd4 := sd2 * sd2
		sd8 := sd4 * sd4
		sd16 := sd8 * sd8
		sd32 := sd16 * sd16
		sparkle := sd32 * sd32
		sparkle *= 3.5
		col.R += sparkle
		col.G += sparkle
		col.B += sparkle
	}

	alpha := 0.30 + 0.65*rim + spec
	alpha += (1 - alpha) * absorb * absorbAlpha
	alpha += core * coreAlpha
	if alpha > 1 {
		alpha = 1
	}
	return col, alpha
}

const sparkleDensity = 70

func sparkleCellDir(gx, gy, gz int32) m.Vec3 {
	h := uint32(gx)*73856093 ^ uint32(gy)*19349663 ^ uint32(gz)*83492791
	h1 := hashU32(h)
	h2 := hashU32(h1)
	h3 := hashU32(h2)

	return m.Vec3{
		X: unitFromHash(h1) - 0.5,
		Y: unitFromHash(h2) - 0.5,
		Z: unitFromHash(h3) - 0.5,
	}.Normalize()
}

const (
	absorbK     = 1.8
	absorbTint  = 0.85
	absorbAlpha = 0.80
	coreGain    = 0.90
	coreAlpha   = 0.35
)

var coreCol = RGBf{1.0, 0.45, 0.15}

func perpDist2(c, v m.Vec3) float32 {
	t := c.Dot(v)
	dx := c.X - t*v.X
	dy := c.Y - t*v.Y
	dz := c.Z - t*v.Z
	return dx*dx + dy*dy + dz*dz
}

var keyLight = m.Vec3{X: 0.4, Y: 0.6, Z: 0.7}.Normalize()

func (p *Pipeline) blendPixel(o int, src RGBf, alpha float32) {
	ia := 1 - alpha
	dr := float32(p.pixels[o])
	dg := float32(p.pixels[o+1])
	db := float32(p.pixels[o+2])
	p.pixels[o] = satU8(src.R*255*alpha + dr*ia)
	p.pixels[o+1] = satU8(src.G*255*alpha + dg*ia)
	p.pixels[o+2] = satU8(src.B*255*alpha + db*ia)
	p.pixels[o+3] = 255
}

func cos32(v float32) float32   { return float32(stdmath.Cos(float64(v))) }
func floor32(v float32) float32 { return float32(stdmath.Floor(float64(v))) }

func fastFloorI(x float32) int32 {
	i := int32(x)
	if x < float32(i) {
		i--
	}
	return i
}
func atan2_32(y, x float32) float32 {
	return float32(stdmath.Atan2(float64(y), float64(x)))
}

func hashU32(h uint32) uint32 {
	h ^= h >> 16
	h *= 0x7feb352d
	h ^= h >> 15
	h *= 0x846ca68b
	h ^= h >> 16
	return h
}

func unitFromHash(h uint32) float32 { return float32(h&0xffffff) / float32(0x1000000) }
