package math

import stdmath "math"

type Vec2 struct{ X, Y float32 }

type Vec3 struct{ X, Y, Z float32 }

type Vec4 struct{ X, Y, Z, W float32 }

func (v Vec2) Add(u Vec2) Vec2      { return Vec2{v.X + u.X, v.Y + u.Y} }
func (v Vec2) Sub(u Vec2) Vec2      { return Vec2{v.X - u.X, v.Y - u.Y} }
func (v Vec2) Scale(s float32) Vec2 { return Vec2{v.X * s, v.Y * s} }

func (v Vec3) Add(u Vec3) Vec3      { return Vec3{v.X + u.X, v.Y + u.Y, v.Z + u.Z} }
func (v Vec3) Sub(u Vec3) Vec3      { return Vec3{v.X - u.X, v.Y - u.Y, v.Z - u.Z} }
func (v Vec3) Scale(s float32) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3) Neg() Vec3            { return Vec3{-v.X, -v.Y, -v.Z} }

func (v Vec3) Dot(u Vec3) float32 { return v.X*u.X + v.Y*u.Y + v.Z*u.Z }

func (v Vec3) Cross(u Vec3) Vec3 {
	return Vec3{
		v.Y*u.Z - v.Z*u.Y,
		v.Z*u.X - v.X*u.Z,
		v.X*u.Y - v.Y*u.X,
	}
}

func (v Vec3) Len() float32 { return float32(stdmath.Sqrt(float64(v.Dot(v)))) }

func (v Vec3) Normalize() Vec3 {
	l := v.Len()
	if l == 0 {
		return v
	}
	inv := 1 / l
	return Vec3{v.X * inv, v.Y * inv, v.Z * inv}
}

func (v Vec3) Rotate(axis Vec3, angle float32) Vec3 {
	c := float32(stdmath.Cos(float64(angle)))
	s := float32(stdmath.Sin(float64(angle)))
	return v.Scale(c).
		Add(axis.Cross(v).Scale(s)).
		Add(axis.Scale(axis.Dot(v) * (1 - c)))
}

func V4(v Vec3, w float32) Vec4 { return Vec4{v.X, v.Y, v.Z, w} }

func (v Vec4) XYZ() Vec3 { return Vec3{v.X, v.Y, v.Z} }
