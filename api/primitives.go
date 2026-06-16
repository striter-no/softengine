package api

import (
	"math"

	"github.com/striter-no/softgo/render"
	"github.com/ungerik/go3d/vec2"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

type tri struct {
	V0, V1, V2    vec3.T
	N0, N1, N2    vec3.T
	UV0, UV1, UV2 vec2.T
}

func fillTri(t tri) render.TBO {
	return render.TBO{
		V0:  t.V0,
		V1:  t.V1,
		V2:  t.V2,
		N0:  t.N0,
		N1:  t.N1,
		N2:  t.N2,
		UV0: t.UV0,
		UV1: t.UV1,
		UV2: t.UV2,
		C0:  vec4.T{1, 1, 1, 1},
		C1:  vec4.T{1, 1, 1, 1},
		C2:  vec4.T{1, 1, 1, 1},
	}
}

func Cube(w, h, d float32) []render.TBO {
	hw, hh, hd := w*0.5, h*0.5, d*0.5

	v := [8]vec3.T{
		{-hw, -hh, -hd}, // 0
		{+hw, -hh, -hd}, // 1
		{+hw, +hh, -hd}, // 2
		{-hw, +hh, -hd}, // 3
		{-hw, -hh, +hd}, // 4
		{+hw, -hh, +hd}, // 5
		{+hw, +hh, +hd}, // 6
		{-hw, +hh, +hd}, // 7
	}

	type face struct {
		a, b, c, d int
		n          vec3.T
	}
	faces := [6]face{
		{0, 3, 2, 1, vec3.T{0, 0, -1}}, // back  (-Z)
		{5, 6, 7, 4, vec3.T{0, 0, +1}}, // front (+Z)
		{4, 7, 3, 0, vec3.T{-1, 0, 0}}, // left  (-X)
		{1, 2, 6, 5, vec3.T{+1, 0, 0}}, // right (+X)
		{3, 7, 6, 2, vec3.T{0, +1, 0}}, // top   (+Y)
		{4, 0, 1, 5, vec3.T{0, -1, 0}}, // bottom(-Y)
	}

	out := make([]render.TBO, 0, 12)
	for _, f := range faces {
		// UV: a=(0,0), b=(1,0), c=(1,1), d=(0,1).
		out = append(out, fillTri(tri{
			V0: v[f.a], V1: v[f.b], V2: v[f.c],
			N0: f.n, N1: f.n, N2: f.n,
			UV0: vec2.T{0, 0}, UV1: vec2.T{1, 0}, UV2: vec2.T{1, 1},
		}))
		out = append(out, fillTri(tri{
			V0: v[f.a], V1: v[f.c], V2: v[f.d],
			N0: f.n, N1: f.n, N2: f.n,
			UV0: vec2.T{0, 0}, UV1: vec2.T{1, 1}, UV2: vec2.T{0, 1},
		}))
	}
	return out
}

func Sphere(radius float32, segments, rings int) []render.TBO {
	if segments < 4 {
		segments = 4
	}
	if rings < 3 {
		rings = 3
	}

	rows := rings + 2
	cols := segments + 1

	verts := make([][]vec3.T, rows)
	norms := make([][][]vec3.T, 0)
	uvs := make([][][]vec2.T, 0)
	_ = norms
	_ = uvs

	for r := range rows {
		verts[r] = make([]vec3.T, cols)
		phi := math.Pi * float64(r) / float64(rows-1)
		sinPhi := float32(math.Sin(phi))
		cosPhi := float32(math.Cos(phi))
		for c := range cols {
			theta := 2 * math.Pi * float64(c) / float64(segments)
			sinT := float32(math.Sin(theta))
			cosT := float32(math.Cos(theta))
			verts[r][c] = vec3.T{
				radius * sinPhi * cosT,
				radius * cosPhi,
				radius * sinPhi * sinT,
			}
		}
	}

	uv := func(r, c int) vec2.T {
		// u = c/segments, v = r/(rows-1).
		return vec2.T{
			float32(c) / float32(segments),
			float32(r) / float64_to_float32(float64(rows-1)),
		}
	}

	out := make([]render.TBO, 0, segments*(rings+1)*2)

	for r := 0; r < rows-1; r++ {
		for c := 0; c < segments; c++ {
			v00 := verts[r][c]
			v01 := verts[r][c+1]
			v10 := verts[r+1][c]
			v11 := verts[r+1][c+1]

			n00 := v00.Normalized()
			n01 := v01.Normalized()
			n10 := v10.Normalized()
			n11 := v11.Normalized()

			uv00 := uv(r, c)
			uv01 := uv(r, c+1)
			uv10 := uv(r+1, c)
			uv11 := uv(r+1, c+1)

			out = append(out, fillTri(tri{
				V0: v00, V1: v10, V2: v11,
				N0: n00, N1: n10, N2: n11,
				UV0: uv00, UV1: uv10, UV2: uv11,
			}))
			out = append(out, fillTri(tri{
				V0: v00, V1: v11, V2: v01,
				N0: n00, N1: n11, N2: n01,
				UV0: uv00, UV1: uv11, UV2: uv01,
			}))
		}
	}
	return out
}

func Plane(w, d float32) []render.TBO {
	hw, hd := w*0.5, d*0.5
	n := vec3.T{0, 1, 0}
	return []render.TBO{
		fillTri(tri{
			V0: vec3.T{-hw, 0, -hd}, V1: vec3.T{+hw, 0, -hd}, V2: vec3.T{+hw, 0, +hd},
			N0: n, N1: n, N2: n,
			UV0: vec2.T{0, 0}, UV1: vec2.T{1, 0}, UV2: vec2.T{1, 1},
		}),
		fillTri(tri{
			V0: vec3.T{-hw, 0, -hd}, V1: vec3.T{+hw, 0, +hd}, V2: vec3.T{-hw, 0, +hd},
			N0: n, N1: n, N2: n,
			UV0: vec2.T{0, 0}, UV1: vec2.T{1, 1}, UV2: vec2.T{0, 1},
		}),
	}
}

func QuadXY(size float32) []render.TBO {
	s := size * 0.5
	n := vec3.T{0, 0, 1}
	return []render.TBO{
		fillTri(tri{
			V0: vec3.T{-s, -s, 0}, V1: vec3.T{+s, -s, 0}, V2: vec3.T{+s, +s, 0},
			N0: n, N1: n, N2: n,
			UV0: vec2.T{0, 0}, UV1: vec2.T{1, 0}, UV2: vec2.T{1, 1},
		}),
		fillTri(tri{
			V0: vec3.T{-s, -s, 0}, V1: vec3.T{+s, +s, 0}, V2: vec3.T{-s, +s, 0},
			N0: n, N1: n, N2: n,
			UV0: vec2.T{0, 0}, UV1: vec2.T{1, 1}, UV2: vec2.T{0, 1},
		}),
	}
}
func Cylinder(radius, height float32, segments int) []render.TBO {
	if segments < 4 {
		segments = 4
	}
	hh := height * 0.5

	top := make([]vec3.T, segments+1)
	bot := make([]vec3.T, segments+1)
	for i := 0; i <= segments; i++ {
		theta := 2 * math.Pi * float64(i) / float64(segments)
		x := radius * float32(math.Cos(theta))
		z := radius * float32(math.Sin(theta))
		top[i] = vec3.T{x, +hh, z}
		bot[i] = vec3.T{x, -hh, z}
	}

	out := make([]render.TBO, 0, segments*4)

	for i := 0; i < segments; i++ {
		theta := 2 * math.Pi * float64(float64(i)+0.5) / float64(segments)
		n := vec3.T{
			float32(math.Cos(theta)),
			0,
			float32(math.Sin(theta)),
		}
		u0 := float32(i) / float32(segments)
		u1 := float32(i+1) / float32(segments)

		// Quad: bot[i] → bot[i+1] → top[i+1] → top[i]
		out = append(out, fillTri(tri{
			V0: bot[i], V1: bot[i+1], V2: top[i+1],
			N0: n, N1: n, N2: n,
			UV0: vec2.T{u0, 0}, UV1: vec2.T{u1, 0}, UV2: vec2.T{u1, 1},
		}))
		out = append(out, fillTri(tri{
			V0: bot[i], V1: top[i+1], V2: top[i],
			N0: n, N1: n, N2: n,
			UV0: vec2.T{u0, 0}, UV1: vec2.T{u1, 1}, UV2: vec2.T{u0, 1},
		}))
	}

	nTop := vec3.T{0, 1, 0}
	for i := 0; i < segments; i++ {
		out = append(out, fillTri(tri{
			V0: vec3.T{0, +hh, 0}, V1: top[i+1], V2: top[i],
			N0: nTop, N1: nTop, N2: nTop,
			UV0: vec2.T{0.5, 0.5}, UV1: vec2.T{1, 1}, UV2: vec2.T{0, 1},
		}))
	}

	nBot := vec3.T{0, -1, 0}
	for i := 0; i < segments; i++ {
		out = append(out, fillTri(tri{
			V0: vec3.T{0, -hh, 0}, V1: bot[i], V2: bot[i+1],
			N0: nBot, N1: nBot, N2: nBot,
			UV0: vec2.T{0.5, 0.5}, UV1: vec2.T{0, 1}, UV2: vec2.T{1, 1},
		}))
	}

	return out
}

func Cone(radius, height float32, segments int) []render.TBO {
	if segments < 4 {
		segments = 4
	}
	hh := height * 0.5

	base := make([]vec3.T, segments+1)
	for i := 0; i <= segments; i++ {
		theta := 2 * math.Pi * float64(i) / float64(segments)
		base[i] = vec3.T{
			radius * float32(math.Cos(theta)),
			-hh,
			radius * float32(math.Sin(theta)),
		}
	}
	apex := vec3.T{0, +hh, 0}

	out := make([]render.TBO, 0, segments*2)

	for i := 0; i < segments; i++ {
		thetaMid := 2 * math.Pi * float64(float64(i)+0.5) / float64(segments)
		n := (&vec3.T{
			float32(math.Cos(thetaMid)),
			radius / height,
			float32(math.Sin(thetaMid)),
		}).Normalized()

		u0 := float32(i) / float32(segments)
		u1 := float32(i+1) / float32(segments)

		out = append(out, fillTri(tri{
			V0: base[i], V1: base[i+1], V2: apex,
			N0: n, N1: n, N2: n,
			UV0: vec2.T{u0, 0}, UV1: vec2.T{u1, 0}, UV2: vec2.T{0.5, 1},
		}))
	}

	nBot := vec3.T{0, -1, 0}
	for i := 0; i < segments; i++ {
		out = append(out, fillTri(tri{
			V0: vec3.T{0, -hh, 0}, V1: base[i], V2: base[i+1],
			N0: nBot, N1: nBot, N2: nBot,
			UV0: vec2.T{0.5, 0.5}, UV1: vec2.T{0, 1}, UV2: vec2.T{1, 1},
		}))
	}

	return out
}

func Torus(R, r float32, majorSeg, minorSeg int) []render.TBO {
	if majorSeg < 4 {
		majorSeg = 4
	}
	if minorSeg < 3 {
		minorSeg = 3
	}

	verts := make([][]vec3.T, majorSeg+1)
	for i := 0; i <= majorSeg; i++ {
		verts[i] = make([]vec3.T, minorSeg+1)
		u := 2 * math.Pi * float64(i) / float64(majorSeg)
		cu, su := math.Cos(u), math.Sin(u)
		for j := 0; j <= minorSeg; j++ {
			v := 2 * math.Pi * float64(j) / float64(minorSeg)
			cv, sv := math.Cos(v), math.Sin(v)
			verts[i][j] = vec3.T{
				float32((R + r*float32(cv)) * float32(cu)),
				float32(r * float32(sv)),
				float32((R + r*float32(cv)) * float32(su)),
			}
		}
	}

	normalAt := func(i, j int) vec3.T {
		u := 2 * math.Pi * float64(i) / float64(majorSeg)
		cu, su := math.Cos(u), math.Sin(u)
		center := vec3.T{float32(R) * float32(cu), 0, float32(R) * float32(su)}
		return (&vec3.T{
			verts[i][j][0] - center[0],
			verts[i][j][1] - center[1],
			verts[i][j][2] - center[2],
		}).Normalized()
	}

	out := make([]render.TBO, 0, majorSeg*minorSeg*2)
	for i := 0; i < majorSeg; i++ {
		for j := 0; j < minorSeg; j++ {
			v00 := verts[i][j]
			v01 := verts[i][j+1]
			v10 := verts[i+1][j]
			v11 := verts[i+1][j+1]

			n00 := normalAt(i, j)
			n01 := normalAt(i, j+1)
			n10 := normalAt(i+1, j)
			n11 := normalAt(i+1, j+1)

			uv00 := vec2.T{float32(i) / float32(majorSeg), float32(j) / float32(minorSeg)}
			uv01 := vec2.T{float32(i) / float32(majorSeg), float32(j+1) / float32(minorSeg)}
			uv10 := vec2.T{float32(i+1) / float32(majorSeg), float32(j) / float32(minorSeg)}
			uv11 := vec2.T{float32(i+1) / float32(majorSeg), float32(j+1) / float32(minorSeg)}

			out = append(out, fillTri(tri{
				V0: v00, V1: v10, V2: v11,
				N0: n00, N1: n10, N2: n11,
				UV0: uv00, UV1: uv10, UV2: uv11,
			}))
			out = append(out, fillTri(tri{
				V0: v00, V1: v11, V2: v01,
				N0: n00, N1: n11, N2: n01,
				UV0: uv00, UV1: uv11, UV2: uv01,
			}))
		}
	}
	return out
}

func float64_to_float32(v float64) float32 {
	return float32(v)
}
