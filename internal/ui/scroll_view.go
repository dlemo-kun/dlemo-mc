package ui

import (
	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

// ScrollView wraps a vertical scrollable list of widgets with smooth lerp scrolling,
// viewport culling, and a sleek scrollbar indicator.
type ScrollView struct {
	X, Y, W, H    float64
	ContentHeight float64
	ScrollY       float64
	TargetScrollY float64
	Children      []Widget

	// Fonts and styling
	NameFace text.Face
	MetaFace text.Face
}

// NewScrollView creates a new scrollable view container.
func NewScrollView(w, h float64) *ScrollView {
	return &ScrollView{
		W: w,
		H: h,
	}
}

// Add appends a widget child to the scroll view.
func (s *ScrollView) Add(w Widget) {
	s.Children = append(s.Children, w)
}

// Clear removes all children and resets scroll state.
func (s *ScrollView) Clear() {
	s.Children = nil
	s.ContentHeight = 0
}

// SetBounds configures the outer viewport box.
func (s *ScrollView) SetBounds(x, y, w, h float64) {
	s.X, s.Y, s.W, s.H = x, y, w, h
}

// Bounds returns the outer viewport box.
func (s *ScrollView) Bounds() (x, y, w, h float64) {
	return s.X, s.Y, s.W, s.H
}

// LayoutVertical automatically positions all child widgets in a vertical list,
// calculating ContentHeight and adjusting width when the scrollbar is needed.
func (s *ScrollView) LayoutVertical(cardH, gap float64) {
	if len(s.Children) == 0 {
		s.ContentHeight = 0
		return
	}

	totalH := float64(len(s.Children))*(cardH+gap) - gap
	if totalH < 0 {
		totalH = 0
	}
	s.ContentHeight = totalH

	scrollbarMargin := 0.0
	if totalH > s.H {
		scrollbarMargin = 16.0
	}
	cardW := s.W - scrollbarMargin
	if cardW < 20 {
		cardW = 20
	}

	curY := s.Y
	for _, child := range s.Children {
		child.SetBounds(s.X, curY, cardW, cardH)
		curY += cardH + gap
	}
}

// ScrollBy offsets the target scroll position smoothly.
func (s *ScrollView) ScrollBy(delta float64) {
	s.TargetScrollY += delta
}

// Update advances smooth scrolling and updates child widgets.
func (s *ScrollView) Update(dt float64) {
	if dt > 0.05 {
		dt = 0.05
	}
	// Smooth scroll approach
	speed := dt * 15.0
	s.ScrollY += (s.TargetScrollY - s.ScrollY) * speed

	// Clamp scroll
	maxScroll := s.ContentHeight - s.H
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.TargetScrollY < 0 {
		s.TargetScrollY = 0
	}
	if s.TargetScrollY > maxScroll {
		s.TargetScrollY = maxScroll
	}
	if s.ScrollY < 0 {
		s.ScrollY = 0
	}
	if s.ScrollY > maxScroll {
		s.ScrollY = maxScroll
	}

	for _, child := range s.Children {
		child.Update(dt)
	}
}

// Draw renders visible children with viewport clipping and draws the scrollbar.
func (s *ScrollView) Draw(cc *gg.Context) {
	if s.W <= 0 || s.H <= 0 {
		return
	}

	// Clip scroll viewport
	cc.Push()
	cc.DrawRectangle(s.X, s.Y, s.W, s.H)
	cc.Clip()

	// Translate by scroll offset
	cc.Translate(0, -s.ScrollY)

	// Cull children that are entirely outside the visible viewport
	viewTop := s.Y + s.ScrollY
	viewBottom := viewTop + s.H
	for _, child := range s.Children {
		_, cy, _, ch := child.Bounds()
		if cy+ch < viewTop || cy > viewBottom {
			continue
		}
		child.Draw(cc)
	}
	cc.Pop()

	// Draw scrollbar indicator when content exceeds viewport
	maxScroll := s.ContentHeight - s.H
	if maxScroll > 0 && s.H > 30 {
		trackX := s.X + s.W - 8
		trackY := s.Y
		trackW := 6.0
		trackH := s.H

		// Subtle track background
		cc.SetRGBA(1, 1, 1, 0.08)
		cc.DrawRoundedRectangle(trackX, trackY, trackW, trackH, 3)
		_ = cc.Fill()

		// Thumb indicator
		ratio := s.H / s.ContentHeight
		thumbH := trackH * ratio
		if thumbH < 24 {
			thumbH = 24
		}
		scrollRatio := s.ScrollY / maxScroll
		if scrollRatio < 0 {
			scrollRatio = 0
		} else if scrollRatio > 1 {
			scrollRatio = 1
		}
		thumbY := trackY + scrollRatio*(trackH-thumbH)

		cc.SetRGBA(0.65, 0.78, 0.98, 0.65)
		cc.DrawRoundedRectangle(trackX, thumbY, trackW, thumbH, 3)
		_ = cc.Fill()
	}
}

// OnMouseMove forwards pointer coordinates, accounting for scroll offset and bounds clipping.
func (s *ScrollView) OnMouseMove(x, y float64) bool {
	if x < s.X || x > s.X+s.W || y < s.Y || y > s.Y+s.H {
		// When mouse is outside scrollview, unhover all children
		handled := false
		for _, child := range s.Children {
			if child.OnMouseMove(-9999, -9999) {
				handled = true
			}
		}
		return handled
	}

	adjustedY := y + s.ScrollY
	handled := false
	for _, child := range s.Children {
		if child.OnMouseMove(x, adjustedY) {
			handled = true
		}
	}
	return handled
}

// OnMousePress handles clicks on visible children inside the scroll view.
func (s *ScrollView) OnMousePress(x, y float64) bool {
	if x < s.X || x > s.X+s.W || y < s.Y || y > s.Y+s.H {
		return false
	}
	adjustedY := y + s.ScrollY
	for _, child := range s.Children {
		cx, cy, cw, ch := child.Bounds()
		if x >= cx && x <= cx+cw && adjustedY >= cy && adjustedY <= cy+ch {
			if child.OnMousePress(x, adjustedY) {
				return true
			}
		}
	}
	return false
}
