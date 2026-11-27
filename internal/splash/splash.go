package splash

import (
	"archive/zip"
	"encoding/json"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	splashesFileName    = "splashes.txt"
	jarSplashesPath     = "assets/minecraft/texts/splashes.txt"
	indexSplashesKey    = "minecraft/texts/splashes.txt"
	maxSplashFileBytes  = 4 << 20 // 4 MiB safety cap
)

type monthDay struct {
	Month time.Month
	Day   int
}

var specialDates = map[monthDay]string{
	{time.October, 31}:  "Have a terrifying Halloween!",
	{time.December, 10}: "Happy birthday, dlemo-kun!",
	{time.December, 24}: "Have a happy Christmas Eve!",
	{time.December, 25}: "Merry Christmas everyone!",
	{time.December, 31}: "The year is coming to an end!",
	{time.January, 1}:   "Long time no see, since last year!",
}

var defaultSplashes = []string{
	"Go Edition!",
	"Made in Go!",
	"100% pure Go!",
}

type assetIndex struct {
	Objects map[string]struct {
		Hash string `json:"hash"`
	} `json:"objects"`
}

// AssetsDir returns the persistent assets directory (%APPDATA%/.dlemo-mc/assets on Windows).
func AssetsDir() string {
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
	}
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return ""
		}
	}
	return filepath.Join(base, ".dlemo-mc", "assets", "texts")
}

// Manager loads splash screen texts from cache (importing from .minecraft on first run)
// and resolves random or holiday-specific splash messages.
type Manager struct {
	mu       sync.RWMutex
	cacheDir string
	mcDir    string
	splashes []string
	now      func() time.Time
}

// NewManager loads cached splash texts from %APPDATA%/.dlemo-kun/assets/splashes.txt,
// or extracts them from .minecraft on the first run.
func NewManager() *Manager {
	return NewManagerWithConfig(AssetsDir(), findMinecraftDir(), time.Now)
}

// NewManagerWithConfig initializes a splash Manager with explicit directories and clock.
func NewManagerWithConfig(cacheDir, mcDir string, now func() time.Time) *Manager {
	if now == nil {
		now = time.Now
	}
	m := &Manager{
		cacheDir: cacheDir,
		mcDir:    mcDir,
		now:      now,
	}

	if !m.loadFromCache() {
		m.importFromMinecraftAndCache()
	}

	m.mu.Lock()
	if len(m.splashes) == 0 {
		m.splashes = append([]string(nil), defaultSplashes...)
	}
	m.mu.Unlock()

	return m
}

// Splashes returns a copy of all loaded splash texts.
func (m *Manager) Splashes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, len(m.splashes))
	copy(out, m.splashes)
	return out
}

// SpecialSplash returns the special splash message if t falls on a configured holiday/date.
func SpecialSplash(t time.Time) (string, bool) {
	msg, ok := specialDates[monthDay{Month: t.Month(), Day: t.Day()}]
	return msg, ok
}

// CurrentSpecial returns the special splash text for the current time if one applies today.
func (m *Manager) CurrentSpecial() (string, bool) {
	return SpecialSplash(m.now())
}

// Current returns the special splash for today if applicable, or a random splash avoiding prev.
func (m *Manager) Current(prev string) (text string, isSpecial bool) {
	if msg, ok := SpecialSplash(m.now()); ok {
		return msg, true
	}

	m.mu.RLock()
	list := m.splashes
	m.mu.RUnlock()

	if len(list) == 0 {
		return defaultSplashes[0], false
	}
	if len(list) == 1 {
		return list[0], false
	}

	idx := rand.IntN(len(list))
	if list[idx] == prev {
		idx = (idx + 1 + rand.IntN(len(list)-1)) % len(list)
	}
	return list[idx], false
}

func (m *Manager) loadFromCache() bool {
	if m.cacheDir == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(m.cacheDir, splashesFileName))
	if err != nil {
		return false
	}
	parsed := parseSplashes(data)
	if len(parsed) == 0 {
		return false
	}
	m.mu.Lock()
	m.splashes = parsed
	m.mu.Unlock()
	log.Printf("splash: loaded %d cached splashes from %s", len(parsed), m.cacheDir)
	return true
}

func (m *Manager) importFromMinecraftAndCache() {
	if m.mcDir == "" {
		return
	}

	data := extractSplashesFromJar(m.mcDir)
	if len(data) == 0 {
		data = extractSplashesFromIndex(m.mcDir)
	}
	if len(data) == 0 {
		return
	}

	parsed := parseSplashes(data)
	if len(parsed) == 0 {
		return
	}

	m.mu.Lock()
	m.splashes = parsed
	m.mu.Unlock()

	if m.cacheDir != "" {
		if err := os.MkdirAll(m.cacheDir, 0o755); err != nil {
			log.Printf("splash: failed to create cache dir %s: %v", m.cacheDir, err)
			return
		}
		outPath := filepath.Join(m.cacheDir, splashesFileName)
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			log.Printf("splash: failed to write %s: %v", outPath, err)
			return
		}
		log.Printf("splash: extracted %d splashes to %s", len(parsed), outPath)
	}
}

func extractSplashesFromJar(mcDir string) []byte {
	versionsDir := filepath.Join(mcDir, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return nil
	}

	var bestJar string
	var bestMod int64
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		jarPath := filepath.Join(versionsDir, e.Name(), e.Name()+".jar")
		info, err := os.Stat(jarPath)
		if err != nil {
			continue
		}
		if info.ModTime().UnixNano() > bestMod {
			bestMod = info.ModTime().UnixNano()
			bestJar = jarPath
		}
	}
	if bestJar == "" {
		return nil
	}

	zr, err := zip.OpenReader(bestJar)
	if err != nil {
		return nil
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.Name == jarSplashesPath {
			rc, err := f.Open()
			if err != nil {
				return nil
			}
			data, err := io.ReadAll(io.LimitReader(rc, maxSplashFileBytes))
			_ = rc.Close()
			if err != nil {
				return nil
			}
			return data
		}
	}
	return nil
}

func extractSplashesFromIndex(mcDir string) []byte {
	indexesDir := filepath.Join(mcDir, "assets", "indexes")
	indexFiles, err := os.ReadDir(indexesDir)
	if err != nil {
		return nil
	}

	for _, f := range indexFiles {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(indexesDir, f.Name()))
		if err != nil {
			continue
		}
		var idx assetIndex
		if err := json.Unmarshal(raw, &idx); err != nil {
			continue
		}
		obj, ok := idx.Objects[indexSplashesKey]
		if !ok || len(obj.Hash) < 2 {
			continue
		}
		objPath := filepath.Join(mcDir, "assets", "objects", obj.Hash[:2], obj.Hash)
		if data, err := os.ReadFile(objPath); err == nil && len(data) > 0 {
			return data
		}
	}
	return nil
}

func parseSplashes(data []byte) []string {
	text := strings.TrimPrefix(string(data), "\uFEFF")
	rawLines := strings.Split(text, "\n")
	out := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func findMinecraftDir() string {
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			candidates = append(candidates, filepath.Join(appData, ".minecraft"))
		}
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, "Library", "Application Support", "minecraft"))
		}
	default:
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, ".minecraft"))
		}
	}
	for _, dir := range candidates {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir
		}
	}
	return ""
}
