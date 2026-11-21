package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Options holds persistent client settings stored in .dlemo-mc/options.json.
type Options struct {
	Language string `json:"language"`
}

// Dir returns the path to the persistent .dlemo-mc data directory.
func Dir() (string, error) {
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
	}
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, ".dlemo-mc"), nil
}

// OptionsPath returns the path to the persistent options.json file.
func OptionsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "options.json"), nil
}

// LoadOptions reads options.json from disk, returning defaults if missing or invalid.
func LoadOptions() Options {
	opts := Options{Language: "en_us"}

	path, err := OptionsPath()
	if err != nil {
		return opts
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return opts
	}

	if err := json.Unmarshal(data, &opts); err != nil {
		log.Printf("config: invalid %s, using defaults: %v", path, err)
		return Options{Language: "en_us"}
	}
	if opts.Language == "" {
		opts.Language = "en_us"
	}
	return opts
}

// SaveOptions persists client options to options.json, creating parent folders if needed.
func SaveOptions(opts Options) error {
	path, err := OptionsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}