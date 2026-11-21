package i18n

import (
	"archive/zip"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/dlemo-kun/dlemo-mc/internal/config"
)

// LanguageInfo describes an available language.
type LanguageInfo struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Region string `json:"region"`
	Hash   string `json:"-"`
}

// Manager loads Minecraft language JSON files from .dlemo-mc/lang cache
// (importing from .minecraft only on first run) and resolves UI keys.
type Manager struct {
	mu          sync.RWMutex
	cacheDir    string
	mcDir       string
	currentCode string
	languages   []LanguageInfo
	langByCode  map[string]LanguageInfo
	enUS        map[string]string
	active      map[string]string
	builtin     map[string]string
}

type assetIndex struct {
	Objects map[string]struct {
		Hash string `json:"hash"`
	} `json:"objects"`
}

type langMeta struct {
	Name   string `json:"language.name"`
	Region string `json:"language.region"`
}

// NewManager loads cached languages from .dlemo-mc/lang, or imports them from
// .minecraft on the first run, and activates initialLang.
func NewManager(initialLang string) *Manager {
	cacheDir := ""
	if base, err := config.Dir(); err == nil {
		cacheDir = filepath.Join(base, "lang")
	}

	m := &Manager{
		cacheDir:   cacheDir,
		langByCode: make(map[string]LanguageInfo),
		enUS:       make(map[string]string),
		active:     make(map[string]string),
		builtin: map[string]string{
			"menu.singleplayer":             "Singleplayer",
			"menu.multiplayer":              "Multiplayer",
			"menu.online":                   "Minecraft Realms",
			"menu.options":                  "Options",
			"menu.quit":                     "Quit Game",
			"narrator.button.language":      "Language",
			"narrator.button.accessibility": "Accessibility",
			"dlemo.menu.friends":            "Friends",
			"multiplayer.title":             "Play Multiplayer",
			"selectServer.select":           "Join Server",
			"selectServer.refresh":          "Refresh",
			"selectWorld.title":             "Select World",
			"selectWorld.select":            "Play Selected World",
			"selectWorld.create":            "Create New World",
			"options.language":              "Language...",
			"options.languageWarning":       "Language translations may not be 100% accurate",
			"gui.back":                      "Back",
			"gui.done":                      "Done",
		},
	}

	if !m.loadFromCache() {
		m.mcDir = findMinecraftDir()
		m.importFromMinecraftAndCache()
	}

	if initialLang == "" {
		initialLang = "en_us"
	}
	m.SetLanguage(initialLang)
	return m
}

// Languages returns all available languages sorted cleanly.
func (m *Manager) Languages() []LanguageInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]LanguageInfo, len(m.languages))
	copy(out, m.languages)
	return out
}

// CurrentCode returns the active language code (e.g. "es_es").
func (m *Manager) CurrentCode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentCode
}

// SetLanguage loads the translation map for code and makes it active.
func (m *Manager) SetLanguage(code string) {
	code = sanitizeCode(code)
	if code == "" {
		code = "en_us"
	}

	loaded := make(map[string]string)
	if code != "en_us" {
		m.mu.RLock()
		cacheDir := m.cacheDir
		info, ok := m.langByCode[code]
		mcDir := m.mcDir
		m.mu.RUnlock()

		loadedOK := false
		if cacheDir != "" {
			cachedFile := filepath.Join(cacheDir, code+".json")
			if data, err := os.ReadFile(cachedFile); err == nil {
				if err := json.Unmarshal(data, &loaded); err == nil {
					loadedOK = true
				}
			}
		}

		// Fallback to .minecraft objects if not in cache
		if !loadedOK && ok && info.Hash != "" && mcDir != "" {
			objPath := filepath.Join(mcDir, "assets", "objects", info.Hash[:2], info.Hash)
			if data, err := os.ReadFile(objPath); err == nil {
				_ = json.Unmarshal(data, &loaded)
			}
		}
	}

	// Custom keys not present in vanilla JSON
	if _, ok := loaded["dlemo.menu.friends"]; !ok {
		if strings.HasPrefix(code, "es_") {
			loaded["dlemo.menu.friends"] = "Amigos"
		} else {
			loaded["dlemo.menu.friends"] = "Friends"
		}
	}

	m.mu.Lock()
	m.currentCode = code
	m.active = loaded
	m.mu.Unlock()
}

// T resolves a translation key using the active language, then en_us, then built-in fallback.
func (m *Manager) T(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if v, ok := m.active[key]; ok && v != "" {
		return v
	}
	if v, ok := m.enUS[key]; ok && v != "" {
		return v
	}
	if v, ok := m.builtin[key]; ok && v != "" {
		return v
	}
	return key
}

func (m *Manager) loadFromCache() bool {
	if m.cacheDir == "" {
		return false
	}

	listPath := filepath.Join(m.cacheDir, "languages.json")
	listData, err := os.ReadFile(listPath)
	if err != nil {
		return false
	}

	var langs []LanguageInfo
	if err := json.Unmarshal(listData, &langs); err != nil || len(langs) == 0 {
		return false
	}

	enPath := filepath.Join(m.cacheDir, "en_us.json")
	if enData, err := os.ReadFile(enPath); err == nil {
		_ = json.Unmarshal(enData, &m.enUS)
	}

	for _, l := range langs {
		m.langByCode[l.Code] = l
	}
	m.languages = langs
	log.Printf("i18n: loaded %d cached languages from %s", len(m.languages), m.cacheDir)
	return true
}

func (m *Manager) importFromMinecraftAndCache() {
	if m.cacheDir != "" {
		_ = os.MkdirAll(m.cacheDir, 0o755)
	}

	enRaw := m.loadEnUSFromJar()
	if len(enRaw) > 0 && m.cacheDir != "" {
		_ = os.WriteFile(filepath.Join(m.cacheDir, "en_us.json"), enRaw, 0o644)
	}

	enName := m.enUS["language.name"]
	if enName == "" {
		enName = "English"
	}
	enRegion := m.enUS["language.region"]
	if enRegion == "" {
		enRegion = "United States"
	}
	m.langByCode["en_us"] = LanguageInfo{Code: "en_us", Name: enName, Region: enRegion}

	importedFromMC := false
	if m.mcDir != "" {
		indexesDir := filepath.Join(m.mcDir, "assets", "indexes")
		if indexFiles, err := os.ReadDir(indexesDir); err == nil {
			var bestIdx assetIndex
			bestCount := 0
			for _, f := range indexFiles {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
					continue
				}
				data, err := os.ReadFile(filepath.Join(indexesDir, f.Name()))
				if err != nil {
					continue
				}
				var idx assetIndex
				if err := json.Unmarshal(data, &idx); err != nil {
					continue
				}
				count := 0
				for k := range idx.Objects {
					if strings.HasPrefix(k, "minecraft/lang/") && strings.HasSuffix(k, ".json") {
						count++
					}
				}
				if count > bestCount {
					bestCount = count
					bestIdx = idx
				}
			}

			for k, obj := range bestIdx.Objects {
				if !strings.HasPrefix(k, "minecraft/lang/") || !strings.HasSuffix(k, ".json") || len(obj.Hash) < 2 {
					continue
				}
				code := sanitizeCode(strings.TrimSuffix(strings.TrimPrefix(k, "minecraft/lang/"), ".json"))
				if code == "" {
					continue
				}
				objPath := filepath.Join(m.mcDir, "assets", "objects", obj.Hash[:2], obj.Hash)
				data, err := os.ReadFile(objPath)
				if err != nil || !json.Valid(data) {
					continue
				}

				// Copy language JSON directly into .dlemo-mc/lang/<code>.json
				if m.cacheDir != "" {
					_ = os.WriteFile(filepath.Join(m.cacheDir, code+".json"), data, 0o644)
				}

				var meta langMeta
				if err := json.Unmarshal(data, &meta); err != nil || meta.Name == "" {
					meta.Name = code
				}
				m.langByCode[code] = LanguageInfo{
					Code:   code,
					Name:   meta.Name,
					Region: meta.Region,
					Hash:   obj.Hash,
				}
				importedFromMC = true
			}
		}
	}

	m.languages = make([]LanguageInfo, 0, len(m.langByCode))
	for _, info := range m.langByCode {
		m.languages = append(m.languages, info)
	}
	sort.Slice(m.languages, func(i, j int) bool {
		if m.languages[i].Code == "en_us" {
			return true
		}
		if m.languages[j].Code == "en_us" {
			return false
		}
		if m.languages[i].Name != m.languages[j].Name {
			return strings.ToLower(m.languages[i].Name) < strings.ToLower(m.languages[j].Name)
		}
		return m.languages[i].Code < m.languages[j].Code
	})

	// Write languages.json last as the completion marker only if Minecraft languages were actually found.
	if importedFromMC && m.cacheDir != "" {
		if data, err := json.MarshalIndent(m.languages, "", "  "); err == nil {
			_ = os.WriteFile(filepath.Join(m.cacheDir, "languages.json"), data, 0o644)
		}
	}

	log.Printf("i18n: imported %d languages (mcDir=%q cached=%v)", len(m.languages), m.mcDir, importedFromMC)
}

func (m *Manager) loadEnUSFromJar() []byte {
	if m.mcDir == "" {
		return nil
	}
	versionsDir := filepath.Join(m.mcDir, "versions")
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
		if f.Name == "assets/minecraft/lang/en_us.json" {
			rc, err := f.Open()
			if err != nil {
				return nil
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil
			}
			_ = json.Unmarshal(data, &m.enUS)
			log.Printf("i18n: extracted en_us.json (%d keys) from %s", len(m.enUS), filepath.Base(bestJar))
			return data
		}
	}
	return nil
}

func sanitizeCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" || strings.ContainsAny(code, `/\.:`) {
		return ""
	}
	return code
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