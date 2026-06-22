package render

import (
	"time"

	m "render-inator/internal/math"
	"render-inator/internal/model"
)

type Camera interface {
	View() m.Mat4
	Projection() m.Mat4
	LightDir() m.Vec3
}

type vert struct {
	sx, sy     float32
	sz         float32
	invW       float32
	n          m.Vec3
	vx, vy, vz float32
	ox, oy, oz float32
	ok         bool
}

const nearEps = 1e-4

type Pipeline struct {
	w, h   int
	pixels []byte
	zbuf   *ZBuffer
	vs     []vert

	cfg  RenderConfig
	tex  Texture
	flat RGBA

	meshMats   []model.Material
	meshMatCol []RGBA
	meshTex    []*ImageTexture

	cullSign int8

	bgBuf []byte

	crystalLUT [crystalLUTSize]RGBf

	sparkleCenter m.Vec3
	sparkleInvR   float32

	backVZ            []float32
	crystalCenterView m.Vec3
	crystalCoreInvR   float32

	spCellX, spCellY, spCellZ int32
	spRN                      m.Vec3
	spCenter                  m.Vec3
	spTw                      float32
	spHas                     bool

	crystalTime float32

	stKey  m.Vec3
	stHalf m.Vec3
	stUp   m.Vec3
	stCam  m.Vec3

	reveal      wireReveal
	prevWire    bool
	forceReveal bool
}

func NewPipeline(w, h int) *Pipeline {
	p := &Pipeline{
		zbuf: NewZBuffer(w, h),
		flat: RGBA{200, 200, 205, 255},
		tex:  defaultTexture(),
	}
	p.initCrystal()
	p.Resize(w, h)
	return p
}

func (p *Pipeline) Resize(w, h int) {
	p.w, p.h = w, h
	n := w * h * 4
	if cap(p.pixels) < n {
		p.pixels = make([]byte, n)
	} else {
		p.pixels = p.pixels[:n]
	}
	p.zbuf.Resize(w, h)
	if cap(p.backVZ) < w*h {
		p.backVZ = make([]float32, w*h)
	} else {
		p.backVZ = p.backVZ[:w*h]
	}
	p.buildBackground()
}

func (p *Pipeline) buildBackground() {
	n := p.w * p.h * 4
	if cap(p.bgBuf) < n {
		p.bgBuf = make([]byte, n)
	} else {
		p.bgBuf = p.bgBuf[:n]
	}
	if p.w == 0 || p.h == 0 {
		return
	}

	const (
		cR, cG, cB = float32(36), float32(30), float32(33)
		eR, eG, eB = float32(10), float32(10), float32(14)
	)
	fw, fh := float32(p.w), float32(p.h)
	cx, cy := fw*0.5, fh*0.5
	inv := 1 / (cx*cx + cy*cy)
	i := 0
	for y := 0; y < p.h; y++ {
		dy := float32(y) + 0.5 - cy
		for x := 0; x < p.w; x++ {
			dx := float32(x) + 0.5 - cx
			d2 := (dx*dx + dy*dy) * inv
			if d2 > 1 {
				d2 = 1
			}

			s := 1 - d2
			v := s * s * (3 - 2*s)
			p.bgBuf[i] = byte(eR + (cR-eR)*v)
			p.bgBuf[i+1] = byte(eG + (cG-eG)*v)
			p.bgBuf[i+2] = byte(eB + (cB-eB)*v)
			p.bgBuf[i+3] = 255
			i += 4
		}
	}
}

func (p *Pipeline) SetConfig(c RenderConfig) { p.cfg = c }

func (p *Pipeline) ReplayWireframe() { p.forceReveal = true }

func (p *Pipeline) Config() RenderConfig { return p.cfg }

func (p *Pipeline) SetTexture(t Texture) { p.tex = t }

func (p *Pipeline) ResetTexture() { p.tex = defaultTexture() }

func (p *Pipeline) SetModel(msh *model.Mesh) {
	p.meshMats = msh.Materials
	p.meshMatCol = p.meshMatCol[:0]
	for i := range msh.Materials {
		p.meshMatCol = append(p.meshMatCol, linToSRGB(msh.Materials[i].BaseColor))
	}
	p.meshTex = p.meshTex[:0]
	for i := range msh.Images {
		img := &msh.Images[i]
		if img.W > 0 && img.H > 0 && len(img.Pix) >= img.W*img.H*4 {
			p.meshTex = append(p.meshTex, NewImageTexture(img.Pix, img.W, img.H))
		} else {
			p.meshTex = append(p.meshTex, nil)
		}
	}
}

func (p *Pipeline) triMaterial(t *model.Tri) (base RGBA, img *ImageTexture, checker Texture) {
	if len(p.meshMats) == 0 {
		base = p.flat
		if it, ok := p.tex.(*ImageTexture); ok && it.W > 0 && it.H > 0 {
			img = it
		} else {
			checker = p.tex
		}
		return
	}
	base = RGBA{200, 200, 205, 255}
	if mi := t.Mat; mi >= 0 && mi < len(p.meshMatCol) {
		base = p.meshMatCol[mi]
		if im := p.meshMats[mi].Image; im >= 0 && im < len(p.meshTex) {
			img = p.meshTex[im]
		}
	}
	return
}

func defaultTexture() Texture {
	return Checker{A: RGBA{40, 40, 48, 255}, B: RGBA{220, 220, 230, 255}, Scale: 8}
}

func (p *Pipeline) Pixels() []byte { return p.pixels }

func (p *Pipeline) Size() (w, h int) { return p.w, p.h }

func (p *Pipeline) Render(msh *model.Mesh, cam Camera) {
	p.clear()
	p.zbuf.Clear()
	if msh == nil {
		p.prevWire = false
		return
	}

	view := cam.View()
	vp := cam.Projection().Mul(view)

	p.projectVerts(msh, vp, view)

	if p.cfg.Crystal {
		p.renderCrystal(msh, view)
		p.prevWire = false
		return
	}

	p.cullSign = 0
	if msh.CullSafe {
		if msh.WindingOutward {
			p.cullSign = -1
		} else {
			p.cullSign = 1
		}
	}

	toLight := cam.LightDir().Normalize().Neg()
	p.setupStudio(view)

	wire := p.cfg.Wireframe
	animating := false
	if wire {
		if !p.prevWire || p.reveal.builtFor != msh || p.forceReveal {
			p.reveal.begin(msh)
		}
		animating = p.reveal.active
	}
	p.forceReveal = false
	p.prevWire = wire

	wireOnly := wire && !p.cfg.Texture && !p.cfg.Lighting
	if !(wireOnly && animating) {
		for ti := range msh.Tris {
			t := &msh.Tris[ti]
			a, b, c := &p.vs[t.V[0]], &p.vs[t.V[1]], &p.vs[t.V[2]]
			if !a.ok || !b.ok || !c.ok {
				continue
			}
			if !wireOnly {
				p.rasterizeTriangle(msh, ti, t, a, b, c, toLight)
			}
			if wire && !animating {
				p.wireTriangle(a, b, c)
			}
		}
	}
	if animating {
		p.reveal.draw(p, time.Now())
	}
}

func (p *Pipeline) projectVerts(msh *model.Mesh, vp, view m.Mat4) {
	if cap(p.vs) < len(msh.Verts) {
		p.vs = make([]vert, len(msh.Verts))
	} else {
		p.vs = p.vs[:len(msh.Verts)]
	}
	fw, fh := float32(p.w), float32(p.h)
	crystal := p.cfg.Crystal
	for i := range msh.Verts {
		clip := vp.MulVec4(m.V4(msh.Verts[i], 1))
		v := &p.vs[i]
		if clip.W <= nearEps {
			v.ok = false
			continue
		}
		inv := 1 / clip.W
		v.sx = (clip.X*inv*0.5 + 0.5) * fw
		v.sy = (1 - (clip.Y*inv*0.5 + 0.5)) * fh
		v.sz = clip.Z * inv
		v.invW = inv
		v.n = msh.VertNormals[i]
		if crystal {
			vp4 := view.MulVec4(m.V4(msh.Verts[i], 1))
			v.vx, v.vy, v.vz = vp4.X, vp4.Y, vp4.Z
			v.ox, v.oy, v.oz = msh.Verts[i].X, msh.Verts[i].Y, msh.Verts[i].Z
		}
		v.ok = true
	}
}

func (p *Pipeline) setupStudio(view m.Mat4) {
	right := m.Vec3{X: view[0], Y: view[1], Z: view[2]}
	up := m.Vec3{X: view[4], Y: view[5], Z: view[6]}
	toCam := m.Vec3{X: view[8], Y: view[9], Z: view[10]}
	key := up.Scale(0.45).Add(right.Scale(0.2)).Add(toCam).Normalize()
	p.stKey = key
	p.stHalf = key.Add(toCam).Normalize()
	p.stUp = up
	p.stCam = toCam
}

func (p *Pipeline) clear() {

	copy(p.pixels, p.bgBuf)
}
