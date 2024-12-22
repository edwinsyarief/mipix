package mipix

import (
	"github.com/edwinsyarief/mipix/internal"
	"github.com/hajimehoshi/ebiten/v2"
)

// Helper type used for fades and durations of some effects.
type TicksDuration = internal.TicksDuration

const ZeroTicks TicksDuration = 0

// Quick alias to the control key for use with [AccessorDebug.Printfk]().
const Ctrl = ebiten.KeyControl

// internal usage
const maxUint32 = 0xFFFF_FFFF
const Pi = 3.141592653589793

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
		slice = slice[:index+1]
		slice[index] = element
		return slice
	}

	// more capacity needed: expand capacity
	slice = slice[:cap(slice)]
	growth := (index + 1) - len(slice)
	if growth == 1 {
		return append(slice, element)
	} else {
		slice = append(slice, make([]T, growth)...)
		slice[index] = element
		return slice
	}
}
