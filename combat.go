package main

import "time"

type AttackBox struct {
	PosX, PosY float64
	W, H       float64
	LifeMS     int
	Created    time.Time
	Damage     int
}

func (a AttackBox) Expired(now time.Time) bool {
	return now.Sub(a.Created) > time.Duration(a.LifeMS)*time.Millisecond
}

// Build a short-lived hurtbox in front of vampire based on facing.
func BuildAttackBox(playerX, playerY float64, direction int) AttackBox {
	w, h := 24.0*3, 20.0*3 // scaled like sprites (x3)
	ax, ay := playerX, playerY
	switch direction {
	case 2: // right
		ax = playerX + spriteW
		ay = playerY + (spriteH-h)/2
	case 1: // up
		ax = playerX + (spriteW-w)/2
		ay = playerY - h
	case 3: // left
		ax = playerX - w
		ay = playerY + (spriteH-h)/2
	case 0: // down
		ax = playerX + (spriteW-w)/2
		ay = playerY + spriteH
	}
	return AttackBox{PosX: ax, PosY: ay, W: w, H: h, LifeMS: 120, Damage: 1}
}

func ApplyAttackToPenguin(a AttackBox, p *PenguinEnemy, now time.Time) bool {
	if a.Expired(now) {
		return false
	}
	if now.Before(p.invulnUntil) {
		return false
	}
	if RectsOverlap(a.PosX, a.PosY, a.W, a.H, p.x, p.y, spriteW, spriteH) {
		p.Health -= a.Damage
		p.invulnUntil = now.Add(200 * time.Millisecond)
		return true
	}
	return false
}
