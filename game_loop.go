package main

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	// assets
	spriteSheet         *ebiten.Image
	stabbingSpriteSheet *ebiten.Image
	enemyPengSheet      *ebiten.Image
	background          *ebiten.Image

	// player (vampire)
	frame              int
	x, y               float64
	direction          int // 0: down, 1: left, 2: right, 3: up
	framesPerDirection int
	frameDelay         int
	idle               bool
	stabbing           bool

	// camera
	cameraX, cameraY float64

	// enemy
	penguin PenguinEnemy

	// combat
	attacks        []AttackBox
	nextAttackAt   time.Time
	attackCooldown time.Duration
}

func (g *Game) Update() error {
	g.idle = true
	g.stabbing = false

	// --- input: movement ---
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.y -= 2
		g.direction = 1
		g.idle = false
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.y += 2
		g.direction = 0
		g.idle = false
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.x -= 2
		g.direction = 2
		g.idle = false
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.x += 2
		g.direction = 3
		g.idle = false
	}
	if ebiten.IsKeyPressed(ebiten.KeyX) {
		g.stabbing = true
		g.idle = false
	}

	// --- animation (player) ---
	if !g.idle && !g.stabbing {
		g.frameDelay++
		if g.frameDelay >= 5 {
			g.frame = (g.frame + 1) % g.framesPerDirection
			g.frameDelay = 0
		}
	} else if g.stabbing {
		g.frameDelay++
		if g.frameDelay >= 5 {
			g.frame = (g.frame + 1) % 12
			g.frameDelay = 0
		}
	} else {
		g.frame = 0
	}

	// --- camera follow ---
	g.cameraX = g.x - 320
	g.cameraY = g.y - 240
	screenWidth := 640.0
	screenHeight := 480.0
	mapWidth := float64(g.background.Bounds().Dx())
	mapHeight := float64(g.background.Bounds().Dy())
	g.cameraX = clamp(g.cameraX, 0, mapWidth-screenWidth)
	g.cameraY = clamp(g.cameraY, 0, mapHeight-screenHeight)

	// --- penguin scare / movement ---
	g.updatePenguin(mapWidth, mapHeight)

	// --- player clamp ---
	g.x = clamp(g.x, 0, mapWidth-spriteW)
	g.y = clamp(g.y, 0, mapHeight-spriteH)

	// --- body collision: push penguin away if overlapping (when visible) ---
	if g.penguin.visible && RectsOverlap(g.x, g.y, spriteW, spriteH, g.penguin.x, g.penguin.y, spriteW, spriteH) {
		ResolveDynamicVsSolid(&g.penguin.x, &g.penguin.y, spriteW, spriteH, g.x, g.y, spriteW, spriteH)
	}

	// --- attack spawn (X just pressed) ---
	now := time.Now()
	if inpututil.IsKeyJustPressed(ebiten.KeyX) && now.After(g.nextAttackAt) {
		box := BuildAttackBox(g.x, g.y, g.direction)
		box.Created = now
		g.attacks = append(g.attacks, box)
		g.nextAttackAt = now.Add(g.attackCooldown)
	}

	// --- apply attacks to penguin & prune expired ---
	if g.penguin.visible && g.penguin.Health > 0 {
		dst := g.attacks[:0]
		for _, a := range g.attacks {
			if ApplyAttackToPenguin(a, &g.penguin, now) {
				// (optional) TODO: sfx/flash/knockback
			}
			if !a.Expired(now) {
				dst = append(dst, a)
			}
		}
		g.attacks = dst
		// if penguin died, hide it for now
		if g.penguin.Health <= 0 {
			g.penguin.visible = false
		}
	} else {
		// still prune expired even if penguin hidden
		dst := g.attacks[:0]
		for _, a := range g.attacks {
			if !a.Expired(now) {
				dst = append(dst, a)
			}
		}
		g.attacks = dst
	}

	return nil
}

func (g *Game) updatePenguin(mapWidth, mapHeight float64) {
	// scare/teleport cycle
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
		}
	}

	// movement/anim when visible
	if g.penguin.visible {
		g.penguin.frameDelay++
		if g.penguin.frameDelay >= 10 {
			g.penguin.frame = (g.penguin.frame + 1) % 2
			g.penguin.frameDelay = 0
		}
		g.penguin.moveTimer++
		if g.penguin.moveTimer >= 60 {
			g.penguin.moveTimer = 0
			g.penguin.directionX = randInt(-1, 1)
			g.penguin.directionY = randInt(-1, 1)
		}
		g.penguin.x += float64(g.penguin.directionX)
		g.penguin.y += float64(g.penguin.directionY)
		g.penguin.x = clamp(g.penguin.x, 0, mapWidth-spriteW)
		g.penguin.y = clamp(g.penguin.y, 0, mapHeight-spriteH)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// background
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Translate(-g.cameraX, -g.cameraY)
	screen.DrawImage(g.background, bgOp)

	// penguin (2-frame sheet)
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

	// choose sheet (stabbing/idle-walk)
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

	// (optional) debug: draw attack boxes, flashes, etc.
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}
