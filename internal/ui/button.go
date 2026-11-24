package ui

import (
	"image"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

// UIButton is a robust, theme-aware button widget supporting hover animations,
// primary states, custom text alignment, icon atlas rendering, and tooltips.
type UIButton struct {
	X, Y, W, H float64
	Label      string
	Key        string // Optional i18n translation key
	Primary    bool
	AlignLeft  bool
	Radius     float64
	Hovered    bool
	HoverT     float64
	OnClick    func()
	Face       text.Face
	TipFace    text.Face

	// Optional icon atlas support
	IconAtlas *gg.ImageBuf
	SrcRect   *image.Rectangle
	Tooltip   string
}

// NewUIButton creates a standard text button.
func NewUIButton(label string, primary bool, onClick func()) *UIButton {
	return &UIButton{
		Label:   label,
		Primary: primary,
		OnClick: onClick,
		Radius:  10.0,
	}
}

// SetBounds sets button position and dimensions.
func (b *UIButton) SetBounds(x, y, w, h float64) {
	b.X, b.Y, b.W, b.H = x, y, w, h
}

// Bounds returns button position and dimensions.
func (b *UIButton) Bounds() (x, y, w, h float64) {
	return b.X, b.Y, b.W, b.H
}

// Update advances the smooth hover animation.
func (b *UIButton) Update(dt float64) {
	if dt > 0.05 {
		dt = 0.05
	}
	speed := dt * 10.0
	b.HoverT = approach(b.HoverT, b.Hovered, speed)
}

// Draw renders the button into the provided 2D context.
func (b *UIButton) Draw(cc *gg.Context) {
	if b.Face != nil {
		cc.SetFont(b.Face)
	}
	t := b.HoverT
	radius := b.Radius
	if radius <= 0 {
		radius = 10.0
	}

	// Background fill
	if b.Primary {
		cc.SetRGBA(lerp(0.24, 0.34, t), lerp(0.22, 0.32, t), lerp(0.58, 0.72, t), lerp(0.88, 0.98, t))
	} else {
		cc.SetRGBA(lerp(0.08, 0.16, t), lerp(0.09, 0.19, t), lerp(0.13, 0.27, t), lerp(0.74, 0.90, t))
	}
	cc.DrawRoundedRectangle(b.X, b.Y, b.W, b.H, radius)
	_ = cc.Fill()

	// Border stroke
	cc.SetRGBA(1, 1, 1, lerp(0.14, 0.42, t))
	cc.SetLineWidth(1)
	cc.DrawRoundedRectangle(b.X, b.Y, b.W, b.H, radius)
	_ = cc.Stroke()

	// Icon rendering (if atlas and rectangle configured)
	if b.IconAtlas != nil && b.SrcRect != nil {
		const pad = 10.0
		iconW := b.W - pad*2
		iconH := b.H - pad*2
		if iconW > 0 && iconH > 0 {
			cc.DrawImageEx(b.IconAtlas, gg.DrawImageOptions{
				X:             b.X + pad,
				Y:             b.Y + pad,
				DstWidth:      iconW,
				DstHeight:     iconH,
				SrcRect:       b.SrcRect,
				Interpolation: gg.InterpBicubic,
				Opacity:       lerp(0.85, 1.0, t),
				BlendMode:     gg.BlendNormal,
			})
		}
	} else if b.SrcRect != nil && b.IconAtlas == nil {
		// Fallback outline if texture image not loaded
		const pad = 10.0
		iconW := b.W - pad*2
		iconH := b.H - pad*2
		cc.SetRGBA(0.85, 0.88, 0.95, lerp(0.45, 0.85, t))
		cc.DrawRoundedRectangle(b.X+pad+2, b.Y+pad+2, iconW-4, iconH-4, 5)
		_ = cc.Stroke()
	}

	// Text label rendering
	if b.Label != "" {
		if b.AlignLeft {
			textX := b.X + 18 + t*4
			textY := b.Y + b.H/2
			cc.SetRGBA(0.95, 0.96, 0.99, lerp(0.86, 1.0, t))
			cc.DrawStringAnchored(b.Label, textX, textY, 0.0, 0.5)
		} else {
			cc.SetRGBA(0.96, 0.97, 1.0, lerp(0.88, 1.0, t))
			cc.DrawStringAnchored(b.Label, b.X+b.W/2, b.Y+b.H/2, 0.5, 0.5)
		}
	}

	// Tooltip rendering if hovered and text available
	if b.Tooltip != "" && t > 0.05 {
		b.DrawTooltip(cc)
	}
}

// DrawTooltip renders an elegant tooltip bubble above the button.
func (b *UIButton) DrawTooltip(cc *gg.Context) {
	t := b.HoverT
	if b.TipFace != nil {
		cc.SetFont(b.TipFace)
	}
	tw, _ := cc.MeasureString(b.Tooltip)
	w := tw + 18
	const h = 24.0
	cx := b.X + b.W/2
	bottomY := b.Y - 10
	x := cx - w/2
	y := bottomY - h

	cc.SetRGBA(0.06, 0.07, 0.10, 0.88*t)
	cc.DrawRoundedRectangle(x, y, w, h, 7)
	_ = cc.Fill()

	cc.SetRGBA(1, 1, 1, 0.18*t)
	cc.SetLineWidth(1)
	cc.DrawRoundedRectangle(x, y, w, h, 7)
	_ = cc.Stroke()

	cc.SetRGBA(0.95, 0.96, 0.99, t)
	cc.DrawStringAnchored(b.Tooltip, cx, y+h/2, 0.5, 0.5)
}

// OnMouseMove updates hover state and returns true if hover state changed.
func (b *UIButton) OnMouseMove(x, y float64) bool {
	hovered := x >= b.X && x <= b.X+b.W && y >= b.Y && y <= b.Y+b.H
	changed := hovered != b.Hovered
	b.Hovered = hovered
	return changed
}

// OnMousePress handles click events.
func (b *UIButton) OnMousePress(x, y float64) bool {
	if x >= b.X && x <= b.X+b.W && y >= b.Y && y <= b.Y+b.H {
		if b.OnClick != nil {
			b.OnClick()
			return true
		}
	}
	return false
}
