package ui

import (
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/dlemo-kun/dlemo-mc/internal/config"
	"github.com/dlemo-kun/dlemo-mc/internal/i18n"
	"github.com/dlemo-kun/dlemo-mc/internal/splash"
	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

const (
	splashCycleDuration = 30.0
	splashFadeDuration  = 0.6
)

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

// TitleMenu holds the layout, theme, and interaction state of the redesigned main title screen.
type TitleMenu struct {
	Width  float64
	Height float64

	vbox        *FlexBox
	buttons     []*UIButton
	hbox        *FlexBox
	iconButtons []*UIButton

	// Kept for backward compatibility and testing inspection
	TitleButtons []*UIButton
	IconButtons  []*UIButton

	theme      *UITheme
	titleFace  text.Face
	subFace    text.Face
	splashFace text.Face

	tr            *i18n.Manager
	splashMgr     *splash.Manager
	splashText    string
	splashSpecial bool
	splashElapsed float64
	splashAlpha   float64
	iconAtlas     *gg.ImageBuf
	dirty         bool
}

// NewTitleMenu initializes the title menu with unified UITheme, FlexBox layout, and UIButton components.
func NewTitleMenu(tr *i18n.Manager, actions TitleMenuActions) *TitleMenu {
	theme := NewUITheme()

	m := &TitleMenu{
		tr:    tr,
		theme: theme,
		dirty: true,
	}

	if theme.TitleFont != nil {
		m.titleFace = theme.TitleFont.Face(48)
		m.splashFace = theme.TitleFont.Face(18)
	}
	if theme.UIFont != nil {
		m.subFace = theme.UIFont.Face(24)
		if m.splashFace == nil {
			m.splashFace = theme.UIFont.Face(18)
		}
	}

	// 1. Unified Text Action Buttons in bottom-left VBox
	m.buttons = []*UIButton{
		{Key: "menu.singleplayer", Label: "Singleplayer", AlignLeft: true, Face: theme.BtnFace, OnClick: actions.OnSingleplayer},
		{Key: "menu.multiplayer", Label: "Multiplayer", AlignLeft: true, Face: theme.BtnFace, OnClick: actions.OnMultiplayer},
		{Key: "menu.online", Label: "Realms", AlignLeft: true, Face: theme.BtnFace, OnClick: actions.OnRealms},
		{Key: "options.title", Label: "Options", AlignLeft: true, Face: theme.BtnFace, OnClick: actions.OnOptions},
		{Key: "menu.quit", Label: "Quit", AlignLeft: true, Face: theme.BtnFace, OnClick: actions.OnQuit},
	}
	m.TitleButtons = m.buttons

	m.vbox = NewFlexBox(Column, 10, 0)
	for _, btn := range m.buttons {
		m.vbox.Add(btn)
	}

	// 2. Bottom-right HBox for Icon Buttons
	iconsDir := ""
	if base, err := config.Dir(); err == nil {
		iconsDir = filepath.Join(base, "assets/icons/ui")
	}

	atlas, rects := buildIconAtlas([]string{
		filepath.Join(iconsDir, "language.png"),
		filepath.Join(iconsDir, "friends.png"),
		filepath.Join(iconsDir, "accessibility.png"),
	})
	m.iconAtlas = atlas

	m.iconButtons = []*UIButton{
		{Key: "options.language.tooltip", Tooltip: "Language", SrcRect: rects[0], IconAtlas: atlas, TipFace: theme.TipFace, Radius: 12, OnClick: actions.OnLanguage},
		{Key: "gui.friends.open.tooltip", Tooltip: "Friends", SrcRect: rects[1], IconAtlas: atlas, TipFace: theme.TipFace, Radius: 12, OnClick: actions.OnFriends},
		{Key: "options.accessibility.tooltip", Tooltip: "Accessibility", SrcRect: rects[2], IconAtlas: atlas, TipFace: theme.TipFace, Radius: 12, OnClick: actions.OnAccessibility},
	}
	m.IconButtons = m.iconButtons

	m.hbox = NewFlexBox(Row, 10, 0)
	for _, btn := range m.iconButtons {
		m.hbox.Add(btn)
	}

	m.SetSplashManager(splash.NewManager())

	return m
}

// SetSplashManager replaces the active splash manager and initializes the first splash text.
func (m *TitleMenu) SetSplashManager(sm *splash.Manager) {
	m.splashMgr = sm
	m.splashElapsed = 0
	m.splashAlpha = 0
	if sm == nil {
		m.splashText = ""
		m.splashSpecial = false
		return
	}
	m.splashText, m.splashSpecial = sm.Current("")
}

// SetViewport recalculates menu layout responsively when the window size changes.
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

	// Bottom-left VBox layout
	totalVBoxH := float64(len(m.buttons))*btnHeight + float64(len(m.buttons)-1)*vboxGap
	startY := h - marginY - totalVBoxH
	m.vbox.SetBounds(marginX, startY, btnWidth, totalVBoxH)

	// Bottom-right HBox layout
	totalHBoxW := float64(len(m.iconButtons))*iconSize + float64(len(m.iconButtons)-1)*hboxGap
	startX := w - marginX - totalHBoxW
	iconY := h - marginY - iconSize
	m.hbox.SetBounds(startX, iconY, totalHBoxW, iconSize)
}

// Update advances hover transitions and the splash fade/rotation animation.
func (m *TitleMenu) Update(dt float64) {
	m.vbox.Update(dt)
	m.hbox.Update(dt)
	m.updateSplash(dt)
}

// OnMouseMove updates hover state across all widgets.
func (m *TitleMenu) OnMouseMove(x, y float64) bool {
	vChanged := m.vbox.OnMouseMove(x, y)
	hChanged := m.hbox.OnMouseMove(x, y)
	if vChanged || hChanged {
		m.dirty = true
		return true
	}
	return false
}

// OnMousePress handles click events on widgets.
func (m *TitleMenu) OnMousePress(x, y float64) {
	if m.vbox.OnMousePress(x, y) {
		return
	}
	_ = m.hbox.OnMousePress(x, y)
}

// Draw renders the title menu, header, and flex containers onto the canvas.
func (m *TitleMenu) Draw(cc *gg.Context) {
	// Dynamically update translated strings for labels and tooltips
	for _, btn := range m.buttons {
		if btn.Key != "" && m.tr != nil {
			btn.Label = m.tr.T(btn.Key)
		}
	}
	for _, btn := range m.iconButtons {
		if btn.Key != "" && m.tr != nil {
			btn.Tooltip = m.tr.T(btn.Key)
		}
	}

	m.drawHeader(cc)
	m.drawSplash(cc)
	m.vbox.Draw(cc)
	m.hbox.Draw(cc)
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

func (m *TitleMenu) updateSplash(dt float64) {
	if m.splashMgr == nil || dt <= 0 {
		return
	}

	if specialText, isSpecial := m.splashMgr.CurrentSpecial(); isSpecial {
		if !m.splashSpecial || m.splashText != specialText {
			m.splashText = specialText
			m.splashSpecial = true
			m.splashElapsed = 0
		}
		m.splashElapsed += dt
		if m.splashElapsed >= splashFadeDuration {
			m.splashElapsed = splashFadeDuration
			m.splashAlpha = 1.0
		} else {
			m.splashAlpha = smoothstep01(m.splashElapsed / splashFadeDuration)
		}
		return
	}

	if m.splashSpecial {
		m.splashSpecial = false
		m.splashText, _ = m.splashMgr.Current(m.splashText)
		m.splashElapsed = 0
	}

	m.splashElapsed += dt
	for m.splashElapsed >= splashCycleDuration {
		m.splashElapsed -= splashCycleDuration
		m.splashText, _ = m.splashMgr.Current(m.splashText)
	}

	switch {
	case m.splashElapsed < splashFadeDuration:
		m.splashAlpha = smoothstep01(m.splashElapsed / splashFadeDuration)
	case m.splashElapsed > splashCycleDuration-splashFadeDuration:
		m.splashAlpha = smoothstep01((splashCycleDuration - m.splashElapsed) / splashFadeDuration)
	default:
		m.splashAlpha = 1.0
	}
}

func (m *TitleMenu) drawSplash(cc *gg.Context) {
	if m.splashText == "" || m.splashAlpha <= 0.001 || m.Width <= 0 {
		return
	}

	if m.splashFace != nil {
		cc.SetFont(m.splashFace)
	}

	const marginX = 44.0
	const centerY = 76.0
	const lineHeight = 22.0

	anchorX := m.Width - marginX
	maxW := math.Max(180.0, m.Width-360.0)
	lines := wrapSplashLines(cc, m.splashText, maxW)
	if len(lines) == 0 {
		return
	}

	startY := centerY - float64(len(lines)-1)*lineHeight/2.0
	alpha := m.splashAlpha
	for i, line := range lines {
		ly := startY + float64(i)*lineHeight
		cc.SetRGBA(0, 0, 0, 0.55*alpha)
		cc.DrawStringAnchored(line, anchorX+1.5, ly+2.0, 1.0, 0.5)
		cc.SetRGBA(1.0, 0.94, 0.22, 0.98*alpha)
		cc.DrawStringAnchored(line, anchorX, ly, 1.0, 0.5)
	}
}

func wrapSplashLines(cc *gg.Context, text string, maxW float64) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if tw, _ := cc.MeasureString(text); tw <= maxW {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) <= 1 {
		return []string{text}
	}

	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		candidate := cur + " " + w
		if tw, _ := cc.MeasureString(candidate); tw <= maxW {
			cur = candidate
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	lines = append(lines, cur)
	return lines
}

func smoothstep01(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3.0 - 2.0*t)
}

// Close satisfies the Screen interface.
func (m *TitleMenu) Close() {}

// MarkDirty flags the screen as needing redraw.
func (m *TitleMenu) MarkDirty() {
	m.dirty = true
}

// ConsumeDirty reports whether the screen needs redraw.
func (m *TitleMenu) ConsumeDirty() bool {
	return true
}

// IsAnimating reports whether any transition is running.
func (m *TitleMenu) IsAnimating() bool {
	return true
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
