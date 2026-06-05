package tui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Section interface {
	// Implemented by every widget's config struct
	SectionName() string
}

func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "launtui", "config.toml"), nil
}

// Overlays config file on top of the defaults
func Load(targets ...Section) error {
	path, err := ConfigPath()

	if err != nil {
		return nil // Keep defaults if no config dir is resolvable
	}

	var raw map[string]toml.Primitive

	md, err := toml.DecodeFile(path, &raw)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("reading %s: %w", path, err)
	}

	for _, t := range targets {
		prim, ok := raw[t.SectionName()]

		if !ok {
			continue // no [section] for this widget; keep its defaults
		}

		if err := md.PrimitiveDecode(prim, t); err != nil {
			return fmt.Errorf("config section [%s]: %w", t.SectionName(), err)
		}
	}

	return nil
}
