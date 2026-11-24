package ui

import (
	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

// CardWidget represents a selectable list item card (for worlds, servers, or languages)
// with hover animations, selection state, and optional status indicators.
type CardWidget struct {
	X, Y, W, H float64
	Title      string
	Subtitle   string
	Meta       string
	Selected   bool
	Hovered    bool
	HoverT     float64
	OnClick    func()
	Radius     float64

	NameFace text.Face
	MetaFace text.Face

	// Optional server status indicator (ping dot and online count)
	ShowStatus bool
	StatusText string
	StatusDotR float64
	StatusDotG float64
	StatusDotB float64
}

// NewCardWidget creates a new card item.
func NewCardWidget(title, subtitle, meta string, selected bool, onClick func()) *CardWidget {
	return &CardWidget{
		Title:    title,
		Subtitle: subtitle,
		Meta:     meta,
		Selected: selected,
		OnClick:  onClick,
		Radius:   12.0,
	}
}

func (c *CardWidget) SetBounds(x, y, w, h float64) {
	c.X, c.Y, c.W, c.H = x, y, w, h
}

func (c *CardWidget) Bounds() (x, y, w, h float64) {
	return c.X, c.Y, c.W, c.H
}

func (c *CardWidget) Update(dt float64) {
	if dt > 0.05 {
		dt = 0.05
	}
	speed := dt * 10.0
	c.HoverT = approach(c.HoverT, c.Hovered, speed)
}

func (c *CardWidget) Draw(cc *gg.Context) {
	t := c.HoverT
	radius := c.Radius
	if radius <= 0 {
		radius = 12.0
	}

	// Card background
	if c.Selected {
		cc.SetRGBA(0.14, 0.22, 0.36, 0.90)
	} else {
		cc.SetRGBA(lerp(0.09, 0.14, t), lerp(0.11, 0.17, t), lerp(0.16, 0.24, t), lerp(0.72, 0.86, t))
	}
	cc.DrawRoundedRectangle(c.X, c.Y, c.W, c.H, radius)
	_ = cc.Fill()

	// Card border
	if c.Selected {
		cc.SetRGBA(0.45, 0.72, 1.0, 0.80)
		cc.SetLineWidth(1.5)
	} else {
		cc.SetRGBA(1, 1, 1, lerp(0.10, 0.28, t))
		cc.SetLineWidth(1)
	}
	cc.DrawRoundedRectangle(c.X, c.Y, c.W, c.H, radius)
	_ = cc.Stroke()

	// Card text info
	padLeft := 18.0
	if c.NameFace != nil {
		cc.SetFont(c.NameFace)
	}
	cc.SetRGBA(0.96, 0.97, 1.0, 0.96)
	if c.Subtitle != "" {
		cc.DrawString(c.Title, c.X+padLeft, c.Y+27)
	} else {
		// Vertically centered single-line title (e.g. language card)
		cc.DrawStringAnchored(c.Title, c.X+padLeft, c.Y+c.H/2, 0.0, 0.5)
	}

	if c.MetaFace != nil {
		cc.SetFont(c.MetaFace)
	}
	cc.SetRGBA(0.72, 0.77, 0.87, 0.88)
	if c.Subtitle != "" {
		cc.DrawString(c.Subtitle, c.X+padLeft, c.Y+49)
	}

	// Status indicator or meta text
	if c.ShowStatus {
		// Ping status dot
		cc.SetRGBA(c.StatusDotR, c.StatusDotG, c.StatusDotB, 0.95)
		cc.DrawCircle(c.X+c.W-24, c.Y+c.H/2, 5)
		_ = cc.Fill()

		// Online players or status text
		cc.SetRGBA(0.88, 0.92, 0.98, 0.92)
		cc.DrawStringAnchored(c.StatusText, c.X+c.W-38, c.Y+c.H/2, 1.0, 0.5)
	} else if c.Meta != "" {
		cc.SetRGBA(0.72, 0.78, 0.88, 0.85)
		cc.DrawStringAnchored(c.Meta, c.X+c.W-18, c.Y+c.H/2, 1.0, 0.5)
	}
}

func (c *CardWidget) OnMouseMove(x, y float64) bool {
	hovered := x >= c.X && x <= c.X+c.W && y >= c.Y && y <= c.Y+c.H
	changed := hovered != c.Hovered
	c.Hovered = hovered
	return changed
}

func (c *CardWidget) OnMousePress(x, y float64) bool {
	if x >= c.X && x <= c.X+c.W && y >= c.Y && y <= c.Y+c.H {
		if c.OnClick != nil {
			c.OnClick()
			return true
		}
	}
	return false
}
