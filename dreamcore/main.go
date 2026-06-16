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

	// goldMaterial := shaders.Material{
	// 	Ambient:   vec3.T{0.24725 * 3, 0.1995 * 3, 0.0745 * 3},
	// 	Diffuse:   vec3.T{0.75164, 0.60648, 0.22648},
	// 	Specular:  vec3.T{0.628281, 0.555802, 0.366065},
	// 	Shininess: 128.0,
	// }

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
		Direction:   vec3.T{-0.2, -1, -0.2},
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

	grassTex, err := entity.NewModelImageTexture("./assets/textures/grass.jpg")
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
				true, false,
			)

			chunkObj.Compose(chunkMesh, grassTex, waterMaterial)

			chunkObj.AddPartLOD(0, api.GenerateLOD(chunkMesh, 0.2), 60)
			chunkObj.AddPartLOD(0, api.GenerateLOD(chunkMesh, 0.5), 30)
			id, _ := engine.AddObject(chunkObj)

			chunks = append(chunks, id)
		}
	}

	// monkey, _ := assets.LoadOBJ("./assets/meshes/suzanne.obj")
	// onigiriTex, _ := entity.NewModelImageTexture("./assets/textures/onigiri.jpg")

	// monkeyObj := entity.NewObject3D(
	// 	vec3.T{0, 100, 0},
	// 	vec3.T{0, 0, 0},
	// 	vec3.T{30, 30, 30},
	// 	monkey, onigiriTex, goldMaterial, true, true,
	// )

	// if _, err = engine.AddObject(monkeyObj); err != nil {
	// 	panic(err)
	// }

	treeObjects, _ := assets.LoadOBJ("./assets/meshes/tree.obj")
	treeTex, _ := entity.NewModelImageTexture("./assets/textures/wood.jpg")
	leafTex, _ := entity.NewModelImageTexture("./assets/textures/branches_2.png")

	treeObj := entity.NewObject3D(
		vec3.T{0, 50, 0},
		vec3.T{0, 0, 0},
		vec3.T{15, 15, 15},
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
			rand.Float32()*2000 - 1000,
			50,
			rand.Float32()*2000 - 1000,
		}

		if _, err = engine.AddObject(clone); err != nil {
			panic(err)
		}
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
		false, false,
	)
	skyboxObj.IsSkybox = true
	skyboxObj.Compose(skyboxMesh["default"], skyboxTex, texturedMaterial)

	if _, err = engine.AddObject(skyboxObj); err != nil {
		panic(err)
	}

	// followId := 0
	// followPath := make([]vec3.T, 0)
	// n := 40

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

		// monkeyObj.LookAt(engine.Camera.Position, true)

		// if engine.TSystem.Ticks%10 == 0 {
		// 	start := monkeyObj.Position
		// 	followPath = make([]vec3.T, 0, n)
		// 	for i := range n {
		// 		t := float32(i) / float32(n-1)
		// 		v := lerp(start, engine.Camera.Position, t)
		// 		followPath = append(followPath, v)
		// 	}
		// 	followId = 0
		// }

		// if followId < len(followPath) {
		// 	monkeyObj.Position = followPath[followId]
		// 	followId++
		// }

		pointlight.Position = engine.Camera.Position
		engine.Camera.Speed = 200 * engine.TSystem.DeltaTime

		inDump := 0
		for cx := -10; cx <= 10; cx++ {
			for cz := -10; cz <= 10; cz++ {
				chunkMesh := generator.GenerateChunk(cx, cz, chunkSize, chunkSize, float64(engine.TSystem.Ticks)/10)

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
				inDump++
			}
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

// func lerp(a, b vec3.T, t float32) vec3.T {
// 	return vec3.T{
// 		a[0] + t*(b[0]-a[0]),
// 		a[1] + t*(b[1]-a[1]),
// 		a[2] + t*(b[2]-a[2]),
// 	}
// }
