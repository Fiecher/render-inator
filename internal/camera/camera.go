package camera

import (
	stdmath "math"

	m "render-inator/internal/math"
)

const maxPitch = 1.55

const smoothRate = 12

var worldUp = m.Vec3{X: 0, Y: 1, Z: 0}

type Camera struct {
	Target m.Vec3
	Yaw    float32
	Pitch  float32
	Dist   float32

	FOV    float32
	Aspect float32
	Near   float32
	Far    float32

	MinDist float32
	MaxDist float32

	curTarget m.Vec3
	curYaw    float32
	curPitch  float32
	curDist   float32
}

func New(target m.Vec3, dist, fovY, aspect float32) *Camera {
	c := &Camera{
		Target:  target,
		Dist:    dist,
		FOV:     fovY,
		Aspect:  aspect,
		Near:    0.1,
		Far:     1000,
		MinDist: 0.1,
		MaxDist: 1000,
	}
	c.Snap()
	return c
}

func (c *Camera) Pos() m.Vec3 {
	cp := cosf(c.curPitch)
	return c.curTarget.Add(m.Vec3{
		X: c.curDist * cp * sinf(c.curYaw),
		Y: c.curDist * sinf(c.curPitch),
		Z: c.curDist * cp * cosf(c.curYaw),
	})
}

func (c *Camera) Update(dt float32) {
	k := 1 - expf(-smoothRate*dt)
	c.curYaw += (c.Yaw - c.curYaw) * k
	c.curPitch += (c.Pitch - c.curPitch) * k
	c.curDist += (c.Dist - c.curDist) * k
	c.curTarget = c.curTarget.Add(c.Target.Sub(c.curTarget).Scale(k))
}

func (c *Camera) Snap() {
	c.curTarget, c.curYaw, c.curPitch, c.curDist = c.Target, c.Yaw, c.Pitch, c.Dist
}

func (c *Camera) Orbit(dYaw, dPitch float32) {
	c.Yaw += dYaw
	c.Pitch = clamp(c.Pitch+dPitch, -maxPitch, maxPitch)
}

func (c *Camera) Zoom(factor float32) {
	if factor <= 0 {
		return
	}
	c.Dist = clamp(c.Dist*factor, c.MinDist, c.MaxDist)
}

func (c *Camera) Pan(dx, dy float32) {
	f := c.fwd()
	right := f.Cross(worldUp).Normalize()
	up := right.Cross(f).Normalize()
	c.Target = c.Target.Add(right.Scale(dx)).Add(up.Scale(dy))
}

func (c *Camera) View() m.Mat4 { return m.LookAt(c.Pos(), c.curTarget, worldUp) }

func (c *Camera) Projection() m.Mat4 {
	return m.Perspective(c.FOV, c.Aspect, c.Near, c.Far)
}

func (c *Camera) LightDir() m.Vec3 { return c.fwd() }

func (c *Camera) fwd() m.Vec3 { return c.curTarget.Sub(c.Pos()).Normalize() }

func sinf(v float32) float32 { return float32(stdmath.Sin(float64(v))) }
func cosf(v float32) float32 { return float32(stdmath.Cos(float64(v))) }
func expf(v float32) float32 { return float32(stdmath.Exp(float64(v))) }

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
