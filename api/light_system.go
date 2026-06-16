package api

import (
	"sync"

	"github.com/striter-no/softengine/lights"
)

type LightSystem struct {
	mu            sync.RWMutex
	incrementalID int

	Config lights.LightingConfig // PointLights, SpotLights, Directional
	Fog    lights.FogConfig
}

func NewLightSystem() *LightSystem {
	return &LightSystem{
		Config: lights.LightingConfig{
			PointLights: make(map[int]*lights.PointLight, 0),
			SpotLights:  make(map[int]*lights.SpotLight, 0),
		},
	}
}

func (ls *LightSystem) NewSpotLight(conf *lights.SpotLight) int {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	id := ls.incrementalID
	ls.Config.SpotLights[id] = conf
	ls.incrementalID++
	return id
}

func (ls *LightSystem) NewPointLight(conf *lights.PointLight) int {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	id := ls.incrementalID
	ls.Config.PointLights[id] = conf
	ls.incrementalID++
	return id
}

func (ls *LightSystem) RemovePointLight(id int) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	delete(ls.Config.PointLights, id)
}

func (ls *LightSystem) RemoveSpotLight(id int) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	delete(ls.Config.SpotLights, id)
}

func (ls *LightSystem) SetFog(conf lights.FogConfig) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.Fog = conf
}

func (ls *LightSystem) End() {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.Config.PointLights = nil
	ls.Config.SpotLights = nil
}
