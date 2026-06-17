package render

import (
	stdmath "math"
	"testing"

	"render-inator/internal/camera"
	m "render-inator/internal/math"
	"render-inator/internal/model"
)

const (
	benchW = 900
	benchH = 900
)

func buildSphere(stacks, slices int) *model.Mesh {
	msh := &model.Mesh{}
	w := slices + 1
	for i := 0; i <= stacks; i++ {
		phi := stdmath.Pi * float64(i) / float64(stacks)
		sp, cp := stdmath.Sincos(phi)
		for j := 0; j <= slices; j++ {
			theta := 2 * stdmath.Pi * float64(j) / float64(slices)
			st, ct := stdmath.Sincos(theta)
			msh.Verts = append(msh.Verts, m.Vec3{X: float32(sp * ct), Y: float32(cp), Z: float32(sp * st)})
			msh.UVs = append(msh.UVs, m.Vec2{X: float32(j) / float32(slices), Y: float32(i) / float32(stacks)})
		}
	}
	for i := 0; i < stacks; i++ {
		for j := 0; j < slices; j++ {
			a := i*w + j
			b := a + 1
			c := a + w
			d := c + 1
			msh.Tris = append(msh.Tris, model.Tri{V: [3]int{a, c, b}, UV: [3]int{a, c, b}})
			msh.Tris = append(msh.Tris, model.Tri{V: [3]int{b, c, d}, UV: [3]int{b, c, d}})
		}
	}
	msh.ComputeNormals()
	msh.CullSafe = true
	msh.WindingOutward = true
	return msh
}

func buildImageTex(size int) *ImageTexture {
	pix := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			o := (y*size + x) * 4
			if (x/16+y/16)&1 == 0 {
				pix[o], pix[o+1], pix[o+2] = 210, 180, 140
			} else {
				pix[o], pix[o+1], pix[o+2] = 60, 70, 90
			}
			pix[o+3] = 255
		}
	}
	return NewImageTexture(pix, size, size)
}

func benchRenderMesh(b *testing.B, cfg RenderConfig, msh *model.Mesh, dist float32, texSize int) {
	p := NewPipeline(benchW, benchH)
	p.SetConfig(cfg)
	if cfg.Texture {
		p.SetTexture(buildImageTex(texSize))
	}
	cam := camera.New(m.Vec3{}, dist, 60*stdmath.Pi/180, float32(benchW)/float32(benchH))
	cam.Snap()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Render(msh, cam)
	}
}

func benchRenderCfg(b *testing.B, cfg RenderConfig, dist float32, texSize int) {
	msh := buildSphere(96, 96)
	p := NewPipeline(benchW, benchH)
	p.SetConfig(cfg)
	if cfg.Texture {
		p.SetTexture(buildImageTex(texSize))
	}
	cam := camera.New(m.Vec3{}, dist, 60*stdmath.Pi/180, float32(benchW)/float32(benchH))
	cam.Snap()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Render(msh, cam)
	}
}

func benchRender(b *testing.B, cfg RenderConfig) { benchRenderCfg(b, cfg, 2.6, 1024) }

func BenchmarkMaterial(b *testing.B)    { benchRender(b, RenderConfig{Lighting: true}) }
func BenchmarkMaterialTex(b *testing.B) { benchRender(b, RenderConfig{Lighting: true, Texture: true}) }
func BenchmarkSolid(b *testing.B)       { benchRender(b, RenderConfig{Lighting: true, Flat: true}) }
func BenchmarkSolidTex(b *testing.B) {
	benchRender(b, RenderConfig{Lighting: true, Texture: true, Flat: true})
}

func BenchmarkMaterialDistant(b *testing.B) {
	benchRenderCfg(b, RenderConfig{Lighting: true}, 9, 1024)
}
func BenchmarkMaterialTexDistant(b *testing.B) {
	benchRenderCfg(b, RenderConfig{Lighting: true, Texture: true}, 9, 2048)
}

func BenchmarkMaterialDense(b *testing.B) {
	benchRenderMesh(b, RenderConfig{Lighting: true}, buildSphere(500, 500), 2.6, 1024)
}
func BenchmarkMaterialTexDense(b *testing.B) {
	benchRenderMesh(b, RenderConfig{Lighting: true, Texture: true}, buildSphere(500, 500), 2.6, 1024)
}
