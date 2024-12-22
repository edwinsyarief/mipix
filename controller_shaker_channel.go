package mipix

import "github.com/edwinsyarief/mipix/shaker"

type shakerChannel struct {
	shaker    shaker.Shaker
	elapsed   TicksDuration
	fadeIn    TicksDuration
	duration  TicksDuration
	fadeOut   TicksDuration
	offsetX   float64
	offsetY   float64
	wasActive bool
}

func (self *shakerChannel) Trigger(fadeIn, duration, fadeOut TicksDuration) {
	self.Start(fadeIn)
	self.duration = duration
	self.fadeOut = fadeOut // TODO: maybe triggered shakes shouldn't stop pre-existing continuous shakes?
}

func (self *shakerChannel) Start(fadeIn TicksDuration) {
	if self.fadeIn == fadeIn && self.IsFadingIn() {
		return
	}
	activity := self.Activity()
	self.fadeIn = fadeIn
	self.duration = maxUint32
	self.fadeOut = 0
	self.elapsed = TicksDuration(float64(fadeIn) * activity)
}

// TODO: a couple EnsureShaking(fadeIn, ...channels) and EnsureNotShaking(fadeOut, ...channels).
//       Or just guarantee that Start and End are safe to use like that, code and document.
func (self *shakerChannel) End(fadeOut TicksDuration) {
	// TODO: I don't like this code at all. going into negative durations,
	//       modifying elapsed... it's all kinda messy. I would like some
	//       solid invariants and stuff making else
	if self.fadeOut == fadeOut && self.IsFadingOut() {
		return
	}
	activity := self.Activity()
	self.duration = self.elapsed - self.fadeIn
	self.fadeOut = fadeOut
	self.elapsed = self.fadeIn + self.duration
	self.elapsed += TicksDuration(float64(fadeOut) * (1.0 - activity))
}

func (self *shakerChannel) IsFadingIn() bool {
	return self.elapsed > 0 && self.elapsed <= self.fadeIn
}

func (self *shakerChannel) IsFadingOut() bool {
	toFadeOut := self.fadeIn + self.duration
	return self.elapsed >= toFadeOut && self.elapsed < toFadeOut+self.fadeOut
}

func (self *shakerChannel) IsShaking() bool {
	if self.elapsed == 0 {
		return self.fadeIn > 0 || self.duration > 0
	} else {
		if self.elapsed < self.duration {
			return true
		}
		return self.elapsed < (self.fadeIn + self.duration + self.fadeOut)
	}
}

func (self *shakerChannel) Update(index int, tickRate uint64) {
	var selfShaker shaker.Shaker = self.shaker
	if selfShaker == nil {
		if index != 0 {
			return
		}
		if defaultShaker == nil {
			defaultShaker = &shaker.Random{}
		}
		selfShaker = defaultShaker
	}

	if self.IsShaking() {
		self.wasActive = true
		activity := self.Activity()
		self.offsetX, self.offsetY = selfShaker.GetShakeOffsets(activity)
		self.elapsed += TicksDuration(tickRate)
	} else if self.wasActive {
		_, _ = selfShaker.GetShakeOffsets(0.0) // termination call
		if self.offsetX != 0.0 || self.offsetY != 0.0 {
			self.offsetX, self.offsetY = 0.0, 0.0
		}
		self.wasActive = false
	}
}

func (self *shakerChannel) Activity() float64 {
	if self.elapsed == 0 {
		return 0
	}
	if self.elapsed < self.fadeIn {
		return float64(self.elapsed) / float64(self.fadeIn)
	} else {
		elapsed := self.elapsed - self.fadeIn
		if elapsed <= self.duration {
			return 1.0
		} // shake in progress
		elapsed -= self.duration
		if elapsed >= self.fadeOut {
			return 0.0
		}
		return 1.0 - float64(elapsed)/float64(self.fadeOut)
	}
}
