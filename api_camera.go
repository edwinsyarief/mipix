package mipix

import "image"

import "github.com/tinne26/mipix/zoomer"
import "github.com/tinne26/mipix/tracker"
import "github.com/tinne26/mipix/shaker"

// See [Camera]().
type AccessorCamera struct{}

// Provides access to camera-related functionality in a structured
// manner. Use through method chaining, e.g.:
//   mipix.Camera().Zoom(2.0)
func Camera() AccessorCamera { return AccessorCamera{} } 

// --- tracking ---

// Returns the current tracker. See [AccessorCamera.SetTracker]()
// for more details.
func (AccessorCamera) GetTracker() tracker.Tracker {
	return pkgController.cameraGetTracker()
}

// Sets the tracker in charge of updating the camera position.
// By default the tracker is nil, and tracking is handled by a
// fallback [tracker.SpringTailer]. If you want something simpler
// at the start, you can easily switch to an instant tracker:
//   import "github.com/tinne26/mipix/tracker"
//   mipix.Camera().SetTracker(tracker.Instant)
func (AccessorCamera) SetTracker(tracker tracker.Tracker) {
	pkgController.cameraSetTracker(tracker)
}

// Feeds the camera the newest target coordinates to point to or
// look at. The time that it takes to reach these new coordinates
// will depend on the behavior of the current [tracker.Tracker].
//
// You can pass coordinates as many times as you want, the
// target position is always set to the most recent pair.
func (AccessorCamera) NotifyCoordinates(x, y float64) {
	pkgController.cameraNotifyCoordinates(x, y)
}

// Immediately sets the camera coordinates to the given values.
// Commonly used when changing scenes or maps.
func (AccessorCamera) ResetCoordinates(x, y float64) {
	pkgController.cameraResetCoordinates(x, y)
}

// This method allows updating the [AccessorCamera.Area]()
// even during [Game].Update(). By default, this happens
// automatically after [Game].Update(), but flushing the
// coordinates can force an earlier update.
//
// Notice that only one camera update can happen per tick,
// so the automatic camera update will be skipped if you
// flush coordinates manually during [Game].Update(). 
// Calling this method multiple times during the same update 
// will only update coordinates on the first invocation.
//
// If you don't need this feature, it's better to forget about
// this method. This is only necessary if you need the camera
// area to remain perfectly consistent between update and draw(s),
// in which case you update the player position first, then notify
// the coordinates and finally flush them.
func (AccessorCamera) FlushCoordinates() {
	pkgController.cameraFlushCoordinates()
}

// Returns the logical area of the game that has to be
// rendered on [Game].Draw()'s canvas or successive logical
// draws. Notice that this can change after each [Game].Update(),
// since the camera might be zoomed or shaking.
//
// Notice that the area will typically be slightly different
// between [Game].Update() and [Game].Draw(). If you need more
// manual control over that, see [AccessorCamera.FlushCoordinates]().
func (AccessorCamera) Area() image.Rectangle {
	return pkgController.cameraAreaGet()
}

// Similar to [AccessorCamera.Area](), but without rounding up
// the coordinates and returning the exact values. Rarely
// necessary in practice.
func (AccessorCamera) AreaF64() (minX, minY, maxX, maxY float64) {
	return pkgController.cameraAreaF64()
}

// --- zoom ---

// Sets a new target zoom level. The transition from the current
// zoom level to the new one is managed by a [zoomer.Zoomer].
func (AccessorCamera) Zoom(newZoomLevel float64) {
	pkgController.cameraZoom(newZoomLevel)
}

// Returns the current [zoomer.Zoomer] interface.
// See [AccessorCamera.SetZoomer]() for more details.
func (AccessorCamera) GetZoomer() zoomer.Zoomer {
	return pkgController.cameraGetZoomer()
}

// Sets the [zoomer.Zoomer] in charge of updating camera zoom levels.
// By default the zoomer is nil, and zoom levels are handled
// by a fallback [zoomer.Quadratic].
func (AccessorCamera) SetZoomer(zoomer zoomer.Zoomer) {
	pkgController.cameraSetZoomer(zoomer)
}

// Returns the current and target zoom levels.
func (AccessorCamera) GetZoom() (current, target float64) {
	return pkgController.cameraGetZoom()
}

// --- screen shaking ---

// Returns the shaker interface associated to the given shaker
// channel (or to the default channel zero if none is passed).
// Passing multiple channels will make the function panic.
// 
// See [AccessorCamera.SetShaker]() for more details.
func (AccessorCamera) GetShaker(channel ...shaker.Channel) shaker.Shaker {
	return pkgController.cameraGetShaker()
}

// Sets a shaker. By default the screen shaker interface is
// nil, and shakes are handled by a fallback [shaker.Random].
//
// If you don't specify any shaker channel, the shaker will be
// set to the default channel zero. Attempting to pass multiple
// channels will make the function panic.
func (AccessorCamera) SetShaker(shaker shaker.Shaker, channel ...shaker.Channel) {
	pkgController.cameraSetShaker(shaker, channel...)
}

// Starts a screen shake that will continue indefinitely until
// stopped by [AccessorCamera.EndShake](). If no shaker channel(s)
// are specified, the shake will start on the default channel zero.
//
// Calling this method repeatedly to force a shaker to start if
// not yet active is possible and safe.
func (AccessorCamera) StartShake(fadeIn TicksDuration, channels ...shaker.Channel) {
	pkgController.cameraStartShake(fadeIn, channels...)
}

// Stops a screen shake. This can be used to stop shakes initiated with
// [AccessorCamera.StartShake](), but also to stop triggered shakes early
// or ensure that no shakes remain active after screen transitions and
// similar. Calling this method repeatedly to make sure that shaking
// is either stopped or stopping is possible and safe.
//
// If no shaker channel(s) are specified, the end shake command will
// be sent to the default channel zero.
func (AccessorCamera) EndShake(fadeOut TicksDuration, channels ...shaker.Channel) {
	pkgController.cameraEndShake(fadeOut, channels...)
}

// If no shaker channel is specified, the function returns whether
// any camera shake is active. If a shaker channel is specified, the
// function will only return whether that specific channel is active.
func (AccessorCamera) IsShaking(channel ...shaker.Channel) bool {
	return pkgController.cameraIsShaking(channel...)
}

// Triggers a screenshake with specific fade in, duration and fade
// out tick durations. If no explicit shaker channels are passed,
// the trigger will be applied to the default channel zero.
//
// Currently, triggering a shake on a channel that's already active
// and with an undefined duration will override the shake duration,
// eventually bringing it to a stop. That being said, this is considered
// an ambiguous situation that you should try to avoid in the first place.
func (AccessorCamera) TriggerShake(fadeIn, duration, fadeOut TicksDuration, channels ...shaker.Channel) {
	pkgController.cameraTriggerShake(fadeIn, duration, fadeOut, channels...)
}

// This might be interesting for ephemerous shakes, so you don't have to be tracking and managing
// everything so manually. That being said, you would still need a pool and to manage everything
// diligently, so maybe there's not much gain here.
// func (AccessorCamera) TriggerEventShake(shaker shaker.Shaker, fadeIn, duration, fadeOut TicksDuration) {
//
// }
