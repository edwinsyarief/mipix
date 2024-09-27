package mipix

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/mipix/internal"

// Helper type used for fades and durations of some effects.
type TicksDuration = internal.TicksDuration

const ZeroTicks TicksDuration = 0

// Quick alias to the control key for use with [AccessorDebug.Printfk]().
const Ctrl = ebiten.KeyControl

// internal usage
const maxUint32 = 0xFFFF_FFFF

// --- helpers ---

func setAt[T any](slice []T, element T, index int) []T {
	// base case: element index already in range
	if index < len(slice) {
		slice[index] = element
		return slice
	}
	
	// append case: element index is the next
	if index == len(slice) {
		return append(slice, element)
	}

	// within capacity: element can be set by expanding capacity
	if index < cap(slice) {
		slice = slice[ : index + 1]
		slice[index] = element
		return slice
	}

	// more capacity needed: expand capacity
	slice = slice[ : cap(slice)]
	growth := (index + 1) - len(slice)
	if growth == 1 {
		return append(slice, element)
	} else {
		slice = append(slice, make([]T, growth)...)
		slice[index] = element
		return slice
	}
}

// --- errors ---
const mixedShakerChans = "can't mix shaker.ChanAll with other explicit channels"
