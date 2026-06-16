package entity

import (
	"errors"
	"math"
	"sort"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/striter-no/softengine/api/shaders"
	textures "github.com/striter-no/softgo/loader"
	"github.com/striter-no/softgo/render"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
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
	BaseColor vec4.T
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

func NewModelColorTexture(r, g, b, a float32) ModelTexture {
	return ModelTexture{
		TexType:   MODEL_TEXTURE_NONE,
		BaseColor: vec4.T{r, g, b, a},
	}
}

type LOD struct {
	Distance float32
	Mesh     []render.TBO
}

type ModelPart struct {
	Mesh     []render.TBO
	Texture  ModelTexture
	Material shaders.Material
	LODs     []LOD
}

type Object3D struct {
	Parts []ModelPart

	Position vec3.T
	Rotation mgl32.Quat
	Scale    vec3.T

	BaseRadius float32

	isDirty     bool
	modelMatrix mgl32.Mat4
	CanBeLit    bool
	CastShadows bool
	IsSkybox    bool

	DebugTriCount int
}

func NewObject3D(position, rotation, scale vec3.T, canBeLit, castShadows bool) *Object3D {
	return &Object3D{
		Parts:         make([]ModelPart, 0),
		Position:      position,
		Rotation:      mgl32.AnglesToQuat(rotation[0], rotation[1], rotation[2], mgl32.XYZ),
		Scale:         scale,
		BaseRadius:    0.0,
		isDirty:       true,
		modelMatrix:   mgl32.Ident4(),
		CanBeLit:      canBeLit,
		CastShadows:   castShadows,
		DebugTriCount: 0,
	}
}

func (o *Object3D) Compose(mesh []render.TBO, texture ModelTexture, material shaders.Material) {
	if len(mesh) == 0 {
		return
	}

	o.Parts = append(o.Parts, ModelPart{
		Mesh:     mesh,
		Texture:  texture,
		Material: material,
		LODs:     make([]LOD, 0),
	})

	o.DebugTriCount += len(mesh)

	o.recalculateBaseRadius()
	o.isDirty = true
}

func (o *Object3D) recalculateBaseRadius() {
	var maxSq float32
	for _, part := range o.Parts {
		for _, tbo := range part.Mesh {
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
	}
	o.BaseRadius = float32(math.Sqrt(float64(maxSq)))
}

func (o *Object3D) ChangeOmniDir(mode bool) {
	for i := range o.Parts {
		for j := range o.Parts[i].Mesh {
			o.Parts[i].Mesh[j].OmniDir = mode
		}
		for k := range o.Parts[i].LODs {
			for j := range o.Parts[i].LODs[k].Mesh {
				o.Parts[i].LODs[k].Mesh[j].OmniDir = mode
			}
		}
	}
}

func (o *Object3D) AddPartLOD(partIndex int, mesh []render.TBO, distance float32) {
	if partIndex < 0 || partIndex >= len(o.Parts) {
		return
	}

	o.Parts[partIndex].LODs = append(o.Parts[partIndex].LODs, LOD{
		Distance: distance,
		Mesh:     mesh,
	})

	sort.Slice(o.Parts[partIndex].LODs, func(i, j int) bool {
		return o.Parts[partIndex].LODs[i].Distance > o.Parts[partIndex].LODs[j].Distance
	})
}

func (o *Object3D) UpdateMat() {
	o.isDirty = true
}

func (o *Object3D) GetActiveMesh(partIndex int, distance float32) []render.TBO {
	if partIndex < 0 || partIndex >= len(o.Parts) {
		return nil
	}

	part := &o.Parts[partIndex]
	activeMesh := part.Mesh
	bestDist := float32(-1.0)

	for _, lod := range part.LODs {
		if distance >= lod.Distance && lod.Distance > bestDist {
			activeMesh = lod.Mesh
			bestDist = lod.Distance
		}
	}

	return activeMesh
}

func (o *Object3D) Clone() *Object3D {
	clonedParts := make([]ModelPart, len(o.Parts))
	copy(clonedParts, o.Parts)

	return &Object3D{
		Parts:       clonedParts,
		Position:    o.Position,
		Rotation:    o.Rotation,
		Scale:       o.Scale,
		BaseRadius:  o.BaseRadius,
		isDirty:     true,
		modelMatrix: mgl32.Ident4(),
		CanBeLit:    o.CanBeLit,
		CastShadows: o.CastShadows,
		IsSkybox:    o.IsSkybox,
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

func (o *Object3D) CalculateDecomposedBoxes(maxBoxes int) ([]vec3.T, []vec3.T) {
	var allTris []render.TBO
	for _, part := range o.Parts {
		allTris = append(allTris, part.Mesh...)
	}

	if len(allTris) == 0 {
		return nil, nil
	}

	groups := [][]render.TBO{allTris}

	for len(groups) < maxBoxes {
		largestIdx := -1
		var maxVolume float32 = -1.0
		var lMin, lMax vec3.T

		for i, group := range groups {
			if len(group) < 2 {
				continue
			}
			min, max := getGroupAABB(group)
			vol := (max[0] - min[0]) * (max[1] - min[1]) * (max[2] - min[2])

			if vol > maxVolume {
				maxVolume = vol
				largestIdx = i
				lMin, lMax = min, max
			}
		}

		if largestIdx == -1 {
			break
		}

		targetGroup := groups[largestIdx]

		dx := lMax[0] - lMin[0]
		dy := lMax[1] - lMin[1]
		dz := lMax[2] - lMin[2]

		axis := 0
		if dy > dx && dy > dz {
			axis = 1
		} else if dz > dx && dz > dy {
			axis = 2
		}

		mid := lMin[axis] + (lMax[axis]-lMin[axis])/2.0

		var left, right []render.TBO
		for _, t := range targetGroup {
			centroid := (t.V0[axis] + t.V1[axis] + t.V2[axis]) / 3.0
			if centroid < mid {
				left = append(left, t)
			} else {
				right = append(right, t)
			}
		}

		if len(left) == 0 || len(right) == 0 {
			half := len(targetGroup) / 2
			left = targetGroup[:half]
			right = targetGroup[half:]
		}

		groups[largestIdx] = left
		groups = append(groups, right)
	}

	var offsets []vec3.T
	var halfSizes []vec3.T

	for _, group := range groups {
		if len(group) == 0 {
			continue
		}

		min, max := getGroupAABB(group)

		halfX := (max[0] - min[0]) / 2
		halfY := (max[1] - min[1]) / 2
		halfZ := (max[2] - min[2]) / 2

		if halfX < 0.01 {
			halfX = 0.01
		}
		if halfY < 0.01 {
			halfY = 0.01
		}
		if halfZ < 0.01 {
			halfZ = 0.01
		}

		offsets = append(offsets, vec3.T{
			(max[0] + min[0]) / 2,
			(max[1] + min[1]) / 2,
			(max[2] + min[2]) / 2,
		})
		halfSizes = append(halfSizes, vec3.T{halfX, halfY, halfZ})
	}

	return offsets, halfSizes
}

func getGroupAABB(group []render.TBO) (vec3.T, vec3.T) {
	var minX, minY, minZ float32 = math.MaxFloat32, math.MaxFloat32, math.MaxFloat32
	var maxX, maxY, maxZ float32 = -math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32

	for _, tbo := range group {
		for _, v := range []vec3.T{tbo.V0, tbo.V1, tbo.V2} {
			if v[0] < minX {
				minX = v[0]
			}
			if v[1] < minY {
				minY = v[1]
			}
			if v[2] < minZ {
				minZ = v[2]
			}

			if v[0] > maxX {
				maxX = v[0]
			}
			if v[1] > maxY {
				maxY = v[1]
			}
			if v[2] > maxZ {
				maxZ = v[2]
			}
		}
	}
	return vec3.T{minX, minY, minZ}, vec3.T{maxX, maxY, maxZ}
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
