package ui

import (
	"strings"
	"sync"

	"github.com/dlemo-kun/dlemo-mc/internal/i18n"
	"github.com/gogpu/gg"
)

// LanguageSelectScreen displays all languages discovered with smooth scrolling,
// modular CardWidget items, and a unified action button.
type LanguageSelectScreen struct {
	mu       sync.Mutex
	tr       *i18n.Manager
	w, h     float64
	onSelect func(code string)
	onBack   func()
	theme    *UITheme

	// Modular UI 2.0 components
	scrollView *ScrollView
	cards      []*CardWidget
	doneBtn    *UIButton

	// Center glass panel bounds
	panelX, panelY, panelW, panelH float64

	dirty bool
}

// NewLanguageSelectScreen creates the language selector populated from tr.
func NewLanguageSelectScreen(tr *i18n.Manager, onSelect func(code string), onBack func()) *LanguageSelectScreen {
	theme := NewUITheme()

	s := &LanguageSelectScreen{
		tr:       tr,
		onSelect: onSelect,
		onBack:   onBack,
		theme:    theme,
		dirty:    true,
	}

	// 1. Scrollable language card list
	s.scrollView = NewScrollView(0, 0)
	langs := tr.Languages()
	activeCode := tr.CurrentCode()

	s.cards = make([]*CardWidget, len(langs))
	for i, l := range langs {
		lang := l
		displayName := lang.Name
		if lang.Region != "" {
			displayName += " (" + lang.Region + ")"
		}

		card := NewCardWidget(displayName, "", lang.Code, lang.Code == activeCode, func() {
			s.selectLanguage(lang.Code)
		})
		card.NameFace = theme.NameFace
		card.MetaFace = theme.MetaFace
		card.Radius = 10.0

		s.cards[i] = card
		s.scrollView.Add(card)
	}

	// 2. Action button centered at bottom
	s.doneBtn = NewUIButton(tr.T("gui.done"), true, onBack)
	s.doneBtn.Key = "gui.done"
	s.doneBtn.Face = theme.BtnFace

	return s
}

func (s *LanguageSelectScreen) selectLanguage(code string) {
	s.mu.Lock()
	for _, card := range s.cards {
		card.Selected = (card.Meta == code)
	}
	cb := s.onSelect
	s.dirty = true
	s.mu.Unlock()

	if cb != nil {
		cb(code)
	}
}

// SetViewport recalculates panel, scrollview, and button layout responsively.
func (s *LanguageSelectScreen) SetViewport(w, h float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w <= 0 || h <= 0 || (s.w == w && s.h == h) {
		return
	}
	s.w, s.h = w, h
	s.dirty = true

	panelX := 44.0
	panelW := w - 88.0
	if panelW > 760 {
		panelW = 760
		panelX = (w - panelW) / 2
	}

	panelY := 108.0
	panelH := h - panelY - 104.0
	if panelH < 120 {
		panelH = 120
	}

	s.panelX, s.panelY, s.panelW, s.panelH = panelX, panelY, panelW, panelH

	// Configure ScrollView bounds inside the panel matching World/Server select screens
	const scrollPad = 14.0
	s.scrollView.SetBounds(panelX+scrollPad, panelY+scrollPad, panelW-scrollPad*2, panelH-scrollPad*2)
	s.scrollView.LayoutVertical(50.0, 8.0)

	// Done button centered at bottom
	const doneW = 240.0
	const doneH = 42.0
	s.doneBtn.SetBounds(panelX+(panelW-doneW)/2, h-44.0-doneH, doneW, doneH)
}

// Update advances smooth scrolling and hover animations.
func (s *LanguageSelectScreen) Update(dt float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scrollView.Update(dt)
	s.doneBtn.Update(dt)
}

// OnMouseMove updates hover states in the scroll view and button.
func (s *LanguageSelectScreen) OnMouseMove(x, y float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.scrollView.OnMouseMove(x, y)
	dn := s.doneBtn.OnMouseMove(x, y)
	if sc || dn {
		s.dirty = true
		return true
	}
	return false
}

// OnMousePress handles item selection or button clicks.
func (s *LanguageSelectScreen) OnMousePress(x, y float64) {
	if s.scrollView.OnMousePress(x, y) {
		return
	}
	_ = s.doneBtn.OnMousePress(x, y)
}

// OnMouseScroll forwards wheel events to the smooth scroll container.
func (s *LanguageSelectScreen) OnMouseScroll(deltaY float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scrollView.ScrollBy(deltaY * 42.0)
	s.dirty = true
}

// Draw renders the language selection screen.
func (s *LanguageSelectScreen) Draw(cc *gg.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Header: Title & Subtitle
	title := strings.TrimSuffix(s.tr.T("options.language"), "...")
	if s.theme.TitleFace != nil {
		cc.SetFont(s.theme.TitleFace)
	}
	cc.SetRGBA(0.96, 0.97, 1.0, 0.96)
	cc.DrawString(title, s.panelX, 66)

	if s.theme.SubFace != nil {
		cc.SetFont(s.theme.SubFace)
	}
	cc.SetRGBA(0.75, 0.80, 0.90, 0.85)
	cc.DrawString(s.tr.T("options.languageWarning"), s.panelX, 88)

	// Center glass panel container
	cc.SetRGBA(0.06, 0.08, 0.12, 0.65)
	cc.DrawRoundedRectangle(s.panelX, s.panelY, s.panelW, s.panelH, 16)
	_ = cc.Fill()

	cc.SetRGBA(1, 1, 1, 0.12)
	cc.SetLineWidth(1)
	cc.DrawRoundedRectangle(s.panelX, s.panelY, s.panelW, s.panelH, 16)
	_ = cc.Stroke()

	// Scrollable language cards
	s.scrollView.Draw(cc)

	// Done button
	s.doneBtn.Label = s.tr.T("gui.done")
	s.doneBtn.Draw(cc)
}

// Close satisfies the Screen interface.
func (s *LanguageSelectScreen) Close() {}

// MarkDirty flags the screen as needing redraw.
func (s *LanguageSelectScreen) MarkDirty() {
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()
}

// ConsumeDirty reports whether the screen needs redraw.
func (s *LanguageSelectScreen) ConsumeDirty() bool {
	return true
}

// IsAnimating reports whether any transition is active.
func (s *LanguageSelectScreen) IsAnimating() bool {
	return true
}
