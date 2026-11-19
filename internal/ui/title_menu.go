package ui

import (
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"os"
	"runtime"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

// TextButton represents a rounded web-style button inside the vertical menu stack.
type TextButton struct {
	Label   string
	X, Y    float64
	W, H    float64
	Hovered bool
	HoverT  float64
	OnClick func()
}

// IconButton represents a square textured button inside the bottom-right horizontal bar.
type IconButton struct {
	Tooltip string
	SrcRect *image.Rectangle
	X, Y    float64
	Size    float64
	Hovered bool
	HoverT  float64
	OnClick func()
}

// TitleMenu holds the layout, assets, and interaction state of the main title screen.
type TitleMenu struct {
	Width  float64
	Height float64

	TitleButtons []*TextButton
	IconButtons  []*IconButton

	titleFont *text.FontSource
	uiFont    *text.FontSource
	titleFace text.Face
	subFace   text.Face
	btnFace   text.Face
	tipFace   text.Face

	iconAtlas *gg.ImageBuf
	dirty     bool
}

// TitleMenuActions defines callbacks triggered when the user clicks menu buttons.
type TitleMenuActions struct {
	OnSingleplayer  func()
	OnMultiplayer   func()
	OnRealms        func()
	OnOptions       func()
	OnQuit          func()
	OnLanguage      func()
	OnFriends       func()
	OnAccessibility func()
}

// NewTitleMenu initializes the title menu fonts, custom icon atlas, and button layout.
func NewTitleMenu(actions TitleMenuActions) *TitleMenu {
	m := &TitleMenu{dirty: true}
	m.loadFonts()

	m.TitleButtons = []*TextButton{
		{Label: "Singleplayer", OnClick: actions.OnSingleplayer},
		{Label: "Multiplayer", OnClick: actions.OnMultiplayer},
		{Label: "Realms", OnClick: actions.OnRealms},
		{Label: "Options", OnClick: actions.OnOptions},
		{Label: "Quit", OnClick: actions.OnQuit},
	}

	atlas, rects := buildIconAtlas([]string{
		"assets/icons/ui/language.png",
		"assets/icons/ui/friends.png",
		"assets/icons/ui/accessibility.png",
	})
	m.iconAtlas = atlas

	m.IconButtons = []*IconButton{
		{Tooltip: "Language", SrcRect: rects[0], OnClick: actions.OnLanguage},
		{Tooltip: "Friends", SrcRect: rects[1], OnClick: actions.OnFriends},
		{Tooltip: "Accessibility", SrcRect: rects[2], OnClick: actions.OnAccessibility},
	}

	return m
}

// SetViewport recalculates button positions whenever the window size changes.
func (m *TitleMenu) SetViewport(w, h float64) {
	if w <= 0 || h <= 0 || (m.Width == w && m.Height == h) {
		return
	}
	m.Width = w
	m.Height = h
	m.dirty = true

	const (
		marginX   = 44.0
		marginY   = 44.0
		btnWidth  = 250.0
		btnHeight = 42.0
		vboxGap   = 10.0
		iconSize  = 46.0
		hboxGap   = 10.0
	)

	// Bottom-left VBox (Un solo jugador .. Salir)
	totalVBoxH := float64(len(m.TitleButtons))*btnHeight + float64(len(m.TitleButtons)-1)*vboxGap
	startY := h - marginY - totalVBoxH
	for i, btn := range m.TitleButtons {
		btn.X = marginX
		btn.Y = startY + float64(i)*(btnHeight+vboxGap)
		btn.W = btnWidth
		btn.H = btnHeight
	}

	// Bottom-right HBox (Idioma, Amigos, Accesibilidad)
	totalHBoxW := float64(len(m.IconButtons))*iconSize + float64(len(m.IconButtons)-1)*hboxGap
	startX := w - marginX - totalHBoxW
	iconY := h - marginY - iconSize
	for i, btn := range m.IconButtons {
		btn.X = startX + float64(i)*(iconSize+hboxGap)
		btn.Y = iconY
		btn.Size = iconSize
	}
}

// ConsumeDirty reports whether the 2D menu needs to be re-rendered and clears the flag.
func (m *TitleMenu) ConsumeDirty() bool {
	if !m.dirty {
		return false
	}
	m.dirty = false
	return true
}

// IsAnimating reports whether any button hover transition is still running.
func (m *TitleMenu) IsAnimating() bool {
	for _, btn := range m.TitleButtons {
		if (btn.Hovered && btn.HoverT < 1) || (!btn.Hovered && btn.HoverT > 0) {
			return true
		}
	}
	for _, btn := range m.IconButtons {
		if (btn.Hovered && btn.HoverT < 1) || (!btn.Hovered && btn.HoverT > 0) {
			return true
		}
	}
	return false
}

// Update advances smooth CSS-like hover transitions.
func (m *TitleMenu) Update(dt float64) {
	if dt > 0.05 {
		dt = 0.05
	}
	speed := dt * 10.0
	for _, btn := range m.TitleButtons {
		next := approach(btn.HoverT, btn.Hovered, speed)
		if next != btn.HoverT {
			btn.HoverT = next
			m.dirty = true
		}
	}
	for _, btn := range m.IconButtons {
		next := approach(btn.HoverT, btn.Hovered, speed)
		if next != btn.HoverT {
			btn.HoverT = next
			m.dirty = true
		}
	}
}

// OnMouseMove updates hover states and reports whether any button state changed.
func (m *TitleMenu) OnMouseMove(x, y float64) bool {
	changed := false
	for _, btn := range m.TitleButtons {
		hovered := x >= btn.X && x <= btn.X+btn.W && y >= btn.Y && y <= btn.Y+btn.H
		if hovered != btn.Hovered {
			btn.Hovered = hovered
			changed = true
		}
	}
	for _, btn := range m.IconButtons {
		hovered := x >= btn.X && x <= btn.X+btn.Size && y >= btn.Y && y <= btn.Y+btn.Size
		if hovered != btn.Hovered {
			btn.Hovered = hovered
			changed = true
		}
	}
	if changed {
		m.dirty = true
	}
	return changed
}

// OnMousePress triggers the clicked button callback, if any.
func (m *TitleMenu) OnMousePress(x, y float64) {
	m.OnMouseMove(x, y)
	for _, btn := range m.TitleButtons {
		if btn.Hovered && btn.OnClick != nil {
			btn.OnClick()
			return
		}
	}
	for _, btn := range m.IconButtons {
		if btn.Hovered && btn.OnClick != nil {
			btn.OnClick()
			return
		}
	}
}

// Draw renders the entire title menu onto the provided 2D context.
func (m *TitleMenu) Draw(cc *gg.Context) {
	m.drawHeader(cc)
	m.drawVBoxButtons(cc)
	m.drawHBoxIconButtons(cc)
}

func (m *TitleMenu) drawHeader(cc *gg.Context) {
	const x = 44.0
	const y = 78.0

	if m.titleFace != nil {
		cc.SetFont(m.titleFace)
	}
	// Subtle drop shadow + crisp main title
	cc.SetRGBA(0, 0, 0, 0.45)
	cc.DrawString("Minecraft", x+2, y+3)
	cc.SetRGBA(1, 1, 1, 0.96)
	cc.DrawString("Minecraft", x, y)

	// Clean subtitle below the title
	if m.subFace != nil {
		cc.SetFont(m.subFace)
	}
	cc.SetRGBA(0, 0, 0, 0.45)
	cc.DrawString("Go Edition", x+1, y+25)
	cc.SetRGBA(0.85, 0.88, 0.94, 0.90)
	cc.DrawString("Go Edition", x, y+24)
}

func (m *TitleMenu) drawVBoxButtons(cc *gg.Context) {
	if m.btnFace != nil {
		cc.SetFont(m.btnFace)
	}

	for _, btn := range m.TitleButtons {
		t := btn.HoverT
		const radius = 10.0

		// Web-style solid/translucent card background
		r := lerp(0.08, 0.16, t)
		g := lerp(0.09, 0.19, t)
		b := lerp(0.13, 0.27, t)
		a := lerp(0.74, 0.90, t)
		cc.SetRGBA(r, g, b, a)
		cc.DrawRoundedRectangle(btn.X, btn.Y, btn.W, btn.H, radius)
		_ = cc.Fill()

		// 1px border highlight on hover
		borderA := lerp(0.14, 0.42, t)
		cc.SetRGBA(1, 1, 1, borderA)
		cc.SetLineWidth(1)
		cc.DrawRoundedRectangle(btn.X, btn.Y, btn.W, btn.H, radius)
		_ = cc.Stroke()

		// Left-aligned text with subtle hover slide
		textX := btn.X + 18 + t*4
		textY := btn.Y + btn.H/2
		cc.SetRGBA(0.95, 0.96, 0.99, lerp(0.86, 1.0, t))
		cc.DrawStringAnchored(btn.Label, textX, textY, 0.0, 0.5)
	}
}

func (m *TitleMenu) drawHBoxIconButtons(cc *gg.Context) {
	for _, btn := range m.IconButtons {
		t := btn.HoverT
		const radius = 12.0

		cc.SetRGBA(lerp(0.08, 0.16, t), lerp(0.09, 0.19, t), lerp(0.13, 0.27, t), lerp(0.74, 0.90, t))
		cc.DrawRoundedRectangle(btn.X, btn.Y, btn.Size, btn.Size, radius)
		_ = cc.Fill()

		cc.SetRGBA(1, 1, 1, lerp(0.14, 0.42, t))
		cc.SetLineWidth(1)
		cc.DrawRoundedRectangle(btn.X, btn.Y, btn.Size, btn.Size, radius)
		_ = cc.Stroke()

		const pad = 10.0
		iconBox := btn.Size - pad*2
		if m.iconAtlas != nil && btn.SrcRect != nil {
			cc.DrawImageEx(m.iconAtlas, gg.DrawImageOptions{
				X:             btn.X + pad,
				Y:             btn.Y + pad,
				DstWidth:      iconBox,
				DstHeight:     iconBox,
				SrcRect:       btn.SrcRect,
				Interpolation: gg.InterpBicubic,
				Opacity:       lerp(0.85, 1.0, t),
				BlendMode:     gg.BlendNormal,
			})
		} else {
			// Fallback placeholder until custom PNG textures are added
			cc.SetRGBA(0.85, 0.88, 0.95, lerp(0.45, 0.85, t))
			cc.DrawRoundedRectangle(btn.X+pad+2, btn.Y+pad+2, iconBox-4, iconBox-4, 5)
			_ = cc.Stroke()
		}

		if t > 0.05 {
			m.drawTooltip(cc, btn.Tooltip, btn.X+btn.Size/2, btn.Y-10, t)
		}
	}
}

func (m *TitleMenu) drawTooltip(cc *gg.Context, label string, cx, bottomY, alpha float64) {
	if m.tipFace != nil {
		cc.SetFont(m.tipFace)
	}
	tw, _ := cc.MeasureString(label)
	w := tw + 18
	const h = 24.0
	x := cx - w/2
	y := bottomY - h

	cc.SetRGBA(0.06, 0.07, 0.10, 0.88*alpha)
	cc.DrawRoundedRectangle(x, y, w, h, 7)
	_ = cc.Fill()

	cc.SetRGBA(1, 1, 1, 0.18*alpha)
	cc.SetLineWidth(1)
	cc.DrawRoundedRectangle(x, y, w, h, 7)
	_ = cc.Stroke()

	cc.SetRGBA(0.95, 0.96, 0.99, alpha)
	cc.DrawStringAnchored(label, cx, y+h/2, 0.5, 0.5)
}

// Close releases font sources held by the menu.
func (m *TitleMenu) Close() {
	if m.titleFont != nil {
		_ = m.titleFont.Close()
	}
	if m.uiFont != nil && m.uiFont != m.titleFont {
		_ = m.uiFont.Close()
	}
}

func (m *TitleMenu) loadFonts() {
	m.titleFont = loadFirstAvailableFont(
		"assets/fonts/Roboto-Black.ttf",
		"assets/fonts/Roboto-Regular.ttf",
		defaultSystemFont(),
	)
	m.uiFont = loadFirstAvailableFont(
		"assets/fonts/Roboto-Regular.ttf",
		defaultSystemFont(),
	)

	if m.titleFont != nil {
		m.titleFace = m.titleFont.Face(44)
	}
	if m.uiFont != nil {
		m.subFace = m.uiFont.Face(13)
		m.btnFace = m.uiFont.Face(15)
		m.tipFace = m.uiFont.Face(12)
	}
}

func loadFirstAvailableFont(paths ...string) *text.FontSource {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		src, err := text.NewFontSourceFromFile(p)
		if err != nil {
			log.Printf("ui: failed to load font %s: %v", p, err)
			continue
		}
		return src
	}
	return nil
}

func defaultSystemFont() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\Windows\Fonts\segoeui.ttf`
	case "darwin":
		return "/System/Library/Fonts/Supplemental/Arial.ttf"
	default:
		return "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
	}
}

func buildIconAtlas(paths []string) (*gg.ImageBuf, []*image.Rectangle) {
	decoded := make([]image.Image, len(paths))
	rects := make([]*image.Rectangle, len(paths))

	totalW, maxH := 0, 0
	for i, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			log.Printf("ui: failed to open icon %s: %v", p, err)
			continue
		}
		img, _, err := image.Decode(f)
		_ = f.Close()
		if err != nil {
			log.Printf("ui: failed to decode icon %s: %v", p, err)
			continue
		}
		decoded[i] = img
		b := img.Bounds()
		if b.Dy() > maxH {
			maxH = b.Dy()
		}
		totalW += b.Dx()
		log.Printf("ui: loaded icon %s (%dx%d)", p, b.Dx(), b.Dy())
	}

	if totalW == 0 || maxH == 0 {
		return nil, rects
	}

	atlas := image.NewRGBA(image.Rect(0, 0, totalW, maxH))
	x := 0
	for i, img := range decoded {
		if img == nil {
			continue
		}
		b := img.Bounds()
		dst := image.Rect(x, 0, x+b.Dx(), b.Dy())
		draw.Draw(atlas, dst, img, b.Min, draw.Src)
		r := dst
		rects[i] = &r
		x += b.Dx()
	}

	return gg.ImageBufFromImage(atlas), rects
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