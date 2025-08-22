package main

import (
	"fmt"
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"image/color"
)

const matchTarget = 5 // first to N wins

type Game struct {
	// assets
	spriteSheet          *ebiten.Image
	stabbingSpriteSheet  *ebiten.Image
	deathSpriteSheet     *ebiten.Image
	enemyPengSheet       *ebiten.Image
	enemyPengAttackSheet *ebiten.Image
	enemyPengDeathSheet  *ebiten.Image
	background           *ebiten.Image

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
	penguin          PenguinEnemy
	pengFramesIdle   int
	pengFramesAttack int
	pengFramesDeath  int

	// combat
	attacks         []AttackBox
	nextAttackAt    time.Time
	attackCooldown  time.Duration
	attackAnimTicks int // latch: remaining ticks of stab anim
	hitCount        int // successful strikes counter

	// UI buttons
	respawnButtonVisible bool
	respawnButtonRect    image.Rectangle

	gameStarted     bool
	startButtonRect image.Rectangle

	newMatchButtonVisible bool
	newMatchButtonRect    image.Rectangle

	// rounds / match
	vampireWins   int
	penguinWins   int
	pendingWinner int
	gameOver      bool
}

const (
	winnerNone = iota
	winnerVampire
	winnerPenguin
)

func (g *Game) Update() error {
	g.idle = true
	now := time.Now()

	// --- PRE-START: Start button / Enter ---
	if !g.gameStarted {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.gameStarted = true
			g.respawnPlayer() // start first round
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			if pointInRect(x, y, g.startButtonRect) {
				g.gameStarted = true
				g.respawnPlayer()
			}
		}
		return nil // Skip game logic until started
	}

	// --- GAME OVER: New match button / N ---
	if g.gameOver {
		if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.startNewMatch()
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			if pointInRect(x, y, g.newMatchButtonRect) {
				g.startNewMatch()
			}
		}
		return nil // Freeze gameplay when match is over
	}

	// --- BETWEEN ROUNDS: Respawn button / R ---
	if g.respawnButtonVisible {
		// keep rect in sync with Draw()
		x, y, w, h := 270, 200, 100, 40
		g.respawnButtonRect = image.Rect(x, y, x+w, y+h)

		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.respawnPlayer()
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			cx, cy := ebiten.CursorPosition()
			if pointInRect(cx, cy, g.respawnButtonRect) {
				g.respawnPlayer()
			}
		}
		return nil // Don't run gameplay updates until respawned
	}

	// --- NORMAL GAMEPLAY (round active) ---

	// Input (disabled when dead)
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

	// Stab animation latch
	g.stabbing = g.attackAnimTicks > 0 && !g.vampireDead
	if g.attackAnimTicks > 0 {
		g.attackAnimTicks--
		g.idle = false
	}

	// Animations (death/walk/stab)
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

	// Camera follow
	g.cameraX = clamp(g.x-320, 0, float64(g.background.Bounds().Dx())-640)
	g.cameraY = clamp(g.y-240, 0, float64(g.background.Bounds().Dy())-480)

	// Penguin AI & movement
	mapWidth := float64(g.background.Bounds().Dx())
	mapHeight := float64(g.background.Bounds().Dy())
	g.updatePenguinAI(mapWidth, mapHeight)

	// Clamp player
	g.x = clamp(g.x, 0, mapWidth-spriteW)
	g.y = clamp(g.y, 0, mapHeight-spriteH)

	// Tight colliders for death/hit detection
	vx, vy, vw, vh := VampCollider(g.x, g.y)
	px, py, pw, ph := PenguinCollider(g.penguin.x, g.penguin.y)

	// If penguin is close/overlapping and not dying, show ATTACK sheet
	if g.penguin.visible && g.penguin.State != PengDeath &&
		RectsOverlap(vx, vy, vw, vh, px, py, pw, ph) {
		g.penguin.State = PengAttack
	} else if g.penguin.State != PengDeath {
		g.penguin.State = PengIdle
	}

	// Vampire dies if tight colliders overlap while penguin is chasing → Penguin scores
	if !g.vampireDead && g.penguin.visible && g.penguin.mode == ModeChase &&
		RectsOverlap(vx, vy, vw, vh, px, py, pw, ph) {

		g.vampireDead = true
		g.deathFrame, g.deathFrameDelay, g.attackAnimTicks = 0, 0, 0

		g.pendingWinner = winnerPenguin
		// g.endRound()
	}

	// Body push = shove penguin ONLY (keeps “run-away” vibe)
	if g.penguin.visible && RectsOverlap(vx, vy, vw, vh, px, py, pw, ph) {
		ResolveDynamicVsSolid(&g.penguin.x, &g.penguin.y, spriteW, spriteH, g.x, g.y, spriteW, spriteH)
	}

	// Apply attacks (switch to flee on hit)
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

	// If penguin died from attacks → Vampire scores, end round (after death anim finishes)
	if g.penguin.visible && g.penguin.State == PengDeath &&
		g.penguin.frame >= g.pengColsDeath()-1 {
		g.penguin.visible = false
		g.vampireWins++
		g.endRound()
	}
	// Finish vampire death → Penguin scores AFTER vampire death anim finishes
	if g.pendingWinner == winnerPenguin && g.vampireDead &&
		g.deathFrame >= g.deathFramesPerDir-1 {

		g.penguinWins++
		g.pendingWinner = winnerNone
		g.penguin.State = PengIdle // show a calm penguin with the overlay
		g.endRound()
	}

	// Prune expired attack boxes
	dst := g.attacks[:0]
	for _, a := range g.attacks {
		if !a.Expired(now) {
			dst = append(dst, a)
		}
	}
	g.attacks = dst

	return nil
}

// Round/match helpers
func (g *Game) endRound() {
	g.attacks = g.attacks[:0]
	g.checkGameOver()
	if !g.gameOver {
		g.respawnButtonVisible = true
	}
}

func (g *Game) checkGameOver() {
	if g.vampireWins >= matchTarget || g.penguinWins >= matchTarget {
		g.gameOver = true
		g.respawnButtonVisible = false
	}
}

func (g *Game) startNewMatch() {
	g.vampireWins, g.penguinWins = 0, 0
	g.gameOver = false
	g.respawnPlayer()
}

func (g *Game) respawnPlayer() {
	// Keep gameStarted = true (only the FIRST time uses Start)
	g.respawnButtonVisible = false
	g.vampireDead = false

	// Reset player (center)
	mapW := float64(g.background.Bounds().Dx())
	mapH := float64(g.background.Bounds().Dy())
	g.x, g.y = mapW/2, mapH/2
	g.frame = 0
	g.deathFrame = 0
	g.hitCount = 0
	g.attackAnimTicks = 0
	g.attacks = g.attacks[:0]

	// Reset penguin state and spawn near edge around player
	g.penguin.Health = 3
	g.pendingWinner = winnerNone
	g.penguin.visible = true
	g.penguin.mode = ModeChase
	g.penguin.speed = 2.5
	g.penguin.frame = 0
	g.penguin.frameDelay = 0
	g.penguin.deathFrameDelay = 0 // IMPORTANT: reset death timer
	g.penguin.State = PengIdle    // IMPORTANT: leave death state
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

// helper function
func pointInRect(x, y int, r image.Rectangle) bool {
	return x >= r.Min.X && x <= r.Max.X && y >= r.Min.Y && y <= r.Max.Y
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
		switch g.penguin.State {
		case PengDeath:
			// Play death sheet once, then wait for round logic to hide/score
			g.penguin.deathFrameDelay++
			if g.penguin.deathFrameDelay >= 6 { // tweak speed
				if g.penguin.frame < g.pengColsDeath()-1 {
					g.penguin.frame++
				}
				g.penguin.deathFrameDelay = 0
			}
			// No movement during death
			return

		case PengAttack:
			// Attack loops while overlapping
			g.penguin.frameDelay++
			if g.penguin.frameDelay >= 8 {
				g.penguin.frame = (g.penguin.frame + 1) % g.pengColsAttack()
				g.penguin.frameDelay = 0
			}

		default: // PengIdle
			g.penguin.frameDelay++
			if g.penguin.frameDelay >= 10 {
				g.penguin.frame = (g.penguin.frame + 1) % g.pengColsIdle()
				g.penguin.frameDelay = 0
			}
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

	// --- Start button (only before first start) ---
	if !g.gameStarted {
		x, y, w, h := 270, 200, 100, 40
		g.startButtonRect = image.Rect(x, y, x+w, y+h)

		btn := ebiten.NewImage(w, h)
		btn.Fill(color.RGBA{50, 150, 50, 255}) // green button
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(btn, op)

		ebitenutil.DebugPrintAt(screen, "Start", x+30, y+12)
		ebitenutil.DebugPrintAt(screen, "(Enter)", x+25, y+26)
		return // Skip drawing world until started
	}

	// Penguin draw
	if g.penguin.visible {
		// Choose sheet by state; fallback safely if a sheet is missing
		var sheet *ebiten.Image
		switch g.penguin.State {
		case PengDeath:
			if g.enemyPengDeathSheet != nil {
				sheet = g.enemyPengDeathSheet
			} else {
				sheet = g.enemyPengSheet
			}
		case PengAttack:
			if g.enemyPengAttackSheet != nil {
				sheet = g.enemyPengAttackSheet
			} else {
				sheet = g.enemyPengSheet
			}
		default:
			sheet = g.enemyPengSheet
		}

		// Auto-detect columns: assumes one row; if frames are square this is exact.
		cols := colsFromSheet(sheet, g.gPengFramesForState())
		if cols <= 0 {
			cols = 1
		}

		pw := sheet.Bounds().Dx() / cols
		ph := sheet.Bounds().Dy()
		if pw > 0 && ph > 0 {
			// Clamp frame
			if g.penguin.frame >= cols {
				g.penguin.frame = cols - 1
			}
			src := image.Rect(g.penguin.frame*pw, 0, (g.penguin.frame+1)*pw, ph)
			img := sheet.SubImage(src).(*ebiten.Image)

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(3, 3)
			op.GeoM.Translate(g.penguin.x-g.cameraX, g.penguin.y-g.cameraY)
			screen.DrawImage(img, op)
		}
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

	// Scoreboard & HUD
	score := fmt.Sprintf("Score  Vampire %d — %d Penguin  (First to %d)", g.vampireWins, g.penguinWins, matchTarget)
	ebitenutil.DebugPrintAt(screen, score, 8, 8)

	// --- Between-rounds: Respawn button ---
	if g.respawnButtonVisible && !g.gameOver {
		x, y, w, h := 270, 200, 100, 40
		g.respawnButtonRect = image.Rect(x, y, x+w, y+h)

		btn := ebiten.NewImage(w, h)
		btn.Fill(color.RGBA{200, 50, 50, 255}) // red button
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(btn, op)

		ebitenutil.DebugPrintAt(screen, "Respawn", x+20, y+12)
		ebitenutil.DebugPrintAt(screen, "(R)", x+40, y+26)
	}

	// --- Game Over: New Match button ---
	if g.gameOver {
		x, y, w, h := 230, 200, 180, 40
		g.newMatchButtonRect = image.Rect(x, y, x+w, y+h)

		btn := ebiten.NewImage(w, h)
		btn.Fill(color.RGBA{50, 50, 200, 255}) // blue button
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(btn, op)

		who := "Vampire"
		if g.penguinWins >= matchTarget {
			who = "Penguin"
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s wins! New Match", who), x+12, y+12)
		ebitenutil.DebugPrintAt(screen, "(N)", x+78, y+26)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("AtkCols:%d  DeathCols:%d  Pending:%d",
			g.gPengFramesAttack(), g.gPengFramesDeath(), g.pendingWinner), 8, 24)
	}
}

// Fallback frame counts if auto-detection can’t infer columns
func (g *Game) gPengFramesIdle() int {
	if g.pengFramesIdle == 0 {
		return 2
	}
	return g.pengFramesIdle
}
func (g *Game) gPengFramesAttack() int {
	if g.pengFramesAttack == 0 {
		return 3
	}
	return g.pengFramesAttack
}
func (g *Game) gPengFramesDeath() int {
	if g.pengFramesDeath == 0 {
		return 2
	}
	return g.pengFramesDeath
}

// Auto-detect columns (for one-row sheets). If frames are square, w%h==0 → use w/h.
// Otherwise use the configured fallback.
func colsFromSheet(img *ebiten.Image, fallback int) int {
	if img == nil {
		return fallback
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if h > 0 && w%h == 0 {
		return w / h
	}
	if fallback > 0 {
		return fallback
	}
	return 1
}

func (g *Game) gPengFramesForState() int {
	switch g.penguin.State {
	case PengDeath:
		return g.gPengFramesDeath()
	case PengAttack:
		return g.gPengFramesAttack()
	default:
		return g.gPengFramesIdle()
	}
}

func (g *Game) pengColsIdle() int { return colsFromSheet(g.enemyPengSheet, g.gPengFramesIdle()) }
func (g *Game) pengColsAttack() int {
	return colsFromSheet(g.enemyPengAttackSheet, g.gPengFramesAttack())
}
func (g *Game) pengColsDeath() int { return colsFromSheet(g.enemyPengDeathSheet, g.gPengFramesDeath()) }

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}
