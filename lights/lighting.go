package lights

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/ungerik/go3d/vec3"
)

type DirectLight struct {
	Color       vec3.T
	Direction   vec3.T
	CastShadows bool
}

type AmbientLight struct {
	Color vec3.T
}

type PointLight struct {
	Color     vec3.T
	Position  vec3.T
	Intensity float32

	Constant  float32 // in general 1.0
	Linear    float32 // in general 0.09
	Quadratic float32 // in general 0.032
}

type SpotLight struct {
	Color     vec3.T
	Position  vec3.T
	Direction vec3.T

	Intensity float32
	Constant  float32
	Linear    float32
	Quadratic float32

	CosCutOff   float32
	OuterCos    float32
	CastShadows bool
}

func NewSpotLight(
	color, position, direction vec3.T,
	intensity, constant, linear, quadratic float32,
	innerAngleDeg, outerAngleDeg float32,
	castShadows bool,
) *SpotLight {
	innerRad := float32(float64(innerAngleDeg) * math.Pi / 180.0)
	outerRad := float32(float64(outerAngleDeg) * math.Pi / 180.0)

	cosInner := float32(math.Cos(float64(innerRad)))
	cosOuter := float32(math.Cos(float64(outerRad)))

	return &SpotLight{
		Color:       color,
		Position:    position,
		Direction:   direction,
		Intensity:   intensity,
		Constant:    constant,
		Linear:      linear,
		Quadratic:   quadratic,
		CosCutOff:   cosInner,
		OuterCos:    cosOuter,
		CastShadows: castShadows,
	}
}

type LightingConfig struct {
	Ambient     AmbientLight
	Directional DirectLight

	PointLights map[int]*PointLight
	SpotLights  map[int]*SpotLight
}

func GetDirectionalLightSpaceMatrix(lightPos, lightDir vec3.T) mgl32.Mat4 {
	pos := mgl32.Vec3{lightPos[0], lightPos[1], lightPos[2]}
	dir := mgl32.Vec3{lightDir[0], lightDir[1], lightDir[2]}
	target := pos.Add(dir)

	up := mgl32.Vec3{0, 1, 0}
	if math.Abs(float64(dir.Y())) > 0.99 {
		up = mgl32.Vec3{0, 0, 1}
	}

	lightView := mgl32.LookAtV(pos, target, up)
	lightProj := mgl32.Ortho(-500, 500, -500, 500, 0.1, 2000.0)

	return lightProj.Mul4(lightView)
}

func GetSpotLightSpaceMatrix(lightPos, lightDir vec3.T, fov, aspect, near, far float32) mgl32.Mat4 {
	pos := mgl32.Vec3{lightPos[0], lightPos[1], lightPos[2]}
	dir := mgl32.Vec3{lightDir[0], lightDir[1], lightDir[2]}
	target := pos.Add(dir)

	up := mgl32.Vec3{0, 1, 0}
	if math.Abs(float64(dir.Y())) > 0.99 {
		up = mgl32.Vec3{0, 0, 1}
	}

	lightView := mgl32.LookAtV(pos, target, up)
	// Перспективная проекция для прожектора
	lightProj := mgl32.Perspective(fov, aspect, near, far)

	return lightProj.Mul4(lightView)
}
