package mipix

import (
	"image"
	"math"

	"github.com/edwinsyarief/mipix/internal"
	"github.com/edwinsyarief/mipix/shaker"
	"github.com/edwinsyarief/mipix/tracker"
	"github.com/edwinsyarief/mipix/zoomer"
)

func (self *controller) cameraAreaGet() image.Rectangle {
	return self.cameraArea
}

func (self *controller) cameraAreaF64() (minX, minY, maxX, maxY float64) {
	zoomedWidth := float64(self.logicalWidth) / self.zoomCurrent
	zoomedHeight := float64(self.logicalHeight) / self.zoomCurrent
	if self.stretchingEnabled && self.keepAspectRatio {
		scale := internal.BestFitFloat(
			self.dynamicScaling,
			self.hiResWidth,
			self.hiResHeight,
			self.bestFitRenderSize.X,
			&self.bestFitRenderSize.Y,
			&self.bestFitContextSize.X,
			&self.bestFitContextSize.Y, true)

		zoomedWidth = float64(self.hiResWidth) / scale / self.zoomCurrent
		zoomedHeight = float64(self.hiResHeight) / scale / self.zoomCurrent
	}
	minX = self.trackerCurrentX - zoomedWidth/2.0 + self.shakerOffsetX
	minY = self.trackerCurrentY - zoomedHeight/2.0 + self.shakerOffsetY
	return minX, minY, minX + zoomedWidth, minY + zoomedHeight
}

func (self *controller) updateCameraArea() {
	minX, minY, maxX, maxY := self.cameraAreaF64()
	self.cameraArea = image.Rect(
		int(math.Floor(minX)), int(math.Floor(minY)),
		int(math.Ceil(maxX)), int(math.Ceil(maxY)),
	)
	internal.BridgedCameraOrigin = self.cameraArea.Min
}

// ---- tracking ----

func (self *controller) cameraGetTracker() tracker.Tracker {
	return self.tracker
}

func (self *controller) cameraSetTracker(tracker tracker.Tracker) {
	if self.inDraw {
		panic("can't set tracker during draw stage")
	}
	self.tracker = tracker
}

func (self *controller) cameraNotifyCoordinates(x, y float64) {
	if self.inDraw {
		panic("can't notify tracking coordinates during draw stage")
	}
	self.trackerTargetX, self.trackerTargetY = x, y
}

func (self *controller) cameraResetCoordinates(x, y float64) {
	if self.inDraw {
		panic("can't reset camera coordinates during draw stage")
	}
	self.trackerTargetX, self.trackerTargetY = x, y
	self.trackerCurrentX, self.trackerCurrentY = x, y
	if self.redrawManaged && (x != self.trackerCurrentX || y != self.trackerCurrentY) {
		self.needsRedraw = true
	}
	self.updateCameraArea()
}

func (self *controller) cameraFlushCoordinates() {
	if self.lastFlushCoordinatesTick == self.currentTick {
		return
	}
	self.lastFlushCoordinatesTick = self.currentTick
	self.updateZoom()
	self.updateTracking()
	self.updateShake()
	self.updateCameraArea()
}

func (self *controller) updateTracking() {
	camTracker := self.cameraGetInternalTracker()
	changeX, changeY := camTracker.Update(
		self.trackerCurrentX, self.trackerCurrentY,
		self.trackerTargetX, self.trackerTargetY,
		self.trackerPrevSpeedX, self.trackerPrevSpeedY,
	)
	self.trackerCurrentX += changeX
	self.trackerCurrentY += changeY
	updateDelta := 1.0 / float64(Tick().UPS())
	self.trackerPrevSpeedX = changeX / updateDelta
	self.trackerPrevSpeedY = changeY / updateDelta

	if self.redrawManaged && (self.trackerPrevSpeedX != 0 || self.trackerPrevSpeedY != 0) {
		self.needsRedraw = true
	}
}

func (self *controller) cameraGetInternalTracker() tracker.Tracker {
	if self.tracker != nil {
		return self.tracker
	}
	if defaultTracker == nil {
		defaultTracker = &tracker.SpringTailer{}
		defaultTracker.Spring.SetParameters(0.8, 2.4)
		defaultTracker.SetCatchUpParameters(0.9, 1.75)
	}
	return defaultTracker
}

// --- zoom ---

func (self *controller) updateZoom() {
	zoomer := self.cameraGetInternalZoomer()
	change := zoomer.Update(self.zoomCurrent, self.zoomTarget)
	if math.IsNaN(change) {
		panic("zoomer returned NaN")
	}
	self.zoomCurrent += change
	internal.CurrentZoom = self.zoomCurrent
	if self.zoomCurrent < 0.005 || self.zoomCurrent > 500.0 {
		panic("something is wrong with the zoomer: after last update, zoom went outside [0.005, 500.0]")
	}

	if self.redrawManaged && change != 0 {
		self.needsRedraw = true
	}
}

func (self *controller) cameraGetInternalZoomer() zoomer.Zoomer {
	if self.zoomer != nil {
		return self.zoomer
	}
	if defaultZoomer == nil {
		defaultZoomer = &zoomer.Quadratic{}
		defaultZoomer.Reset()
	}
	return defaultZoomer
}

func (self *controller) updateShake() {
	// compute new offsets
	var offsetX, offsetY float64
	for i := range self.shakerChannels {
		self.shakerChannels[i].Update(i, self.tickRate)
		offsetX += self.shakerChannels[i].offsetX
		offsetY += self.shakerChannels[i].offsetY
	}

	// set needsRedraw flag if necessary
	if self.redrawManaged && (offsetX != self.shakerOffsetX || offsetY != self.shakerOffsetY) {
		self.needsRedraw = true
	}

	// register new offsets
	self.shakerOffsetX = offsetX
	self.shakerOffsetY = offsetY
}

func (self *controller) cameraZoom(newZoomLevel float64) {
	if self.inDraw {
		panic("can't zoom during draw stage")
	}
	self.zoomTarget = newZoomLevel
}

func (self *controller) cameraZoomReset(zoomLevel float64) {
	if self.inDraw {
		panic("can't reset zoom during draw stage")
	}
	self.zoomCurrent, self.zoomTarget, internal.CurrentZoom = zoomLevel, zoomLevel, zoomLevel
	self.cameraGetInternalZoomer().Reset()
}

func (self *controller) cameraGetZoomer() zoomer.Zoomer {
	return self.zoomer
}

func (self *controller) cameraSetZoomer(zoomer zoomer.Zoomer) {
	if self.inDraw {
		panic("can't change zoomer during draw stage")
	}
	self.zoomer = zoomer
}

func (self *controller) cameraGetZoom() (current, target float64) {
	return self.zoomCurrent, self.zoomTarget
}

// ---- screenshake ----

func (self *controller) cameraSetShaker(newShaker shaker.Shaker, channels ...shaker.Channel) {
	if self.inDraw {
		panic("can't SetShaker during draw stage")
	}
	if len(channels) > 1 {
		panic("can't pass multiple shaker channels to SetShaker")
	} else if len(channels) == 0 {
		self.shakerChannels[0].shaker = newShaker
	} else {
		index := int(channels[0])
		if newShaker == nil && index >= len(self.shakerChannels) {
			return
		}
		newChan := shakerChannel{shaker: newShaker}
		self.shakerChannels = setAt(self.shakerChannels, newChan, index)

		// compact nils at the end of the slice
		compactCount := 0
		for i := len(self.shakerChannels) - 1; i > 0; i-- {
			if self.shakerChannels[i].shaker != nil {
				break
			}
			compactCount += 1
		}
		if compactCount > 0 {
			self.shakerChannels = self.shakerChannels[:len(self.shakerChannels)-compactCount]
		}
	}
}

func (self *controller) cameraGetShaker(channels ...shaker.Channel) shaker.Shaker {
	if len(channels) == 0 {
		return self.shakerChannels[0].shaker
	} else if len(channels) > 1 {
		panic("can't GetShaker for multiple shaker channels at once")
	} else if int(channels[0]) >= len(self.shakerChannels) {
		return nil
	} else {
		return self.shakerChannels[channels[0]].shaker
	}
}

func (self *controller) cameraStartShake(fadeIn TicksDuration, channels ...shaker.Channel) {
	if self.inDraw {
		panic("can't StartShake during draw stage")
	}
	if len(channels) == 0 {
		self.shakerChannels[0].Start(fadeIn)
	} else {
		for _, channel := range channels {
			if !self.shakerChannelAccessible(channel) {
				panic("can't StartShake on uninitialized channels")
			}
			self.shakerChannels[channel].Start(fadeIn)
		}
	}
}

func (self *controller) cameraEndShake(fadeOut TicksDuration, channels ...shaker.Channel) {
	if self.inDraw {
		panic("can't EndShake during draw stage")
	}
	if len(channels) == 0 {
		self.shakerChannels[0].End(fadeOut)
	} else {
		for _, channel := range channels {
			if !self.shakerChannelAccessible(channel) {
				panic("can't EndShake on uninitialized channels")
			}
			self.shakerChannels[channel].End(fadeOut)
		}
	}
}

func (self *controller) cameraTriggerShake(fadeIn, duration, fadeOut TicksDuration, channels ...shaker.Channel) {
	if self.inDraw {
		panic("can't TriggerShake during draw stage")
	}
	if len(channels) == 0 {
		self.shakerChannels[0].Trigger(fadeIn, duration, fadeOut)
	} else {
		for _, channel := range channels {
			if !self.shakerChannelAccessible(channel) {
				panic("can't TriggerShake on uninitialized channels")
			}
			self.shakerChannels[channel].Trigger(fadeIn, duration, fadeOut)
		}
	}
}

func (self *controller) cameraIsShaking(channels ...shaker.Channel) bool {
	if len(channels) > 1 {
		panic("IsShaking accepts at most one shaker channel as argument")
	}

	if len(channels) == 0 {
		for i := range self.shakerChannels {
			if self.shakerChannels[i].IsShaking() {
				return true
			}
		}
		return false
	} else if !self.shakerChannelAccessible(channels[0]) {
		return false
	} else {
		return self.shakerChannels[channels[0]].IsShaking()
	}
}

func (self *controller) shakerChannelAccessible(channel shaker.Channel) bool {
	return (channel == 0 || (int(channel) < len(self.shakerChannels) &&
		self.shakerChannels[channel].shaker != nil))
}
