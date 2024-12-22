package tracker

import (
	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/mipix/internal"
)

type tracker = Tracker

// A few stateless built-in trackers.
var (
	// Update(...) always returns (0, 0).
	Frozen tracker = frozenTracker{}

	// Update(...) always returns (target - current).
	Instant tracker = instantTracker{}

	// Applies a lerp between current and target position.
	Linear tracker = linearTracker{}
)

type frozenTracker struct{}

func (frozenTracker) Update(currentX, currentY, targetX, targetY, prevSpeedX, prevSpeedY float64) (float64, float64) {
	return 0, 0
}

type instantTracker struct{}

func (instantTracker) Update(currentX, currentY, targetX, targetY, prevSpeedX, prevSpeedY float64) (float64, float64) {
	return targetX - currentX, targetY - currentY
}

// A simple linear interpolation tracker.
type linearTracker struct{}

func (self linearTracker) Update(currentX, currentY, targetX, targetY, prevSpeedX, prevSpeedY float64) (float64, float64) {
	// stabilization
	if ebimath.Abs(targetX-currentX) < 0.001 && ebimath.Abs(targetY-currentY) < 0.001 {
		return targetX - currentX, targetY - currentY
	}

	// general update
	w, h := internal.GetResolution()
	zoom := internal.GetCurrentZoom()
	widthF64, heightF64 := float64(w)/zoom, float64(h)/zoom

	updateDelta := 1.0 / float64(internal.GetUPS())
	maxHorzAdvance := 6.0 * zoom * widthF64 * updateDelta  // use higher values for a more rigid / strict tracking
	maxVertAdvance := 6.0 * zoom * heightF64 * updateDelta // use lower values for a more elastic / softer tracking
	minAdvance := 0.01 * updateDelta
	refHorzMaxDist := 2.0 * widthF64 // higher values lead to smoother tracking
	refVertMaxDist := 2.0 * heightF64

	horzAdvance := computeLinComponent(currentX, targetX, minAdvance, maxHorzAdvance, refHorzMaxDist)
	vertAdvance := computeLinComponent(currentY, targetY, minAdvance, maxVertAdvance, refVertMaxDist)
	return horzAdvance, vertAdvance
}
