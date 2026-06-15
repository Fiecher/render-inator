package math

import stdmath "math"

func Translate(t Vec3) Mat4 {
	m := Identity()
	m[3] = t.X
	m[7] = t.Y
	m[11] = t.Z
	return m
}

func Scale(s Vec3) Mat4 {
	m := Identity()
	m[0] = s.X
	m[5] = s.Y
	m[10] = s.Z
	return m
}

func Perspective(fovY, aspect, near, far float32) Mat4 {
	f := float32(1 / stdmath.Tan(float64(fovY)/2))
	var m Mat4
	m[0] = f / aspect
	m[5] = f
	m[10] = (far + near) / (near - far)
	m[11] = (2 * far * near) / (near - far)
	m[14] = -1
	return m
}

func LookAt(eye, center, up Vec3) Mat4 {
	f := center.Sub(eye).Normalize()
	s := f.Cross(up).Normalize()
	u := s.Cross(f)

	return Mat4{
		s.X, s.Y, s.Z, -s.Dot(eye),
		u.X, u.Y, u.Z, -u.Dot(eye),
		-f.X, -f.Y, -f.Z, f.Dot(eye),
		0, 0, 0, 1,
	}
}
