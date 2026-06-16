package main

import (
	"context"
	"log"
	"math/rand"

	"github.com/striter-no/softengine/api"
	"github.com/striter-no/softengine/api/shaders"
	"github.com/striter-no/softengine/entity"
	"github.com/striter-no/softengine/lights"
	"github.com/striter-no/softgo/api/assets"
	"github.com/striter-no/softgo/api/keyboard"
	"github.com/ungerik/go3d/vec3"
)

func main() {
	engine, err := api.NewEngine(context.Background())
	if err != nil {
		panic(err)
	}

	defer engine.End()

	// Materials

	waterMaterial := shaders.Material{
		Ambient:   vec3.T{0.81, 0.81, 0.81},
		Diffuse:   vec3.T{0.8, 0.9, 1},
		Specular:  vec3.T{0.9, 0.9, 0.9},
		Shininess: 4.0,
	}

	goldMaterial := shaders.Material{
		Ambient:   vec3.T{0.24725 * 3, 0.1995 * 3, 0.0745 * 3},
		Diffuse:   vec3.T{0.75164, 0.60648, 0.22648},
		Specular:  vec3.T{0.628281, 0.555802, 0.366065},
		Shininess: 128.0,
	}

	texturedMaterial := shaders.Material{
		Ambient:   vec3.T{0.1, 0.1, 0.1},
		Diffuse:   vec3.T{1.0, 1.0, 1.0},
		Specular:  vec3.T{0.5, 0.5, 0.5},
		Shininess: 32.0,
	}

	// Init
	engine.RScreen.BackColor = vec3.T{0.8, 0.8, 1}
	engine.LightConfig.Ambient = lights.AmbientLight{Color: vec3.T{0.1, 0.1, 0.1}}
	engine.LightConfig.Directional = lights.DirectLight{
		Color:       vec3.T{1, 1, 1},
		Direction:   vec3.T{-0.2, -.5, -0.2},
		CastShadows: false,
	}
	spotlight := lights.NewSpotLight(
		vec3.T{1.0, .0, 1.0},
		vec3.T{0.0, 150.0, 0.0},
		vec3.T{0.0, -1.0, -0.2},
		4.0, 1.0, 0.009, 0.00032, // attenuation
		12.5, 17.5,
		false,
	)
	engine.NewSpotLight(spotlight)

	pointlight := &lights.PointLight{
		Color:     vec3.T{1., 1., 1.},
		Position:  vec3.T{110, 263, -70},
		Intensity: 50,
		Constant:  2,
		Linear:    0.09,
		Quadratic: 0.032,
	}
	engine.NewPointLight(pointlight)

	engine.UpdateShaders(
		shaders.NewBaseFragmentShader(),
		shaders.NewBaseVertexShader(),
	)

	engine.InitCamera(vec3.T{0, 0, 2}, 0.08, 100, 0.1, 2000, 80)
	engine.Camera.Locked = true

	// Ambient

	windID := engine.SoundSystem.AddSpeaker("./assets/sounds/wind.mp3", 1, 1)
	if windID == -1 {
		log.Fatal("Failed to load sound")
	}

	shotID := engine.SoundSystem.AddSpeaker("./assets/sounds/gunshot.mp3", 0, 16)
	if shotID == -1 {
		log.Fatal("Failed to load sound")
	}

	engine.SoundSystem.SetVolume(shotID, 3)

	engine.SoundSystem.PlayID(windID)

	// Objects

	grassTex, err := entity.NewModelImageTexture("./assets/textures/water.jpg")
	grassTex.Texture.GenerateMipmaps(6)

	if err != nil {
		panic(err)
	}

	// grassMesh, err := assets.LoadOBJ("./assets/meshes/plane.obj")
	cellSize := float32(20.0)
	chunkSize := 5
	generator := entity.NewTerrainGenerator(200.0, 0.1, cellSize)

	chunks := make([]int, 0)
	for cx := -10; cx <= 10; cx++ {
		for cz := -10; cz <= 10; cz++ {

			chunkMesh := generator.GenerateChunk(cx, cz, chunkSize, chunkSize, 0.0)

			posX := float32(cx*chunkSize) * cellSize
			posZ := float32(cz*chunkSize) * cellSize

			chunkObj := entity.NewObject3D(
				vec3.T{posX, 0, posZ},
				vec3.T{0, 0, 0},
				vec3.T{1, 1, 1},
				chunkMesh, grassTex, waterMaterial, true, false,
			)

			chunkObj.AddLOD(api.GenerateLOD(chunkMesh, 0.2), 60)
			chunkObj.AddLOD(api.GenerateLOD(chunkMesh, 0.5), 30)
			id, _ := engine.AddObject(chunkObj)

			chunks = append(chunks, id)
		}
	}

	monkey, _ := assets.LoadOBJ("./assets/meshes/suzanne.obj")
	onigiriTex, _ := entity.NewModelImageTexture("./assets/textures/onigiri.jpg")

	monkeyObj := entity.NewObject3D(
		vec3.T{0, 100, 0},
		vec3.T{0, 0, 0},
		vec3.T{30, 30, 30},
		monkey, onigiriTex, goldMaterial, true, true,
	)

	if _, err = engine.AddObject(monkeyObj); err != nil {
		panic(err)
	}

	monkeys := make([]int, 0)
	var s float32 = 300.
	for range 10 {
		clone := monkeyObj.Clone()
		clone.Position = vec3.T{
			rand.Float32()*s*2 - s,
			180 + rand.Float32()*20,
			rand.Float32()*s*2 - s,
		}
		sc := rand.Float32() + 0.5
		clone.SetScale(vec3.T{sc * 20, sc * 20, sc * 20})

		var id int
		if id, err = engine.AddObject(clone); err != nil {
			panic(err)
		}

		monkeys = append(monkeys, id)
	}

	// Skybox

	skyboxTex, err := entity.NewModelImageTexture("./assets/textures/skybox.png")
	if err != nil {
		panic(err)
	}

	skyboxMesh, err := assets.LoadOBJ("./assets/meshes/skybox.obj")

	skyboxObj := entity.NewObject3D(
		vec3.T{0, 0, 0},
		vec3.T{0, 0, 0},
		vec3.T{10, 10, 10},
		skyboxMesh, skyboxTex, texturedMaterial, false, false,
	)
	skyboxObj.IsSkybox = true

	if _, err = engine.AddObject(skyboxObj); err != nil {
		panic(err)
	}

	followId := 0
	followPath := make([]vec3.T, 0)
	n := 40

	// Run
	zoom := 1.
	engine.RScreen.SSAAFactor = 1
	for engine.IsRunning() {
		if engine.Keyboard.IsKeyPressed(keyboard.KeyEsc) {
			break
		}

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

		engine.UpdateHID()
		engine.SoundSystem.UpdateListener(engine.Camera.Position)
		engine.SoundSystem.ChangeIDPosition(windID, engine.Camera.Position)

		engine.Camera.FOV = float32(90. / realZoom)

		skyboxObj.Position = engine.Camera.Position
		skyboxObj.UpdateMat()

		monkeyObj.LookAt(engine.Camera.Position, true)

		if engine.TSystem.Ticks%10 == 0 {
			start := monkeyObj.Position
			followPath = make([]vec3.T, 0, n)
			for i := range n {
				t := float32(i) / float32(n-1)
				v := lerp(start, engine.Camera.Position, t)
				followPath = append(followPath, v)
			}
			followId = 0
		}

		if followId < len(followPath) {
			monkeyObj.Position = followPath[followId]
			followId++
		}

		pointlight.Position = engine.Camera.Position
		engine.Camera.Speed = 200 * engine.TSystem.DeltaTime

		inDump := 0
		for cx := -10; cx <= 10; cx++ {
			for cz := -10; cz <= 10; cz++ {
				chunkMesh := generator.GenerateChunk(cx, cz, chunkSize, chunkSize, float64(engine.TSystem.Ticks)/10)

				obj, err := engine.GetObject(chunks[inDump])
				if err != nil {
					panic(err)
				}
				obj.Mesh = chunkMesh

				obj.LODs = make([]entity.LOD, 0)
				obj.AddLOD(api.GenerateLOD(chunkMesh, 0.2), 60)
				obj.AddLOD(api.GenerateLOD(chunkMesh, 0.5), 30)
				inDump++
			}
		}

		for _, mid := range monkeys {
			obj, err := engine.GetObject(mid)
			if err != nil {
				panic(err)
			}
			obj.RotateEuler(vec3.T{0.1, 0, 0.3})
		}

		if engine.TSystem.Ticks%50 == 0 {
			engine.SoundSystem.ChangeIDPosition(shotID, vec3.T{
				rand.Float32()*300 - 150,
				50,
				rand.Float32()*300 - 150,
			})
			engine.SoundSystem.PlayID(shotID)
		}

		if err := engine.DrawObjects(); err != nil {
			panic(err)
		}
		engine.Blit()
	}
}

func lerp(a, b vec3.T, t float32) vec3.T {
	return vec3.T{
		a[0] + t*(b[0]-a[0]),
		a[1] + t*(b[1]-a[1]),
		a[2] + t*(b[2]-a[2]),
	}
}
