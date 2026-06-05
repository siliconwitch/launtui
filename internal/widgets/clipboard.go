package widgets

type ClipboardConfig struct {
	MaxEntries int `toml:"max_entries"`
}

func (ClipboardConfig) SectionName() string { return "clipboard" }

func DefaultClipboardConfig() ClipboardConfig {
	return ClipboardConfig{MaxEntries: 50}
}
