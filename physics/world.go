package physics

// World is the physics scene: a flat list of bodies + gravity.
// No broad-phase, no spatial structure — O(n^2) per step. Fine for a basic engine.
type World struct {
	Bodies  []*Body
	Gravity [3]float32
	// Iterations controls how many resolve passes we run per step.
	// More iterations = more stable stacking, but slower.
	Iterations int
}

// NewWorld returns a world with earth-like gravity and 4 solver iterations.
func NewWorld() *World {
	return &World{
		Gravity:    [3]float32{0, -9.81, 0},
		Iterations: 4,
	}
}

// Add inserts a body into the world.
func (w *World) Add(b *Body) {
	w.Bodies = append(w.Bodies, b)
}

// Step advances the simulation by dt seconds.
//
// Pipeline:
//  1. Integrate (apply gravity to velocity, then velocity to position)
//  2. Detect + resolve collisions N times (solver iterations)
//     — resolution pushes bodies apart and zeroes velocity along
//     the collision normal ("stop on collision", no bounce).
func (w *World) Step(dt float32) {
	// 1. Integrate
	for _, b := range w.Bodies {
		if b.Static {
			continue
		}
		b.Vel[0] += w.Gravity[0] * dt
		b.Vel[1] += w.Gravity[1] * dt
		b.Vel[2] += w.Gravity[2] * dt

		b.Pos[0] += b.Vel[0] * dt
		b.Pos[1] += b.Vel[1] * dt
		b.Pos[2] += b.Vel[2] * dt
	}

	// 2. Resolve collisions
	for iter := 0; iter < w.Iterations; iter++ {
		for i := 0; i < len(w.Bodies); i++ {
			for j := i + 1; j < len(w.Bodies); j++ {
				a, b := w.Bodies[i], w.Bodies[j]
				if a.Static && b.Static {
					continue
				}
				resolve(a, b)
			}
		}
	}
}

// resolve dispatches to the right narrow-phase based on shapes.
// It uses coldet only as a yes/no gate; the actual push-out + velocity
// cancellation is done here, because coldet returns just bool.
func resolve(a, b *Body) {
	switch {
	case a.Kind == ShapeAABB && b.Kind == ShapeAABB:
		// Build coldet AABBs (size = full dims) and check.
		a1 := NewBoundingBox(a.Pos, a.Half[0]*2, a.Half[1]*2, a.Half[2]*2)
		a2 := NewBoundingBox(b.Pos, b.Half[0]*2, b.Half[1]*2, b.Half[2]*2)
		if !CheckAabbVsAabb(*a1, *a2) {
			return
		}
		resolveAABBs(a, b)

	case a.Kind == ShapeSphere && b.Kind == ShapeSphere:
		s1 := NewBoundingSphere(a.Pos, a.Radius)
		s2 := NewBoundingSphere(b.Pos, b.Radius)
		if !CheckSphereVsSphere(*s1, *s2) {
			return
		}
		resolveSpheres(a, b)

	default:
		// one is sphere, one is AABB — order them so `s` is sphere, `bx` is AABB.
		var s, bx *Body
		if a.Kind == ShapeSphere {
			s, bx = a, b
		} else {
			s, bx = b, a
		}
		box := NewBoundingBox(bx.Pos, bx.Half[0]*2, bx.Half[1]*2, bx.Half[2]*2)
		sph := NewBoundingSphere(s.Pos, s.Radius)
		if !CheckSphereVsAabb(*sph, *box) {
			return
		}
		resolveSphereAABB(s, bx)
	}
}
