package ui

import (
	"github.com/gogpu/gg"
)

// FlexDirection specifies layout stacking direction.
type FlexDirection int

const (
	Column FlexDirection = iota
	Row
)

// FlexBox is a container widget that arranges child widgets in a vertical or horizontal stack.
type FlexBox struct {
	X, Y, W, H float64
	Direction  FlexDirection
	Gap        float64
	Padding    float64
	Children   []Widget
}

// NewFlexBox creates a new flex container.
func NewFlexBox(dir FlexDirection, gap, padding float64) *FlexBox {
	return &FlexBox{
		Direction: dir,
		Gap:       gap,
		Padding:   padding,
	}
}

// Add appends a child widget to the flex container.
func (f *FlexBox) Add(w Widget) {
	f.Children = append(f.Children, w)
}

// Clear removes all children from the flex container.
func (f *FlexBox) Clear() {
	f.Children = nil
}

// SetBounds configures container dimensions and arranges child widgets.
func (f *FlexBox) SetBounds(x, y, w, h float64) {
	f.X, f.Y, f.W, f.H = x, y, w, h
	f.layoutChildren()
}

// Bounds returns container bounds.
func (f *FlexBox) Bounds() (x, y, w, h float64) {
	return f.X, f.Y, f.W, f.H
}

func (f *FlexBox) layoutChildren() {
	if len(f.Children) == 0 {
		return
	}

	availW := f.W - (f.Padding * 2)
	availH := f.H - (f.Padding * 2)

	if f.Direction == Column {
		totalGap := float64(len(f.Children)-1) * f.Gap
		childH := (availH - totalGap) / float64(len(f.Children))
		if childH < 10 {
			childH = 10
		}
		curY := f.Y + f.Padding
		for _, child := range f.Children {
			child.SetBounds(f.X+f.Padding, curY, availW, childH)
			curY += childH + f.Gap
		}
	} else {
		totalGap := float64(len(f.Children)-1) * f.Gap
		childW := (availW - totalGap) / float64(len(f.Children))
		if childW < 10 {
			childW = 10
		}
		curX := f.X + f.Padding
		for _, child := range f.Children {
			child.SetBounds(curX, f.Y+f.Padding, childW, availH)
			curX += childW + f.Gap
		}
	}
}

// Update advances all child widgets.
func (f *FlexBox) Update(dt float64) {
	for _, child := range f.Children {
		child.Update(dt)
	}
}

// Draw renders all child widgets.
func (f *FlexBox) Draw(cc *gg.Context) {
	for _, child := range f.Children {
		child.Draw(cc)
	}
}

// OnMouseMove forwards pointer motion to child widgets.
func (f *FlexBox) OnMouseMove(x, y float64) bool {
	handled := false
	for _, child := range f.Children {
		if child.OnMouseMove(x, y) {
			handled = true
		}
	}
	return handled
}

// OnMousePress dispatches mouse press to the child located at (x, y).
func (f *FlexBox) OnMousePress(x, y float64) bool {
	for _, child := range f.Children {
		cx, cy, cw, ch := child.Bounds()
		if x >= cx && x <= cx+cw && y >= cy && y <= cy+ch {
			if child.OnMousePress(x, y) {
				return true
			}
		}
	}
	return false
}
