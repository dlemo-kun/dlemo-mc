package render

import "github.com/gogpu/gogpu"

type Renderer struct {
	ready bool
}

func New() *Renderer { return &Renderer{} }

func (r *Renderer) Frame(dc *gogpu.Context) {
	if !r.ready {
		r.ready = true
	}
}