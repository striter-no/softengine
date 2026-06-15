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

	engine.UpdateShaders(
		shaders.NewBaseFragmentShader(),
		shaders.NewBaseVertexShader(),
	)

	engine.InitCamera(vec3.T{0, 0, 2}, 0.08, 100, 0.01, 1000, 80)
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
				chunkMesh, grassTex, true, false,
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
		monkey, onigiriTex, true, true,
	)

	if _, err = engine.AddObject(monkeyObj); err != nil {
		panic(err)
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
		skyboxMesh, skyboxTex, false, false,
	)
	skyboxObj.IsSkybox = true

	if _, err = engine.AddObject(skyboxObj); err != nil {
		panic(err)
	}

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

		// if realZoom > 3 {
		// 	engine.Camera.Near = 0.8
		// } else {

		// }

		skyboxObj.Position = engine.Camera.Position
		skyboxObj.UpdateMat()

		monkeyObj.LookAt(engine.Camera.Position, true)

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

		// grassMesh := generator.Generate(150, 150, float64(engine.TSystem.Ticks)/10)
		// grassObj.Mesh = grassMesh

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
