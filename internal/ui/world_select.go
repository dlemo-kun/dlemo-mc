package ui

import (
	"fmt"
	"sync"

	"github.com/dlemo-kun/dlemo-mc/internal/i18n"
	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

// WorldEntry describes a singleplayer world shown in the world list.
type WorldEntry struct {
	Name       string
	GameMode   string
	LastPlayed string

	x, y, w, h float64
	hovered    bool
	hoverT     float64
}

// WorldSelectActions defines callbacks triggered from the world selection screen.
type WorldSelectActions struct {
	OnPlay   func(world *WorldEntry)
	OnCreate func()
	OnBack   func()
}

// WorldSelectScreen displays the singleplayer world list and controls.
type WorldSelectScreen struct {
	mu       sync.Mutex
	tr       *i18n.Manager
	w, h     float64
	worlds   []*WorldEntry
	selected int
	buttons  []*TextButton
	actions  WorldSelectActions

	titleFont *FontStack
	uiFont    *FontStack
	titleFace text.Face
	subFace   text.Face
	nameFace  text.Face
	metaFace  text.Face
	btnFace   text.Face

	dirty bool
}

// NewWorldSelectScreen creates a new singleplayer world selection screen.
func NewWorldSelectScreen(tr *i18n.Manager, worlds []*WorldEntry, actions WorldSelectActions) *WorldSelectScreen {
	s := &WorldSelectScreen{
		tr:       tr,
		worlds:   worlds,
		selected: 0,
		actions:  actions,
		dirty:    true,
	}
	if len(s.worlds) == 0 {
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
			Key:   "selectWorld.select",
			Label: "Play Selected World",
			OnClick: func() {
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
			},
		},
		{
			Key:     "selectWorld.create",
			Label:   "Create New World",
			OnClick: actions.OnCreate,
		},
		{
			Key:     "gui.back",
			Label:   "Back",
			OnClick: actions.OnBack,
		},
	}

	return s
}

// SetViewport recalculates world card and footer button positions.
func (s *WorldSelectScreen) SetViewport(w, h float64) {
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
	for i, world := range s.worlds {
		world.x = panelX + 16
		world.y = cardY + float64(i)*(cardH+cardGap)
		world.w = panelW - 32
		world.h = cardH
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
func (s *WorldSelectScreen) Update(dt float64) {
	if dt > 0.05 {
		dt = 0.05
	}
	speed := dt * 10.0

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, world := range s.worlds {
		world.hoverT = approach(world.hoverT, world.hovered, speed)
	}
	for _, btn := range s.buttons {
		btn.HoverT = approach(btn.HoverT, btn.Hovered, speed)
	}
}

// OnMouseMove updates hover state for world cards and buttons.
func (s *WorldSelectScreen) OnMouseMove(x, y float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for _, world := range s.worlds {
		h := x >= world.x && x <= world.x+world.w && y >= world.y && y <= world.y+world.h
		if h != world.hovered {
			world.hovered = h
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

// OnMousePress selects a world card or triggers a footer button action.
func (s *WorldSelectScreen) OnMousePress(x, y float64) {
	s.OnMouseMove(x, y)

	var clickAction func()
	s.mu.Lock()
	for i, world := range s.worlds {
		if world.hovered {
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

// Draw renders the singleplayer world selection UI.
func (s *WorldSelectScreen) Draw(cc *gg.Context) {
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
	cc.DrawString(s.resolveText("selectWorld.title", "Select World"), panelX, 66)

	if s.subFace != nil {
		cc.SetFont(s.subFace)
	}
	cc.SetRGBA(0.75, 0.80, 0.90, 0.85)
	cc.DrawString(s.resolveText("selectWorld.newWorld", "Create New World"), panelX, 88)

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

	for i, world := range s.worlds {
		s.drawWorldCard(cc, world, i == s.selected)
	}

	for i, btn := range s.buttons {
		btn.Label = s.resolveText(btn.Key, btn.Label)
		drawActionButton(cc, btn, s.btnFace, i == 0)
	}
}

func (s *WorldSelectScreen) resolveText(key, fallback string) string {
	if s.tr != nil && key != "" {
		return s.tr.T(key)
	}
	return fallback
}

func (s *WorldSelectScreen) drawWorldCard(cc *gg.Context, world *WorldEntry, selected bool) {
	t := world.hoverT
	if selected {
		cc.SetRGBA(0.14, 0.22, 0.34, 0.86)
	} else {
		cc.SetRGBA(lerp(0.09, 0.14, t), lerp(0.11, 0.17, t), lerp(0.16, 0.24, t), lerp(0.72, 0.86, t))
	}
	cc.DrawRoundedRectangle(world.x, world.y, world.w, world.h, 12)
	_ = cc.Fill()

	if selected {
		cc.SetRGBA(0.45, 0.72, 1.0, 0.75)
		cc.SetLineWidth(1.5)
	} else {
		cc.SetRGBA(1, 1, 1, lerp(0.10, 0.28, t))
		cc.SetLineWidth(1)
	}
	cc.DrawRoundedRectangle(world.x, world.y, world.w, world.h, 12)
	_ = cc.Stroke()

	if s.nameFace != nil {
		cc.SetFont(s.nameFace)
	}
	cc.SetRGBA(0.96, 0.97, 1.0, 0.96)
	cc.DrawString(world.Name, world.x+18, world.y+27)

	if s.metaFace != nil {
		cc.SetFont(s.metaFace)
	}
	cc.SetRGBA(0.72, 0.77, 0.87, 0.88)
	cc.DrawString(fmt.Sprintf("%s  •  %s", world.GameMode, world.LastPlayed), world.x+18, world.y+49)
}

// Close releases no owned resources: fonts are application-shared resources.
func (s *WorldSelectScreen) Close() {}

// MarkDirty invalidates the retained canvas after activation or viewport changes.
func (s *WorldSelectScreen) MarkDirty() {
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()
}

func (s *WorldSelectScreen) ConsumeDirty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirty {
		s.dirty = false
		return true
	}
	return false
}

func (s *WorldSelectScreen) IsAnimating() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, world := range s.worlds {
		if (world.hovered && world.hoverT < 1) || (!world.hovered && world.hoverT > 0) {
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
