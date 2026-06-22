package render

import stdmath "math"

type RGBA struct{ R, G, B, A uint8 }

func linToSRGB(c [4]float32) RGBA {
	return RGBA{srgbByte(c[0]), srgbByte(c[1]), srgbByte(c[2]), satU8(c[3] * 255)}
}

func srgbByte(x float32) uint8 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 255
	}
	var s float32
	if x <= 0.0031308 {
		s = x * 12.92
	} else {
		s = 1.055*float32(stdmath.Pow(float64(x), 1.0/2.4)) - 0.055
	}
	return uint8(s*255 + 0.5)
}

func (c RGBA) scale(f float32) RGBA {
	if f < 0 {
		f = 0
	}
	r := float32(c.R) * f
	g := float32(c.G) * f
	b := float32(c.B) * f
	return RGBA{satU8(r), satU8(g), satU8(b), c.A}
}

func satU8(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v)
}
