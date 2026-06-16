package phyapi

import "math"

func AbsF(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func ClampF(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func SqrtF(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

func MinF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func MaxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
