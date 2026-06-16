package physics

// BodyKind describes the collider shape attached to a Body.
type BodyKind int

const (
	ShapeAABB BodyKind = iota
	ShapeSphere
)

// Body is a rigid body: position + velocity + mass + collider.
//
// A body with Static == true ignores gravity and is never moved
// by collision resolution (walls, floor, etc.).
type Body struct {
	Pos    [3]float32 // center of the body
	Vel    [3]float32 // m/s
	Mass   float32    // > 0 for dynamic, ignored if Static
	Static bool

	Kind   BodyKind
	Half   [3]float32 // AABB half-extents on (X, Y, Z)
	Radius float32    // sphere radius
}

// NewAABB creates an AABB body.
// size is the FULL dimensions (width X, height Y, length Z),
// matching coldet's AABB constructor convention.
func NewAABB(pos [3]float32, size [3]float32, mass float32, static bool) *Body {
	return &Body{
		Pos:    pos,
		Mass:   mass,
		Static: static,
		Kind:   ShapeAABB,
		Half:   [3]float32{size[0] * 0.5, size[1] * 0.5, size[2] * 0.5},
	}
}

// NewSphere creates a sphere body.
func NewSphere(pos [3]float32, radius float32, mass float32, static bool) *Body {
	return &Body{
		Pos:    pos,
		Mass:   mass,
		Static: static,
		Kind:   ShapeSphere,
		Radius: radius,
	}
}

// invMass returns 0 for static bodies (treated as infinite mass).
func (b *Body) invMass() float32 {
	if b.Static || b.Mass <= 0 {
		return 0
	}
	return 1.0 / b.Mass
}

// min/max helpers (AABB bounds from center + half-extents).
func (b *Body) aabbMin() [3]float32 {
	return [3]float32{b.Pos[0] - b.Half[0], b.Pos[1] - b.Half[1], b.Pos[2] - b.Half[2]}
}
func (b *Body) aabbMax() [3]float32 {
	return [3]float32{b.Pos[0] + b.Half[0], b.Pos[1] + b.Half[1], b.Pos[2] + b.Half[2]}
}
