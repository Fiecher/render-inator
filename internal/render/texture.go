package render

import "math"

type Texture interface {
	ColorAt(u, v float32) RGBA
}

type Checker struct {
	A, B  RGBA
	Scale float32
}

func (c Checker) ColorAt(u, v float32) RGBA {
	iu := int(math.Floor(float64(u * c.Scale)))
	iv := int(math.Floor(float64(v * c.Scale)))
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
	u -= float32(math.Floor(float64(u)))
	v -= float32(math.Floor(float64(v)))
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
