package api

import (
	"errors"
	"sync"

	"github.com/striter-no/softengine/entity"
)

type SceneSystem struct {
	mu            sync.RWMutex
	objects       map[int]*entity.Object3D
	incrementalID int
}

func NewSceneSystem() *SceneSystem {
	return &SceneSystem{
		objects: make(map[int]*entity.Object3D),
	}
}

func (s *SceneSystem) Add(obj *entity.Object3D) (int, error) {
	if obj == nil {
		return 0, errors.New("cannot add nil object")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.incrementalID
	s.objects[id] = obj
	s.incrementalID++
	return id, nil
}

func (s *SceneSystem) Get(id int) (*entity.Object3D, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if obj, ok := s.objects[id]; ok {
		return obj, nil
	}
	return nil, errors.New("object not found")
}

func (s *SceneSystem) Remove(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, id)
}

func (s *SceneSystem) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.objects)
}

func (s *SceneSystem) ForEach(fn func(id int, obj *entity.Object3D) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id, obj := range s.objects {
		if !fn(id, obj) {
			return
		}
	}
}

func (s *SceneSystem) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects = nil
}
