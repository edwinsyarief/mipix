# Introduction to mipix

Key ideas to understand mipix's model, plus a few code samples to illustrate each concept.

## Terminology

You may skip this section, it's only a clarification for some people that get confused about some terms:
- *Logical vs high resolution*: when talking about "logical" or "low resolution" spaces, coordinates or canvases, we refer to pure pixel units, where pixel art maps 1-to-1 to our canvas, space or coordinates. In other words: 1 unit of our canvas, space or coordinates match 1 pixel of our art. "High resolution" or "screen" spaces, coordinates and canvases, instead, have an arbitrary size and will require scaling or projections.
- *Filters*: filters in the context of graphics often refer to shader effects like vignetting, lens, color transformations and others. In the context of mipix, though, the documentation always uses the word to refer to *scaling filters* instead, which are all about performing color interpolations for projections. These are completely different things, but it's easy to mix them up due to the same word being used.

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
- Our game struct must satisfy the [`mipix.Game`](https://pkg.go.dev/github.com/tinne26/mipix#Game) interface, which is just like Ebitengine's, but without the `Layout()` method.
- You have to use [`mipix.Run()`](https://pkg.go.dev/github.com/tinne26/mipix#Run) instead of [`ebiten.RunGame()`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#RunGame).
- You must always set the game's resolution before [`mipix.Run()`](https://pkg.go.dev/github.com/tinne26/mipix#Run) through [`mipix.SetResolution()`](https://pkg.go.dev/github.com/tinne26/mipix#SetResolution). More advice on how to choose resolutions on the footnotes[^1].

[^1]: Based on [Steam's surveys](https://store.steampowered.com/hwsurvey) the most common display resolution is by far 1920x1080 (full HD), with over 50% share as of 2024. If you are going for a 16:9 aspect ratio, then you should choose a resolution that's a divisor of 1920x1080, like 128x72, 160x90, 320x180, 384x216 (\*), 480x270 (\*), 640x360, 960x540 (\*) and so on (the ones marked with (\*) are not perfectly compatible with QHD). For other aspect ratios, you will typically go more square-ish (less wide), in which case black bars on the sides are unavoidable on common displays. Here you should try to stick to common divisors of the most used vertical resolutions instead. For example, 360 is the greatest common divisor of 1080 (full HD) and 1440 (QHD), so that would be a great choice. Other divisors like 180, 120, 90 and 60 would be equally good choices. This is just basic advice, there are some exceptions to these rules.

## The draw model

While the `Update()` method should be coded pretty much in the same way for both raw Ebitengine and mipix, the `Draw()` method is a different story.

Here are the two key points you really have to engrave in your soul:
- The canvas you receive on `Draw()` represents pixels 1-to-1. Your pixel art must be drawn directly to the canvas, in its original size. Forget about display scaling factors and projections... *Just draw your pixel art*.
- The canvas you receive on `Draw()` does *not* always have the same size, and it does *not* necessarily match the resolution you set for your game with `mipix.SetResolution(W, H)`. This can happen due to zoom effects (which mipix handles internally), but also something as simple as moving around. For example: even if your resolution is 128x72, if you move around in any direction you might have to render half a pixel for one of the borders of the canvas and another half for the opposite one.  In these cases, even without zoom, mipix would ask you to draw a 129x72, 128x73 or 129x73 area. In conclusion: you should not be thinking about "render my WxH canvas", but "render all the logical area requested by mipix" instead (which is given by `mipix.Camera().Area()`).
- You can still render *some* elements at decimal positions and in high resolution, but we will touch on that later.

Summarizing: draw things one-to-one, don't assume a specific canvas resolution, draw the area that mipix asks you to.

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

Notice that the image will be drawn at the center of the screen in this example. This is because the camera target defaults to (0, 0), and our image is 6x6 but we set its origin at (-3, -3).

You can shorten the code using some [`mipix/utils`](https://pkg.go.dev/github.com/tinne26/mipix/utils) helpers, but make sure to understand the general approach first.
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

Most of mipix's functionality is grouped on "accessors". Accessors are just dummy types used to group together related functionality. The most important accessor is the [`AccessorCamera`](https://pkg.go.dev/github.com/tinne26/mipix#AccessorCamera), which gives you access to zoom, shakes, camera position updates and the most important of all: the rect of the currently visible area, which is what we were already using in previous examples to know what to draw.

So now let's see how to make the camera move. For this example, the `Update()` method will implement some basic movement logic, while the `Draw()` method will just render a small square at (0, 0) that can serve as a visual reference while we move around.

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

## Cursor and touch positions

When you are using mipix, mipix is requesting Ebitengine the highest resolution canvas it can possibly get, and then handling scaling internally, trying to not bother you with it. Unfortunately, there are still *some* visible side effects to this internal trickery. In particular, `ebiten.CursorPosition()` and screen touches will stop behaving as you might expect, since they will return coordinates for a high resolution screen that's kinda outside your mental model when working with mipix.

To make life easier in the face of all this, mipix exposes an [`AccessorConvert`](https://pkg.go.dev/github.com/tinne26/mipix#AccessorConvert) with multiple functions to convert coordinates from high resolution to your logical space.

```Golang
package main

import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/mipix"

type Game struct {
	HiCursorX, HiCursorY int
	LoCursorX, LoCursorY float64
	LoRelativeX, LoRelativeY float64
	LoGameX, LoGameY float64
}

func (game *Game) Update() error {
	hiX, hiY := ebiten.CursorPosition()
	loX, loY := mipix.Convert().ToLogicalCoords(hiX, hiY)
	reX, reY := mipix.Convert().ToRelativeCoords(hiX, hiY)
	gmX, gmY := mipix.Convert().ToGameResolution(hiX, hiY)
	game.HiCursorX, game.HiCursorY = hiX, hiY
	game.LoCursorX, game.LoCursorY = loX, loY
	game.LoRelativeX, game.LoRelativeY = reX, reY
	game.LoGameX, game.LoGameY = gmX, gmY
	return nil
}

func (game *Game) Draw(canvas *ebiten.Image) {
	canvas.Fill(color.RGBA{128, 128, 128, 255})
	mipix.Debug().Drawf("[ Cursor Position ]")
	mipix.Debug().Drawf("High-res screen: (%d, %d)", game.HiCursorX, game.HiCursorY)
	mipix.Debug().Drawf("Low-res relative: (%.02f, %.02f)", game.LoRelativeX, game.LoRelativeY)
	mipix.Debug().Drawf("Low-res screen: (%.02f, %.02f)", game.LoGameX, game.LoGameY)
	mipix.Debug().Drawf("Low-res global: (%.02f, %.02f)", game.LoCursorX, game.LoCursorY)
}

func main() {
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	mipix.SetResolution(100, 100)
	err := mipix.Run(&Game{})
	if err != nil { panic(err) }
}
```

> [!TIP]
> *You can try this example directly from your terminal with:  
> `go run github.com/tinne26/mipix-examples/src/tutorial/cursor_position@latest`*

In this example, you can also see some debug functions, which are very useful for quick information display, both within the game or on your terminal.

## Multi-layered drawing

While mipix focuses on low resolution rendering, you can also combine logical draws with high resolution draws when needed. Common uses for high resolution drawing are vectorial text rendering and UI, smoothly moving elements and shader effects. The next example includes both text rendering and a smoothly moving player.

In order to make these interleaved draws, mipix exposes [`mipix.QueueDraw()`](https://pkg.go.dev/github.com/tinne26/mipix#QueueDraw) and [`mipix.QueueHiResDraw()`](https://pkg.go.dev/github.com/tinne26/mipix#QueueHiResDraw), which receive the drawing functions as parameters. These queueing functions can be invoked at any point of the draw stage, and will be triggered in order once the current drawing logic finishes.

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

func (game *Game) Update() error {
	// elided for brevity: same as the camera example but
	// using PlayerCX/PlayerCY instead of LookAtX/LookAtY
}

// This draw function exemplifies how to combine low
// and high resolution draws with QueueHiResDraw()
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
	// low-res draws here if you needed it

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
> *You can try this example directly from your terminal with:  
> `go run github.com/tinne26/mipix-examples/src/tutorial/multi_layered@latest`*

It's important to notice that combining logical and high resolution draws has some caveats: some elements that are contiguous in logical space might display slight gaps after interleaving low and high resolution draws (due to necessary internal projections). I might implement some techniques to avoid this in the future, but they would be opt-in and expensive, as they can basically only be applied as post-corrections.

## This is enough for today

While mipix still packs a few more features, they aren't particularly important to discuss here; you can figure them out by taking a [look at the API docs](https://pkg.go.dev/github.com/tinne26/mipix). The drawing model is the only essential part to understand when using mipix, and I think that has been reasonably covered already. If you need more code samples, the [mipix-examples](https://github.com/tinne26/mipix-examples) repository contains a few more programs and even small games that you can try online.

Hopefully this should be enough to get you started, but if you have any other questions don't hesitate to reach out! I should be fairly responsive both on Github discussions and Ebitengine's discord.

