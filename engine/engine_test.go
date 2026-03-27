package engine

import (
	"manicminer/action"
	"testing"
)

func TestNewGameEnv(t *testing.T) {
	env := NewGameEnv()
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
	env := NewGameEnv()
	initialAir := env.Air

	// Run 100 frames with no input.
	noAction := action.Action{}
	for i := 0; i < 100; i++ {
		env.Step(noAction)
	}

	// Air should have decreased.
	if env.Air >= initialAir {
		t.Fatalf("expected air to decrease from 0x%02X, got 0x%02X", initialAir, env.Air)
	}
}

func TestStepDeterministic(t *testing.T) {
	// Same action sequence should produce identical results.
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
		env := NewGameEnv()
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
		t.Fatalf("non-deterministic: run1 Willy=(%d,%d) run2 Willy=(%d,%d)",
			obs1.WillyX, obs1.WillyY, obs2.WillyX, obs2.WillyY)
	}
	if obs1.Airborne != obs2.Airborne {
		t.Fatalf("non-deterministic airborne: %d vs %d", obs1.Airborne, obs2.Airborne)
	}
}

func TestReset(t *testing.T) {
	env := NewGameEnv()

	// Step a few frames.
	for i := 0; i < 10; i++ {
		env.Step(action.Action{Right: true})
	}

	// Reset should restore initial state.
	obs := env.Reset(0)
	if obs.WillyX == 0 && obs.WillyY == 0 {
		t.Fatal("reset returned zero position")
	}
	if obs.Air != 0x3F {
		t.Fatalf("expected air 0x3F after reset, got 0x%02X", obs.Air)
	}
}
