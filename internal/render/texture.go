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

type ImageTexture struct {
	Pix  []byte
	W, H int
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
