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

	deathSheet, _, err := ebitenutil.NewImageFromFile("graphics/vampire/Death/Vampires2_Death_full.png")
	if err != nil {
		log.Fatal(err)
	}

	enemyPengAttackSheet, _, err := ebitenutil.NewImageFromFile("graphics/peng/cute_penguin_attack.png")
	if err != nil {
		log.Fatal(err)
	}

	enemyPengDeathSheet, _, err := ebitenutil.NewImageFromFile("graphics/peng/cute_penguin_death.png")
	if err != nil {
		log.Fatal(err)
	}

	background, _, err := ebitenutil.NewImageFromFile("graphics/background.png")
	if err != nil {
		log.Fatal(err)
	}

	game := &Game{
		spriteSheet:          spriteSheet,
		stabbingSpriteSheet:  stabbingSpriteSheet,
		deathSpriteSheet:     deathSheet,
		enemyPengSheet:       enemyPengSheet,
		enemyPengAttackSheet: enemyPengAttackSheet,
		enemyPengDeathSheet:  enemyPengDeathSheet,
		background:           background,
		framesPerDirection:   6,
		idle:                 true,
		x:                    float64(background.Bounds().Dx()) / 2,
		y:                    float64(background.Bounds().Dy()) / 2,

		deathFramesPerDir: 11, // guess; tweak to match your sheet if needed
		pengFramesIdle:    2,
		pengFramesAttack:  3,
		pengFramesDeath:   2,
		attackCooldown:    300 * time.Millisecond,

		penguin: PenguinEnemy{
			x:             300,
			y:             300,
			visible:       true,
			scareInterval: 600, // ~10s at 60fps
			Health:        3,
			mode:          ModeChase, // starts aggressive
			speed:         2.5,       // faster when chasing
			State:         PengIdle,
		},
	}

	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowTitle("Waddle of Terror")
	ebiten.SetWindowSize(640, 480)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
