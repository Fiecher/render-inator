package math

type Mat4 [16]float32

func Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func (a Mat4) Mul(b Mat4) Mat4 {
	var m Mat4
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			m[r*4+c] = a[r*4+0]*b[0*4+c] +
				a[r*4+1]*b[1*4+c] +
				a[r*4+2]*b[2*4+c] +
				a[r*4+3]*b[3*4+c]
		}
	}
	return m
}

func (a Mat4) MulVec4(v Vec4) Vec4 {
	return Vec4{
		a[0]*v.X + a[1]*v.Y + a[2]*v.Z + a[3]*v.W,
		a[4]*v.X + a[5]*v.Y + a[6]*v.Z + a[7]*v.W,
		a[8]*v.X + a[9]*v.Y + a[10]*v.Z + a[11]*v.W,
		a[12]*v.X + a[13]*v.Y + a[14]*v.Z + a[15]*v.W,
	}
}
