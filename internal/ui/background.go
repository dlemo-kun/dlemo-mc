package ui

import (
	"github.com/gogpu/gg"
)

// BackgroundProvider defines the background rendering contract for menus and screens.
// This allows rendering a 3D panorama, active game world, or fallback solid/gradient background
// behind any UI screen.
type BackgroundProvider interface {
	// DrawBackground renders the background layer onto the gg context.
	DrawBackground(cc *gg.Context, w, h float64)

	// Update advances background animations (e.g. rotating panorama camera).
	Update(dt float64)
}

// SolidBackground is a default BackgroundProvider that renders a solid color or gradient.
type SolidBackground struct {
	R, G, B, A float64
}

func (s *SolidBackground) DrawBackground(cc *gg.Context, w, h float64) {
	cc.SetRGBA(s.R, s.G, s.B, s.A)
	cc.Clear()
}

func (s *SolidBackground) Update(dt float64) {}
