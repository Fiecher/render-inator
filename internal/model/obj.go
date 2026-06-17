package model

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	m "render-inator/internal/math"
)

func ParseOBJ(data []byte) (*Mesh, error) {
	msh := &Mesh{}

	var fileNormals []m.Vec3
	faceHasNormals := true

	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	ln := 0
	for sc.Scan() {
		ln++
		text := strings.TrimSpace(sc.Text())
		if text == "" || text[0] == '#' {
			continue
		}
		fields := strings.Fields(text)

		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				return nil, fmt.Errorf("obj line %d: v needs 3 coordinates", ln)
			}
			v, err := parseVec3(fields[1:4])
			if err != nil {
				return nil, fmt.Errorf("obj line %d: %w", ln, err)
			}
			msh.Verts = append(msh.Verts, v)

		case "vt":
			if len(fields) < 2 {
				return nil, fmt.Errorf("obj line %d: vt needs at least u", ln)
			}
			u, err := parseF(fields[1])
			if err != nil {
				return nil, fmt.Errorf("obj line %d: %w", ln, err)
			}
			var vv float32
			if len(fields) >= 3 {
				if vv, err = parseF(fields[2]); err != nil {
					return nil, fmt.Errorf("obj line %d: %w", ln, err)
				}
			}
			msh.UVs = append(msh.UVs, m.Vec2{X: u, Y: vv})

		case "vn":
			if len(fields) < 4 {
				return nil, fmt.Errorf("obj line %d: vn needs 3 components", ln)
			}
			vn, err := parseVec3(fields[1:4])
			if err != nil {
				return nil, fmt.Errorf("obj line %d: %w", ln, err)
			}
			fileNormals = append(fileNormals, vn)

		case "f":
			if len(fields) < 4 {
				return nil, fmt.Errorf("obj line %d: face needs >= 3 vertices", ln)
			}
			n := len(fields) - 1
			vs := make([]int, n)
			ts := make([]int, n)
			ns := make([]int, n)
			for i := 0; i < n; i++ {
				vi, ti, ni, err := parseRef(fields[i+1], len(msh.Verts), len(msh.UVs), len(fileNormals))
				if err != nil {
					return nil, fmt.Errorf("obj line %d: %w", ln, err)
				}
				vs[i], ts[i], ns[i] = vi, ti, ni
				if ni < 0 {
					faceHasNormals = false
				}
			}

			for i := 1; i < n-1; i++ {
				msh.Tris = append(msh.Tris, Tri{
					V:  [3]int{vs[0], vs[i], vs[i+1]},
					UV: [3]int{ts[0], ts[i], ts[i+1]},
					N:  [3]int{ns[0], ns[i], ns[i+1]},
				})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("obj: read error: %w", err)
	}
	if len(msh.Verts) == 0 {
		return nil, fmt.Errorf("obj: no vertices found")
	}
	if len(msh.Tris) == 0 {
		return nil, fmt.Errorf("obj: no faces found")
	}

	msh.ComputeNormals()
	if len(fileNormals) > 0 && faceHasNormals {
		msh.Normals = make([]m.Vec3, len(fileNormals))
		for i, vn := range fileNormals {
			msh.Normals[i] = vn.Normalize()
		}
	} else {
		msh.ComputeShadingNormals()
	}
	msh.analyzeCulling()
	return msh, nil
}

func parseVec3(f []string) (m.Vec3, error) {
	x, err := parseF(f[0])
	if err != nil {
		return m.Vec3{}, err
	}
	y, err := parseF(f[1])
	if err != nil {
		return m.Vec3{}, err
	}
	z, err := parseF(f[2])
	if err != nil {
		return m.Vec3{}, err
	}
	return m.Vec3{X: x, Y: y, Z: z}, nil
}

func parseF(s string) (float32, error) {
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, fmt.Errorf("bad float %q", s)
	}
	return float32(v), nil
}

func parseRef(tok string, nv, nt, nn int) (vIdx, tIdx, nIdx int, err error) {
	parts := strings.Split(tok, "/")

	vIdx, err = resolveIndex(parts[0], nv)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bad face vertex %q: %w", tok, err)
	}

	tIdx = -1
	if len(parts) >= 2 && parts[1] != "" {
		tIdx, err = resolveIndex(parts[1], nt)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("bad face texcoord %q: %w", tok, err)
		}
	}

	nIdx = -1
	if len(parts) >= 3 && parts[2] != "" {
		nIdx, err = resolveIndex(parts[2], nn)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("bad face normal %q: %w", tok, err)
		}
	}

	return vIdx, tIdx, nIdx, nil
}

func resolveIndex(s string, count int) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("not an integer %q", s)
	}
	if i > 0 {
		i--
	} else if i < 0 {
		i += count
	} else {
		return 0, fmt.Errorf("index 0 is invalid")
	}
	if i < 0 || i >= count {
		return 0, fmt.Errorf("index out of range")
	}
	return i, nil
}
