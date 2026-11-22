package ui

import (
	"fmt"
	"sync"

	"github.com/dlemo-kun/dlemo-mc/internal/i18n"
	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
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

	x, y, w, h float64
	hovered    bool
	hoverT     float64
}

// ServerSelectActions defines callbacks triggered from the server selection screen.
type ServerSelectActions struct {
	OnConnect func(host string, port int)
	OnRefresh func(servers []*ServerEntry)
	OnBack    func()
}

// ServerSelectScreen displays the multiplayer server list and connection controls.
type ServerSelectScreen struct {
	mu       sync.Mutex
	tr       *i18n.Manager
	w, h     float64
	servers  []*ServerEntry
	selected int
	buttons  []*TextButton
	actions  ServerSelectActions

	titleFont *FontStack
	uiFont    *FontStack
	titleFace text.Face
	subFace   text.Face
	nameFace  text.Face
	metaFace  text.Face
	btnFace   text.Face

	dirty bool
}

// NewServerSelectScreen creates a new multiplayer server selection screen.
func NewServerSelectScreen(tr *i18n.Manager, servers []*ServerEntry, actions ServerSelectActions) *ServerSelectScreen {
	s := &ServerSelectScreen{
		tr:       tr,
		servers:  servers,
		selected: 0,
		actions:  actions,
		dirty:    true,
	}
	if len(s.servers) == 0 {
		s.selected = -1
	}

	titleFont, uiFont := loadFont()

	s.titleFont = titleFont
	s.uiFont = uiFont

	if s.titleFont != nil {
		s.titleFace = s.titleFont.Face(32)
	}
	if s.uiFont != nil {
		s.subFace = s.uiFont.Face(13)
		s.nameFace = s.uiFont.Face(16)
		s.metaFace = s.uiFont.Face(13)
		s.btnFace = s.uiFont.Face(15)
	}

	s.buttons = []*TextButton{
		{
			Key:   "selectServer.select",
			Label: "Join Server",
			OnClick: func() {
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
			},
		},
		{
			Key:   "selectServer.refresh",
			Label: "Refresh",
			OnClick: func() {
				s.mu.Lock()
				list := append([]*ServerEntry(nil), s.servers...)
				cb := s.actions.OnRefresh
				s.mu.Unlock()
				if cb != nil {
					cb(list)
				}
			},
		},
		{
			Key:     "gui.back",
			Label:   "Back",
			OnClick: actions.OnBack,
		},
	}

	return s
}

// Servers returns a snapshot slice of the configured servers.
func (s *ServerSelectScreen) Servers() []*ServerEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*ServerEntry(nil), s.servers...)
}

// SetServerStatus updates ping status fields for a server entry safely from any goroutine.
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
}

// SetViewport recalculates server card and footer button positions.
func (s *ServerSelectScreen) SetViewport(w, h float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w <= 0 || h <= 0 || (s.w == w && s.h == h) {
		return
	}
	s.w, s.h = w, h

	panelX := 44.0
	panelW := w - 88.0
	if panelW > 760 {
		panelW = 760
		panelX = (w - panelW) / 2
	}

	cardY := 124.0
	const cardH = 68.0
	const cardGap = 10.0
	for i, srv := range s.servers {
		srv.x = panelX + 16
		srv.y = cardY + float64(i)*(cardH+cardGap)
		srv.w = panelW - 32
		srv.h = cardH
	}

	const btnH = 42.0
	const gap = 12.0
	btnY := h - 44.0 - btnH
	btnW := (panelW - float64(len(s.buttons)-1)*gap) / float64(len(s.buttons))
	for i, btn := range s.buttons {
		btn.X = panelX + float64(i)*(btnW+gap)
		btn.Y = btnY
		btn.W = btnW
		btn.H = btnH
	}
}

// Update advances hover animations.
func (s *ServerSelectScreen) Update(dt float64) {
	if dt > 0.05 {
		dt = 0.05
	}
	speed := dt * 10.0

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, srv := range s.servers {
		srv.hoverT = approach(srv.hoverT, srv.hovered, speed)
	}
	for _, btn := range s.buttons {
		btn.HoverT = approach(btn.HoverT, btn.Hovered, speed)
	}
}

// OnMouseMove updates hover state for cards and action buttons.
func (s *ServerSelectScreen) OnMouseMove(x, y float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for _, srv := range s.servers {
		h := x >= srv.x && x <= srv.x+srv.w && y >= srv.y && y <= srv.y+srv.h
		if h != srv.hovered {
			srv.hovered = h
			changed = true
		}
	}
	for _, btn := range s.buttons {
		h := x >= btn.X && x <= btn.X+btn.W && y >= btn.Y && y <= btn.Y+btn.H
		if h != btn.Hovered {
			btn.Hovered = h
			changed = true
		}
	}
	return changed
}

// OnMousePress selects a server card or triggers a footer button action.
func (s *ServerSelectScreen) OnMousePress(x, y float64) {
	s.OnMouseMove(x, y)

	var clickAction func()
	s.mu.Lock()
	for i, srv := range s.servers {
		if srv.hovered {
			s.selected = i
			s.mu.Unlock()
			return
		}
	}
	for _, btn := range s.buttons {
		if btn.Hovered {
			clickAction = btn.OnClick
			break
		}
	}
	s.mu.Unlock()

	if clickAction != nil {
		clickAction()
	}
}

// Draw renders the multiplayer server selection UI.
func (s *ServerSelectScreen) Draw(cc *gg.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	panelX := 44.0
	panelW := s.w - 88.0
	if panelW > 760 {
		panelW = 760
		panelX = (s.w - panelW) / 2
	}

	if s.titleFace != nil {
		cc.SetFont(s.titleFace)
	}
	cc.SetRGBA(0.96, 0.97, 1.0, 0.96)
	cc.DrawString(s.resolveText("multiplayer.title", "Play Multiplayer"), panelX, 66)

	if s.subFace != nil {
		cc.SetFont(s.subFace)
	}
	cc.SetRGBA(0.75, 0.80, 0.90, 0.85)
	cc.DrawString(s.resolveText("selectServer.defaultName", "Minecraft Server"), panelX, 88)

	panelY := 108.0
	panelH := s.h - panelY - 104.0
	if panelH > 120 {
		cc.SetRGBA(0.06, 0.08, 0.12, 0.65)
		cc.DrawRoundedRectangle(panelX, panelY, panelW, panelH, 16)
		_ = cc.Fill()

		cc.SetRGBA(1, 1, 1, 0.12)
		cc.SetLineWidth(1)
		cc.DrawRoundedRectangle(panelX, panelY, panelW, panelH, 16)
		_ = cc.Stroke()
	}

	for i, srv := range s.servers {
		s.drawServerCard(cc, srv, i == s.selected)
	}

	for i, btn := range s.buttons {
		btn.Label = s.resolveText(btn.Key, btn.Label)
		drawActionButton(cc, btn, s.btnFace, i == 0)
	}
}

func (s *ServerSelectScreen) resolveText(key, fallback string) string {
	if s.tr != nil && key != "" {
		return s.tr.T(key)
	}
	return fallback
}

func (s *ServerSelectScreen) drawServerCard(cc *gg.Context, srv *ServerEntry, selected bool) {
	t := srv.hoverT
	if selected {
		cc.SetRGBA(0.14, 0.22, 0.34, 0.86)
	} else {
		cc.SetRGBA(lerp(0.09, 0.14, t), lerp(0.11, 0.17, t), lerp(0.16, 0.24, t), lerp(0.72, 0.86, t))
	}
	cc.DrawRoundedRectangle(srv.x, srv.y, srv.w, srv.h, 12)
	_ = cc.Fill()

	if selected {
		cc.SetRGBA(0.45, 0.72, 1.0, 0.75)
		cc.SetLineWidth(1.5)
	} else {
		cc.SetRGBA(1, 1, 1, lerp(0.10, 0.28, t))
		cc.SetLineWidth(1)
	}
	cc.DrawRoundedRectangle(srv.x, srv.y, srv.w, srv.h, 12)
	_ = cc.Stroke()

	if s.nameFace != nil {
		cc.SetFont(s.nameFace)
	}
	cc.SetRGBA(0.96, 0.97, 1.0, 0.96)
	cc.DrawString(srv.Name, srv.x+18, srv.y+27)

	if s.metaFace != nil {
		cc.SetFont(s.metaFace)
	}
	sub := srv.MOTD
	if sub == "" {
		sub = fmt.Sprintf("%s:%d", srv.Host, srv.Port)
	} else {
		sub = fmt.Sprintf("%s:%d  •  %s", srv.Host, srv.Port, sub)
	}
	cc.SetRGBA(0.72, 0.77, 0.87, 0.88)
	cc.DrawString(sub, srv.x+18, srv.y+49)

	statusText := "Offline"
	dotR, dotG, dotB := 0.90, 0.36, 0.36
	if srv.Loading {
		statusText = "..."
		dotR, dotG, dotB = 0.75, 0.78, 0.85
	} else if srv.IsUp {
		statusText = fmt.Sprintf("%d/%d", srv.Online, srv.Max)
		dotR, dotG, dotB = 0.35, 0.88, 0.56
	}

	cc.SetRGBA(dotR, dotG, dotB, 0.95)
	cc.DrawCircle(srv.x+srv.w-24, srv.y+srv.h/2, 5)
	_ = cc.Fill()

	cc.SetRGBA(0.88, 0.92, 0.98, 0.92)
	cc.DrawStringAnchored(statusText, srv.x+srv.w-38, srv.y+srv.h/2, 1.0, 0.5)
}

// MarkDirty invalidates the retained canvas after activation or viewport changes.
func (s *ServerSelectScreen) MarkDirty() {
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()
}

func (s *ServerSelectScreen) ConsumeDirty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirty {
		s.dirty = false
		return true
	}
	return false
}

func (s *ServerSelectScreen) IsAnimating() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, srv := range s.servers {
		if (srv.hovered && srv.hoverT < 1) || (!srv.hovered && srv.hoverT > 0) {
			return true
		}
	}
	for _, btn := range s.buttons {
		if (btn.Hovered && btn.HoverT < 1) || (!btn.Hovered && btn.HoverT > 0) {
			return true
		}
	}
	return false
}

// Close releases no owned resources: fonts and the GPU canvas are shared and
// are released by CloseSharedFonts and Renderer.Close respectively.
func (s *ServerSelectScreen) Close() {}
