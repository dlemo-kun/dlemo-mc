package ui

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/dlemo-kun/dlemo-mc/internal/config"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

// FontStack agrupa varias FontSource en orden de prioridad para hacer fallback por carácter.
type FontStack struct {
	sources []*text.FontSource
}

// Face construye una MultiFace al tamaño indicado combinando todas las fuentes cargadas.
func (fs *FontStack) Face(size float64) text.Face {
	if fs == nil || len(fs.sources) == 0 {
		return nil
	}
	if len(fs.sources) == 1 {
		return fs.sources[0].Face(size)
	}

	faces := make([]text.Face, 0, len(fs.sources))
	for _, src := range fs.sources {
		if f := src.Face(size); f != nil {
			faces = append(faces, f)
		}
	}
	switch len(faces) {
	case 0:
		return nil
	case 1:
		return faces[0]
	}

	mf, err := text.NewMultiFace(faces...)
	if err != nil {
		log.Printf("ui: failed to create MultiFace: %v", err)
		return faces[0]
	}
	return mf
}

// closeAll libera todas las fuentes cargadas en el stack.
// Es privado porque los stacks son recursos compartidos entre todas las pantallas.
func (fs *FontStack) closeAll() error {
	if fs == nil {
		return nil
	}
	var firstErr error
	for _, src := range fs.sources {
		if src == nil {
			continue
		}
		if err := src.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	fs.sources = nil
	return firstErr
}

// Close y face ya no deben usarse en pantallas individuales: solo
// CloseSharedFonts puede liberar las fuentes compartidas.

var (
	fontsOnce       sync.Once
	cachedTitleFont *FontStack
	cachedUIFont    *FontStack
)

func loadFont() (*FontStack, *FontStack) {
	fontsOnce.Do(func() {
		var titlePaths, uiPaths []string

		if base, err := config.Dir(); err == nil {
			fontsDir := filepath.Join(base, "assets", "fonts", "noto-fonts")
			titlePaths = []string{
				filepath.Join(fontsDir, "NotoSans-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansArabic-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansArmenian-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansDevanagari-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansGeorgian-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansGothic-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansHebrew-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansKannada-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansLao-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansTamil-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansThai-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansSymbols-Black.ttf"),
				filepath.Join(fontsDir, "NotoSansSymbols2-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansCJKhk-Black.otf"),
				filepath.Join(fontsDir, "NotoSansCJKjp-Black.otf"),
				filepath.Join(fontsDir, "NotoSansCJKkr-Black.otf"),
				filepath.Join(fontsDir, "NotoSansCJKsc-Black.otf"),
				filepath.Join(fontsDir, "NotoSansCJKtc-Black.otf"),
			}
			uiPaths = []string{
				filepath.Join(fontsDir, "NotoSans-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansArabic-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansArmenian-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansDevanagari-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansGeorgian-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansGothic-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansHebrew-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansKannada-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansLao-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansTamil-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansThai-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansSymbols-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansSymbols2-Regular.ttf"),
				filepath.Join(fontsDir, "NotoSansCJKhk-Regular.otf"),
				filepath.Join(fontsDir, "NotoSansCJKjp-Regular.otf"),
				filepath.Join(fontsDir, "NotoSansCJKkr-Regular.otf"),
				filepath.Join(fontsDir, "NotoSansCJKsc-Regular.otf"),
				filepath.Join(fontsDir, "NotoSansCJKtc-Regular.otf"),
			}
		}

		titlePaths = append(titlePaths, defaultSystemFont())
		uiPaths = append(uiPaths, defaultSystemFont())

		cachedTitleFont = loadFontStack(titlePaths...)
		cachedUIFont = loadFontStack(uiPaths...)
	})

	return cachedTitleFont, cachedUIFont
}

// CloseSharedFonts libera las fuentes cacheadas al cerrar la aplicación.
func CloseSharedFonts() {
	if cachedTitleFont != nil {
		_ = cachedTitleFont.closeAll()
		cachedTitleFont = nil
	}
	if cachedUIFont != nil {
		_ = cachedUIFont.closeAll()
		cachedUIFont = nil
	}
}

func loadFontStack(paths ...string) *FontStack {
	var sources []*text.FontSource
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		src, err := text.NewFontSourceFromFile(p)
		if err != nil {
			log.Printf("ui: failed to load font %s: %v", p, err)
			continue
		}
		sources = append(sources, src)
	}
	return &FontStack{sources: sources}
}

func defaultSystemFont() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\Windows\Fonts\segoeui.ttf`
	default:
		return "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
	}
}

// DirtyScreen permite evitar redibujados constantes de pantallas estáticas.
type DirtyScreen interface {
	ConsumeDirty() bool
	IsAnimating() bool
}

// MarkDirtyScreen es implementado por pantallas que pueden invalidarse al activarse.
type MarkDirtyScreen interface {
	MarkDirty()
}

// Screen represents any navigable UI view rendered onto the 2D overlay canvas.
type Screen interface {
	SetViewport(w, h float64)
	Update(dt float64)
	OnMouseMove(x, y float64) bool
	OnMousePress(x, y float64)
	Draw(cc *gg.Context)
	Close()
}

// Scrollable is an optional interface implemented by screens with scrollable lists.
type Scrollable interface {
	OnMouseScroll(deltaY float64)
}

// ScreenManager holds the active screen and dispatches input/update calls safely.
type ScreenManager struct {
	mu         sync.RWMutex
	current    Screen
	background BackgroundProvider
	w, h       float64
}

// NewScreenManager creates a ScreenManager with an initial active screen.
func NewScreenManager(initial Screen) *ScreenManager {
	return &ScreenManager{
		current:    initial,
		background: &SolidBackground{R: 0.067, G: 0.082, B: 0.125, A: 1.0}, // Default #111520
	}
}

// SetBackground sets the active BackgroundProvider (e.g. panorama or world renderer).
func (sm *ScreenManager) SetBackground(bg BackgroundProvider) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.background = bg
}

// Background returns the current BackgroundProvider.
func (sm *ScreenManager) Background() BackgroundProvider {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.background
}

// SetScreen switches the active screen and applies the latest known viewport size.
func (sm *ScreenManager) SetScreen(next Screen) {
	sm.mu.Lock()
	sm.current = next
	w, h := sm.w, sm.h
	sm.mu.Unlock()

	if next != nil {
		if w > 0 && h > 0 {
			next.SetViewport(w, h)
		}
		if dirty, ok := next.(MarkDirtyScreen); ok {
			dirty.MarkDirty()
		}
	}
}

// Current returns the currently active screen.
func (sm *ScreenManager) Current() Screen {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// Update advances animations on the active screen.
func (sm *ScreenManager) Update(dt float64) {
	if s := sm.Current(); s != nil {
		s.Update(dt)
	}
}

// OnMouseMove forwards pointer motion to the active screen.
func (sm *ScreenManager) OnMouseMove(x, y float64) bool {
	if s := sm.Current(); s != nil {
		return s.OnMouseMove(x, y)
	}
	return false
}

// OnMousePress forwards mouse clicks to the active screen.
func (sm *ScreenManager) OnMousePress(x, y float64) {
	if s := sm.Current(); s != nil {
		s.OnMousePress(x, y)
	}
}

// OnMouseScroll forwards wheel scroll events when the active screen supports scrolling.
func (sm *ScreenManager) OnMouseScroll(deltaY float64) {
	if s, ok := sm.Current().(Scrollable); ok {
		s.OnMouseScroll(deltaY)
	}
}

// SetViewport tracks the window size for viewport-aware screen switches.
func (sm *ScreenManager) SetViewport(w, h float64) {
	sm.mu.Lock()
	sm.w, sm.h = w, h
	s := sm.current
	sm.mu.Unlock()

	if s != nil {
		s.SetViewport(w, h)
	}
}
