package ui

import (
	"fmt"
	"sync"

	"github.com/dlemo-kun/dlemo-mc/internal/i18n"
	"github.com/gogpu/gg"
)

// WorldEntry describes a singleplayer world shown in the world list.
type WorldEntry struct {
	Name       string
	GameMode   string
	LastPlayed string
}

// WorldSelectActions defines callbacks triggered from the world selection screen.
type WorldSelectActions struct {
	OnPlay   func(world *WorldEntry)
	OnCreate func()
	OnBack   func()
}

// WorldSelectScreen displays the singleplayer world list and controls using UI 2.0 modular widgets.
type WorldSelectScreen struct {
	mu       sync.Mutex
	tr       *i18n.Manager
	w, h     float64
	worlds   []*WorldEntry
	selected int
	actions  WorldSelectActions
	theme    *UITheme

	// Modular UI 2.0 components
	scrollView *ScrollView
	cards      []*CardWidget
	footerBox  *FlexBox
	buttons    []*UIButton

	// Center glass panel coordinates
	panelX, panelY, panelW, panelH float64

	dirty bool
}

// NewWorldSelectScreen creates a new singleplayer world selection screen.
func NewWorldSelectScreen(tr *i18n.Manager, worlds []*WorldEntry, actions WorldSelectActions) *WorldSelectScreen {
	theme := NewUITheme()

	s := &WorldSelectScreen{
		tr:       tr,
		worlds:   worlds,
		selected: 0,
		actions:  actions,
		theme:    theme,
		dirty:    true,
	}
	if len(s.worlds) == 0 {
		s.selected = -1
	}

	// 1. Scrollable world list container
	s.scrollView = NewScrollView(0, 0)
	s.cards = make([]*CardWidget, len(worlds))
	for i, world := range worlds {
		idx := i
		wItem := world
		subtitle := fmt.Sprintf("%s  •  %s", wItem.GameMode, wItem.LastPlayed)
		card := NewCardWidget(wItem.Name, subtitle, "", i == s.selected, func() {
			s.selectWorld(idx)
		})
		card.NameFace = theme.NameFace
		card.MetaFace = theme.MetaFace
		s.cards[i] = card
		s.scrollView.Add(card)
	}

	// 2. Responsive footer buttons with FlexBox
	playBtn := NewUIButton(s.resolveText("selectWorld.select", "Play Selected World"), true, func() {
		s.mu.Lock()
		var target *WorldEntry
		if s.selected >= 0 && s.selected < len(s.worlds) {
			target = s.worlds[s.selected]
		}
		cb := s.actions.OnPlay
		s.mu.Unlock()
		if target != nil && cb != nil {
			cb(target)
		}
	})
	playBtn.Key = "selectWorld.select"
	playBtn.Face = theme.BtnFace

	createBtn := NewUIButton(s.resolveText("selectWorld.create", "Create New World"), false, actions.OnCreate)
	createBtn.Key = "selectWorld.create"
	createBtn.Face = theme.BtnFace

	backBtn := NewUIButton(s.resolveText("gui.back", "Back"), false, actions.OnBack)
	backBtn.Key = "gui.back"
	backBtn.Face = theme.BtnFace

	s.buttons = []*UIButton{playBtn, createBtn, backBtn}
	s.footerBox = NewFlexBox(Row, 12, 0)
	for _, btn := range s.buttons {
		s.footerBox.Add(btn)
	}

	return s
}

func (s *WorldSelectScreen) selectWorld(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selected = idx
	for i, card := range s.cards {
		card.Selected = (i == idx)
	}
	s.dirty = true
}

// SetViewport recalculates panel, scrollview, and footer layout responsively.
func (s *WorldSelectScreen) SetViewport(w, h float64) {
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

	// Configure ScrollView bounds inside the glass panel
	const scrollPad = 14.0
	s.scrollView.SetBounds(panelX+scrollPad, panelY+scrollPad, panelW-scrollPad*2, panelH-scrollPad*2)
	s.scrollView.LayoutVertical(68.0, 10.0)

	// Configure footer buttons row
	const btnH = 42.0
	btnY := h - 44.0 - btnH
	s.footerBox.SetBounds(panelX, btnY, panelW, btnH)
}

// Update advances smooth scrolling and hover animations.
func (s *WorldSelectScreen) Update(dt float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scrollView.Update(dt)
	s.footerBox.Update(dt)
}

// OnMouseMove updates hover states in the scroll view and footer buttons.
func (s *WorldSelectScreen) OnMouseMove(x, y float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.scrollView.OnMouseMove(x, y)
	fc := s.footerBox.OnMouseMove(x, y)
	if sc || fc {
		s.dirty = true
		return true
	}
	return false
}

// OnMousePress handles item selection or footer button clicks.
func (s *WorldSelectScreen) OnMousePress(x, y float64) {
	if s.scrollView.OnMousePress(x, y) {
		return
	}
	_ = s.footerBox.OnMousePress(x, y)
}

// OnMouseScroll forwards wheel events to the smooth scroll container.
func (s *WorldSelectScreen) OnMouseScroll(deltaY float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scrollView.ScrollBy(deltaY * 42.0)
	s.dirty = true
}

// Draw renders the singleplayer world selection screen.
func (s *WorldSelectScreen) Draw(cc *gg.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Screen Header (Title & Subtitle)
	if s.theme.TitleFace != nil {
		cc.SetFont(s.theme.TitleFace)
	}
	cc.SetRGBA(0.96, 0.97, 1.0, 0.96)
	cc.DrawString(s.resolveText("selectWorld.title", "Select World"), s.panelX, 66)

	if s.theme.SubFace != nil {
		cc.SetFont(s.theme.SubFace)
	}
	cc.SetRGBA(0.75, 0.80, 0.90, 0.85)
	cc.DrawString(s.resolveText("selectWorld.newWorld", "Create New World"), s.panelX, 88)

	// Center glass panel container
	if s.panelH > 100 {
		cc.SetRGBA(0.06, 0.08, 0.12, 0.65)
		cc.DrawRoundedRectangle(s.panelX, s.panelY, s.panelW, s.panelH, 16)
		_ = cc.Fill()

		cc.SetRGBA(1, 1, 1, 0.12)
		cc.SetLineWidth(1)
		cc.DrawRoundedRectangle(s.panelX, s.panelY, s.panelW, s.panelH, 16)
		_ = cc.Stroke()
	}

	// Render scrollable card list
	s.scrollView.Draw(cc)

	// Refresh localized labels for footer buttons and draw
	for _, btn := range s.buttons {
		if btn.Key != "" && s.tr != nil {
			btn.Label = s.tr.T(btn.Key)
		}
	}
	s.footerBox.Draw(cc)
}

func (s *WorldSelectScreen) resolveText(key, fallback string) string {
	if s.tr != nil && key != "" {
		return s.tr.T(key)
	}
	return fallback
}

// Close satisfies the Screen interface.
func (s *WorldSelectScreen) Close() {}

// MarkDirty flags the screen as needing redraw.
func (s *WorldSelectScreen) MarkDirty() {
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()
}

// ConsumeDirty reports whether the screen needs redraw.
func (s *WorldSelectScreen) ConsumeDirty() bool {
	return true
}

// IsAnimating reports whether any transition is active.
func (s *WorldSelectScreen) IsAnimating() bool {
	return true
}
