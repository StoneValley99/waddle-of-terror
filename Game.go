package main

import (
	"image"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
	spriteSheet         *ebiten.Image
	stabbingSpriteSheet *ebiten.Image
	enemyPengSheet      *ebiten.Image
	background          *ebiten.Image
	penguin             PenguinEnemy
	frame               int
	x, y                float64
	direction           int  // 0: down, 1: left, 2: right, 3: up
	framesPerDirection  int  // Number of frames per direction
	frameDelay          int  // Delay to slow down frame rotation
	idle                bool // Flag to check if the character is idle
	stabbing            bool //Flag to check if the character is stabbing
	cameraX, cameraY    float64
}

type PenguinEnemy struct {
	x, y                   float64
	frame                  int
	frameDelay             int
	directionX, directionY int
	moveTimer              int
	visible                bool // 👈 is the penguin visible?
	teleportTimer          int  // 👈 timer before teleport
	scareInterval          int  // 👈 how often to scare
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (g *Game) Update() error {
	g.idle = true
	g.stabbing = false

	// Player movement
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

	// Player animation
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

	// Camera follows player
	g.cameraX = g.x - 320
	g.cameraY = g.y - 240
	screenWidth := 640.0
	screenHeight := 480.0
	mapWidth := float64(g.background.Bounds().Dx())
	mapHeight := float64(g.background.Bounds().Dy())
	maxCameraX := mapWidth - screenWidth
	maxCameraY := mapHeight - screenHeight
	g.cameraX = clamp(g.cameraX, 0, maxCameraX)
	g.cameraY = clamp(g.cameraY, 0, maxCameraY)

	// 🐧 Penguin scare mechanic
	g.penguin.teleportTimer++
	if g.penguin.visible {
		// Disappear when timer hits scareInterval
		if g.penguin.teleportTimer >= g.penguin.scareInterval {
			g.penguin.visible = false
			g.penguin.teleportTimer = 0
		}
	} else {
		// Reappear 1 second later
		if g.penguin.teleportTimer >= 60 {
			g.penguin.visible = true
			g.penguin.teleportTimer = 0

			// Reappear just off-screen around player
			offset := 100.0
			side := rand.Intn(4)
			switch side {
			case 0: // top
				g.penguin.x = g.x
				g.penguin.y = g.y - 240 - offset
			case 1: // bottom
				g.penguin.x = g.x
				g.penguin.y = g.y + 240 + offset
			case 2: // left
				g.penguin.x = g.x - 320 - offset
				g.penguin.y = g.y
			case 3: // right
				g.penguin.x = g.x + 320 + offset
				g.penguin.y = g.y
			}
		}
	}

	// 🐧 Penguin movement & animation (only when visible)
	if g.penguin.visible {
		// Animate penguin
		g.penguin.frameDelay++
		if g.penguin.frameDelay >= 10 {
			g.penguin.frame = (g.penguin.frame + 1) % 2
			g.penguin.frameDelay = 0
		}

		// Move penguin randomly s
		g.penguin.moveTimer++
		if g.penguin.moveTimer >= 60 {
			g.penguin.moveTimer = 0
			g.penguin.directionX = rand.Intn(3) - 1
			g.penguin.directionY = rand.Intn(3) - 1
		}

		// Update and clamp penguin position
		spriteWidth := 32.0 * 3
		spriteHeight := 32.0 * 3
		g.penguin.x += float64(g.penguin.directionX)
		g.penguin.y += float64(g.penguin.directionY)
		g.penguin.x = clamp(g.penguin.x, 0, mapWidth-spriteWidth)
		g.penguin.y = clamp(g.penguin.y, 0, mapHeight-spriteHeight)
	}

	// Clamp player to map
	spriteWidth := 32.0 * 3
	spriteHeight := 32.0 * 3
	g.x = clamp(g.x, 0, mapWidth-spriteWidth)
	g.y = clamp(g.y, 0, mapHeight-spriteHeight)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw background
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Translate(-g.cameraX, -g.cameraY)
	screen.DrawImage(g.background, bgOp)

	// Draw penguin at a fixed location (for now)

	// Assume 2 frames in the penguin sheet (horizontal)
	if g.penguin.visible {
		// Assume 2 frames in the penguin sheet (horizontal)
		pengFrameWidth := g.enemyPengSheet.Bounds().Dx() / 2
		pengFrameHeight := g.enemyPengSheet.Bounds().Dy()

		// Crop only the current frame
		pengSrc := image.Rect(
			g.penguin.frame*pengFrameWidth, 0,
			(g.penguin.frame+1)*pengFrameWidth, pengFrameHeight,
		)
		pengFrame := g.enemyPengSheet.SubImage(pengSrc).(*ebiten.Image)

		pengOp := &ebiten.DrawImageOptions{}
		pengOp.GeoM.Scale(3, 3)
		pengOp.GeoM.Translate(g.penguin.x-g.cameraX, g.penguin.y-g.cameraY)
		screen.DrawImage(pengFrame, pengOp)
	}

	// Select sprite sheet (stabbing or walking)
	var spriteSheet *ebiten.Image
	var frameWidth, frameHeight, framesPerDirection int
	if g.stabbing {
		spriteSheet = g.stabbingSpriteSheet
		framesPerDirection = 12
		frameWidth = spriteSheet.Bounds().Dx() / framesPerDirection
		frameHeight = spriteSheet.Bounds().Dy() / 4
	} else {
		spriteSheet = g.spriteSheet
		framesPerDirection = g.framesPerDirection
		frameWidth = spriteSheet.Bounds().Dx() / framesPerDirection
		frameHeight = spriteSheet.Bounds().Dy() / 4
	}

	// Draw player (vampire)
	srcX := (g.frame % framesPerDirection) * frameWidth
	srcY := g.direction * frameHeight
	srcRect := image.Rect(srcX, srcY, srcX+frameWidth, srcY+frameHeight)
	subImage := spriteSheet.SubImage(srcRect).(*ebiten.Image)

	playerOp := &ebiten.DrawImageOptions{}
	playerOp.GeoM.Scale(3, 3)
	playerOp.GeoM.Translate(g.x-g.cameraX, g.y-g.cameraY)
	screen.DrawImage(subImage, playerOp)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}

func main() {

	spriteSheet, _, err := ebitenutil.NewImageFromFile("graphics/vampire/Walk/Vampires2_Walk_full.png")
	if err != nil {
		log.Fatal(err)
	}

	enemyPengSheet, _, err := ebitenutil.NewImageFromFile("graphics/peng/cute_penguin_idle.png")
	if err != nil {
		log.Fatal(err)
	}

	stabbingSpriteSheet, _, err := ebitenutil.NewImageFromFile("graphics/vampire/Attack/Vampires2_Attack_full.png")
	if err != nil {
		log.Fatal(err)
	}

	background, _, err := ebitenutil.NewImageFromFile("graphics/background.png")
	if err != nil {
		log.Fatal(err)
	}

	game := &Game{
		spriteSheet:         spriteSheet,
		stabbingSpriteSheet: stabbingSpriteSheet,
		enemyPengSheet:      enemyPengSheet,
		background:          background,
		framesPerDirection:  6,    // Set the number of frames per direction
		frameDelay:          0,    // Initialize frame delay
		idle:                true, // Initialize idle state
		x:                   float64(background.Bounds().Dx()) / 2,
		y:                   float64(background.Bounds().Dy()) / 2,
		penguin: PenguinEnemy{
			x:             300,
			y:             300,
			visible:       true,
			scareInterval: 600, // every 10 seconds at 60fpsß
		},
	}

	rand.Seed(time.Now().UnixNano())

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
