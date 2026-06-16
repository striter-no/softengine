package phyapi

import (
	"context"
	"sync"
	"time"

	"github.com/ungerik/go3d/vec3"
)

const (
	DefaultGravity    = -200.0
	DefaultIterations = 4
	DefaultFriction   = 0.3
)

type PhysicsSystem struct {
	mu    sync.RWMutex
	nodes map[int]*PhysicsNode

	TerrainHeightFunc func(x, z float32) float32
	Iterations        int

	ObjectSink func(n *PhysicsNode)

	Paused bool

	cancel context.CancelFunc
}

func NewPhysicsSystem() *PhysicsSystem {
	return &PhysicsSystem{
		nodes:      make(map[int]*PhysicsNode),
		Iterations: DefaultIterations,
	}
}

func (ps *PhysicsSystem) SetObjectSink(fn func(n *PhysicsNode)) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.ObjectSink = fn
}

func (ps *PhysicsSystem) AddNode(node *PhysicsNode) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.nodes[node.ObjectID] = node
}

func (ps *PhysicsSystem) RemoveNode(id int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.nodes, id)
}

func (ps *PhysicsSystem) Get(id int) (*PhysicsNode, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	n, ok := ps.nodes[id]
	return n, ok
}

func (ps *PhysicsSystem) Count() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.nodes)
}

func (ps *PhysicsSystem) ForEach(fn func(id int, n *PhysicsNode) bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	for id, n := range ps.nodes {
		if !fn(id, n) {
			return
		}
	}
}

func (ps *PhysicsSystem) Lock()   { ps.mu.Lock() }
func (ps *PhysicsSystem) Unlock() { ps.mu.Unlock() }

func (ps *PhysicsSystem) MutateNode(id int, fn func(n *PhysicsNode) bool) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	n, ok := ps.nodes[id]
	if !ok {
		return false
	}
	return fn(n)
}

func (ps *PhysicsSystem) SetTerrainHeightFunc(f func(x, z float32) float32) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.TerrainHeightFunc = f
}

func (ps *PhysicsSystem) SetPaused(p bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.Paused = p
}

func (ps *PhysicsSystem) IsPaused() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.Paused
}

func (ps *PhysicsSystem) Update(dt float64) {
	ps.mu.RLock()
	if ps.Paused {
		ps.mu.RUnlock()
		return
	}

	nodes := make([]*PhysicsNode, 0, len(ps.nodes))
	for _, n := range ps.nodes {
		nodes = append(nodes, n)
	}
	terrainFn := ps.TerrainHeightFunc
	iters := ps.Iterations
	sink := ps.ObjectSink
	if iters <= 0 {
		iters = DefaultIterations
	}
	ps.mu.RUnlock()

	dt32 := float32(dt)

	// 1. Integrate.
	for _, n := range nodes {
		n.IsGrounded = false
		n.ContactNormals = n.ContactNormals[:0]

		if n.IsStatic {
			continue
		}

		n.Velocity[0] += n.Acceleration[0] * dt32
		n.Velocity[1] += (n.Acceleration[1] + DefaultGravity) * dt32
		n.Velocity[2] += n.Acceleration[2] * dt32

		n.Acceleration[0] = 0
		n.Acceleration[1] = 0
		n.Acceleration[2] = 0

		if n.LinearDamping > 0 && n.LinearDamping < 1 {
			n.Velocity[0] *= n.LinearDamping
			n.Velocity[1] *= n.LinearDamping
			n.Velocity[2] *= n.LinearDamping
		}

		n.Position[0] += n.Velocity[0] * dt32
		n.Position[1] += n.Velocity[1] * dt32
		n.Position[2] += n.Velocity[2] * dt32
	}

	// 2. Terrain.
	if terrainFn != nil {
		for _, n := range nodes {
			if n.IsStatic {
				continue
			}
			terrainY := terrainFn(n.Position[0], n.Position[2])
			if n.Position[1] < terrainY {
				n.Position[1] = terrainY
				if n.Velocity[1] < 0 {
					n.Velocity[1] = 0
				}
				n.IsGrounded = true
				n.ContactNormals = append(n.ContactNormals, vec3.T{0, 1, 0})
			}
		}
	}

	// 3. Narrowphase.
	for iter := 0; iter < iters; iter++ {
		for i := 0; i < len(nodes); i++ {
			for j := i + 1; j < len(nodes); j++ {
				a, b := nodes[i], nodes[j]
				if a.IsStatic && b.IsStatic {
					continue
				}
				if !a.HasCollider() || !b.HasCollider() {
					continue
				}
				resolvePair(a, b)
			}
		}
	}

	// 4. Friction.
	applyFriction(nodes, dt32, DefaultGravity)

	if sink != nil {
		for _, n := range nodes {
			sink(n)
		}
	}
}

func (ps *PhysicsSystem) StartLoop(ctx context.Context, tps int) {
	innerCtx, cancel := context.WithCancel(ctx)
	ps.mu.Lock()
	ps.cancel = cancel
	ps.mu.Unlock()

	ticker := time.NewTicker(time.Second / time.Duration(tps))
	dt := 1.0 / float64(tps)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-innerCtx.Done():
				return
			case <-ticker.C:
				ps.Update(dt)
			}
		}
	}()
}

func (ps *PhysicsSystem) End() {
	ps.mu.Lock()
	if ps.cancel != nil {
		ps.cancel()
		ps.cancel = nil
	}
	ps.nodes = nil
	ps.mu.Unlock()
}
