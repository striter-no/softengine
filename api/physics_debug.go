package api

import (
	"github.com/striter-no/softengine/api/phyapi"
	"github.com/striter-no/softengine/api/shaders"
	"github.com/striter-no/softengine/entity"
	"github.com/striter-no/softgo/render"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

func SyncToObject3D(n *phyapi.PhysicsNode) {
	obj, ok := n.UserData.(*entity.Object3D)
	if !ok || obj == nil {
		return
	}
	obj.Position = n.Position
	obj.UpdateMat()
	obj.GetModelMatrix()
}

// api.LinkNode(engine.Physics, node, obj)
func LinkNode(physics *phyapi.PhysicsSystem, node *phyapi.PhysicsNode, obj *entity.Object3D) {
	node.UserData = obj
	physics.SetObjectSink(SyncToObject3D)
}

type PhysicsDebugStyle struct {
	DynamicColor       vec4.T
	StaticColor        vec4.T
	GroundedColor      vec4.T
	ShowContactNormals bool
	ContactNormalLen   float32
	ContactNormalColor vec4.T
	Alpha              float32
}

func DefaultPhysicsDebugStyle() PhysicsDebugStyle {
	return PhysicsDebugStyle{
		DynamicColor:       vec4.T{1, 0, 0, 200},
		StaticColor:        vec4.T{0, 0, 1, 200},
		GroundedColor:      vec4.T{0, 1, 0, 200},
		ShowContactNormals: true,
		ContactNormalLen:   20.0,
		ContactNormalColor: vec4.T{1, 1, 0, 220},
		Alpha:              200,
	}
}

type PhysicsDebugSystem struct {
	physics *phyapi.PhysicsSystem
	scene   *SceneSystem

	obj     *entity.Object3D
	style   PhysicsDebugStyle
	enabled bool
}

func NewPhysicsDebugSystem(physics *phyapi.PhysicsSystem, scene *SceneSystem) (*PhysicsDebugSystem, int, error) {
	obj := entity.NewObject3D(
		vec3.T{0, 0, 0},
		vec3.T{0, 0, 0},
		vec3.T{1, 1, 1},
		false, false,
	)
	obj.IsSkybox = false

	tex := entity.NewModelColorTexture(255, 0, 0, 100)
	obj.Compose(nil, tex, defaultDebugMaterial())

	id, err := scene.Add(obj)
	if err != nil {
		return nil, 0, err
	}

	return &PhysicsDebugSystem{
		physics: physics,
		scene:   scene,
		obj:     obj,
		style:   DefaultPhysicsDebugStyle(),
		enabled: true,
	}, id, nil
}

func (d *PhysicsDebugSystem) Disable() {
	d.enabled = false
	d.clearMesh()
}

func (d *PhysicsDebugSystem) clearMesh() {
	if len(d.obj.Parts) > 0 {
		d.obj.Parts[0].Mesh = []render.TBO{}
	}
}

func (d *PhysicsDebugSystem) SetStyle(s PhysicsDebugStyle) { d.style = s }
func (d *PhysicsDebugSystem) Enable()                      { d.enabled = true }
func (d *PhysicsDebugSystem) Enabled() bool                { return d.enabled }
func (d *PhysicsDebugSystem) Object() *entity.Object3D     { return d.obj }
func (d *PhysicsDebugSystem) SetMaterial(m shaders.Material) {
	if len(d.obj.Parts) > 0 {
		d.obj.Parts[0].Material = m
	}
}

func (d *PhysicsDebugSystem) Rebuild() {
	if !d.enabled {
		d.clearMesh()
		return
	}

	var tris []render.TBO

	d.physics.ForEach(func(_ int, n *phyapi.PhysicsNode) bool {
		color := d.colorFor(n)
		tris = append(tris, d.nodeMesh(n, color)...)
		if d.style.ShowContactNormals {
			tris = append(tris, d.contactNormalMesh(n)...)
		}
		return true
	})

	if len(d.obj.Parts) > 0 {
		d.obj.Parts[0].Mesh = tris
	} else {
		d.obj.Compose(tris, entity.NewModelColorTexture(255, 0, 0, 100), defaultDebugMaterial())
	}
}

func (d *PhysicsDebugSystem) colorFor(n *phyapi.PhysicsNode) vec4.T {
	if n.IsStatic {
		return d.style.StaticColor
	}
	if n.IsGrounded {
		return d.style.GroundedColor
	}
	return d.style.DynamicColor
}

func (d *PhysicsDebugSystem) nodeMesh(n *phyapi.PhysicsNode, color vec4.T) []render.TBO {
	if n.Compound != nil && len(n.Compound.Items) > 0 {
		var out []render.TBO
		for _, it := range n.Compound.Items {
			center := vec3.T{
				n.Position[0] + it.Offset[0],
				n.Position[1] + it.Offset[1],
				n.Position[2] + it.Offset[2],
			}
			out = append(out, colliderMesh(it.Collider, center, color)...)
		}
		return out
	}
	return colliderMesh(n.Collider, n.Position, color)
}

func colliderMesh(c phyapi.Collider, center vec3.T, color vec4.T) []render.TBO {
	switch c.Kind {
	case phyapi.ColliderAABB:
		return generateDebugBoxAt(center, c.Half, color)
	case phyapi.ColliderSphere:
		half := vec3.T{c.Radius, c.Radius, c.Radius}
		return generateDebugBoxAt(center, half, color)
	default:
		return nil
	}
}

func (d *PhysicsDebugSystem) contactNormalMesh(n *phyapi.PhysicsNode) []render.TBO {
	if len(n.ContactNormals) == 0 {
		return nil
	}
	var out []render.TBO
	L := d.style.ContactNormalLen
	color := d.style.ContactNormalColor
	thickness := float32(0.5)

	for _, cn := range n.ContactNormals {
		start := n.Position
		end := vec3.T{
			start[0] + cn[0]*L,
			start[1] + cn[1]*L,
			start[2] + cn[2]*L,
		}
		out = append(out, generateDebugLine(start, end, thickness, color)...)
	}
	return out
}

func generateDebugBoxAt(center, half vec3.T, color vec4.T) []render.TBO {
	v := [8]vec3.T{
		{center[0] - half[0], center[1] - half[1], center[2] - half[2]},
		{center[0] + half[0], center[1] - half[1], center[2] - half[2]},
		{center[0] + half[0], center[1] + half[1], center[2] - half[2]},
		{center[0] - half[0], center[1] + half[1], center[2] - half[2]},
		{center[0] - half[0], center[1] - half[1], center[2] + half[2]},
		{center[0] + half[0], center[1] - half[1], center[2] + half[2]},
		{center[0] + half[0], center[1] + half[1], center[2] + half[2]},
		{center[0] - half[0], center[1] + half[1], center[2] + half[2]},
	}
	idx := []int{
		0, 1, 2, 0, 2, 3,
		5, 4, 7, 5, 7, 6,
		4, 0, 3, 4, 3, 7,
		1, 5, 6, 1, 6, 2,
		3, 2, 6, 3, 6, 7,
		4, 5, 1, 4, 1, 0,
	}
	out := make([]render.TBO, 0, len(idx)/3)
	for i := 0; i < len(idx); i += 3 {
		out = append(out, render.TBO{
			V0: v[idx[i]], V1: v[idx[i+1]], V2: v[idx[i+2]],
			C0: color, C1: color, C2: color,
		})
	}
	return out
}

func generateDebugLine(a, b vec3.T, thickness float32, color vec4.T) []render.TBO {
	dx := (b[0] - a[0]) * 0.5
	dy := (b[1] - a[1]) * 0.5
	dz := (b[2] - a[2]) * 0.5
	center := vec3.T{(a[0] + b[0]) * 0.5, (a[1] + b[1]) * 0.5, (a[2] + b[2]) * 0.5}
	half := vec3.T{
		absFMax(dx, thickness),
		absFMax(dy, thickness),
		absFMax(dz, thickness),
	}
	return generateDebugBoxAt(center, half, color)
}

func absFMax(v, min float32) float32 {
	if v < 0 {
		v = -v
	}
	if v < min {
		return min
	}
	return v
}

func defaultDebugMaterial() shaders.Material {
	return shaders.Material{
		Ambient:   vec3.T{1, 0, 0},
		Diffuse:   vec3.T{1, 0, 0},
		Specular:  vec3.T{0, 0, 0},
		Shininess: 1.0,
	}
}
