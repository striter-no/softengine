package api

import (
	"context"
	"time"

	"github.com/striter-no/softengine/api/phyapi"
	"github.com/striter-no/softengine/sounds"
	sapi "github.com/striter-no/softgo/api"
	"github.com/striter-no/softgo/api/keyboard"
	"github.com/striter-no/softgo/api/mouse"
	"github.com/ungerik/go3d/vec3"
)

type Engine struct {
	ctx      context.Context
	Mouse    mouse.WindowMouse
	Keyboard keyboard.WindowKeyboard

	Camera  *sapi.Camera
	RScreen *sapi.RenderScreen
	TSystem TimeSystem

	lastUpdate time.Time

	Render  *RenderSystem
	Lights  *LightSystem
	Physics *phyapi.PhysicsSystem
	Scene   *SceneSystem
	Sound   *sounds.SoundSystem
}

func NewEngine(shadowRes int, ctx context.Context) (*Engine, error) {
	mouseDev, err := mouse.NewWindowMouse()
	if err != nil {
		return nil, err
	}

	keyboardDev, err := keyboard.NewWindowKeyboard()
	if err != nil {
		return nil, err
	}

	screen, err := sapi.NewRenderScreen(ctx)
	if err != nil {
		return nil, err
	}

	soundSys, err := sounds.NewSoundSystem(vec3.T{0, 0, 0})
	if err != nil {
		return nil, err
	}

	mouseDev.LockCursor()
	mouseDev.HideMouse()

	screen.SSAAFactor = 1
	screen.Init()

	return &Engine{
		ctx:      ctx,
		Mouse:    mouseDev,
		Keyboard: keyboardDev,
		RScreen:  screen,
		TSystem:  TimeSystem{},

		lastUpdate: time.Now(),

		Render:  NewRenderSystem(screen, shadowRes),
		Lights:  NewLightSystem(),
		Physics: phyapi.NewPhysicsSystem(),
		Scene:   NewSceneSystem(),
		Sound:   soundSys,
	}, nil
}

func (e *Engine) InitCamera(position vec3.T, sensitivity, speed, near, far, fov float32) {
	e.Camera = sapi.NewCamera(position, sensitivity, speed, e.Mouse, e.Keyboard, near, far, fov)
}

func (e *Engine) IsRunning() bool {
	return e.RScreen.IsOpen()
}

func (e *Engine) UpdateHID(movement bool, mouse bool, keboard bool) {
	e.lastUpdate = time.Now()

	if mouse {
		e.Mouse.PollEvents()
	}
	if keboard {
		e.Keyboard.PollEvents()
	}

	if e.RScreen.Screen.Height == 0 {
		return
	}

	aspect := float32(e.RScreen.Screen.Width) / float32(e.RScreen.Screen.Height)
	e.Camera.UpdateOnHID(aspect, movement)
}

func (e *Engine) DrawObjects() error {
	return e.Render.Draw(e.Camera, e.Scene, e.Lights)
}

func (e *Engine) DrawScene() {
	e.Render.PresentScene(e.Camera, e.Scene)
}

func (e *Engine) Blit() {
	e.Render.Blit(&e.TSystem, e.lastUpdate)
}

func (e *Engine) StartPhysicsLoop(ctx context.Context, tps int) {
	e.Physics.StartLoop(ctx, tps)
}

func (e *Engine) End() {
	e.Physics.End()

	e.Mouse.UnlockCursor()
	e.Mouse.ShowMouse()
	e.Mouse.Close()

	e.Keyboard.Close()
	e.RScreen.End()

	e.Render.End()
	e.Lights.End()
	e.Scene.End()
	e.Sound.End()
}
