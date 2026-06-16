package phyapi

import "github.com/ungerik/go3d/vec3"

// PhysicsNode — rigid body: position + velocity + mass + collider.

type PhysicsNode struct {
	ObjectID int

	UserData any

	// Kinematic state.
	Position     vec3.T
	Velocity     vec3.T
	Acceleration vec3.T

	// Body properties.
	Mass          float32
	IsStatic      bool
	LinearDamping float32

	// Surface properties.
	Friction float32

	// Runtime collision info.
	IsGrounded     bool
	ContactNormals []vec3.T

	// Collider.
	Collider Collider
	Compound *CompoundCollider
}

func NewStaticNode(objID int, position vec3.T) *PhysicsNode {
	return &PhysicsNode{
		ObjectID: objID,
		Position: position,
		Mass:     0,
		IsStatic: true,
	}
}

func NewDynamicNode(objID int, position vec3.T, mass float32) *PhysicsNode {
	return &PhysicsNode{
		ObjectID:      objID,
		Position:      position,
		Mass:          mass,
		IsStatic:      false,
		LinearDamping: 0.99,
		Friction:      DefaultFriction,
	}
}

func (n *PhysicsNode) SetCollider(c Collider) {
	n.Collider = c
	n.Compound = nil
}

func (n *PhysicsNode) SetCompound(cc *CompoundCollider) {
	n.Compound = cc
	n.Collider = Collider{Kind: ColliderNone}
}

func (n *PhysicsNode) HasCollider() bool {
	return n.Collider.Kind != ColliderNone || (n.Compound != nil && len(n.Compound.Items) > 0)
}

func (n *PhysicsNode) ApplyForce(force vec3.T) {
	if n.IsStatic || n.Mass <= 0 {
		return
	}
	n.Acceleration[0] += force[0] / n.Mass
	n.Acceleration[1] += force[1] / n.Mass
	n.Acceleration[2] += force[2] / n.Mass
}

func (n *PhysicsNode) ApplyImpulse(impulse vec3.T) {
	if n.IsStatic || n.Mass <= 0 {
		return
	}
	n.Velocity[0] += impulse[0] / n.Mass
	n.Velocity[1] += impulse[1] / n.Mass
	n.Velocity[2] += impulse[2] / n.Mass
}

func (n *PhysicsNode) invMass() float32 {
	if n.IsStatic || n.Mass <= 0 {
		return 0
	}
	return 1.0 / n.Mass
}
