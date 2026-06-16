package phyapi

import "github.com/ungerik/go3d/vec3"

// Supports
//   - AABB vs AABB
//   - Sphere vs Sphere
//   - Sphere vs AABB
func resolvePair(a, b *PhysicsNode) {
	if a.Compound != nil || b.Compound != nil {
		ResolveCompound(a, b)
		return
	}

	switch {
	case a.Collider.Kind == ColliderAABB && b.Collider.Kind == ColliderAABB:
		ResolveAABBs(a, b)

	case a.Collider.Kind == ColliderSphere && b.Collider.Kind == ColliderSphere:
		ResolveSpheres(a, b)

	case a.Collider.Kind == ColliderSphere && b.Collider.Kind == ColliderAABB:
		ResolveSphereAABB(a, b)

	case a.Collider.Kind == ColliderAABB && b.Collider.Kind == ColliderSphere:
		ResolveSphereAABB(b, a)
	}
}

func ResolveAABBs(a, b *PhysicsNode) {
	ox := (a.Collider.Half[0] + b.Collider.Half[0]) - AbsF(a.Position[0]-b.Position[0])
	oy := (a.Collider.Half[1] + b.Collider.Half[1]) - AbsF(a.Position[1]-b.Position[1])
	oz := (a.Collider.Half[2] + b.Collider.Half[2]) - AbsF(a.Position[2]-b.Position[2])
	if ox <= 0 || oy <= 0 || oz <= 0 {
		return
	}

	axis, overlap := 0, ox
	if oy < overlap {
		axis, overlap = 1, oy
	}
	if oz < overlap {
		axis, overlap = 2, oz
	}

	sign := float32(1)
	if a.Position[axis] < b.Position[axis] {
		sign = -1
	}

	invA, invB := a.invMass(), b.invMass()
	sum := invA + invB
	if sum == 0 {
		return
	}

	a.Position[axis] += sign * overlap * (invA / sum)
	b.Position[axis] -= sign * overlap * (invB / sum)

	rel := a.Velocity[axis] - b.Velocity[axis]
	if sign*rel < 0 {
		a.Velocity[axis] -= rel * (invA / sum)
		b.Velocity[axis] += rel * (invB / sum)
	}

	RegisterContact(a, b, axis, sign)
}

func ResolveSpheres(a, b *PhysicsNode) {
	dx := a.Position[0] - b.Position[0]
	dy := a.Position[1] - b.Position[1]
	dz := a.Position[2] - b.Position[2]
	distSq := dx*dx + dy*dy + dz*dz

	var nx, ny, nz, dist float32
	if distSq > 1e-6 {
		dist = SqrtF(distSq)
		inv := 1.0 / dist
		nx, ny, nz = dx*inv, dy*inv, dz*inv
	} else {
		nx, ny, nz = 0, 1, 0
		dist = 0
	}

	overlap := a.Collider.Radius + b.Collider.Radius - dist
	if overlap <= 0 {
		return
	}

	invA, invB := a.invMass(), b.invMass()
	sum := invA + invB
	if sum == 0 {
		return
	}

	a.Position[0] += nx * overlap * (invA / sum)
	a.Position[1] += ny * overlap * (invA / sum)
	a.Position[2] += nz * overlap * (invA / sum)
	b.Position[0] -= nx * overlap * (invB / sum)
	b.Position[1] -= ny * overlap * (invB / sum)
	b.Position[2] -= nz * overlap * (invB / sum)

	relX := a.Velocity[0] - b.Velocity[0]
	relY := a.Velocity[1] - b.Velocity[1]
	relZ := a.Velocity[2] - b.Velocity[2]
	vn := relX*nx + relY*ny + relZ*nz
	if vn < 0 {
		imp := vn / sum
		a.Velocity[0] -= nx * imp * invA
		a.Velocity[1] -= ny * imp * invA
		a.Velocity[2] -= nz * imp * invA
		b.Velocity[0] += nx * imp * invB
		b.Velocity[1] += ny * imp * invB
		b.Velocity[2] += nz * imp * invB
	}

	RegisterContactNormal(a, b, nx, ny, nz)
}

func ResolveSphereAABB(s, bx *PhysicsNode) {
	cx := ClampF(s.Position[0], bx.Position[0]-bx.Collider.Half[0], bx.Position[0]+bx.Collider.Half[0])
	cy := ClampF(s.Position[1], bx.Position[1]-bx.Collider.Half[1], bx.Position[1]+bx.Collider.Half[1])
	cz := ClampF(s.Position[2], bx.Position[2]-bx.Collider.Half[2], bx.Position[2]+bx.Collider.Half[2])

	dx := s.Position[0] - cx
	dy := s.Position[1] - cy
	dz := s.Position[2] - cz
	distSq := dx*dx + dy*dy + dz*dz
	r := s.Collider.Radius

	if distSq == 0 {
		dpx := bx.Collider.Half[0] - AbsF(s.Position[0]-bx.Position[0])
		dpy := bx.Collider.Half[1] - AbsF(s.Position[1]-bx.Position[1])
		dpz := bx.Collider.Half[2] - AbsF(s.Position[2]-bx.Position[2])
		axis, d := 0, dpx
		if dpy < d {
			axis, d = 1, dpy
		}
		if dpz < d {
			axis, d = 2, dpz
		}
		var n vec3.T
		if s.Position[axis] >= bx.Position[axis] {
			n[axis] = 1
		} else {
			n[axis] = -1
		}
		PushAlongNormal(s, bx, n, r+d)
		RegisterContactNormal(s, bx, n[0], n[1], n[2])
		return
	}

	dist := SqrtF(distSq)
	if dist >= r {
		return
	}
	inv := 1.0 / dist
	n := vec3.T{dx * inv, dy * inv, dz * inv}
	PushAlongNormal(s, bx, n, r-dist)
	RegisterContactNormal(s, bx, n[0], n[1], n[2])
}

func ResolveCompound(a, b *PhysicsNode) {
	aItems := compoundItems(a)
	bItems := compoundItems(b)

	aPosOrig := a.Position
	bPosOrig := b.Position

	for _, ai := range aItems {
		for _, bi := range bItems {

			a.Position = ai.worldCenter
			a.Collider = ai.col
			b.Position = bi.worldCenter
			b.Collider = bi.col

			resolvePairSimple(a, b)

			aDelta := vec3.T{
				a.Position[0] - ai.worldCenter[0],
				a.Position[1] - ai.worldCenter[1],
				a.Position[2] - ai.worldCenter[2],
			}
			bDelta := vec3.T{
				b.Position[0] - bi.worldCenter[0],
				b.Position[1] - bi.worldCenter[1],
				b.Position[2] - bi.worldCenter[2],
			}

			aPosOrig[0] += aDelta[0]
			aPosOrig[1] += aDelta[1]
			aPosOrig[2] += aDelta[2]
			bPosOrig[0] += bDelta[0]
			bPosOrig[1] += bDelta[1]
			bPosOrig[2] += bDelta[2]
		}
	}

	a.Position = aPosOrig
	b.Position = bPosOrig

	a.Collider = Collider{Kind: ColliderNone}
	b.Collider = Collider{Kind: ColliderNone}
}

type compoundItem struct {
	worldCenter vec3.T
	col         Collider
}

func compoundItems(n *PhysicsNode) []compoundItem {
	if n.Compound == nil || len(n.Compound.Items) == 0 {
		return []compoundItem{{worldCenter: n.Position, col: n.Collider}}
	}
	out := make([]compoundItem, len(n.Compound.Items))
	for i, it := range n.Compound.Items {
		out[i] = compoundItem{
			worldCenter: vec3.T{
				n.Position[0] + it.Offset[0],
				n.Position[1] + it.Offset[1],
				n.Position[2] + it.Offset[2],
			},
			col: it.Collider,
		}
	}
	return out
}
func resolvePairSimple(a, b *PhysicsNode) {
	switch {
	case a.Collider.Kind == ColliderAABB && b.Collider.Kind == ColliderAABB:
		ResolveAABBs(a, b)
	case a.Collider.Kind == ColliderSphere && b.Collider.Kind == ColliderSphere:
		ResolveSpheres(a, b)
	case a.Collider.Kind == ColliderSphere && b.Collider.Kind == ColliderAABB:
		ResolveSphereAABB(a, b)
	case a.Collider.Kind == ColliderAABB && b.Collider.Kind == ColliderSphere:
		ResolveSphereAABB(b, a)
	}
}

func PushAlongNormal(s, bx *PhysicsNode, n vec3.T, overlap float32) {
	invA, invB := s.invMass(), bx.invMass()
	sum := invA + invB
	if sum == 0 {
		return
	}

	s.Position[0] += n[0] * overlap * (invA / sum)
	s.Position[1] += n[1] * overlap * (invA / sum)
	s.Position[2] += n[2] * overlap * (invA / sum)
	bx.Position[0] -= n[0] * overlap * (invB / sum)
	bx.Position[1] -= n[1] * overlap * (invB / sum)
	bx.Position[2] -= n[2] * overlap * (invB / sum)

	relX := s.Velocity[0] - bx.Velocity[0]
	relY := s.Velocity[1] - bx.Velocity[1]
	relZ := s.Velocity[2] - bx.Velocity[2]
	vn := relX*n[0] + relY*n[1] + relZ*n[2]
	if vn < 0 {
		imp := vn / sum
		s.Velocity[0] -= n[0] * imp * invA
		s.Velocity[1] -= n[1] * imp * invA
		s.Velocity[2] -= n[2] * imp * invA
		bx.Velocity[0] += n[0] * imp * invB
		bx.Velocity[1] += n[1] * imp * invB
		bx.Velocity[2] += n[2] * imp * invB
	}
}

func RegisterContact(a, b *PhysicsNode, axis int, sign float32) {
	var na, nb vec3.T
	na[axis] = sign
	nb[axis] = -sign
	a.ContactNormals = append(a.ContactNormals, na)
	b.ContactNormals = append(b.ContactNormals, nb)
	if na[1] > 0.5 {
		a.IsGrounded = true
	}
	if nb[1] > 0.5 {
		b.IsGrounded = true
	}
}

func RegisterContactNormal(a, b *PhysicsNode, nx, ny, nz float32) {
	na := vec3.T{nx, ny, nz}
	nb := vec3.T{-nx, -ny, -nz}
	a.ContactNormals = append(a.ContactNormals, na)
	b.ContactNormals = append(b.ContactNormals, nb)
	if ny > 0.5 {
		a.IsGrounded = true
	}
	if -ny > 0.5 {
		b.IsGrounded = true
	}
}

func applyFriction(nodes []*PhysicsNode, dt float32, gravity float32) {
	gX, gY, gZ := float32(0), gravity, float32(0)
	for _, n := range nodes {
		if n.IsStatic || n.Friction <= 0 || len(n.ContactNormals) == 0 {
			continue
		}
		for _, cn := range n.ContactNormals {
			vn := n.Velocity[0]*cn[0] + n.Velocity[1]*cn[1] + n.Velocity[2]*cn[2]
			tx := n.Velocity[0] - vn*cn[0]
			ty := n.Velocity[1] - vn*cn[1]
			tz := n.Velocity[2] - vn*cn[2]
			tLen := SqrtF(tx*tx + ty*ty + tz*tz)
			if tLen < 1e-6 {
				continue
			}

			gn := gX*cn[0] + gY*cn[1] + gZ*cn[2]
			if gn >= 0 {
				continue
			}

			decel := n.Friction * (-gn)
			deltaV := decel * dt
			if deltaV >= tLen {
				n.Velocity[0] -= tx
				n.Velocity[1] -= ty
				n.Velocity[2] -= tz
			} else {
				f := deltaV / tLen
				n.Velocity[0] -= tx * f
				n.Velocity[1] -= ty * f
				n.Velocity[2] -= tz * f
			}
		}
	}
}
