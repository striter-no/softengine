package phyapi

import (
	"github.com/ungerik/go3d/vec3"
)

// ColliderKind describes which shape a PhysicsNode uses.
type ColliderKind int

const (
	ColliderNone   ColliderKind = iota // no collider — pure kinematic body
	ColliderAABB                       // axis-aligned box (Half-extents)
	ColliderSphere                     // sphere (Radius)
)

type Collider struct {
	Kind   ColliderKind
	Half   vec3.T  // AABB half-extents on (X, Y, Z) — full size is Half*2
	Radius float32 // sphere radius
}

func AABBHalf(half vec3.T) Collider {
	return Collider{Kind: ColliderAABB, Half: half}
}

func AABBSize(size vec3.T) Collider {
	return Collider{Kind: ColliderAABB, Half: vec3.T{size[0] * 0.5, size[1] * 0.5, size[2] * 0.5}}
}

func SphereR(r float32) Collider {
	return Collider{Kind: ColliderSphere, Radius: r}
}

func (c Collider) Scaled(scale vec3.T) Collider {
	switch c.Kind {
	case ColliderAABB:
		return Collider{
			Kind: ColliderAABB,
			Half: vec3.T{
				c.Half[0] * scale[0],
				c.Half[1] * scale[1],
				c.Half[2] * scale[2],
			},
		}
	case ColliderSphere:
		m := scale[0]
		if scale[1] > m {
			m = scale[1]
		}
		if scale[2] > m {
			m = scale[2]
		}
		return Collider{Kind: ColliderSphere, Radius: c.Radius * m}
	default:
		return c
	}
}

type CompoundCollider struct {
	Items []CompoundItem
}

type CompoundItem struct {
	Offset   vec3.T
	Collider Collider
}

func NewCompoundCollider(cap int) *CompoundCollider {
	return &CompoundCollider{Items: make([]CompoundItem, 0, cap)}
}

func (cc *CompoundCollider) Add(offset vec3.T, c Collider) {
	cc.Items = append(cc.Items, CompoundItem{Offset: offset, Collider: c})
}

func (cc *CompoundCollider) WorldCenter(nodePos vec3.T, i int) vec3.T {
	it := cc.Items[i]
	return vec3.T{
		nodePos[0] + it.Offset[0],
		nodePos[1] + it.Offset[1],
		nodePos[2] + it.Offset[2],
	}
}
