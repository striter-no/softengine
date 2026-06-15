package entity

import (
	"errors"
	"math"

	"github.com/go-gl/mathgl/mgl32"
	textures "github.com/striter-no/softgo/loader"
	"github.com/striter-no/softgo/render"
	"github.com/ungerik/go3d/vec3"
)

type ModelTexType int

const (
	MODEL_TEXTURE_IMAGE ModelTexType = iota
	MODEL_TEXTURE_NONE
	MODEL_TEXTURE_ANIM
)

type ModelTexture struct {
	TexType   ModelTexType
	Texture   *render.Texture
	Animation *textures.Animation
	BaseColor vec3.T
}

func NewModelImageTexture(filename string) (ModelTexture, error) {
	tex := textures.ConvertImageToTexture(filename)
	if tex == nil {
		return ModelTexture{}, errors.New("Failed to load image")
	}

	return ModelTexture{
		TexType: MODEL_TEXTURE_IMAGE,
		Texture: tex,
	}, nil
}

func NewModelAnimTexture(filename string) (ModelTexture, error) {
	anim, err := textures.ConvertGIFToAnimation(filename)
	if err != nil {
		return ModelTexture{}, err
	}

	return ModelTexture{
		TexType:   MODEL_TEXTURE_ANIM,
		Animation: anim,
	}, nil
}

func NewModelColorTexture(r, g, b float32) ModelTexture {
	return ModelTexture{
		TexType:   MODEL_TEXTURE_NONE,
		BaseColor: vec3.T{r, g, b},
	}
}

type LOD struct {
	Distance float32
	Mesh     []render.TBO
}

type Object3D struct {
	Texture ModelTexture

	Position vec3.T
	Rotation mgl32.Quat
	Scale    vec3.T

	Mesh       []render.TBO
	BaseRadius float32
	LODs       []LOD

	isDirty     bool
	modelMatrix mgl32.Mat4
	CanBeLit    bool
	CastShadows bool
	IsSkybox    bool
}

func NewObject3D(position, rotation, scale vec3.T, mesh []render.TBO, texture ModelTexture, canBeLit, castShadows bool) *Object3D {
	var maxSq float32
	for _, tbo := range mesh {
		v0Sq := tbo.V0[0]*tbo.V0[0] + tbo.V0[1]*tbo.V0[1] + tbo.V0[2]*tbo.V0[2]
		if v0Sq > maxSq {
			maxSq = v0Sq
		}

		v1Sq := tbo.V1[0]*tbo.V1[0] + tbo.V1[1]*tbo.V1[1] + tbo.V1[2]*tbo.V1[2]
		if v1Sq > maxSq {
			maxSq = v1Sq
		}

		v2Sq := tbo.V2[0]*tbo.V2[0] + tbo.V2[1]*tbo.V2[1] + tbo.V2[2]*tbo.V2[2]
		if v2Sq > maxSq {
			maxSq = v2Sq
		}
	}
	baseRadius := float32(math.Sqrt(float64(maxSq)))

	return &Object3D{
		LODs:        make([]LOD, 0),
		Texture:     texture,
		Position:    position,
		Rotation:    mgl32.AnglesToQuat(rotation[0], rotation[1], rotation[2], mgl32.XYZ),
		Scale:       scale,
		Mesh:        mesh,
		BaseRadius:  baseRadius,
		isDirty:     true,
		modelMatrix: mgl32.Ident4(),
		CanBeLit:    canBeLit,
		CastShadows: castShadows,
	}
}

func (o *Object3D) ChangeOmniDir(mode bool) {
	for i := range o.Mesh {
		o.Mesh[i].OmniDir = mode
	}

	for k := range o.LODs {
		for i := range o.LODs[k].Mesh {
			o.LODs[k].Mesh[i].OmniDir = mode
		}
	}
}

func (o *Object3D) AddLOD(mesh []render.TBO, distance float32) {
	o.LODs = append(o.LODs, LOD{
		Distance: distance,
		Mesh:     mesh,
	})
}

func (o *Object3D) UpdateMat() {
	o.isDirty = true
}

func (o *Object3D) GetActiveMesh(distance float32) []render.TBO {
	activeMesh := o.Mesh
	bestDist := float32(-1.0)

	for _, lod := range o.LODs {
		if distance >= lod.Distance && lod.Distance > bestDist {
			activeMesh = lod.Mesh
			bestDist = lod.Distance
		}
	}

	return activeMesh
}

func (o *Object3D) Clone() *Object3D {
	return &Object3D{
		CanBeLit: o.CanBeLit,
		LODs:     o.LODs,
		Texture:  o.Texture,
		Position: o.Position,
		Rotation: o.Rotation,
		Scale:    o.Scale,
		Mesh:     o.Mesh,
		isDirty:  true,
	}
}

func (o *Object3D) Translate(vec vec3.T) {
	o.Position[0] += vec[0]
	o.Position[1] += vec[1]
	o.Position[2] += vec[2]
	o.isDirty = true
}

func (o *Object3D) SetScale(vec vec3.T) {
	o.Scale[0] = vec[0]
	o.Scale[1] = vec[1]
	o.Scale[2] = vec[2]
	o.isDirty = true
}

func (o *Object3D) RotateEuler(vec vec3.T) {
	delta := mgl32.AnglesToQuat(vec[0], vec[1], vec[2], mgl32.XYZ)

	o.Rotation = o.Rotation.Mul(delta).Normalize()
	o.isDirty = true
}

func (o *Object3D) SetRotationEuler(vec vec3.T) {
	o.Rotation = mgl32.AnglesToQuat(vec[0], vec[1], vec[2], mgl32.XYZ)
	o.isDirty = true
}

func (o *Object3D) LookAt(pos vec3.T, inverse bool) {
	eye := mgl32.Vec3{o.Position[0], o.Position[1], o.Position[2]}
	center := mgl32.Vec3{pos[0], pos[1], pos[2]}
	up := mgl32.Vec3{0, 1, 0}

	if center.Sub(eye).Len() < 1e-5 {
		return
	}

	if inverse {
		direction := center.Sub(eye)
		center = eye.Sub(direction)
	}

	viewMat := mgl32.LookAtV(eye, center, up)

	rotMat := mgl32.Mat4{
		viewMat[0], viewMat[4], viewMat[8], 0,
		viewMat[1], viewMat[5], viewMat[9], 0,
		viewMat[2], viewMat[6], viewMat[10], 0,
		0, 0, 0, 1,
	}

	o.Rotation = mgl32.Mat4ToQuat(rotMat).Normalize()
	o.isDirty = true
}

func (o *Object3D) RotateAxisAngle(axis mgl32.Vec3, angle float32) {
	axis = axis.Normalize()
	delta := mgl32.QuatRotate(angle, axis)

	o.Rotation = o.Rotation.Mul(delta).Normalize()
	o.isDirty = true
}

func (o *Object3D) GetModelMatrix() mgl32.Mat4 {
	if o.isDirty {
		scaleMat := mgl32.Scale3D(o.Scale[0], o.Scale[1], o.Scale[2])
		rotMat := o.Rotation.Mat4()
		translateMat := mgl32.Translate3D(o.Position[0], o.Position[1], o.Position[2])

		o.modelMatrix = translateMat.Mul4(rotMat).Mul4(scaleMat)
		o.isDirty = false
	}

	return o.modelMatrix
}
