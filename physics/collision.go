package physics

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// resolveAABBs separates two AABBs along the axis of minimum penetration
// and zeroes the velocity component along that axis.
//
// This is the standard "minimum translation vector" approach for AABBs.
// It gives natural-looking behavior: a body falling onto the floor stops
// on Y but can still slide along X/Z.
func resolveAABBs(a, b *Body) {
	// Overlap on each axis. Positive = overlapping.
	ox := (a.Half[0] + b.Half[0]) - abs(a.Pos[0]-b.Pos[0])
	oy := (a.Half[1] + b.Half[1]) - abs(a.Pos[1]-b.Pos[1])
	oz := (a.Half[2] + b.Half[2]) - abs(a.Pos[2]-b.Pos[2])
	if ox <= 0 || oy <= 0 || oz <= 0 {
		return // not actually overlapping (shouldn't happen, coldet already checked)
	}

	// Pick the axis with smallest overlap = cheapest push-out.
	axis, overlap := 0, ox
	if oy < overlap {
		axis, overlap = 1, oy
	}
	if oz < overlap {
		axis, overlap = 2, oz
	}

	// Sign: which direction should `a` be pushed.
	sign := float32(1)
	if a.Pos[axis] < b.Pos[axis] {
		sign = -1
	}

	// Distribute push by inverse mass.
	invA, invB := a.invMass(), b.invMass()
	sum := invA + invB
	if sum == 0 {
		return
	}
	a.Pos[axis] += sign * overlap * (invA / sum)
	b.Pos[axis] -= sign * overlap * (invB / sum)

	// Stop velocity along the collision normal.
	// "Stop" here means: cancel the relative velocity component that is
	// pushing the bodies into each other. The tangent component is kept,
	// so bodies slide along walls/floors.
	rel := a.Vel[axis] - b.Vel[axis]
	if sign*rel < 0 { // moving towards each other along this axis
		a.Vel[axis] -= rel * (invA / sum)
		b.Vel[axis] += rel * (invB / sum)
	}
}

// resolveSpheres separates two spheres along the line connecting their centers
// and zeroes the velocity along that normal.
func resolveSpheres(a, b *Body) {
	pa := mgl32.Vec3{a.Pos[0], a.Pos[1], a.Pos[2]}
	pb := mgl32.Vec3{b.Pos[0], b.Pos[1], b.Pos[2]}
	delta := pa.Sub(pb)
	dist := delta.Len()

	// Spheres are concentric — pick any axis (Y is usually a safe fallback
	// for "one on top of the other" gameplay cases).
	var n mgl32.Vec3
	if dist > 1e-6 {
		n = delta.Mul(1.0 / dist)
	} else {
		n = mgl32.Vec3{0, 1, 0}
	}

	overlap := a.Radius + b.Radius - dist
	if overlap <= 0 {
		return
	}

	invA, invB := a.invMass(), b.invMass()
	sum := invA + invB
	if sum == 0 {
		return
	}

	// Push `a` along +n, `b` along -n.
	push := n.Mul(overlap)
	a.Pos[0] += push.X() * (invA / sum)
	a.Pos[1] += push.Y() * (invA / sum)
	a.Pos[2] += push.Z() * (invA / sum)
	b.Pos[0] -= push.X() * (invB / sum)
	b.Pos[1] -= push.Y() * (invB / sum)
	b.Pos[2] -= push.Z() * (invB / sum)

	// Cancel the normal component of the relative velocity (no bounce).
	va := mgl32.Vec3{a.Vel[0], a.Vel[1], a.Vel[2]}
	vb := mgl32.Vec3{b.Vel[0], b.Vel[1], b.Vel[2]}
	rel := va.Sub(vb)
	vn := rel.Dot(n)
	if vn < 0 { // approaching
		// Remove exactly the approaching component.
		imp := n.Mul(vn)
		a.Vel[0] -= imp.X() * (invA / sum)
		a.Vel[1] -= imp.Y() * (invA / sum)
		a.Vel[2] -= imp.Z() * (invA / sum)
		b.Vel[0] += imp.X() * (invB / sum)
		b.Vel[1] += imp.Y() * (invB / sum)
		b.Vel[2] += imp.Z() * (invB / sum)
	}
}

// resolveSphereAABB separates a sphere from an AABB.
// The normal is the direction from the closest point on the AABB
// to the sphere center. If the sphere center is inside the AABB,
// we use the axis of least penetration to the nearest face.
func resolveSphereAABB(s, bx *Body) {
	// Closest point on AABB to sphere center.
	cx := clampF(s.Pos[0], bx.Pos[0]-bx.Half[0], bx.Pos[0]+bx.Half[0])
	cy := clampF(s.Pos[1], bx.Pos[1]-bx.Half[1], bx.Pos[1]+bx.Half[1])
	cz := clampF(s.Pos[2], bx.Pos[2]-bx.Half[2], bx.Pos[2]+bx.Half[2])

	dx := s.Pos[0] - cx
	dy := s.Pos[1] - cy
	dz := s.Pos[2] - cz
	distSq := dx*dx + dy*dy + dz*dz
	r := s.Radius

	// Sphere center inside the box: pick the nearest face to push out.
	if distSq == 0 {
		// distance to each face
		dpx := bx.Half[0] - abs(s.Pos[0]-bx.Pos[0])
		dpy := bx.Half[1] - abs(s.Pos[1]-bx.Pos[1])
		dpz := bx.Half[2] - abs(s.Pos[2]-bx.Pos[2])
		axis, d := 0, dpx
		if dpy < d {
			axis, d = 1, dpy
		}
		if dpz < d {
			axis, d = 2, dpz
		}
		// normal points from box face towards sphere
		n := [3]float32{}
		if s.Pos[axis] >= bx.Pos[axis] {
			n[axis] = 1
		} else {
			n[axis] = -1
		}
		overlap := r + d
		pushAlongNormal(s, bx, n, overlap)
		return
	}

	dist := sqrtF(distSq)
	if dist >= r {
		return // not colliding (coldet should already have caught this)
	}
	// Normal from closest point to sphere center.
	inv := 1.0 / dist
	n := [3]float32{dx * inv, dy * inv, dz * inv}
	overlap := r - dist
	pushAlongNormal(s, bx, n, overlap)
}

// pushAlongNormal moves the sphere along +n by `overlap` (split by inverse mass)
// and cancels the approaching velocity component along n.
// The AABB is treated as the other body — it can be dynamic too.
func pushAlongNormal(s, bx *Body, n [3]float32, overlap float32) {
	invA, invB := s.invMass(), bx.invMass()
	sum := invA + invB
	if sum == 0 {
		return
	}
	s.Pos[0] += n[0] * overlap * (invA / sum)
	s.Pos[1] += n[1] * overlap * (invA / sum)
	s.Pos[2] += n[2] * overlap * (invA / sum)
	bx.Pos[0] -= n[0] * overlap * (invB / sum)
	bx.Pos[1] -= n[1] * overlap * (invB / sum)
	bx.Pos[2] -= n[2] * overlap * (invB / sum)

	// Cancel approaching velocity along n.
	relX := s.Vel[0] - bx.Vel[0]
	relY := s.Vel[1] - bx.Vel[1]
	relZ := s.Vel[2] - bx.Vel[2]
	vn := relX*n[0] + relY*n[1] + relZ*n[2]
	if vn < 0 {
		imp := vn / sum
		s.Vel[0] -= n[0] * imp * invA
		s.Vel[1] -= n[1] * imp * invA
		s.Vel[2] -= n[2] * imp * invA
		bx.Vel[0] += n[0] * imp * invB
		bx.Vel[1] += n[1] * imp * invB
		bx.Vel[2] += n[2] * imp * invB
	}
}

// ---- small float32 helpers (avoid float64 roundtrips in hot path) ----

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func clampF(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sqrtF(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}
