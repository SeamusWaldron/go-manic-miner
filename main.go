package main

import (
	"log"

	"manicminer/game"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	g := game.New()
	ebiten.SetWindowSize(game.WindowWidth, game.WindowHeight)
	ebiten.SetWindowTitle("Manic Miner")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
