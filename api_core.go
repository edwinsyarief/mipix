package ebipixel

import (
	"fmt"
	"image/color"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

var _ fmt.Formatter

// --- game ---

// The game interface for ebipixel, which is the equivalent to
// [ebiten.Game] on Ebitengine but without the Layout() method.
type Game interface {
	// Updates the game logic.
	//
	// You can implement this almost exactly in the same
	// way you would do for a pure Ebitengine game.
	Update() error

	// Draws the game contents.
	//
	// Unlike Ebitengine, the canvas you receive from ebipixel can vary in
	// size depending on the zoom level or camera position, but it always
	// represents pixels one-to-one. Your mindset should be ignoring the
	// canvas size and focusing on "rendering the game area specified by
	// ebipixel.Camera().Area()".
	Draw(logicalCanvas *ebiten.Image)
}

// Equivalent to [ebiten.RunGame](), but expecting a ebipixel [Game]
// instead of an [ebiten.Game].
//
// Will panic if invoked before [SetResolution]().
func Run(game Game) error {
	return pkgController.run(game)
}

// --- core ---

// Returns the game's base resolution. See [SetResolution]()
// for more details.
func GetResolution() (width, height int) {
	return pkgController.getResolution()
}

// Sets the game's base resolution. This defines the game's
// aspect ratio and logical canvas size at integer coordinates
// and zoom = 1.0.
func SetResolution(width, height int) {
	pkgController.setResolution(width, height)
}

// Ignore this function unless you are already using [QueueHiResDraw]().
// This function is only relevant when trying to interleave logical
// and high resolution draws.
//
// The canvas passed to the callback will be preemptively cleared if
// the previous draw was a high resolution draw.
//
// Must only be called from [Game].Draw() or successive draw callbacks.
func QueueDraw(handler func(logicalCanvas *ebiten.Image)) {
	pkgController.queueDraw(handler)
}

// Schedules the given handler to be invoked after the current
// drawing function and any other queued draws finish.
//
// The viewport passed to the handler is the full game screen canvas,
// including any possibly unused borders, while hiResCanvas is a subimage
// corresponding to the active area of the viewport.
//
// Using this function is necessary if you want to render high resolution
// graphics. This includes vectorial text, some UI, smoothly moving entities,
// shader effects and more.
//
// Must only be called from [Game].Draw() or successive draw callbacks.
// See also [QueueDraw]().
func QueueHiResDraw(handler func(viewport, hiResCanvas *ebiten.Image)) {
	pkgController.queueHiResDraw(handler)
}

// Returns whether a layout change has happened on the current tick.
// Layout changes happen whenever the game window is resized in windowed
// mode, the game switches between windowed and fullscreen modes, or
// the device scale factor changes (possibly due to a monitor change).
//
// This function is only relevant if you need to redraw game borders manually
// and efficiently or resize offscreens. Even if you have [AccessorRedraw.SetManaged](true),
// you rarely need to worry about the state of the layout; any changes will
// automatically trigger a redraw request.
func LayoutHasChanged() bool {
	return pkgController.layoutHasChanged
}

// --- high resolution drawing ---

// See [HiRes]().
type AccessorHiRes struct{}

// Provides access to high resolution drawing methods in
// a structured manner. Use through method chaining, e.g.:
//
//	ebipixel.HiRes().Draw(target, source, x, y)
func HiRes() AccessorHiRes { return AccessorHiRes{} }

func (self AccessorHiRes) Width() int {
	return pkgController.hiResWidth
}

func (self AccessorHiRes) Height() int {
	return pkgController.hiResHeight
}

// Draws the source into the given target at the given global logical
// coordinates (camera origin is automatically subtracted).
//
// Notice that ebipixel's main focus is not high resolution drawing, and this
// method is not expected to be used more than a dozen times per frame.
// If you are only drawing the main character or a few entities at floating
// point positions, using this method should be fine. If you are trying to
// draw every element of your game with this, or relying on this for a
// particle system, you are misusing ebipixel.
//
// Many more high resolution drawing features could be provided, and some
// might be added in the future, but this is not the main goal of the project.
//
// All that being said, this is not a recommendation to avoid this method.
// This method is perfectly functional and a very practical tool in many
// scenarios.
func (self AccessorHiRes) Draw(target, source *ebiten.Image, transform *ebimath.Transform) {
	pkgController.hiResDraw(target, source, transform)
}

// Fills the logical area designated by the given coordinates with fillColor.
// If you need fills with alpha blending directly without high resolution,
// see the utils subpackage.
func (self AccessorHiRes) FillOverRect(target *ebiten.Image, minX, minY, maxX, maxY float64, fillColor color.Color) {
	pkgController.hiResFillOverRect(target, minX, minY, maxX, maxY, fillColor)
}

// --- scaling ---

// See [Scaling]().
type AccessorScaling struct{}

// Provides access to scaling-related functionality in a structured
// manner. Use through method chaining, e.g.:
//
//	ebipixel.Scaling().SetFilter(ebipixel.Hermite)
func Scaling() AccessorScaling { return AccessorScaling{} }

// See [AccessorScaling.SetFilter]().
//
// Multiple filter options are provided mostly as comparison points.
// In general, sticking to [AASamplingSoft] is recommended.
type ScalingFilter uint8

const (
	// Anti-aliased pixel art point sampling. Good default, reasonably
	// performant, decent balance between sharpness and stability during
	// zooms and small movements.
	AASamplingSoft ScalingFilter = iota

	// Like AASamplingSoft, but slightly sharper and slightly less stable
	// during zooms and small movements.
	AASamplingSharp

	// No interpolation. Sharpest and fastest filter, but can lead
	// to distorted geometry. Very unstable, zooming and small movements
	// will be really jumpy and ugly.
	Nearest

	// Slightly blurrier than AASamplingSoft and more unstable than
	// AASamplingSharp. Still provides fairly decent results at
	// reasonable performance.
	Hermite

	// The most expensive filter by quite a lot. Slightly less sharp than
	// Hermite, but quite a bit more stable. Might slightly misrepresent
	// some colors throughout high contrast areas.
	Bicubic

	// Offered mostly for comparison purposes. Slightly blurrier than
	// Hermite, but quite a bit more stable.
	Bilinear

	// Offered for comparison purposes only. Non high-resolution aware
	// scaling filter, more similar to what naive scaling will look like.
	SrcHermite

	// Offered for comparison purposes only. Non high-resolution aware
	// scaling filter, more similar to what naive scaling will look like.
	SrcBicubic

	// Offered for comparison purposes only. Non high-resolution aware
	// scaling filter, more similar to what naive scaling will look like.
	// This is what Ebitengine will do by default with the FilterLinear
	// filter.
	SrcBilinear

	scalingFilterEndSentinel
)

// Returns a string representation of the scaling filter.
func (self ScalingFilter) String() string {
	switch self {
	case AASamplingSoft:
		return "AASamplingSoft"
	case AASamplingSharp:
		return "AASamplingSharp"
	case Nearest:
		return "Nearest"
	case Hermite:
		return "Hermite"
	case Bicubic:
		return "Bicubic"
	case Bilinear:
		return "Bilinear"
	case SrcHermite:
		return "SrcHermite"
	case SrcBicubic:
		return "SrcBicubic"
	case SrcBilinear:
		return "SrcBilinear"
	default:
		panic("invalid ScalingFilter")
	}
}

// Set to true to avoid black borders and completely fill the screen
// no matter how ugly it gets. By default, stretching is disabled. In
// general you only want to expose stretching as a setting for players;
// don't set it to true on your own.
//
// Must only be called during initialization or [Game].Update().
func (AccessorScaling) SetStretchingAllowed(allowed, keepAspectRatio, dynamicScaling bool) {
	pkgController.scalingSetStretchingAllowed(allowed, keepAspectRatio, dynamicScaling)
}

// Returns whether stretching is allowed for screen scaling.
// See [AccessorScaling.SetStretchingAllowed]() for more details.
func (AccessorScaling) GetStretchingAllowed() bool {
	return pkgController.scalingGetStretchingAllowed()
}

func (AccessorScaling) SetBestFitContextSize(width, height int) {
	pkgController.setBestFitContextSize(width, height)
	pkgController.setBestFitRenderSize(pkgController.logicalWidth, pkgController.logicalHeight)
}

// Changes the scaling filter. The default is [AASamplingSoft].
//
// Must only be called during initialization or [Game].Update().
//
// The first time you set a filter explicitly, its shader will also
// be compiled. This means that this function can be effectively used
// to precompile the relevant shaders. Otherwise, the shader will be
// compiled the first time it has to be used.
func (AccessorScaling) SetFilter(filter ScalingFilter) {
	pkgController.scalingSetFilter(filter)
}

// Returns the current scaling filter. The default is [AASamplingSoft].
func (AccessorScaling) GetFilter() ScalingFilter {
	return pkgController.scalingGetFilter()
}

// --- conversions ---

// See [Convert]().
type AccessorConvert struct{}

// Provides access to coordinate conversions in a structured
// manner. Use through method chaining, e.g.:
//
//	cx, cy := ebiten.CursorPosition()
//	lx, ly := ebipixel.Convert().ToLogicalCoords(cx, cy)
func Convert() AccessorConvert { return AccessorConvert{} }

// Transforms coordinates obtained from [ebiten.CursorPosition]() and
// similar functions to coordinates within the game's global logical
// space.
//
// Commonly used to see what is being clicked on the game's world.
func (AccessorConvert) ToLogicalCoords(x, y int) (float64, float64) {
	return pkgController.convertToLogicalCoords(x, y)
}

// Transforms coordinates obtained from [ebiten.CursorPosition]() and
// similar functions to relative screen coordinates between 0 and 1.
//
// Commonly used to see what is being clicked on the game's UI or
// applying fancy shaders and effects that depend on the cursor's
// relative position on screen.
func (AccessorConvert) ToRelativeCoords(x, y int) (float64, float64) {
	return pkgController.convertToRelativeCoords(x, y)
}

// Transforms coordinates obtained from [ebiten.CursorPosition]() and
// similar functions to screen coordinates rescaled between (0, 0) and
// (GameWidth, GameHeight).
//
// Commonly used to see what is being clicked on the game's UI (when
// the UI is pure pixel art).
func (AccessorConvert) ToGameResolution(x, y int) (float64, float64) {
	return pkgController.convertToGameResolution(x, y)
}

// --- debug ---

// See [Debug]().
type AccessorDebug struct{}

// Provides access to debugging functionality in a structured
// manner. Use through method chaining, e.g.:
//
//	ebipixel.Debug().Drawf("current tick: %d", ebipixel.Tick().Now())
func Debug() AccessorDebug { return AccessorDebug{} }

// Similar to Printf debugging, but drawing the text on the top
// left of the screen (instead of printing on the terminal).
// Multi-line text is not supported; use multiple Drawf commands
// in sequence instead.
//
// You can call this function at any point, even during [Game].Update().
// Strings will be queued and rendered at the end of the next draw.
func (AccessorDebug) Drawf(format string, args ...any) {
	pkgController.debugDrawf(format, args...)
}

// Similar to [fmt.Printf](), but expects two tick counts as the first
// arguments. The function will only print during the period elapsed
// between those two tick counts.
// Some examples:
//
//	ebipixel.Debug().Printfr(0, 0, "only print on the first tick\n")
//	ebipixel.Debug().Printfr(180, 300, "print from 3s to 5s lapse\n")
func (AccessorDebug) Printfr(firstTick, lastTick uint64, format string, args ...any) {
	pkgController.debugPrintfr(firstTick, lastTick, format, args...)
}

// Similar to [fmt.Printf](), but only prints every N ticks. For
// example, in most games using N = 60 will lead to print once
// per second.
func (AccessorDebug) Printfe(everyNTicks uint64, format string, args ...any) {
	// 61 is prime
	pkgController.debugPrintfe(everyNTicks, format, args...)
}

// Similar to [fmt.Printf](), but only prints if the given key is pressed.
// Common keys: [ebiten.KeyShiftLeft], [ebiten.KeyControl], [ebiten.KeyDigit1].
func (AccessorDebug) Printfk(key ebiten.Key, format string, args ...any) {
	pkgController.debugPrintfk(key, format, args...)
}

// --- ticks ---

// See [Tick]().
type AccessorTick struct{}

// Provides access to game tick functions in a structured
// manner. Use through method chaining, e.g.:
//
//	currentTick := ebipixel.Tick().Now()
func Tick() AccessorTick { return AccessorTick{} }

// Returns the current tick.
func (AccessorTick) Now() uint64 {
	return pkgController.tickNow()
}

// Returns the updates per second. This is [ebiten.TPS](),
// but ebipixel considers a more advanced model for [ticks
// and updates].
//
// [ticks and updates]: https://github.com/edwinsyarief/mipix/blob/main/docs/ups-vs-tps.md
func (AccessorTick) UPS() int {
	return ebiten.TPS()
}

// This is just [ebiten.SetTPS]() under the hood, but ebipixel
// considers a more advanced model for [ticks and updates].
//
// [ticks and updates]: https://github.com/edwinsyarief/mipix/blob/main/docs/ups-vs-tps.md
func (AccessorTick) SetUPS(updatesPerSecond int) {
	ebiten.SetTPS(updatesPerSecond)
}

// Returns the ticks per second. This is UPS()*TickRate.
// Notice that this is not [ebiten.TPS](), as ebipixel considers
// a more advanced model for [ticks and updates].
//
// [ticks and updates]: https://github.com/edwinsyarief/mipix/blob/main/docs/ups-vs-tps.md
func (AccessorTick) TPS() int {
	return ebiten.TPS() * int(pkgController.tickRate)
}

// Sets the tick rate (ticks per update, often refered to as "TPU").
// Notice that this is not [ebiten.SetTPS](), as ebipixel considers
// a more advanced model for [ticks and updates].
//
// [ticks and updates]: https://github.com/edwinsyarief/mipix/blob/main/docs/ups-vs-tps.md
func (AccessorTick) SetRate(tickRate int) {
	pkgController.tickSetRate(tickRate)
}

// Returns the current tick rate. Defaults to 1.
// See [AccessorTick.SetRate]() for more context.
func (AccessorTick) GetRate() int {
	return pkgController.tickGetRate()
}
