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
