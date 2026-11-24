package ui

import (
	"sync"
	"testing"

	"github.com/dlemo-kun/dlemo-mc/internal/i18n"
	"github.com/gogpu/gg"
)

// Ensure all redesigned screens implement the Screen interface.
var (
	_ Screen          = (*TitleMenu)(nil)
	_ Screen          = (*WorldSelectScreen)(nil)
	_ Screen          = (*ServerSelectScreen)(nil)
	_ Screen          = (*LanguageSelectScreen)(nil)
	_ Scrollable      = (*WorldSelectScreen)(nil)
	_ Scrollable      = (*ServerSelectScreen)(nil)
	_ Scrollable      = (*LanguageSelectScreen)(nil)
	_ DirtyScreen     = (*TitleMenu)(nil)
	_ DirtyScreen     = (*WorldSelectScreen)(nil)
	_ DirtyScreen     = (*ServerSelectScreen)(nil)
	_ DirtyScreen     = (*LanguageSelectScreen)(nil)
	_ MarkDirtyScreen = (*TitleMenu)(nil)
	_ MarkDirtyScreen = (*WorldSelectScreen)(nil)
	_ MarkDirtyScreen = (*ServerSelectScreen)(nil)
	_ MarkDirtyScreen = (*LanguageSelectScreen)(nil)
)

func TestTitleMenu(t *testing.T) {
	var singleplayerClicked, quitClicked, langClicked bool
	actions := TitleMenuActions{
		OnSingleplayer: func() { singleplayerClicked = true },
		OnQuit:         func() { quitClicked = true },
		OnLanguage:     func() { langClicked = true },
	}

	tr := i18n.NewManager("en_us")
	menu := NewTitleMenu(tr, actions)

	if menu == nil {
		t.Fatal("NewTitleMenu returned nil")
	}

	// 1. Viewport test
	menu.SetViewport(1280, 720)
	if menu.Width != 1280 || menu.Height != 720 {
		t.Errorf("expected viewport 1280x720, got %fx%f", menu.Width, menu.Height)
	}

	// 2. VBox button interaction
	if len(menu.buttons) != 5 {
		t.Fatalf("expected 5 title buttons, got %d", len(menu.buttons))
	}
	spBtn := menu.buttons[0]
	bx, by, bw, bh := spBtn.Bounds()
	if bw <= 0 || bh <= 0 {
		t.Errorf("invalid button bounds: %f, %f, %f, %f", bx, by, bw, bh)
	}

	// Hover and click Singleplayer
	menu.OnMouseMove(bx+10, by+10)
	if !spBtn.Hovered {
		t.Errorf("expected Singleplayer button to be hovered")
	}
	menu.OnMousePress(bx+10, by+10)
	if !singleplayerClicked {
		t.Errorf("expected Singleplayer callback to be called")
	}

	// 3. HBox icon button interaction
	if len(menu.iconButtons) != 3 {
		t.Fatalf("expected 3 icon buttons, got %d", len(menu.iconButtons))
	}
	langBtn := menu.iconButtons[0]
	lx, ly, _, _ := langBtn.Bounds()
	menu.OnMouseMove(lx+5, ly+5)
	if !langBtn.Hovered {
		t.Errorf("expected Language icon button to be hovered")
	}
	menu.OnMousePress(lx+5, ly+5)
	if !langClicked {
		t.Errorf("expected Language callback to be called")
	}

	// 4. Update and Draw
	menu.Update(0.016)
	cc := gg.NewContext(1280, 720)
	menu.Draw(cc)

	// 5. Screen interface compliance
	menu.MarkDirty()
	if !menu.ConsumeDirty() {
		t.Errorf("expected ConsumeDirty to return true")
	}
	if !menu.IsAnimating() {
		t.Errorf("expected IsAnimating to return true")
	}
	menu.Close()
	_ = quitClicked
}

func TestWorldSelectScreen(t *testing.T) {
	worlds := []*WorldEntry{
		{Name: "Survival World", GameMode: "Survival", LastPlayed: "Today"},
		{Name: "Creative Hub", GameMode: "Creative", LastPlayed: "Yesterday"},
	}

	var playedWorld *WorldEntry
	var createClicked, backClicked bool

	actions := WorldSelectActions{
		OnPlay:   func(w *WorldEntry) { playedWorld = w },
		OnCreate: func() { createClicked = true },
		OnBack:   func() { backClicked = true },
	}

	tr := i18n.NewManager("en_us")
	screen := NewWorldSelectScreen(tr, worlds, actions)
	screen.SetViewport(1024, 768)

	if len(screen.cards) != 2 {
		t.Fatalf("expected 2 world cards, got %d", len(screen.cards))
	}

	// Check initial selection
	if !screen.cards[0].Selected {
		t.Errorf("expected first world to be selected initially")
	}

	// Click second world
	cx, cy, _, _ := screen.cards[1].Bounds()
	screen.OnMouseMove(cx+10, cy+10)
	screen.OnMousePress(cx+10, cy+10)
	if !screen.cards[1].Selected || screen.cards[0].Selected {
		t.Errorf("expected second card selected and first card unselected")
	}

	// Click Play button
	playBtn := screen.buttons[0]
	px, py, _, _ := playBtn.Bounds()
	screen.OnMouseMove(px+5, py+5)
	screen.OnMousePress(px+5, py+5)
	if playedWorld == nil || playedWorld.Name != "Creative Hub" {
		t.Errorf("expected Creative Hub to be played, got %v", playedWorld)
	}

	// Click Back button
	backBtn := screen.buttons[2]
	bx, by, _, _ := backBtn.Bounds()
	screen.OnMouseMove(bx+5, by+5)
	screen.OnMousePress(bx+5, py+5)
	if !backClicked {
		t.Errorf("expected back callback to be invoked")
	}

	// Scroll wheel test
	screen.OnMouseScroll(-2.0)
	screen.Update(0.016)

	// Draw verification
	cc := gg.NewContext(1024, 768)
	screen.Draw(cc)
	screen.Close()
	_ = createClicked
}

func TestServerSelectScreen(t *testing.T) {
	servers := []*ServerEntry{
		{Name: "Hub 1", Host: "127.0.0.1", Port: 25565},
		{Name: "Survival 1", Host: "127.0.0.1", Port: 25566},
	}

	var connectedHost string
	var connectedPort int
	var refreshed bool
	var backClicked bool

	actions := ServerSelectActions{
		OnConnect: func(host string, port int) {
			connectedHost = host
			connectedPort = port
		},
		OnRefresh: func(s []*ServerEntry) { refreshed = true },
		OnBack:    func() { backClicked = true },
	}

	tr := i18n.NewManager("en_us")
	screen := NewServerSelectScreen(tr, servers, actions)
	screen.SetViewport(1024, 768)

	// Test concurrent async ping updates
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(iter int) {
			defer wg.Done()
			screen.SetServerStatus(0, "Awesome Server MOTD", 12+iter, 100, true, false)
		}(i)
		go func(iter int) {
			defer wg.Done()
			screen.SetServerStatus(1, "Survival Realm", 5+iter, 50, true, false)
		}(i)
	}
	wg.Wait()

	// Verify server status card updated
	if screen.cards[0].StatusText == "Offline" || screen.cards[0].StatusText == "..." {
		t.Errorf("expected online status on card 0, got %s", screen.cards[0].StatusText)
	}

	// Click Join Server button
	joinBtn := screen.buttons[0]
	jx, jy, _, _ := joinBtn.Bounds()
	screen.OnMouseMove(jx+5, jy+5)
	screen.OnMousePress(jx+5, jy+5)
	if connectedHost != "127.0.0.1" || connectedPort != 25565 {
		t.Errorf("expected connection to 127.0.0.1:25565, got %s:%d", connectedHost, connectedPort)
	}

	// Click Refresh button
	refreshBtn := screen.buttons[1]
	rx, ry, _, _ := refreshBtn.Bounds()
	screen.OnMouseMove(rx+5, ry+5)
	screen.OnMousePress(rx+5, ry+5)
	if !refreshed {
		t.Errorf("expected refresh callback to be called")
	}

	screen.OnMouseScroll(1.0)
	screen.Update(0.016)

	cc := gg.NewContext(1024, 768)
	screen.Draw(cc)
	screen.Close()
	_ = backClicked
}

func TestLanguageSelectScreen(t *testing.T) {
	tr := i18n.NewManager("en_us")

	var selectedLang string
	var backClicked bool

	screen := NewLanguageSelectScreen(
		tr,
		func(code string) { selectedLang = code },
		func() { backClicked = true },
	)
	screen.SetViewport(1024, 768)

	if len(screen.cards) == 0 {
		t.Fatalf("expected language cards from tr, got 0")
	}

	// Click first card
	c0 := screen.cards[0]
	cx, cy, _, _ := c0.Bounds()
	screen.OnMouseMove(cx+10, cy+10)
	screen.OnMousePress(cx+10, cy+10)
	if selectedLang == "" {
		t.Errorf("expected language code to be selected")
	}

	// Done button
	doneX, doneY, _, _ := screen.doneBtn.Bounds()
	screen.OnMouseMove(doneX+5, doneY+5)
	screen.OnMousePress(doneX+5, doneY+5)
	if !backClicked {
		t.Errorf("expected back callback from Done button")
	}

	screen.OnMouseScroll(1.0)
	screen.Update(0.016)

	cc := gg.NewContext(1024, 768)
	screen.Draw(cc)
	screen.Close()
}
