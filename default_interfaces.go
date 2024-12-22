package ebipixel

import (
	"github.com/edwinsyarief/mipix/shaker"
	"github.com/edwinsyarief/mipix/tracker"
	"github.com/edwinsyarief/mipix/zoomer"
)

var defaultZoomer *zoomer.Quadratic
var defaultTracker *tracker.SpringTailer
var defaultShaker *shaker.Random
