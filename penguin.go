package main

import "time"

type PenguinEnemy struct {
	x, y                   float64
	frame                  int
	frameDelay             int
	directionX, directionY int
	moveTimer              int
	visible                bool
	teleportTimer          int
	scareInterval          int

	// combat/AI
	Health      int
	invulnUntil time.Time
	mode        int     // ModeChase or ModeFlee
	speed       float64 // pixels per tick (e.g., 2.5 when chasing)
}
