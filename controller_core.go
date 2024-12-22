package mipix

import (
	"image"
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/mipix/internal"
	"github.com/edwinsyarief/mipix/tracker"
	"github.com/edwinsyarief/mipix/utils"
	"github.com/edwinsyarief/mipix/zoomer"
	"github.com/hajimehoshi/ebiten/v2"
	_ "github.com/silbinarywolf/preferdiscretegpu"
)

var pkgController controller

func init() {
	pkgController.cameraZoomReset(1.0)
	pkgController.tickSetRate(1)
	pkgController.shakerChannels = make([]shakerChannel, 1)
	pkgController.lastFlushCoordinatesTick = 0xFFFF_FFFF_FFFF_FFFF
	pkgController.bestFitRenderSize = ebimath.V(180, 180)
	pkgController.bestFitContextSize = ebimath.V(1000, 1000)
	pkgController.needsRedraw = true
}

type controller struct {
	// core state
	game                  Game
	queuedDraws           []queuedDraw
	reusableCanvas        *ebiten.Image // this preserves the highest size requested by resolution or zooms
	logicalWidth          int
	logicalHeight         int
	hiResWidth            int
	hiResHeight           int
	prevHiResCanvasWidth  int // used to update layoutHasChanged even on unexpected cases *
	prevHiResCanvasHeight int // used to update layoutHasChanged even on unexpected cases
	// * https://github.com/hajimehoshi/ebiten/issues/2978
	layoutHasChanged   bool
	inDraw             bool
	redrawManaged      bool
	needsRedraw        bool
	needsClear         bool
	stretchingEnabled  bool
	keepAspectRatio    bool
	dynamicScaling     bool
	scalingFilter      ScalingFilter
	bestFitRenderSize  ebimath.Vector
	bestFitContextSize ebimath.Vector

	// camera
	lastFlushCoordinatesTick uint64
	cameraArea               image.Rectangle

	// tracking
	tracker           tracker.Tracker
	trackerCurrentX   float64
	trackerCurrentY   float64
	trackerTargetX    float64
	trackerTargetY    float64
	trackerPrevSpeedX float64
	trackerPrevSpeedY float64

	// zoom
	zoomer      zoomer.Zoomer
	zoomCurrent float64
	zoomTarget  float64

	// shake
	shakerChannels []shakerChannel
	shakerOffsetX  float64
	shakerOffsetY  float64

	// ticks
	currentTick uint64
	tickRate    uint64

	// shaders
	shaderOpts        ebiten.DrawTrianglesShaderOptions
	shaderVertices    []ebiten.Vertex
	shaderVertIndices []uint16
	shaders           [scalingFilterEndSentinel]*ebiten.Shader

	// debug
	debugInfo      []string
	debugOffscreen *Offscreen
}

// --- ebiten.Game implementation ---

func (self *controller) Update() error {
	self.currentTick += self.tickRate
	err := self.game.Update()
	if err != nil {
		return err
	}
	self.cameraFlushCoordinates()
	self.layoutHasChanged = false
	return nil
}

func (self *controller) Draw(hiResCanvas *ebiten.Image) {
	self.inDraw = true

	// get bounds and update hi res canvas size
	hiResBounds := hiResCanvas.Bounds()
	hiResWidth, hiResHeight := hiResBounds.Dx(), hiResBounds.Dy()
	if hiResWidth != self.prevHiResCanvasWidth || hiResHeight != self.prevHiResCanvasHeight {
		// * TODO: all this is kind of a temporary hack until ebitengine
		//   can guarantee that layout returned sizes and draw received
		//   canvas sizes will match
		self.prevHiResCanvasWidth = hiResWidth
		self.prevHiResCanvasHeight = hiResHeight
		self.layoutHasChanged = true
		self.needsRedraw = true
	}

	logicalCanvas := self.getLogicalCanvas()
	activeCanvas := self.getActiveHiResCanvas(hiResCanvas)
	if self.needsClear {
		self.needsClear = false
		hiResCanvas.Clear()
		logicalCanvas.Clear()
	}
	self.game.Draw(logicalCanvas)

	var drawIndex int = 0
	var prevDrawWasHiRes bool = false
	for drawIndex < len(self.queuedDraws) {
		if self.queuedDraws[drawIndex].IsHighResolution() {
			if !prevDrawWasHiRes {
				self.projectLogical(logicalCanvas, activeCanvas)
			}
			self.queuedDraws[drawIndex].hiResFunc(hiResCanvas, activeCanvas)
			prevDrawWasHiRes = true
		} else {
			if prevDrawWasHiRes {
				logicalCanvas.Clear()
				prevDrawWasHiRes = false
			}
			self.queuedDraws[drawIndex].logicalFunc(logicalCanvas)
		}
		drawIndex += 1
	}
	self.queuedDraws = self.queuedDraws[:0]

	// final projection
	if !self.redrawManaged || self.needsRedraw {
		if !prevDrawWasHiRes {
			self.projectLogical(logicalCanvas, activeCanvas)
		}
		self.debugDrawAll(activeCanvas)
	}
	self.needsRedraw = false
	self.inDraw = false
}

func (self *controller) getLogicalCanvas() *ebiten.Image {
	width := self.cameraArea.Dx()
	height := self.cameraArea.Dy()

	if self.reusableCanvas == nil {
		refWidth := max(width, self.logicalWidth+1)    // +1 because smooth movement will..
		refHeight := max(height, self.logicalHeight+1) // ..force this in most games anyways
		self.reusableCanvas = ebiten.NewImage(refWidth, refHeight)
		return utils.SubImage(self.reusableCanvas, 0, 0, width, height)
	} else {
		bounds := self.reusableCanvas.Bounds()
		availableWidth, availableHeight := bounds.Dx(), bounds.Dy()
		if width == availableWidth && height == availableHeight {
			if ebiten.IsScreenClearedEveryFrame() {
				self.reusableCanvas.Clear() // TODO: is this the best place to do it?
			}
			return self.reusableCanvas
		} else if width <= availableWidth && height <= availableHeight {
			canvas := utils.SubImage(self.reusableCanvas, 0, 0, width, height)
			if ebiten.IsScreenClearedEveryFrame() {
				canvas.Clear()
			} // TODO: is this the best place to do it?
			return canvas
		} else { // insufficient width or height
			// taking into account target zoom will make it easier to pre-request
			// a single bigger canvas without having to thrash the GPU memory
			// continuously through a zoom-out transition.
			zoomTarget := min(max(self.zoomTarget, 0.05), 1.0)
			refWidth := int(math.Ceil(float64(width+1.0) / zoomTarget))
			refHeight := int(math.Ceil(float64(height+1.0) / zoomTarget))
			self.reusableCanvas = ebiten.NewImage(refWidth, refHeight)
			return utils.SubImage(self.reusableCanvas, 0, 0, width, height)
		}
	}
}

func (self *controller) getActiveHiResCanvas(hiResCanvas *ebiten.Image) *ebiten.Image {
	// trivial case if stretching is used
	if self.stretchingEnabled {
		return hiResCanvas
	}

	// crop margins based on aspect ratios
	hiBounds := hiResCanvas.Bounds()
	hiWidth, hiHeight := hiBounds.Dx(), hiBounds.Dy()
	hiAspectRatio := float64(hiWidth) / float64(hiHeight)
	loAspectRatio := float64(self.logicalWidth) / float64(self.logicalHeight)

	switch {
	case hiAspectRatio == loAspectRatio: // just scaling
		return hiResCanvas
	case hiAspectRatio > loAspectRatio: // horz margins
		xMargin := int((float64(hiWidth) - loAspectRatio*float64(hiHeight)) / 2.0)
		return utils.SubImage(hiResCanvas, xMargin, 0, hiWidth-xMargin, hiHeight)
	case loAspectRatio > hiAspectRatio: // vert margins
		yMargin := int((float64(hiHeight) - float64(hiWidth)/loAspectRatio) / 2.0)
		return utils.SubImage(hiResCanvas, 0, yMargin, hiWidth, hiHeight-yMargin)
	default:
		panic("unreachable")
	}
}

func (self *controller) Layout(logicWinWidth, logicWinHeight int) (int, int) {
	monitor := ebiten.Monitor()
	scale := monitor.DeviceScaleFactor()
	hiResWidth := int(float64(logicWinWidth) * scale)
	hiResHeight := int(float64(logicWinHeight) * scale)

	if self.stretchingEnabled && self.keepAspectRatio {
		hiResWidth = logicWinWidth
		hiResHeight = logicWinHeight
	}
	if hiResWidth != self.hiResWidth || hiResHeight != self.hiResHeight {
		self.layoutHasChanged = true
		self.needsRedraw = true
		self.hiResWidth, self.hiResHeight = hiResWidth, hiResHeight
	}
	return self.hiResWidth, self.hiResHeight
}

func (self *controller) LayoutF(logicWinWidth, logicWinHeight float64) (float64, float64) {
	monitor := ebiten.Monitor()
	scale := monitor.DeviceScaleFactor()
	outWidth := math.Ceil(logicWinWidth * scale)
	outHeight := math.Ceil(logicWinHeight * scale)

	if self.stretchingEnabled && self.keepAspectRatio {
		outWidth = logicWinWidth
		outHeight = logicWinHeight
	}
	if int(outWidth) != self.hiResWidth || int(outHeight) != self.hiResHeight {
		self.layoutHasChanged = true
		self.needsRedraw = true
		self.hiResWidth, self.hiResHeight = int(outWidth), int(outHeight)
	}
	return outWidth, outHeight
}

// --- run and queued draws ---

func (self *controller) run(game Game) error {
	self.game = game
	if self.logicalWidth == 0 || self.logicalHeight == 0 {
		panic("must set the game resolution with mipix.SetResolution(width, height) before mipix.Run()")
	}
	self.trackerCurrentX = self.trackerTargetX
	self.trackerCurrentY = self.trackerTargetY
	return ebiten.RunGame(self)
}

// --- resolution ---

func (self *controller) getResolution() (width, height int) {
	return self.logicalWidth, self.logicalHeight
}

func (self *controller) setResolution(width, height int) {
	if self.inDraw {
		panic("can't change resolution during draw stage")
	}
	if width < 1 || height < 1 {
		panic("game resolution must be at least (1, 1)")
	}
	if width != self.logicalWidth || height != self.logicalHeight {
		self.needsRedraw = true
		self.logicalWidth, self.logicalHeight = width, height
		internal.BridgedLogicalWidth, internal.BridgedLogicalHeight = width, height // hyper massive hack
		self.updateCameraArea()
	}
}

func (self *controller) setBestFitRenderSize(width, height int) {
	if self.inDraw {
		panic("can't change resolution during draw stage")
	}
	if width < 1 || height < 1 {
		panic("game resolution must be at least (1, 1)")
	}
	if self.stretchingEnabled && self.keepAspectRatio &&
		(width != int(self.bestFitRenderSize.X) || height != int(self.bestFitRenderSize.Y)) {
		self.bestFitRenderSize = ebimath.V(float64(width), float64(height))
		self.needsRedraw = true
	}

}

func (self *controller) setBestFitContextSize(width, height int) {
	if self.inDraw {
		panic("can't change resolution during draw stage")
	}
	if width < 1 || height < 1 {
		panic("game resolution must be at least (1, 1)")
	}
	if self.stretchingEnabled && self.keepAspectRatio &&
		(width != int(self.bestFitContextSize.X) || height != int(self.bestFitContextSize.Y)) {
		self.bestFitContextSize = ebimath.V(float64(width), float64(height))
		self.needsRedraw = true
	}
}

// --- scaling ---

func (self *controller) scalingSetFilter(filter ScalingFilter) {
	if self.inDraw {
		panic("can't change scaling filter during draw stage")
	}
	if filter != self.scalingFilter {
		self.needsRedraw = true
		self.scalingFilter = filter
	}
	if self.shaders[filter] == nil {
		self.compileShader(filter)
	}
}

func (self *controller) scalingGetFilter() ScalingFilter {
	return self.scalingFilter
}

func (self *controller) scalingSetStretchingAllowed(allowed, keepAspectRatio, dynamicScaling bool) {
	if self.inDraw {
		panic("can't change stretching mode during draw stage")
	}
	if allowed != self.stretchingEnabled {
		self.needsRedraw = true
		self.stretchingEnabled = allowed
		self.keepAspectRatio = keepAspectRatio
		self.dynamicScaling = dynamicScaling
		if !allowed {
			self.needsClear = true
		}
	}
}

func (self *controller) scalingGetStretchingAllowed() bool {
	return self.stretchingEnabled
}

// --- redraw ---

func (self *controller) redrawSetManaged(managed bool) {
	if self.inDraw {
		panic("can't change redraw management during draw stage")
	}
	self.redrawManaged = managed
}

func (self *controller) redrawIsManaged() bool {
	return self.redrawManaged
}

func (self *controller) redrawRequest() {
	if self.inDraw {
		panic("can't request redraw during draw stage")
	}
	self.needsRedraw = true
}

func (self *controller) redrawPending() bool {
	return self.needsRedraw || !self.redrawManaged
}

func (self *controller) redrawScheduleClear() {
	self.needsClear = true
}

// --- hi res ---

func (self *controller) hiResDraw(target, source *ebiten.Image, transform *ebimath.Transform) {
	if !self.inDraw {
		panic("can't mipix.HiRes().Draw() outside draw stage")
	}
	self.internalHiResDraw(target, source, transform)
}

func (self *controller) hiResFillOverRect(target *ebiten.Image, minX, minY, maxX, maxY float64, fillColor color.Color) {
	targetBounds := target.Bounds()
	targetWidth, targetHeight := float64(targetBounds.Dx()), float64(targetBounds.Dy())
	hrMinX, hrMinY := self.logicalToHiResCanvasCoords(minX, minY, targetWidth, targetHeight)
	hrMaxX, hrMaxY := self.logicalToHiResCanvasCoords(maxX, maxY, targetWidth, targetHeight)
	ox, oy := float64(targetBounds.Min.X), float64(targetBounds.Min.Y)

	xl, xr := float32(hrMinX+ox), float32(hrMaxX+ox)
	yt, yb := float32(hrMinY+oy), float32(hrMaxY+oy)
	internal.FillOverRectF32(target, xl, yt, xr, yb, fillColor)
}

// TODO: something like this is not only important, but should be exposed directly,
// both for low res and high res, honestly. Or on Convert().
func (self *controller) logicalToHiResCanvasCoords(x, y, targetWidth, targetHeight float64) (float64, float64) {
	camMinX, camMinY, camMaxX, camMaxY := self.cameraAreaF64()
	x64, y64 := float64(x), float64(y)
	xOffset, yOffset := x64-camMinX, y64-camMinY
	return targetWidth * (xOffset / (camMaxX - camMinX)), targetHeight * (yOffset / (camMaxY - camMinY))
}

func (self *controller) internalHiResDraw(target, source *ebiten.Image, transform *ebimath.Transform) {
	// view culling
	camMinX, camMinY, camMaxX, camMaxY := self.cameraAreaF64()
	t := transform
	realPos := ebimath.V2(0).Apply(t.Matrix())

	if realPos.X > camMaxX || realPos.Y > camMaxY {
		return // outside view
	}

	sourceBounds := source.Bounds()
	sourceWidth, sourceHeight := float64(sourceBounds.Dx()), float64(sourceBounds.Dy())
	if realPos.X+sourceWidth < camMinX {
		return // outside view
	}
	if realPos.Y+sourceHeight < camMinY {
		return // outside view
	}

	// compile shader if necessary
	if self.shaders[self.scalingFilter] == nil {
		self.compileShader(self.scalingFilter)
	}

	// set triangle vertex coordinates
	targetBounds := target.Bounds()
	targetMinX, targetMinY := float64(targetBounds.Min.X), float64(targetBounds.Min.Y)
	targetWidth, targetHeight := float64(targetBounds.Dx()), float64(targetBounds.Dy())
	xFactor := self.zoomCurrent * targetWidth / float64(self.logicalWidth)
	yFactor := self.zoomCurrent * targetHeight / float64(self.logicalHeight)
	if self.stretchingEnabled && self.keepAspectRatio {
		if self.stretchingEnabled && self.keepAspectRatio {
			scale := internal.BestFitFloat(
				self.dynamicScaling,
				self.hiResWidth,
				self.hiResHeight,
				self.bestFitRenderSize.X,
				&self.bestFitRenderSize.Y,
				&self.bestFitContextSize.X,
				&self.bestFitContextSize.Y, true)
			xFactor = scale
			yFactor = scale
		}
	}

	srcProjMinX := realPos.X * xFactor
	srcProjMinY := realPos.Y * yFactor
	srcProjMaxX := srcProjMinX + sourceWidth*xFactor*t.Scale().X
	srcProjMaxY := srcProjMinY + sourceHeight*yFactor*t.Scale().Y
	left, right := float32(targetMinX+srcProjMinX), float32(targetMinX+srcProjMaxX)
	top, bottom := float32(targetMinY+srcProjMinY), float32(targetMinY+srcProjMaxY)

	p0 := ebimath.V(float64(left), float64(top))
	p1 := ebimath.V(float64(right), p0.Y)
	p2 := ebimath.V(p1.X, float64(bottom))
	p3 := ebimath.V(p0.X, p2.Y)

	if t.Rotation() != 0 {
		srcOffset := ebimath.V(srcProjMinX, srcProjMinY)
		p0 = p0.RotateAround(srcOffset, t.Rotation())
		p1 = p1.RotateAround(srcOffset, t.Rotation())
		p2 = p2.RotateAround(srcOffset, t.Rotation())
		p3 = p3.RotateAround(srcOffset, t.Rotation())
	}

	self.shaderVertices[0].DstX = float32(p0.X)
	self.shaderVertices[0].DstY = float32(p0.Y)
	self.shaderVertices[1].DstX = float32(p1.X)
	self.shaderVertices[1].DstY = float32(p1.Y)
	self.shaderVertices[2].DstX = float32(p2.X)
	self.shaderVertices[2].DstY = float32(p2.Y)
	self.shaderVertices[3].DstX = float32(p3.X)
	self.shaderVertices[3].DstY = float32(p3.Y)

	self.shaderVertices[0].SrcX = float32(sourceBounds.Min.X)
	self.shaderVertices[0].SrcY = float32(sourceBounds.Min.Y)
	self.shaderVertices[1].SrcX = float32(sourceBounds.Max.X)
	self.shaderVertices[1].SrcY = self.shaderVertices[0].SrcY
	self.shaderVertices[2].SrcX = self.shaderVertices[1].SrcX
	self.shaderVertices[2].SrcY = float32(sourceBounds.Max.Y)
	self.shaderVertices[3].SrcX = self.shaderVertices[0].SrcX
	self.shaderVertices[3].SrcY = self.shaderVertices[2].SrcY

	self.shaderOpts.Images[0] = source
	self.shaderOpts.Uniforms["SourceRelativeTextureUnitX"] = float32(float64(self.logicalWidth) / targetWidth)
	self.shaderOpts.Uniforms["SourceRelativeTextureUnitY"] = float32(float64(self.logicalHeight) / targetHeight)
	target.DrawTrianglesShader(
		self.shaderVertices, self.shaderVertIndices,
		self.shaders[self.scalingFilter], &self.shaderOpts,
	)
	self.shaderOpts.Images[0] = nil
}
