package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/striter-no/softengine/api"
	"github.com/striter-no/softengine/api/phyapi"
	"github.com/striter-no/softengine/api/shaders"
	"github.com/striter-no/softengine/entity"
	"github.com/striter-no/softengine/lights"
	"github.com/striter-no/softgo/api/assets"
	"github.com/striter-no/softgo/api/keyboard"
	"github.com/ungerik/go3d/vec3"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC: %v", r)
		}
	}()

	engine, err := api.NewEngine(512, context.Background())
	if err != nil {
		panic(err)
	}
	defer engine.End()

	console := api.NewDebugConsole(engine.RScreen, 12)
	console.SetPos(0, 10)

	console.Enable()
	api.SetGlobalDebugConsole(console)

	const T_SIZE = 100

	// Materials
	waterMaterial := shaders.Material{
		Ambient:   vec3.T{0.81, 0.81, 0.81},
		Diffuse:   vec3.T{0.8, 0.9, 1},
		Specular:  vec3.T{0.9, 0.9, 0.9},
		Shininess: 4.0,
	}
	texturedMaterial := shaders.Material{
		Ambient:   vec3.T{0.1, 0.1, 0.1},
		Diffuse:   vec3.T{1.0, 1.0, 1.0},
		Specular:  vec3.T{0.5, 0.5, 0.5},
		Shininess: 64.0,
	}

	// --- Lights & fog (через LightSystem) ---
	engine.Lights.Fog.Color = vec3.T{0.8, 0.8, 0.8}
	engine.Lights.Fog.Density = 0

	engine.RScreen.BackColor = vec3.T{0.8, 0.8, 1}
	engine.Lights.Config.Ambient = lights.AmbientLight{Color: vec3.T{0.1, 0.1, 0.1}}
	engine.Lights.Config.Directional = lights.DirectLight{
		Color:       vec3.T{1, 1, 1},
		Direction:   vec3.T{-0.2, -0.7, -.5},
		CastShadows: true,
	}

	pointlight := &lights.PointLight{
		Color:     vec3.T{1., .0, 1.},
		Position:  vec3.T{110, 263, -70},
		Intensity: 50,
		Constant:  2,
		Linear:    0.09,
		Quadratic: 0.032,
	}
	engine.Lights.NewPointLight(pointlight)

	engine.Render.UpdateShaders(
		shaders.NewBaseFragmentShader(),
		shaders.NewBaseVertexShader(),
	)

	engine.InitCamera(vec3.T{0, 300, 200}, 0.08, 100, 0.1, 1000, 90)
	engine.Camera.Locked = true

	// --- Sound ---
	windID := engine.Sound.AddSpeaker("./assets/sounds/wind.mp3", 1, 1)
	if windID == -1 {
		log.Fatal("Failed to load sound")
	}
	shotID := engine.Sound.AddSpeaker("./assets/sounds/gunshot.mp3", 0, 16)
	if shotID == -1 {
		log.Fatal("Failed to load sound")
	}
	engine.Sound.SetVolume(shotID, 3)

	stepID := engine.Sound.AddSpeaker("./assets/sounds/grass-footsteps.mp3", 0, 2)
	if stepID == -1 {
		log.Fatal("Failed to load sound")
	}
	engine.Sound.PlayID(windID)

	// --- Terrain ---
	grassTex, err := entity.NewModelImageTexture("./assets/textures/grass.jpg")
	grassTex.Texture.GenerateMipmaps(6, 50)
	if err != nil {
		panic(err)
	}

	cellSize := float32(20.0)
	chunkSize := 5
	generator := entity.NewTerrainGenerator(200.0, 0.1, cellSize)

	for cx := -(T_SIZE / 2); cx <= (T_SIZE / 2); cx++ {
		for cz := -(T_SIZE / 2); cz <= (T_SIZE / 2); cz++ {
			chunkMesh := generator.GenerateChunk(cx, cz, chunkSize, chunkSize, 0.0)

			posX := float32(cx*chunkSize) * cellSize
			posZ := float32(cz*chunkSize) * cellSize

			chunkObj := entity.NewObject3D(
				vec3.T{posX, 0, posZ},
				vec3.T{0, 0, 0},
				vec3.T{1, 1, 1},
				true, false,
			)
			chunkObj.Compose(chunkMesh, grassTex, waterMaterial)
			chunkObj.AddPartLOD(0, api.GenerateLOD(chunkMesh, 0.2), 60)
			chunkObj.AddPartLOD(0, api.GenerateLOD(chunkMesh, 0.5), 30)

			if _, err := engine.Scene.Add(chunkObj); err != nil {
				panic(err)
			}
		}
	}

	engine.Physics.SetTerrainHeightFunc(func(x, z float32) float32 {
		return generator.GetHeightAt(x, z, 0.0)
	})

	// --- Player ---
	// playerMeshes, _ := assets.LoadOBJ("./assets/meshes/player.obj")
	playerMesh := api.Cube(3, 7, 3)
	playerTex := entity.NewModelColorTexture(0, 0, 0, 255)
	playerObj := entity.NewObject3D(
		vec3.T{0, 60, 0},
		vec3.T{0, 0, 0},
		vec3.T{5, 5, 5},
		true, true,
	)
	// playerObj.Compose(playerMeshes["Cube"], playerTex, texturedMaterial)
	playerObj.Compose(playerMesh, playerTex, texturedMaterial)

	playerID, _ := engine.Scene.Add(playerObj)

	linkCompoundBoxes(engine, playerID, playerObj, 2, 70.0)

	// --- Skyscraper ---
	scrapers, _ := assets.LoadOBJ("./assets/meshes/skyscraper.obj")
	scTex, _ := entity.NewModelImageTexture("./assets/textures/city_diffuse.png")
	scraperObj := entity.NewObject3D(
		vec3.T{500, 0, 1000},
		vec3.T{0, 0, 0},
		vec3.T{15, 15, 15},
		true, true,
	)
	for name, objs := range scrapers {
		if !strings.HasPrefix(name, "Cube.001_Background_Night_Buildings_0") {
			continue
		}
		scraperObj.Compose(objs, scTex, texturedMaterial)
	}

	skyId, err := engine.Scene.Add(scraperObj)
	if err != nil {
		panic(err)
	}

	linkCompoundBoxes(engine, skyId, scraperObj, 30, 0.0)

	// -- Eleavator --
	elevatorMesh := api.Cube(3, 7, 3)
	elevatorTex := entity.NewModelColorTexture(0, 0, 0, 255)
	elevatorObj := entity.NewObject3D(
		vec3.T{831, 120, 1022},
		vec3.T{0, 0, 0},
		vec3.T{50, 5, 50},
		true, true,
	)
	// elevatorObj.Compose(elevatorMeshes["Cube"], elevatorTex, texturedMaterial)
	elevatorObj.Compose(elevatorMesh, elevatorTex, texturedMaterial)

	elevatorID, _ := engine.Scene.Add(elevatorObj)

	linkCompoundBoxes(engine, elevatorID, elevatorObj, 1, 0.0)

	// --- Trees ---
	treeObjects, _ := assets.LoadOBJ("./assets/meshes/tree.obj")
	treeTex, _ := entity.NewModelImageTexture("./assets/textures/wood.jpg")
	leafTex, _ := entity.NewModelImageTexture("./assets/textures/branches_2.png")

	treeTex.Texture.GenerateMipmaps(6, 50)
	leafTex.Texture.GenerateMipmaps(6, 200)

	treeObj := entity.NewObject3D(
		vec3.T{0, 50, 0},
		vec3.T{0, 0, 0},
		vec3.T{15, 30, 15},
		true, true,
	)
	treeObj.Compose(treeObjects["tree"], treeTex, texturedMaterial)
	treeObj.Compose(treeObjects["leaves"], leafTex, texturedMaterial)

	treeObj.AddPartLOD(0, api.GenerateLOD(treeObjects["tree"], 0.5), 70)
	treeObj.AddPartLOD(0, api.GenerateLOD(treeObjects["tree"], 0.7), 90)
	treeObj.AddPartLOD(1, api.GenerateLOD(treeObjects["leaves"], 0.5), 30)
	treeObj.AddPartLOD(1, api.GenerateLOD(treeObjects["leaves"], 0.7), 50)

	for range 40 {
		clone := treeObj.Clone()
		clone.Position = vec3.T{
			rand.Float32()*1000 - 500,
			50,
			rand.Float32()*1000 - 500,
		}
		id, err := engine.Scene.Add(clone)
		if err != nil {
			panic(err)
		}

		linkCompoundBoxes(engine, id, clone, 15, 0.0)
	}

	// --- Physics Debug System ---
	dbg, _, err := api.NewPhysicsDebugSystem(engine.Physics, engine.Scene)
	if err != nil {
		panic(err)
	}
	style := api.DefaultPhysicsDebugStyle()
	style.ShowContactNormals = true
	dbg.SetStyle(style)
	dbg.Disable()

	// --- Skybox ---
	skyboxTex := entity.NewModelColorTexture(200, 200, 220, 255)
	skyboxMesh, err := assets.LoadOBJ("./assets/meshes/skybox.obj")
	skyboxObj := entity.NewObject3D(
		vec3.T{0, 0, 0},
		vec3.T{0, 0, 0},
		vec3.T{1, 1, 1},
		false, false,
	)
	skyboxObj.IsSkybox = true
	skyboxObj.Compose(skyboxMesh["default"], skyboxTex, texturedMaterial)

	// --- Run ---
	engine.StartPhysicsLoop(context.Background(), 60)

	var f3WasPressed, pWasPressed, cursorLocked, altWasPressed bool
	cursorLocked = true

	zoom := 1.
	engine.RScreen.SSAAFactor = 1
	for engine.IsRunning() {
		f3Now := engine.Keyboard.IsKeyPressed(keyboard.KeyF3)
		if f3Now && !f3WasPressed {
			if dbg.Enabled() {
				dbg.Disable()
			} else {
				dbg.Enable()
			}
		}
		f3WasPressed = f3Now

		if engine.Keyboard.IsKeyPressed(keyboard.KeyAlt) && !altWasPressed {
			cursorLocked = !cursorLocked
			if cursorLocked {
				engine.Mouse.LockCursor()
				engine.Mouse.HideMouse()
				api.DebugLog("cursor: locked+hidden")
			} else {
				engine.Mouse.UnlockCursor()
				engine.Mouse.ShowMouse()
				api.DebugLog("cursor: unlocked+visible")
			}
		}
		altWasPressed = engine.Keyboard.IsKeyPressed(keyboard.KeyAlt)

		if engine.Keyboard.IsKeyPressed(keyboard.KeyP) && !pWasPressed {
			engine.Physics.SetPaused(!engine.Physics.IsPaused())
			api.DebugLog("physics paused=%v", engine.Physics.IsPaused())
		}
		pWasPressed = engine.Keyboard.IsKeyPressed(keyboard.KeyP)

		if engine.Keyboard.IsKeyPressed(keyboard.KeyCtrl) && engine.TSystem.Ticks%20 == 0 {
			engine.Sound.PlayID(stepID)
		}
		if engine.Keyboard.IsKeyPressed(keyboard.KeyEsc) {
			break
		}

		// Zoom (Sprint/zoom)
		realZoom := zoom
		if engine.Keyboard.IsKeyPressed(keyboard.KeyShift) {
			if zoom < 9 {
				zoom *= 1.1
				realZoom = zoom
			} else {
				zoom *= 1.001
				realZoom = 9
			}
		} else {
			zoom *= 0.8
			if zoom <= 1.1 {
				zoom = 1
			}
			realZoom = zoom
		}

		// Camera-relative movement basis.
		yawRad := float64(mgl32.DegToRad(engine.Camera.Rotation[1]))
		fwdX := float32(math.Sin(yawRad))
		fwdZ := float32(-math.Cos(yawRad))
		rightX := float32(math.Cos(yawRad))
		rightZ := float32(math.Sin(yawRad))

		stepped := false
		dir := vec3.T{0, 0, 0}
		if engine.Keyboard.IsKeyPressed(keyboard.KeyW) {
			if !engine.Physics.IsPaused() && engine.TSystem.Ticks%20 == 0 {
				engine.Sound.PlayID(stepID)
			}
			dir[0] += fwdX
			dir[2] += fwdZ

			stepped = true
		}
		if engine.Keyboard.IsKeyPressed(keyboard.KeyS) {
			if !engine.Physics.IsPaused() && engine.TSystem.Ticks%30 == 0 {
				engine.Sound.PlayID(stepID)
			}
			dir[0] -= fwdX
			dir[2] -= fwdZ

			stepped = true
		}
		if engine.Keyboard.IsKeyPressed(keyboard.KeyA) {
			if !engine.Physics.IsPaused() && engine.TSystem.Ticks%30 == 0 {
				engine.Sound.ChangeIDPosition(stepID, engine.Camera.Position)
				engine.Sound.PlayID(stepID)
			}
			dir[0] -= rightX
			dir[2] -= rightZ

			stepped = true
		}
		if engine.Keyboard.IsKeyPressed(keyboard.KeyD) {
			if !engine.Physics.IsPaused() && engine.TSystem.Ticks%30 == 0 {
				engine.Sound.PlayID(stepID)
			}
			dir[0] += rightX
			dir[2] += rightZ

			stepped = true
		}
		if engine.Keyboard.IsKeyPressed(keyboard.KeySpace) {
			dir[1] += 200
		}

		if !stepped {
			engine.Sound.StopID(stepID)
		}

		// Normalize horizontal.
		length := float32(math.Sqrt(float64(dir[0]*dir[0] + dir[2]*dir[2])))
		if length > 0 {
			dir[0] /= length
			dir[2] /= length
		}

		// Apply movement through physics.
		if !engine.Physics.IsPaused() {
			MovePlayer(engine, playerID, dir, 200.0)
		}

		phase := float64(engine.TSystem.Ticks%600) / 600.0 * 2 * math.Pi
		y := 120 + (1.0-float32(math.Cos(phase)))*0.5*(1400-120)

		engine.Physics.MutateNode(elevatorID, func(n *phyapi.PhysicsNode) bool {
			n.Position[1] = y
			return true
		})

		if !engine.Physics.IsPaused() {
			pPos := playerObj.Position

			engine.Camera.Position = vec3.T{
				pPos[0],
				pPos[1] + 30.0,
				pPos[2],
			}
		}
		engine.Sound.ChangeIDPosition(stepID, engine.Camera.Position)

		engine.UpdateHID(engine.Physics.Paused, cursorLocked, true)

		engine.Sound.UpdateListener(engine.Camera.Position)
		engine.Sound.ChangeIDPosition(windID, engine.Camera.Position)

		engine.Camera.FOV = float32(90. / realZoom)

		skyboxObj.Position = engine.Camera.Position
		skyboxObj.UpdateMat()

		pointlight.Position = engine.Camera.Position
		engine.Camera.Speed = 400 * engine.TSystem.DeltaTime

		if engine.TSystem.Ticks%50 == 0 {
			engine.Sound.ChangeIDPosition(shotID, vec3.T{
				rand.Float32()*300 - 150,
				50,
				rand.Float32()*300 - 150,
			})
			engine.Sound.PlayID(shotID)
		}

		if engine.TSystem.Ticks%5 == 0 {
			dbg.Rebuild()
		}

		if err := engine.DrawObjects(); err != nil {
			panic(err)
		}
		engine.DrawScene()

		console.SetCounters([]string{
			fmt.Sprintf("FPS: %.1f", engine.TSystem.FPS),
			fmt.Sprintf("Tris: %d  Objects: %d", engine.Render.TriCount, engine.Scene.Count()),
			fmt.Sprintf("Player: %v", playerObj.Position),
			fmt.Sprintf("Camera: %v", engine.Camera.Position),
		})
		console.Tick()
		console.Render()

		engine.Blit()
	}
}

// Helpers

func linkCompoundBoxes(engine *api.Engine, objID int, obj *entity.Object3D, nCubes int, mass float32) {
	offsets, halfSizes := obj.CalculateDecomposedBoxes(nCubes)

	cc := phyapi.NewCompoundCollider(len(offsets))
	for i := range offsets {
		halfWorld := vec3.T{
			halfSizes[i][0] * obj.Scale[0],
			halfSizes[i][1] * obj.Scale[1],
			halfSizes[i][2] * obj.Scale[2],
		}

		offWorld := vec3.T{
			offsets[i][0] * obj.Scale[0],
			offsets[i][1] * obj.Scale[1],
			offsets[i][2] * obj.Scale[2],
		}
		cc.Add(offWorld, phyapi.AABBHalf(halfWorld))
	}

	var node *phyapi.PhysicsNode
	if mass <= 0 {
		node = phyapi.NewStaticNode(objID, obj.Position)
	} else {
		node = phyapi.NewDynamicNode(objID, obj.Position, mass)
	}

	node.SetCompound(cc)
	node.UserData = obj
	engine.Physics.SetObjectSink(api.SyncToObject3D)
	engine.Physics.AddNode(node)
}

func MovePlayer(engine *api.Engine, playerID int, dir vec3.T, speed float32) {
	engine.Physics.MutateNode(playerID, func(node *phyapi.PhysicsNode) bool {
		currentVel := node.Velocity
		targetVel := vec3.T{dir[0] * speed, 0, dir[2] * speed}

		for _, norm := range node.ContactNormals {
			dot := targetVel[0]*norm[0] + targetVel[1]*norm[1] + targetVel[2]*norm[2]
			if dot < 0 {
				targetVel[0] -= dot * norm[0]
				targetVel[1] -= dot * norm[1]
				targetVel[2] -= dot * norm[2]
			}
		}

		accelRate := float32(engine.TSystem.DeltaTime * 20.0)
		if accelRate > 1.0 {
			accelRate = 1.0
		}

		newX := currentVel[0] + (targetVel[0]-currentVel[0])*accelRate
		newZ := currentVel[2] + (targetVel[2]-currentVel[2])*accelRate

		newY := currentVel[1]
		if dir[1] > 0 && node.IsGrounded {
			newY = dir[1]
		}

		node.Velocity = vec3.T{newX, newY, newZ}
		return true
	})
}
