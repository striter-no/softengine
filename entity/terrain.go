package entity

import (
	"math"
	"math/rand"
	"time"

	"github.com/aquilax/go-perlin"
	"github.com/striter-no/softgo/render"
	"github.com/ungerik/go3d/vec2"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

func vecAdd(a, b vec3.T) vec3.T { return vec3.T{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func vecSub(a, b vec3.T) vec3.T { return vec3.T{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func vecCross(a, b vec3.T) vec3.T {
	return vec3.T{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}
func vecNormalize(v vec3.T) vec3.T {
	l := float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
	if l == 0 {
		return vec3.T{0, 1, 0}
	}
	return vec3.T{v[0] / l, v[1] / l, v[2] / l}
}

type TerrainGenerator struct {
	heightScale float32
	cellSize    float32
	noiseScale  float64

	p *perlin.Perlin
}

func NewTerrainGenerator(heightScale float32, noiseScale float64, cellSize float32) *TerrainGenerator {
	return &TerrainGenerator{
		p:           perlin.NewPerlinRandSource(2, 2, 2, rand.NewSource(time.Now().Unix())),
		heightScale: heightScale,
		noiseScale:  noiseScale,
		cellSize:    cellSize,
	}
}

func (g *TerrainGenerator) perlinNoise(x, y float64) float64 {
	return g.p.Noise2D(x, y)
}

func (g *TerrainGenerator) Generate(width, depth int) []render.TBO {
	return generateTerrainMesh(width, depth, g.cellSize, g.heightScale, g.noiseScale, g.perlinNoise)
}

func generateTerrainMesh(
	width, depth int,
	cellSize float32,
	heightScale float32,
	noiseScale float64,
	noiseFunc func(x, y float64) float64,
) []render.TBO {

	verts := make([][]vec3.T, width)
	colors := make([][]vec4.T, width)

	for x := range width {
		verts[x] = make([]vec3.T, depth)
		colors[x] = make([]vec4.T, depth)

		for z := 0; z < depth; z++ {
			rawNoise := noiseFunc(float64(x)*noiseScale, float64(z)*noiseScale)

			rawNoise = (rawNoise + 1.0) / 2.0

			h := float32(rawNoise)
			y := h * heightScale

			xPos := float32(x)*cellSize - float32(width)*cellSize/2.0
			zPos := float32(z)*cellSize - float32(depth)*cellSize/2.0

			verts[x][z] = vec3.T{xPos, y, zPos}
			colors[x][z] = vec4.T{1.0, 1.0, 1.0, 1.0}
		}
	}

	normals := make([][]vec3.T, width)
	for x := range width {
		normals[x] = make([]vec3.T, depth)
	}

	for x := 0; x < width-1; x++ {
		for z := 0; z < depth-1; z++ {
			v00 := verts[x][z]
			v01 := verts[x][z+1]
			v10 := verts[x+1][z]
			v11 := verts[x+1][z+1]

			n1 := vecCross(vecSub(v01, v00), vecSub(v10, v00))
			normals[x][z] = vecAdd(normals[x][z], n1)
			normals[x][z+1] = vecAdd(normals[x][z+1], n1)
			normals[x+1][z] = vecAdd(normals[x+1][z], n1)

			n2 := vecCross(vecSub(v11, v10), vecSub(v01, v10))
			normals[x+1][z] = vecAdd(normals[x+1][z], n2)
			normals[x][z+1] = vecAdd(normals[x][z+1], n2)
			normals[x+1][z+1] = vecAdd(normals[x+1][z+1], n2)
		}
	}

	for x := range width {
		for z := range depth {
			normals[x][z] = vecNormalize(normals[x][z])
		}
	}

	var mesh []render.TBO
	for x := 0; x < width-1; x++ {
		for z := 0; z < depth-1; z++ {

			uv00 := vec2.T{0, 0}
			uv01 := vec2.T{0, 1}
			uv10 := vec2.T{1, 0}
			uv11 := vec2.T{1, 1}

			mesh = append(mesh, render.TBO{
				V0: verts[x][z], V1: verts[x][z+1], V2: verts[x+1][z],
				UV0: uv00, UV1: uv01, UV2: uv10,
				N0: normals[x][z], N1: normals[x][z+1], N2: normals[x+1][z],
				C0: colors[x][z], C1: colors[x][z+1], C2: colors[x+1][z],
				OmniDir: false,
			})

			mesh = append(mesh, render.TBO{
				V0: verts[x+1][z], V1: verts[x][z+1], V2: verts[x+1][z+1],
				UV0: uv10, UV1: uv01, UV2: uv11,
				N0: normals[x+1][z], N1: normals[x][z+1], N2: normals[x+1][z+1],
				C0: colors[x+1][z], C1: colors[x][z+1], C2: colors[x+1][z+1],
				OmniDir: false,
			})
		}
	}

	return mesh
}
