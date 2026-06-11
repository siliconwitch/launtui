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
	SectionName() string
}

func ConfigPath() (string, error) {
	if path := os.Getenv("LAUNTUI_CONFIG"); path != "" {
		return path, nil
	}

	base := os.Getenv("XDG_CONFIG_HOME")

	if base == "" {
		home, err := os.UserHomeDir()

		if err != nil {
			return "", err
		}

		base = filepath.Join(home, ".config")
	}

	return filepath.Join(base, "launtui", "config.toml"), nil
}

func Load(targets ...Section) error {
	path, err := ConfigPath()

	if err != nil {
		return err
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
			continue
		}

		if err := md.PrimitiveDecode(prim, t); err != nil {
			return fmt.Errorf("config section [%s]: %w", t.SectionName(), err)
		}
	}

	return nil
}
