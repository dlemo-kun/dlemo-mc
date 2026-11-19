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

// RenderTitleMenu draws the title menu overlay onto the current swapchain view,
// re-rasterizing the 2D canvas only when resized or marked dirty.
// RenderTitleMenu draws the title menu overlay onto the current swapchain view.
func (r *Renderer) RenderTitleMenu(dc *gogpu.Context, provider gpucontext.DeviceProvider, menu *ui.TitleMenu) {
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

	menu.SetViewport(float64(w), float64(h))

	r.canvas.Draw(func(cc *gg.Context) {
		// Clear with transparent alpha so the 3D background beneath stays visible.
		cc.Clear()
		menu.Draw(cc)
	})

	if err := r.canvas.Render(dc.RenderTarget()); err != nil {
		log.Printf("render: failed to composite canvas: %v", err)
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