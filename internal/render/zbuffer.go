package render

import "math"

type ZBuffer struct {
	data []float32
	w, h int
}

func NewZBuffer(w, h int) *ZBuffer {
	z := &ZBuffer{}
	z.Resize(w, h)
	return z
}

func (z *ZBuffer) Resize(w, h int) {
	z.w, z.h = w, h
	n := w * h
	if cap(z.data) < n {
		z.data = make([]float32, n)
	} else {
		z.data = z.data[:n]
	}
}

func (z *ZBuffer) Clear() {
	for i := range z.data {
		z.data[i] = math.MaxFloat32
	}
}
