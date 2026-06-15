package render

type RGBA struct{ R, G, B, A uint8 }

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
