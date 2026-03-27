package engine

import (
	"manicminer/action"
	"testing"
)

// newPlayingEnv creates a GameEnv directly in playing state for testing.
func newPlayingEnv() *GameEnv {
	e := NewGameEnv()
	e.startGame() // Skip title, go straight to playing.
	return e
}

func TestNewGameEnv(t *testing.T) {
	env := NewGameEnv()
	if env.State != StateTitle {
		t.Fatalf("expected StateTitle, got %d", env.State)
	}

	// Start game and check playing state.
	env.startGame()
	obs := env.GetObservation()
	if obs.CavernName == "" {
		t.Fatal("expected cavern name, got empty string")
	}
	if obs.Air != 0x3F {
		t.Fatalf("expected air 0x3F, got 0x%02X", obs.Air)
	}
	if obs.Lives != 2 {
		t.Fatalf("expected 2 lives, got %d", obs.Lives)
	}
}

func TestStepNoInput(t *testing.T) {
	env := newPlayingEnv()
	initialAir := env.Air

	noAction := action.Action{}
	for i := 0; i < 100; i++ {
		env.Step(noAction)
	}

	if env.Air >= initialAir {
		t.Fatalf("expected air to decrease from 0x%02X, got 0x%02X", initialAir, env.Air)
	}
}

func TestStepDeterministic(t *testing.T) {
	actions := []action.Action{
		{Right: true},
		{Right: true},
		{Right: true},
		{Right: true},
		{Jump: true},
		{},
		{},
		{},
	}

	run := func() Observation {
		env := newPlayingEnv()
		var obs Observation
		for _, a := range actions {
			result := env.Step(a)
			obs = result.Obs
		}
		return obs
	}

	obs1 := run()
	obs2 := run()

	if obs1.WillyX != obs2.WillyX || obs1.WillyY != obs2.WillyY {
		t.Fatalf("non-deterministic: run1=(%d,%d) run2=(%d,%d)",
			obs1.WillyX, obs1.WillyY, obs2.WillyX, obs2.WillyY)
	}
}

func TestReset(t *testing.T) {
	env := newPlayingEnv()
	for i := 0; i < 10; i++ {
		env.Step(action.Action{Right: true})
	}
	obs := env.Reset(0)
	if obs.Air != 0x3F {
		t.Fatalf("expected air 0x3F after reset, got 0x%02X", obs.Air)
	}
}

func TestTitleToPlaying(t *testing.T) {
	env := NewGameEnv()
	if env.State != StateTitle {
		t.Fatal("expected title state")
	}

	// Press jump to start game.
	env.Step(action.Action{Jump: true})

	if env.State != StatePlaying {
		t.Fatalf("expected playing state after input, got %d", env.State)
	}
}
