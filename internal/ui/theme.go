package ui

import (
	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

// UITheme holds standard colors, fonts, and spacing tokens for the UI 2.0 system.
type UITheme struct {
	TitleFont *FontStack
	UIFont    *FontStack

	// Faces at standard sizes
	TitleFace text.Face
	SubFace   text.Face
	NameFace  text.Face
	MetaFace  text.Face
	BtnFace   text.Face
	TipFace   text.Face
}

// NewUITheme initializes theme font faces from cached font stacks.
func NewUITheme() *UITheme {
	titleFont, uiFont := loadFont()
	t := &UITheme{
		TitleFont: titleFont,
		UIFont:    uiFont,
	}
	if titleFont != nil {
		t.TitleFace = titleFont.Face(32)
	}
	if uiFont != nil {
		t.SubFace = uiFont.Face(13)
		t.NameFace = uiFont.Face(16)
		t.MetaFace = uiFont.Face(13)
		t.BtnFace = uiFont.Face(15)
		t.TipFace = uiFont.Face(12)
	}
	return t
}

// Widget represents any interactive or layout UI component in the UI 2.0 system.
type Widget interface {
	SetBounds(x, y, w, h float64)
	Bounds() (x, y, w, h float64)
	Update(dt float64)
	Draw(cc *gg.Context)
	OnMouseMove(x, y float64) bool
	OnMousePress(x, y float64) bool
}

func approach(cur float64, target bool, delta float64) float64 {
	if target {
		cur += delta
		if cur > 1 {
			return 1
		}
		return cur
	}
	cur -= delta
	if cur < 0 {
		return 0
	}
	return cur
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}
