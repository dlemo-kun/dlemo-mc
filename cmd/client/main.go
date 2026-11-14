package main

import (
	"github.com/gogpu/gogpu"
)

func main() {
	app := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle("dlemo-mc").
		WithSize(800, 600).
		WithGraphicsAPI(gogpu.GraphicsAPIVulkan))

	app.Run()
}
