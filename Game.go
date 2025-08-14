package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

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
		framesPerDirection:  6,
		idle:                true,
		x:                   float64(background.Bounds().Dx()) / 2,
		y:                   float64(background.Bounds().Dy()) / 2,
		attackCooldown:      300 * time.Millisecond,
		penguin: PenguinEnemy{
			x:             300,
			y:             300,
			visible:       true,
			scareInterval: 600, // ~10s at 60fps
			Health:        3,
		},
	}

	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowTitle("Waddle of Terror")
	ebiten.SetWindowSize(640, 480)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
