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

	Health      int
	invulnUntil time.Time
}
