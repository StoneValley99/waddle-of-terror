package main

import "math/rand"

const (
	spriteW = 32.0 * 3 // 96x96 approximate collision box
	spriteH = 32.0 * 3
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
