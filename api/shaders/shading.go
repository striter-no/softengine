package shaders

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/striter-no/softengine/lights"
	"github.com/striter-no/softgo/render"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

type ShaderContext struct {
	MVP     mgl32.Mat4
	Model   mgl32.Mat4
	Texture *render.Texture
	Color   vec4.T
	ViewPos vec3.T

	Lights     lights.LightingConfig
	IsStraight bool
	IsSkybox   bool

	// --- Directional Shadow ---
	HasDirShadow        bool
	DirLightSpaceMatrix mgl32.Mat4
	DirShadowDepth      []float32
	DirShadowWidth      int
	DirShadowHeight     int

	// --- Spot Shadow ---
	HasSpotShadow        bool
	SpotLightSpaceMatrix mgl32.Mat4
	SpotShadowDepth      []float32
	SpotShadowWidth      int
	SpotShadowHeight     int
}
