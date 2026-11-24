package app

import (
	"encoding/json"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlemo-kun/dlemo-mc/internal/client"
	"github.com/dlemo-kun/dlemo-mc/internal/config"
	"github.com/dlemo-kun/dlemo-mc/internal/i18n"
	mcnet "github.com/dlemo-kun/dlemo-mc/internal/net"
	"github.com/dlemo-kun/dlemo-mc/internal/render"
	"github.com/dlemo-kun/dlemo-mc/internal/ui"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/gogpu/gmath"
	"github.com/gogpu/gpucontext"
	"github.com/gogpu/gputypes"
)

type pingResponse struct {
	Description json.RawMessage `json:"description"`
	Players     struct {
		Max    int `json:"max"`
		Online int `json:"online"`
	} `json:"players"`
}

// Run initializes the application window, UI screens, translations, and 30 FPS render loop.
func Run() error {
	opts := config.LoadOptions()
	tr := i18n.NewManager(opts.Language)

	cfg := withPlatformGraphicsAPI(gogpu.DefaultConfig()).
		WithTitle("dlemo-mc").
		WithSize(1024, 640).
		WithMinSize(800, 600).
		WithContinuousRender(true).
		WithPowerPreference(gputypes.PowerPreferenceHighPerformance)

	a := gogpu.NewApp(cfg)
	r := render.New()
	nav := ui.NewScreenManager(nil)

	var connecting atomic.Bool
	connectTo := func(host string, port int) {
		if !connecting.CompareAndSwap(false, true) {
			return
		}
		log.Printf("UI: Connecting to %s:%d...", host, port)
		go func() {
			defer connecting.Store(false)
			if err := client.Connect(host, port, "Player"); err != nil {
				log.Printf("Client stopped: %v", err)
			}
		}()
	}

	var (
		titleMenu    *ui.TitleMenu
		serverScreen *ui.ServerSelectScreen
		worldScreen  *ui.WorldSelectScreen
		langScreen   *ui.LanguageSelectScreen
	)

	refreshServers := func(servers []*ui.ServerEntry) {
		for idx, srv := range servers {
			serverScreen.SetServerStatus(idx, srv.MOTD, 0, 0, false, true)
			go func(i int, host string, port int) {
				raw, err := mcnet.Ping(host, port)
				if err != nil {
					serverScreen.SetServerStatus(i, "No reachable", 0, 0, false, false)
					return
				}
				var parsed pingResponse
				if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
					serverScreen.SetServerStatus(i, "Online", 0, 0, true, false)
					return
				}
				motd := extractMOTD(parsed.Description)
				serverScreen.SetServerStatus(i, motd, parsed.Players.Online, parsed.Players.Max, true, false)
			}(idx, srv.Host, srv.Port)
		}
	}

	serverScreen = ui.NewServerSelectScreen(
		tr,
		[]*ui.ServerEntry{
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
			{Name: "Servidor Local", Host: "localhost", Port: 25565},
			{Name: "Localhost Alt", Host: "127.0.0.1", Port: 25566},
		},
		ui.ServerSelectActions{
			OnConnect: connectTo,
			OnRefresh: refreshServers,
			OnBack:    func() { nav.SetScreen(titleMenu) },
		},
	)

	worldScreen = ui.NewWorldSelectScreen(
		tr,
		[]*ui.WorldEntry{
			{Name: "Mi Mundo", GameMode: "Supervivencia", LastPlayed: "Hoy"},
			{Name: "Laboratorio Redstone", GameMode: "Creativo", LastPlayed: "Ayer"},
		},
		ui.WorldSelectActions{
			OnPlay: func(w *ui.WorldEntry) {
				log.Printf("UI: Opening world %q (singleplayer engine not implemented yet)", w.Name)
			},
			OnCreate: func() { log.Println("UI: Crear mundo clicked") },
			OnBack:   func() { nav.SetScreen(titleMenu) },
		},
	)

	langScreen = ui.NewLanguageSelectScreen(
		tr,
		func(code string) {
			tr.SetLanguage(code)
			opts.Language = code
			if err := config.SaveOptions(opts); err != nil {
				log.Printf("config: failed to save options.json: %v", err)
			}
		},
		func() { nav.SetScreen(titleMenu) },
	)

	titleMenu = ui.NewTitleMenu(tr, ui.TitleMenuActions{
		OnSingleplayer: func() {
			nav.SetScreen(worldScreen)
		},
		OnMultiplayer: func() {
			nav.SetScreen(serverScreen)
			refreshServers(serverScreen.Servers())
		},
		OnRealms:        func() { log.Println("UI: Realms clicked") },
		OnOptions:       func() { log.Println("UI: Opciones clicked") },
		OnLanguage:      func() { nav.SetScreen(langScreen) },
		OnFriends:       func() { log.Println("UI: Amigos clicked") },
		OnAccessibility: func() { log.Println("UI: Accesibilidad clicked") },
		OnQuit:          func() { a.Quit() },
	})

	nav.SetScreen(titleMenu)

	events := a.EventSource()
	events.OnMouseMove(func(x, y float64) {
		nav.OnMouseMove(x, y)
	})
	events.OnMousePress(func(button gpucontext.MouseButton, x, y float64) {
		if button == gpucontext.MouseButtonLeft {
			nav.OnMousePress(x, y)
		}
	})

	a.OnUpdate(func(dt float64) {
		for ev, ok := a.PollInputEvent(); ok; ev, ok = a.PollInputEvent() {
			if scroll, ok := ev.(gpucontext.ScrollEvent); ok {
				nav.OnMouseScroll(float64(scroll.DeltaY))
			}
		}
		nav.Update(dt)
	})

	a.OnDraw(func(dc *gogpu.Context) {
		nav.SetViewport(float64(dc.Width()), float64(dc.Height()))
		dc.ClearColor(gmath.Hex(0x111520))
		r.RenderScreen(dc, a.GPUContextProvider(), nav.Current(), nav.Background())
		a.RequestRedraw()
	})

	stopRender := make(chan struct{})
	var startOnce, stopOnce sync.Once
	stopTicker := func() {
		stopOnce.Do(func() { close(stopRender) })
	}
	defer stopTicker()

	a.OnSurfaceAvailable(func() {
		startOnce.Do(func() {
			go func() {
				ticker := time.NewTicker(time.Second / 30)
				defer ticker.Stop()
				for {
					select {
					case <-stopRender:
						return
					case <-ticker.C:
						a.RequestRedraw()
					}
				}
			}()
		})
	})

	a.OnClose(func() {
		stopTicker()
		r.Close()
		titleMenu.Close()
		serverScreen.Close()
		worldScreen.Close()
		langScreen.Close()
		ui.CloseSharedFonts()
	})

	return a.Run()
}

func extractMOTD(raw json.RawMessage) string {
	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil && plain != "" {
		return plain
	}
	var obj struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Text
	}
	return ""
}

// withPlatformGraphicsAPI sets the graphics backend from DLEMO_GPU if provided,
// or selects the preferred backend for the current operating system.
func withPlatformGraphicsAPI(cfg gogpu.Config) gogpu.Config {
	switch strings.ToLower(os.Getenv("DLEMO_GPU")) {
	case "dx12":
		return cfg.WithGraphicsAPI(gogpu.GraphicsAPIDX12)
	case "vulkan":
		return cfg.WithGraphicsAPI(gogpu.GraphicsAPIVulkan)
	case "opengl", "gles":
		return cfg.WithGraphicsAPI(gogpu.GraphicsAPIGLES)
	case "software":
		return cfg.WithGraphicsAPI(gogpu.GraphicsAPISoftware)
	case "auto":
		return cfg.WithGraphicsAPI(gogpu.GraphicsAPIAuto)
	}

	switch runtime.GOOS {
	case "windows":
		return cfg.WithGraphicsAPI(gogpu.GraphicsAPIDX12)
	case "linux":
		return cfg.WithGraphicsAPI(gogpu.GraphicsAPIVulkan)
	default:
		return cfg.WithGraphicsAPI(gogpu.GraphicsAPIAuto)
	}
}
