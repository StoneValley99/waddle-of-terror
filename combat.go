package main

import (
	"math"
	"time"
)

type AttackBox struct {
	X, Y float64
	W, H float64

	LifeMS  int
	Created time.Time
	Damage  int
	Hit     bool // ensure each box only counts once
}

func (a AttackBox) Expired(now time.Time) bool {
	return now.Sub(a.Created) > time.Duration(a.LifeMS)*time.Millisecond
}

// Build a short-lived hurtbox in front of the vampire based on direction.
// Tighter size + slightly shorter life so you can't hit from far away.
func BuildAttackBox(x, y float64, dir int) AttackBox {
	sizeW := spriteW * 1 // wider than vampire
	sizeH := spriteH * 1 // taller than vampire
	offset := 60.0       // how far in front of vampire the box extends

	var cx, cy float64

	switch dir {
	case DirUp:
		cx = x + (spriteW-sizeW)/2
		cy = y - offset
	case DirDown:
		cx = x + (spriteW-sizeW)/2
		cy = y + spriteH - sizeH + offset
	case DirLeft:
		cx = x - offset
		cy = y + (spriteH-sizeH)/2
	case DirRight:
		cx = x + spriteW - sizeW + offset
		cy = y + (spriteH-sizeH)/2
	}

	return AttackBox{
		X:       cx,
		Y:       cy,
		W:       sizeW,
		H:       sizeH,
		Created: time.Now(),
		LifeMS:  150, // lasts ~0.15s
		Damage:  1,
	}
}

func ApplyAttackToPenguin(a AttackBox, p *PenguinEnemy, now time.Time) bool {
	if a.Expired(now) || a.Hit {
		return false
	}
	if now.Before(p.invulnUntil) || p.State == PengDeath {
		return false
	}

	px, py, pw, ph := PenguinCollider(p.x, p.y)
	if !RectsOverlap(a.X, a.Y, a.W, a.H, px, py, pw, ph) {
		return false
	}

	// Register the hit
	p.invulnUntil = now.Add(150 * time.Millisecond)

	// Damage (non-lethal unless HP runs out)
	if p.Health > 0 {
		p.Health -= max(1, a.Damage)
	}

	// Small knockback away from the attack center (feels good!)
	ax := a.X + a.W/2
	ay := a.Y + a.H/2
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
