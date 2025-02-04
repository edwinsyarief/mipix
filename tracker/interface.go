// This package defines a [Tracker] interface that the ebipixel
// camera can use to update its position, and provides a few
// default implementations.
//
// All provided implementations respect a few properties:
//   - Resolution independent: range of motion for the tracking
//     is not hardcoded, but proportional to the game's resolution.
//   - Tick-rate independent: tracking preserves the same relative
//     speed regardless of your Tick().UPS() and Tick().GetRate()
//     values. See [ups-vs-tps] if you need more context.
//
// These are nice properties for public implementations, but if you
// are writing your own, remember that most often these properties
// won't be relevant to you. You can ignore them and make your life
// easier if you are only getting started.
//
// [ups-vs-tps]: https://github.com/edwinsyarief/mipix/blob/main/docs/ups-vs-tps.md
package tracker

// The interface for ebipixel camera tracking.
//
// Given current and target coordinates, a tracker must return
// the position change for a single update.
type Tracker interface {
	Update(currentX, currentY, targetX, targetY, prevSpeedX, prevSpeedY float64) (float64, float64)
}
