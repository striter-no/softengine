package shaders

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/striter-no/softgo/api"
	"github.com/striter-no/softgo/render"
	"github.com/ungerik/go3d/vec2"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

func vertShader(vert *vec3.T, normal *vec3.T, color *vec4.T, uv *vec2.T, s *api.VertexShader) render.VertexOut {
	ctxAny, _ := s.GetUniform("ctx")
	ctx := ctxAny.(*ShaderContext)

	v := mgl32.Vec4{vert[0], vert[1], vert[2], 1.0}
	worldPos := ctx.Model.Mul4x1(v)

	n := mgl32.Vec4{normal[0], normal[1], normal[2], 0.0}
	transformedNormal := ctx.Model.Mul4x1(n)

	clipPos := ctx.MVP.Mul4x1(v)

	if ctx.IsSkybox {
		clipPos[2] = clipPos[3] * 0.99999
	}

	return render.VertexOut{
		Pos:     clipPos,
		Normal:  vec3.T{transformedNormal[0], transformedNormal[1], transformedNormal[2]},
		UV:      *uv,
		Color:   *color,
		FragPos: vec3.T{worldPos[0], worldPos[1], worldPos[2]},
	}
}

func NewBaseVertexShader() *api.VertexShader {
	return api.NewVertexShader(vertShader)
}
