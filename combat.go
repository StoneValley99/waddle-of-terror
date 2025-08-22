package main

import (
	"math"
	"time"
)

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
// Tighter size + slightly shorter life so you can't hit from far away.
func BuildAttackBox(playerX, playerY float64, direction int) AttackBox {
	w, h := 20.0*3, 16.0*3 // was 24x20 (tighter now)
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

	return AttackBox{PosX: ax, PosY: ay, W: w, H: h, LifeMS: 120, Damage: 1}
}

func ApplyAttackToPenguin(a AttackBox, p *PenguinEnemy, now time.Time) bool {
	if a.Expired(now) || a.Hit {
		return false
	}
	if now.Before(p.invulnUntil) || p.State == PengDeath {
		return false
	}

	px, py, pw, ph := PenguinCollider(p.x, p.y)
	if !RectsOverlap(a.PosX, a.PosY, a.W, a.H, px, py, pw, ph) {
		return false
	}

	// Register the hit
	p.invulnUntil = now.Add(150 * time.Millisecond)

	// Damage (non-lethal unless HP runs out)
	if p.Health > 0 {
		p.Health -= max(1, a.Damage)
	}

	// Small knockback away from the attack center (feels good!)
	ax := a.PosX + a.W/2
	ay := a.PosY + a.H/2
	cx := px + pw/2
	cy := py + ph/2
	dx := cx - ax
	dy := cy - ay
	if L := math.Hypot(dx, dy); L > 0 {
		dx /= L
		dy /= L
		knock := 8.0
		p.x += dx * knock
		p.y += dy * knock
	}

	if p.Health <= 0 {
		// Lethal → play death anim, stop moving; round ends after anim in Update()
		p.State = PengDeath
		p.frame = 0
		p.deathFrameDelay = 0
		p.speed = 0
		// keep p.visible = true so the death anim actually shows
	} else {
		// Non-lethal → flee
		p.mode = ModeFlee
		if p.speed < 3.2 {
			p.speed = 3.2
		}
		// Visual state is driven by overlap elsewhere; this ensures it doesn't stay in Attack
		if p.State != PengDeath {
			p.State = PengIdle
		}
	}

	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
