//go:build js && wasm

package main

import (
	stdmath "math"
	"syscall/js"
	"time"

	"render-inator/internal/camera"
	m "render-inator/internal/math"
	"render-inator/internal/model"
	"render-inator/internal/render"
)

const (
	fovY             = 60 * stdmath.Pi / 180
	orbitSensitivity = 0.005
)

var (
	pipe *render.Pipeline
	cam  *camera.Camera
	mesh *model.Mesh

	jsPix     js.Value
	jsClamped js.Value

	lastFrame time.Time
)

func main() {
	cam = camera.New(m.Vec3{}, 5, fovY, 1)
	pipe = render.NewPipeline(640, 480)
	allocPixelBuffer(640, 480)

	g := js.Global()
	g.Set("goLoadModel", js.FuncOf(loadModel))
	g.Set("goClearModel", js.FuncOf(clearModel))
	g.Set("goLoadTexture", js.FuncOf(loadTexture))
	g.Set("goSetRenderConfig", js.FuncOf(setRenderConfig))
	g.Set("goResizeBuffer", js.FuncOf(resizeBuffer))
	g.Set("goInputCamera", js.FuncOf(inputCamera))
	g.Set("goResetCamera", js.FuncOf(resetCamera))
	g.Set("goRenderFrame", js.FuncOf(renderFrame))
	g.Set("goGetPixelBuffer", js.FuncOf(getPixelBuffer))

	if cb := g.Get("onWasmReady"); cb.Type() == js.TypeFunction {
		cb.Invoke()
	}
	select {}
}

func loadModel(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return "loadModel: missing obj text"
	}
	msh, err := model.ParseOBJ([]byte(args[0].String()))
	if err != nil {
		return err.Error()
	}
	mesh = msh
	pipe.ResetTexture()
	frameCamera(msh)
	cam.Snap()
	return nil
}

func clearModel(_ js.Value, _ []js.Value) any {
	mesh = nil
	return nil
}

func resetCamera(_ js.Value, _ []js.Value) any {
	if mesh != nil {
		frameCamera(mesh)
	}
	return nil
}

func loadTexture(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return "loadTexture: want (rgba, w, h)"
	}
	w, h := args[1].Int(), args[2].Int()
	if w <= 0 || h <= 0 {
		return "loadTexture: bad dimensions"
	}
	pix := make([]byte, w*h*4)
	if n := js.CopyBytesToGo(pix, args[0]); n != len(pix) {
		return "loadTexture: rgba byte count mismatch"
	}
	pipe.SetTexture(&render.ImageTexture{Pix: pix, W: w, H: h})
	return nil
}

func setRenderConfig(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return nil
	}
	cull := true
	if len(args) >= 4 {
		cull = args[3].Bool()
	}
	crystal := false
	if len(args) >= 5 {
		crystal = args[4].Bool()
	}
	flat := false
	if len(args) >= 6 {
		flat = args[5].Bool()
	}
	pipe.SetConfig(render.RenderConfig{
		Wireframe: args[0].Bool(),
		Texture:   args[1].Bool(),
		Lighting:  args[2].Bool(),
		Cull:      cull,
		Crystal:   crystal,
		Flat:      flat,
	})
	return nil
}

func resizeBuffer(_ js.Value, args []js.Value) any {
	w, h := args[0].Int(), args[1].Int()
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	pipe.Resize(w, h)
	cam.Aspect = float32(w) / float32(h)
	allocPixelBuffer(w, h)
	return nil
}

func inputCamera(_ js.Value, args []js.Value) any {
	if len(args) < 5 {
		return nil
	}
	orbitDX := float32(args[0].Float())
	orbitDY := float32(args[1].Float())
	panDX := float32(args[2].Float())
	panDY := float32(args[3].Float())
	zoom := float32(args[4].Float())

	cam.Orbit(-orbitDX*orbitSensitivity, orbitDY*orbitSensitivity)
	if panDX != 0 || panDY != 0 {

		_, h := pipe.Size()
		wpp := 2 * cam.Dist * tanf(fovY/2) / float32(h)
		cam.Pan(-panDX*wpp, panDY*wpp)
	}
	cam.Zoom(zoom)
	return nil
}

func tanf(v float32) float32 { return float32(stdmath.Tan(float64(v))) }

func renderFrame(_ js.Value, _ []js.Value) any {
	now := time.Now()
	dt := float32(now.Sub(lastFrame).Seconds())
	lastFrame = now
	if dt > 0.1 {
		dt = 0.1
	}
	cam.Update(dt)
	pipe.Render(mesh, cam)
	return nil
}

func getPixelBuffer(_ js.Value, _ []js.Value) any {
	js.CopyBytesToJS(jsPix, pipe.Pixels())
	return jsClamped
}

func allocPixelBuffer(w, h int) {
	n := w * h * 4
	jsPix = js.Global().Get("Uint8Array").New(n)
	jsClamped = js.Global().Get("Uint8ClampedArray").New(jsPix.Get("buffer"))
}

func frameCamera(msh *model.Mesh) {
	lo, hi := msh.Verts[0], msh.Verts[0]
	for _, v := range msh.Verts {
		lo = m.Vec3{X: minF(lo.X, v.X), Y: minF(lo.Y, v.Y), Z: minF(lo.Z, v.Z)}
		hi = m.Vec3{X: maxF(hi.X, v.X), Y: maxF(hi.Y, v.Y), Z: maxF(hi.Z, v.Z)}
	}
	center := lo.Add(hi).Scale(0.5)
	radius := hi.Sub(lo).Len() * 0.5
	if radius == 0 {
		radius = 1
	}
	dist := radius/tanf(fovY/2)*1.4 + radius

	cam.Target = center
	cam.Yaw, cam.Pitch = 0, 0
	cam.Dist = dist
	cam.MinDist = radius * 0.2
	cam.MaxDist = dist * 8
	cam.Near = radius * 0.01
	cam.Far = cam.MaxDist + radius*4
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
