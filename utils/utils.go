package utils

import (
	"image"
	"image/color"

	"github.com/edwinsyarief/mipix/internal"
	"github.com/hajimehoshi/ebiten/v2"
)

// Alias for image.Rectangle.
type Rectangle = image.Rectangle

// Syntax sugar for [ebiten.Image.SubImage]() passing explicit
// coordinates instead of [image.Rectangle] and returning [*ebiten.Image]
// instead of [image.Image].
func SubImage(source *ebiten.Image, minX, minY, maxX, maxY int) *ebiten.Image {
	return source.SubImage(Rect(minX, minY, maxX, maxY)).(*ebiten.Image)
}

// Similar to [ebiten.Image.Fill](), but with alpha blending.
// See also [FillOverRect]().
func FillOver(target *ebiten.Image, fillColor color.Color) {
	internal.FillOver(target, fillColor)
}

// Create a low resolution image from a simple mask. The value
// 0 is always reserved for transparent, and higher values will
// index the given colors. If no colors are given, 1 will be
// white by default. Example usage:
//
//	heart := utils.MaskToImage([]uint8{
//	    0, 1, 1, 0, 1, 1, 0,
//	    1, 1, 1, 1, 1, 1, 1,
//	    1, 1, 1, 1, 1, 1, 1,
//	    0, 1, 1, 1, 1, 1, 0,
//	    0, 0, 1, 1, 1, 0, 0,
//	    0, 0, 0, 1, 0, 0, 0,
//	}, 7, utils.RGB(255, 0, 0))
func MaskToImage(width int, mask []uint8, colors ...color.RGBA) *ebiten.Image {
	return MaskToImageWithOrigin(0, 0, width, mask, colors...)
}

// Same as [MaskToImage](), but additionally receiving the image
// bounds origin coordinates as the first two arguments.
func MaskToImageWithOrigin(ox, oy int, width int, mask []uint8, colors ...color.RGBA) *ebiten.Image {
	// safety assertions
	if width <= 0 {
		panic("expected width > 0")
	}
	height := len(mask) / width
	if height*width != len(mask) {
		panic("given width can't split given mask into rows of equal length")
	}

	// no colors fallback
	if len(colors) == 0 {
		colors = []color.RGBA{{255, 255, 255, 255}}
	}

	// create image
	rgba := image.NewRGBA(image.Rect(ox, oy, ox+width, oy+height))
	for index, value := range mask {
		pixelIndex := index << 2
		if value != 0 {
			clr := colors[value-1]
			rgba.Pix[pixelIndex+0] = clr.R
			rgba.Pix[pixelIndex+1] = clr.G
			rgba.Pix[pixelIndex+2] = clr.B
			rgba.Pix[pixelIndex+3] = clr.A
		}
	}

	var opts ebiten.NewImageFromImageOptions
	opts.PreserveBounds = true
	return ebiten.NewImageFromImageWithOptions(rgba, &opts)
}

// Returns the given bounds translated by the given (x, y) values.
// Equivalent to bounds.Add(image.Pt(x, y)).
func Shift(bounds image.Rectangle, x, y int) image.Rectangle {
	return bounds.Add(image.Pt(x, y))
}

// Returns the GeoM that would be used to draw the given image
// on the logical ebipixel canvas at the logical global coordinates
// (x, y).
func GeoMAt(source *ebiten.Image, x, y int) ebiten.GeoM {
	var geom ebiten.GeoM
	localXY := image.Pt(x, y).Sub(internal.BridgedCameraOrigin)
	localXY = localXY.Add(source.Bounds().Min) // *
	// * origin is not automatically applied when using
	//   an image as source, so we need to add it manually
	geom.Translate(float64(localXY.X), float64(localXY.Y))
	return geom
}

// Returns the image options with a GeoM set up to draw the
// given image at the logical global coordinates (x, y).
// Makes basic image drawing simpler. Example code:
//
//	opts := utils.DrawImageOptionsAt(myImage, 8, 8)
//	canvas.DrawImage(myImage, &opts)
func DrawImageOptionsAt(source *ebiten.Image, x, y int) ebiten.DrawImageOptions {
	var opts ebiten.DrawImageOptions
	localXY := image.Pt(x, y).Sub(internal.BridgedCameraOrigin)
	localXY = localXY.Add(source.Bounds().Min) // *
	// * origin is not automatically applied when using
	//   an image as source, so we need to add it manually
	opts.GeoM.Translate(float64(localXY.X), float64(localXY.Y))
	return opts
}

// Similar to [ebiten.Image.Fill](), but with alpha blending
// and explicit target bounds. See also [FillOver]().
func FillOverRect(target *ebiten.Image, bounds image.Rectangle, fillColor color.Color) {
	internal.FillOverRect(target, bounds, fillColor)
}

// (we already have HiRes().FillOverRect() and stuff)
// func FillOverRectF64(target *ebiten.Image, minX, minY, maxX, maxY float64, fillColor color.Color) {
// 	internal.FillOverRectF32(target, float32(minX), float32(minY), float32(maxX), float32(maxY), fillColor)
// }

// func FillOverRectF32(target *ebiten.Image, minX, minY, maxX, maxY float32, fillColor color.Color) {
// 	internal.FillOverRectF32(target, minX, minY, maxX, maxY, fillColor)
// }

// Alias for [image.Rect]().
func Rect(minX, minY, maxX, maxY int) image.Rectangle {
	return image.Rect(minX, minY, maxX, maxY)
}

// func RoundCoords(x, y float64) (int, int) {
// 	return int(math.Round(x)), int(math.Round(y))
// }

// func Round(value float64) float64 {
// 	return math.Round(value)
// }

// Returns [color.RGBA]{r, g, b, 255}.
func RGB(r, g, b uint8) color.RGBA {
	return color.RGBA{r, g, b, 255}
}

// Returns [color.RGBA]{r, g, b, a} after checking that the
// given values constitute a valid premultiplied-alpha color
// (a >= r,g,b). On invalid colors, the function panics.
func RGBA(r, g, b, a uint8) color.RGBA {
	if r > a || g > a || b > a {
		panic("invalid color.RGBA values: premultiplied-alpha requires a >= r,g,b")
	}
	return color.RGBA{r, g, b, a}
}

// Converts a color to float32 RGBA values in [0, 1] range.
//
// This is the format that [ebiten.Vertex] expects.
func ColorToF32(clr color.Color) (r, g, b, a float32) {
	r16, g16, b16, a16 := clr.RGBA()
	return float32(r16) / 65535.0, float32(g16) / 65535.0, float32(b16) / 65535.0, float32(a16) / 65535.0
}

// // Similar to [ebiten.IsKeyPressed](), but allowing multiple keys.
// func AnyKeyPressed(keys ...ebiten.Key) bool {
// 	for _, key := range keys {
// 		if ebiten.IsKeyPressed(key) {
// 			return true
// 		}
// 	}
// 	return false
// }
