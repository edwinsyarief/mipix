package ebipixel

import (
	"image"
	"image/color"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/mipix/internal"
	"github.com/hajimehoshi/ebiten/v2"
)

// Offscreens are logically sized canvases that you can draw to
// and later project to high resolution space. They simplify the
// job of drawing pixel-perfect UI and other camera-independent
// elements of your game (but most of your game should still be
// drawn directly to the canvas received by [Game].Draw(), not
// offscreens).
//
// Creating an offscreen involves creating an [*ebiten.Image], so
// you want to store and reuse them. They also have to be manually
// cleared when required.
type Offscreen struct {
	canvas        *ebiten.Image
	width         int
	height        int
	drawImageOpts ebiten.DrawImageOptions
}

// Creates a new offscreen with the given logical size.
//
// Never invoke this per frame, always reuse offscreens.
func NewOffscreen(width, height int) *Offscreen {
	return &Offscreen{
		canvas: ebiten.NewImage(width, height),
		width:  width, height: height,
	}
}

// Returns the underlying canvas for the offscreen.
func (self *Offscreen) Target() *ebiten.Image {
	return self.canvas
}

// Returns the size of the offscreen.
func (self *Offscreen) Size() (width, height int) {
	return self.width, self.height
}

// Equivalent to [ebiten.Image.DrawImage]().
func (self *Offscreen) Draw(source *ebiten.Image, opts *ebiten.DrawImageOptions) {
	self.canvas.DrawImage(source, opts)
}

// Handy version of [Offscreen.Draw]() with specific coordinates.
func (self *Offscreen) DrawAt(source *ebiten.Image, transform *ebimath.Transform) {
	m := transform.Matrix()
	self.drawImageOpts.GeoM = m
	self.canvas.DrawImage(source, &self.drawImageOpts)
	self.drawImageOpts.GeoM.Reset()
}

// Similar to [ebiten.Image.Fill](), but with BlendSourceOver
// instead of BlendCopy.
func (self *Offscreen) Coat(fillColor color.Color) {
	internal.FillOverRect(self.canvas, self.canvas.Bounds(), fillColor)
}

// Similar to [Offscreen.Coat](), but restricted to a specific
// rectangular area.
func (self *Offscreen) CoatRect(bounds image.Rectangle, fillColor color.Color) {
	internal.FillOverRect(self.canvas, bounds, fillColor)
}

// Clears the underlying offscreen canvas.
func (self *Offscreen) Clear() {
	self.canvas.Clear()
}

// Projects the offscreen into the given target. In most cases,
// you will want to draw to the active high resolution target of
// your game (the second argument of a [QueueHiResDraw]() handler).
func (self *Offscreen) Project(target *ebiten.Image) {
	pkgController.project(self.canvas, target)
}
