package ui

import (
	"strings"
	"sync"

	"github.com/dlemo-kun/dlemo-mc/internal/i18n"
	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

type langItem struct {
	info    i18n.LanguageInfo
	hovered bool
	hoverT  float64
}

// LanguageSelectScreen displays all languages discovered in .minecraft.
type LanguageSelectScreen struct {
	mu       sync.Mutex
	tr       *i18n.Manager
	w, h     float64
	items    []*langItem
	scrollY  float64
	maxScroll float64

	panelX, panelY, panelW, panelH float64
	listX, listY, listW, listH     float64

	upBtn   *TextButton
	downBtn *TextButton
	doneBtn *TextButton

	onSelect func(code string)
	onBack   func()
	
	titleFont *FontStack
	uiFont    *FontStack
	titleFace text.Face
	subFace   text.Face
	nameFace  text.Face
	metaFace  text.Face
	btnFace   text.Face
}

const (
	langRowH   = 52.0
	langRowGap = 8.0
)

// NewLanguageSelectScreen creates the language selector populated from tr.
func NewLanguageSelectScreen(tr *i18n.Manager, onSelect func(code string), onBack func()) *LanguageSelectScreen {
	langs := tr.Languages()
	items := make([]*langItem, len(langs))
	for i, l := range langs {
		items[i] = &langItem{info: l}
	}

	s := &LanguageSelectScreen{
		tr:       tr,
		items:    items,
		onSelect: onSelect,
		onBack:   onBack,
	}

	titleFont, uiFont := loadFont()

	s.titleFont = titleFont
	s.uiFont = uiFont

	if s.titleFont != nil {
		s.titleFace = s.titleFont.Face(32)
	}
	if s.uiFont != nil {
		s.subFace = s.uiFont.Face(13)
		s.nameFace = s.uiFont.Face(15)
		s.metaFace = s.uiFont.Face(12)
		s.btnFace = s.uiFont.Face(15)
	}

	s.upBtn = &TextButton{
		Label: "▲",
		OnClick: func() {
			s.OnMouseScroll(2.0)
		},
	}
	s.downBtn = &TextButton{
		Label: "▼",
		OnClick: func() {
			s.OnMouseScroll(-2.0)
		},
	}
	s.doneBtn = &TextButton{
		Label:   "Done",
		OnClick: onBack,
	}

	return s
}

// SetViewport recalculates panel, scroll bounds, and footer button layout.
func (s *LanguageSelectScreen) SetViewport(w, h float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w <= 0 || h <= 0 || (s.w == w && s.h == h) {
		return
	}
	s.w, s.h = w, h

	s.panelX = 44.0
	s.panelW = w - 88.0
	if s.panelW > 760 {
		s.panelW = 760
		s.panelX = (w - s.panelW) / 2
	}
	s.panelY = 108.0
	s.panelH = h - s.panelY - 104.0
	if s.panelH < 120 {
		s.panelH = 120
	}

	s.listX = s.panelX + 16
	s.listY = s.panelY + 14
	s.listW = s.panelW - 72
	s.listH = s.panelH - 28

	totalH := float64(len(s.items))*(langRowH+langRowGap) - langRowGap
	if totalH < 0 {
		totalH = 0
	}
	s.maxScroll = totalH - s.listH
	if s.maxScroll < 0 {
		s.maxScroll = 0
	}
	if s.scrollY > s.maxScroll {
		s.scrollY = s.maxScroll
	}

	// Scroll up/down buttons on right edge of panel
	scrollBtnX := s.panelX + s.panelW - 44
	s.upBtn.X, s.upBtn.Y, s.upBtn.W, s.upBtn.H = scrollBtnX, s.listY, 30, 32
	s.downBtn.X, s.downBtn.Y, s.downBtn.W, s.downBtn.H = scrollBtnX, s.listY+s.listH-32, 30, 32

	// Done button centered at bottom
	const btnW = 240.0
	const btnH = 42.0
	s.doneBtn.X = s.panelX + (s.panelW-btnW)/2
	s.doneBtn.Y = h - 44.0 - btnH
	s.doneBtn.W = btnW
	s.doneBtn.H = btnH
}

// OnMouseScroll scrolls the language list vertically.
func (s *LanguageSelectScreen) OnMouseScroll(deltaY float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scrollY += deltaY * 42.0
	if s.scrollY < 0 {
		s.scrollY = 0
	}
	if s.scrollY > s.maxScroll {
		s.scrollY = s.maxScroll
	}
}

// Update advances hover animations.
func (s *LanguageSelectScreen) Update(dt float64) {
	if dt > 0.05 {
		dt = 0.05
	}
	speed := dt * 10.0

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.items {
		item.hoverT = approach(item.hoverT, item.hovered, speed)
	}
	for _, b := range []*TextButton{s.upBtn, s.downBtn, s.doneBtn} {
		b.HoverT = approach(b.HoverT, b.Hovered, speed)
	}
}

// OnMouseMove updates hover state for visible language rows and buttons.
func (s *LanguageSelectScreen) OnMouseMove(x, y float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false

	inList := x >= s.listX && x <= s.listX+s.listW && y >= s.listY && y <= s.listY+s.listH
	for i, item := range s.items {
		ry := s.listY + float64(i)*(langRowH+langRowGap) - s.scrollY
		h := inList && y >= ry && y <= ry+langRowH
		if h != item.hovered {
			item.hovered = h
			changed = true
		}
	}

	for _, b := range []*TextButton{s.upBtn, s.downBtn, s.doneBtn} {
		h := x >= b.X && x <= b.X+b.W && y >= b.Y && y <= b.Y+b.H
		if h != b.Hovered {
			b.Hovered = h
			changed = true
		}
	}
	return changed
}

// OnMousePress selects a language or triggers scroll/done buttons.
func (s *LanguageSelectScreen) OnMousePress(x, y float64) {
	s.OnMouseMove(x, y)

	var selectedCode string
	var clickAction func()

	s.mu.Lock()
	for _, item := range s.items {
		if item.hovered {
			selectedCode = item.info.Code
			break
		}
	}
	if selectedCode == "" {
		for _, b := range []*TextButton{s.upBtn, s.downBtn, s.doneBtn} {
			if b.Hovered {
				clickAction = b.OnClick
				break
			}
		}
	}
	cbSelect := s.onSelect
	s.mu.Unlock()

	if selectedCode != "" && cbSelect != nil {
		cbSelect(selectedCode)
		return
	}
	if clickAction != nil {
		clickAction()
	}
}

// Draw renders the scrollable language selector.
func (s *LanguageSelectScreen) Draw(cc *gg.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	title := strings.TrimSuffix(s.tr.T("options.language"), "...")
	if s.titleFace != nil {
		cc.SetFont(s.titleFace)
	}
	cc.SetRGBA(0.96, 0.97, 1.0, 0.96)
	cc.DrawString(title, s.panelX, 66)

	if s.subFace != nil {
		cc.SetFont(s.subFace)
	}
	cc.SetRGBA(0.75, 0.80, 0.90, 0.85)
	cc.DrawString(s.tr.T("options.languageWarning"), s.panelX, 88)

	// Center glass panel
	cc.SetRGBA(0.06, 0.08, 0.12, 0.65)
	cc.DrawRoundedRectangle(s.panelX, s.panelY, s.panelW, s.panelH, 16)
	_ = cc.Fill()

	cc.SetRGBA(1, 1, 1, 0.12)
	cc.SetLineWidth(1)
	cc.DrawRoundedRectangle(s.panelX, s.panelY, s.panelW, s.panelH, 16)
	_ = cc.Stroke()

	activeCode := s.tr.CurrentCode()

	// Clip scrollable language rows inside the panel
	cc.Push()
	cc.ClipRect(s.listX, s.listY, s.listW, s.listH)
	for i, item := range s.items {
		ry := s.listY + float64(i)*(langRowH+langRowGap) - s.scrollY
		if ry+langRowH < s.listY || ry > s.listY+s.listH {
			continue
		}
		s.drawLangRow(cc, item, ry, item.info.Code == activeCode)
	}
	cc.Pop()

	// Scrollbar indicator
	if s.maxScroll > 0 {
		trackX := s.panelX + s.panelW - 33
		trackY := s.upBtn.Y + s.upBtn.H + 8
		trackH := s.downBtn.Y - 8 - trackY
		if trackH > 20 {
			cc.SetRGBA(1, 1, 1, 0.08)
			cc.DrawRoundedRectangle(trackX, trackY, 8, trackH, 4)
			_ = cc.Fill()

			ratio := s.listH / (s.listH + s.maxScroll)
			thumbH := trackH * ratio
			if thumbH < 24 {
				thumbH = 24
			}
			thumbY := trackY + (s.scrollY/s.maxScroll)*(trackH-thumbH)
			cc.SetRGBA(0.65, 0.78, 0.98, 0.65)
			cc.DrawRoundedRectangle(trackX, thumbY, 8, thumbH, 4)
			_ = cc.Fill()
		}
	}

	drawActionButton(cc, s.upBtn, s.metaFace, false)
	drawActionButton(cc, s.downBtn, s.metaFace, false)

	s.doneBtn.Label = s.tr.T("gui.done")
	drawActionButton(cc, s.doneBtn, s.btnFace, true)
}

func (s *LanguageSelectScreen) drawLangRow(cc *gg.Context, item *langItem, y float64, active bool) {
	t := item.hoverT
	if active {
		cc.SetRGBA(0.14, 0.25, 0.36, 0.90)
	} else {
		cc.SetRGBA(lerp(0.09, 0.14, t), lerp(0.11, 0.17, t), lerp(0.16, 0.24, t), lerp(0.72, 0.86, t))
	}
	cc.DrawRoundedRectangle(s.listX, y, s.listW, langRowH, 10)
	_ = cc.Fill()

	if active {
		cc.SetRGBA(0.45, 0.78, 1.0, 0.80)
		cc.SetLineWidth(1.5)
	} else {
		cc.SetRGBA(1, 1, 1, lerp(0.10, 0.28, t))
		cc.SetLineWidth(1)
	}
	cc.DrawRoundedRectangle(s.listX, y, s.listW, langRowH, 10)
	_ = cc.Stroke()

	if s.nameFace != nil {
		cc.SetFont(s.nameFace)
	}
	cc.SetRGBA(0.96, 0.97, 1.0, 0.96)
	cc.DrawStringAnchored(item.info.Name, s.listX+16, y+langRowH/2, 0.0, 0.5)

	if s.metaFace != nil {
		cc.SetFont(s.metaFace)
	}
	meta := item.info.Code
	if item.info.Region != "" {
		meta = item.info.Region + "  •  " + item.info.Code
	}
	cc.SetRGBA(0.72, 0.78, 0.88, 0.85)
	cc.DrawStringAnchored(meta, s.listX+s.listW-16, y+langRowH/2, 1.0, 0.5)
}

// Close releases font sources owned by this screen.
func (s *LanguageSelectScreen) Close() {
	if s.titleFont != nil {
		_ = s.titleFont.Close()
	}
	if s.uiFont != nil && s.uiFont != s.titleFont {
		_ = s.uiFont.Close()
	}
}