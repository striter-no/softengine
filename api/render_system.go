package api

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/striter-no/softengine/api/shaders"
	"github.com/striter-no/softengine/entity"
	"github.com/striter-no/softengine/lights"
	sapi "github.com/striter-no/softgo/api"
	"github.com/striter-no/softgo/render"
	"github.com/ungerik/go3d/vec2"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

type RenderSystem struct {
	mu sync.RWMutex

	screen *sapi.RenderScreen

	MainFBO          *render.Framebuffer
	ShadowFBO        *render.Framebuffer // for Directional Light
	SpotShadowFBO    *render.Framebuffer // for Spot Light
	ShadowVertShader *sapi.VertexShader
	ShadowFragShader *sapi.FragmentShader

	TriCount int
}

func NewRenderSystem(screen *sapi.RenderScreen, shadowRes int) *RenderSystem {
	shadowVert := sapi.NewVertexShader(func(pos, norm vec3.T, color vec4.T, uv vec2.T, shader *sapi.VertexShader) render.VertexOut {
		scAny, _ := shader.GetUniform("ctx")
		sc := scAny.(*shaders.ShaderContext)
		v4 := mgl32.Vec4{pos[0], pos[1], pos[2], 1.0}
		posN := sc.MVP.Mul4x1(v4)
		return render.VertexOut{Pos: posN}
	})
	shadowFrag := sapi.NewFragShader(func(u, v float32, c vec4.T, n vec3.T, fp vec4.T, ctx *sapi.FragmentShader) vec4.T {
		return vec4.T{0, 0, 0, 1.0}
	})

	return &RenderSystem{
		screen:           screen,
		MainFBO:          render.NewFramebuffer(screen.Screen.Width*screen.SSAAFactor, screen.Screen.Height*screen.SSAAFactor, false),
		ShadowFBO:        render.NewFramebuffer(shadowRes, shadowRes, true),
		SpotShadowFBO:    render.NewFramebuffer(shadowRes, shadowRes, true),
		ShadowVertShader: shadowVert,
		ShadowFragShader: shadowFrag,
	}
}

func (rs *RenderSystem) UpdateShaders(frag *sapi.FragmentShader, vert *sapi.VertexShader) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.screen.FragShader = frag
	rs.screen.VertexShader = vert
}

type renderNode struct {
	Obj              *entity.Object3D
	DistanceToCamera float32
	MVP              mgl32.Mat4
	ModelMat         mgl32.Mat4
}

func (rs *RenderSystem) Draw(
	camera *sapi.Camera,
	scene *SceneSystem,
	ls *LightSystem,
) error {
	screen := rs.screen

	screen.Clear()

	if rs.MainFBO.Width != screen.Screen.Width*screen.SSAAFactor ||
		rs.MainFBO.Height != screen.Screen.Height*screen.SSAAFactor {
		rs.MainFBO = render.NewFramebuffer(screen.Screen.Width*screen.SSAAFactor, screen.Screen.Height*screen.SSAAFactor, false)
	}
	rs.MainFBO.Clear(screen.BackColor)

	ogShaderV := *screen.VertexShader
	ogShaderF := *screen.FragShader

	var dirLightSpaceMatrix mgl32.Mat4
	var hasDirShadow bool
	var spotLightSpaceMatrix mgl32.Mat4
	var hasSpotShadow bool

	queue := rs.buildRenderQueue(camera, scene)

	// 2. Shadow pass: Directional.
	if ls.Config.Directional.CastShadows {
		rs.drawDirectionalShadow(queue, ls, camera, &dirLightSpaceMatrix, &hasDirShadow)
	}

	// 3. Shadow pass: Spot
	for _, spot := range ls.Config.SpotLights {
		if spot.CastShadows {
			rs.drawSpotShadow(queue, spot, &spotLightSpaceMatrix, &hasSpotShadow)
			break
		}
	}

	// 4. Main pass.
	screen.VertexShader = &ogShaderV
	screen.FragShader = &ogShaderF

	rs.TriCount = 0
	for _, node := range queue {
		obj := node.Obj
		for i := range len(obj.Parts) {
			activeMesh := obj.GetActiveMesh(i, node.DistanceToCamera)
			rs.TriCount += len(activeMesh)

			part := &obj.Parts[i]
			ctx := &shaders.ShaderContext{
				MVP:     node.MVP,
				Model:   node.ModelMat,
				ViewPos: camera.Position,

				Texture: part.Texture.Texture,
				Color:   vec4.T{part.Texture.BaseColor[0], part.Texture.BaseColor[1], part.Texture.BaseColor[2], part.Texture.BaseColor[3]},

				Lights:     ls.Config,
				IsStraight: !obj.CanBeLit,

				HasDirShadow:        hasDirShadow,
				DirLightSpaceMatrix: dirLightSpaceMatrix,
				DirShadowDepth:      rs.ShadowFBO.DepthBuffer,
				DirShadowWidth:      rs.ShadowFBO.Width,
				DirShadowHeight:     rs.ShadowFBO.Height,

				HasSpotShadow:        hasSpotShadow,
				SpotLightSpaceMatrix: spotLightSpaceMatrix,
				SpotShadowDepth:      rs.SpotShadowFBO.DepthBuffer,
				SpotShadowWidth:      rs.SpotShadowFBO.Width,
				SpotShadowHeight:     rs.SpotShadowFBO.Height,
				IsSkybox:             obj.IsSkybox,
				Fog:                  ls.Fog,

				Material: part.Material,
			}

			screen.VertexShader.SetUniform("ctx", ctx)
			screen.FragShader.SetUniform("ctx", ctx)
			if err := screen.DrawCall(activeMesh, rs.MainFBO); err != nil {
				return err
			}
		}
	}

	return nil
}

func (rs *RenderSystem) buildRenderQueue(camera *sapi.Camera, scene *SceneSystem) []renderNode {
	scene.mu.RLock()
	defer scene.mu.RUnlock()

	vp := camera.VP
	planes := extractFrustumPlanes(vp)

	var queue []renderNode
	for _, obj := range scene.objects {
		center := mgl32.Vec3{obj.Position[0], obj.Position[1], obj.Position[2]}

		maxScale := obj.Scale[0]
		if obj.Scale[1] > maxScale {
			maxScale = obj.Scale[1]
		}
		if obj.Scale[2] > maxScale {
			maxScale = obj.Scale[2]
		}
		actualRadius := obj.BaseRadius * maxScale * 1.1

		if !sphereInFrustum(planes, center, actualRadius) {
			continue
		}

		model := obj.GetModelMatrix()
		mvp := camera.VP.Mul4(model)

		clipCenter := mvp.Mul4x1(mgl32.Vec4{0, 0, 0, 1})
		dist := clipCenter.W()

		queue = append(queue, renderNode{
			Obj:              obj,
			DistanceToCamera: dist,
			ModelMat:         model,
			MVP:              mvp,
		})
	}

	sort.Slice(queue, func(i, j int) bool {
		return queue[i].DistanceToCamera > queue[j].DistanceToCamera
	})
	return queue
}

func (rs *RenderSystem) drawDirectionalShadow(
	queue []renderNode,
	ls *LightSystem,
	camera *sapi.Camera,
	outMat *mgl32.Mat4,
	outHas *bool,
) {
	screen := rs.screen

	rs.ShadowFBO.Clear(vec3.T{})

	dir := ls.Config.Directional.Direction
	dirPos := vec3.T{
		camera.Position[0] - dir[0]*1000,
		camera.Position[1] - dir[1]*1000,
		camera.Position[2] - dir[2]*1000,
	}

	shadowRange := float32(500.0)
	*outMat = lights.GetDirectionalLightSpaceMatrix(dirPos, dir, shadowRange)
	*outHas = true

	screen.VertexShader = rs.ShadowVertShader
	screen.FragShader = rs.ShadowFragShader

	for _, obj := range queue {
		if !obj.Obj.CastShadows {
			continue
		}

		dist := float32(math.Sqrt(math.Pow(float64(obj.Obj.Position[0]-camera.Position[0]), 2) +
			math.Pow(float64(obj.Obj.Position[2]-camera.Position[2]), 2)))

		if dist > shadowRange*1.5 {
			continue
		}

		model := obj.ModelMat
		ctx := &shaders.ShaderContext{
			MVP: outMat.Mul4(model),
		}
		rs.ShadowVertShader.SetUniform("ctx", ctx)
		for i := range len(obj.Obj.Parts) {
			screen.DrawCall(obj.Obj.GetActiveMesh(i, 30), rs.ShadowFBO)
		}
	}
}

func (rs *RenderSystem) drawSpotShadow(
	queue []renderNode,
	spot *lights.SpotLight,
	outMat *mgl32.Mat4,
	outHas *bool,
) {
	screen := rs.screen

	rs.SpotShadowFBO.Clear(vec3.T{})

	halfAngleRad := float32(math.Acos(float64(spot.OuterCos)))
	fovRadians := halfAngleRad * 2.0
	fovRadians += float32(2.0 * math.Pi / 180.0)

	*outMat = lights.GetSpotLightSpaceMatrix(spot.Position, spot.Direction, fovRadians, 1.0, 0.1, 1000.0)
	*outHas = true

	screen.VertexShader = rs.ShadowVertShader
	screen.FragShader = rs.ShadowFragShader

	for _, obj := range queue {
		if !obj.Obj.CastShadows {
			continue
		}

		model := obj.ModelMat
		ctx := &shaders.ShaderContext{
			MVP: outMat.Mul4(model),
		}
		rs.ShadowVertShader.SetUniform("ctx", ctx)
		for i := range len(obj.Obj.Parts) {
			screen.DrawCall(obj.Obj.GetActiveMesh(i, 30), rs.SpotShadowFBO)
		}
	}
}

func (rs *RenderSystem) PresentScene(
	camera *sapi.Camera,
	scene *SceneSystem,
) {
	screen := rs.screen
	screen.Present(rs.MainFBO)
}

func (rs *RenderSystem) Blit(
	tsystem *TimeSystem,
	lastUpdate time.Time,
) {

	screen := rs.screen

	screen.Screen.Blit()
	tsystem.FPS = float32(screen.CurrentFPS)
	tsystem.DeltaTime = float32(time.Since(lastUpdate).Milliseconds()) / 1000
	tsystem.Ticks++
}

func (rs *RenderSystem) End() {}

// Порядок: left, right, bottom, top, near, far.
type frustumPlanes [6]mgl32.Vec4

func extractFrustumPlanes(vp mgl32.Mat4) frustumPlanes {
	row0 := mgl32.Vec4{vp[0], vp[4], vp[8], vp[12]}
	row1 := mgl32.Vec4{vp[1], vp[5], vp[9], vp[13]}
	row2 := mgl32.Vec4{vp[2], vp[6], vp[10], vp[14]}
	row3 := mgl32.Vec4{vp[3], vp[7], vp[11], vp[15]}

	var p frustumPlanes

	p[0] = row3.Add(row0) // left
	p[1] = row3.Sub(row0) // right
	p[2] = row3.Add(row1) // bottom
	p[3] = row3.Sub(row1) // top
	p[4] = row3.Add(row2) // near
	p[5] = row3.Sub(row2) // far

	for i := range p {
		n := mgl32.Vec3{p[i][0], p[i][1], p[i][2]}
		len := n.Len()
		if len > 1e-6 {
			inv := 1.0 / len
			p[i][0] *= inv
			p[i][1] *= inv
			p[i][2] *= inv
			p[i][3] *= inv
		}
	}
	return p
}

func sphereInFrustum(p frustumPlanes, center mgl32.Vec3, radius float32) bool {
	for i := range p {
		d := p[i][0]*center[0] + p[i][1]*center[1] + p[i][2]*center[2] + p[i][3]
		if d < -radius {
			return false
		}
	}
	return true
}
