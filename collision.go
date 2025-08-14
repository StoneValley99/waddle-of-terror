package main

func RectsOverlap(ax, ay, aw, ah, bx, by, bw, bh float64) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}

// Push (mx,my) out of (sx,sy) along the smallest overlap axis.
func ResolveDynamicVsSolid(mx, my *float64, mw, mh float64, sx, sy, sw, sh float64) {
	ax1, ay1 := *mx, *my
	ax2, ay2 := ax1+mw, ay1+mh

	bx1, by1 := sx, sy
	bx2, by2 := bx1+sw, by1+sh

	overLeft := bx2 - ax1  // push left  (positive moves actor left)
	overRight := ax2 - bx1 // push right (positive moves actor right)
	overUp := by2 - ay1    // push up
	overDown := ay2 - by1  // push down

	// pick smallest absolute overlap
	minX := overLeft
	if overRight < minX {
		minX = -overRight
	}
	minY := overUp
	if overDown < minY {
		minY = -overDown
	}

	if abs(minX) < abs(minY) {
		*mx += minX
	} else {
		*my += minY
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
