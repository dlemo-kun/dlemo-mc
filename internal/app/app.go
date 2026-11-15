package app

import (
	"github.com/dlemo-kun/dlemo-mc/internal/render"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/gogpu/gmath"
)

/* Run:
*/
func Run() error {
	a := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle("dlemo-mc").
		WithSize(800, 600).
		WithGraphicsAPI(gogpu.GraphicsAPIDX12))

	r := render.New()

	a.OnDraw(func(dc *gogpu.Context) {
		dc.ClearColor(gmath.Hex(0x0f0f0f))
		r.Frame(dc)
	})

	return a.Run()
}