package main

import "time"

type AttackBox struct {
	PosX, PosY float64
	W, H       float64
	LifeMS     int
	Created    time.Time
	Damage     int
	Hit        bool // ensure each box only counts once
}

func (a AttackBox) Expired(now time.Time) bool {
	return now.Sub(a.Created) > time.Duration(a.LifeMS)*time.Millisecond
}

// Build a short-lived hurtbox in front of the vampire based on direction.
func BuildAttackBox(playerX, playerY float64, direction int) AttackBox {
	// hitbox sized relative to 3x sprite scale
	w, h := 24.0*3, 20.0*3
	ax, ay := playerX, playerY

	switch direction {
	case DirRight:
		ax = playerX + spriteW
		ay = playerY + (spriteH-h)/2
	case DirLeft:
		ax = playerX - w
		ay = playerY + (spriteH-h)/2
	case DirUp:
		ax = playerX + (spriteW-w)/2
		ay = playerY - h
	case DirDown:
		ax = playerX + (spriteW-w)/2
		ay = playerY + spriteH
	}

	return AttackBox{PosX: ax, PosY: ay, W: w, H: h, LifeMS: 140, Damage: 1}
}

func ApplyAttackToPenguin(a AttackBox, p *PenguinEnemy, now time.Time) bool {
	if a.Expired(now) || a.Hit {
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
