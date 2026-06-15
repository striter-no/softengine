package api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/striter-no/softengine/api/shaders"
	"github.com/striter-no/softengine/entity"
	"github.com/striter-no/softengine/lights"
	"github.com/striter-no/softengine/sounds"
	"github.com/striter-no/softgo/api"
	sapi "github.com/striter-no/softgo/api"
	"github.com/striter-no/softgo/api/keyboard"
	"github.com/striter-no/softgo/api/mouse"
	"github.com/striter-no/softgo/render"
	"github.com/striter-no/stg/graphics"
	"github.com/ungerik/go3d/vec2"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

type Engine struct {
	ctx      context.Context
	Mouse    mouse.WindowMouse
	Keyboard keyboard.WindowKeyboard

	Camera  *sapi.Camera
	RScreen *sapi.RenderScreen
	TSystem TimeSystem

	FragShader **sapi.FragmentShader
	VertShader **sapi.VertexShader

	Objects       map[int]*entity.Object3D
	incrementalID int

	lastUpdate  time.Time
	LightConfig lights.LightingConfig

	MainFBO     *render.Framebuffer
	SoundSystem *sounds.SoundSystem

	ShadowFBO        *render.Framebuffer // for Directional Light
	SpotShadowFBO    *render.Framebuffer // for Spot Light
	ShadowVertShader *sapi.VertexShader
	ShadowFragShader *sapi.FragmentShader
}

func NewEngine(ctx context.Context) (*Engine, error) {

	shadowRes := 512

	shadowVert := api.NewVertexShader(func(pos, norm *vec3.T, color *vec4.T, uv *vec2.T, shader *sapi.VertexShader) render.VertexOut {
		scAny, _ := shader.GetUniform("ctx")
		sc := scAny.(*shaders.ShaderContext)
		v4 := mgl32.Vec4{pos[0], pos[1], pos[2], 1.0}
		posN := sc.MVP.Mul4x1(v4)
		return render.VertexOut{Pos: posN}
	})
	shadowFrag := api.NewFragShader(func(u, v float32, c vec4.T, n vec3.T, fp vec4.T, ctx *sapi.FragmentShader) vec4.T {
		return vec4.T{0, 0, 0, 1.0}
	})

	Mouse, err := mouse.NewWindowMouse()
	if err != nil {
		return nil, err
	}

	Keyboard, err := keyboard.NewWindowKeyboard()
	if err != nil {
		return nil, err
	}

	s, err := sapi.NewRenderScreen(ctx)
	if err != nil {
		return nil, err
	}

	SoundSystem, err := sounds.NewSoundSystem(vec3.T{0, 0, 0})
	if err != nil {
		return nil, err
	}

	Mouse.LockCursor()
	Mouse.HideMouse()

	s.SSAAFactor = 1
	s.Init()

	return &Engine{
		ctx:        ctx,
		Mouse:      Mouse,
		Keyboard:   Keyboard,
		Objects:    make(map[int]*entity.Object3D),
		RScreen:    s,
		FragShader: &s.FragShader,
		VertShader: &s.VertexShader,
		LightConfig: lights.LightingConfig{
			PointLights: make(map[int]*lights.PointLight, 0),
			SpotLights:  make(map[int]*lights.SpotLight, 0),
		},
		MainFBO:     render.NewFramebuffer(s.Screen.Width*s.SSAAFactor, s.Screen.Height*s.SSAAFactor, false),
		SoundSystem: SoundSystem,

		ShadowFBO:        render.NewFramebuffer(shadowRes, shadowRes, true),
		SpotShadowFBO:    render.NewFramebuffer(shadowRes, shadowRes, true),
		ShadowVertShader: shadowVert,
		ShadowFragShader: shadowFrag,
	}, nil
}

func (e *Engine) InitCamera(position vec3.T, sensitivity, speed, near, far, fov float32) {
	e.Camera = sapi.NewCamera(position, sensitivity, speed, e.Mouse, e.Keyboard, near, far, fov)
}

func (e *Engine) AddObject(obj *entity.Object3D) (int, error) {
	if obj == nil {
		return 0, errors.New("Cannot add nil object")
	}

	id := e.incrementalID
	e.Objects[id] = obj

	e.incrementalID++
	return id, nil
}

func (e *Engine) GetObject(id int) (*entity.Object3D, error) {
	if obj, ok := e.Objects[id]; ok {
		return obj, nil
	}

	return nil, errors.New("failed to get object")
}

func (e *Engine) RemoveObject(id int) {
	delete(e.Objects, id)
}

func (e *Engine) IsRunning() bool {
	return e.RScreen.IsOpen()
}

func (e *Engine) UpdateHID() {
	e.lastUpdate = time.Now()

	e.Mouse.PollEvents()
	e.Keyboard.PollEvents()

	if e.RScreen.Screen.Height == 0 {
		return
	}

	aspect := float32(e.RScreen.Screen.Width) / (float32(e.RScreen.Screen.Height))
	e.Camera.UpdateOnHID(aspect)
}

func (e *Engine) UpdateShaders(
	fragShader *sapi.FragmentShader,
	vertShader *sapi.VertexShader,
) {
	e.RScreen.FragShader = fragShader
	e.RScreen.VertexShader = vertShader
}

func (e *Engine) NewSpotLight(conf *lights.SpotLight) int {
	e.LightConfig.SpotLights[e.incrementalID] = conf
	e.incrementalID++

	return e.incrementalID - 1
}

func (e *Engine) NewPointLight(conf *lights.PointLight) int {
	e.LightConfig.PointLights[e.incrementalID] = conf
	e.incrementalID++

	return e.incrementalID - 1
}

func (e *Engine) RemovePointLigth(id int) {
	delete(e.LightConfig.PointLights, id)
}

func (e *Engine) RemoveSpotLigth(id int) {
	delete(e.LightConfig.SpotLights, id)
}

func (e *Engine) DrawObjects() error {
	e.RScreen.Clear()
	if e.MainFBO.Width != e.RScreen.Screen.Width*e.RScreen.SSAAFactor ||
		e.MainFBO.Height != e.RScreen.Screen.Height*e.RScreen.SSAAFactor {

		e.MainFBO = render.NewFramebuffer(e.RScreen.Screen.Width*e.RScreen.SSAAFactor, e.RScreen.Screen.Height*e.RScreen.SSAAFactor, false)
	}

	e.MainFBO.Clear(e.RScreen.BackColor)

	ogShaderV := *e.RScreen.VertexShader
	ogShaderF := *e.RScreen.FragShader

	var dirLightSpaceMatrix mgl32.Mat4
	var hasDirShadow bool
	var spotLightSpaceMatrix mgl32.Mat4
	var hasSpotShadow bool

	if e.LightConfig.Directional.CastShadows {
		e.ShadowFBO.Clear(vec3.T{})
		e.RScreen.VertexShader = e.ShadowVertShader
		e.RScreen.FragShader = e.ShadowFragShader

		dir := e.LightConfig.Directional.Direction
		dirPos := vec3.T{-dir[0] * 50000.0, -dir[1] * 50000.0, -dir[2] * 50000.0}

		dirLightSpaceMatrix = lights.GetDirectionalLightSpaceMatrix(dirPos, dir)
		hasDirShadow = true

		for _, obj := range e.Objects {
			if !obj.CastShadows {
				continue
			}

			model := obj.GetModelMatrix()
			ctx := &shaders.ShaderContext{
				MVP: dirLightSpaceMatrix.Mul4(model),
			}
			(*e.ShadowVertShader).SetUniform("ctx", ctx)
			e.RScreen.DrawCall(obj.GetActiveMesh(10), e.ShadowFBO)
		}
	}

	// 2. PASS: Spot Light Shadow
	for _, spot := range e.LightConfig.SpotLights {
		if spot.CastShadows {
			e.SpotShadowFBO.Clear(vec3.T{})
			e.RScreen.VertexShader = e.ShadowVertShader
			e.RScreen.FragShader = e.ShadowFragShader

			halfAngleRad := float32(math.Acos(float64(spot.OuterCos)))
			fovRadians := halfAngleRad * 2.0

			fovRadians += float32(2.0 * math.Pi / 180.0)

			spotLightSpaceMatrix = lights.GetSpotLightSpaceMatrix(spot.Position, spot.Direction, fovRadians, 1.0, 0.1, 1000.0)
			hasSpotShadow = true

			for _, obj := range e.Objects {
				if !obj.CastShadows {
					continue
				}

				model := obj.GetModelMatrix()
				ctx := &shaders.ShaderContext{
					MVP: spotLightSpaceMatrix.Mul4(model),
				}
				(*e.ShadowVertShader).SetUniform("ctx", ctx)
				e.RScreen.DrawCall(obj.GetActiveMesh(10), e.SpotShadowFBO)
			}
			break
		}
	}

	e.RScreen.VertexShader = &ogShaderV
	e.RScreen.FragShader = &ogShaderF

	type RenderNode struct {
		Obj              *entity.Object3D
		DistanceToCamera float32
		MVP              mgl32.Mat4
	}

	var renderQueue []RenderNode

	for _, obj := range e.Objects {
		model := obj.GetModelMatrix()
		mvp := e.Camera.VP.Mul4(model)

		center := mgl32.Vec4{0, 0, 0, 1}
		clipCenter := mvp.Mul4x1(center)

		maxScale := obj.Scale[0]
		if obj.Scale[1] > maxScale {
			maxScale = obj.Scale[1]
		}
		if obj.Scale[2] > maxScale {
			maxScale = obj.Scale[2]
		}

		actualRadius := obj.BaseRadius * maxScale

		if clipCenter.W() < -actualRadius {
			continue
		}

		if clipCenter.W() > 0 {
			ndcX := clipCenter.X() / clipCenter.W()
			ndcY := clipCenter.Y() / clipCenter.W()
			ndcZ := clipCenter.Z() / clipCenter.W()

			bound := 1.5 * (1.0 + (actualRadius / clipCenter.W()))
			zbound := (1.0 + (actualRadius / clipCenter.W()))

			if ndcX < -bound || ndcX > bound || ndcY < -bound || ndcY > bound || ndcZ > zbound || ndcZ < -zbound {
				continue
			}
		}

		renderQueue = append(renderQueue, RenderNode{
			Obj:              obj,
			DistanceToCamera: clipCenter.W(),
			MVP:              mvp,
		})
	}

	sort.Slice(renderQueue, func(i, j int) bool {
		return renderQueue[i].DistanceToCamera > renderQueue[j].DistanceToCamera
	})

	for _, node := range renderQueue {
		obj := node.Obj
		activeMesh := obj.GetActiveMesh(node.DistanceToCamera)

		ctx := &shaders.ShaderContext{
			MVP:     node.MVP,
			Model:   obj.GetModelMatrix(),
			ViewPos: e.Camera.Position,

			Texture: obj.Texture.Texture,
			Color:   vec4.T{obj.Texture.BaseColor[0], obj.Texture.BaseColor[1], obj.Texture.BaseColor[2], 1},

			Lights:     e.LightConfig,
			IsStraight: !obj.CanBeLit,

			HasDirShadow:        hasDirShadow,
			DirLightSpaceMatrix: dirLightSpaceMatrix,
			DirShadowDepth:      e.ShadowFBO.DepthBuffer,
			DirShadowWidth:      e.ShadowFBO.Width,
			DirShadowHeight:     e.ShadowFBO.Height,

			HasSpotShadow:        hasSpotShadow,
			SpotLightSpaceMatrix: spotLightSpaceMatrix,
			SpotShadowDepth:      e.SpotShadowFBO.DepthBuffer,
			SpotShadowWidth:      e.SpotShadowFBO.Width,
			SpotShadowHeight:     e.SpotShadowFBO.Height,
			IsSkybox:             obj.IsSkybox,
		}

		(*e.VertShader).SetUniform("ctx", ctx)
		(*e.FragShader).SetUniform("ctx", ctx)
		if err := e.RScreen.DrawCall(activeMesh, e.MainFBO); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) Blit() {

	e.RScreen.Present(e.MainFBO)

	e.RScreen.Screen.SetText(
		0, 0, fmt.Sprintf("FPS: %.1f", e.RScreen.CurrentFPS), graphics.NewFGPixel(255, 255, 255, ""),
	)

	e.RScreen.Screen.Blit()
	e.TSystem.FPS = float32(e.RScreen.CurrentFPS)
	e.TSystem.DeltaTime = float32(time.Since(e.lastUpdate).Milliseconds()) / 1000
	e.TSystem.Ticks++
}

func (e *Engine) End() {

	e.Mouse.UnlockCursor()
	e.Mouse.ShowMouse()
	e.Mouse.Close()

	e.Keyboard.Close()
	e.RScreen.End()

	e.SoundSystem.End()
}
