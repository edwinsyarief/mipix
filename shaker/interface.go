// This package defines a [Shaker] interface that the mipix
// camera can use to perform screen shakes, and provides
// a few default implementations.
//
// All provided implementations respect a few properties:
//  - Resolution independent: range of motion for the shakes
//    is not hardcoded, but proportional to the game's resolution.
//  - Tick-rate independent: results are visually similar
//    regardless of your Tick().UPS() and Tick().GetRate() values.
//    See [ups-vs-tps] if you need more context.
// These are nice properties for public implementations, but if you
// are writing your own, remember that most often these properties
// won't be relevant to you. You can ignore them and make your life
// easier if you are only getting started.
//
// [ups-vs-tps]: https://github.com/tinne26/mipix/blob/main/docs/ups-vs-tps.md
package shaker

// The interface for mipix screen shakers.
//
// Given a level that transitions linearly between 0 and 1
// during the fade in and fade out stages, GetShakeOffsets()
// returns the logical offsets for the camera.
//
// After stoping, there will be one call with level = 0 that
// can be used to reset the shaker state. The results of this
// call will be disregarded.
//
// Minor detail: all built-in implementations happen to normalize
// the fade in/out level with a cubic smoothstep, just to make
// things nicer.
type Shaker interface {
	GetShakeOffsets(level float64) (float64, float64)
}

// Used by mipix in case multiple shakes need to be active at the same time.
//
// Channel zero is special and will use a fallback shaker even if uninitialized
// (though it's best practice to always set your shakers explicitly). It's also
// the channel that will be automatically selected for most shaker functions in
// mipix if no channel is explicitly passed.
//
// Here's an example of when multiple channels are useful:
//  - You need an always-on shake for camera motion or environment shaking,
//    like being in a ship or hot air ballon.
//  - You need the typical triggered shakes for momentary impacts, explosions,
//    earthquakes and so on.
//  - You have some alter states like drunk or confused that might use some
//    extra shaker channels.
// In these cases, you should define your own channel constants, e.g:
//   const (
//     ChanBackground shaker.Channel = iota
//     ChanTrigger
//     ChanDrunk
//   )
// In even more complex cases, you might decide to treat your channels like a
// shaker pool to manage everything more dynamically; who knows, it all depends
// on the game.
type Channel uint8
