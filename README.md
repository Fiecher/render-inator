# render-inator

Software 3D rasterizer written in Go and compiled to WebAssembly. Renders textured, lit, wireframe and crystal-shaded meshes in the browser entirely on the CPU, with no WebGL or WebGPU.

**Live demo:** served from `docs/` via [GitHub Pages](https://fiecher.github.io/render-inator/).

## What it does

The whole graphics pipeline is implemented from scratch in Go: vertex transformation, triangle rasterization, a depth buffer, perspective-correct attribute interpolation and per-pixel shading. There are no third-party rendering libraries. The compiled `.wasm` module fills an RGBA pixel buffer each frame; JavaScript does nothing but hand camera input to Go and blit the finished buffer with `putImageData`.

### Shading modes

- **Wireframe.** Edge rendering with an animated reveal that traces the mesh on load;
- **Solid.** Flat-shaded faces;
- **Material.** Per-pixel diffuse/specular studio lighting with an optional image texture;
- **Crystal.** Translucent, two-pass iridescent shader: view-angle diffraction, colored twinkling sparkles, matcap-style environment reflection, per-channel RGB separation, thickness-based absorption and a glowing core.

### Camera Controls

- Orbit (drag), pan, wheel/pinch zoom, reset camera (button or double-click).
- Model library persisted in IndexedDB; load your own `.obj` (plus an accompanying image for the texture) by file picker or drag-and-drop; ships with a default teapot and a reset-all action.

## Build & run

Requires Go 1.23+. A `Makefile` wraps the common tasks:

```sh
make build    # copy the wasm_exec.js shim and compile Go → docs/main.wasm
make serve    # serve docs/ at http://127.0.0.1:8080
make test     # run the Go test suite
make vet      # vet host packages and the js/wasm package
make fmt      # gofmt the tree
```

Then open <http://127.0.0.1:8080>.

Building by hand:

```sh
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" docs/wasm_exec.js
GOOS=js GOARCH=wasm go build -o docs/main.wasm ./cmd/wasm
go run ./cmd/serve docs 127.0.0.1:8080
```
