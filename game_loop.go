package main

import (
	"fmt"
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	// assets
	spriteSheet         *ebiten.Image
	stabbingSpriteSheet *ebiten.Image
	deathSpriteSheet    *ebiten.Image
	enemyPengSheet      *ebiten.Image
	background          *ebiten.Image

	// player (vampire)
	frame              int
	x, y               float64
	direction          int // DirDown/DirUp/DirLeft/DirRight
	framesPerDirection int
	frameDelay         int
	idle               bool
	stabbing           bool // latched visual state

	// death animation
	vampireDead       bool
	deathFrame        int
	deathFrameDelay   int
	deathFramesPerDir int

	// camera
	cameraX, cameraY float64

	// enemy
	penguin PenguinEnemy

	// combat
	attacks         []AttackBox
	nextAttackAt    time.Time
	attackCooldown  time.Duration
	attackAnimTicks int // latch: remaining ticks of stab anim
	hitCount        int // successful strikes counter
}

func (g *Game) Update() error {
	g.idle = true
	now := time.Now()

	// --- Input (disabled when dead) ---
	if !g.vampireDead {
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			g.y -= 2
			g.direction = DirUp
			g.idle = false
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			g.y += 2
			g.direction = DirDown
			g.idle = false
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
			g.x -= 2
			g.direction = DirLeft
			g.idle = false
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
			g.x += 2
			g.direction = DirRight
			g.idle = false
		}

		// Attack trigger (tap once, plays out)
		if inpututil.IsKeyJustPressed(ebiten.KeyX) && now.After(g.nextAttackAt) {
			box := BuildAttackBox(g.x, g.y, g.direction)
			box.Created = now
			g.attacks = append(g.attacks, box)
			g.nextAttackAt = now.Add(g.attackCooldown)
			g.attackAnimTicks = 12 * 5 // latch whole 12-frame anim @5 ticks each
		}
	}

	// --- Stab animation latch ---
	g.stabbing = g.attackAnimTicks > 0 && !g.vampireDead
	if g.attackAnimTicks > 0 {
		g.attackAnimTicks--
		g.idle = false
	}

	// --- Animations (death/walk/stab) ---
	if g.vampireDead {
		g.deathFrameDelay++
		if g.deathFrameDelay >= 5 {
			if g.deathFrame < g.deathFramesPerDir-1 {
				g.deathFrame++
			}
			g.deathFrameDelay = 0
		}
	} else if g.stabbing {
		g.frameDelay++
		if g.frameDelay >= 5 {
			g.frame = (g.frame + 1) % 12
			g.frameDelay = 0
		}
	} else if !g.idle {
		g.frameDelay++
		if g.frameDelay >= 5 {
			g.frame = (g.frame + 1) % g.framesPerDirection
			g.frameDelay = 0
		}
	} else {
		g.frame = 0
	}

	// --- Camera follow ---
	g.cameraX = clamp(g.x-320, 0, float64(g.background.Bounds().Dx())-640)
	g.cameraY = clamp(g.y-240, 0, float64(g.background.Bounds().Dy())-480)

	// --- Penguin AI & movement ---
	mapWidth := float64(g.background.Bounds().Dx())
	mapHeight := float64(g.background.Bounds().Dy())
	g.updatePenguinAI(mapWidth, mapHeight)

	// --- Clamp player ---
	g.x = clamp(g.x, 0, mapWidth-spriteW)
	g.y = clamp(g.y, 0, mapHeight-spriteH)

	// --- Tight colliders for death/hit detection ---
	vx, vy, vw, vh := VampCollider(g.x, g.y)
	px, py, pw, ph := PenguinCollider(g.penguin.x, g.penguin.y)

	// Vampire dies if tight colliders overlap while penguin is chasing
	if !g.vampireDead && g.penguin.visible && g.penguin.mode == ModeChase &&
		RectsOverlap(vx, vy, vw, vh, px, py, pw, ph) {
		g.vampireDead = true
		g.deathFrame, g.deathFrameDelay, g.attackAnimTicks = 0, 0, 0
	}

	// Body push = shove penguin ONLY (keeps “run-away” vibe)
	// We check overlap with tight colliders, but push using sprite-sized boxes for simplicity.
	if g.penguin.visible && RectsOverlap(vx, vy, vw, vh, px, py, pw, ph) {
		ResolveDynamicVsSolid(&g.penguin.x, &g.penguin.y, spriteW, spriteH, g.x, g.y, spriteW, spriteH)
	}

	// --- Apply attacks (switch to flee on hit) ---
	if g.penguin.visible && g.penguin.Health > 0 && !g.vampireDead {
		for i := range g.attacks {
			a := &g.attacks[i]
			if !a.Expired(now) && !a.Hit && ApplyAttackToPenguin(*a, &g.penguin, now) {
				a.Hit = true
				g.hitCount++
				// Penguin flees after getting hit
				g.penguin.mode = ModeFlee
				if g.penguin.speed == 0 {
					g.penguin.speed = 2.5
				}
			}
		}
	}

	// Prune expired attack boxes
	dst := g.attacks[:0]
	for _, a := range g.attacks {
		if !a.Expired(now) {
			dst = append(dst, a)
		}
	}
	g.attacks = dst

	// Hide penguin on death (placeholder)
	if g.penguin.Health <= 0 {
		g.penguin.visible = false
	}

	return nil
}

func (g *Game) updatePenguinAI(mapWidth, mapHeight float64) {
	// Safety: ensure a nonzero speed
	if g.penguin.speed == 0 {
		if g.penguin.mode == ModeChase {
			g.penguin.speed = 2.5
		} else {
			g.penguin.speed = 2.5
		}
	}

	// Scare/teleport cycle
	g.penguin.teleportTimer++
	if g.penguin.visible {
		if g.penguin.teleportTimer >= g.penguin.scareInterval {
			g.penguin.visible = false
			g.penguin.teleportTimer = 0
		}
	} else {
		if g.penguin.teleportTimer >= 60 { // ~1s
			g.penguin.visible = true
			g.penguin.teleportTimer = 0

			// Spawn just off-screen around the player
			offset := 100.0
			switch randInt(0, 3) {
			case 0:
				g.penguin.x, g.penguin.y = g.x, g.y-240-offset
			case 1:
				g.penguin.x, g.penguin.y = g.x, g.y+240+offset
			case 2:
				g.penguin.x, g.penguin.y = g.x-320-offset, g.y
			case 3:
				g.penguin.x, g.penguin.y = g.x+320+offset, g.y
			}

			// On spawn: go aggressive
			g.penguin.mode = ModeChase
			g.penguin.speed = 2.5
		}
	}

	// Movement/anim when visible
	if g.penguin.visible {
		// Simple 2-frame anim
		g.penguin.frameDelay++
		if g.penguin.frameDelay >= 10 {
			g.penguin.frame = (g.penguin.frame + 1) % 2
			g.penguin.frameDelay = 0
		}

		// Chase or flee
		var stepX, stepY float64
		if g.penguin.mode == ModeChase {
			stepX = signf(g.x-g.penguin.x) * g.penguin.speed
			stepY = signf(g.y-g.penguin.y) * g.penguin.speed
		} else { // ModeFlee
			stepX = signf(g.penguin.x-g.x) * g.penguin.speed
			stepY = signf(g.penguin.y-g.y) * g.penguin.speed
		}

		// (Optional) record step sign for future facing logic
		switch {
		case stepX > 0:
			g.penguin.directionX = 1
		case stepX < 0:
			g.penguin.directionX = -1
		default:
			g.penguin.directionX = 0
		}
		switch {
		case stepY > 0:
			g.penguin.directionY = 1
		case stepY < 0:
			g.penguin.directionY = -1
		default:
			g.penguin.directionY = 0
		}

		// Move & clamp
		g.penguin.x = clamp(g.penguin.x+stepX, 0, mapWidth-spriteW)
		g.penguin.y = clamp(g.penguin.y+stepY, 0, mapHeight-spriteH)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Background
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Translate(-g.cameraX, -g.cameraY)
	screen.DrawImage(g.background, bgOp)

	// Penguin (2-frame sheet)
	if g.penguin.visible {
		pw := g.enemyPengSheet.Bounds().Dx() / 2
		ph := g.enemyPengSheet.Bounds().Dy()
		src := image.Rect(g.penguin.frame*pw, 0, (g.penguin.frame+1)*pw, ph)
		img := g.enemyPengSheet.SubImage(src).(*ebiten.Image)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(3, 3)
		op.GeoM.Translate(g.penguin.x-g.cameraX, g.penguin.y-g.cameraY)
		screen.DrawImage(img, op)
	}

	// Vampire: death vs normal
	if g.vampireDead {
		sheet := g.deathSpriteSheet
		fw := sheet.Bounds().Dx() / g.deathFramesPerDir
		fh := sheet.Bounds().Dy() / 4
		srcX := g.deathFrame * fw
		srcY := g.direction * fh
		src := image.Rect(srcX, srcY, srcX+fw, srcY+fh)
		img := sheet.SubImage(src).(*ebiten.Image)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(3, 3)
		op.GeoM.Translate(g.x-g.cameraX, g.y-g.cameraY)
		screen.DrawImage(img, op)
	} else {
		// Choose sheet (stabbing vs walk/idle)
		var sheet *ebiten.Image
		var framesPerDir int
		if g.stabbing {
			sheet = g.stabbingSpriteSheet
			framesPerDir = 12
		} else {
			sheet = g.spriteSheet
			framesPerDir = g.framesPerDirection
		}
		fw := sheet.Bounds().Dx() / framesPerDir
		fh := sheet.Bounds().Dy() / 4
		srcX := (g.frame % framesPerDir) * fw
		srcY := g.direction * fh
		src := image.Rect(srcX, srcY, srcX+fw, srcY+fh)
		img := sheet.SubImage(src).(*ebiten.Image)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(3, 3)
		op.GeoM.Translate(g.x-g.cameraX, g.y-g.cameraY)
		screen.DrawImage(img, op)
	}

	// UI / debug
	modeText := "Chase"
	if g.penguin.mode == ModeFlee {
		modeText = "Flee"
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Hits: %d  Mode:%s  Speed:%.1f", g.hitCount, modeText, g.penguin.speed), 8, 8)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}
