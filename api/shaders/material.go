package shaders

import "github.com/ungerik/go3d/vec3"

type Material struct {
	Ambient   vec3.T
	Diffuse   vec3.T // base color
	Specular  vec3.T // specular color
	Shininess float32
}
