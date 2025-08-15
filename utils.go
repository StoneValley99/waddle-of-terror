package main

import "math/rand"

const (
	spriteW = 32.0 * 3 // 96x96
	spriteH = 32.0 * 3

	DirDown  = 0
	DirUp    = 1
	DirLeft  = 2
	DirRight = 3

	ModeChase = 0
	ModeFlee  = 1
)

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// randInt returns integer in [min,max], inclusive.
func randInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}

func signf(v float64) float64 {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

// --- Tight AABBs (centered), tuned smaller than the full sprite ---
// Tweak these scale factors if you want even tighter/looser boxes.
func VampCollider(x, y float64) (cx, cy, cw, ch float64) {
	cw = spriteW * 0.48 // ~46px of 96
	ch = spriteH * 0.58 // ~56px of 96
	cx = x + (spriteW-cw)/2
	// Slightly lower to match body mass (down from center ~10% of spriteH)
	cy = y + (spriteH-ch)*0.7
	return
}

func PenguinCollider(x, y float64) (cx, cy, cw, ch float64) {
	cw = spriteW * 0.42 // ~40px of 96
	ch = spriteH * 0.50 // ~48px of 96
	cx = x + (spriteW-cw)/2
	cy = y + (spriteH-ch)/2
	return
}
