package render

type Texture interface {
	ColorAt(u, v float32) RGBA
}

func ffloor(f float32) int {
	i := int(f)
	if float32(i) > f {
		i--
	}
	return i
}

type Checker struct {
	A, B  RGBA
	Scale float32
}

func (c Checker) ColorAt(u, v float32) RGBA {
	iu := ffloor(u * c.Scale)
	iv := ffloor(v * c.Scale)
	if (iu+iv)&1 == 0 {
		return c.A
	}
	return c.B
}

type mipLevel struct {
	pix  []byte
	w, h int
}

type ImageTexture struct {
	Pix    []byte
	W, H   int
	levels []mipLevel
}

func NewImageTexture(pix []byte, w, h int) *ImageTexture {
	t := &ImageTexture{Pix: pix, W: w, H: h}
	t.buildMips()
	return t
}

func (t *ImageTexture) buildMips() {
	t.levels = t.levels[:0]
	if t.W <= 0 || t.H <= 0 || len(t.Pix) < t.W*t.H*4 {
		return
	}
	t.levels = append(t.levels, mipLevel{pix: t.Pix, w: t.W, h: t.H})
	pw, ph, prev := t.W, t.H, t.Pix
	for pw > 1 || ph > 1 {
		nw, nh := pw/2, ph/2
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
		dst := make([]byte, nw*nh*4)
		for y := 0; y < nh; y++ {
			sy0 := y * 2
			sy1 := sy0 + 1
			if sy1 >= ph {
				sy1 = ph - 1
			}
			for x := 0; x < nw; x++ {
				sx0 := x * 2
				sx1 := sx0 + 1
				if sx1 >= pw {
					sx1 = pw - 1
				}
				o00 := (sy0*pw + sx0) * 4
				o01 := (sy0*pw + sx1) * 4
				o10 := (sy1*pw + sx0) * 4
				o11 := (sy1*pw + sx1) * 4
				d := (y*nw + x) * 4
				dst[d] = byte((uint32(prev[o00]) + uint32(prev[o01]) + uint32(prev[o10]) + uint32(prev[o11])) >> 2)
				dst[d+1] = byte((uint32(prev[o00+1]) + uint32(prev[o01+1]) + uint32(prev[o10+1]) + uint32(prev[o11+1])) >> 2)
				dst[d+2] = byte((uint32(prev[o00+2]) + uint32(prev[o01+2]) + uint32(prev[o10+2]) + uint32(prev[o11+2])) >> 2)
				dst[d+3] = byte((uint32(prev[o00+3]) + uint32(prev[o01+3]) + uint32(prev[o10+3]) + uint32(prev[o11+3])) >> 2)
			}
		}
		t.levels = append(t.levels, mipLevel{pix: dst, w: nw, h: nh})
		prev = dst
		pw, ph = nw, nh
	}
}

func (t *ImageTexture) mipFor(texelArea, screenArea float32) ([]byte, int, int, float32, float32) {
	n := len(t.levels)
	if n == 0 {
		return t.Pix, t.W, t.H, float32(t.W), float32(t.H)
	}
	lvl := 0
	if n > 1 && screenArea > 0 {
		ratio := texelArea / screenArea
		for ratio > 2 && lvl < n-1 {
			ratio *= 0.25
			lvl++
		}
	}
	ml := t.levels[lvl]
	return ml.pix, ml.w, ml.h, float32(ml.w), float32(ml.h)
}

func (t *ImageTexture) ColorAt(u, v float32) RGBA {
	if t.W == 0 || t.H == 0 {
		return RGBA{255, 0, 255, 255}
	}
	u -= float32(ffloor(u))
	v -= float32(ffloor(v))
	x := int(u * float32(t.W))
	y := int((1 - v) * float32(t.H))
	if x >= t.W {
		x = t.W - 1
	}
	if y >= t.H {
		y = t.H - 1
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	o := (y*t.W + x) * 4
	return RGBA{t.Pix[o], t.Pix[o+1], t.Pix[o+2], t.Pix[o+3]}
}
