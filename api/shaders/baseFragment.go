package shaders

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/striter-no/softgo/api"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

func calculateShadow(fragPos vec4.T, lightSpaceMatrix mgl32.Mat4, shadowDepth []float32, shadowWidth, shadowHeight int) float32 {
	fragPos4 := mgl32.Vec4{fragPos[0], fragPos[1], fragPos[2], 1.0}
	lightPosV := lightSpaceMatrix.Mul4x1(fragPos4)

	ndcX := lightPosV.X() / lightPosV.W()
	ndcY := lightPosV.Y() / lightPosV.W()
	ndcZ := lightPosV.Z() / lightPosV.W()

	screenX := (ndcX + 1.0) * 0.5 * float32(shadowWidth)
	screenY := (1.0 - ndcY) * 0.5 * float32(shadowHeight)

	shadow := float32(0.0)
	px := int(screenX)
	py := int(screenY)

	if px >= 0 && px < shadowWidth && py >= 0 && py < shadowHeight && ndcZ < 1.0 && ndcZ > -1.0 {
		idx := py*shadowWidth + px
		closestDepth := shadowDepth[idx]
		currentDepth := ndcZ
		bias := float32(0.005)

		if currentDepth-bias > closestDepth {
			shadow = 0.7
		}
	}
	return shadow
}

func calculateLightContribution(
	lightDir vec3.T, distance float32, lightColor vec3.T, intensity float32,
	constant, linear, quadratic float32, norm vec3.T, viewDir vec3.T,
	matDiffuse vec3.T, matSpecular vec3.T, shininess float32,
) vec3.T {
	// 1. Diffuse
	diffuseFactor := norm[0]*lightDir[0] + norm[1]*lightDir[1] + norm[2]*lightDir[2]
	if diffuseFactor < 0 {
		diffuseFactor = 0
	}

	// 2. Specular (Phong)
	specularFactor := float32(0.0)
	if diffuseFactor > 0 {
		negL := vec3.T{-lightDir[0], -lightDir[1], -lightDir[2]}
		dotNL := negL[0]*norm[0] + negL[1]*norm[1] + negL[2]*norm[2]
		reflectDir := vec3.T{
			negL[0] - 2.0*dotNL*norm[0],
			negL[1] - 2.0*dotNL*norm[1],
			negL[2] - 2.0*dotNL*norm[2],
		}
		specDot := viewDir[0]*reflectDir[0] + viewDir[1]*reflectDir[1] + viewDir[2]*reflectDir[2]
		if specDot < 0 {
			specDot = 0
		}
		specularFactor = float32(math.Pow(float64(specDot), float64(shininess)))
	}

	// 3. Attenuation
	attenuation := intensity / (constant + linear*distance + quadratic*(distance*distance))

	return vec3.T{
		(matDiffuse[0]*lightColor[0]*diffuseFactor + matSpecular[0]*lightColor[0]*specularFactor) * attenuation,
		(matDiffuse[1]*lightColor[1]*diffuseFactor + matSpecular[1]*lightColor[1]*specularFactor) * attenuation,
		(matDiffuse[2]*lightColor[2]*diffuseFactor + matSpecular[2]*lightColor[2]*specularFactor) * attenuation,
	}
}

func fragShader(u float32, v float32, col vec4.T, norm vec3.T, fragPos vec4.T, s *api.FragmentShader) vec4.T {
	ctxAny, _ := s.GetUniform("ctx")
	ctx := ctxAny.(*ShaderContext)

	var texColor vec4.T
	if ctx.Texture != nil {

		if len(ctx.Texture.Mipmaps) != 0 {
			lDir := vec3.T{
				ctx.ViewPos[0] - fragPos[0],
				ctx.ViewPos[1] - fragPos[1],
				ctx.ViewPos[2] - fragPos[2],
			}
			distance := float32(math.Sqrt(float64(lDir[0]*lDir[0] + lDir[1]*lDir[1] + lDir[2]*lDir[2])))

			texColor = ctx.Texture.SampleLod(u, v, distance)
		} else {
			texColor = ctx.Texture.Sample(u, v)
		}
	} else {
		texColor = ctx.Color //col
	}

	texR := texColor[0] / 255.0
	texG := texColor[1] / 255.0
	texB := texColor[2] / 255.0
	alpha := texColor[3] / 255.0

	if ctx.IsStraight {
		resultR := texR
		resultG := texG
		resultB := texB

		if ctx.IsSkybox {
			dl := ctx.Lights.Directional
			lightDir := vec3.T{-dl.Direction[0], -dl.Direction[1], -dl.Direction[2]}
			lenDL := float32(math.Sqrt(float64(lightDir[0]*lightDir[0] + lightDir[1]*lightDir[1] + lightDir[2]*lightDir[2])))
			if lenDL > 0 {
				lightDir[0] /= lenDL
				lightDir[1] /= lenDL
				lightDir[2] /= lenDL
			}

			viewDir := vec3.T{ctx.ViewPos[0] - fragPos[0], ctx.ViewPos[1] - fragPos[1], ctx.ViewPos[2] - fragPos[2]}
			lenV := float32(math.Sqrt(float64(viewDir[0]*viewDir[0] + viewDir[1]*viewDir[1] + viewDir[2]*viewDir[2])))
			if lenV > 0 {
				viewDir[0] /= lenV
				viewDir[1] /= lenV
				viewDir[2] /= lenV
			}

			dotViewLight := viewDir[0]*-lightDir[0] + viewDir[1]*-lightDir[1] + viewDir[2]*-lightDir[2]

			if dotViewLight > 0.96 {
				isBlocked := false
				if ctx.HasDirShadow {
					shadowVal := calculateShadow(fragPos, ctx.DirLightSpaceMatrix, ctx.DirShadowDepth, ctx.DirShadowWidth, ctx.DirShadowHeight)
					if shadowVal > 0.5 {
						isBlocked = true
					}
				}

				if !isBlocked {
					flareIntensity := float32(math.Pow(float64(dotViewLight), 128.0))
					resultR += dl.Color[0] * flareIntensity
					resultG += dl.Color[1] * flareIntensity
					resultB += dl.Color[2] * flareIntensity
				}
			}
		}

		return vec4.T{resultR, resultG, resultB, alpha}
	}

	lenN := float32(math.Sqrt(float64(norm[0]*norm[0] + norm[1]*norm[1] + norm[2]*norm[2])))
	if lenN > 0 {
		norm[0] /= lenN
		norm[1] /= lenN
		norm[2] /= lenN
	}

	viewDir := vec3.T{ctx.ViewPos[0] - fragPos[0], ctx.ViewPos[1] - fragPos[1], ctx.ViewPos[2] - fragPos[2]}
	lenV := float32(math.Sqrt(float64(viewDir[0]*viewDir[0] + viewDir[1]*viewDir[1] + viewDir[2]*viewDir[2])))
	if lenV > 0 {
		viewDir[0] /= lenV
		viewDir[1] /= lenV
		viewDir[2] /= lenV
	}

	// fogFactor := (lenV - ctx.Fog.Start) / (ctx.Fog.End - ctx.Fog.Start)
	fogFactor := float32(0)
	if ctx.Fog.Density != 0 {
		fogFactor = 1.0 - float32(math.Exp(-float64(lenV*ctx.Fog.Density)))
	}

	if fogFactor < 0 {
		fogFactor = 0
	}
	if fogFactor > 1 {
		fogFactor = 1
	}

	// 1. Ambient
	resultR := texR * ctx.Material.Ambient[0]
	resultG := texG * ctx.Material.Ambient[1]
	resultB := texB * ctx.Material.Ambient[2]

	baseDiffuseR := texR * ctx.Material.Diffuse[0]
	baseDiffuseG := texG * ctx.Material.Diffuse[1]
	baseDiffuseB := texB * ctx.Material.Diffuse[2]

	// 2. Directional Light
	dl := ctx.Lights.Directional
	lightDir := vec3.T{-dl.Direction[0], -dl.Direction[1], -dl.Direction[2]}
	lenDL := float32(math.Sqrt(float64(lightDir[0]*lightDir[0] + lightDir[1]*lightDir[1] + lightDir[2]*lightDir[2])))
	if lenDL > 0 {
		lightDir[0] /= lenDL
		lightDir[1] /= lenDL
		lightDir[2] /= lenDL
	}

	dirShadow := float32(0.0)
	if ctx.HasDirShadow {
		dirShadow = calculateShadow(fragPos, ctx.DirLightSpaceMatrix, ctx.DirShadowDepth, ctx.DirShadowWidth, ctx.DirShadowHeight)
	}

	contrib := calculateLightContribution(
		lightDir, 0.0, dl.Color, 1.0, 1.0, 0.0, 0.0,
		norm, viewDir,
		vec3.T{baseDiffuseR, baseDiffuseG, baseDiffuseB},
		ctx.Material.Specular, ctx.Material.Shininess,
	)

	shadowFactor := 1.0 - dirShadow
	resultR += contrib[0] * shadowFactor
	resultG += contrib[1] * shadowFactor
	resultB += contrib[2] * shadowFactor

	// 3. Point Lights
	for _, pl := range ctx.Lights.PointLights {
		lDir := vec3.T{pl.Position[0] - fragPos[0], pl.Position[1] - fragPos[1], pl.Position[2] - fragPos[2]}
		distance := float32(math.Sqrt(float64(lDir[0]*lDir[0] + lDir[1]*lDir[1] + lDir[2]*lDir[2])))
		if distance > 0 {
			lDir[0] /= distance
			lDir[1] /= distance
			lDir[2] /= distance
		}

		contrib := calculateLightContribution(
			lDir, distance, pl.Color, pl.Intensity, pl.Constant, pl.Linear, pl.Quadratic,
			norm, viewDir,
			vec3.T{baseDiffuseR, baseDiffuseG, baseDiffuseB},
			ctx.Material.Specular, ctx.Material.Shininess,
		)

		resultR += contrib[0]
		resultG += contrib[1]
		resultB += contrib[2]
	}

	// 4. Spot Lights
	for _, sl := range ctx.Lights.SpotLights {
		lDir := vec3.T{sl.Position[0] - fragPos[0], sl.Position[1] - fragPos[1], sl.Position[2] - fragPos[2]}
		distance := float32(math.Sqrt(float64(lDir[0]*lDir[0] + lDir[1]*lDir[1] + lDir[2]*lDir[2])))
		if distance > 0 {
			lDir[0] /= distance
			lDir[1] /= distance
			lDir[2] /= distance
		}

		theta := lDir[0]*(-sl.Direction[0]) + lDir[1]*(-sl.Direction[1]) + lDir[2]*(-sl.Direction[2])

		if theta > sl.CosCutOff {
			epsilon := sl.CosCutOff - sl.OuterCos
			spotIntensity := float32(1.0)
			if epsilon > 0 {
				spotIntensity = (theta - sl.OuterCos) / epsilon
				if spotIntensity > 1.0 {
					spotIntensity = 1.0
				}
				if spotIntensity < 0.0 {
					spotIntensity = 0.0
				}
			}

			spotShadow := float32(0.0)
			if ctx.HasSpotShadow {
				spotShadow = calculateShadow(fragPos, ctx.SpotLightSpaceMatrix, ctx.SpotShadowDepth, ctx.SpotShadowWidth, ctx.SpotShadowHeight)
			}

			contrib := calculateLightContribution(
				lDir, distance, sl.Color, sl.Intensity*spotIntensity, sl.Constant, sl.Linear, sl.Quadratic,
				norm, viewDir,
				vec3.T{baseDiffuseR, baseDiffuseG, baseDiffuseB},
				ctx.Material.Specular, ctx.Material.Shininess,
			)

			spotShadowFactor := 1.0 - spotShadow
			resultR += contrib[0] * spotShadowFactor
			resultG += contrib[1] * spotShadowFactor
			resultB += contrib[2] * spotShadowFactor
		}
	}

	resultR = resultR*(1.0-fogFactor) + ctx.Fog.Color[0]*fogFactor
	resultG = resultG*(1.0-fogFactor) + ctx.Fog.Color[1]*fogFactor
	resultB = resultB*(1.0-fogFactor) + ctx.Fog.Color[2]*fogFactor

	// Clamp
	if resultR > 1.0 {
		resultR = 1.0
	}
	if resultG > 1.0 {
		resultG = 1.0
	}
	if resultB > 1.0 {
		resultB = 1.0
	}

	return vec4.T{resultR, resultG, resultB, alpha}
}

func NewBaseFragmentShader() *api.FragmentShader {
	return api.NewFragShader(fragShader)
}
