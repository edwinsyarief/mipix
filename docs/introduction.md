# Introduction to mipix

Key ideas to understand mipix's model, plus a few code samples to illustrate each concept.

## Empty game and `mipix.SetResolution()`

```Golang
package main

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/mipix"

type Game struct {}

func (game *Game) Update() error {
	return nil
}

func (game *Game) Draw(canvas *ebiten.Image) {
	// ...
}

func main() {
	mipix.SetResolution(128, 72)
	err := mipix.Run(&Game{})
	if err != nil { panic(err) }
}
```

Key highlights:
- Our game struct must satisfy the [`mipix.Game`](https://pkg.go.dev/github.com/tinne26/mipix#Game) interface, which is just like ebitengine's, but without the `Layout()` method.
- You have to use `mipix.Run()` instead of `ebiten.Run()`.
- You must always set the game's resolution before `mipix.Run()` through `mipix.SetResolution()`. More advice on how to choose resolutions on the footnotes[^1].

[^1]: Based on [Steam's surveys](https://store.steampowered.com/hwsurvey) the most common display resolution is by far 1920x1080 (full HD), with over 50% share as of 2024. If you are going for a 16:9 aspect ratio, then you should choose a divisor of 1920x1080, like 128x72, 160x90, 320x180, 384x216 (\*), 480x270 (\*), 640x360, 960x540 (\*), etc (the ones marked with \* are not perfectly compatible with QHD). For other aspect ratios, you will typically go more square-ish (less wide), in which case black bars on the sides are unavoidable on common displays. In this case, you should only try to stick to divisors of the most common vertical resolutions. For example, 360 is the greatest common divisor of 1080 (full HD) and 1440 (QHD), so that would be a great choice. Other divisors like 180, 120, 90, 60 and so on would be equally good choices. This is only some basic advice, there are some exceptions to these rules.

## The draw model

While the `Update()` method should be coded pretty much in the same way for both raw Ebitengine and mipix, the `Draw()` method is a different story.

Here are the two key points you really have to engrave in your soul:
- The canvas you receive on draw represents pixels 1-to-1. Your pixel art must be drawn directly to the canvas, in its original size. Forget about display scaling factors and projections... *Just draw your pixel art*.
- The canvas you receive on draw does *not* always have the same size, and it does *not* necessarily match the resolution you set at the start with `mipix.SetResolution(W, H)`. This can happen due to zoom effects (which mipix handles internally), but also something as simple as moving around. For example, even if your resolution is 128x72, if you move around in any direction you might have to render half a pixel for one of the borders of the canvas and half a pixel for the opposite one. In any case, you should not be in the mindset of "render my WxH canvas", but "render all the logical area requested by mipix".
- You can still render *some* elements at decimal positions and in high resolution, but we will touch on that later.

Again: draw things one-to-one, don't assume a specific canvas resolution, just draw the area that mipix asks you to.

**Simple image draw**
```Golang
import "github.com/tinne26/mipix/utils"

const PrawnOX, PrawnOY = -3, -3 // place any desired coords here
var Prawn *ebiten.Image = utils.MaskToImage(6, []uint8{
	0, 0, 0, 0, 1, 0, // example low-res image
	0, 0, 0, 0, 1, 1,
	0, 0, 1, 1, 0, 0,
	0, 1, 1, 1, 0, 0,
	1, 1, 1, 0, 0, 0,
	1, 1, 0, 0, 0, 0,
}, utils.RGB(219, 86, 32))

func (game *Game) Draw(canvas *ebiten.Image) {
	// set some background color
	canvas.Fill(utils.RGB(255, 255, 255))

	// obtain the camera area that mipix is asking
	// us to draw. this is the most critical function
	// that has to be used when drawing with mipix
	camArea := mipix.Camera().Area()

	// see if our content overlaps the area we need to
	// draw, and if it does, we subtract the camera
	// origin coordinates to our object's global coords
	prawnGlobalRect := utils.Shift(Prawn.Bounds(), PrawnOX, PrawnOY)
	if prawnGlobalRect.Overlaps(camArea) {
		// translate from global to local (canvas) coordinates
		prawnLocalRect := prawnGlobalRect.Sub(camArea.Min)

		// create DrawImageOptions and apply draw position
		var opts ebiten.DrawImageOptions
		tx := prawnLocalRect.Min.X
		ty := prawnLocalRect.Min.Y
		opts.GeoM.Translate(float64(tx), float64(ty))
		canvas.DrawImage(Prawn, &opts)
	}
}
```
[*(full example code here)*](https://github.com/tinne26/mipix-examples/tree/main/src/tutorial/draw_image)

Notice that the image will be drawn in the center of the screen in this example. This is because the camera, by default, points at (0, 0), and our image is 6x6 but drawn at (-3, -3).

That was the general code to draw an image, but there are some shorter ways using `mipix/utils` too:
```Golang
prawnGlobalRect := utils.Shift(Prawn.Bounds(), PrawnOX, PrawnOY)
if prawnGlobalRect.Overlaps(camArea) {
	opts := utils.DrawImageOptionsAt(Prawn, PrawnOX, PrawnOY)
	canvas.DrawImage(Prawn, &opts)
}
```

**Simple rectangle fill**
```Golang
import "github.com/tinne26/mipix/utils"

const RectCX, RectCY = 0, 0 // place any desired coords here
var SomeRect = utils.Rect(RectCX - 1, RectCY - 1, RectCX + 1, RectCY + 1)
var RectColor = utils.RGBA(128, 128, 128, 128)

func (game *Game) Draw(canvas *ebiten.Image) {
	// white background fill
	canvas.Fill(utils.RGB(255, 255, 255))

	// fill the rect, which is defined in logical global coords
	camArea := mipix.Camera().Area()
	if SomeRect.Overlaps(camArea) {
		localRect := SomeRect.Sub(camArea.Min)
		utils.FillOverRect(canvas, localRect, RectColor)
	}
}
```
[*(full example code here)*](https://github.com/tinne26/mipix-examples/tree/main/src/tutorial/draw_rect)

## Accessors and camera movement

Now that we have learned how to draw some content on screen, we can try to move the camera around.

Most of mipix's functionality is grouped on "accessors". Accessors are just dummy types used to group together related functionality. The most important accessor is the `AccessorCamera`, which gives you access to the area for drawing, to notify new camera targets and to make the camera zoom and shake.

Let's start with camera movement. We will fill the `Update()` method with some basic logic, while we leave a 2x2 square drawn on (0, 0) so we can notice the movement.

```Golang
package main

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/mipix"
import "github.com/tinne26/mipix/utils"

type Game struct {
	LookAtX, LookAtY float64
}

func (game *Game) Update() error {
	// detect directions
	up    := ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	down  := ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	left  := ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	right := ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	if up   && down  { up  , down  = false, false }
	if left && right { left, right = false, false }
	
	// apply diagonal speed reduction if needed
	var speed float64 = 0.2
	if (up || down) && (left || right) {
		speed *= 0.7
	}

	// apply speed to camera target
	if up    { game.LookAtY -= speed }
	if down  { game.LookAtY += speed }
	if left  { game.LookAtX -= speed }
	if right { game.LookAtX += speed }

	// notify new camera target
	mipix.Camera().NotifyCoordinates(game.LookAtX, game.LookAtY)

	return nil
}

func (game *Game) Draw(canvas *ebiten.Image) {
	// fill background
	canvas.Fill(utils.RGB(255, 255, 255))

	// draw 2x2 square centered at (0, 0)
	camArea := mipix.Camera().Area()
	centerRect := utils.Rect(-1, -1, 1, 1)
	if centerRect.Overlaps(camArea) {
		drawRect := centerRect.Sub(camArea.Min)
		utils.FillOverRect(canvas, drawRect, utils.RGB(200, 0, 200))
	}
}

func main() {
	mipix.SetResolution(128, 72)
	err := mipix.Run(&Game{})
	if err != nil { panic(err) }
}
```

The camera following behavior is defined by the [`Tracker`](https://pkg.go.dev/github.com/tinne26/mipix#AccessorCamera.SetTracker) interface, which you can customize. You can also customize camera zooms and shakes.

## Multi-layered drawing

You can also combine logical draws with high resolution draws when needed. Common uses for high resolution drawing are UI (text in particular), smoothly moving elements and shader effects. The next example includes both text rendering and a smoothly moving player.

The way to achieve these interleaved draws in mipix is to use `mipix.QueueDraw()` and `mipix.QueueHiResDraw()`, which receive the drawing functions as parameters. These queueing functions can be invoked during draw, and are called as additional drawing stages or layers once the current drawing logic is done.

```Golang
package main

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/mipix"
import "github.com/tinne26/mipix/utils"

import "github.com/hajimehoshi/ebiten/v2/text/v2"
import "golang.org/x/image/font/opentype"
import "golang.org/x/image/font"
import "github.com/tinne26/fonts/liberation/lbrtsans"

// Helper type for decorative tiles
type Grass struct { X, Y int }
func (g Grass) Draw(canvas *ebiten.Image, cameraArea utils.Rectangle) {
	rect := utils.Rect(g.X*5, g.Y*5, g.X*5 + 5, g.Y*5 + 5)
	if rect.Overlaps(cameraArea) {
		fillRect := rect.Sub(cameraArea.Min)
		utils.FillOverRect(canvas, fillRect, utils.RGB(83, 141, 106))
	}
}

// Main game struct
type Game struct {
	PlayerCX, PlayerCY float64
	GrassTiles []Grass
	FontFace text.Face
	FontSize float64
}

// Same concept as the camera movement example, nothing new here
func (game *Game) Update() error {
	// detect directions
	up    := ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	down  := ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	left  := ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	right := ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	if up   && down  { up  , down  = false, false }
	if left && right { left, right = false, false }
	
	// apply diagonal speed reduction if needed
	var speed float64 = 0.2
	if (up || down) && (left || right) {
		speed *= 0.7
	}

	// apply speed to camera target
	if up    { game.PlayerCY -= speed }
	if down  { game.PlayerCY += speed }
	if left  { game.PlayerCX -= speed }
	if right { game.PlayerCX += speed }

	// notify new camera target
	mipix.Camera().NotifyCoordinates(game.PlayerCX, game.PlayerCY)

	return nil
}

// This draw function exemplifies how to combine low and
// high resolution draws.
func (game *Game) Draw(canvas *ebiten.Image) {
	// fill background
	canvas.Fill(utils.RGB(128, 207, 169))

	// draw grass on the logical canvas
	cameraArea := mipix.Camera().Area()
	for _, grass := range game.GrassTiles {
		grass.Draw(canvas, cameraArea)
	}

	// queue draw for player rect at high resolution
	mipix.QueueHiResDraw(func(_, hiResCanvas *ebiten.Image) {
		ox, oy := game.PlayerCX - 1.5, game.PlayerCY - 1.5
		fx, fy := game.PlayerCX + 1.5, game.PlayerCY + 1.5
		rgba := utils.RGB(66, 67, 66)
		mipix.HiRes().FillOverRect(hiResCanvas, ox, oy, fx, fy, rgba)
	})

	// you could interleave more mipix.QueueDraw()
	// low-res draws here if you needed that

	// queue text rendering on high resolution too
	mipix.QueueHiResDraw(game.DrawText)
}

// High resolution text rendering function
func (game *Game) DrawText(_, hiResCanvas *ebiten.Image) {
	// determine text size
	bounds := hiResCanvas.Bounds()
	height := float64(bounds.Dy())
	fontSize := height/10.0

	// (re)initialize font face if necessary
	if game.FontSize != fontSize {
		var opts opentype.FaceOptions
		opts.DPI = 72.0
		opts.Size = fontSize
		opts.Hinting = font.HintingFull
		face, err := opentype.NewFace(lbrtsans.Font(), &opts)
		game.FontFace = text.NewGoXFace(face)
		game.FontSize = fontSize
		if err != nil { panic(err) }
	}

	// draw text
	var textOpts text.DrawOptions
	textOpts.PrimaryAlign = text.AlignCenter
	ox, oy := float64(bounds.Min.X), float64(bounds.Min.Y)
	textOpts.GeoM.Translate(ox + float64(bounds.Dx())/2.0, oy + (height - height/6.0))
	textOpts.ColorScale.ScaleWithColor(utils.RGB(30, 51, 39))
	textOpts.Blend = ebiten.BlendLighter
	text.Draw(hiResCanvas, "NOTHINGNESS AWAITS", game.FontFace, &textOpts)
}

func main() {
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	mipix.SetResolution(128, 72)
	err := mipix.Run(&Game{
		GrassTiles: []Grass{ // add some little decoration
			{-6, -6}, {8, -6}, {9, -6}, {-2, -5}, {4, -5}, {8, -5}, {-5, -4}, {-1, -4}, {2, -4},
			{-5, -3}, {-4, -3}, {-2, -3}, {-1, -3}, {1, -3}, {2, -3}, {3, -3}, 
			{-6, -2}, {-5, -2}, {-4, -2}, {-3, -2}, {-1, -2}, {0, -2}, {1, -2}, {2, -2}, {3, -2}, 
			{-4, -1}, {-3, -1}, {-2, -1}, {-1, -1}, {0, -1}, {1, -1}, {4, -1}, {-9, -1},
			{-5, 0}, {-3, 0}, {-2, 0}, {-1, 0}, {0, 0}, {1, 0}, {2, 0}, {-10, 0}, {-9, 0}, {-8, 0}, 
			{-3, 1}, {-2, 1}, {-1, 1}, {1, 1}, {2, 1}, {3, 1}, {-9, 1}, {-8, 1},
			{-2, 2}, {0, 2}, {1, 2}, {5, 2}, {-10, 2}, {-8, 2}, {-7, 2},
			{-3, 3}, {1, 3}, {3, 3}, {4, 3}, {5, 3}, {-8, 3},
			{3, 4}, {4, 4}, {-6, 5}, {11, 0}, {12, 1}, {12, 2},
		},
	})
	if err != nil { panic(err) }
}
```

> [!TIP]
> Example above can be run from your terminal with:
> `go run github.com/tinne26/mipix-examples/src/tutorial/multi_layered`

It's important to notice that combining logical and high resolution draws has some caveats: some elements that are contiguous in logical space might display slight gaps after interleaving low and high resolution draws (due to necessary internal projections). I might implement some techniques to avoid this in the future, but they would be opt-in and expensive, as they can basically only be post-corrections.

## This is enough

While mipix has quite a few more features, they aren't particularly important to discuss here; you can figure them out by taking a [look at the API](https://pkg.go.dev/github.com/tinne26/mipix). The drawing model is the only essential part to understand when using mipix, and that has been reasonably covered already.

If you need more code samples, the [mipix-examples](https://github.com/tinne26/mipix-examples) contains a few more programs and even small games that you can try online. Hopefully all this will be enough to get you started, but if you have any additional doubts or questions, don't hesitate to reach out! I should be fairly responsive both on Github discussions and Ebitengine's discord!

