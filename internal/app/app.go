package app

import (
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlemo-kun/dlemo-mc/internal/client"
	"github.com/dlemo-kun/dlemo-mc/internal/render"
	"github.com/dlemo-kun/dlemo-mc/internal/ui"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/gogpu/gmath"
	"github.com/gogpu/gpucontext"
	"github.com/gogpu/gputypes"
)

// Run initializes the application window, title menu, and 30 FPS render loop.
func Run() error {
	cfg := withPlatformGraphicsAPI(gogpu.DefaultConfig()).
		WithTitle("dlemo-mc").
		WithSize(1024, 640).
		WithMinSize(800, 600).
		WithPowerPreference(gputypes.PowerPreferenceHighPerformance)

	a := gogpu.NewApp(cfg)
	r := render.New()

	var connecting atomic.Bool
	connectLocal := func() {
		if !connecting.CompareAndSwap(false, true) {
			return
		}
		log.Println("UI: Connecting to localhost:25565...")
		go func() {
			defer connecting.Store(false)
			if err := client.Connect("localhost", 25565, "Player"); err != nil {
				log.Printf("Client stopped: %v", err)
			}
		}()
	}

	menu := ui.NewTitleMenu(ui.TitleMenuActions{
		OnSingleplayer:  connectLocal,
		OnMultiplayer:   connectLocal,
		OnRealms:        func() { log.Println("UI: Realms clicked") },
		OnOptions:       func() { log.Println("UI: Options clicked") },
		OnLanguage:      func() { log.Println("UI: Language clicked") },
		OnFriends:       func() { log.Println("UI: Friends clicked") },
		OnAccessibility: func() { log.Println("UI: Accessibility clicked") },
		OnQuit: func() {
			a.Quit()
		},
	})

	events := a.EventSource()
	events.OnMouseMove(func(x, y float64) {
		menu.OnMouseMove(x, y)
	})
	events.OnMousePress(func(button gpucontext.MouseButton, x, y float64) {
		if button == gpucontext.MouseButtonLeft {
			menu.OnMousePress(x, y)
		}
	})

	a.OnUpdate(func(dt float64) {
		menu.Update(dt)
	})

	a.OnDraw(func(dc *gogpu.Context) {
		dc.ClearColor(gmath.Hex(0x111520))
		r.RenderTitleMenu(dc, a.GPUContextProvider(), menu)
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
		menu.Close()
	})

	return a.Run()
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