package game

import (
	"encoding/json"

	"github.com/signal-k/haphazard-martians/engine"
)

// Analytics (KES-505): the game reports what players do as "track" events,
// and the browser host forwards them to PostHog when it is configured. Events
// carry only game facts (level, parts, times), never anything about the player.
// Names start with hm_ because the PostHog project is shared with Landnam.
const (
	EvTutorialStep  = "hm_tutorial_step"  // step, name
	EvTutorialDone  = "hm_tutorial_done"  // skipped
	EvTestDrop      = "hm_test_drop"      // level, verdict
	EvBuildLaunched = "hm_build_launched" // level, cost, cells, parts by kind, expedition, lesson
	EvRunAbandoned  = "hm_run_abandoned"  // level, how (abort|restart), progress, time
	EvAccountOpened = "hm_account_opened" // sync state, has_username
	EvDailyStarted  = "hm_daily_started"  // date, level
)

// Track emits one analytics event through the host.
func Track(h engine.Host, event string, props map[string]any) {
	if props == nil {
		props = map[string]any{}
	}
	b, err := json.Marshal(map[string]any{"event": event, "props": props})
	if err != nil {
		return
	}
	h.Emit("track", b)
}

// tutStepNames name the build tutorial's steps in analytics.
var tutStepNames = [...]string{tutBeam: "beam", tutWheel: "wheel", tutMotor: "motor", tutMast: "mast", tutTest: "test", tutGo: "go"}

// designProps describes a design for analytics: its cost, size and parts.
func designProps(d Design) map[string]any {
	parts := map[string]int{}
	for _, c := range d.Cells {
		parts[string(c.K)]++
	}
	return map[string]any{"cost": d.Cost(), "cells": len(d.Cells), "parts": parts}
}

// sessionProps are what every in-mission event says about where it happened.
func sessionProps(ses *Session) map[string]any {
	p := map[string]any{"level": ses.Level.ID, "body": BodyOf(ses.Level.LevelDef).Key}
	if ses.Lesson > 0 {
		p["lesson"] = ses.Lesson
	}
	if ses.Exp != nil {
		p["expedition_stage"] = ses.Exp.Index + 1
	}
	return p
}
