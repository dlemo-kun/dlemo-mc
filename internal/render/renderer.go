package render

import (
	"log"

	"github.com/dlemo-kun/dlemo-mc/internal/ui"
	"github.com/gogpu/gg"
	_ "github.com/gogpu/gg/gpu" // Enable GPU SDF and text acceleration
	"github.com/gogpu/gg/integration/ggcanvas"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/gpucontext"
)

// Renderer manages the 2D/3D GPU rendering pipeline for the client window.
type Renderer struct {
	canvas *ggcanvas.Canvas
}

// New creates a new Renderer instance.
func New() *Renderer {
	return &Renderer{}
}

// RenderScreen draws any active ui.Screen overlay onto the current swapchain view.
func (r *Renderer) RenderScreen(dc *gogpu.Context, provider gpucontext.DeviceProvider, screen ui.Screen, bg ui.BackgroundProvider) {
	if screen == nil {
		return
	}
	w, h := dc.Width(), dc.Height()
	if w <= 0 || h <= 0 || provider == nil {
		return
	}

	if r.canvas == nil {
		c, err := ggcanvas.New(provider, w, h)
		if err != nil {
			log.Printf("render: failed to create ggcanvas: %v", err)
			return
		}
		r.canvas = c
	}

	cw, ch := r.canvas.Size()
	if cw != w || ch != h {
		if err := r.canvas.Resize(w, h); err != nil {
			log.Printf("render: failed to resize ggcanvas: %v", err)
			return
		}
	}

	screen.SetViewport(float64(w), float64(h))
	if bg != nil {
		bg.Update(0.016) // Approx frame time update for background
	}

	r.canvas.MarkDirty()
	if err := r.canvas.Draw(func(cc *gg.Context) {
		cc.Clear()
		if bg != nil {
			bg.DrawBackground(cc, float64(w), float64(h))
		} else {
			cc.Clear()
		}
		screen.Draw(cc)
	}); err != nil {
		log.Printf("render: failed to draw canvas: %v", err)
		return
	}

	if err := r.canvas.Render(dc.RenderTarget()); err != nil {
		// Fallback: recreate canvas on render failure (e.g. lost GPU device/texture)
		_ = r.canvas.Close()
		r.canvas = nil
		c, errNew := ggcanvas.New(provider, w, h)
		if errNew != nil {
			log.Printf("render: failed to recreate ggcanvas: %v", errNew)
			return
		}
		r.canvas = c
		r.canvas.MarkDirty()
		_ = r.canvas.Draw(func(cc *gg.Context) {
			cc.Clear()
			if bg != nil {
				bg.DrawBackground(cc, float64(w), float64(h))
			} else {
				cc.Clear()
			}
			screen.Draw(cc)
		})
		_ = r.canvas.Render(dc.RenderTarget())
	}
}

// Close releases GPU canvas and accelerator resources.
func (r *Renderer) Close() {
	if r.canvas != nil {
		_ = r.canvas.Close()
		r.canvas = nil
	}
	gg.CloseAccelerator()
}
