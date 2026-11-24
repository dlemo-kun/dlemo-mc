package ui

import (
	"fmt"
	"sync"

	"github.com/dlemo-kun/dlemo-mc/internal/i18n"
	"github.com/gogpu/gg"
)

// ServerEntry describes a multiplayer server shown in the server list.
type ServerEntry struct {
	Name    string
	Host    string
	Port    int
	MOTD    string
	Online  int
	Max     int
	IsUp    bool
	Loading bool
}

// ServerSelectActions defines callbacks triggered from the server selection screen.
type ServerSelectActions struct {
	OnConnect func(host string, port int)
	OnRefresh func(servers []*ServerEntry)
	OnBack    func()
}

// ServerSelectScreen displays the multiplayer server list with async ping updates,
// smooth scrolling via ScrollView, and modular CardWidget items.
type ServerSelectScreen struct {
	mu       sync.Mutex
	tr       *i18n.Manager
	w, h     float64
	servers  []*ServerEntry
	selected int
	actions  ServerSelectActions
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

// NewServerSelectScreen creates a new multiplayer server selection screen.
func NewServerSelectScreen(tr *i18n.Manager, servers []*ServerEntry, actions ServerSelectActions) *ServerSelectScreen {
	theme := NewUITheme()

	s := &ServerSelectScreen{
		tr:       tr,
		servers:  servers,
		selected: 0,
		actions:  actions,
		theme:    theme,
		dirty:    true,
	}
	if len(s.servers) == 0 {
		s.selected = -1
	}

	// 1. Scrollable server list container
	s.scrollView = NewScrollView(0, 0)
	s.cards = make([]*CardWidget, len(servers))
	for i, srv := range servers {
		idx := i
		sItem := srv
		sub := sItem.MOTD
		if sub == "" {
			sub = fmt.Sprintf("%s:%d", sItem.Host, sItem.Port)
		} else {
			sub = fmt.Sprintf("%s:%d  •  %s", sItem.Host, sItem.Port, sub)
		}

		card := NewCardWidget(sItem.Name, sub, "", i == s.selected, func() {
			s.selectServer(idx)
		})
		card.NameFace = theme.NameFace
		card.MetaFace = theme.MetaFace
		card.ShowStatus = true
		if sItem.Loading {
			card.StatusText = "..."
			card.StatusDotR, card.StatusDotG, card.StatusDotB = 0.75, 0.78, 0.85
		} else if sItem.IsUp {
			card.StatusText = fmt.Sprintf("%d/%d", sItem.Online, sItem.Max)
			card.StatusDotR, card.StatusDotG, card.StatusDotB = 0.35, 0.88, 0.56
		} else {
			card.StatusText = "Offline"
			card.StatusDotR, card.StatusDotG, card.StatusDotB = 0.90, 0.36, 0.36
		}

		s.cards[i] = card
		s.scrollView.Add(card)
	}

	// 2. Responsive footer buttons with FlexBox
	joinBtn := NewUIButton(s.resolveText("selectServer.select", "Join Server"), true, func() {
		s.mu.Lock()
		var target *ServerEntry
		if s.selected >= 0 && s.selected < len(s.servers) {
			target = s.servers[s.selected]
		}
		cb := s.actions.OnConnect
		s.mu.Unlock()
		if target != nil && cb != nil {
			cb(target.Host, target.Port)
		}
	})
	joinBtn.Key = "selectServer.select"
	joinBtn.Face = theme.BtnFace

	refreshBtn := NewUIButton(s.resolveText("selectServer.refresh", "Refresh"), false, func() {
		s.mu.Lock()
		list := append([]*ServerEntry(nil), s.servers...)
		cb := s.actions.OnRefresh
		s.mu.Unlock()
		if cb != nil {
			cb(list)
		}
	})
	refreshBtn.Key = "selectServer.refresh"
	refreshBtn.Face = theme.BtnFace

	backBtn := NewUIButton(s.resolveText("gui.back", "Back"), false, actions.OnBack)
	backBtn.Key = "gui.back"
	backBtn.Face = theme.BtnFace

	s.buttons = []*UIButton{joinBtn, refreshBtn, backBtn}
	s.footerBox = NewFlexBox(Row, 12, 0)
	for _, btn := range s.buttons {
		s.footerBox.Add(btn)
	}

	return s
}

// Servers returns a snapshot slice of configured servers.
func (s *ServerSelectScreen) Servers() []*ServerEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*ServerEntry(nil), s.servers...)
}

// SetServerStatus updates ping status fields safely from asynchronous ping routines.
func (s *ServerSelectScreen) SetServerStatus(index int, motd string, online, max int, up, loading bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.servers) {
		return
	}
	srv := s.servers[index]
	srv.MOTD = motd
	srv.Online = online
	srv.Max = max
	srv.IsUp = up
	srv.Loading = loading

	// Update corresponding card widget in scrollview
	if index < len(s.cards) {
		card := s.cards[index]
		sub := motd
		if sub == "" {
			sub = fmt.Sprintf("%s:%d", srv.Host, srv.Port)
		} else {
			sub = fmt.Sprintf("%s:%d  •  %s", srv.Host, srv.Port, sub)
		}
		card.Subtitle = sub

		card.ShowStatus = true
		if loading {
			card.StatusText = "..."
			card.StatusDotR, card.StatusDotG, card.StatusDotB = 0.75, 0.78, 0.85
		} else if up {
			card.StatusText = fmt.Sprintf("%d/%d", online, max)
			card.StatusDotR, card.StatusDotG, card.StatusDotB = 0.35, 0.88, 0.56
		} else {
			card.StatusText = "Offline"
			card.StatusDotR, card.StatusDotG, card.StatusDotB = 0.90, 0.36, 0.36
		}
	}
	s.dirty = true
}

func (s *ServerSelectScreen) selectServer(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selected = idx
	for i, card := range s.cards {
		card.Selected = (i == idx)
	}
	s.dirty = true
}

// SetViewport recalculates panel, scrollview, and footer layout responsively.
func (s *ServerSelectScreen) SetViewport(w, h float64) {
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
func (s *ServerSelectScreen) Update(dt float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scrollView.Update(dt)
	s.footerBox.Update(dt)
}

// OnMouseMove updates hover states in the scroll view and footer buttons.
func (s *ServerSelectScreen) OnMouseMove(x, y float64) bool {
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
func (s *ServerSelectScreen) OnMousePress(x, y float64) {
	if s.scrollView.OnMousePress(x, y) {
		return
	}
	_ = s.footerBox.OnMousePress(x, y)
}

// OnMouseScroll forwards wheel events to the smooth scroll container.
func (s *ServerSelectScreen) OnMouseScroll(deltaY float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scrollView.ScrollBy(deltaY * 42.0)
	s.dirty = true
}

// Draw renders the multiplayer server selection screen.
func (s *ServerSelectScreen) Draw(cc *gg.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Screen Header (Title & Subtitle)
	if s.theme.TitleFace != nil {
		cc.SetFont(s.theme.TitleFace)
	}
	cc.SetRGBA(0.96, 0.97, 1.0, 0.96)
	cc.DrawString(s.resolveText("multiplayer.title", "Play Multiplayer"), s.panelX, 66)

	if s.theme.SubFace != nil {
		cc.SetFont(s.theme.SubFace)
	}
	cc.SetRGBA(0.75, 0.80, 0.90, 0.85)
	cc.DrawString(s.resolveText("selectServer.defaultName", "Minecraft Server"), s.panelX, 88)

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

	// Render scrollable server cards
	s.scrollView.Draw(cc)

	// Refresh localized labels for footer buttons and draw
	for _, btn := range s.buttons {
		if btn.Key != "" && s.tr != nil {
			btn.Label = s.tr.T(btn.Key)
		}
	}
	s.footerBox.Draw(cc)
}

func (s *ServerSelectScreen) resolveText(key, fallback string) string {
	if s.tr != nil && key != "" {
		return s.tr.T(key)
	}
	return fallback
}

// Close satisfies the Screen interface.
func (s *ServerSelectScreen) Close() {}

// MarkDirty flags the screen as needing redraw.
func (s *ServerSelectScreen) MarkDirty() {
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()
}

// ConsumeDirty reports whether the screen needs redraw.
func (s *ServerSelectScreen) ConsumeDirty() bool {
	return true
}

// IsAnimating reports whether any transition is active.
func (s *ServerSelectScreen) IsAnimating() bool {
	return true
}
