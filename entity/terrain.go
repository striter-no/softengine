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

func (g *TerrainGenerator) perlinNoise(x, y, z float64) float64 {
	return g.p.Noise3D(x, y, z)
}

func (g *TerrainGenerator) GenerateChunk(chunkX, chunkZ int, cellsX, cellsZ int, zShift float64) []render.TBO {

	gridW := cellsX + 3
	gridD := cellsZ + 3

	verts := make([][]vec3.T, gridW)
	normals := make([][]vec3.T, gridW)

	for i := range gridW {
		verts[i] = make([]vec3.T, gridD)
		normals[i] = make([]vec3.T, gridD)

		lx := i - 1
		gx := chunkX*cellsX + lx

		for j := range gridD {
			lz := j - 1
			gz := chunkZ*cellsZ + lz

			rawNoise := g.perlinNoise(float64(gx)*g.noiseScale, float64(gz)*g.noiseScale, zShift*g.noiseScale)
			rawNoise = (rawNoise + 1.0) / 2.0

			y := float32(rawNoise) * g.heightScale

			xPos := float32(lx) * g.cellSize
			zPos := float32(lz) * g.cellSize

			verts[i][j] = vec3.T{xPos, y, zPos}
		}
	}

	for i := 0; i < gridW-1; i++ {
		for j := 0; j < gridD-1; j++ {
			v00 := verts[i][j]
			v01 := verts[i][j+1]
			v10 := verts[i+1][j]
			v11 := verts[i+1][j+1]

			n1 := vecCross(vecSub(v01, v00), vecSub(v10, v00))
			normals[i][j] = vecAdd(normals[i][j], n1)
			normals[i][j+1] = vecAdd(normals[i][j+1], n1)
			normals[i+1][j] = vecAdd(normals[i+1][j], n1)

			n2 := vecCross(vecSub(v11, v10), vecSub(v01, v10))
			normals[i+1][j] = vecAdd(normals[i+1][j], n2)
			normals[i][j+1] = vecAdd(normals[i][j+1], n2)
			normals[i+1][j+1] = vecAdd(normals[i+1][j+1], n2)
		}
	}

	for i := range gridW {
		for j := range gridD {
			normals[i][j] = vecNormalize(normals[i][j])
		}
	}

	var mesh []render.TBO
	for i := 1; i <= cellsX; i++ {
		for j := 1; j <= cellsZ; j++ {

			v00 := verts[i][j]
			v01 := verts[i][j+1]
			v10 := verts[i+1][j]
			v11 := verts[i+1][j+1]

			n00 := normals[i][j]
			n01 := normals[i][j+1]
			n10 := normals[i+1][j]
			n11 := normals[i+1][j+1]

			uv00 := vec2.T{0, 0}
			uv01 := vec2.T{0, 1}
			uv10 := vec2.T{1, 0}
			uv11 := vec2.T{1, 1}

			c00 := vec4.T{1.0, 1.0, 1.0, 1.0}

			mesh = append(mesh, render.TBO{
				V0: v00, V1: v01, V2: v10,
				UV0: uv00, UV1: uv01, UV2: uv10,
				N0: n00, N1: n01, N2: n10,
				C0: c00, C1: c00, C2: c00,
				OmniDir: false,
			})

			mesh = append(mesh, render.TBO{
				V0: v10, V1: v01, V2: v11,
				UV0: uv10, UV1: uv01, UV2: uv11,
				N0: n10, N1: n01, N2: n11,
				C0: c00, C1: c00, C2: c00,
				OmniDir: false,
			})
		}
	}

	return mesh
}
