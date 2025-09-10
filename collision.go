package main

// --- Colliders (tight AABBs), tuned smaller than full sprite ---
// These are centered rectangles smaller than the drawn sprite so collisions feel "fairer".
func VampCollider(x, y float64) (cx, cy, cw, ch float64) {
	cw = spriteW * 0.45 // about half the width
	ch = spriteH * 0.5  // about half the height (torso/legs only)

	// Anchor to bottom center of sprite
	cx = x + (spriteW-cw)/2 + 30
	cy = y + spriteH - ch // push box to bottom half of sprite

	return
}

func PenguinCollider(x, y float64) (cx, cy, cw, ch float64) {
	cw = spriteW * 0.6 // penguin is small round
	ch = spriteH * 0.6
	cx = x + (spriteW-cw)/2
	cy = y + (spriteH-ch)/2
	return
}

// --- Rectangle overlap check ---
func RectsOverlap(ax, ay, aw, ah, bx, by, bw, bh float64) bool {
	return ax < bx+bw &&
		ax+aw > bx &&
		ay < by+bh &&
		ay+ah > by
}

// --- Collision resolution (simple pushback) ---
// Shoves dynamic object (dx,dy) away from solid (sx,sy).
func ResolveDynamicVsSolid(dx, dy *float64, dw, dh, sx, sy, sw, sh float64) {
	// Basic overlap resolution using minimum translation vector
	if !RectsOverlap(*dx, *dy, dw, dh, sx, sy, sw, sh) {
		return
	}

	// Distances to edges
	leftPenetration := (*dx + dw) - sx
	rightPenetration := (sx + sw) - *dx
	topPenetration := (*dy + dh) - sy
	bottomPenetration := (sy + sh) - *dy

	// Choose smallest axis to resolve
	minPen := leftPenetration
	axis := "left"

	if rightPenetration < minPen {
		minPen = rightPenetration
		axis = "right"
	}
	if topPenetration < minPen {
		minPen = topPenetration
		axis = "top"
	}
	if bottomPenetration < minPen {
		minPen = bottomPenetration
		axis = "bottom"
	}

	switch axis {
	case "left":
		*dx -= minPen
	case "right":
		*dx += minPen
	case "top":
		*dy -= minPen
	case "bottom":
		*dy += minPen
	}
}
